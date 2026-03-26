package saga

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"fincore/pkg/ids"
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
}

func NewTransferSaga(uow ports.UnitOfWork, auth *commands.AuthorizePaymentHandler, settle *commands.SettlePaymentHandler, fail *commands.FailPaymentHandler) *TransferSaga {
	return &TransferSaga{
		uow:       uow,
		authorize: auth,
		settle:    settle,
		fail:      fail,
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

	log.Printf("[Saga] Payment %s initiated. Triggering authorization...", ev.PaymentID)
	
	// In a real saga, we would check external systems here.
	// For Phase 3, we auto-authorize if basic validation passes.
	_, err := s.authorize.Handle(ctx, commands.AuthorizePayment{
		PaymentID:      ids.ID(ev.PaymentID),
		IdempotencyKey: fmt.Sprintf("auth-%s", ev.PaymentID),
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
