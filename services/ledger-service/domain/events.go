package domain

import (
	"time"

	"fincore/pkg/ids"
)

type Event interface {
	EventType() string
	OccurredAtUTC() time.Time
}

type EntryPosted struct {
	EntryID    ids.ID
	AccountID  ids.ID
	EntryType  EntryType
	AmountKobo int64
	Currency   string
	Narration  string
	OccurredAt time.Time
}

func (e EntryPosted) EventType() string        { return "ledger.entry_posted.v1" }
func (e EntryPosted) OccurredAtUTC() time.Time { return e.OccurredAt }
