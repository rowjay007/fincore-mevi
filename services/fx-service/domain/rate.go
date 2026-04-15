package domain

import (
	"fmt"
	"math/big"
	"time"
)

type Rate struct {
	BaseCurrency   string
	TargetCurrency string
	Value          *big.Float
	FetchedAt      time.Time
}

func NewRate(base, target string, val string) (*Rate, error) {
	v, ok := new(big.Float).SetPrec(256).SetString(val)
	if !ok {
		return nil, fmt.Errorf("invalid rate format: %s", val)
	}
	return &Rate{
		BaseCurrency:   base,
		TargetCurrency: target,
		Value:          v,
		FetchedAt:      time.Now(),
	}, nil
}

func (r *Rate) Convert(amount *big.Float) *big.Float {
	return new(big.Float).SetPrec(256).Mul(amount, r.Value)
}
