package database

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// AdminTenantRepository handles admin tenant database operations on global DB
type AdminTenantRepository struct {
	db *DBConnection
}

// NewAdminTenantRepository creates a new admin tenant repository
func NewAdminTenantRepository(db *DBConnection) *AdminTenantRepository {
	return &AdminTenantRepository{db: db}
}

// GetAllTenants retrieves all tenants from global database
func (atr *AdminTenantRepository) GetAllTenants() ([]models.Tenant, error) {
	query := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		ORDER BY created_at DESC
	`

	rows, err := atr.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tenants: %w", err)
	}
	defer rows.Close()

	var tenants []models.Tenant
	for rows.Next() {
		var tenant models.Tenant
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

	return tenants, nil
}

// GetTenantByID retrieves a specific tenant by ID
func (atr *AdminTenantRepository) GetTenantByID(tenantID string) (*models.Tenant, error) {
	query := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		WHERE tenant_id = $1
	`

	var tenant models.Tenant
	err := atr.db.QueryRow(query, tenantID).Scan(
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant: %w", err)
	}

	return &tenant, nil
}

// GetTenantByClientID retrieves a tenant using a client_id via tenant_mappings table
func (atr *AdminTenantRepository) GetTenantByClientID(clientID string) (*models.Tenant, error) {
	query := `
		SELECT t.id, t.tenant_id, t.tenant_db, t.email, t.username, t.password_hash,
			t.provider, t.provider_id, t.avatar, t.name, t.source, t.status, t.last_login,
			t.created_at, t.updated_at, t.tenant_domain
		FROM tenants t
		JOIN tenant_mappings tm ON t.id = tm.tenant_id
		WHERE tm.client_id = $1
	`

	var tenant models.Tenant
	err := atr.db.QueryRow(query, clientID).Scan(
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found for client_id: %s", clientID)
		}
		return nil, fmt.Errorf("failed to get tenant for client_id: %w", err)
	}

	return &tenant, nil
}

