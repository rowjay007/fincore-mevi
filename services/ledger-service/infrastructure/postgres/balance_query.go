package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BalanceQuery struct {
	pool *pgxpool.Pool
}

func NewBalanceQuery(pool *pgxpool.Pool) *BalanceQuery {
	return &BalanceQuery{pool: pool}
}

func (q *BalanceQuery) GetBalanceKobo(ctx context.Context, accountID string) (int64, error) {
	row := q.pool.QueryRow(ctx, `select balance_kobo from ledger_account_balances where account_id = $1`, accountID)
	var bal int64
	err := row.Scan(&bal)
	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return bal, nil
}
