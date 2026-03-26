package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"fincore/services/payment-service/application/ports"

	"github.com/nats-io/nats.go"
)

type OutboxWorker struct {
	store     ports.OutboxStore
	nc        *nats.Conn
	interval  time.Duration
	batchSize int
}

func NewOutboxWorker(store ports.OutboxStore, nc *nats.Conn) *OutboxWorker {
	return &OutboxWorker{
		store:     store,
		nc:        nc,
		interval:  500 * time.Millisecond,
		batchSize: 50,
	}
}

func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Printf("[Outbox] Worker started with interval %v", w.interval)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processBatch(ctx); err != nil {
				log.Printf("[Outbox] Error processing batch: %v", err)
			}
		}
	}
}

func (w *OutboxWorker) processBatch(ctx context.Context) error {
	msgs, err := w.store.GetPending(ctx, w.batchSize)
	if err != nil {
		return fmt.Errorf("get pending: %w", err)
	}
	if len(msgs) == 0 {
		return nil
	}

	var processedIDs []string
	for _, msg := range msgs {
		err := w.nc.PublishMsg(&nats.Msg{
			Subject: msg.Topic,
			Data:    msg.Value,
			Header: func() nats.Header {
				h := make(nats.Header)
				for k, v := range msg.Headers {
					h.Set(k, v)
				}
				return h
			}(),
		})
		if err != nil {
			log.Printf("[Outbox] Failed to publish message %s: %v", msg.ID, err)
			continue
		}
		processedIDs = append(processedIDs, msg.ID)
	}

	if len(processedIDs) > 0 {
		if err := w.store.MarkProcessed(ctx, processedIDs); err != nil {
			return fmt.Errorf("mark processed: %w", err)
		}
		log.Printf("[Outbox] Published %d messages", len(processedIDs))
	}

	return nil
}
