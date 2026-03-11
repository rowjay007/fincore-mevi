package ports

import (
	"context"

	"fincore/pkg/ids"
	"fincore/pkg/money"
)

type LedgerEntryType string

const (
	LedgerEntryTypeDeposit    LedgerEntryType = "DEPOSIT"
	LedgerEntryTypeWithdrawal LedgerEntryType = "WITHDRAWAL"
)

type LedgerClient interface {
	PostEntry(ctx context.Context, idempotencyKey string, accountID ids.ID, typ LedgerEntryType, amount money.Money, narration string) (entryID ids.ID, err error)
	GetBalance(ctx context.Context, accountID ids.ID) (availableBalance money.Money, err error)
}
