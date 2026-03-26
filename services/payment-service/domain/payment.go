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
)

type PaymentInitiated struct {
	PaymentID     ids.ID      `json:"payment_id"`
	FromAccountID ids.ID      `json:"from_account_id"`
	ToAccountID   ids.ID      `json:"to_account_id"`
	Amount        money.Money `json:"amount"`
	Narration     string      `json:"narration"`
	OccurredAt    time.Time   `json:"occurred_at"`
}

func (e PaymentInitiated) EventType() string        { return "payment.initiated.v1" }
func (e PaymentInitiated) OccurredAtUTC() time.Time { return e.OccurredAt.UTC() }

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
