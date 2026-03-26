package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"fincore/services/payment-service/application/commands"
	"fincore/services/payment-service/application/ports"
	"fincore/services/payment-service/domain"
)

// TransferSaga coordinates the payment lifecycle across Ledger and Account services.
// In a production environment, this would typically be a state machine or a workflow engine (e.g., Temporal).
// For now, we implement a simplified version triggered by payment events.
type TransferSaga struct {
	uow       ports.UnitOfWork
	authorize *commands.AuthorizePaymentHandler
	settle    *commands.SettlePaymentHandler
	fail      *commands.FailPaymentHandler
	ledger    ports.LedgerClient
	account   ports.AccountClient
}

func NewTransferSaga(
	uow ports.UnitOfWork,
	auth *commands.AuthorizePaymentHandler,
	settle *commands.SettlePaymentHandler,
	fail *commands.FailPaymentHandler,
	ledger ports.LedgerClient,
	account ports.AccountClient,
) *TransferSaga {
	return &TransferSaga{
		uow:       uow,
		authorize: auth,
		settle:    settle,
		fail:      fail,
		ledger:    ledger,
		account:   account,
	}
}

// ProcessEvent handles payment-related events to drive the saga forward.
// This is typically called by an event consumer.
func (s *TransferSaga) ProcessEvent(ctx context.Context, eventType string, payload []byte) error {
	switch eventType {
	case domain.EventPaymentInitiated:
		return s.handlePaymentInitiated(ctx, payload)
	case "ledger.v1.TransactionRecorded": // Example external event
		return s.handleLedgerRecorded(ctx, payload)
	case "account.v1.BalanceUpdated": // Example external event
		return s.handleAccountUpdated(ctx, payload)
	default:
		return nil
	}
}

func (s *TransferSaga) handlePaymentInitiated(ctx context.Context, payload []byte) error {
	var ev domain.PaymentInitiated
	if err := json.Unmarshal(payload, &ev); err != nil {
		return err
	}

	log.Printf("[Saga] Payment %s initiated. Processing dual-entry accounting...", ev.PaymentID)

	// 1. Withdraw from source account
	_, err := s.account.Withdraw(ctx, ev.FromAccountID.String(), ev.Amount, fmt.Sprintf("wdr-%s", ev.PaymentID), ev.Narration)
	if err != nil {
		log.Printf("[Saga] Withdrawal failed for payment %s: %v", ev.PaymentID, err)
		_, err := s.fail.Handle(ctx, commands.FailPayment{
			PaymentID:      ev.PaymentID,
			IdempotencyKey: fmt.Sprintf("fail-wdr-%s", ev.PaymentID),
			Reason:         fmt.Sprintf("withdrawal failed: %v", err),
		})
		return err
	}

	// 2. Post to Ledger (Source Account)
	_, err = s.ledger.PostEntry(ctx, ev.FromAccountID.String(), ev.Amount, "withdrawal", fmt.Sprintf("ldg-wdr-%s", ev.PaymentID), ev.Narration)
	if err != nil {
		log.Printf("[Saga] Ledger withdrawal entry failed for payment %s: %v", ev.PaymentID, err)
		// In a real system, we'd need compensating actions here (Refund withdrawal)
	}

	// 3. Deposit to destination account
	_, err = s.account.Deposit(ctx, ev.ToAccountID.String(), ev.Amount, fmt.Sprintf("dep-%s", ev.PaymentID), ev.Narration)
	if err != nil {
		log.Printf("[Saga] Deposit failed for payment %s: %v", ev.PaymentID, err)
		// Compounding failure - requires manual intervention or sophisticated retry/compensation
		return err
	}

	// 4. Post to Ledger (Destination Account)
	_, err = s.ledger.PostEntry(ctx, ev.ToAccountID.String(), ev.Amount, "deposit", fmt.Sprintf("ldg-dep-%s", ev.PaymentID), ev.Narration)
	if err != nil {
		log.Printf("[Saga] Ledger deposit entry failed for payment %s: %v", ev.PaymentID, err)
	}

	// 5. Authorize and Settle payment
	_, err = s.authorize.Handle(ctx, commands.AuthorizePayment{
		PaymentID:      ev.PaymentID,
		IdempotencyKey: fmt.Sprintf("auth-%s", ev.PaymentID),
	})
	if err != nil {
		return err
	}

	_, err = s.settle.Handle(ctx, commands.SettlePayment{
		PaymentID:      ev.PaymentID,
		IdempotencyKey: fmt.Sprintf("settle-%s", ev.PaymentID),
	})
	return err
}

func (s *TransferSaga) handleLedgerRecorded(ctx context.Context, payload []byte) error {
	// Logic to transition from Authorized to Settled once Ledger confirms
	return nil
}

func (s *TransferSaga) handleAccountUpdated(ctx context.Context, payload []byte) error {
	// Logic for account balance confirmation
	return nil
}
