package postgres

import (
	"context"

	espg "fincore/pkg/eventstore/postgres"
	obpg "fincore/pkg/outbox/postgres"
	"fincore/services/ledger-service/application/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type UnitOfWork struct {
	pool *pgxpool.Pool
}

func NewUnitOfWork(pool *pgxpool.Pool) *UnitOfWork {
	return &UnitOfWork{pool: pool}
}

func (u *UnitOfWork) WithTx(ctx context.Context, fn func(ctx context.Context, es ports.LedgerEventStore, bal ports.BalanceRepository, idem ports.IdempotencyRepository, ob ports.OutboxStore) error) error {
	tx, err := u.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	es := espg.New(tx)
	bal := NewBalanceRepo(tx)
	idem := NewIdempotencyRepo(tx)
	ob := obpg.New(tx)

	if err := fn(ctx, es, bal, idem, ob); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (u *UnitOfWork) LedgerStore() ports.LedgerEventStore {
	return espg.NewPool(u.pool)
}

func (u *UnitOfWork) Balance() ports.BalanceRepository {
	return NewBalanceRepo(u.pool)
}

var _ ports.UnitOfWork = (*UnitOfWork)(nil)
