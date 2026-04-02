package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/pkg/outbox"
	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/domain"
)

const paymentEventsTopicV1 = "banking.payments.v1"

type InitiatePayment struct {
	FromAccountID  ids.ID
	ToAccountID    ids.ID
	Amount         money.Money
	Narration      string
	IdempotencyKey string
}

type InitiatePaymentResult struct {
	PaymentID ids.ID
	Version   int64
	Status    domain.Status
}

type InitiatePaymentHandler struct {
	uow      ports.UnitOfWork
	temporal ports.TemporalClient
}

func NewInitiatePaymentHandler(uow ports.UnitOfWork, temporal ports.TemporalClient) *InitiatePaymentHandler {
	return &InitiatePaymentHandler{uow: uow, temporal: temporal}
}

type outboxEventEnvelope struct {
	EventType     string    `json:"event_type"`
	AggregateID   string    `json:"aggregate_id"`
	AggregateType string    `json:"aggregate_type"`
	Version       int64     `json:"version"`
	OccurredAtUTC time.Time `json:"occurred_at_utc"`
	Data          []byte    `json:"data"`
}

type eventRecord struct {
	typ           string
	data          []byte
	occurredAtUTC time.Time
}

func (h *InitiatePaymentHandler) Handle(ctx context.Context, cmd InitiatePayment) (*InitiatePaymentResult, error) {
	if cmd.FromAccountID == "" {
		return nil, errors.New("from_account_id required")
	}
	if cmd.ToAccountID == "" {
		return nil, errors.New("to_account_id required")
	}
	if cmd.IdempotencyKey == "" {
		return nil, errors.New("idempotency_key required")
	}
	if cmd.FromAccountID == cmd.ToAccountID {
		return nil, errors.New("from_account_id and to_account_id must differ")
	}
	if cmd.Amount.Currency() != money.NGN {
		return nil, errors.New("unsupported currency")
	}
	if cmd.Amount.AmountKobo() <= 0 {
		return nil, errors.New("amount must be positive")
	}

	var res *InitiatePaymentResult
	var createdProjection ports.PaymentProjection
	err := h.uow.WithTx(ctx, func(ctx context.Context, es ports.PaymentEventStore, ob ports.OutboxStore, proj ports.PaymentProjectionRepository) error {
		p, err := domain.NewPayment(cmd.FromAccountID, cmd.ToAccountID, cmd.Amount, cmd.Narration)
		if err != nil {
			return err
		}
		events := p.PullChanges()
		stored := make([]eventRecord, 0, len(events))
		for _, e := range events {
			switch ev := e.(type) {
			case domain.PaymentInitiated:
				b, err := json.Marshal(ev)
				if err != nil {
					return fmt.Errorf("marshal event: %w", err)
				}
				stored = append(stored, eventRecord{typ: ev.EventType(), data: b, occurredAtUTC: ev.OccurredAtUTC()})
			default:
				return errors.New("unsupported event")
			}
		}

		toAppend := make([]eventstore.Event, 0, len(stored))
		for i, s := range stored {
			ver := int64(i + 1)
			toAppend = append(toAppend, eventstore.Event{
				ID:            ids.New().String(),
				AggregateID:   p.ID().String(),
				AggregateType: "payment",
				Version:       ver,
				Type:          s.typ,
				OccurredAt:    s.occurredAtUTC,
				Data:          s.data,
			})

			payload, err := json.Marshal(outboxEventEnvelope{
				EventType:     s.typ,
				AggregateID:   p.ID().String(),
				AggregateType: "payment",
				Version:       ver,
				OccurredAtUTC: s.occurredAtUTC,
				Data:          s.data,
			})
			if err != nil {
				return fmt.Errorf("marshal outbox envelope: %w", err)
			}
			if err := ob.Enqueue(ctx, outbox.Message{
				ID:    ids.New().String(),
				Topic: paymentEventsTopicV1,
				Key:   []byte(p.ID().String()),
				Value: payload,
				Headers: map[string]string{
					"event_type":     s.typ,
					"aggregate_id":   p.ID().String(),
					"aggregate_type": "payment",
					"schema":         "json-envelope.v1",
				},
				CreatedAt: time.Now().UTC(),
			}); err != nil {
				return fmt.Errorf("enqueue outbox: %w", err)
			}
		}

		if err := es.Append(ctx, toAppend); err != nil {
			return fmt.Errorf("append events: %w", err)
		}

		createdProjection = ports.PaymentProjection{
			PaymentID:     p.ID().String(),
			FromAccountID: cmd.FromAccountID.String(),
			ToAccountID:   cmd.ToAccountID.String(),
			Currency:      string(cmd.Amount.Currency()),
			AmountKobo:    cmd.Amount.AmountKobo(),
			Narration:     cmd.Narration,
			Status:        string(p.Status()),
			Version:       p.Version(),
		}
		if err := proj.Upsert(ctx, createdProjection); err != nil {
			return err
		}

		res = &InitiatePaymentResult{PaymentID: p.ID(), Version: p.Version(), Status: p.Status()}
		return nil
	})
	if err != nil {
		return nil, err
	}

	if h.temporal != nil {
		wid, rid, err := h.temporal.StartTransferWorkflow(ctx, ports.TransferWorkflowInput{
			PaymentID:      res.PaymentID.String(),
			FromAccountID:  createdProjection.FromAccountID,
			ToAccountID:    createdProjection.ToAccountID,
			Currency:       createdProjection.Currency,
			AmountKobo:     createdProjection.AmountKobo,
			Narration:      createdProjection.Narration,
			IdempotencyKey: cmd.IdempotencyKey,
		})
		if err != nil {
			return nil, err
		}

		if err := h.uow.WithTx(ctx, func(ctx context.Context, es ports.PaymentEventStore, ob ports.OutboxStore, proj ports.PaymentProjectionRepository) error {
			cur, ok, err := proj.GetByID(ctx, res.PaymentID.String())
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("payment not found")
			}
			cur.TemporalWorkflowID = wid
			cur.TemporalRunID = rid
			return proj.Upsert(ctx, cur)
		}); err != nil {
			return nil, err
		}
	}
	return res, nil
}
