package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"time"

	auditv1 "fincore/gen/go/audit/v1"

	"github.com/jackc/pgx/v5"
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
	// 1. Fetch the hash of the latest entry to create the Merkle Chain using the monotonic sequence
	var lastHash string
	err := r.db.QueryRow(ctx, "select current_hash from audit_logs order by sequence desc limit 1").Scan(&lastHash)
	if err != nil && err != pgx.ErrNoRows {
		// Only log real errors; empty result is expected for first entry
		log.Printf("failed to fetch last hash: %v", err)
	}

	// 2. Calculate the hash for the current entry
	currentHash := r.CalculateHash(entry, lastHash)

	// 3. Persist with the hash link
	var payload interface{}
	if entry.Payload != nil {
		payload = entry.Payload.AsMap()
	}

	_, err = r.db.Exec(ctx, `
		insert into audit_logs (
			user_id, action, resource_type, resource_id, payload, correlation_id, trace_id, service_name, previous_hash, current_hash
		) values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, entry.UserId, entry.Action, entry.ResourceType, entry.ResourceId, payload, entry.CorrelationId, entry.TraceId, entry.ServiceName, lastHash, currentHash)

	return err
}

func (r *AuditRepository) CalculateHash(entry *auditv1.AuditLogEntry, previousHash string) string {
	payloadBytes := []byte("{}")
	if entry.Payload != nil {
		payloadBytes, _ = json.Marshal(entry.Payload.AsMap())
	}
	dataToHash := fmt.Sprintf("%s|%s|%s|%s|%s|%s",
		entry.UserId, entry.Action, entry.ResourceType, entry.ResourceId, string(payloadBytes), previousHash)

	hash := sha256.Sum256([]byte(dataToHash))
	return hex.EncodeToString(hash[:])
}

func (r *AuditRepository) ValidateIntegrity(ctx context.Context, startID, endID string) (bool, int32, string, error) {
	// Mastery: Full cryptographic chain verification using sequence-based ordering
	rows, err := r.db.Query(ctx, `
		select id, user_id, action, resource_type, resource_id, payload, correlation_id, trace_id, service_name, previous_hash, current_hash
		from audit_logs
		order by sequence asc
	`)
	if err != nil {
		return false, 0, "", err
	}
	defer rows.Close()

	var count int32
	var expectedPrevHash string

	for rows.Next() {
		var e auditv1.AuditLogEntry
		var payload map[string]interface{}
		var prevHash, currHash string

		err := rows.Scan(
			&e.Id, &e.UserId, &e.Action, &e.ResourceType, &e.ResourceId, &payload,
			&e.CorrelationId, &e.TraceId, &e.ServiceName, &prevHash, &currHash,
		)
		if err != nil {
			return false, count, "", err
		}

		if payload != nil {
			p, _ := structpb.NewStruct(payload)
			e.Payload = p
		}

		// Verify chain link
		if prevHash != expectedPrevHash {
			return false, count, e.Id, nil
		}

		// Verify current hash
		calculated := r.CalculateHash(&e, prevHash)
		if calculated != currHash {
			return false, count, e.Id, nil
		}

		expectedPrevHash = currHash
		count++
	}

	return true, count, "", nil
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
		order by sequence desc
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

func (r *AuditRepository) Get(ctx context.Context, id string) (*auditv1.AuditLogEntry, error) {
	var e auditv1.AuditLogEntry
	var payload map[string]interface{}
	var createdAt time.Time

	err := r.db.QueryRow(ctx, `
		select id, user_id, action, resource_type, resource_id, payload, correlation_id, trace_id, service_name, created_at
		from audit_logs
		where id = $1
	`, id).Scan(
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
	return &e, nil
}
