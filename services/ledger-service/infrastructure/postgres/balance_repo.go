package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Queryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type BalanceRepo struct {
	q Queryer
}

func NewBalanceRepo(q Queryer) *BalanceRepo {
	return &BalanceRepo{q: q}
}

func (r *BalanceRepo) GetBalanceKobo(ctx context.Context, accountID string, version int) (int64, error) {
	row := r.q.QueryRow(ctx, "select balance_kobo from ledger_account_balances where account_id = $1 and projection_version = $2", accountID, version)
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

func (r *BalanceRepo) ApplyDelta(ctx context.Context, accountID string, deltaKobo int64, version int) error {
	_, err := r.q.Exec(ctx, `
		insert into ledger_account_balances (account_id, balance_kobo, projection_version, updated_at)
		values ($1, $2, $3, now())
		on conflict (account_id, projection_version) do update
		set balance_kobo = ledger_account_balances.balance_kobo + $2,
		    updated_at = now()
	`, accountID, deltaKobo, version)
	return err
}
