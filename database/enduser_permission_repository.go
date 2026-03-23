package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
) // EndUserPermissionRepository handles end-user permission operations on tenant DB
type EndUserPermissionRepository struct {
	db interface{} // Can be *DBConnection or *sql.DB depending on tenant connection
}

// NewEndUserPermissionRepository creates a new end-user permission repository
func NewEndUserPermissionRepository(db interface{}) *EndUserPermissionRepository {
	return &EndUserPermissionRepository{db: db}
}

// setDB sets the database connection (for dynamic tenant connections)
func (eupr *EndUserPermissionRepository) setDB(db interface{}) {
	eupr.db = db
}

// executeQuery executes a query on the current database connection
func (eupr *EndUserPermissionRepository) executeQuery(query string, args ...interface{}) (*sql.Rows, error) {
	switch db := eupr.db.(type) {
	case *DBConnection:
		return db.Query(query, args...)
	case *sql.DB:
		return db.Query(query, args...)
	default:
		return nil, fmt.Errorf("unsupported database connection type")
	}
}

// executeQueryRow executes a query that returns a single row
func (eupr *EndUserPermissionRepository) executeQueryRow(query string, args ...interface{}) *sql.Row {
	switch db := eupr.db.(type) {
	case *DBConnection:
		return db.QueryRow(query, args...)
	case *sql.DB:
		return db.QueryRow(query, args...)
	default:
		// Return a row that will error when scanned
		return nil
	}
}

// executeExec executes a command that doesn't return rows
func (eupr *EndUserPermissionRepository) executeExec(query string, args ...interface{}) (sql.Result, error) {
	switch db := eupr.db.(type) {
	case *DBConnection:
		return db.Exec(query, args...)
	case *sql.DB:
		return db.Exec(query, args...)
	default:
		return nil, fmt.Errorf("unsupported database connection type")
	}
}

