package nats

import (
	"context"
	"encoding/json"
	"log"
	"time"

	repo "fincore/services/audit-service/infrastructure/postgres"
	"github.com/nats-io/nats.go"
)

type MoneyMovementEvent struct {
	EventType     string    `json:"type"`
	AmountKobo    int64     `json:"amount_kobo"`
	Currency      string    `json:"currency"`
	AccountID     string    `json:"account_id"`
	UserID        string    `json:"user_id"`
	CorrelationID string    `json:"correlation_id"`
	TraceID       string    `json:"trace_id"`
	OccurredAt    time.Time `json:"occurred_at"`
}

type ReportingConsumer struct {
	nc   *nats.Conn
	repo *repo.ReportingProjectionRepo
}

func NewReportingConsumer(nc *nats.Conn, repo *repo.ReportingProjectionRepo) *ReportingConsumer {
	return &ReportingConsumer{nc: nc, repo: repo}
}

func (c *ReportingConsumer) Start(ctx context.Context) error {
	// Subscribe to ledger and payment events for money movement reporting.
	subjects := []string{
		"ledger.entry_posted.v1",
		"payment.settled.v1",
	}

	for _, sub := range subjects {
		_, err := c.nc.Subscribe(sub, func(msg *nats.Msg) {
			var ev MoneyMovementEvent
			if err := json.Unmarshal(msg.Data, &ev); err != nil {
				log.Printf("failed to unmarshal money movement event: %v", err)
				return
			}

			m := repo.MoneyMovement{
				EventType:     ev.EventType,
				AmountKobo:    ev.AmountKobo,
				Currency:      ev.Currency,
				AccountID:     ev.AccountID,
				UserID:        ev.UserID,
				CorrelationID: ev.CorrelationID,
				TraceID:       ev.TraceID,
				OccurredAt:    ev.OccurredAt,
			}

			if err := c.repo.UpsertMovement(context.Background(), m); err != nil {
				log.Printf("failed to project money movement: %v", err)
			}
		})
		if err != nil {
			return err
		}
	}

	log.Printf("ReportingConsumer started, listening on %v", subjects)
	return nil
}
