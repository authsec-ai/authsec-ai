package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"

	"github.com/google/uuid"
)

// PostgresAuditRepository implements the AuditRepository interface
type PostgresAuditRepository struct {
	db *sql.DB
}

// NewPostgresAuditRepository creates a new audit repository
func NewPostgresAuditRepository(db *sql.DB) repositories.AuditRepository {
	return &PostgresAuditRepository{db: db}
}

// Create creates a new audit log entry
func (r *PostgresAuditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	query := `
		INSERT INTO audit_icp_logs (id, tenant_id, event_type, workload_id, certificate_id, spiffe_id,
			success, error_message, metadata, ip_address, user_agent, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	if log.ID == "" {
		log.ID = uuid.New().String()
	}
	log.CreatedAt = time.Now()

	// Convert metadata to JSON
	metadataJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		return errors.NewInternalError("Failed to marshal metadata", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		log.ID,
		log.TenantID,
		log.EventType,
		log.WorkloadID,
		log.CertificateID,
		log.SpiffeID,
		log.Success,
		log.ErrorMessage,
		metadataJSON,
		log.IPAddress,
		log.UserAgent,
		log.CreatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to create audit log", err)
	}

	return nil
}

// List retrieves audit logs for a tenant
func (r *PostgresAuditRepository) List(ctx context.Context, tenantID string, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, tenant_id, event_type, workload_id, certificate_id, spiffe_id,
			success, error_message, metadata, ip_address, user_agent, created_at
		FROM audit_icp_logs
		WHERE tenant_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	return r.queryAuditLogs(ctx, query, tenantID, limit, offset)
}

// ListByWorkload retrieves audit logs for a specific workload
func (r *PostgresAuditRepository) ListByWorkload(ctx context.Context, tenantID, workloadID string, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, tenant_id, event_type, workload_id, certificate_id, spiffe_id,
			success, error_message, metadata, ip_address, user_agent, created_at
		FROM audit_icp_logs
		WHERE tenant_id = $1 AND workload_id = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	return r.queryAuditLogs(ctx, query, tenantID, workloadID, limit, offset)
}

// ListByEventType retrieves audit logs by event type
func (r *PostgresAuditRepository) ListByEventType(ctx context.Context, tenantID string, eventType models.AuditEventType, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, tenant_id, event_type, workload_id, certificate_id, spiffe_id,
			success, error_message, metadata, ip_address, user_agent, created_at
		FROM audit_icp_logs
		WHERE tenant_id = $1 AND event_type = $2
		ORDER BY created_at DESC
		LIMIT $3 OFFSET $4
	`

	return r.queryAuditLogs(ctx, query, tenantID, eventType, limit, offset)
}

// ListByDateRange retrieves audit logs within a date range
func (r *PostgresAuditRepository) ListByDateRange(ctx context.Context, tenantID string, from, to time.Time, limit, offset int) ([]*models.AuditLog, error) {
	query := `
		SELECT id, tenant_id, event_type, workload_id, certificate_id, spiffe_id,
			success, error_message, metadata, ip_address, user_agent, created_at
		FROM audit_icp_logs
		WHERE tenant_id = $1 AND created_at BETWEEN $2 AND $3
		ORDER BY created_at DESC
		LIMIT $4 OFFSET $5
	`

	return r.queryAuditLogs(ctx, query, tenantID, from, to, limit, offset)
}

// queryAuditLogs is a helper function to execute audit log queries
func (r *PostgresAuditRepository) queryAuditLogs(ctx context.Context, query string, args ...interface{}) ([]*models.AuditLog, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternalError("Failed to query audit logs", err)
	}
	defer rows.Close()

	var logs []*models.AuditLog
	for rows.Next() {
		log := &models.AuditLog{}
		var metadataJSON []byte

		err := rows.Scan(
			&log.ID,
			&log.TenantID,
			&log.EventType,
			&log.WorkloadID,
			&log.CertificateID,
			&log.SpiffeID,
			&log.Success,
			&log.ErrorMessage,
			&metadataJSON,
			&log.IPAddress,
			&log.UserAgent,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalError("Failed to scan audit log", err)
		}

		// Unmarshal metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &log.Metadata); err != nil {
				return nil, errors.NewInternalError("Failed to unmarshal metadata", err)
			}
		}

		logs = append(logs, log)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.NewInternalError("Error iterating audit logs", err)
	}

	return logs, nil
}