// GetUserPermissions retrieves all permissions for a specific user in tenant DB
func (eupr *EndUserPermissionRepository) GetUserPermissions(userID uuid.UUID, tenantID string) ([]models.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.tenant_id, p.resource, p.action, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		INNER JOIN user_roles ur ON rp.role_id = ur.role_id
		WHERE ur.user_id = $1 AND ur.tenant_id = $2
	`

	rows, err := eupr.executeQuery(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID,
			&permission.TenantID,
			&permission.Resource,
			&permission.Action,
			&permission.Description,
			&permission.CreatedAt,
			&permission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
}

// GetUserEffectivePermissions retrieves effective permissions for a user (including group permissions)
func (eupr *EndUserPermissionRepository) GetUserEffectivePermissions(userID uuid.UUID, tenantID string) ([]models.Permission, error) {
	query := `
		SELECT DISTINCT p.id, p.tenant_id, p.resource, p.action, p.description, p.created_at, p.updated_at
		FROM permissions p
		INNER JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id IN (
			-- Direct user roles
			SELECT ur.role_id FROM user_roles ur WHERE ur.user_id = $1 AND ur.tenant_id = $2
			UNION
			-- Group roles
			SELECT gr.role_id FROM group_roles gr
			INNER JOIN user_groups ug ON gr.group_id = ug.group_id
			WHERE ug.user_id = $1 AND ug.tenant_id = $2
		)
	`

	rows, err := eupr.executeQuery(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query effective permissions: %w", err)
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		var permission models.Permission
		err := rows.Scan(
			&permission.ID,
			&permission.TenantID,
			&permission.Resource,
			&permission.Action,
			&permission.Description,
			&permission.CreatedAt,
			&permission.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan permission: %w", err)
		}
		permissions = append(permissions, permission)
	}

	return permissions, nil
} // CheckUserPermission checks if a user has a specific permission
func (eupr *EndUserPermissionRepository) CheckUserPermission(userID uuid.UUID, tenantID string, resource string, method string, scope string) (bool, error) {
	query := `
SELECT COUNT(*) > 0
FROM permissions p
INNER JOIN resources r ON p.resource_id = r.id
INNER JOIN resource_methods rm ON p.resource_method_id = rm.id
INNER JOIN scopes s ON p.scope_id = s.id
WHERE p.role_id IN (
-- Direct user roles
SELECT ur.role_id FROM user_roles ur WHERE ur.user_id = $1 AND ur.tenant_id = $2
UNION
-- Group roles
SELECT gr.role_id FROM group_roles gr
INNER JOIN user_groups ug ON gr.group_id = ug.group_id
WHERE ug.user_id = $1 AND ug.tenant_id = $2
)
AND LOWER(r.name) = LOWER($3)
AND LOWER(rm.method) = LOWER($4)
AND LOWER(s.name) = LOWER($5)
`

	var hasPermission bool
	row := eupr.executeQueryRow(query, userID, tenantID, resource, method, scope)
	if row == nil {
		return false, fmt.Errorf("database connection error")
	}

	err := row.Scan(&hasPermission)
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return hasPermission, nil
}

// GetUserRoles retrieves all roles for a specific user in tenant DB
func (eupr *EndUserPermissionRepository) GetUserRoles(userID uuid.UUID, tenantID string) ([]models.Role, error) {
	query := `
SELECT DISTINCT r.id, r.name, r.description, r.tenant_id, r.created_at, r.updated_at
FROM roles r
WHERE r.id IN (
-- Direct user roles
SELECT ur.role_id FROM user_roles ur WHERE ur.user_id = $1 AND ur.tenant_id = $2
UNION
-- Group roles
SELECT gr.role_id FROM group_roles gr
INNER JOIN user_groups ug ON gr.group_id = ug.group_id
WHERE ug.user_id = $1 AND ug.tenant_id = $2
)
ORDER BY r.created_at DESC
`

	rows, err := eupr.executeQuery(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.TenantID,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// GetAvailableRoles retrieves all available roles for a tenant
func (eupr *EndUserPermissionRepository) GetAvailableRoles(tenantID string) ([]models.Role, error) {
	query := `
SELECT id, name, description, tenant_id, created_at, updated_at
FROM roles
WHERE tenant_id = $1
ORDER BY name ASC
`

	rows, err := eupr.executeQuery(query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query available roles: %w", err)
	}
	defer rows.Close()

	var roles []models.Role
	for rows.Next() {
		var role models.Role
		err := rows.Scan(
			&role.ID,
			&role.Name,
			&role.Description,
			&role.TenantID,
			&role.CreatedAt,
			&role.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role: %w", err)
		}
		roles = append(roles, role)
	}

	return roles, nil
}

// RequestRoleAssignment creates a role assignment request
func (eupr *EndUserPermissionRepository) RequestRoleAssignment(userID uuid.UUID, tenantID string, roleID uuid.UUID, reason string) error {
	query := `
INSERT INTO role_assignment_requests (id, user_id, tenant_id, role_id, reason, status, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, 'pending', $6, $6)
`

	requestID := uuid.New()
	now := time.Now()

	_, err := eupr.executeExec(query, requestID, userID, tenantID, roleID, reason, now)
	if err != nil {
		return fmt.Errorf("failed to create role assignment request: %w", err)
	}

	return nil
}

// GetUserRoleRequests retrieves role assignment requests for a user
func (eupr *EndUserPermissionRepository) GetUserRoleRequests(userID uuid.UUID, tenantID string) ([]models.RoleAssignmentRequest, error) {
	query := `
SELECT r.id, r.user_id, r.tenant_id, r.role_id, r.reason, r.status, r.created_at, r.updated_at
FROM role_assignment_requests r
WHERE r.user_id = $1 AND r.tenant_id = $2
ORDER BY r.created_at DESC
`

	rows, err := eupr.executeQuery(query, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to query role requests: %w", err)
	}
	defer rows.Close()

	var requests []models.RoleAssignmentRequest
	for rows.Next() {
		var request models.RoleAssignmentRequest
		err := rows.Scan(
			&request.ID,
			&request.UserID,
			&request.TenantID,
			&request.RoleID,
			&request.Reason,
			&request.Status,
			&request.CreatedAt,
			&request.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan role request: %w", err)
		}
		requests = append(requests, request)
	}

	return requests, nil
}
