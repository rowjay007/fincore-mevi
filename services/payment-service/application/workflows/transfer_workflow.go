package workflows

import (
	"context"
	"fmt"
	"time"

	"fincore/pkg/ids"
	"fincore/pkg/money"
	"fincore/services/payment-service/application/commands"
	"fincore/services/payment-service/application/ports"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const TransferWorkflowName = "payment.transfer.v1"

type TransferWorkflowInput struct {
	PaymentID      string
	FromAccountID  string
	ToAccountID    string
	Currency       string
	AmountKobo     int64
	Narration      string
	IdempotencyKey string
}

type TransferActivities struct {
	Ledger    ports.LedgerClient
	Account   ports.AccountClient
	Authorize *commands.AuthorizePaymentHandler
	Settle    *commands.SettlePaymentHandler
	Fail      *commands.FailPaymentHandler
}

func (w *TransferActivities) Withdraw(ctx context.Context, in TransferWorkflowInput) error {
	amt, err := money.New(in.AmountKobo, money.Currency(in.Currency))
	if err != nil {
		return err
	}
	_, err = w.Account.Withdraw(ctx, in.FromAccountID, amt, fmt.Sprintf("wdr-%s", in.PaymentID), in.Narration)
	return err
}

func (w *TransferActivities) LedgerPostWithdraw(ctx context.Context, in TransferWorkflowInput) error {
	amt, err := money.New(in.AmountKobo, money.Currency(in.Currency))
	if err != nil {
		return err
	}
	_, err = w.Ledger.PostEntry(ctx, in.FromAccountID, amt, "withdrawal", fmt.Sprintf("ldg-wdr-%s", in.PaymentID), in.Narration)
	return err
}

func (w *TransferActivities) Deposit(ctx context.Context, in TransferWorkflowInput) error {
	amt, err := money.New(in.AmountKobo, money.Currency(in.Currency))
	if err != nil {
		return err
	}
	_, err = w.Account.Deposit(ctx, in.ToAccountID, amt, fmt.Sprintf("dep-%s", in.PaymentID), in.Narration)
	return err
}

func (w *TransferActivities) LedgerPostDeposit(ctx context.Context, in TransferWorkflowInput) error {
	amt, err := money.New(in.AmountKobo, money.Currency(in.Currency))
	if err != nil {
		return err
	}
	_, err = w.Ledger.PostEntry(ctx, in.ToAccountID, amt, "deposit", fmt.Sprintf("ldg-dep-%s", in.PaymentID), in.Narration)
	return err
}

func (w *TransferActivities) Refund(ctx context.Context, in TransferWorkflowInput) error {
	amt, err := money.New(in.AmountKobo, money.Currency(in.Currency))
	if err != nil {
		return err
	}
	_, err = w.Account.Deposit(ctx, in.FromAccountID, amt, fmt.Sprintf("refund-%s", in.PaymentID), in.Narration)
	return err
}

func (w *TransferActivities) AuthorizePayment(ctx context.Context, in TransferWorkflowInput) error {
	_, err := w.Authorize.Handle(ctx, commands.AuthorizePayment{PaymentID: ids.ID(in.PaymentID), IdempotencyKey: fmt.Sprintf("auth-%s", in.PaymentID)})
	return err
}

func (w *TransferActivities) SettlePayment(ctx context.Context, in TransferWorkflowInput) error {
	_, err := w.Settle.Handle(ctx, commands.SettlePayment{PaymentID: ids.ID(in.PaymentID), IdempotencyKey: fmt.Sprintf("settle-%s", in.PaymentID)})
	return err
}

func (w *TransferActivities) FailPayment(ctx context.Context, paymentID string, reason string) error {
	_, err := w.Fail.Handle(ctx, commands.FailPayment{PaymentID: ids.ID(paymentID), IdempotencyKey: fmt.Sprintf("fail-%s", paymentID), Reason: reason})
	return err
}

func TransferWorkflow(ctx workflow.Context, in TransferWorkflowInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    10 * time.Second,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	recordedWithdraw := false

	exec := func(name string) error {
		return workflow.ExecuteActivity(ctx, name, in).Get(ctx, nil)
	}

	if err := exec("Withdraw"); err != nil {
		_ = workflow.ExecuteActivity(ctx, "FailPayment", in.PaymentID, fmt.Sprintf("withdrawal failed: %v", err)).Get(ctx, nil)
		return err
	}
	recordedWithdraw = true

	if err := exec("LedgerPostWithdraw"); err != nil {
		if recordedWithdraw {
			_ = workflow.ExecuteActivity(ctx, "Refund", in).Get(ctx, nil)
		}
		_ = workflow.ExecuteActivity(ctx, "FailPayment", in.PaymentID, fmt.Sprintf("ledger withdrawal entry failed: %v", err)).Get(ctx, nil)
		return err
	}

	if err := exec("Deposit"); err != nil {
		if recordedWithdraw {
			_ = workflow.ExecuteActivity(ctx, "Refund", in).Get(ctx, nil)
		}
		_ = workflow.ExecuteActivity(ctx, "FailPayment", in.PaymentID, fmt.Sprintf("deposit failed: %v", err)).Get(ctx, nil)
		return err
	}

	if err := exec("LedgerPostDeposit"); err != nil {
		if recordedWithdraw {
			_ = workflow.ExecuteActivity(ctx, "Refund", in).Get(ctx, nil)
		}
		_ = workflow.ExecuteActivity(ctx, "FailPayment", in.PaymentID, fmt.Sprintf("ledger deposit entry failed: %v", err)).Get(ctx, nil)
		return err
	}

	if err := exec("AuthorizePayment"); err != nil {
		if recordedWithdraw {
			_ = workflow.ExecuteActivity(ctx, "Refund", in).Get(ctx, nil)
		}
		_ = workflow.ExecuteActivity(ctx, "FailPayment", in.PaymentID, fmt.Sprintf("authorize failed: %v", err)).Get(ctx, nil)
		return err
	}

	if err := exec("SettlePayment"); err != nil {
		if recordedWithdraw {
			_ = workflow.ExecuteActivity(ctx, "Refund", in).Get(ctx, nil)
		}
		_ = workflow.ExecuteActivity(ctx, "FailPayment", in.PaymentID, fmt.Sprintf("settle failed: %v", err)).Get(ctx, nil)
		return err
	}

	return nil
}

func RegisterTransferWorker(w workerRegistry, acts *TransferActivities) {
	w.RegisterWorkflowWithOptions(TransferWorkflow, workflow.RegisterOptions{Name: TransferWorkflowName})

	w.RegisterActivityWithOptions(acts.Withdraw, activity.RegisterOptions{Name: "Withdraw"})
	w.RegisterActivityWithOptions(acts.LedgerPostWithdraw, activity.RegisterOptions{Name: "LedgerPostWithdraw"})
	w.RegisterActivityWithOptions(acts.Deposit, activity.RegisterOptions{Name: "Deposit"})
	w.RegisterActivityWithOptions(acts.LedgerPostDeposit, activity.RegisterOptions{Name: "LedgerPostDeposit"})
	w.RegisterActivityWithOptions(acts.Refund, activity.RegisterOptions{Name: "Refund"})
	w.RegisterActivityWithOptions(acts.AuthorizePayment, activity.RegisterOptions{Name: "AuthorizePayment"})
	w.RegisterActivityWithOptions(acts.SettlePayment, activity.RegisterOptions{Name: "SettlePayment"})
	w.RegisterActivityWithOptions(acts.FailPayment, activity.RegisterOptions{Name: "FailPayment"})
}

type workerRegistry interface {
	RegisterWorkflowWithOptions(interface{}, workflow.RegisterOptions)
	RegisterActivityWithOptions(interface{}, activity.RegisterOptions)
}
