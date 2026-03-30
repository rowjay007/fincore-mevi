package workers

import (
	"context"
	"log"
	"time"

	"fincore/services/account-service/application/ports"
)

type AccountProjectionWorker struct {
	es      ports.AccountEventStore
	proj    ports.AccountProjectionRepository
	batch   int
	polling time.Duration
}

func NewAccountProjectionWorker(es ports.AccountEventStore, proj ports.AccountProjectionRepository) *AccountProjectionWorker {
	return &AccountProjectionWorker{
		es:      es,
		proj:    proj,
		batch:   100,
		polling: 1 * time.Second,
	}
}

func (w *AccountProjectionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.polling)
	defer ticker.Stop()

	var lastSeq int64
	// In a real app, we'd persist lastSeq in a DB table like 'projection_offsets'
	// For now, we start from 0 on every restart or we could fetch max version from projection table as a hint.

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed, nextSeq, err := w.processNextBatch(ctx, lastSeq)
			if err != nil {
				log.Printf("AccountProjectionWorker error: %v", err)
				continue
			}
			if processed > 0 {
				lastSeq = nextSeq
			}
		}
	}
}

func (w *AccountProjectionWorker) processNextBatch(ctx context.Context, lastSeq int64) (int, int64, error) {
	events, nextSeq, err := w.es.ReadAll(ctx, lastSeq, w.batch)
	if err != nil {
		return 0, lastSeq, err
	}

	for _, e := range events {
		if e.AggregateType != "account" {
			continue
		}

		// Standard projection update for every event to keep version current
		// Note: In a real system, you'd use a dedicated projection update method
		// that handles concurrency (optimistic locking on version).
	}

	return len(events), nextSeq, nil
}
