package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/services/account-service/application/ports"
	"fincore/services/account-service/domain"
)

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
	err := h.uow.WithTx(ctx, func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore) error {
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
			toAppend = append(toAppend, eventstore.Event{
				ID:            ids.New().String(),
				AggregateID:   string(ag.ID()),
				AggregateType: "account",
				Version:       int64(i + 1),
				Type:          s.typ,
				OccurredAt:    s.occurredAtUTC,
				Data:          s.data,
			})
		}
		if err := es.Append(ctx, toAppend); err != nil {
			return fmt.Errorf("append events: %w", err)
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
