package workers

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"fincore/services/account-service/application/ports"
	"fincore/services/account-service/domain"
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

func (w *AccountProjectionWorker) ProcessNextBatch(ctx context.Context, lastSeq int64) (int, int64, error) {
	return w.processNextBatch(ctx, lastSeq)
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

		p, ok, err := w.proj.GetByID(ctx, e.AggregateID)
		if err != nil {
			return 0, lastSeq, err
		}
		if !ok {
			p = ports.AccountProjection{AccountID: e.AggregateID}
		}

		switch e.Type {
		case "account.opened.v1":
			var ev domain.AccountOpened
			if err := json.Unmarshal(e.Data, &ev); err != nil {
				return 0, lastSeq, err
			}
			p.CustomerID = ev.CustomerID.String()
			p.Status = string(domain.StatusActive)
		case "account.frozen.v1":
			p.Status = string(domain.StatusFrozen)
		case "account.closed.v1":
			p.Status = string(domain.StatusClosed)
		case "account.money_deposited.v1":
			// Status/customer unchanged
		case "account.money_withdrawn.v1":
			// Status/customer unchanged
		default:
			continue
		}

		p.Version = e.Version
		if p.CustomerID == "" {
			// Can't project without a customer_id; wait until we see account.opened.v1.
			continue
		}
		if p.Status == "" {
			p.Status = string(domain.StatusActive)
		}
		if err := w.proj.Upsert(ctx, p); err != nil {
			return 0, lastSeq, err
		}
	}

	return len(events), nextSeq, nil
}
