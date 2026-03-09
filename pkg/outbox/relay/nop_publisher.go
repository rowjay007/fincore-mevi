package relay

import "context"

type NopPublisher struct{}

func (NopPublisher) Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error {
	return nil
}
