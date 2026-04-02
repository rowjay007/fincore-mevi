package ports

import "context"

type TransferWorkflowInput struct {
	PaymentID      string
	FromAccountID  string
	ToAccountID    string
	Currency       string
	AmountKobo     int64
	Narration      string
	IdempotencyKey string
}

type TemporalClient interface {
	StartTransferWorkflow(ctx context.Context, in TransferWorkflowInput) (workflowID string, runID string, err error)
}
