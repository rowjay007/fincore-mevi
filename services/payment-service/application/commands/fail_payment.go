package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/pkg/outbox"
	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/domain"
)

type FailPayment struct {
	PaymentID      ids.ID
	IdempotencyKey string
	Reason         string
}

type FailPaymentResult struct {
	PaymentID ids.ID
	Version   int64
	Status    domain.Status
}

type FailPaymentHandler struct {
	uow ports.UnitOfWork
}

func NewFailPaymentHandler(uow ports.UnitOfWork) *FailPaymentHandler {
	return &FailPaymentHandler{uow: uow}
}

func (h *FailPaymentHandler) Handle(ctx context.Context, cmd FailPayment) (*FailPaymentResult, error) {
	if cmd.PaymentID == "" {
		return nil, errors.New("payment_id required")
	}
	if cmd.IdempotencyKey == "" {
		return nil, errors.New("idempotency_key required")
	}
	cmd.Reason = strings.TrimSpace(cmd.Reason)

	var res *FailPaymentResult
	err := h.uow.WithTx(ctx, func(ctx context.Context, es ports.PaymentEventStore, ob ports.OutboxStore, proj ports.PaymentProjectionRepository) error {
		p, err := LoadPayment(ctx, es, cmd.PaymentID)
		if err != nil {
			return err
		}
		if err := p.Fail(cmd.Reason); err != nil {
			return err
		}
		changes := p.PullChanges()
		if len(changes) != 1 {
			return errors.New("unexpected number of changes")
		}

		ev, ok := changes[0].(domain.PaymentFailed)
		if !ok {
			return errors.New("unexpected event")
		}
		b, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}

		ver := p.Version()
		if err := es.Append(ctx, []eventstore.Event{{
			ID:            ids.New().String(),
			AggregateID:   cmd.PaymentID.String(),
			AggregateType: "payment",
			Version:       ver,
			Type:          ev.EventType(),
			OccurredAt:    ev.OccurredAtUTC(),
			Data:          b,
		}}); err != nil {
			return fmt.Errorf("append events: %w", err)
		}

		cur, ok, err := proj.GetByID(ctx, cmd.PaymentID.String())
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("payment not found")
		}
		cur.Status = string(p.Status())
		cur.Version = ver
		if err := proj.Upsert(ctx, cur); err != nil {
			return err
		}

		payload, err := json.Marshal(outboxEventEnvelope{
			EventType:     ev.EventType(),
			AggregateID:   cmd.PaymentID.String(),
			AggregateType: "payment",
			Version:       ver,
			OccurredAtUTC: ev.OccurredAtUTC(),
			Data:          b,
		})
		if err != nil {
			return err
		}
		if err := ob.Enqueue(ctx, outbox.Message{
			ID:    ids.New().String(),
			Topic: paymentEventsTopicV1,
			Key:   []byte(cmd.PaymentID.String()),
			Value: payload,
			Headers: map[string]string{
				"event_type":     ev.EventType(),
				"aggregate_id":   cmd.PaymentID.String(),
				"aggregate_type": "payment",
				"schema":         "json-envelope.v1",
			},
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}

		res = &FailPaymentResult{PaymentID: cmd.PaymentID, Version: ver, Status: p.Status()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
