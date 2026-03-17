package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// TenantRepository handles tenant database operations without GORM
type TenantRepository struct {
	db *DBConnection
}

// NewTenantRepository creates a new tenant repository
func NewTenantRepository(db *DBConnection) *TenantRepository {
	return &TenantRepository{db: db}
}

// CreateTenant creates a new tenant record
func (tr *TenantRepository) CreateTenant(tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	now := time.Now()
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = now
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = now
	}

	_, err := tr.db.Exec(query,
		tenant.ID,
		tenant.TenantID,
		tenant.TenantDB,
		tenant.Email,
		tenant.Username,
		tenant.PasswordHash,
		tenant.Provider,
		tenant.ProviderID,
		tenant.Avatar,
		tenant.Name,
		tenant.Source,
		tenant.Status,
		tenant.LastLogin,
		tenant.CreatedAt,
		tenant.UpdatedAt,
		tenant.TenantDomain,
	)

	return err
}

// GetTenantByEmail retrieves a tenant by email (case-insensitive)
func (tr *TenantRepository) GetTenantByEmail(email string) (*models.Tenant, error) {
	query := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		WHERE LOWER(email) = LOWER($1)
	`

	tenant := &models.Tenant{}
	var lastLogin sql.NullTime
	var username, providerID, avatar sql.NullString

	err := tr.db.QueryRow(query, email).Scan(
		&tenant.ID,
		&tenant.TenantID,
		&tenant.TenantDB,
		&tenant.Email,
		&username,
		&tenant.PasswordHash,
		&tenant.Provider,
		&providerID,
		&avatar,
		&tenant.Name,
		&tenant.Source,
		&tenant.Status,
		&lastLogin,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.TenantDomain,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if username.Valid {
		tenant.Username = &username.String
	}
	if providerID.Valid {
		tenant.ProviderID = &providerID.String
	}
	if avatar.Valid {
		tenant.Avatar = &avatar.String
	}
	if lastLogin.Valid {
		tenant.LastLogin = &lastLogin.Time
	}

	return tenant, nil
}

// GetTenantByTenantID retrieves a tenant by tenant_id
func (tr *TenantRepository) GetTenantByTenantID(tenantID string) (*models.Tenant, error) {
	query := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		WHERE tenant_id = $1
	`

	tenant := &models.Tenant{}
	var lastLogin sql.NullTime
	var username, providerID, avatar sql.NullString

	err := tr.db.QueryRow(query, tenantID).Scan(
		&tenant.ID,
		&tenant.TenantID,
		&tenant.TenantDB,
		&tenant.Email,
		&username,
		&tenant.PasswordHash,
		&tenant.Provider,
		&providerID,
		&avatar,
		&tenant.Name,
		&tenant.Source,
		&tenant.Status,
		&lastLogin,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.TenantDomain,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, err
	}

	// Handle nullable fields
	if username.Valid {
		tenant.Username = &username.String
	}
	if providerID.Valid {
		tenant.ProviderID = &providerID.String
	}
	if avatar.Valid {
		tenant.Avatar = &avatar.String
	}
	if lastLogin.Valid {
		tenant.LastLogin = &lastLogin.Time
	}

	return tenant, nil
}

// UpdateTenantDB updates the tenant_db field for a tenant
func (tr *TenantRepository) UpdateTenantDB(tenantID uuid.UUID, dbName string) error {
	query := `
		UPDATE tenants
		SET tenant_db = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := tr.db.Exec(query, dbName, time.Now(), tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// UpdateTenantLogin updates the last_login timestamp for a tenant
func (tr *TenantRepository) UpdateTenantLogin(tenantID uuid.UUID) error {
	query := `
		UPDATE tenants
		SET last_login = $1, updated_at = $2
		WHERE id = $3
	`

	now := time.Now()
	result, err := tr.db.Exec(query, now, now, tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// TenantExists checks if a tenant exists by email (case-insensitive)
func (tr *TenantRepository) TenantExists(email string) (bool, error) {
	// Validate database connection
	if tr.db == nil || tr.db.DB == nil {
		return false, fmt.Errorf("database connection is not initialized")
	}

	// Ensure connection is valid
	if err := tr.db.DB.Ping(); err != nil {
		return false, fmt.Errorf("database connection failed: %w", err)
	}

	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE LOWER(email) = LOWER($1))`
	var exists bool
	err := tr.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check tenant existence: %w", err)
	}
	return exists, nil
}

