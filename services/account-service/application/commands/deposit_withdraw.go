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
	"fincore/services/account-service/application/ports"
	"fincore/services/account-service/domain"
)

type DepositMoney struct {
	AccountID      ids.ID
	Amount         money.Money
	IdempotencyKey string
	Narration      string
}

type WithdrawMoney struct {
	AccountID      ids.ID
	Amount         money.Money
	IdempotencyKey string
	Narration      string
}

type DepositWithdrawResult struct {
	AccountID ids.ID
	EntryID   ids.ID
}

type DepositMoneyHandler struct {
	uow    ports.UnitOfWork
	ledger ports.LedgerClient
}

type WithdrawMoneyHandler struct {
	uow    ports.UnitOfWork
	ledger ports.LedgerClient
}

func NewDepositMoneyHandler(uow ports.UnitOfWork, ledger ports.LedgerClient) *DepositMoneyHandler {
	return &DepositMoneyHandler{uow: uow, ledger: ledger}
}

func NewWithdrawMoneyHandler(uow ports.UnitOfWork, ledger ports.LedgerClient) *WithdrawMoneyHandler {
	return &WithdrawMoneyHandler{uow: uow, ledger: ledger}
}

func (h *DepositMoneyHandler) Handle(ctx context.Context, cmd DepositMoney) (*DepositWithdrawResult, error) {
	if cmd.AccountID == "" {
		return nil, errors.New("account id required")
	}
	if cmd.IdempotencyKey == "" {
		return nil, errors.New("idempotency key required")
	}
	if err := domain.ValidateNGNAmountPositive(cmd.Amount); err != nil {
		return nil, err
	}

	entryID, err := h.ledger.PostEntry(ctx, cmd.IdempotencyKey, cmd.AccountID, ports.LedgerEntryTypeDeposit, cmd.Amount, cmd.Narration)
	if err != nil {
		return nil, err
	}

	var res *DepositWithdrawResult
	err = h.uow.WithTx(ctx, func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore, proj ports.AccountProjectionRepository) error {
		ag, err := LoadAccount(ctx, es, cmd.AccountID)
		if err != nil {
			return err
		}
		if ag.Status() != domain.StatusActive {
			return domain.ErrAccountNotActive
		}

		ev := domain.MoneyDeposited{AccountID: cmd.AccountID, Amount: cmd.Amount, EntryID: entryID, OccurredAt: time.Now().UTC()}
		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}

		ver := ag.Version() + 1
		if err := es.Append(ctx, []eventstore.Event{{
			ID:            ids.New().String(),
			AggregateID:   cmd.AccountID.String(),
			AggregateType: "account",
			Version:       ver,
			Type:          ev.EventType(),
			OccurredAt:    ev.OccurredAtUTC(),
			Data:          b,
		}}); err != nil {
			return fmt.Errorf("append account event: %w", err)
		}
		if ver%50 == 0 {
			if err := saveSnapshot(ctx, es, ag, cmd.AccountID, ver); err != nil {
				return err
			}
		}
		if err := proj.Upsert(ctx, ports.AccountProjection{
			AccountID:  cmd.AccountID.String(),
			CustomerID: ag.CustomerID().String(),
			Status:     string(ag.Status()),
			Version:    ver,
		}); err != nil {
			return err
		}

		payload, err := json.Marshal(outboxEventEnvelope{
			EventType:     ev.EventType(),
			AggregateID:   cmd.AccountID.String(),
			AggregateType: "account",
			Version:       ver,
			OccurredAtUTC: ev.OccurredAtUTC(),
			Data:          b,
		})
		if err != nil {
			return err
		}
		if err := ob.Enqueue(ctx, outbox.Message{
			ID:        ids.New().String(),
			Topic:     accountEventsTopicV1,
			Key:       []byte(cmd.AccountID.String()),
			Value:     payload,
			Headers:   map[string]string{"event_type": ev.EventType(), "aggregate_id": cmd.AccountID.String(), "aggregate_type": "account", "schema": "json-envelope.v1"},
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}

		res = &DepositWithdrawResult{AccountID: cmd.AccountID, EntryID: entryID}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (h *WithdrawMoneyHandler) Handle(ctx context.Context, cmd WithdrawMoney) (*DepositWithdrawResult, error) {
	if cmd.AccountID == "" {
		return nil, errors.New("account id required")
	}
	if cmd.IdempotencyKey == "" {
		return nil, errors.New("idempotency key required")
	}
	if err := domain.ValidateNGNAmountPositive(cmd.Amount); err != nil {
		return nil, err
	}

	entryID, err := h.ledger.PostEntry(ctx, cmd.IdempotencyKey, cmd.AccountID, ports.LedgerEntryTypeWithdrawal, cmd.Amount, cmd.Narration)
	if err != nil {
		return nil, err
	}

	var res *DepositWithdrawResult
	err = h.uow.WithTx(ctx, func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore, proj ports.AccountProjectionRepository) error {
		ag, err := LoadAccount(ctx, es, cmd.AccountID)
		if err != nil {
			return err
		}
		if ag.Status() != domain.StatusActive {
			return domain.ErrAccountNotActive
		}

		ev := domain.MoneyWithdrawn{AccountID: cmd.AccountID, Amount: cmd.Amount, EntryID: entryID, OccurredAt: time.Now().UTC()}
		b, err := json.Marshal(ev)
		if err != nil {
			return err
		}

		ver := ag.Version() + 1
		if err := es.Append(ctx, []eventstore.Event{{
			ID:            ids.New().String(),
			AggregateID:   cmd.AccountID.String(),
			AggregateType: "account",
			Version:       ver,
			Type:          ev.EventType(),
			OccurredAt:    ev.OccurredAtUTC(),
			Data:          b,
		}}); err != nil {
			return fmt.Errorf("append account event: %w", err)
		}
		if ver%50 == 0 {
			if err := saveSnapshot(ctx, es, ag, cmd.AccountID, ver); err != nil {
				return err
			}
		}
		if err := proj.Upsert(ctx, ports.AccountProjection{
			AccountID:  cmd.AccountID.String(),
			CustomerID: ag.CustomerID().String(),
			Status:     string(ag.Status()),
			Version:    ver,
		}); err != nil {
			return err
		}

		payload, err := json.Marshal(outboxEventEnvelope{
			EventType:     ev.EventType(),
			AggregateID:   cmd.AccountID.String(),
			AggregateType: "account",
			Version:       ver,
			OccurredAtUTC: ev.OccurredAtUTC(),
			Data:          b,
		})
		if err != nil {
			return err
		}
		if err := ob.Enqueue(ctx, outbox.Message{
			ID:        ids.New().String(),
			Topic:     accountEventsTopicV1,
			Key:       []byte(cmd.AccountID.String()),
			Value:     payload,
			Headers:   map[string]string{"event_type": ev.EventType(), "aggregate_id": cmd.AccountID.String(), "aggregate_type": "account", "schema": "json-envelope.v1"},
			CreatedAt: time.Now().UTC(),
		}); err != nil {
			return err
		}

		res = &DepositWithdrawResult{AccountID: cmd.AccountID, EntryID: entryID}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

func LoadAccount(ctx context.Context, es ports.AccountEventStore, accountID ids.ID) (*domain.Account, error) {
	fromVer := int64(0)
	var snap *eventstore.Snapshot
	var snapState *domain.Snapshot
	var err error
	snap, err = es.LoadLatestSnapshot(ctx, accountID.String())
	if err != nil {
		return nil, err
	}
	if snap != nil {
		var s domain.Snapshot
		if err := json.Unmarshal(snap.Data, &s); err != nil {
			return nil, err
		}
		snapState = &s
		fromVer = snap.Version
	}
	events, err := es.Read(ctx, accountID.String(), fromVer, 10000)
	if err != nil {
		return nil, err
	}
	var des []domain.Event
	for _, se := range events {
		ev, err := UnmarshalAccountEvent(se.Type, se.Data)
		if err != nil {
			return nil, err
		}
		des = append(des, ev)
	}
	if snapState != nil {
		return domain.RehydrateFromSnapshot(*snapState, des)
	}
	return domain.Rehydrate(des)
}

func saveSnapshot(ctx context.Context, es ports.AccountEventStore, ag *domain.Account, aggregateID ids.ID, version int64) error {
	s := ag.Snapshot()
	s.Version = version
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return es.SaveSnapshot(ctx, eventstore.Snapshot{
		AggregateID:   aggregateID.String(),
		AggregateType: "account",
		Version:       version,
		CreatedAt:     time.Now().UTC(),
		Data:          b,
	})
}

func UnmarshalAccountEvent(typ string, data []byte) (domain.Event, error) {
	switch typ {
	case "account.opened.v1":
		var e domain.AccountOpened
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "account.frozen.v1":
		var e domain.AccountFrozen
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "account.closed.v1":
		var e domain.AccountClosed
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "account.money_deposited.v1":
		var e domain.MoneyDeposited
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	case "account.money_withdrawn.v1":
		var e domain.MoneyWithdrawn
		if err := json.Unmarshal(data, &e); err != nil {
			return nil, err
		}
		return e, nil
	default:
		return nil, fmt.Errorf("unknown account event type: %s", typ)
	}
}