// CreateTenant creates a new tenant in global database
func (atr *AdminTenantRepository) CreateTenant(tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, tenant_id, tenant_db, email, username, password_hash,
provider, provider_id, avatar, name, source, status, last_login,
created_at, updated_at, tenant_domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	now := time.Now()
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
	if tenant.CreatedAt.IsZero() {
		tenant.CreatedAt = now
	}
	if tenant.UpdatedAt.IsZero() {
		tenant.UpdatedAt = now
	}

	_, err := atr.db.Exec(query,
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

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// UpdateTenant updates an existing tenant
func (atr *AdminTenantRepository) UpdateTenant(tenantID string, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	query := "UPDATE tenants SET "
	args := []interface{}{}
	argCount := 1

	for field, value := range updates {
		query += field + " = $" + fmt.Sprintf("%d", argCount) + ", "
		args = append(args, value)
		argCount++
	}

	query += "updated_at = $" + fmt.Sprintf("%d", argCount)
	args = append(args, time.Now())
	argCount++

	query += " WHERE tenant_id = $" + fmt.Sprintf("%d", argCount)
	args = append(args, tenantID)

	_, err := atr.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update tenant: %w", err)
	}

	return nil
}

// GetTenantUsers retrieves all users for a specific tenant
func (atr *AdminTenantRepository) GetTenantUsers(tenantID string) ([]models.User, error) {
	// This would typically query the tenant's database, but for admin view
	// we might want to show summary info from global DB
	// For now, return empty slice as this would need tenant DB access
	return []models.User{}, nil
}

// GetTenantByDomain retrieves a tenant by its domain
// Supports custom domains via tenant_domains table lookup
func (atr *AdminTenantRepository) GetTenantByDomain(tenantDomain string) (*models.Tenant, error) {
	log.Printf("DEBUG GetTenantByDomain: Looking up domain='%s'", tenantDomain)

	// First try to find tenant via tenant_domains table (supports custom domains)
	query := `
		SELECT t.id, t.tenant_id, t.tenant_db, t.email, t.username, t.password_hash,
			t.provider, t.provider_id, t.avatar, t.name, t.source, t.status, t.last_login,
			t.created_at, t.updated_at, t.tenant_domain
		FROM tenants t
		INNER JOIN tenant_domains td ON t.tenant_id = td.tenant_id
		WHERE LOWER(td.domain) = LOWER($1) AND td.is_verified = true
		LIMIT 1
	`

	var tenant models.Tenant
	err := atr.db.QueryRow(query, tenantDomain).Scan(
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

	// If found via tenant_domains, return it
	if err == nil {
		log.Printf("DEBUG GetTenantByDomain: Found tenant via tenant_domains table: tenant_id=%s, tenant_domain=%s", tenant.TenantID, tenant.TenantDomain)
		return &tenant, nil
	}

	log.Printf("DEBUG GetTenantByDomain: Not found in tenant_domains (error: %v), trying fallback lookup", err)

	// Fallback: try direct lookup in tenants.tenant_domain (legacy/backwards compatibility)
	fallbackQuery := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		WHERE tenant_domain LIKE $1 OR tenant_domain = $2
	`

	// Support both full domain and subdomain prefix
	err = atr.db.QueryRow(fallbackQuery, tenantDomain+"%", tenantDomain).Scan(
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
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("failed to get tenant by domain: %w", err)
	}

	return &tenant, nil
}

// GetTenantByUUID retrieves a tenant by UUID
func (atr *AdminTenantRepository) GetTenantByUUID(tenantID uuid.UUID) (*models.Tenant, error) {
	return atr.GetTenantByID(tenantID.String())
}

// CreateTenantTx creates a new tenant within a transaction
func (atr *AdminTenantRepository) CreateTenantTx(tx *sql.Tx, tenant *models.Tenant) error {
	query := `
		INSERT INTO tenants (id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	now := time.Now()
	if tenant.ID == uuid.Nil {
		tenant.ID = uuid.New()
	}
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

	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	return nil
}

// CreateProjectTx creates a new project within a transaction
func (atr *AdminTenantRepository) CreateProjectTx(tx *sql.Tx, projectID, tenantID, userID uuid.UUID, name string) error {
	query := `
		INSERT INTO projects (id, tenant_id, name, description, user_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	_, err := tx.Exec(query,
		projectID,
		tenantID,
		name,
		"Default project",
		userID,
		true,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

// CreateAdminUserTx creates a new admin user within a transaction
func (atr *AdminTenantRepository) CreateAdminUserTx(tx *sql.Tx, user *models.AdminUser) error {
	query := `
		INSERT INTO users (id, email, username, password_hash, name, tenant_id, project_id,
			client_id, tenant_domain, provider, provider_id, avatar_url, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	now := time.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	_, err := tx.Exec(query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.Name,
		user.TenantID,
		user.ProjectID,
		user.ClientID,
		user.TenantDomain,
		user.Provider,
		user.ProviderID,
		user.AvatarURL,
		user.Active,
		now,
		now,
	)

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	return nil
}

// GetAdminUserByID retrieves an admin user by ID
func (atr *AdminTenantRepository) GetAdminUserByID(userID uuid.UUID) (*models.AdminUser, error) {
	query := `
		SELECT id, email, username, password_hash, name, tenant_id, project_id,
			client_id, tenant_domain, provider, provider_id, avatar_url, active,
			created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.AdminUser
	var tenantID, projectID, clientID sql.NullString
	var providerID, avatarURL sql.NullString

	err := atr.db.QueryRow(query, userID).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&tenantID,
		&projectID,
		&clientID,
		&user.TenantDomain,
		&user.Provider,
		&providerID,
		&avatarURL,
		&user.Active,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("admin user not found")
		}
		return nil, fmt.Errorf("failed to get admin user: %w", err)
	}

	if tenantID.Valid {
		id, _ := uuid.Parse(tenantID.String)
		user.TenantID = &id
	}
	if projectID.Valid {
		id, _ := uuid.Parse(projectID.String)
		user.ProjectID = &id
	}
	if clientID.Valid {
		id, _ := uuid.Parse(clientID.String)
		user.ClientID = &id
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}

	return &user, nil
}
