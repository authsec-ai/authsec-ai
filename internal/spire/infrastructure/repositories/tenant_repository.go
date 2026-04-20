package repositories

import (
	"context"
	"database/sql"
	"time"

	"github.com/authsec-ai/authsec/internal/spire/domain/models"
	"github.com/authsec-ai/authsec/internal/spire/domain/repositories"
	"github.com/authsec-ai/authsec/internal/spire/errors"
)

// PostgresTenantRepository implements the TenantRepository interface
type PostgresTenantRepository struct {
	db *sql.DB
}

// NewPostgresTenantRepository creates a new tenant repository
func NewPostgresTenantRepository(db *sql.DB) repositories.TenantRepository {
	return &PostgresTenantRepository{db: db}
}

// GetByID retrieves a tenant by ID (queries by tenant_id UUID from JWT)
func (r *PostgresTenantRepository) GetByID(ctx context.Context, id string) (*models.Tenant, error) {
	query := `
		SELECT
			tenant_id::text,
			name,
			COALESCE(vault_mount, tenant_domain) as vault_mount,
			status,
			created_at,
			updated_at
		FROM tenants
		WHERE tenant_id = $1::uuid AND status = 'active'
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.VaultMount,
		&tenant.Status,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Tenant not found", err)
	}
	if err != nil {
		return nil, errors.NewInternalError("Failed to get tenant", err)
	}

	return tenant, nil
}

// GetByDomain retrieves a tenant by domain name
func (r *PostgresTenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	query := `
		SELECT
			tenant_id::text,
			name,
			COALESCE(vault_mount, tenant_domain) as vault_mount,
			status,
			created_at,
			updated_at
		FROM tenants
		WHERE tenant_domain = $1 AND status = 'active'
	`

	tenant := &models.Tenant{}
	err := r.db.QueryRowContext(ctx, query, domain).Scan(
		&tenant.ID,
		&tenant.Name,
		&tenant.VaultMount,
		&tenant.Status,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, errors.NewNotFoundError("Tenant not found", err)
	}
	if err != nil {
		return nil, errors.NewInternalError("Failed to get tenant", err)
	}

	return tenant, nil
}

// Create creates a new tenant
func (r *PostgresTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, name, vault_mount, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	now := time.Now()
	tenant.CreatedAt = now
	tenant.UpdatedAt = now

	if tenant.Status == "" {
		tenant.Status = "active"
	}

	_, err := r.db.ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.VaultMount,
		//tenant.VaultNamespace,
		tenant.Status,
		tenant.CreatedAt,
		tenant.UpdatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to create tenant", err)
	}

	return nil
}

// Update updates an existing tenant
func (r *PostgresTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	query := `
		UPDATE tenants
		SET name = $2, vault_mount = $3,status = $4, updated_at = $5
		WHERE id = $1
	`

	tenant.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		tenant.ID,
		tenant.Name,
		tenant.VaultMount,
		//tenant.VaultNamespace,
		tenant.Status,
		tenant.UpdatedAt,
	)

	if err != nil {
		return errors.NewInternalError("Failed to update tenant", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Tenant not found", nil)
	}

	return nil
}

// Delete soft deletes a tenant
func (r *PostgresTenantRepository) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE tenants
		SET status = 'deleted', updated_at = $2
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id, time.Now())
	if err != nil {
		return errors.NewInternalError("Failed to delete tenant", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return errors.NewInternalError("Failed to get affected rows", err)
	}

	if rows == 0 {
		return errors.NewNotFoundError("Tenant not found", nil)
	}

	return nil
}

// List retrieves all active tenants
func (r *PostgresTenantRepository) List(ctx context.Context) ([]*models.Tenant, error) {
	query := `
		SELECT id, name, vault_mount, status, created_at, updated_at
		FROM tenants
		WHERE status = 'active'
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, errors.NewInternalError("Failed to list tenants", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant := &models.Tenant{}
		err := rows.Scan(
			&tenant.ID,
			&tenant.Name,
			&tenant.VaultMount,
			//&tenant.VaultNamespace,
			&tenant.Status,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
		)
		if err != nil {
			return nil, errors.NewInternalError("Failed to scan tenant", err)
		}
		tenants = append(tenants, tenant)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.NewInternalError("Error iterating tenants", err)
	}

	return tenants, nil
}
