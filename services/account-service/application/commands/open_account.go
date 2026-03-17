package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/pkg/outbox"
	"fincore/services/account-service/application/ports"
	"fincore/services/account-service/domain"
)

const accountEventsTopicV1 = "banking.accounts.v1"

type OpenAccount struct {
	CustomerID     ids.ID
	IdempotencyKey string
}

type OpenAccountResult struct {
	AccountID ids.ID
	Version   int64
}

type OpenAccountHandler struct {
	uow ports.UnitOfWork
}

func NewOpenAccountHandler(uow ports.UnitOfWork) *OpenAccountHandler {
	return &OpenAccountHandler{uow: uow}
}

func (h *OpenAccountHandler) Handle(ctx context.Context, cmd OpenAccount) (*OpenAccountResult, error) {
	if cmd.CustomerID == "" {
		return nil, errors.New("customer id required")
	}
	if cmd.IdempotencyKey == "" {
		return nil, errors.New("idempotency key required")
	}

	var res *OpenAccountResult
	err := h.uow.WithTx(ctx, func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore, proj ports.AccountProjectionRepository) error {
		ag, err := domain.NewAccount(cmd.CustomerID)
		if err != nil {
			return err
		}
		events := ag.PullChanges()
		stored := make([]eventRecord, 0, len(events))
		for _, e := range events {
			b, err := json.Marshal(e)
			if err != nil {
				return fmt.Errorf("marshal event: %w", err)
			}
			stored = append(stored, eventRecord{typ: e.EventType(), data: b, occurredAtUTC: e.OccurredAtUTC()})
		}

		toAppend := make([]eventstore.Event, 0, len(stored))
		for i, s := range stored {
			ver := int64(i + 1)
			toAppend = append(toAppend, eventstore.Event{
				ID:            ids.New().String(),
				AggregateID:   string(ag.ID()),
				AggregateType: "account",
				Version:       ver,
				Type:          s.typ,
				OccurredAt:    s.occurredAtUTC,
				Data:          s.data,
			})

			payload, err := json.Marshal(outboxEventEnvelope{
				EventType:     s.typ,
				AggregateID:   string(ag.ID()),
				AggregateType: "account",
				Version:       ver,
				OccurredAtUTC: s.occurredAtUTC,
				Data:          s.data,
			})
			if err != nil {
				return fmt.Errorf("marshal outbox envelope: %w", err)
			}
			if err := ob.Enqueue(ctx, outbox.Message{
				ID:    ids.New().String(),
				Topic: accountEventsTopicV1,
				Key:   []byte(ag.ID().String()),
				Value: payload,
				Headers: map[string]string{
					"event_type":     s.typ,
					"aggregate_id":   ag.ID().String(),
					"aggregate_type": "account",
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
		if err := proj.Upsert(ctx, ports.AccountProjection{
			AccountID:  ag.ID().String(),
			CustomerID: ag.CustomerID().String(),
			Status:     string(ag.Status()),
			Version:    ag.Version(),
		}); err != nil {
			return err
		}

		res = &OpenAccountResult{AccountID: ag.ID(), Version: ag.Version()}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

type eventRecord struct {
	typ           string
	data          []byte
	occurredAtUTC time.Time
}

type outboxEventEnvelope struct {
	EventType     string    `json:"event_type"`
	AggregateID   string    `json:"aggregate_id"`
	AggregateType string    `json:"aggregate_type"`
	Version       int64     `json:"version"`
	OccurredAtUTC time.Time `json:"occurred_at_utc"`
	Data          []byte    `json:"data"`
}
