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

func (s *Store) GetPending(ctx context.Context, limit int) ([]outbox.Message, error) {
	rows, err := s.q.Query(ctx, `select id, topic, key, value, headers from outbox_messages where processed_at is null order by created_at asc limit $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []outbox.Message
	for rows.Next() {
		var m outbox.Message
		if err := rows.Scan(&m.ID, &m.Topic, &m.Key, &m.Value, &m.Headers); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}

func (s *Store) MarkProcessed(ctx context.Context, ids []string) error {
	_, err := s.q.Exec(ctx, `update outbox_messages set processed_at = now() where id = any($1)`, ids)
	return err
}
