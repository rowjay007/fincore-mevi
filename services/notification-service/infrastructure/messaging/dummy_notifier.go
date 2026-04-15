package messaging

import (
	"context"
	"fmt"
	"log"

	"fincore/services/notification-service/domain"
)

type DummyNotifier struct{}

func NewDummyNotifier() *DummyNotifier {
	return &DummyNotifier{}
}

func (n *DummyNotifier) Send(ctx context.Context, notif domain.Notification) (string, error) {
	// Security: In production, this would use AWS SES, Twilio, or FCM.
	// Templates are rendered using html/template before sending.
	log.Printf("NOTIFICATION_INFRA: Sending %s to user %s via %s (Data: %v)", 
		notif.TemplateID, notif.UserID, notif.Channel, notif.Data)
	
	return fmt.Sprintf("notif_%s_%s", notif.Channel, notif.UserID), nil
}

func (n *DummyNotifier) GetStatus(ctx context.Context, id string) (domain.NotificationStatus, error) {
	return domain.NotificationStatus{
		ID:     id,
		Status: "sent",
	}, nil
}

var _ domain.NotificationPort = (*DummyNotifier)(nil)
