package database

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var ErrAdminUserNotFound = errors.New("admin user not found")

// AdminUserRepository handles admin user database operations on global DB
type AdminUserRepository struct {
	db *DBConnection
}

// NewAdminUserRepository creates a new admin user repository
func NewAdminUserRepository(db *DBConnection) *AdminUserRepository {
	return &AdminUserRepository{db: db}
}

// EnsureAdminRole returns the admin role id for the given tenant, creating it if needed.
// This now uses the full seeding function to ensure permissions are also created.
func (aur *AdminUserRepository) EnsureAdminRole(tenantID uuid.UUID) (uuid.UUID, error) {
	return NewAdminSeedRepository(aur.db).EnsureAdminRoleAndPermissions(tenantID)
}

// ListAdminUsersByTenant returns active admin users scoped to a tenant
// Uses role_bindings for role assignments (user_roles is deprecated)
func (aur *AdminUserRepository) ListAdminUsersByTenant(tenantID uuid.UUID) ([]models.AdminUser, error) {
	return aur.ListAdminUsersByTenantWithFilter(tenantID, "")
}

// ListAdminUsersByTenantWithFilter returns active admin users scoped to a tenant with optional provider filter
// Uses role_bindings for role assignments (user_roles is deprecated)
func (aur *AdminUserRepository) ListAdminUsersByTenantWithFilter(tenantID uuid.UUID, provider string) ([]models.AdminUser, error) {
	// Build query with optional provider filter
	queryBase := `
		SELECT DISTINCT u.id, u.email, u.username, u.password_hash, u.name,
		       u.client_id, u.tenant_id, u.project_id, u.tenant_domain, u.provider,
		       COALESCE(u.provider_id, '') AS provider_id,
		       COALESCE(u.provider_data, '{}'::jsonb) AS provider_data,
		       COALESCE(u.avatar_url, '') AS avatar_url, u.active, u.mfa_enabled,
		       u.mfa_method, COALESCE(u.mfa_default_method, '') AS mfa_default_method,
		       u.mfa_enrolled_at, u.mfa_verified,
		       COALESCE(u.external_id, '') AS external_id,
		       COALESCE(u.sync_source, '') AS sync_source,
		       u.last_sync_at, u.is_synced_user,
		       u.last_login, u.created_at, u.updated_at,
		       u.temporary_password, u.temporary_password_expires_at,
		       COALESCE(u.is_primary_admin, false) AS is_primary_admin
		FROM users u
		JOIN role_bindings rb ON u.id = rb.user_id AND rb.tenant_id = $1
		JOIN roles r ON rb.role_id = r.id AND r.tenant_id = $1
		WHERE u.active = true
		  AND u.tenant_id = $1
		  AND LOWER(r.name) IN ('admin', 'administrator', 'super_admin')`

	// Add provider filter if specified
	var rows *sql.Rows
	var err error

	if provider != "" {
		query := queryBase + ` AND u.provider = $2 ORDER BY u.created_at DESC`
		rows, err = aur.db.Query(query, tenantID, provider)
	} else {
		query := queryBase + ` ORDER BY u.created_at DESC`
		rows, err = aur.db.Query(query, tenantID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query admin users: %w", err)
	}
	defer rows.Close()

	var users []models.AdminUser
	for rows.Next() {
		var user models.AdminUser
		if err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.Name,
			&user.ClientID,
			&user.TenantID,
			&user.ProjectID,
			&user.TenantDomain,
			&user.Provider,
			&user.ProviderID,
			&user.ProviderData,
			&user.AvatarURL,
			&user.Active,
			&user.MFAEnabled,
			pq.Array(&user.MFAMethod),
			&user.MFADefaultMethod,
			&user.MFAEnrolledAt,
			&user.MFAVerified,
			&user.ExternalID,
			&user.SyncSource,
			&user.LastSyncAt,
			&user.IsSyncedUser,
			&user.LastLogin,
			&user.CreatedAt,
			&user.UpdatedAt,
			&user.TemporaryPassword,
			&user.TemporaryPasswordExpiresAt,
			&user.IsPrimaryAdmin,
		); err != nil {
			return nil, fmt.Errorf("failed to scan admin user: %w", err)
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("admin user query error: %w", err)
	}

	return users, nil
}

// UserRole represents a role assigned to a user
type UserRole struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// GetUserRoles returns all roles assigned to a user for a specific tenant
func (aur *AdminUserRepository) GetUserRoles(userID, tenantID uuid.UUID) ([]UserRole, error) {
	query := `
		SELECT DISTINCT r.id, r.name
		FROM roles r
		JOIN role_bindings rb ON r.id = rb.role_id
		WHERE rb.user_id = $1 AND rb.tenant_id = $2
		ORDER BY r.name
	`

	rows, err := aur.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	var roles []UserRole
	for rows.Next() {
		var role UserRole
		if err := rows.Scan(&role.ID, &role.Name); err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	if roles == nil {
		roles = []UserRole{}
	}

	return roles, rows.Err()
}

// HasPendingRegistration checks if a user has a pending registration entry
func (aur *AdminUserRepository) HasPendingRegistration(email string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pending_registrations 
			WHERE LOWER(email) = LOWER($1) AND expires_at > NOW()
		)
	`
	var exists bool
	err := aur.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check pending registration: %w", err)
	}
	return exists, nil
}

// UpdateAdminUserActive updates the active flag for a global admin user.
func (aur *AdminUserRepository) UpdateAdminUserActive(userID uuid.UUID, active bool) (bool, error) {
	query := `
		UPDATE users
		SET active = $1,
		    updated_at = $2
		WHERE id = $3
	`

	result, err := aur.db.Exec(query, active, time.Now(), userID)
	if err != nil {
		return false, fmt.Errorf("failed to update admin user active flag: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to determine rows affected: %w", err)
	}

	return rows > 0, nil
}

// EnsureTenantAdminRoleAssignment makes sure the tenant's primary admin user is mapped to the admin role.
func (aur *AdminUserRepository) EnsureTenantAdminRoleAssignment(tenantID uuid.UUID) error {
	// Seed admin role, scopes, and permissions per tenant
	adminRoleID, err := NewAdminSeedRepository(aur.db).EnsureAdminRoleAndPermissions(tenantID)
	if err != nil {
		log.Printf("Warning: Could not ensure admin role/permissions for tenant %s: %v", tenantID, err)
		return nil
	}

	// Try to find admin users for this tenant by:
	// 1. Matching tenant_id and email with tenants table
	// 2. Or just by tenant_id if the join fails (for OIDC users)
	var adminUserID uuid.UUID
	query := `
		SELECT u.id
		FROM users u
		JOIN tenants t ON t.tenant_id::text = u.tenant_id::text
		WHERE u.tenant_id::text = $1
		  AND LOWER(u.email) = LOWER(t.email)
		LIMIT 1
	`
	err = aur.db.QueryRow(query, tenantID.String()).Scan(&adminUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Fallback: try to find any user with this tenant_id (for OIDC registered users)
			fallbackQuery := `
				SELECT id
				FROM users
				WHERE tenant_id::text = $1
				  AND active = true
				ORDER BY created_at ASC
				LIMIT 1
			`
			if err := aur.db.QueryRow(fallbackQuery, tenantID.String()).Scan(&adminUserID); err != nil {
				if err == sql.ErrNoRows {
					// No users found for this tenant yet, that's okay
					// This is normal for new tenants
					return nil
				}
				// Log error but don't fail - role assignment might have been done during registration
				log.Printf("Warning: Could not locate tenant admin user for tenant %s: %v", tenantID, err)
				return nil
			}
		} else {
			// Log error but don't fail - role assignment might have been done during registration
			log.Printf("Warning: Error finding tenant admin user for tenant %s: %v", tenantID, err)
			return nil
		}
	}

	// Assign admin role via role_bindings (user_roles is deprecated)
	// This is now the primary mechanism for role assignment
	if err := aur.ensureAdminRoleBinding(adminUserID, tenantID, adminRoleID); err != nil {
		// Log warning but don't fail the reconciliation
		log.Printf("Warning: Could not ensure admin role binding for user %s tenant %s: %v", adminUserID, tenantID, err)
	}

	return nil
}

// ensureAdminRoleBinding creates a tenant-wide role binding for the admin user if missing.
func (aur *AdminUserRepository) ensureAdminRoleBinding(userID, tenantID, roleID uuid.UUID) error {
	if aur == nil || aur.db == nil {
		return fmt.Errorf("admin user repository not initialized")
	}

	insertQuery := `
		INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at)
		SELECT $1, $2, $3, $4, NULL, NULL, NOW()
		WHERE NOT EXISTS (
			SELECT 1 FROM role_bindings
			WHERE tenant_id = $2
			  AND user_id = $3
			  AND role_id = $4
			  AND scope_type IS NULL
			  AND scope_id IS NULL
		)
	`

	_, err := aur.db.Exec(insertQuery, uuid.New(), tenantID, userID, roleID)
	if err != nil {
		return fmt.Errorf("create admin role binding: %w", err)
	}

	return nil
}

func (aur *AdminUserRepository) GetAdminUserAccessContext(userID uuid.UUID) ([]string, []string, []string, []string, error) {
	// Uses role_bindings for role assignments (user_roles is deprecated)
	roleQuery := `
		SELECT r.name
		FROM role_bindings rb
		JOIN roles r ON rb.role_id = r.id
		WHERE rb.user_id = $1
	`

	resourceQuery := `
		SELECT DISTINCT p.resource
		FROM role_bindings rb
		JOIN role_permissions rp ON rb.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE rb.user_id = $1
	`

	permissionQuery := `
		SELECT DISTINCT p.resource || ':' || p.action
		FROM role_bindings rb
		JOIN role_permissions rp ON rb.role_id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE rb.user_id = $1
	`

	roles, err := aur.collectStringValues(roleQuery, userID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch admin roles: %w", err)
	}

	// Scopes are no longer used - return empty slice
	scopes := []string{}

	resources, err := aur.collectStringValues(resourceQuery, userID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch admin resources: %w", err)
	}

	permissions, err := aur.collectStringValues(permissionQuery, userID)
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to fetch admin permissions: %w", err)
	}

	return roles, scopes, resources, permissions, nil
}

func (aur *AdminUserRepository) collectStringValues(query string, args ...interface{}) ([]string, error) {
	rows, err := aur.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	seen := make(map[string]struct{})
	var values []string

	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, err
		}
		normalized := strings.TrimSpace(strings.ToLower(value))
		if normalized == "" {
			continue
		}
		if _, exists := seen[normalized]; !exists {
			seen[normalized] = struct{}{}
			values = append(values, normalized)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return values, nil
}

// CreateAdminUser creates a new admin user in global database
func (aur *AdminUserRepository) CreateAdminUser(user *models.AdminUser) error {
	// Validate that password hash is set for non-synced users
	// Synced users (from AD/Entra ID) authenticate via their provider, so password hash can be empty
	if user.PasswordHash == "" && !user.IsSyncedUser && user.Provider != "ad_sync" && user.Provider != "entra_id" {
		return fmt.Errorf("password hash must be set before creating admin user")
	}

	query := `
		INSERT INTO users (id, email, username, password_hash, name,
			provider, active, temporary_password, temporary_password_expires_at,
			created_at, updated_at, client_id, tenant_id, project_id, tenant_domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	now := time.Now()
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	if user.UpdatedAt.IsZero() {
		user.UpdatedAt = now
	}

	_, err := aur.db.Exec(query,
		user.ID,
		user.Email,
		user.Username,
		user.PasswordHash,
		user.Name,
		user.Provider,
		user.Active,
		user.TemporaryPassword,
		user.TemporaryPasswordExpiresAt,
		user.CreatedAt,
		user.UpdatedAt,
		user.ClientID,
		user.TenantID,
		user.ProjectID,
		user.TenantDomain,
	)

	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	tenantID := uuid.Nil
	if user.TenantID != nil {
		tenantID = *user.TenantID
	}

	roleID, err := aur.EnsureAdminRole(tenantID)
	if err != nil {
		fmt.Printf("WARNING: Failed to ensure admin role for tenant %s: %v\n", tenantID, err)
		fmt.Printf("WARNING: User created but without admin role - they will not be able to login via /admin/login\n")
		return nil
	}

	// Use role_bindings (user_roles is deprecated)
	bindingID := uuid.New()
	result, err := aur.db.Exec(`
		INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
		SELECT $1, $2, $3, $4, NULL, NULL, NOW(), NOW()
		WHERE NOT EXISTS (
			SELECT 1 FROM role_bindings WHERE tenant_id = $2 AND user_id = $3 AND role_id = $4 AND scope_type IS NULL
		)
	`, bindingID, tenantID, user.ID, roleID)
	if err != nil {
		fmt.Printf("WARNING: Failed to assign admin role to user %s: %v\n", user.ID, err)
		fmt.Printf("WARNING: User created but without admin role - they will not be able to login via /admin/login\n")
	} else {
		rowsAffected, _ := result.RowsAffected()
		if rowsAffected == 0 {
			fmt.Printf("WARNING: Admin role not assigned to user %s - admin role may already exist\n", user.ID)
		} else {
			fmt.Printf("INFO: Admin role successfully assigned to user %s\n", user.ID)
		}
	}

	return nil
}

// GetAdminUserByEmail retrieves an admin user by email (case-insensitive)
// Uses role_bindings for role assignments (user_roles is deprecated)
func (aur *AdminUserRepository) GetAdminUserByEmail(email string) (*models.AdminUser, error) {
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, COALESCE(u.name, '') AS name,
			u.client_id, u.tenant_id, u.project_id, COALESCE(u.tenant_domain, '') AS tenant_domain, COALESCE(u.provider, '') AS provider,
			u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
			u.avatar_url, u.active, u.mfa_enabled,
			COALESCE(u.mfa_method, ARRAY[]::text[]) AS mfa_method, u.mfa_default_method,
			u.mfa_enrolled_at, u.mfa_verified,
			u.external_id, u.sync_source,
			u.last_sync_at, u.is_synced_user,
			u.last_login, u.temporary_password, u.temporary_password_expires_at,
			u.created_at, u.updated_at
		FROM users u
		JOIN role_bindings rb ON u.id = rb.user_id
		JOIN roles r ON rb.role_id = r.id
		WHERE LOWER(u.email) = LOWER($1) AND u.active = true AND LOWER(r.name) = 'admin'
	`

	fmt.Printf("UserFlow:Debug:: Query to get user for email %s: %s\n", email, strings.ReplaceAll(query, "\n", " "))
	var (
		username              sql.NullString
		name                  sql.NullString
		clientIDStr           sql.NullString
		tenantIDStr           sql.NullString
		projectIDStr          sql.NullString
		tenantDomain          sql.NullString
		provider              sql.NullString
		providerID            sql.NullString
		providerData          sql.NullString
		avatarURL             sql.NullString
		mfaDefaultMethod      sql.NullString
		mfaEnrolledAt         sql.NullTime
		externalID            sql.NullString
		syncSource            sql.NullString
		lastSyncAt            sql.NullTime
		lastLogin             sql.NullTime
		tempPasswordExpiresAt sql.NullTime
		mfaMethodRaw          interface{} // Scan as interface{} to handle NULL and array
	)
	var (
		clientID  *uuid.UUID
		tenantID  *uuid.UUID
		projectID *uuid.UUID
	)
	var user models.AdminUser

	err := aur.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&username,
		&user.PasswordHash,
		&name,
		&clientIDStr,
		&tenantIDStr,
		&projectIDStr,
		&tenantDomain,
		&provider,
		&providerID,
		&providerData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethodRaw,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&externalID,
		&syncSource,
		&lastSyncAt,
		&user.IsSyncedUser,
		&lastLogin,
		&user.TemporaryPassword,
		&tempPasswordExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get admin user by email: %w", err)
	}

	// Parse mfa_method array
	user.MFAMethod = []string{}
	if mfaMethodRaw != nil {
		// Try to use pq.Array to scan the value
		var mfaArray pq.StringArray
		if err := mfaArray.Scan(mfaMethodRaw); err == nil {
			user.MFAMethod = []string(mfaArray)
		} else {
			// Fallback: if it's a string representation, parse it manually
			if strVal, ok := mfaMethodRaw.(string); ok && strVal != "" && strVal != "{}" {
				// Remove braces and split by comma
				strVal = strings.Trim(strVal, "{}")
				if strVal != "" {
					user.MFAMethod = strings.Split(strVal, ",")
				}
			}
		}
	}

	// Assign nullable fields
	if username.Valid {
		user.Username = username.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if tenantDomain.Valid {
		user.TenantDomain = tenantDomain.String
	}
	if provider.Valid {
		user.Provider = provider.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	// ProviderData is stored as JSONB; use raw bytes when available
	if providerData.Valid {
		user.ProviderData = []byte(providerData.String)
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = mfaDefaultMethod.String
	}
	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if syncSource.Valid {
		user.SyncSource = syncSource.String
	}
	if mfaEnrolledAt.Valid {
		ts := mfaEnrolledAt.Time
		user.MFAEnrolledAt = &ts
	}
	if lastSyncAt.Valid {
		ts := lastSyncAt.Time
		user.LastSyncAt = &ts
	}
	if lastLogin.Valid {
		ts := lastLogin.Time
		user.LastLogin = &ts
	}
	if tempPasswordExpiresAt.Valid {
		ts := tempPasswordExpiresAt.Time
		user.TemporaryPasswordExpiresAt = &ts
	}

	if clientIDStr.Valid && strings.TrimSpace(clientIDStr.String) != "" {
		if parsed, err := uuid.Parse(clientIDStr.String); err == nil {
			clientID = &parsed
		}
	}
	if tenantIDStr.Valid && strings.TrimSpace(tenantIDStr.String) != "" {
		if parsed, err := uuid.Parse(tenantIDStr.String); err == nil {
			tenantID = &parsed
		}
	}
	if projectIDStr.Valid && strings.TrimSpace(projectIDStr.String) != "" {
		if parsed, err := uuid.Parse(projectIDStr.String); err == nil {
			projectID = &parsed
		}
	}

	user.ClientID = clientID
	user.TenantID = tenantID
	user.ProjectID = projectID

	return &user, nil
}

// GetAdminUserByEmailAndTenantDomain retrieves an admin user by email and tenant_domain (case-insensitive)
// This method enforces tenant isolation by requiring the user's tenant_domain to match
// Uses role_bindings for role assignments (user_roles is deprecated)
// NOTE: This is a relaxed version that finds any active user with admin role AND matching tenant_domain.
// If no role binding exists yet (e.g., newly invited user), it falls back to just email+domain match.
func (aur *AdminUserRepository) GetAdminUserByEmailAndTenantDomain(email, tenantDomain string) (*models.AdminUser, error) {
	// First try with role_bindings (user has admin role)
	queryWithRole := `
		SELECT u.id, u.email, u.username, u.password_hash, COALESCE(u.name, '') AS name,
			u.client_id, u.tenant_id, u.project_id, COALESCE(u.tenant_domain, '') AS tenant_domain, COALESCE(u.provider, '') AS provider,
			u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
			u.avatar_url, u.active, u.mfa_enabled,
			COALESCE(u.mfa_method, ARRAY[]::text[]) AS mfa_method, u.mfa_default_method,
			u.mfa_enrolled_at, u.mfa_verified,
			u.external_id, u.sync_source,
			u.last_sync_at, u.is_synced_user,
			u.last_login, u.temporary_password, u.temporary_password_expires_at,
			u.created_at, u.updated_at
		FROM users u
		JOIN role_bindings rb ON u.id = rb.user_id
		JOIN roles r ON rb.role_id = r.id
		WHERE LOWER(u.email) = LOWER($1) AND LOWER(u.tenant_domain) = LOWER($2) AND u.active = true AND LOWER(r.name) = 'admin'
	`

	fmt.Printf("UserFlow:Debug:: Query to get user for email %s, tenant_domain %s (with role check)\n", email, tenantDomain)

	user, err := aur.scanAdminUserFromQuery(queryWithRole, email, tenantDomain)
	if err == nil {
		return user, nil
	}

	// If no role binding exists (e.g., newly invited user before role binding is complete),
	// try without the role check - but ONLY if tenant_domain matches exactly
	if err == sql.ErrNoRows {
		queryWithoutRole := `
			SELECT u.id, u.email, u.username, u.password_hash, COALESCE(u.name, '') AS name,
				u.client_id, u.tenant_id, u.project_id, COALESCE(u.tenant_domain, '') AS tenant_domain, COALESCE(u.provider, '') AS provider,
				u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
				u.avatar_url, u.active, u.mfa_enabled,
				COALESCE(u.mfa_method, ARRAY[]::text[]) AS mfa_method, u.mfa_default_method,
				u.mfa_enrolled_at, u.mfa_verified,
				u.external_id, u.sync_source,
				u.last_sync_at, u.is_synced_user,
				u.last_login, u.temporary_password, u.temporary_password_expires_at,
				u.created_at, u.updated_at
			FROM users u
			WHERE LOWER(u.email) = LOWER($1) AND LOWER(u.tenant_domain) = LOWER($2) AND u.active = true
		`
		fmt.Printf("UserFlow:Debug:: Fallback query without role check for email %s, tenant_domain %s\n", email, tenantDomain)
		user, err = aur.scanAdminUserFromQuery(queryWithoutRole, email, tenantDomain)
		if err == nil {
			return user, nil
		}
	}

	return nil, err
}

// scanAdminUserFromQuery is a helper to scan admin user from a query
func (aur *AdminUserRepository) scanAdminUserFromQuery(query string, args ...interface{}) (*models.AdminUser, error) {
	var (
		username              sql.NullString
		name                  sql.NullString
		clientIDStr           sql.NullString
		tenantIDStr           sql.NullString
		projectIDStr          sql.NullString
		tenantDomain          sql.NullString
		provider              sql.NullString
		providerID            sql.NullString
		providerData          sql.NullString
		avatarURL             sql.NullString
		mfaDefaultMethod      sql.NullString
		mfaEnrolledAt         sql.NullTime
		externalID            sql.NullString
		syncSource            sql.NullString
		lastSyncAt            sql.NullTime
		lastLogin             sql.NullTime
		tempPasswordExpiresAt sql.NullTime
		mfaMethodRaw          interface{}
	)
	var (
		clientID  *uuid.UUID
		tenantID  *uuid.UUID
		projectID *uuid.UUID
	)
	var user models.AdminUser

	err := aur.db.QueryRow(query, args...).Scan(
		&user.ID,
		&user.Email,
		&username,
		&user.PasswordHash,
		&name,
		&clientIDStr,
		&tenantIDStr,
		&projectIDStr,
		&tenantDomain,
		&provider,
		&providerID,
		&providerData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethodRaw,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&externalID,
		&syncSource,
		&lastSyncAt,
		&user.IsSyncedUser,
		&lastLogin,
		&user.TemporaryPassword,
		&tempPasswordExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get admin user: %w", err)
	}

	// Parse mfa_method array
	user.MFAMethod = []string{}
	if mfaMethodRaw != nil {
		var mfaArray pq.StringArray
		if err := mfaArray.Scan(mfaMethodRaw); err == nil {
			user.MFAMethod = []string(mfaArray)
		} else if strVal, ok := mfaMethodRaw.(string); ok && strVal != "" && strVal != "{}" {
			strVal = strings.Trim(strVal, "{}")
			if strVal != "" {
				user.MFAMethod = strings.Split(strVal, ",")
			}
		}
	}

	// Assign nullable fields
	if username.Valid {
		user.Username = username.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if tenantDomain.Valid {
		user.TenantDomain = tenantDomain.String
	}
	if provider.Valid {
		user.Provider = provider.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if providerData.Valid {
		user.ProviderData = []byte(providerData.String)
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = mfaDefaultMethod.String
	}
	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if syncSource.Valid {
		user.SyncSource = syncSource.String
	}
	if mfaEnrolledAt.Valid {
		ts := mfaEnrolledAt.Time
		user.MFAEnrolledAt = &ts
	}
	if lastSyncAt.Valid {
		ts := lastSyncAt.Time
		user.LastSyncAt = &ts
	}
	if lastLogin.Valid {
		ts := lastLogin.Time
		user.LastLogin = &ts
	}
	if tempPasswordExpiresAt.Valid {
		ts := tempPasswordExpiresAt.Time
		user.TemporaryPasswordExpiresAt = &ts
	}

	if clientIDStr.Valid && strings.TrimSpace(clientIDStr.String) != "" {
		if parsed, err := uuid.Parse(clientIDStr.String); err == nil {
			clientID = &parsed
		}
	}
	if tenantIDStr.Valid && strings.TrimSpace(tenantIDStr.String) != "" {
		if parsed, err := uuid.Parse(tenantIDStr.String); err == nil {
			tenantID = &parsed
		}
	}
	if projectIDStr.Valid && strings.TrimSpace(projectIDStr.String) != "" {
		if parsed, err := uuid.Parse(projectIDStr.String); err == nil {
			projectID = &parsed
		}
	}

	user.ClientID = clientID
	user.TenantID = tenantID
	user.ProjectID = projectID

	return &user, nil
}

// GetAdminUserByID retrieves an admin user by ID
// Uses role_bindings for role assignments (user_roles is deprecated)
func (aur *AdminUserRepository) GetAdminUserByID(id uuid.UUID) (*models.AdminUser, error) {
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, u.name,
			u.client_id, u.tenant_id, u.project_id, u.tenant_domain, u.provider,
			u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
			COALESCE(u.avatar_url, '') AS avatar_url, u.active, u.mfa_enabled,
			u.mfa_method, COALESCE(u.mfa_default_method, '') AS mfa_default_method,
			u.mfa_enrolled_at, u.mfa_verified,
			COALESCE(u.external_id, '') AS external_id,
			COALESCE(u.sync_source, '') AS sync_source,
			u.last_sync_at, u.is_synced_user,
			u.last_login, u.created_at, u.updated_at
		FROM users u
		JOIN role_bindings rb ON u.id = rb.user_id
		JOIN roles r ON rb.role_id = r.id
		WHERE u.id = $1 AND u.active = true AND r.name = 'admin'
	`

	var user models.AdminUser
	err := aur.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Username,
		&user.PasswordHash,
		&user.Name,
		&user.ClientID,
		&user.TenantID,
		&user.ProjectID,
		&user.TenantDomain,
		&user.Provider,
		&user.ProviderID,
		&user.ProviderData,
		&user.AvatarURL,
		&user.Active,
		&user.MFAEnabled,
		pq.Array(&user.MFAMethod),
		&user.MFADefaultMethod,
		&user.MFAEnrolledAt,
		&user.MFAVerified,
		&user.ExternalID,
		&user.SyncSource,
		&user.LastSyncAt,
		&user.IsSyncedUser,
		&user.LastLogin,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get admin user: %w", err)
	}

	return &user, nil
}

