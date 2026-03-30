package eventstore

import "context"

type Appender interface {
	Append(ctx context.Context, events []Event) error
}

type Reader interface {
	Read(ctx context.Context, aggregateID string, fromVersionExclusive int64, limit int) ([]Event, error)
	ReadAll(ctx context.Context, fromSequenceExclusive int64, limit int) ([]Event, int64, error)
}

type Store interface {
	Appender
	Reader
}

type SnapshotWriter interface {
	SaveSnapshot(ctx context.Context, s Snapshot) error
}

type SnapshotReader interface {
	LoadLatestSnapshot(ctx context.Context, aggregateID string) (*Snapshot, error)
}

type SnapshotStore interface {
	SnapshotWriter
	SnapshotReader
}
