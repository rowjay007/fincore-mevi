package postgres

import (
	"context"
	"time"

	"fincore/pkg/eventstore"

	"github.com/jackc/pgx/v5"
)

type Store struct {
	q pgx.Tx
}

func New(tx pgx.Tx) *Store {
	return &Store{q: tx}
}

func (s *Store) Append(ctx context.Context, events []eventstore.Event) error {
	for _, e := range events {
		_, err := s.q.Exec(ctx, `insert into event_store_events(
			id, aggregate_id, aggregate_type, version, type, occurred_at, data, metadata
		) values ($1,$2,$3,$4,$5,$6,$7,$8)`,
			e.ID, e.AggregateID, e.AggregateType, e.Version, e.Type, e.OccurredAt, e.Data, e.Metadata,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Read(ctx context.Context, aggregateID string, fromVersionExclusive int64, limit int) ([]eventstore.Event, error) {
	if limit <= 0 {
		limit = 1000
	}
	rows, err := s.q.Query(ctx, `select id, aggregate_id, aggregate_type, version, type, occurred_at, data, metadata
		from event_store_events
		where aggregate_id = $1 and version > $2
		order by version asc
		limit $3`, aggregateID, fromVersionExclusive, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []eventstore.Event
	for rows.Next() {
		var e eventstore.Event
		err := rows.Scan(&e.ID, &e.AggregateID, &e.AggregateType, &e.Version, &e.Type, &e.OccurredAt, &e.Data, &e.Metadata)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) SaveSnapshot(ctx context.Context, snap eventstore.Snapshot) error {
	_, err := s.q.Exec(ctx, `insert into event_store_snapshots(
		aggregate_id, aggregate_type, version, created_at, data
	) values ($1,$2,$3,$4,$5)`,
		snap.AggregateID, snap.AggregateType, snap.Version, snap.CreatedAt, snap.Data,
	)
	return err
}

func (s *Store) LoadLatestSnapshot(ctx context.Context, aggregateID string) (*eventstore.Snapshot, error) {
	row := s.q.QueryRow(ctx, `select aggregate_id, aggregate_type, version, created_at, data
		from event_store_snapshots
		where aggregate_id = $1
		order by version desc
		limit 1`, aggregateID)

	var snap eventstore.Snapshot
	var createdAt time.Time
	err := row.Scan(&snap.AggregateID, &snap.AggregateType, &snap.Version, &createdAt, &snap.Data)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	snap.CreatedAt = createdAt
	return &snap, nil
}

var _ eventstore.SnapshotStore = (*Store)(nil)
