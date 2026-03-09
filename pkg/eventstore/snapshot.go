package eventstore

import "time"

type Snapshot struct {
	AggregateID   string
	AggregateType string
	Version       int64
	CreatedAt     time.Time
	Data          []byte
}
