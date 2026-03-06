package eventstore

import "time"

type Event struct {
	ID            string
	AggregateID   string
	AggregateType string
	Version       int64
	Type          string
	OccurredAt    time.Time
	Data          []byte
	Metadata      []byte
}
