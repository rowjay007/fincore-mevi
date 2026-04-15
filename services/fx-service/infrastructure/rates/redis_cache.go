package rates

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateCache struct {
	client *redis.Client
}

func NewRateCache(client *redis.Client) *RateCache {
	return &RateCache{client: client}
}

func (c *RateCache) Get(ctx context.Context, base, target string) (string, bool, error) {
	key := fmt.Sprintf("fx:rate:%s:%s", base, target)
	val, err := c.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return val, true, nil
}

func (c *RateCache) Set(ctx context.Context, base, target string, rate string, ttl time.Duration) error {
	key := fmt.Sprintf("fx:rate:%s:%s", base, target)
	return c.client.Set(ctx, key, rate, ttl).Err()
}
