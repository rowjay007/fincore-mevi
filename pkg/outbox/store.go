package outbox

import "context"

type Store interface {
	Enqueue(ctx context.Context, msg Message) error
}
