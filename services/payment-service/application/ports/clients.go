package ports

import (
	"context"
	"fincore/pkg/money"
)

type LedgerClient interface {
	PostEntry(ctx context.Context, accountID string, amount money.Money, entryType string, idempotencyKey string, narration string) (string, error)
}

type AccountClient interface {
	Deposit(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error)
	Withdraw(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error)
}
