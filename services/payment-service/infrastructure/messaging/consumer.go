package messaging

import (
	"context"
	"fmt"
	"log"

	"fincore/pkg/messaging/nats"
	"fincore/services/payment-service/application/saga"

	natsio "github.com/nats-io/nats.go"
)

type EventConsumer struct {
	subscriber nats.Subscriber
	saga       *saga.TransferSaga
}

func NewEventConsumer(subscriber nats.Subscriber, saga *saga.TransferSaga) *EventConsumer {
	return &EventConsumer{
		subscriber: subscriber,
		saga:       saga,
	}
}

func (c *EventConsumer) Start(ctx context.Context) error {
	// For Phase 4, we subscribe to payment events.
	// In a real scenario, this would be more dynamic or mapped.
	subjects := []string{"payment.initiated.v1"}
	queueGroup := "payment-service-saga"

	for _, subject := range subjects {
		err := c.subscriber.Subscribe(subject, queueGroup, func(msg *natsio.Msg) {
			log.Printf("[Consumer] Received event on subject %s", subject)

			// Extract event type from header or subject
			// For this implementation, we'll assume the subject is the event type
			err := c.saga.ProcessEvent(ctx, subject, msg.Data)
			if err != nil {
				log.Printf("[Consumer] Error processing event %s: %v", subject, err)
				// In a real system, we'd handle retries or dead-letter queues here
			}
		})
		if err != nil {
			return fmt.Errorf("subscribe to %s: %w", subject, err)
		}
	}

	log.Printf("[Consumer] Event consumer started for subjects: %v", subjects)
	return nil
}
