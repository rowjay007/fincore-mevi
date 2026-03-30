package workers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"fincore/services/ledger-service/application/ports"
	"fincore/services/ledger-service/domain"
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

func (w *LedgerProjectionWorker) ProcessNextBatch(ctx context.Context, lastSeq int64) (int, int64, error) {
	return w.processNextBatch(ctx, lastSeq)
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

		if e.Type != "ledger.entry_posted.v1" {
			continue
		}

		var ev domain.EntryPosted
		if err := json.Unmarshal(e.Data, &ev); err != nil {
			return 0, lastSeq, err
		}

		delta := ev.AmountKobo
		if ev.EntryType == domain.EntryTypeWithdrawal {
			delta = -delta
		}
		if err := w.bal.ApplyDelta(ctx, ev.AccountID.String(), delta); err != nil {
			return 0, lastSeq, err
		}
	}

	return len(events), nextSeq, nil
}