// TenantExistsByTenantID checks if a tenant exists by tenant_id
func (tr *TenantRepository) TenantExistsByTenantID(tenantID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE tenant_id = $1)`

	var exists bool
	err := tr.db.QueryRow(query, tenantID).Scan(&exists)
	return exists, err
}

// Transaction support

// CreateTenantTx creates a tenant within a transaction
func (tr *TenantRepository) CreateTenantTx(tx *sql.Tx, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	now := time.Now()
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = now
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = now
	}

	_, err := tx.Exec(query,
		tenant.ID,
		tenant.TenantID,
		tenant.TenantDB,
		tenant.Email,
		tenant.Username,
		tenant.PasswordHash,
		tenant.Provider,
		tenant.ProviderID,
		tenant.Avatar,
		tenant.Name,
		tenant.Source,
		tenant.Status,
		tenant.LastLogin,
		tenant.CreatedAt,
		tenant.UpdatedAt,
		tenant.TenantDomain,
	)

	return err
}

// UpdateTenantDBTx updates tenant_db within a transaction
func (tr *TenantRepository) UpdateTenantDBTx(tx *sql.Tx, tenantID uuid.UUID, dbName string) error {
	query := `
		UPDATE tenants
		SET tenant_db = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := tx.Exec(query, dbName, time.Now(), tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// GetAllTenants retrieves all tenants from the database
func (tr *TenantRepository) GetAllTenants() ([]*models.Tenant, error) {
	query := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		ORDER BY created_at DESC
	`

	rows, err := tr.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []*models.Tenant
	for rows.Next() {
		tenant := &models.Tenant{}
		err := rows.Scan(
			&tenant.ID,
			&tenant.TenantID,
			&tenant.TenantDB,
			&tenant.Email,
			&tenant.Username,
			&tenant.PasswordHash,
			&tenant.Provider,
			&tenant.ProviderID,
			&tenant.Avatar,
			&tenant.Name,
			&tenant.Source,
			&tenant.Status,
			&tenant.LastLogin,
			&tenant.CreatedAt,
			&tenant.UpdatedAt,
			&tenant.TenantDomain,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tenant: %w", err)
		}
		tenants = append(tenants, tenant)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return tenants, nil
}

// UpdateTenantStatusTx updates tenant status within a transaction
func (tr *TenantRepository) UpdateTenantStatusTx(tx *sql.Tx, tenantID uuid.UUID, status string) error {
	query := `
		UPDATE tenants
		SET status = $1, updated_at = $2
		WHERE id = $3
	`

	result, err := tx.Exec(query, status, time.Now(), tenantID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("tenant not found")
	}

	return nil
}

// DeleteTenant permanently deletes a tenant and all related data from the master database.
// Note: This does NOT delete the tenant database - call TenantDBService.DropTenantDatabase separately.
func (tr *TenantRepository) DeleteTenant(tenantID uuid.UUID) (map[string]int64, error) {
	deletedCounts := make(map[string]int64)

	tx, err := tr.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Helper function to execute delete and count rows
	execDelete := func(table, query string, args ...interface{}) error {
		result, err := tx.Exec(query, args...)
		if err != nil {
			return fmt.Errorf("failed to delete from %s: %w", table, err)
		}
		if rows, err := result.RowsAffected(); err == nil {
			deletedCounts[table] = rows
		}
		return nil
	}

	// Delete in order of dependencies (child tables first)

	// 1. Delete role_bindings for users in this tenant
	if err := execDelete("role_bindings", "DELETE FROM role_bindings WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 2. Delete role_permissions for roles in this tenant
	if err := execDelete("role_permissions", "DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE tenant_id = $1)", tenantID); err != nil {
		return nil, err
	}

	// 3. Delete roles for this tenant
	if err := execDelete("roles", "DELETE FROM roles WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 4. Delete permissions for this tenant
	if err := execDelete("permissions", "DELETE FROM permissions WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 5. Delete api_scopes for this tenant
	if err := execDelete("api_scopes", "DELETE FROM api_scopes WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 6. Delete totp_secrets for users in this tenant
	if err := execDelete("totp_secrets", "DELETE FROM totp_secrets WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 7. Delete backup_codes for users in this tenant
	if err := execDelete("backup_codes", "DELETE FROM backup_codes WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 8. Delete webauthn_credentials for users in this tenant
	if err := execDelete("webauthn_credentials", "DELETE FROM webauthn_credentials WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 9. Delete sessions for users in this tenant
	if err := execDelete("sessions", "DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE tenant_id = $1)", tenantID); err != nil {
		return nil, err
	}

	// 10. Delete refresh_tokens for users in this tenant
	if err := execDelete("refresh_tokens", "DELETE FROM refresh_tokens WHERE user_id IN (SELECT id FROM users WHERE tenant_id = $1)", tenantID); err != nil {
		return nil, err
	}

	// 11. Delete user_groups for users in this tenant
	if err := execDelete("user_groups", "DELETE FROM user_groups WHERE user_id IN (SELECT id FROM users WHERE tenant_id = $1)", tenantID); err != nil {
		return nil, err
	}

	// 12. Delete oauth_clients for this tenant
	if err := execDelete("oauth_clients", "DELETE FROM oauth_clients WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 13. Delete projects for this tenant
	if err := execDelete("projects", "DELETE FROM projects WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 14. Delete tenant_mappings for this tenant
	if err := execDelete("tenant_mappings", "DELETE FROM tenant_mappings WHERE tenant_id = $1", tenantID.String()); err != nil {
		return nil, err
	}

	// 15. Delete users for this tenant
	if err := execDelete("users", "DELETE FROM users WHERE tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// 16. Finally, delete the tenant record itself
	if err := execDelete("tenants", "DELETE FROM tenants WHERE id = $1 OR tenant_id = $1", tenantID); err != nil {
		return nil, err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return deletedCounts, nil
}
