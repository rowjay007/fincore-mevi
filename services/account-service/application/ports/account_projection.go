package ports

import (
	"context"
)

type AccountProjection struct {
	AccountID  string
	CustomerID string
	Status     string
	Version    int64
}

type AccountProjectionRepository interface {
	Upsert(ctx context.Context, p AccountProjection) error
	GetByID(ctx context.Context, accountID string) (AccountProjection, bool, error)
}
