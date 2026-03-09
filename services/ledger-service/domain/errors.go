package domain

import "errors"

var (
	ErrInvalidAmount     = errors.New("invalid amount")
	ErrCurrencyNotNGN    = errors.New("currency must be NGN")
	ErrInsufficientFunds = errors.New("insufficient funds")
)
