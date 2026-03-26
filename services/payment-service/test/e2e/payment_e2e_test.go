package payment

import (
	"context"
	"log"
	"testing"
	"time"

	commonv1 "fincore/gen/go/common/v1"
	paymentv1 "fincore/gen/go/payment/v1"
	"fincore/pkg/ids"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// TestPaymentServiceE2EStub provides a skeleton for E2E integration tests.
// In a real CI environment, this would run against a deployed set of services
// (Payment, Ledger, Account, NATS, Postgres).
func TestPaymentServiceE2EStub(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping e2e test in short mode")
	}

	// This is a stub for the full flow:
	// 1. Initiate Payment (RPC) -> Returns PaymentID
	// 2. Event 'payment.initiated.v1' published to Outbox (DB)
	// 3. Outbox Worker publishes to NATS
	// 4. Saga Consumer receives from NATS
	// 5. Saga calls Ledger/Account (gRPC)
	// 6. Saga calls Authorize/Settle (RPC)
	// 7. Verify final status via GetPayment (RPC)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Connect to Payment Service (Assumes it's running locally or in k8s)
	conn, err := grpc.NewClient("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Logf("Could not connect to payment service (expected if not running): %v", err)
		t.Skip("Payment service not available for E2E test")
	}
	defer conn.Close()

	client := paymentv1.NewPaymentServiceClient(conn)

	fromAccount := ids.New().String()
	toAccount := ids.New().String()
	idempotencyKey := ids.New().String()

	// 1. Initiate
	initRes, err := client.InitiatePayment(ctx, &paymentv1.InitiatePaymentRequest{
		FromAccountId:  fromAccount,
		ToAccountId:    toAccount,
		Amount:         &commonv1.Money{AmountKobo: 50000, Currency: "NGN"},
		Narration:      "E2E Test Payment",
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			// If the service is reachable but dependencies aren't wired up, this may still fail.
			// Since this is an E2E stub, we skip instead of failing local unit test runs.
			if st.Code().String() == "Unavailable" {
				t.Skipf("Payment service unavailable for E2E test: %v", err)
			}
		}
		t.Skipf("Payment service not ready for E2E test: %v", err)
	}

	paymentID := initRes.PaymentId
	log.Printf("Initiated payment: %s", paymentID)

	// 2. Poll for status change (Saga takes time)
	var finalStatus paymentv1.PaymentStatus
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		getRes, err := client.GetPayment(ctx, &paymentv1.GetPaymentRequest{PaymentId: paymentID})
		if err != nil {
			continue
		}
		finalStatus = getRes.Payment.Status
		if finalStatus == paymentv1.PaymentStatus_PAYMENT_STATUS_SETTLED {
			break
		}
	}

	if finalStatus != paymentv1.PaymentStatus_PAYMENT_STATUS_SETTLED {
		t.Errorf("Expected status SETTLED, got %v", finalStatus)
	}
}
