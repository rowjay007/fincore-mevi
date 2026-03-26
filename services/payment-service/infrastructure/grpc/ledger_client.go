package grpc

import (
	"context"
	"fmt"

	commonv1 "fincore/gen/go/common/v1"
	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/money"
	"fincore/services/payment-service/application/ports"

	"google.golang.org/grpc"
)

type ledgerClient struct {
	client ledgerv1.LedgerServiceClient
}

func NewLedgerClient(conn *grpc.ClientConn) ports.LedgerClient {
	return &ledgerClient{
		client: ledgerv1.NewLedgerServiceClient(conn),
	}
}

func (c *ledgerClient) PostEntry(ctx context.Context, accountID string, amount money.Money, entryType string, idempotencyKey string, narration string) (string, error) {
	var et ledgerv1.EntryType
	switch entryType {
	case "deposit":
		et = ledgerv1.EntryType_ENTRY_TYPE_DEPOSIT
	case "withdrawal":
		et = ledgerv1.EntryType_ENTRY_TYPE_WITHDRAWAL
	default:
		return "", fmt.Errorf("invalid entry type: %s", entryType)
	}

	req := &ledgerv1.PostEntryRequest{
		IdempotencyKey: idempotencyKey,
		EntryType:      et,
		Account:        &ledgerv1.AccountRef{AccountId: accountID},
		Amount: &commonv1.Money{
			Currency:   string(amount.Currency()),
			AmountKobo: amount.AmountKobo(),
		},
		Narration: narration,
	}

	res, err := c.client.PostEntry(ctx, req)
	if err != nil {
		return "", fmt.Errorf("ledger post entry: %w", err)
	}
	return res.EntryId, nil
}
