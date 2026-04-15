package domain

import (
	"context"
)

type Notification struct {
	ID         string
	UserID     string
	TemplateID string
	Channel    string // email, sms, push, webhook
	Data       map[string]interface{}
}

type NotificationStatus struct {
	ID     string
	Status string // queued, sent, failed
	Error  string
}

type NotificationPort interface {
	Send(ctx context.Context, n Notification) (string, error)
	GetStatus(ctx context.Context, id string) (NotificationStatus, error)
}
