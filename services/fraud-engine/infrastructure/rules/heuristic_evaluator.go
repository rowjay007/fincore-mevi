package rules

import (
	"context"
	"math/big"

	"fincore/services/fraud-engine/domain"
)

type HeuristicEvaluator struct{}

func NewHeuristicEvaluator() *HeuristicEvaluator {
	return &HeuristicEvaluator{}
}

func (e *HeuristicEvaluator) Evaluate(ctx context.Context, txn *domain.Transaction) (domain.RuleResult, error) {
	score := float32(0.01) // Base "safe" score
	var reasons []string
	decision := "approve"

	// Threshold Rule
	threshold, _ := new(big.Float).SetString("10000.00")
	if txn.Amount.Cmp(threshold) > 0 {
		score += 0.4
		reasons = append(reasons, "high_value_transaction")
	}

	// Device Rule
	if txn.IsNewDevice {
		score += 0.3
		reasons = append(reasons, "new_device_detected")
	}

	if txn.DeviceID == "" {
		score += 0.2
		reasons = append(reasons, "missing_device_id")
	}

	// Decision Logic
	if score >= 0.8 {
		decision = "reject"
	} else if score >= 0.4 {
		decision = "review"
	}

	return domain.RuleResult{
		Score:    score,
		Decision: decision,
		Reasons:  reasons,
		}, nil
}
