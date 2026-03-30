package postgres

import (
	"context"

	espg "fincore/pkg/eventstore/postgres"
	obpg "fincore/pkg/outbox/postgres"
	"fincore/services/account-service/application/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) WithTx(ctx context.Context, fn func(ctx context.Context, es ports.AccountEventStore, ob ports.OutboxStore, proj ports.AccountProjectionRepository) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	es := espg.New(tx)
	obStore := obpg.New(tx)
	proj := NewAccountProjectionRepo(tx)

	if err := fn(ctx, es, obStore, proj); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (u *UnitOfWork) AccountStore() ports.AccountEventStore {
	return espg.NewPool(u.pool)
}

func (u *UnitOfWork) Projection() ports.AccountProjectionRepository {
	return NewAccountProjectionRepo(u.pool)
}

var _ ports.UnitOfWork = (*UnitOfWork)(nil)
