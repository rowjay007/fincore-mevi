package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"fincore/pkg/outbox/relay"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Relay struct {
	pool      *pgxpool.Pool
	publisher relay.Publisher
	cfg       relay.Config
}

func New(pool *pgxpool.Pool, publisher relay.Publisher, cfg relay.Config) (*Relay, error) {
	if pool == nil {
		return nil, errors.New("pool required")
	}
	if publisher == nil {
		return nil, errors.New("publisher required")
	}
	return &Relay{pool: pool, publisher: publisher, cfg: cfg}, nil
}

func (r *Relay) Run(ctx context.Context) error {
	cfg := r.cfg
	cfg = cfg.WithDefaults()

	t := time.NewTicker(cfg.PollInterval)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			if err := r.tick(ctx, cfg); err != nil {
				return err
			}
		}
	}
}

type row struct {
	id      string
	topic   string
	key     []byte
	value   []byte
	headers map[string]string
}

func (r *Relay) tick(ctx context.Context, cfg relay.Config) error {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	rows, err := tx.Query(ctx, `select id, topic, key, value, headers
		from outbox_messages
		where published_at is null
		order by created_at asc
		limit $1
		for update skip locked`, cfg.BatchSize)
	if err != nil {
		return err
	}
	defer rows.Close()

	var batch []row
	for rows.Next() {
		var rrow row
		var headersJSON []byte
		if err := rows.Scan(&rrow.id, &rrow.topic, &rrow.key, &rrow.value, &headersJSON); err != nil {
			return err
		}
		if len(headersJSON) > 0 {
			_ = json.Unmarshal(headersJSON, &rrow.headers)
		}
		batch = append(batch, rrow)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if len(batch) == 0 {
		return tx.Commit(ctx)
	}

	now := time.Now().UTC()
	for _, m := range batch {
		pctx, cancel := context.WithTimeout(ctx, cfg.PublishTimeout)
		err := r.publisher.Publish(pctx, m.topic, m.key, m.value, m.headers)
		cancel()
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `update outbox_messages set published_at = $1 where id = $2`, now, m.id)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}
