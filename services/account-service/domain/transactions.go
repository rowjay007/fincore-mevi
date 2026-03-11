package domain

import (
	"time"

	"fincore/pkg/ids"
	"fincore/pkg/money"
)

type MoneyDeposited struct {
	AccountID  ids.ID
	Amount     money.Money
	EntryID    ids.ID
	OccurredAt time.Time
}

func (e MoneyDeposited) EventType() string         { return "account.money_deposited.v1" }
func (e MoneyDeposited) OccurredAtUTC() time.Time { return e.OccurredAt }

type MoneyWithdrawn struct {
	AccountID  ids.ID
	Amount     money.Money
	EntryID    ids.ID
	OccurredAt time.Time
}

func (e MoneyWithdrawn) EventType() string         { return "account.money_withdrawn.v1" }
func (e MoneyWithdrawn) OccurredAtUTC() time.Time { return e.OccurredAt }
