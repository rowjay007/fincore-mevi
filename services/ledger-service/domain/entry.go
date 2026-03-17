package domain

import (
	"errors"
	"time"

	"fincore/pkg/ids"
	"fincore/pkg/money"
)

type EntryType string

const (
	EntryTypeDeposit    EntryType = "DEPOSIT"
	EntryTypeWithdrawal EntryType = "WITHDRAWAL"
)

type Entry struct {
	id            ids.ID
	accountID     ids.ID
	entryType     EntryType
	amount        money.Money
	narration     string
	occurredAtUTC time.Time
	version       int64
	changes       []Event
}

type Snapshot struct {
	EntryID       ids.ID
	AccountID     ids.ID
	EntryType     EntryType
	AmountKobo    int64
	Currency      string
	Narration     string
	OccurredAtUTC time.Time
	Version       int64
}

func NewEntry(accountID ids.ID, typ EntryType, amt money.Money, narration string, occurredAtUTC time.Time) (*Entry, error) {
	if accountID == "" {
		return nil, errors.New("account id required")
	}
	if err := validateAmount(amt); err != nil {
		return nil, err
	}
	if occurredAtUTC.IsZero() {
		occurredAtUTC = time.Now().UTC()
	}
	e := &Entry{}
	e.raise(EntryPosted{
		EntryID:    ids.New(),
		AccountID:  accountID,
		EntryType:  typ,
		AmountKobo: amt.AmountKobo(),
		Currency:   string(amt.Currency()),
		Narration:  narration,
		OccurredAt: occurredAtUTC,
	})
	return e, nil
}

func (e *Entry) ID() ids.ID               { return e.id }
func (e *Entry) AccountID() ids.ID        { return e.accountID }
func (e *Entry) EntryType() EntryType     { return e.entryType }
func (e *Entry) Amount() money.Money      { return e.amount }
func (e *Entry) OccurredAtUTC() time.Time { return e.occurredAtUTC }
func (e *Entry) Version() int64           { return e.version }

func (e *Entry) PullChanges() []Event {
	out := e.changes
	e.changes = nil
	return out
}

func (e *Entry) Snapshot() Snapshot {
	return Snapshot{
		EntryID:       e.id,
		AccountID:     e.accountID,
		EntryType:     e.entryType,
		AmountKobo:    e.amount.AmountKobo(),
		Currency:      string(e.amount.Currency()),
		Narration:     e.narration,
		OccurredAtUTC: e.occurredAtUTC,
		Version:       e.version,
	}
}

func (e *Entry) raise(ev Event) {
	e.changes = append(e.changes, ev)
	e.apply(ev)
	e.version++
}

func (e *Entry) apply(ev Event) {
	switch v := ev.(type) {
	case EntryPosted:
		e.id = v.EntryID
		e.accountID = v.AccountID
		e.entryType = v.EntryType
		e.amount = money.MustNew(v.AmountKobo, money.Currency(v.Currency))
		e.narration = v.Narration
		e.occurredAtUTC = v.OccurredAt
	}
}

func validateAmount(amt money.Money) error {
	if amt.Currency() != money.NGN {
		return ErrCurrencyNotNGN
	}
	if amt.IsNegativeOrZero() {
		return ErrInvalidAmount
	}
	return nil
}

func Rehydrate(events []Event) (*Entry, error) {
	e := &Entry{}
	for _, ev := range events {
		e.apply(ev)
		e.version++
	}
	return e, nil
}

func RehydrateFromSnapshot(s Snapshot, events []Event) (*Entry, error) {
	amt := money.MustNew(s.AmountKobo, money.Currency(s.Currency))
	e := &Entry{
		id:            s.EntryID,
		accountID:     s.AccountID,
		entryType:     s.EntryType,
		amount:        amt,
		narration:     s.Narration,
		occurredAtUTC: s.OccurredAtUTC,
		version:       s.Version,
	}
	for _, ev := range events {
		e.apply(ev)
		e.version++
	}
	return e, nil
}
