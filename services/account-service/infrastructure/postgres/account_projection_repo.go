package postgres

import (
	"context"

	"fincore/services/account-service/application/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Queryer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type AccountProjectionRepo struct {
	q Queryer
}

func NewAccountProjectionRepo(q Queryer) *AccountProjectionRepo {
	return &AccountProjectionRepo{q: q}
}

func (r *AccountProjectionRepo) Upsert(ctx context.Context, p ports.AccountProjection) error {
	_, err := r.q.Exec(ctx, `insert into accounts_projection(account_id, customer_id, status, version, updated_at)
		values ($1,$2,$3,$4, now())
		on conflict (account_id) do update set
			customer_id = excluded.customer_id,
			status = excluded.status,
			version = excluded.version,
			updated_at = now()`,
		p.AccountID, p.CustomerID, p.Status, p.Version,
	)
	return err
}

func (r *AccountProjectionRepo) GetByID(ctx context.Context, accountID string) (ports.AccountProjection, bool, error) {
	row := r.q.QueryRow(ctx, `select account_id, customer_id, status, version from accounts_projection where account_id = $1`, accountID)
	var p ports.AccountProjection
	err := row.Scan(&p.AccountID, &p.CustomerID, &p.Status, &p.Version)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ports.AccountProjection{}, false, nil
		}
		return ports.AccountProjection{}, false, err
	}
	return p, true, nil
}

var _ ports.AccountProjectionRepository = (*AccountProjectionRepo)(nil)