// UpdateAdminUser updates an admin user
func (aur *AdminUserRepository) UpdateAdminUser(id uuid.UUID, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return fmt.Errorf("no updates provided")
	}

	query := "UPDATE users SET "
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

	query += " WHERE id = $" + fmt.Sprintf("%d", argCount)
	args = append(args, id)

	_, err := aur.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update admin user: %w", err)
	}

	return nil
}

// DeleteAdminUser soft deletes an admin user
func (aur *AdminUserRepository) DeleteAdminUser(id uuid.UUID) error {
	query := "UPDATE users SET active = false, updated_at = $1 WHERE id = $2"

	res, err := aur.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete admin user: %w", err)
	}

	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return ErrAdminUserNotFound
	}

	return nil
}

// UpdateAdminUserActiveStatus toggles the active flag for an admin user.
func (aur *AdminUserRepository) UpdateAdminUserActiveStatus(id uuid.UUID, active bool) error {
	query := "UPDATE users SET active = $1, updated_at = $2 WHERE id = $3"

	res, err := aur.db.Exec(query, active, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update admin user status: %w", err)
	}

	if rows, err := res.RowsAffected(); err == nil && rows == 0 {
		return ErrAdminUserNotFound
	}

	return nil
}

// GetAllAdminUsers retrieves all admin users
// Uses role_bindings for role assignments (user_roles is deprecated)
func (aur *AdminUserRepository) GetAllAdminUsers() ([]models.AdminUser, error) {
	query := `
		SELECT DISTINCT u.id, u.email, u.username, u.password_hash, u.name,
			u.client_id, u.tenant_id, u.project_id, u.tenant_domain, u.provider,
			u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
			COALESCE(u.avatar_url, '') AS avatar_url, u.active, u.mfa_enabled,
			u.mfa_method, COALESCE(u.mfa_default_method, '') AS mfa_default_method,
			u.mfa_enrolled_at, u.mfa_verified,
			COALESCE(u.external_id, '') AS external_id,
			COALESCE(u.sync_source, '') AS sync_source,
			u.last_sync_at, u.is_synced_user,
			u.last_login, u.created_at, u.updated_at
		FROM users u
		JOIN role_bindings rb ON u.id = rb.user_id
		JOIN roles r ON rb.role_id = r.id
		WHERE u.active = true AND r.name = 'admin'
		ORDER BY u.created_at DESC
	`

	rows, err := aur.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query admin users: %w", err)
	}
	defer rows.Close()

	var users []models.AdminUser
	for rows.Next() {
		var user models.AdminUser
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Username,
			&user.PasswordHash,
			&user.Name,
			&user.ClientID,
			&user.TenantID,
			&user.ProjectID,
			&user.TenantDomain,
			&user.Provider,
			&user.ProviderID,
			&user.ProviderData,
			&user.AvatarURL,
			&user.Active,
			&user.MFAEnabled,
			pq.Array(&user.MFAMethod),
			&user.MFADefaultMethod,
			&user.MFAEnrolledAt,
			&user.MFAVerified,
			&user.ExternalID,
			&user.SyncSource,
			&user.LastSyncAt,
			&user.IsSyncedUser,
			&user.LastLogin,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan admin user: %w", err)
		}
		users = append(users, user)
	}

	return users, nil
}

