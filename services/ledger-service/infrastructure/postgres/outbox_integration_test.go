package postgres

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"
	"time"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/pkg/outbox"
	"fincore/pkg/outbox/relay"
	relaypg "fincore/pkg/outbox/relay/postgres"
	"fincore/pkg/postgres"
	"fincore/services/ledger-service/application/commands"
	"fincore/services/ledger-service/application/ports"
	"fincore/services/ledger-service/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type stubPublisher struct {
	mu    sync.Mutex
	calls int
	idErr error
}

func (p *stubPublisher) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return p.idErr
}

func (p *stubPublisher) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

func testDSNLedger() string {
	if v := os.Getenv("LEDGER_TEST_DB_DSN"); v != "" {
		return v
	}
	return os.Getenv("FINCORE_TEST_DB_DSN")
}

func ensureLedgerSchema(ctx context.Context, pool *pgxpool.Pool, t *testing.T) {
	t.Helper()

	up1, err := os.ReadFile("../../migrations/000001_init.up.sql")
	if err != nil {
		t.Fatalf("read migration 000001: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up1)); err != nil {
		t.Fatalf("apply migration 000001: %v", err)
	}

	_, _ = pool.Exec(ctx, `truncate table outbox_messages, event_store_snapshots, event_store_events, ledger_idempotency, ledger_account_balances`)
}

func TestTransactionalOutbox_LedgerService_CommitAndRelayPublish(t *testing.T) {
	dsn := testDSNLedger()
	if dsn == "" {
		t.Skip("set LEDGER_TEST_DB_DSN or FINCORE_TEST_DB_DSN to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ensureLedgerSchema(ctx, pool, t)

	uow := NewUnitOfWork(pool)
	h := commands.NewPostEntryHandler(uow)

	amt, _ := money.New(1000, money.NGN)
	if _, err := h.Handle(ctx, commands.PostEntry{IdempotencyKey: "idem-1", AccountID: ids.New(), Type: domain.EntryTypeDeposit, Amount: amt, Narration: "n"}); err != nil {
		t.Fatalf("post entry: %v", err)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages where published_at is null`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 outbox message, got %d", outboxCount)
	}

	pub := &stubPublisher{}
	r, err := relaypg.New(pool, pub, relay.Config{BatchSize: 10, PollInterval: 10 * time.Millisecond, PublishTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("new relay: %v", err)
	}

	rctx, rcancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	go func() { done <- r.Run(rctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if pub.Calls() >= 1 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	rcancel()
	<-done

	if pub.Calls() < 1 {
		t.Fatalf("expected relay publish calls >= 1, got %d", pub.Calls())
	}

	var remaining int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages where published_at is null`).Scan(&remaining); err != nil {
		t.Fatalf("count remaining outbox: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected 0 unpublished outbox messages after relay run, got %d", remaining)
	}
}

func TestTransactionalOutbox_LedgerService_RollbackPersistsNeither(t *testing.T) {
	dsn := testDSNLedger()
	if dsn == "" {
		t.Skip("set LEDGER_TEST_DB_DSN or FINCORE_TEST_DB_DSN to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ensureLedgerSchema(ctx, pool, t)

	uow := NewUnitOfWork(pool)

	beforeEvents := 0
	_ = pool.QueryRow(ctx, `select count(*) from event_store_events`).Scan(&beforeEvents)

	err = uow.WithTx(ctx, func(ctx context.Context, es ports.LedgerEventStore, bal ports.BalanceRepository, idem ports.IdempotencyRepository, ob ports.OutboxStore) error {
		_ = bal
		_ = idem
		ev := []byte(`{"test":"event"}`)
		if err := es.Append(ctx, []eventstore.Event{{
			ID:            ids.New().String(),
			AggregateID:   ids.New().String(),
			AggregateType: "ledger_entry",
			Version:       1,
			Type:          "test.v1",
			OccurredAt:    time.Now().UTC(),
			Data:          ev,
		}}); err != nil {
			return err
		}
		if err := ob.Enqueue(ctx, outbox.Message{ID: ids.New().String(), Topic: "t", Value: []byte("v"), CreatedAt: time.Now().UTC()}); err != nil {
			return err
		}
		return errors.New("force rollback")
	})
	if err == nil {
		t.Fatalf("expected rollback error")
	}

	afterEvents := 0
	if err := pool.QueryRow(ctx, `select count(*) from event_store_events`).Scan(&afterEvents); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if afterEvents != beforeEvents {
		t.Fatalf("expected no new events on rollback")
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected no outbox messages on rollback, got %d", outboxCount)
	}
}

func TestLedger_IdempotencyKeyDoesNotDoubleApply(t *testing.T) {
	dsn := testDSNLedger()
	if dsn == "" {
		t.Skip("set LEDGER_TEST_DB_DSN or FINCORE_TEST_DB_DSN to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ensureLedgerSchema(ctx, pool, t)

	uow := NewUnitOfWork(pool)
	h := commands.NewPostEntryHandler(uow)

	acct := ids.New()
	amt, _ := money.New(1000, money.NGN)

	res1, err := h.Handle(ctx, commands.PostEntry{IdempotencyKey: "idem-same-1", AccountID: acct, Type: domain.EntryTypeDeposit, Amount: amt, Narration: "n"})
	if err != nil {
		t.Fatalf("post entry 1: %v", err)
	}
	res2, err := h.Handle(ctx, commands.PostEntry{IdempotencyKey: "idem-same-1", AccountID: acct, Type: domain.EntryTypeDeposit, Amount: amt, Narration: "n"})
	if err != nil {
		t.Fatalf("post entry 2: %v", err)
	}
	if res1.EntryID != res2.EntryID {
		t.Fatalf("expected same entry id on idempotent retry")
	}

	var bal int64
	if err := pool.QueryRow(ctx, `select balance_kobo from ledger_account_balances where account_id = $1`, acct.String()).Scan(&bal); err != nil {
		t.Fatalf("get balance: %v", err)
	}
	if bal != 1000 {
		t.Fatalf("expected balance 1000, got %d", bal)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 1 {
		t.Fatalf("expected 1 outbox message total, got %d", outboxCount)
	}
}

func TestLedger_InsufficientFundsDoesNotPersist(t *testing.T) {
	dsn := testDSNLedger()
	if dsn == "" {
		t.Skip("set LEDGER_TEST_DB_DSN or FINCORE_TEST_DB_DSN to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ensureLedgerSchema(ctx, pool, t)

	uow := NewUnitOfWork(pool)
	h := commands.NewPostEntryHandler(uow)

	acct := ids.New()
	amt, _ := money.New(1000, money.NGN)

	_, err = h.Handle(ctx, commands.PostEntry{IdempotencyKey: "idem-withdraw-1", AccountID: acct, Type: domain.EntryTypeWithdrawal, Amount: amt, Narration: "n"})
	if err == nil {
		t.Fatalf("expected insufficient funds error")
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 0 {
		t.Fatalf("expected 0 outbox messages, got %d", outboxCount)
	}

	var eventsCount int
	if err := pool.QueryRow(ctx, `select count(*) from event_store_events`).Scan(&eventsCount); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if eventsCount != 0 {
		t.Fatalf("expected 0 events, got %d", eventsCount)
	}

	var bal int64
	row := pool.QueryRow(ctx, `select balance_kobo from ledger_account_balances where account_id = $1`, acct.String())
	if err := row.Scan(&bal); err == nil {
		t.Fatalf("expected no balance row")
	}
}
