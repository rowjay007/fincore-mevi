package postgres

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type MoneyMovement struct {
	EventType     string
	AmountKobo    int64
	Currency      string
	AccountID     string
	UserID        string
	CorrelationID string
	TraceID       string
	OccurredAt    time.Time
}

type ReportingProjectionRepo struct {
	db *pgxpool.Pool
}

func NewReportingProjectionRepo(db *pgxpool.Pool) *ReportingProjectionRepo {
	return &ReportingProjectionRepo{db: db}
}

func (r *ReportingProjectionRepo) UpsertMovement(ctx context.Context, m MoneyMovement) error {
	_, err := r.db.Exec(ctx, `
		insert into reporting_money_movements (
			event_type, amount_kobo, currency, account_id, user_id, correlation_id, trace_id, occurred_at
		) values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, m.EventType, m.AmountKobo, m.Currency, m.AccountID, m.UserID, m.CorrelationID, m.TraceID, m.OccurredAt)
	return err
}
