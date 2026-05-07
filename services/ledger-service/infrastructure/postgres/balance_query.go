package postgres

import (
	"context"
	"errors"
	"time"

	"fincore/pkg/security"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BalanceQuery struct {
	pool *pgxpool.Pool
}

func NewBalanceQuery(pool *pgxpool.Pool) *BalanceQuery {
	return &BalanceQuery{pool: pool}
}

func (q *BalanceQuery) GetBalanceKobo(ctx context.Context, accountID string, version int) (int64, error) {
	requiredLSN, _ := ctx.Value(security.LSNContextKey).(uint64)

	if requiredLSN != 0 {
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()

		timeout := time.After(2 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-timeout:
				return 0, errors.New("timeout waiting for causal consistency")
			case <-ticker.C:
				var currentLSN int64
				err := q.pool.QueryRow(ctx, `select sequence from event_store_events order by sequence desc limit 1`).Scan(&currentLSN)
				if err != nil && err != pgx.ErrNoRows {
					return 0, err
				}
				if uint64(currentLSN) >= requiredLSN {
					goto READY
				}
			}
		}
	}

READY:
	row := q.pool.QueryRow(ctx, `select balance_kobo from ledger_account_balances where account_id = $1 and projection_version = $2`, accountID, version)
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
