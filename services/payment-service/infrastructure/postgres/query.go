package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"fincore/services/payment-service/application/ports"
)

type PaymentQuery struct {
	pool *pgxpool.Pool
}

func NewPaymentQuery(pool *pgxpool.Pool) *PaymentQuery {
	return &PaymentQuery{pool: pool}
}

func (q *PaymentQuery) GetByID(ctx context.Context, paymentID string) (ports.PaymentProjection, bool, error) {
	row := q.pool.QueryRow(ctx, `select payment_id, from_account_id, to_account_id, currency, amount_kobo, narration, status, version from payments_projection where payment_id = $1`, paymentID)
	var p ports.PaymentProjection
	if err := row.Scan(&p.PaymentID, &p.FromAccountID, &p.ToAccountID, &p.Currency, &p.AmountKobo, &p.Narration, &p.Status, &p.Version); err != nil {
		if err == pgx.ErrNoRows {
			return ports.PaymentProjection{}, false, nil
		}
		return ports.PaymentProjection{}, false, err
	}
	return p, true, nil
}
