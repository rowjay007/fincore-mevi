package relay

import (
	"context"
	"time"
)

type Publisher interface {
	Publish(ctx context.Context, topic string, key []byte, value []byte, headers map[string]string) error
}

type Config struct {
	BatchSize      int
	PollInterval   time.Duration
	LockTimeout    time.Duration
	PublishTimeout time.Duration
}

func (c Config) WithDefaults() Config {
	if c.BatchSize <= 0 {
		c.BatchSize = 100
	}
	if c.PollInterval <= 0 {
		c.PollInterval = 250 * time.Millisecond
	}
	if c.LockTimeout <= 0 {
		c.LockTimeout = 5 * time.Second
	}
	if c.PublishTimeout <= 0 {
		c.PublishTimeout = 5 * time.Second
	}
	return c
}
