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
	"fincore/services/account-service/application/commands"
	"fincore/services/account-service/application/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type stubPublisher struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (p *stubPublisher) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls++
	return p.err
}

func (p *stubPublisher) Calls() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls
}

type stubLedgerClient struct{}

func (l *stubLedgerClient) PostEntry(ctx context.Context, idempotencyKey string, accountID ids.ID, typ ports.LedgerEntryType, amount money.Money, narration string) (ids.ID, error) {
	return ids.New(), nil
}

func (l *stubLedgerClient) GetBalance(ctx context.Context, accountID ids.ID) (money.Money, error) {
	return money.New(0, money.NGN)
}

func testDSNAccount() string {
	if v := os.Getenv("ACCOUNT_TEST_DB_DSN"); v != "" {
		return v
	}
	return os.Getenv("FINCORE_TEST_DB_DSN")
}

func ensureAccountSchema(ctx context.Context, pool *pgxpool.Pool, t *testing.T) {
	t.Helper()

	up1, err := os.ReadFile("../../migrations/000001_init.up.sql")
	if err != nil {
		t.Fatalf("read migration 000001: %v", err)
	}
	up2, err := os.ReadFile("../../migrations/000002_accounts_projection.up.sql")
	if err != nil {
		t.Fatalf("read migration 000002: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up1)); err != nil {
		t.Fatalf("apply migration 000001: %v", err)
	}
	if _, err := pool.Exec(ctx, string(up2)); err != nil {
		t.Fatalf("apply migration 000002: %v", err)
	}

	_, _ = pool.Exec(ctx, `truncate table accounts_projection, outbox_messages, event_store_snapshots, event_store_events`)
}

func TestTransactionalOutbox_AccountService_CommitAndRelayPublish(t *testing.T) {
	dsn := testDSNAccount()
	if dsn == "" {
		t.Skip("set ACCOUNT_TEST_DB_DSN or FINCORE_TEST_DB_DSN to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ensureAccountSchema(ctx, pool, t)

	uow := NewUnitOfWork(pool)
	open := commands.NewOpenAccountHandler(uow)
	ledger := &stubLedgerClient{}
	deposit := commands.NewDepositMoneyHandler(uow, ledger)

	openRes, err := open.Handle(ctx, commands.OpenAccount{CustomerID: ids.New(), IdempotencyKey: "idem-open-1"})
	if err != nil {
		t.Fatalf("open account: %v", err)
	}

	amt, _ := money.New(1000, money.NGN)
	if _, err := deposit.Handle(ctx, commands.DepositMoney{AccountID: openRes.AccountID, Amount: amt, IdempotencyKey: "idem-dep-1"}); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	var outboxCount int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages where published_at is null`).Scan(&outboxCount); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	if outboxCount != 2 {
		t.Fatalf("expected 2 outbox messages, got %d", outboxCount)
	}

	var projExists bool
	if err := pool.QueryRow(ctx, `select exists(select 1 from accounts_projection where account_id = $1)`, openRes.AccountID.String()).Scan(&projExists); err != nil {
		t.Fatalf("check projection: %v", err)
	}
	if !projExists {
		t.Fatalf("expected accounts_projection row")
	}

	pub := &stubPublisher{}
	r, err := relaypg.New(pool, pub, relay.Config{BatchSize: 10, PollInterval: 10 * time.Millisecond, PublishTimeout: 2 * time.Second})
	if err != nil {
		t.Fatalf("new relay: %v", err)
	}

	rctx, rcancel := context.WithCancel(ctx)
	defer rcancel()
	done := make(chan error, 1)
	go func() { done <- r.Run(rctx) }()

	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if pub.Calls() >= 2 {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	rcancel()
	_ = <-done

	if pub.Calls() < 2 {
		t.Fatalf("expected relay publish calls >= 2, got %d", pub.Calls())
	}

	var remaining int
	if err := pool.QueryRow(ctx, `select count(*) from outbox_messages where published_at is null`).Scan(&remaining); err != nil {
		t.Fatalf("count remaining outbox: %v", err)
	}
	if remaining != 0 {
		t.Fatalf("expected 0 unpublished outbox messages after relay run, got %d", remaining)
	}
}

func TestTransactionalOutbox_AccountService_RollbackPersistsNeither(t *testing.T) {
	dsn := testDSNAccount()
	if dsn == "" {
		t.Skip("set ACCOUNT_TEST_DB_DSN or FINCORE_TEST_DB_DSN to run integration tests")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	pool, err := postgres.NewPool(ctx, postgres.Config{DSN: dsn})
	if err != nil {
		t.Fatalf("connect db: %v", err)
	}
	defer pool.Close()

	ensureAccountSchema(ctx, pool, t)

	uow := NewUnitOfWork(pool)

	beforeEvents := 0
	_ = pool.QueryRow(ctx, `select count(*) from event_store_events`).Scan(&beforeEvents)

	err = uow.WithTx(ctx, func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore, proj ports.AccountProjectionRepository) error {
		_ = proj
		ev := []byte(`{"test":"event"}`)
		if err := es.Append(ctx, []eventstore.Event{{
			ID:            ids.New().String(),
			AggregateID:   ids.New().String(),
			AggregateType: "account",
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
