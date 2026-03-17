package domain

import (
	"time"

	"fincore/pkg/ids"
)

type Event interface {
	EventType() string
	OccurredAtUTC() time.Time
}

type AccountOpened struct {
	AccountID  ids.ID
	CustomerID ids.ID
	OccurredAt time.Time
}

func (e AccountOpened) EventType() string        { return "account.opened.v1" }
func (e AccountOpened) OccurredAtUTC() time.Time { return e.OccurredAt }

type AccountFrozen struct {
	AccountID  ids.ID
	OccurredAt time.Time
}

func (e AccountFrozen) EventType() string        { return "account.frozen.v1" }
func (e AccountFrozen) OccurredAtUTC() time.Time { return e.OccurredAt }

type AccountClosed struct {
	AccountID  ids.ID
	OccurredAt time.Time
}

func (e AccountClosed) EventType() string        { return "account.closed.v1" }
func (e AccountClosed) OccurredAtUTC() time.Time { return e.OccurredAt }
