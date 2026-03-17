package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AdminSeedRepository handles per-tenant admin role/scopes/permissions seeding.
type AdminSeedRepository struct {
	db *DBConnection
}

type sqlExecutor interface {
	QueryRow(query string, args ...any) *sql.Row
	Exec(query string, args ...any) (sql.Result, error)
}

func NewAdminSeedRepository(db *DBConnection) *AdminSeedRepository {
	return &AdminSeedRepository{db: db}
}

// EnsureAdminRoleAndPermissions creates admin role, default scopes, and permissions for a tenant and returns the role ID.
func (asr *AdminSeedRepository) EnsureAdminRoleAndPermissions(tenantID uuid.UUID) (uuid.UUID, error) {
	return asr.ensureAdminRoleAndPermissions(asr.db.DB, tenantID)
}

// EnsureAdminRoleAndPermissionsTx performs the same operation using an existing transaction.
func (asr *AdminSeedRepository) EnsureAdminRoleAndPermissionsTx(tx *sql.Tx, tenantID uuid.UUID) (uuid.UUID, error) {
	return asr.ensureAdminRoleAndPermissions(tx, tenantID)
}

func (asr *AdminSeedRepository) ensureAdminRoleAndPermissions(exec sqlExecutor, tenantID uuid.UUID) (uuid.UUID, error) {
	if asr == nil || asr.db == nil {
		return uuid.Nil, fmt.Errorf("admin seed repository not initialized")
	}
	if exec == nil {
		return uuid.Nil, fmt.Errorf("executor is nil")
	}

	now := time.Now()

	// Ensure admin role - use index name for ON CONFLICT (different DBs may have different constraint names)
	roleID := uuid.New()
	if err := exec.QueryRow(`
		INSERT INTO roles (id, tenant_id, name, description, created_at, updated_at)
		VALUES ($1, $2, 'admin', 'Administrator with full access', $3, $3)
		ON CONFLICT (tenant_id, name) DO UPDATE SET updated_at = EXCLUDED.updated_at
		RETURNING id
	`, roleID, tenantID, now).Scan(&roleID); err != nil {
		return uuid.Nil, fmt.Errorf("ensure admin role: %w", err)
	}

	// Seed baseline permissions for admin role
	permRows := []struct {
		Resource string
		Action   string
	}{
		{"admin", "access"},
		{"users", "create"}, {"users", "read"}, {"users", "update"}, {"users", "delete"}, {"users", "manage"},
		{"tenants", "create"}, {"tenants", "read"}, {"tenants", "update"}, {"tenants", "delete"}, {"tenants", "manage"},
		{"projects", "create"}, {"projects", "read"}, {"projects", "update"}, {"projects", "delete"}, {"projects", "manage"},
		{"roles", "create"}, {"roles", "read"}, {"roles", "update"}, {"roles", "delete"}, {"roles", "manage"},
		{"permissions", "create"}, {"permissions", "read"}, {"permissions", "update"}, {"permissions", "delete"}, {"permissions", "manage"},
		{"scopes", "create"}, {"scopes", "read"}, {"scopes", "update"}, {"scopes", "delete"}, {"scopes", "manage"},
		{"role-bindings", "create"}, {"role-bindings", "read"}, {"role-bindings", "update"}, {"role-bindings", "delete"}, {"role-bindings", "manage"},
		{"policy", "create"}, {"policy", "read"}, {"policy", "update"}, {"policy", "delete"}, {"policy", "manage"},
		{"groups", "create"}, {"groups", "read"}, {"groups", "update"}, {"groups", "delete"}, {"groups", "manage"},
		{"sync", "create"}, {"sync", "read"}, {"sync", "update"}, {"sync", "delete"}, {"sync", "manage"},
		{"sync-configs", "create"}, {"sync-configs", "read"}, {"sync-configs", "update"}, {"sync-configs", "delete"}, {"sync-configs", "manage"},
		{"oidc", "create"}, {"oidc", "read"}, {"oidc", "update"}, {"oidc", "delete"}, {"oidc", "manage"},
		{"endusers", "create"}, {"endusers", "read"}, {"endusers", "update"}, {"endusers", "delete"}, {"endusers", "manage"},
		{"clients", "create"}, {"clients", "read"}, {"clients", "update"}, {"clients", "delete"}, {"clients", "manage"},
		{"user-endusers", "create"}, {"user-endusers", "read"}, {"user-endusers", "update"}, {"user-endusers", "delete"}, {"user-endusers", "manage"},
		{"user-rbac-roles", "create"}, {"user-rbac-roles", "read"}, {"user-rbac-roles", "update"}, {"user-rbac-roles", "delete"}, {"user-rbac-roles", "manage"},
		{"user-rbac-permissions", "create"}, {"user-rbac-permissions", "read"}, {"user-rbac-permissions", "update"}, {"user-rbac-permissions", "delete"}, {"user-rbac-permissions", "manage"},
		{"user-rbac-scopes", "create"}, {"user-rbac-scopes", "read"}, {"user-rbac-scopes", "update"}, {"user-rbac-scopes", "delete"}, {"user-rbac-scopes", "manage"},
		{"user-permissions", "create"}, {"user-permissions", "read"}, {"user-permissions", "update"}, {"user-permissions", "delete"}, {"user-permissions", "manage"},
		{"user-groups", "create"}, {"user-groups", "read"}, {"user-groups", "update"}, {"user-groups", "delete"}, {"user-groups", "manage"},
		{"user-clients", "create"}, {"user-clients", "read"}, {"user-clients", "update"}, {"user-clients", "delete"}, {"user-clients", "manage"},
		{"user-projects", "create"}, {"user-projects", "read"}, {"user-projects", "update"}, {"user-projects", "delete"}, {"user-projects", "manage"},
		{"scopes", "create"}, {"scopes", "read"}, {"scopes", "update"}, {"scopes", "delete"}, {"scopes", "manage_permissions"},
		{"health", "read"},
		{"external-service", "create"}, {"external-service", "read"}, {"external-service", "update"}, {"external-service", "delete"}, {"external-service", "credentials"}, {"external-service", "manage"},
	}

	for _, p := range permRows {
		permID := uuid.New()
		if err := exec.QueryRow(`
			INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
			VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (tenant_id, resource, action) DO UPDATE SET description = EXCLUDED.description
			RETURNING id
		`, permID, tenantID, p.Resource, p.Action, fmt.Sprintf("%s %s", p.Resource, p.Action), now).Scan(&permID); err != nil {
			return uuid.Nil, fmt.Errorf("ensure permission %s:%s: %w", p.Resource, p.Action, err)
		}

		if _, err := exec.Exec(`
			INSERT INTO role_permissions (role_id, permission_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, roleID, permID); err != nil {
			return uuid.Nil, fmt.Errorf("bind permission %s:%s: %w", p.Resource, p.Action, err)
		}

	}

	return roleID, nil
}
