package domain

import (
	"errors"
	"strings"
	"time"

	"fincore/pkg/ids"
	"fincore/pkg/money"
)

type Status string

const (
	StatusInitiated  Status = "initiated"
	StatusAuthorized Status = "authorized"
	StatusSettled    Status = "settled"
	StatusFailed     Status = "failed"

	EventPaymentInitiated  = "payment.initiated.v1"
	EventPaymentAuthorized = "payment.authorized.v1"
	EventPaymentSettled    = "payment.settled.v1"
	EventPaymentFailed     = "payment.failed.v1"
)

type PaymentInitiated struct {
	PaymentID     ids.ID      `json:"payment_id"`
	FromAccountID ids.ID      `json:"from_account_id"`
	ToAccountID   ids.ID      `json:"to_account_id"`
	Amount        money.Money `json:"amount"`
	Narration     string      `json:"narration"`
	OccurredAt    time.Time   `json:"occurred_at"`
}

func (e PaymentInitiated) EventType() string        { return EventPaymentInitiated }
func (e PaymentInitiated) OccurredAtUTC() time.Time { return e.OccurredAt.UTC() }

type PaymentAuthorized struct {
	PaymentID  ids.ID    `json:"payment_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e PaymentAuthorized) EventType() string        { return EventPaymentAuthorized }
func (e PaymentAuthorized) OccurredAtUTC() time.Time { return e.OccurredAt.UTC() }

type PaymentSettled struct {
	PaymentID  ids.ID    `json:"payment_id"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e PaymentSettled) EventType() string        { return EventPaymentSettled }
func (e PaymentSettled) OccurredAtUTC() time.Time { return e.OccurredAt.UTC() }

type PaymentFailed struct {
	PaymentID  ids.ID    `json:"payment_id"`
	Reason     string    `json:"reason"`
	OccurredAt time.Time `json:"occurred_at"`
}

func (e PaymentFailed) EventType() string        { return EventPaymentFailed }
func (e PaymentFailed) OccurredAtUTC() time.Time { return e.OccurredAt.UTC() }

type Payment struct {
	id            ids.ID
	fromAccountID ids.ID
	toAccountID   ids.ID
	amount        money.Money
	narration     string
	status        Status
	version       int64
	changes       []any
}

func NewPayment(fromAccountID, toAccountID ids.ID, amount money.Money, narration string) (*Payment, error) {
	if strings.TrimSpace(fromAccountID.String()) == "" {
		return nil, errors.New("from_account_id required")
	}
	if strings.TrimSpace(toAccountID.String()) == "" {
		return nil, errors.New("to_account_id required")
	}
	if fromAccountID == toAccountID {
		return nil, errors.New("from_account_id and to_account_id must differ")
	}
	if err := validateAmount(amount); err != nil {
		return nil, err
	}
	p := &Payment{id: ids.New(), fromAccountID: fromAccountID, toAccountID: toAccountID, amount: amount, narration: narration, status: StatusInitiated}
	p.raise(PaymentInitiated{PaymentID: p.id, FromAccountID: fromAccountID, ToAccountID: toAccountID, Amount: amount, Narration: narration, OccurredAt: time.Now().UTC()})
	return p, nil
}

func (p *Payment) ID() ids.ID     { return p.id }
func (p *Payment) Status() Status { return p.status }
func (p *Payment) Version() int64 { return p.version }

func (p *Payment) Authorize() error {
	if p.status != StatusInitiated {
		return errors.New("payment not in initiated state")
	}
	p.raise(PaymentAuthorized{PaymentID: p.id, OccurredAt: time.Now().UTC()})
	return nil
}

func (p *Payment) Settle() error {
	if p.status != StatusAuthorized {
		return errors.New("payment not in authorized state")
	}
	p.raise(PaymentSettled{PaymentID: p.id, OccurredAt: time.Now().UTC()})
	return nil
}

func (p *Payment) Fail(reason string) error {
	if p.status == StatusSettled {
		return errors.New("cannot fail a settled payment")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "failed"
	}
	p.raise(PaymentFailed{PaymentID: p.id, Reason: reason, OccurredAt: time.Now().UTC()})
	return nil
}

func (p *Payment) Apply(e any) error {
	switch ev := e.(type) {
	case PaymentInitiated:
		p.id = ev.PaymentID
		p.fromAccountID = ev.FromAccountID
		p.toAccountID = ev.ToAccountID
		p.amount = ev.Amount
		p.narration = ev.Narration
		p.status = StatusInitiated
		p.version++
		return nil
	case PaymentAuthorized:
		p.status = StatusAuthorized
		p.version++
		return nil
	case PaymentSettled:
		p.status = StatusSettled
		p.version++
		return nil
	case PaymentFailed:
		p.status = StatusFailed
		p.version++
		return nil
	default:
		return errors.New("unknown event")
	}
}

func (p *Payment) PullChanges() []any {
	out := p.changes
	p.changes = nil
	return out
}

func (p *Payment) raise(e any) {
	_ = p.Apply(e)
	p.changes = append(p.changes, e)
}

func validateAmount(m money.Money) error {
	if m.Currency() != money.NGN {
		return errors.New("unsupported currency")
	}
	if m.AmountKobo() <= 0 {
		return errors.New("amount must be positive")
	}
	return nil
}
