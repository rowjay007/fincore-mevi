package workers

import (
	"context"
	"log"
	"time"

	"fincore/services/ledger-service/application/ports"
)

type LedgerProjectionWorker struct {
	es      ports.LedgerEventStore
	bal     ports.BalanceRepository
	batch   int
	polling time.Duration
}

func NewLedgerProjectionWorker(es ports.LedgerEventStore, bal ports.BalanceRepository) *LedgerProjectionWorker {
	return &LedgerProjectionWorker{
		es:      es,
		bal:     bal,
		batch:   100,
		polling: 1 * time.Second,
	}
}

func (w *LedgerProjectionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.polling)
	defer ticker.Stop()

	var lastSeq int64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			processed, nextSeq, err := w.processNextBatch(ctx, lastSeq)
			if err != nil {
				log.Printf("LedgerProjectionWorker error: %v", err)
				continue
			}
			if processed > 0 {
				lastSeq = nextSeq
			}
		}
	}
}

func (w *LedgerProjectionWorker) processNextBatch(ctx context.Context, lastSeq int64) (int, int64, error) {
	events, nextSeq, err := w.es.ReadAll(ctx, lastSeq, w.batch)
	if err != nil {
		return 0, lastSeq, err
	}

	for _, e := range events {
		if e.AggregateType != "ledger_entry" {
			continue
		}

		// In a real system, we'd use these events to update denormalized balance tables
		// or secondary indices. The current BalanceRepository already updates
		// during the command phase, but this worker ensures eventually consistent
		// read models are kept in sync if we had them.
	}

	return len(events), nextSeq, nil
}
