package services

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// PermissionService handles permission resolution for users
type PermissionService struct {
	db *sql.DB
}

// NewPermissionService creates a new permission service instance
func NewPermissionService(db *sql.DB) *PermissionService {
	return &PermissionService{db: db}
}

// Permission represents a structured permission for JWT claims
type Permission struct {
	Resource string   `json:"r"`
	Actions  []string `json:"a"`
}

// GetUserPermissions returns structured permissions for a user in JWT-compatible format
// Uses role_bindings for role assignments (user_roles is deprecated)
func (ps *PermissionService) GetUserPermissions(userID, tenantID string) []Permission {
	query := `
		SELECT DISTINCT
			p.resource,
			p.action
		FROM role_bindings rb
		JOIN roles ro ON rb.role_id = ro.id
		JOIN role_permissions rp ON ro.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE rb.user_id::text = $1
		  AND (rb.tenant_id IS NULL OR rb.tenant_id::text = $2)
		  AND (ro.tenant_id IS NULL OR ro.tenant_id::text = $2)
		  AND (p.tenant_id IS NULL OR p.tenant_id::text = $2)
		ORDER BY p.resource, p.action
	`

	rows, err := ps.db.Query(query, userID, tenantID)
	if err != nil {
		log.Printf("Error querying user permissions: %v", err)
		return []Permission{}
	}
	defer rows.Close()

	// Group permissions by resource
	resourcePerms := make(map[string][]string)
	for rows.Next() {
		var resource, action string
		if err := rows.Scan(&resource, &action); err != nil {
			log.Printf("Error scanning permission row: %v", err)
			continue
		}
		resourcePerms[resource] = append(resourcePerms[resource], action)
	}

	// Convert to Permission structs
	var permissions []Permission
	for resource, actions := range resourcePerms {
		permissions = append(permissions, Permission{
			Resource: resource,
			Actions:  actions,
		})
	}

	return permissions
}

// GetUserScopes returns scopes in string format for JWT claims (resource:action format)
// Uses role_bindings for role assignments (user_roles is deprecated)
func (ps *PermissionService) GetUserScopes(userID, tenantID string) []string {
	query := `
		SELECT DISTINCT
			CONCAT(p.resource, ':', p.action) as scope_string
		FROM role_bindings rb
		JOIN roles ro ON rb.role_id = ro.id
		JOIN role_permissions rp ON ro.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE rb.user_id::text = $1
		  AND (rb.tenant_id IS NULL OR rb.tenant_id::text = $2)
		  AND (ro.tenant_id IS NULL OR ro.tenant_id::text = $2)
		  AND (p.tenant_id IS NULL OR p.tenant_id::text = $2)
		ORDER BY scope_string
	`

	rows, err := ps.db.Query(query, userID, tenantID)
	if err != nil {
		log.Printf("Error querying user scopes: %v", err)
		return []string{}
	}
	defer rows.Close()

	var scopes []string
	for rows.Next() {
		var scope string
		if err := rows.Scan(&scope); err != nil {
			log.Printf("Error scanning scope row: %v", err)
			continue
		}
		scopes = append(scopes, scope)
	}

	return scopes
}

// ResolvePermissionsFromRoles returns permissions for given role names
// Uses the main database schema: roles -> role_permissions -> permissions
func (ps *PermissionService) ResolvePermissionsFromRoles(roleNames []string, tenantID string) ([]Permission, []string) {
	if len(roleNames) == 0 {
		return []Permission{}, []string{}
	}

	// Create placeholders for IN clause
	placeholders := make([]string, len(roleNames))
	args := make([]interface{}, len(roleNames)+1)
	args[0] = tenantID

	for i, role := range roleNames {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = role
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT
			p.resource,
			p.action,
			CONCAT(p.resource, ':', p.action) as scope_string
		FROM roles ro
		JOIN role_permissions rp ON ro.id = rp.role_id
		JOIN permissions p ON rp.permission_id = p.id
		WHERE ro.name IN (%s)
		  AND (ro.tenant_id IS NULL OR ro.tenant_id::text = $1)
		  AND (p.tenant_id IS NULL OR p.tenant_id::text = $1)
		ORDER BY p.resource, p.action
	`, strings.Join(placeholders, ","))

	rows, err := ps.db.Query(query, args...)
	if err != nil {
		log.Printf("Error querying permissions from roles: %v", err)
		return []Permission{}, []string{}
	}
	defer rows.Close()

	// Group permissions by resource and collect scopes
	resourcePerms := make(map[string][]string)
	var scopes []string

	for rows.Next() {
		var resource, action, scopeString string
		if err := rows.Scan(&resource, &action, &scopeString); err != nil {
			log.Printf("Error scanning role permission row: %v", err)
			continue
		}
		resourcePerms[resource] = append(resourcePerms[resource], action)
		scopes = append(scopes, scopeString)
	}

	// Convert to Permission structs
	var permissions []Permission
	for resource, actions := range resourcePerms {
		permissions = append(permissions, Permission{
			Resource: resource,
			Actions:  actions,
		})
	}

	return permissions, scopes
}
