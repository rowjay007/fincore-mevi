package postgres

import (
	"context"

	"fincore/pkg/outbox"
	"github.com/jackc/pgx/v5"
)

type Store struct {
	q pgx.Tx
}

func New(tx pgx.Tx) *Store {
	return &Store{q: tx}
}

func (s *Store) Enqueue(ctx context.Context, msg outbox.Message) error {
	_, err := s.q.Exec(ctx, `insert into outbox_messages(
		id, topic, key, value, headers
	) values ($1,$2,$3,$4,$5)`, msg.ID, msg.Topic, msg.Key, msg.Value, msg.Headers)
	return err
}
