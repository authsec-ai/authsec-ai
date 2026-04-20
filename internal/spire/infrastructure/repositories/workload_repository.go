package repositories

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
)

// PostgresWorkloadRepository implements the WorkloadRepository interface
type PostgresWorkloadRepository struct {
	db *sql.DB
}

// NewPostgresWorkloadRepository creates a new workload repository
func NewPostgresWorkloadRepository(db *sql.DB) repositories.WorkloadRepository {
	return &PostgresWorkloadRepository{db: db}
}

// GetByID retrieves a workload by ID
func (r *PostgresWorkloadRepository) GetByID(ctx context.Context, tenantID, id string) (*models.Workload, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, selectors, vault_role, status, attestation_type, created_at, updated_at
		FROM workloads
		WHERE id = $1 AND tenant_id = $2
	`

	return r.scanWorkload(ctx, query, id, tenantID)
}

// GetBySpiffeID retrieves a workload by SPIFFE ID
func (r *PostgresWorkloadRepository) GetBySpiffeID(ctx context.Context, tenantID, spiffeID string) (*models.Workload, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, selectors, vault_role, status, attestation_type, created_at, updated_at
		FROM workloads
		WHERE spiffe_id = $1 AND tenant_id = $2
	`

	return r.scanWorkload(ctx, query, spiffeID, tenantID)
}

// Create creates a new workload
func (r *PostgresWorkloadRepository) Create(ctx context.Context, workload *models.Workload) error {
	query := `
		INSERT INTO workloads (id, tenant_id, spiffe_id, selectors, vault_role, status, attestation_type, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	now := time.Now()
	workload.CreatedAt = now
	workload.UpdatedAt = now

	selectorsJSON, err := json.Marshal(workload.Selectors)
	if err != nil {
		return errors.NewInternalError("Failed to marshal selectors", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		workload.ID,
		workload.TenantID,
		workload.SpiffeID,
		selectorsJSON,
		workload.VaultRole,
		workload.Status,
		workload.AttestationType,
		workload.CreatedAt,
		workload.UpdatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to create workload", err)
	}

	return nil
}

// Update updates an existing workload
func (r *PostgresWorkloadRepository) Update(ctx context.Context, workload *models.Workload) error {
	query := `
		UPDATE workloads
		SET selectors = $3, vault_role = $4, status = $5, attestation_type = $6, updated_at = $7
		WHERE id = $1 AND tenant_id = $2
	`

	workload.UpdatedAt = time.Now()

	selectorsJSON, err := json.Marshal(workload.Selectors)
	if err != nil {
		return errors.NewInternalError("Failed to marshal selectors", err)
	}

	result, err := r.db.ExecContext(ctx, query,
		workload.ID,
		workload.TenantID,
		selectorsJSON,
		workload.VaultRole,
		workload.Status,
		workload.AttestationType,
		workload.UpdatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to update workload", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Workload not found", nil)
	}

	return nil
}

// Delete deletes a workload
func (r *PostgresWorkloadRepository) Delete(ctx context.Context, tenantID, id string) error {
	query := `DELETE FROM workloads WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id, tenantID)
	if err != nil {
		return errors.NewInternalError("Failed to delete workload", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Workload not found", nil)
	}

	return nil
}

// ListByTenant retrieves all workloads for a tenant
func (r *PostgresWorkloadRepository) ListByTenant(ctx context.Context, tenantID string) ([]*models.Workload, error) {
	query := `
		SELECT id, tenant_id, spiffe_id, selectors, vault_role, status, attestation_type, created_at, updated_at
		FROM workloads
		WHERE tenant_id = $1
		ORDER BY created_at DESC
	`

	return r.queryWorkloads(ctx, query, tenantID)
}

// FindBySelectors finds workloads matching the given selectors
func (r *PostgresWorkloadRepository) FindBySelectors(ctx context.Context, tenantID string, selectors map[string]string) ([]*models.Workload, error) {
	// This is a simplified implementation
	// In production, you'd want more sophisticated JSONB querying
	query := `
		SELECT id, tenant_id, spiffe_id, selectors, vault_role, status, attestation_type, created_at, updated_at
		FROM workloads
		WHERE tenant_id = $1
	`

	workloads, err := r.queryWorkloads(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}

	// Filter by selectors in memory (not ideal for large datasets)
	var filtered []*models.Workload
	for _, w := range workloads {
		if matchesSelectors(w.Selectors, selectors) {
			filtered = append(filtered, w)
		}
	}

	return filtered, nil
}

// scanWorkload scans a single workload
func (r *PostgresWorkloadRepository) scanWorkload(ctx context.Context, query string, args ...interface{}) (*models.Workload, error) {
	workload := &models.Workload{}
	var selectorsJSON []byte

	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&workload.ID,
		&workload.TenantID,
		&workload.SpiffeID,
		&selectorsJSON,
		&workload.VaultRole,
		&workload.Status,
		&workload.AttestationType,
		&workload.CreatedAt,
		&workload.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Workload not found", err)
	}
	if err != nil {
		return nil, errors.NewInternalError("Failed to get workload", err)
	}

	if err := json.Unmarshal(selectorsJSON, &workload.Selectors); err != nil {
		return nil, errors.NewInternalError("Failed to unmarshal selectors", err)
	}

	return workload, nil
}

// queryWorkloads queries multiple workloads
func (r *PostgresWorkloadRepository) queryWorkloads(ctx context.Context, query string, args ...interface{}) ([]*models.Workload, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, errors.NewInternalError("Failed to query workloads", err)
	}
	defer rows.Close()

	var workloads []*models.Workload
	for rows.Next() {
		workload := &models.Workload{}
		var selectorsJSON []byte

		err := rows.Scan(
			&workload.ID,
			&workload.TenantID,
			&workload.SpiffeID,
			&selectorsJSON,
			&workload.VaultRole,
			&workload.Status,
			&workload.AttestationType,
			&workload.CreatedAt,
			&workload.UpdatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalError("Failed to scan workload", err)
		}

		if err := json.Unmarshal(selectorsJSON, &workload.Selectors); err != nil {
			return nil, errors.NewInternalError("Failed to unmarshal selectors", err)
		}

		workloads = append(workloads, workload)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.NewInternalError("Error iterating workloads", err)
	}

	return workloads, nil
}

// matchesSelectors checks if workload selectors match the given selectors
func matchesSelectors(workloadSelectors, querySelectors map[string]string) bool {
	for key, value := range querySelectors {
		if workloadSelectors[key] != value {
			return false
		}
	}
	return true
}
