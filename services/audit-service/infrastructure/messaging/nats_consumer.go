package nats

import (
	"context"
	"encoding/json"
	"log"

	auditv1 "fincore/gen/go/audit/v1"
	repo "fincore/services/audit-service/infrastructure/postgres"
	"github.com/nats-io/nats.go"
)

type AuditConsumer struct {
	nc   *nats.Conn
	repo *repo.AuditRepository
}

func NewAuditConsumer(nc *nats.Conn, repo *repo.AuditRepository) *AuditConsumer {
	return &AuditConsumer{nc: nc, repo: repo}
}

func (c *AuditConsumer) Start(ctx context.Context) error {
	// Subscribe to all security and money movement events.
	// In a real system, we'd use JetStream for durability.
	subjects := []string{
		"auth.user.*",
		"auth.client.*",
		"payment.transfer.*",
		"account.transaction.*",
	}

	for _, sub := range subjects {
		_, err := c.nc.Subscribe(sub, func(msg *nats.Msg) {
			var entry auditv1.AuditLogEntry
			if err := json.Unmarshal(msg.Data, &entry); err != nil {
				log.Printf("failed to unmarshal audit entry: %v", err)
				return
			}

			if err := c.repo.Save(context.Background(), &entry); err != nil {
				log.Printf("failed to save audit log: %v", err)
			}
		})
		if err != nil {
			return err
		}
	}

	log.Printf("AuditConsumer started, listening on %v", subjects)
	return nil
}
