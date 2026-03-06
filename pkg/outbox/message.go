package outbox

import "time"

type Message struct {
	ID          string
	Topic       string
	Key         []byte
	Value       []byte
	Headers     map[string]string
	CreatedAt   time.Time
	PublishedAt *time.Time
}
