package postgres

import (
	"context"
	auditv1 "fincore/gen/go/audit/v1"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AuditRepository struct {
	db *pgxpool.Pool
}

func NewAuditRepository(db *pgxpool.Pool) *AuditRepository {
	return &AuditRepository{db: db}
}

func (r *AuditRepository) Save(ctx context.Context, entry *auditv1.AuditLogEntry) error {
	var payload interface{}
	if entry.Payload != nil {
		payload = entry.Payload.AsMap()
	}

	_, err := r.db.Exec(ctx, `
		insert into audit_logs (
			user_id, action, resource_type, resource_id, payload, correlation_id, trace_id, service_name
		) values ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.UserId, entry.Action, entry.ResourceType, entry.ResourceId, payload, entry.CorrelationId, entry.TraceId, entry.ServiceName)
	return err
}

func (r *AuditRepository) List(ctx context.Context, req *auditv1.ListAuditLogsRequest) ([]*auditv1.AuditLogEntry, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := r.db.Query(ctx, `
		select id, user_id, action, resource_type, resource_id, payload, correlation_id, trace_id, service_name, created_at
		from audit_logs
		where ($1 = '' or user_id = $1)
		  and ($2 = '' or resource_type = $2)
		  and ($3 = '' or resource_id = $3)
		order by created_at desc
		limit $4
	`, req.UserId, req.ResourceType, req.ResourceId, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []*auditv1.AuditLogEntry
	for rows.Next() {
		var e auditv1.AuditLogEntry
		var payload map[string]interface{}
		var createdAt time.Time
		err := rows.Scan(
			&e.Id, &e.UserId, &e.Action, &e.ResourceType, &e.ResourceId, &payload, &e.CorrelationId, &e.TraceId, &e.ServiceName, &createdAt,
		)
		if err != nil {
			return nil, err
		}
		if payload != nil {
			p, err := structpb.NewStruct(payload)
			if err == nil {
				e.Payload = p
			}
		}
		e.Timestamp = timestamppb.New(createdAt)
		entries = append(entries, &e)
	}
	return entries, nil
}
