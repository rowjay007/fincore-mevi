package eventstore

import "context"

type Appender interface {
	Append(ctx context.Context, events []Event) error
}

type Reader interface {
	Read(ctx context.Context, aggregateID string, fromVersionExclusive int64, limit int) ([]Event, error)
}

type Store interface {
	Appender
	Reader
}
