package commands

import (
	"context"
	"testing"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/pkg/outbox"
	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/domain"
)

type mockUoW struct {
	es   ports.PaymentEventStore
	ob   ports.OutboxStore
	proj ports.PaymentProjectionRepository
}

func (m *mockUoW) WithTx(ctx context.Context, fn func(context.Context, ports.PaymentEventStore, ports.OutboxStore, ports.PaymentProjectionRepository) error) error {
	return fn(ctx, m.es, m.ob, m.proj)
}

func (m *mockUoW) Outbox() ports.OutboxStore {
	return m.ob
}

func TestPaymentLifecycle(t *testing.T) {
	// Simple in-memory mocks for testing the handlers' orchestration
	es := &mockEventStore{events: make(map[string][]eventstore.Event)}
	ob := &mockOutboxStore{}
	proj := &mockProjRepo{data: make(map[string]ports.PaymentProjection)}
	uow := &mockUoW{es: es, ob: ob, proj: proj}

	initiateH := NewInitiatePaymentHandler(uow, nil)
	authH := NewAuthorizePaymentHandler(uow)
	settleH := NewSettlePaymentHandler(uow)

	ctx := context.Background()
	fromID := ids.New()
	toID := ids.New()

	amt := money.MustNew(100, money.NGN)
	// 1. Initiate
	initRes, err := initiateH.Handle(ctx, InitiatePayment{
		FromAccountID:  fromID,
		ToAccountID:    toID,
		Amount:         amt,
		IdempotencyKey: "key1",
	})
	if err != nil {
		t.Fatalf("initiate failed: %v", err)
	}
	if initRes.Status != domain.StatusInitiated {
		t.Errorf("expected initiated, got %v", initRes.Status)
	}
	// Use the generated payment ID
	paymentID := initRes.PaymentID

	// 2. Authorize
	authRes, err := authH.Handle(ctx, AuthorizePayment{
		PaymentID:      paymentID,
		IdempotencyKey: "key2",
	})
	if err != nil {
		t.Fatalf("authorize failed: %v", err)
	}
	if authRes.Status != domain.StatusAuthorized {
		t.Errorf("expected authorized, got %v", authRes.Status)
	}

	// 3. Settle
	settleRes, err := settleH.Handle(ctx, SettlePayment{
		PaymentID:      paymentID,
		IdempotencyKey: "key3",
	})
	if err != nil {
		t.Fatalf("settle failed: %v", err)
	}
	if settleRes.Status != domain.StatusSettled {
		t.Errorf("expected settled, got %v", settleRes.Status)
	}

	// Verify projection state
	p, ok, _ := proj.GetByID(ctx, paymentID.String())
	if !ok || p.Status != string(domain.StatusSettled) {
		t.Errorf("projection status mismatch: %v", p.Status)
	}
}

type mockEventStore struct {
	events map[string][]eventstore.Event
}

func (m *mockEventStore) Append(ctx context.Context, events []eventstore.Event) error {
	for _, e := range events {
		m.events[e.AggregateID] = append(m.events[e.AggregateID], e)
	}
	return nil
}

func (m *mockEventStore) Read(ctx context.Context, aggregateID string, fromVersion int64, limit int) ([]eventstore.Event, error) {
	all := m.events[aggregateID]
	var res []eventstore.Event
	for _, e := range all {
		if e.Version > fromVersion {
			res = append(res, e)
		}
		if limit > 0 && len(res) >= limit {
			break
		}
	}
	return res, nil
}

func (m *mockEventStore) ReadAll(ctx context.Context, fromSequenceExclusive int64, limit int) ([]eventstore.Event, int64, error) {
	var all []eventstore.Event
	for _, evs := range m.events {
		all = append(all, evs...)
	}
	return all, fromSequenceExclusive, nil
}

func (m *mockEventStore) LoadLatestSnapshot(ctx context.Context, aggregateID string) (*eventstore.Snapshot, error) {
	return nil, nil
}

func (m *mockEventStore) SaveSnapshot(ctx context.Context, snap eventstore.Snapshot) error {
	return nil
}

type mockOutboxStore struct {
	messages []outbox.Message
}

func (m *mockOutboxStore) Enqueue(ctx context.Context, msg outbox.Message) error {
	m.messages = append(m.messages, msg)
	return nil
}

func (m *mockOutboxStore) GetPending(ctx context.Context, limit int) ([]outbox.Message, error) {
	return nil, nil
}

func (m *mockOutboxStore) MarkProcessed(ctx context.Context, ids []string) error {
	return nil
}

type mockProjRepo struct {
	data map[string]ports.PaymentProjection
}

func (m *mockProjRepo) Upsert(ctx context.Context, p ports.PaymentProjection) error {
	m.data[p.PaymentID] = p
	return nil
}

func (m *mockProjRepo) GetByID(ctx context.Context, id string) (ports.PaymentProjection, bool, error) {
	p, ok := m.data[id]
	return p, ok, nil
}
