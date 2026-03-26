package postgres

import (
	"context"

	espg "fincore/pkg/eventstore/postgres"
	obpg "fincore/pkg/outbox/postgres"
	"fincore/services/payment-service/application/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) Outbox() ports.OutboxStore {
	return obpg.NewPool(u.pool)
}

func (u *UnitOfWork) WithTx(ctx context.Context, fn func(ctx context.Context, es ports.PaymentEventStore, ob ports.OutboxStore, proj ports.PaymentProjectionRepository) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	es := espg.New(tx)
	ob := obpg.New(tx)
	proj := NewPaymentProjectionRepo(tx)

	if err := fn(ctx, es, ob, proj); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

var _ ports.UnitOfWork = (*UnitOfWork)(nil)
