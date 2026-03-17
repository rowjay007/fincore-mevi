package money

import (
	"errors"
)

type Currency string

const (
	NGN Currency = "NGN"
)

type Money struct {
	amountKobo int64
	currency   Currency
}

func New(amountKobo int64, currency Currency) (Money, error) {
	if currency == "" {
		return Money{}, errors.New("currency required")
	}
	return Money{amountKobo: amountKobo, currency: currency}, nil
}

func MustNew(amountKobo int64, currency Currency) Money {
	m, err := New(amountKobo, currency)
	if err != nil {
		panic(err)
	}
	return m
}

func (m Money) AmountKobo() int64  { return m.amountKobo }
func (m Money) Currency() Currency { return m.currency }

func (m Money) Add(x Money) (Money, error) {
	if m.currency != x.currency {
		return Money{}, errors.New("currency mismatch")
	}
	return Money{amountKobo: m.amountKobo + x.amountKobo, currency: m.currency}, nil
}

func (m Money) Sub(x Money) (Money, error) {
	if m.currency != x.currency {
		return Money{}, errors.New("currency mismatch")
	}
	return Money{amountKobo: m.amountKobo - x.amountKobo, currency: m.currency}, nil
}

func (m Money) IsZero() bool           { return m.amountKobo == 0 }
func (m Money) IsNegative() bool       { return m.amountKobo < 0 }
func (m Money) IsNegativeOrZero() bool { return m.amountKobo <= 0 }
