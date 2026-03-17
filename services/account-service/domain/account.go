package domain

import (
	"errors"
	"time"

	"fincore/pkg/ids"
	"fincore/pkg/money"
)

type Status string

const (
	StatusPending Status = "PENDING"
	StatusActive  Status = "ACTIVE"
	StatusFrozen  Status = "FROZEN"
	StatusClosed  Status = "CLOSED"
)

var (
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrAccountNotActive  = errors.New("account not active")
	ErrAccountFrozen     = errors.New("account frozen")
	ErrAccountClosed     = errors.New("account closed")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrCurrencyNotNGN    = errors.New("currency must be NGN")
)

type Account struct {
	id         ids.ID
	customerID ids.ID
	status     Status
	version    int64
	changes    []Event
}

type Snapshot struct {
	AccountID  ids.ID
	CustomerID ids.ID
	Status     Status
	Version    int64
}

func NewAccount(customerID ids.ID) (*Account, error) {
	if customerID == "" {
		return nil, errors.New("customer id required")
	}
	a := &Account{}
	a.raise(AccountOpened{
		AccountID:  ids.New(),
		CustomerID: customerID,
		OccurredAt: time.Now().UTC(),
	})
	return a, nil
}

func (a *Account) ID() ids.ID         { return a.id }
func (a *Account) CustomerID() ids.ID { return a.customerID }
func (a *Account) Status() Status     { return a.status }
func (a *Account) Version() int64     { return a.version }

func (a *Account) Freeze() error {
	if a.status == StatusClosed {
		return ErrAccountClosed
	}
	if a.status != StatusActive {
		return ErrAccountNotActive
	}
	a.raise(AccountFrozen{AccountID: a.id, OccurredAt: time.Now().UTC()})
	return nil
}

func (a *Account) Close() error {
	if a.status == StatusClosed {
		return nil
	}
	if a.status != StatusActive {
		return ErrAccountNotActive
	}
	a.raise(AccountClosed{AccountID: a.id, OccurredAt: time.Now().UTC()})
	return nil
}

func ValidateNGNAmountPositive(amt money.Money) error {
	if amt.Currency() != money.NGN {
		return ErrCurrencyNotNGN
	}
	if amt.IsNegativeOrZero() {
		return ErrInvalidAmount
	}
	return nil
}

func (a *Account) PullChanges() []Event {
	out := a.changes
	a.changes = nil
	return out
}

func (a *Account) Snapshot() Snapshot {
	return Snapshot{AccountID: a.id, CustomerID: a.customerID, Status: a.status, Version: a.version}
}

func (a *Account) raise(e Event) {
	a.changes = append(a.changes, e)
	a.apply(e)
	a.version++
}

func (a *Account) apply(e Event) {
	switch ev := e.(type) {
	case AccountOpened:
		a.id = ev.AccountID
		a.customerID = ev.CustomerID
		a.status = StatusActive
	case AccountFrozen:
		a.status = StatusFrozen
	case AccountClosed:
		a.status = StatusClosed
	case MoneyDeposited:
		// balance is sourced from ledger-service; this event is kept for audit trail
	case MoneyWithdrawn:
		// balance is sourced from ledger-service; this event is kept for audit trail
	}
}

func Rehydrate(events []Event) (*Account, error) {
	a := &Account{}
	for _, e := range events {
		a.apply(e)
		a.version++
	}
	return a, nil
}

func RehydrateFromSnapshot(s Snapshot, events []Event) (*Account, error) {
	a := &Account{id: s.AccountID, customerID: s.CustomerID, status: s.Status, version: s.Version}
	for _, e := range events {
		a.apply(e)
		a.version++
	}
	return a, nil
}
