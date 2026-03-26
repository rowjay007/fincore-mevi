package postgres

import (
	"context"

	"fincore/services/payment-service/application/ports"

	"github.com/jackc/pgx/v5"
)

type PaymentProjectionRepo struct {
	q pgx.Tx
}

func NewPaymentProjectionRepo(tx pgx.Tx) *PaymentProjectionRepo {
	return &PaymentProjectionRepo{q: tx}
}

func (r *PaymentProjectionRepo) Upsert(ctx context.Context, p ports.PaymentProjection) error {
	_, err := r.q.Exec(ctx, `insert into payments_projection(payment_id, from_account_id, to_account_id, currency, amount_kobo, narration, status, version, updated_at)
		values ($1,$2,$3,$4,$5,$6,$7,$8, now())
		on conflict (payment_id) do update set
			from_account_id = excluded.from_account_id,
			to_account_id = excluded.to_account_id,
			currency = excluded.currency,
			amount_kobo = excluded.amount_kobo,
			narration = excluded.narration,
			status = excluded.status,
			version = excluded.version,
			updated_at = now()`,
		p.PaymentID, p.FromAccountID, p.ToAccountID, p.Currency, p.AmountKobo, p.Narration, p.Status, p.Version,
	)
	return err
}

func (r *PaymentProjectionRepo) GetByID(ctx context.Context, paymentID string) (ports.PaymentProjection, bool, error) {
	row := r.q.QueryRow(ctx, `select payment_id, from_account_id, to_account_id, currency, amount_kobo, narration, status, version from payments_projection where payment_id = $1`, paymentID)
	var p ports.PaymentProjection
	err := row.Scan(&p.PaymentID, &p.FromAccountID, &p.ToAccountID, &p.Currency, &p.AmountKobo, &p.Narration, &p.Status, &p.Version)
	if err != nil {
		if err == pgx.ErrNoRows {
			return ports.PaymentProjection{}, false, nil
		}
		return ports.PaymentProjection{}, false, err
	}
	return p, true, nil
}

var _ ports.PaymentProjectionRepository = (*PaymentProjectionRepo)(nil)
