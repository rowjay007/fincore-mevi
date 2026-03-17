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
	"fincore/services/ledger-service/application/ports"
	"fincore/services/ledger-service/domain"
)

const ledgerEventsTopicV1 = "banking.ledger.v1"

type PostEntry struct {
	IdempotencyKey string
	AccountID      ids.ID
	Type           domain.EntryType
	Amount         money.Money
	Narration      string
	OccurredAtUTC  time.Time
}

type PostEntryResult struct {
	EntryID ids.ID
}

type PostEntryHandler struct {
	uow ports.UnitOfWork
}

func NewPostEntryHandler(uow ports.UnitOfWork) *PostEntryHandler {
	return &PostEntryHandler{uow: uow}
}

func (h *PostEntryHandler) Handle(ctx context.Context, cmd PostEntry) (*PostEntryResult, error) {
	if cmd.IdempotencyKey == "" {
		return nil, errors.New("idempotency key required")
	}
	if cmd.AccountID == "" {
		return nil, errors.New("account id required")
	}
	if cmd.Type != domain.EntryTypeDeposit && cmd.Type != domain.EntryTypeWithdrawal {
		return nil, errors.New("entry type required")
	}
	if err := validateMoney(cmd.Amount); err != nil {
		return nil, err
	}

	var res *PostEntryResult
	returnErr := h.uow.WithTx(ctx, func(ctx context.Context, es ports.LedgerEventStore, bal ports.BalanceRepository, idem ports.IdempotencyRepository, ob ports.OutboxStore) error {
		if entryID, ok, err := idem.GetByKey(ctx, cmd.IdempotencyKey); err != nil {
			return err
		} else if ok {
			res = &PostEntryResult{EntryID: ids.ID(entryID)}
			return nil
		}

		currentBal, err := bal.GetBalanceKobo(ctx, cmd.AccountID.String())
		if err != nil {
			return err
		}

		delta := cmd.Amount.AmountKobo()
		if cmd.Type == domain.EntryTypeWithdrawal {
			delta = -delta
			if currentBal+delta < 0 {
				return domain.ErrInsufficientFunds
			}
		}

		entry, err := domain.NewEntry(cmd.AccountID, cmd.Type, cmd.Amount, cmd.Narration, cmd.OccurredAtUTC)
		if err != nil {
			return err
		}

		events := entry.PullChanges()
		toAppend := make([]eventstore.Event, 0, len(events))
		for i, e := range events {
			b, err := json.Marshal(e)
			if err != nil {
				return fmt.Errorf("marshal event: %w", err)
			}
			toAppend = append(toAppend, eventstore.Event{
				ID:            ids.New().String(),
				AggregateID:   entry.ID().String(),
				AggregateType: "ledger_entry",
				Version:       int64(i + 1),
				Type:          e.EventType(),
				OccurredAt:    e.OccurredAtUTC(),
				Data:          b,
			})
		}
		if err := es.Append(ctx, toAppend); err != nil {
			return err
		}
		if entry.Version()%50 == 0 {
			if err := saveSnapshot(ctx, es, entry); err != nil {
				return err
			}
		}
		if err := bal.ApplyDelta(ctx, cmd.AccountID.String(), delta); err != nil {
			return err
		}
		if err := idem.Save(ctx, cmd.IdempotencyKey, entry.ID().String()); err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]any{
			"event_type":      "ledger.entry_posted.v1",
			"aggregate_id":    entry.ID().String(),
			"aggregate_type":  "ledger_entry",
			"account_id":      cmd.AccountID.String(),
			"delta_kobo":      delta,
			"occurred_at_utc": entry.OccurredAtUTC(),
		})
		if err != nil {
			return err
		}
		if err := ob.Enqueue(ctx, outbox.Message{
			ID:        ids.New().String(),
			Topic:     ledgerEventsTopicV1,
			Key:       []byte(cmd.AccountID.String()),
			Value:     payload,
			Headers:   map[string]string{"schema": "json.v1"},
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}

		res = &PostEntryResult{EntryID: entry.ID()}
		return nil
	})
	if returnErr != nil {
		return nil, returnErr
	}
	return res, nil
}

func saveSnapshot(ctx context.Context, es ports.LedgerEventStore, entry *domain.Entry) error {
	s := entry.Snapshot()
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return es.SaveSnapshot(ctx, eventstore.Snapshot{
		AggregateID:   entry.ID().String(),
		AggregateType: "ledger_entry",
		Version:       s.Version,
		CreatedAt:     time.Now().UTC(),
		Data:          b,
	})
}

func validateMoney(m money.Money) error {
	if m.Currency() != money.NGN {
		return domain.ErrCurrencyNotNGN
	}
	if m.IsNegativeOrZero() {
		return domain.ErrInvalidAmount
	}
	return nil
}
