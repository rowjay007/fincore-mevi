package saga

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"fincore/pkg/eventstore"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/pkg/outbox"
	"fincore/services/payment-service/application/commands"
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

type mockLedgerClient struct {
	PostEntryFunc func(ctx context.Context, accountID string, amount money.Money, entryType string, idempotencyKey string, narration string) (string, error)
}

func (m *mockLedgerClient) PostEntry(ctx context.Context, accountID string, amount money.Money, entryType string, idempotencyKey string, narration string) (string, error) {
	if m.PostEntryFunc != nil {
		return m.PostEntryFunc(ctx, accountID, amount, entryType, idempotencyKey, narration)
	}
	return "entry-id", nil
}

type mockAccountClient struct {
	DepositFunc  func(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error)
	WithdrawFunc func(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error)
}

func (m *mockAccountClient) Deposit(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error) {
	if m.DepositFunc != nil {
		return m.DepositFunc(ctx, accountID, amount, idempotencyKey, narration)
	}
	return "dep-id", nil
}

func (m *mockAccountClient) Withdraw(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error) {
	if m.WithdrawFunc != nil {
		return m.WithdrawFunc(ctx, accountID, amount, idempotencyKey, narration)
	}
	return "wdr-id", nil
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

func TestTransferSaga_handlePaymentInitiated(t *testing.T) {
	es := &mockEventStore{events: make(map[string][]eventstore.Event)}
	ob := &mockOutboxStore{}
	proj := &mockProjRepo{data: make(map[string]ports.PaymentProjection)}
	uow := &mockUoW{es: es, ob: ob, proj: proj}

	authH := commands.NewAuthorizePaymentHandler(uow)
	settleH := commands.NewSettlePaymentHandler(uow)
	failH := commands.NewFailPaymentHandler(uow)
	lc := &mockLedgerClient{}
	ac := &mockAccountClient{}

	s := NewTransferSaga(uow, authH, settleH, failH, lc, ac)

	paymentID := ids.New()
	fromID := ids.New()
	toID := ids.New()
	amt := money.MustNew(100, money.NGN)

	ev := domain.PaymentInitiated{
		PaymentID:     paymentID,
		FromAccountID: fromID,
		ToAccountID:   toID,
		Amount:        amt,
		Narration:     "Test Saga",
		OccurredAt:    time.Now().UTC(),
	}
	// Manually add the initiated event so LoadPayment works
	b, _ := json.Marshal(ev)
	es.Append(context.Background(), []eventstore.Event{{
		AggregateID: paymentID.String(),
		Version:     1,
		Data:        b,
		Type:        domain.EventPaymentInitiated,
	}})
	// Manually add projection so GetByID works
	proj.Upsert(context.Background(), ports.PaymentProjection{
		PaymentID: paymentID.String(),
		Status:    string(domain.StatusInitiated),
		Version:   1,
	})

	payload, _ := json.Marshal(ev)
	err := s.ProcessEvent(context.Background(), domain.EventPaymentInitiated, payload)
	if err != nil {
		t.Fatalf("ProcessEvent failed: %v", err)
	}

	// Verify final state in projection
	p, ok, _ := proj.GetByID(context.Background(), paymentID.String())
	if !ok {
		t.Fatal("projection not found")
	}
	if p.Status != string(domain.StatusSettled) {
		t.Errorf("expected status settled, got %s", p.Status)
	}
}
