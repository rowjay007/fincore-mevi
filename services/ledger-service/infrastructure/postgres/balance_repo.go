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

func (r *BalanceRepo) GetBalanceKobo(ctx context.Context, accountID string) (int64, error) {
	row := r.q.QueryRow(ctx, `select balance_kobo from ledger_account_balances where account_id = $1`, accountID)
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

func (r *BalanceRepo) ApplyDelta(ctx context.Context, accountID string, deltaKobo int64) error {
	_, err := r.q.Exec(ctx, `insert into ledger_account_balances(account_id, balance_kobo)
		values ($1, $2)
		on conflict (account_id) do update set balance_kobo = ledger_account_balances.balance_kobo + excluded.balance_kobo`,
		accountID, deltaKobo)
	return err
}
