package workflows

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
	"go.temporal.io/sdk/workflow"
)

func TestTransferWorkflow_Success(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(TransferWorkflow, workflow.RegisterOptions{Name: TransferWorkflowName})

	var withdrawCalls int32
	var ledgerWdrCalls int32
	var depositCalls int32
	var ledgerDepCalls int32
	var authCalls int32
	var settleCalls int32
	var refundCalls int32
	var failCalls int32

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&withdrawCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "Withdraw"})

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&ledgerWdrCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "LedgerPostWithdraw"})

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&depositCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "Deposit"})

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&ledgerDepCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "LedgerPostDeposit"})

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&refundCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "Refund"})

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&authCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "AuthorizePayment"})

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&settleCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "SettlePayment"})

	env.RegisterActivityWithOptions(func(ctx context.Context, paymentID string, reason string) error {
		atomic.AddInt32(&failCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "FailPayment"})

	env.ExecuteWorkflow(TransferWorkflowName, TransferWorkflowInput{
		PaymentID:      "pay-1",
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Currency:       "NGN",
		AmountKobo:     100,
		Narration:      "n",
		IdempotencyKey: "idem",
	})

	if err := env.GetWorkflowError(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if atomic.LoadInt32(&withdrawCalls) != 1 {
		t.Fatalf("expected Withdraw called once")
	}
	if atomic.LoadInt32(&ledgerWdrCalls) != 1 {
		t.Fatalf("expected LedgerPostWithdraw called once")
	}
	if atomic.LoadInt32(&depositCalls) != 1 {
		t.Fatalf("expected Deposit called once")
	}
	if atomic.LoadInt32(&ledgerDepCalls) != 1 {
		t.Fatalf("expected LedgerPostDeposit called once")
	}
	if atomic.LoadInt32(&authCalls) != 1 {
		t.Fatalf("expected AuthorizePayment called once")
	}
	if atomic.LoadInt32(&settleCalls) != 1 {
		t.Fatalf("expected SettlePayment called once")
	}
	if atomic.LoadInt32(&refundCalls) != 0 {
		t.Fatalf("expected Refund not called")
	}
	if atomic.LoadInt32(&failCalls) != 0 {
		t.Fatalf("expected FailPayment not called")
	}
}

func TestTransferWorkflow_DepositFails_RefundsAndFailsPayment(t *testing.T) {
	var suite testsuite.WorkflowTestSuite
	env := suite.NewTestWorkflowEnvironment()

	env.RegisterWorkflowWithOptions(TransferWorkflow, workflow.RegisterOptions{Name: TransferWorkflowName})

	var refundCalls int32
	var failCalls int32
	var settleCalls int32

	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error { return nil }, activity.RegisterOptions{Name: "Withdraw"})
	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error { return nil }, activity.RegisterOptions{Name: "LedgerPostWithdraw"})
	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		return errors.New("deposit failed")
	}, activity.RegisterOptions{Name: "Deposit"})
	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error { return nil }, activity.RegisterOptions{Name: "LedgerPostDeposit"})
	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&refundCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "Refund"})
	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error { return nil }, activity.RegisterOptions{Name: "AuthorizePayment"})
	env.RegisterActivityWithOptions(func(ctx context.Context, in TransferWorkflowInput) error {
		atomic.AddInt32(&settleCalls, 1)
		return nil
	}, activity.RegisterOptions{Name: "SettlePayment"})
	env.RegisterActivityWithOptions(func(ctx context.Context, paymentID string, reason string) error {
		atomic.AddInt32(&failCalls, 1)
		if paymentID == "" {
			return errors.New("paymentID empty")
		}
		if reason == "" {
			return errors.New("reason empty")
		}
		return nil
	}, activity.RegisterOptions{Name: "FailPayment"})

	env.ExecuteWorkflow(TransferWorkflowName, TransferWorkflowInput{
		PaymentID:      "pay-2",
		FromAccountID:  "acc-1",
		ToAccountID:    "acc-2",
		Currency:       "NGN",
		AmountKobo:     100,
		Narration:      "n",
		IdempotencyKey: "idem",
	})

	if err := env.GetWorkflowError(); err == nil {
		t.Fatalf("expected error, got nil")
	}

	if atomic.LoadInt32(&refundCalls) != 1 {
		t.Fatalf("expected Refund called once")
	}
	if atomic.LoadInt32(&failCalls) != 1 {
		t.Fatalf("expected FailPayment called once")
	}
	if atomic.LoadInt32(&settleCalls) != 0 {
		t.Fatalf("expected SettlePayment not called")
	}
}
