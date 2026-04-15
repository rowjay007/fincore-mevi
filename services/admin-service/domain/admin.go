package domain

import (
	"context"
	"time"
)

type AdminOperation struct {
	ID         string
	OperatorID string
	Action     string
	ResourceID string
	Details    map[string]interface{}
	Status     string // pending, executed, failed
	CreatedAt  time.Time
}

type AdminPort interface {
	Propose(ctx context.Context, op AdminOperation) (string, error)
	Approve(ctx context.Context, opID string, approverID string) error
	SetFeatureFlag(ctx context.Context, key string, enabled bool, rollout float32) error
}
