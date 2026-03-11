package grpc

import (
	"context"
	"errors"

	commonv1 "fincore/gen/go/common/v1"
	ledgerv1 "fincore/gen/go/ledger/v1"
	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/services/account-service/application/ports"

	"google.golang.org/grpc"
)

type LedgerClient struct {
	c ledgerv1.LedgerServiceClient
}

func NewLedgerClient(c ledgerv1.LedgerServiceClient) (*LedgerClient, error) {
	if c == nil {
		return nil, errors.New("ledger grpc client required")
	}
	return &LedgerClient{c: c}, nil
}

func (l *LedgerClient) PostEntry(ctx context.Context, idempotencyKey string, accountID ids.ID, typ ports.LedgerEntryType, amount money.Money, narration string) (ids.ID, error) {
	var et ledgerv1.EntryType
	switch typ {
	case ports.LedgerEntryTypeDeposit:
		et = ledgerv1.EntryType_ENTRY_TYPE_DEPOSIT
	case ports.LedgerEntryTypeWithdrawal:
		et = ledgerv1.EntryType_ENTRY_TYPE_WITHDRAWAL
	default:
		return "", errors.New("unsupported entry type")
	}
	resp, err := l.c.PostEntry(ctx, &ledgerv1.PostEntryRequest{
		IdempotencyKey: idempotencyKey,
		EntryType:      et,
		Account:        &ledgerv1.AccountRef{AccountId: accountID.String()},
		Amount:         &commonv1.Money{Currency: string(amount.Currency()), AmountKobo: amount.AmountKobo()},
		Narration:      narration,
	}, []grpc.CallOption{}...)
	if err != nil {
		return "", err
	}
	return ids.ID(resp.EntryId), nil
}

func (l *LedgerClient) GetBalance(ctx context.Context, accountID ids.ID) (money.Money, error) {
	resp, err := l.c.GetBalance(ctx, &ledgerv1.GetBalanceRequest{Account: &ledgerv1.AccountRef{AccountId: accountID.String()}}, []grpc.CallOption{}...)
	if err != nil {
		return money.Money{}, err
	}
	if resp.AvailableBalance == nil {
		return money.New(0, money.NGN)
	}
	return money.New(resp.AvailableBalance.AmountKobo, money.Currency(resp.AvailableBalance.Currency))
}

var _ ports.LedgerClient = (*LedgerClient)(nil)
