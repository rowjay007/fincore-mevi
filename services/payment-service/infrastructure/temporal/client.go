package temporal

import (
	"context"
	"fmt"

	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/application/workflows"

	"go.temporal.io/sdk/client"
)

type Client struct {
	c         client.Client
	taskQueue string
}

func NewClient(c client.Client, taskQueue string) *Client {
	return &Client{c: c, taskQueue: taskQueue}
}

func (t *Client) StartTransferWorkflow(ctx context.Context, in ports.TransferWorkflowInput) (string, string, error) {
	workflowID := fmt.Sprintf("payment-transfer-%s", in.PaymentID)

	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: t.taskQueue,
	}

	run, err := t.c.ExecuteWorkflow(ctx, opts, workflows.TransferWorkflowName, workflows.TransferWorkflowInput{
		PaymentID:      in.PaymentID,
		FromAccountID:  in.FromAccountID,
		ToAccountID:    in.ToAccountID,
		Currency:       in.Currency,
		AmountKobo:     in.AmountKobo,
		Narration:      in.Narration,
		IdempotencyKey: in.IdempotencyKey,
	})
	if err != nil {
		return "", "", err
	}
	return run.GetID(), run.GetRunID(), nil
}

var _ ports.TemporalClient = (*Client)(nil)
