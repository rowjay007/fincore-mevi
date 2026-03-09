package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type IdempotencyRepo struct {
	q pgx.Tx
}

func NewIdempotencyRepo(tx pgx.Tx) *IdempotencyRepo {
	return &IdempotencyRepo{q: tx}
}

func (r *IdempotencyRepo) GetByKey(ctx context.Context, key string) (entryID string, ok bool, err error) {
	row := r.q.QueryRow(ctx, `select entry_id from ledger_idempotency where key = $1`, key)
	err = row.Scan(&entryID)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", false, nil
		}
		return "", false, err
	}
	return entryID, true, nil
}

func (r *IdempotencyRepo) Save(ctx context.Context, key string, entryID string) error {
	_, err := r.q.Exec(ctx, `insert into ledger_idempotency(key, entry_id) values ($1, $2)`, key, entryID)
	return err
}