// UpdateLastLogin updates the last login time for an admin user
func (aur *AdminUserRepository) UpdateLastLogin(id uuid.UUID) error {
	query := "UPDATE users SET last_login = $1, updated_at = $1 WHERE id = $2"

	_, err := aur.db.Exec(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// VerifyPassword verifies an admin user's password
func (aur *AdminUserRepository) VerifyPassword(email, password string) (*models.AdminUser, error) {
	user, err := aur.GetAdminUserByEmail(email)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid password")
	}

	return user, nil
}

// GetAdminUserByEmailAndTenant retrieves an admin user by email and tenant ID (case-insensitive)
// This method respects the new composite unique constraint (email, tenant_id)
// Uses role_bindings for role assignments (user_roles is deprecated)
func (aur *AdminUserRepository) GetAdminUserByEmailAndTenant(email string, tenantID uuid.UUID) (*models.AdminUser, error) {
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, COALESCE(u.name, '') AS name,
			u.client_id, u.tenant_id, u.project_id, COALESCE(u.tenant_domain, '') AS tenant_domain, COALESCE(u.provider, '') AS provider,
			u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
			u.avatar_url, u.active, u.mfa_enabled,
			COALESCE(u.mfa_method, ARRAY[]::text[]) AS mfa_method, u.mfa_default_method,
			u.mfa_enrolled_at, u.mfa_verified,
			u.external_id, u.sync_source,
			u.last_sync_at, u.is_synced_user,
			u.last_login, u.temporary_password, u.temporary_password_expires_at,
			u.created_at, u.updated_at
		FROM users u
		JOIN role_bindings rb ON u.id = rb.user_id
		JOIN roles r ON rb.role_id = r.id
		WHERE LOWER(u.email) = LOWER($1) AND u.tenant_id = $2 AND u.active = true AND LOWER(r.name) = 'admin'
	`

	var (
		username              sql.NullString
		name                  sql.NullString
		clientIDStr           sql.NullString
		tenantIDStr           sql.NullString
		projectIDStr          sql.NullString
		tenantDomain          sql.NullString
		provider              sql.NullString
		providerID            sql.NullString
		providerData          sql.NullString
		avatarURL             sql.NullString
		mfaDefaultMethod      sql.NullString
		mfaEnrolledAt         sql.NullTime
		externalID            sql.NullString
		syncSource            sql.NullString
		lastSyncAt            sql.NullTime
		lastLogin             sql.NullTime
		tempPasswordExpiresAt sql.NullTime
		mfaMethodRaw          interface{}
	)
	var (
		clientID       *uuid.UUID
		tenantIDParsed *uuid.UUID
		projectID      *uuid.UUID
	)
	var user models.AdminUser

	err := aur.db.QueryRow(query, email, tenantID).Scan(
		&user.ID,
		&user.Email,
		&username,
		&user.PasswordHash,
		&name,
		&clientIDStr,
		&tenantIDStr,
		&projectIDStr,
		&tenantDomain,
		&provider,
		&providerID,
		&providerData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethodRaw,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&externalID,
		&syncSource,
		&lastSyncAt,
		&user.IsSyncedUser,
		&lastLogin,
		&user.TemporaryPassword,
		&tempPasswordExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, sql.ErrNoRows
		}
		return nil, fmt.Errorf("failed to get admin user by email and tenant: %w", err)
	}

	// Parse mfa_method array
	user.MFAMethod = []string{}
	if mfaMethodRaw != nil {
		var mfaArray pq.StringArray
		if err := mfaArray.Scan(mfaMethodRaw); err == nil {
			user.MFAMethod = []string(mfaArray)
		} else {
			if strVal, ok := mfaMethodRaw.(string); ok && strVal != "" && strVal != "{}" {
				strVal = strings.Trim(strVal, "{}")
				if strVal != "" {
					user.MFAMethod = strings.Split(strVal, ",")
				}
			}
		}
	}

	// Assign nullable fields
	if username.Valid {
		user.Username = username.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if tenantDomain.Valid {
		user.TenantDomain = tenantDomain.String
	}
	if provider.Valid {
		user.Provider = provider.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if providerData.Valid {
		user.ProviderData = []byte(providerData.String)
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = mfaDefaultMethod.String
	}
	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if syncSource.Valid {
		user.SyncSource = syncSource.String
	}
	if mfaEnrolledAt.Valid {
		ts := mfaEnrolledAt.Time
		user.MFAEnrolledAt = &ts
	}
	if lastSyncAt.Valid {
		ts := lastSyncAt.Time
		user.LastSyncAt = &ts
	}
	if lastLogin.Valid {
		ts := lastLogin.Time
		user.LastLogin = &ts
	}
	if tempPasswordExpiresAt.Valid {
		ts := tempPasswordExpiresAt.Time
		user.TemporaryPasswordExpiresAt = &ts
	}

	if clientIDStr.Valid && strings.TrimSpace(clientIDStr.String) != "" {
		if parsed, err := uuid.Parse(clientIDStr.String); err == nil {
			clientID = &parsed
		}
	}
	if tenantIDStr.Valid && strings.TrimSpace(tenantIDStr.String) != "" {
		if parsed, err := uuid.Parse(tenantIDStr.String); err == nil {
			tenantIDParsed = &parsed
		}
	}
	if projectIDStr.Valid && strings.TrimSpace(projectIDStr.String) != "" {
		if parsed, err := uuid.Parse(projectIDStr.String); err == nil {
			projectID = &parsed
		}
	}

	user.ClientID = clientID
	user.TenantID = tenantIDParsed
	user.ProjectID = projectID

	return &user, nil
}

// GetAdminUserWithProviders retrieves an admin user by email with available auth providers
// Returns user info and list of configured providers (email, google, etc.)
// Note: This method does NOT require the admin role to be assigned, as it's used for precheck
func (aur *AdminUserRepository) GetAdminUserWithProviders(email string) (*models.AdminUser, []string, error) {
	// Query user without requiring admin role (for precheck purposes)
	query := `
		SELECT u.id, u.email, u.username, u.password_hash, COALESCE(u.name, '') AS name,
			u.client_id, u.tenant_id, u.project_id, COALESCE(u.tenant_domain, '') AS tenant_domain, COALESCE(u.provider, '') AS provider,
			u.provider_id, COALESCE(u.provider_data::text, '{}') AS provider_data,
			u.avatar_url, u.active, u.mfa_enabled,
			COALESCE(u.mfa_method, ARRAY[]::text[]) AS mfa_method, u.mfa_default_method,
			u.mfa_enrolled_at, u.mfa_verified,
			u.external_id, u.sync_source,
			u.last_sync_at, u.is_synced_user,
			u.last_login, u.temporary_password, u.temporary_password_expires_at,
			u.created_at, u.updated_at
		FROM users u
		WHERE LOWER(u.email) = LOWER($1) AND u.active = true
	`

	var (
		username              sql.NullString
		name                  sql.NullString
		clientIDStr           sql.NullString
		tenantIDStr           sql.NullString
		projectIDStr          sql.NullString
		tenantDomain          sql.NullString
		provider              sql.NullString
		providerID            sql.NullString
		providerData          sql.NullString
		avatarURL             sql.NullString
		mfaDefaultMethod      sql.NullString
		mfaEnrolledAt         sql.NullTime
		externalID            sql.NullString
		syncSource            sql.NullString
		lastSyncAt            sql.NullTime
		lastLogin             sql.NullTime
		tempPasswordExpiresAt sql.NullTime
		mfaMethodRaw          interface{}
	)
	var (
		clientID  *uuid.UUID
		tenantID  *uuid.UUID
		projectID *uuid.UUID
	)
	var user models.AdminUser

	err := aur.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&username,
		&user.PasswordHash,
		&name,
		&clientIDStr,
		&tenantIDStr,
		&projectIDStr,
		&tenantDomain,
		&provider,
		&providerID,
		&providerData,
		&avatarURL,
		&user.Active,
		&user.MFAEnabled,
		&mfaMethodRaw,
		&mfaDefaultMethod,
		&mfaEnrolledAt,
		&user.MFAVerified,
		&externalID,
		&syncSource,
		&lastSyncAt,
		&user.IsSyncedUser,
		&lastLogin,
		&user.TemporaryPassword,
		&tempPasswordExpiresAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			// User doesn't exist, return empty
			return nil, []string{"email"}, nil
		}
		return nil, nil, fmt.Errorf("failed to get admin user: %w", err)
	}

	// Parse mfa_method array
	user.MFAMethod = []string{}
	if mfaMethodRaw != nil {
		var mfaArray pq.StringArray
		if err := mfaArray.Scan(mfaMethodRaw); err == nil {
			user.MFAMethod = []string(mfaArray)
		} else {
			if strVal, ok := mfaMethodRaw.(string); ok && strVal != "" && strVal != "{}" {
				strVal = strings.Trim(strVal, "{}")
				if strVal != "" {
					user.MFAMethod = strings.Split(strVal, ",")
				}
			}
		}
	}

	// Assign nullable fields
	if username.Valid {
		user.Username = username.String
	}
	if name.Valid {
		user.Name = name.String
	}
	if tenantDomain.Valid {
		user.TenantDomain = tenantDomain.String
	}
	if provider.Valid {
		user.Provider = provider.String
	}
	if providerID.Valid {
		user.ProviderID = providerID.String
	}
	if providerData.Valid {
		user.ProviderData = []byte(providerData.String)
	}
	if avatarURL.Valid {
		user.AvatarURL = avatarURL.String
	}
	if mfaDefaultMethod.Valid {
		user.MFADefaultMethod = mfaDefaultMethod.String
	}
	if externalID.Valid {
		user.ExternalID = externalID.String
	}
	if syncSource.Valid {
		user.SyncSource = syncSource.String
	}
	if mfaEnrolledAt.Valid {
		ts := mfaEnrolledAt.Time
		user.MFAEnrolledAt = &ts
	}
	if lastSyncAt.Valid {
		ts := lastSyncAt.Time
		user.LastSyncAt = &ts
	}
	if lastLogin.Valid {
		ts := lastLogin.Time
		user.LastLogin = &ts
	}
	if tempPasswordExpiresAt.Valid {
		ts := tempPasswordExpiresAt.Time
		user.TemporaryPasswordExpiresAt = &ts
	}

	if clientIDStr.Valid && strings.TrimSpace(clientIDStr.String) != "" {
		if parsed, err := uuid.Parse(clientIDStr.String); err == nil {
			clientID = &parsed
		}
	}
	if tenantIDStr.Valid && strings.TrimSpace(tenantIDStr.String) != "" {
		if parsed, err := uuid.Parse(tenantIDStr.String); err == nil {
			tenantID = &parsed
		}
	}
	if projectIDStr.Valid && strings.TrimSpace(projectIDStr.String) != "" {
		if parsed, err := uuid.Parse(projectIDStr.String); err == nil {
			projectID = &parsed
		}
	}

	user.ClientID = clientID
	user.TenantID = tenantID
	user.ProjectID = projectID

	// Get configured providers for this tenant/client
	providers := []string{"email"} // Email is always available

	// Check if user has OAuth providers configured
	if user.Provider != "" && user.Provider != "email" {
		providers = append(providers, user.Provider)
	}

	// Query tenant configuration for available providers
	if user.TenantID != nil {
		providerQuery := `
			SELECT DISTINCT provider_name
			FROM oauth_configs
			WHERE tenant_id = $1 AND enabled = true
		`
		rows, err := aur.db.Query(providerQuery, user.TenantID)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var providerName string
				if err := rows.Scan(&providerName); err == nil {
					// Add provider if not already in list
					found := false
					for _, p := range providers {
						if p == providerName {
							found = true
							break
						}
					}
					if !found {
						providers = append(providers, providerName)
					}
				}
			}
		}
	}

	return &user, providers, nil
}

// GetAdminRoles fetches the role names for an admin user in a specific tenant
// Queries role_bindings -> roles to get role names
func (aur *AdminUserRepository) GetAdminRoles(userID uuid.UUID, tenantID uuid.UUID) ([]string, error) {
	query := `
		SELECT DISTINCT r.name
		FROM role_bindings rb
		JOIN roles r ON rb.role_id = r.id
		WHERE rb.user_id = $1 
		  AND rb.tenant_id = $2
		ORDER BY r.name
	`

	rows, err := aur.db.Query(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query admin roles: %w", err)
	}
	defer rows.Close()

	var roles []string
	for rows.Next() {
		var roleName string
		if err := rows.Scan(&roleName); err != nil {
			return nil, fmt.Errorf("failed to scan role name: %w", err)
		}
		roles = append(roles, roleName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating roles: %w", err)
	}

	return roles, nil
}
