package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DSN string
}

func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, errors.New("dsn required")
	}
	c, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, err
	}
	c.MaxConns = 30
	c.MinConns = 5
	c.MaxConnLifetime = 30 * time.Minute
	c.MaxConnIdleTime = 5 * time.Minute
	return pgxpool.NewWithConfig(ctx, c)
}
