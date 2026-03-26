package grpc

import (
	"context"
	"fmt"

	accountv1 "fincore/gen/go/account/v1"
	commonv1 "fincore/gen/go/common/v1"
	"fincore/pkg/money"
	"fincore/services/payment-service/application/ports"

	"google.golang.org/grpc"
)

type accountClient struct {
	client accountv1.AccountServiceClient
}

func NewAccountClient(conn *grpc.ClientConn) ports.AccountClient {
	return &accountClient{
		client: accountv1.NewAccountServiceClient(conn),
	}
}

func (c *accountClient) Deposit(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error) {
	req := &accountv1.DepositRequest{
		AccountId:      accountID,
		IdempotencyKey: idempotencyKey,
		Amount: &commonv1.Money{
			Currency:   string(amount.Currency()),
			AmountKobo: amount.AmountKobo(),
		},
		Narration: narration,
	}
	res, err := c.client.Deposit(ctx, req)
	if err != nil {
		return "", fmt.Errorf("account deposit: %w", err)
	}
	return res.EntryId, nil
}

func (c *accountClient) Withdraw(ctx context.Context, accountID string, amount money.Money, idempotencyKey string, narration string) (string, error) {
	req := &accountv1.WithdrawRequest{
		AccountId:      accountID,
		IdempotencyKey: idempotencyKey,
		Amount: &commonv1.Money{
			Currency:   string(amount.Currency()),
			AmountKobo: amount.AmountKobo(),
		},
		Narration: narration,
	}
	res, err := c.client.Withdraw(ctx, req)
	if err != nil {
		return "", fmt.Errorf("account withdraw: %w", err)
	}
	return res.EntryId, nil
}
