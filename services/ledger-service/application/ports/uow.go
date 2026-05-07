package ports

import (
	"context"

	"fincore/pkg/eventstore"
	"fincore/pkg/outbox"
)

type LedgerEventStore interface {
	eventstore.Store
	eventstore.SnapshotStore
}

type OutboxStore interface {
	outbox.Store
}

type BalanceRepository interface {
	GetBalanceKobo(ctx context.Context, accountID string, version int) (int64, error)
	ApplyDelta(ctx context.Context, accountID string, deltaKobo int64, version int) error
}

type IdempotencyRepository interface {
	GetByKey(ctx context.Context, key string) (entryID string, ok bool, err error)
	Save(ctx context.Context, key string, entryID string) error
}

type UnitOfWork interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, es LedgerEventStore, bal BalanceRepository, idem IdempotencyRepository, ob OutboxStore) error) error
}
