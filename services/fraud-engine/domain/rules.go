package domain

import (
	"context"
	"math/big"
)

type RuleResult struct {
	Score    float32
	Decision string
	Reasons  []string
}

// Transaction represents the core domain entity for fraud evaluation.
type Transaction struct {
	ID          string
	UserID      string
	Amount      *big.Float
	Currency    string
	DeviceID    string
	IPAddress   string
	IsNewDevice bool
}

// Evaluator defines the port for fraud scoring logic.
type Evaluator interface {
	Evaluate(ctx context.Context, txn *Transaction) (RuleResult, error)
}
