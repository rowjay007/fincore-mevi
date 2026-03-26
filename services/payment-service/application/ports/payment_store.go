package ports

import (
	"context"

	"fincore/pkg/eventstore"
	"fincore/pkg/outbox"
)

type PaymentEventStore interface {
	eventstore.Store
	eventstore.SnapshotStore
}

type OutboxStore interface {
	outbox.Store
	GetPending(ctx context.Context, limit int) ([]outbox.Message, error)
	MarkProcessed(ctx context.Context, ids []string) error
}

type PaymentProjection struct {
	PaymentID     string
	FromAccountID string
	ToAccountID   string
	Currency      string
	AmountKobo    int64
	Narration     string
	Status        string
	Version       int64
}

type PaymentProjectionRepository interface {
	Upsert(ctx context.Context, p PaymentProjection) error
	GetByID(ctx context.Context, paymentID string) (PaymentProjection, bool, error)
}

type UnitOfWork interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, es PaymentEventStore, ob OutboxStore, proj PaymentProjectionRepository) error) error
	Outbox() OutboxStore
}
