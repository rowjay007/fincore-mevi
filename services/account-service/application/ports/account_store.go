package ports

import (
	"context"

	"fincore/pkg/eventstore"
	"fincore/pkg/outbox"
)

type AccountEventStore interface {
	eventstore.Store
}

type OutboxStore interface {
	outbox.Store
}

type UnitOfWork interface {
	WithTx(ctx context.Context, fn func(ctx context.Context, es AccountEventStore, ob OutboxStore) error) error
}
