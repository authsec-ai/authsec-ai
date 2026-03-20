package services

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClientsPermissions defines all permissions for the clients service.
var ClientsPermissions = []struct {
	Resource    string
	Action      string
	Description string
}{
	{"clients", "create", "Create new client entries"},
	{"clients", "read", "Read client information"},
	{"clients", "update", "Update client information"},
	{"clients", "delete", "Delete client entries"},
	{"clients", "list", "List client entries"},
	{"clients", "activate", "Activate client entries"},
	{"clients", "deactivate", "Deactivate client entries"},
	{"clients", "admin", "Full administrative access to clients"},
	{"external-service", "create", "Create new external-service entries"},
	{"external-service", "read", "Read external-service information"},
	{"external-service", "update", "Update external-service information"},
	{"external-service", "delete", "Delete external-service entries"},
	{"external-service", "list", "List external-service entries"},
	{"external-service", "activate", "Activate external-service entries"},
	{"external-service", "deactivate", "Deactivate external-service entries"},
	{"external-service", "admin", "Administrative access to external-service"},
}

// SeedClientAdminRBAC ensures clients permissions exist for the tenant and are mapped to the admin role.
// It is idempotent and safe to call on each client creation.
//
// IMPORTANT: db should be the MAIN database connection (config.DB), NOT a tenant database.
func SeedClientAdminRBAC(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) error {
	if db == nil {
		return fmt.Errorf("db is required")
	}
	if tenantID == uuid.Nil {
		return fmt.Errorf("tenant_id is required")
	}

	tx := db.WithContext(ctx)

	var existingCount int64
	if err := tx.Raw(`
		SELECT COUNT(*) FROM permissions
		WHERE tenant_id = ? AND resource IN ('clients', 'external-service')
	`, tenantID).Scan(&existingCount).Error; err != nil {
		return fmt.Errorf("check existing permissions: %w", err)
	}

	expectedCount := int64(len(ClientsPermissions))
	if existingCount >= expectedCount {
		log.Printf("[RBAC-SEED] permissions already exist (%d/%d) for tenant %s, skipping", existingCount, expectedCount, tenantID)
		return nil
	}

	for _, p := range ClientsPermissions {
		if err := tx.Exec(`
			INSERT INTO permissions (id, tenant_id, resource, action, description, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, ?, NOW(), NOW())
			ON CONFLICT (tenant_id, resource, action)
			DO UPDATE SET description = EXCLUDED.description, updated_at = NOW()
		`, tenantID, p.Resource, p.Action, p.Description).Error; err != nil {
			return fmt.Errorf("seed permissions (%s:%s): %w", p.Resource, p.Action, err)
		}
	}

	var adminRoleID uuid.UUID
	if err := tx.Raw(`SELECT id FROM roles WHERE tenant_id = ? AND name = 'admin' LIMIT 1`, tenantID).
		Scan(&adminRoleID).Error; err != nil {
		return fmt.Errorf("fetch admin role: %w", err)
	}
	if adminRoleID == uuid.Nil {
		adminRoleID = uuid.New()
		if err := tx.Exec(`
			INSERT INTO roles (id, tenant_id, name, description, is_system, created_at, updated_at)
			VALUES (?, ?, 'admin', 'Tenant admin with full access', true, NOW(), NOW())
			ON CONFLICT (tenant_id, name) DO NOTHING
		`, adminRoleID, tenantID).Error; err != nil {
			return fmt.Errorf("create admin role: %w", err)
		}
		if err := tx.Raw(`SELECT id FROM roles WHERE tenant_id = ? AND name = 'admin' LIMIT 1`, tenantID).
			Scan(&adminRoleID).Error; err != nil {
			return fmt.Errorf("re-fetch admin role: %w", err)
		}
	}

	var permIDs []uuid.UUID
	if err := tx.Raw(`
		SELECT id FROM permissions
		WHERE tenant_id = ? AND resource IN ('clients', 'external-service')
	`, tenantID).Scan(&permIDs).Error; err != nil {
		return fmt.Errorf("fetch client permissions: %w", err)
	}

	for _, pid := range permIDs {
		if err := tx.Exec(`
			INSERT INTO role_permissions (role_id, permission_id)
			VALUES (?, ?)
			ON CONFLICT DO NOTHING
		`, adminRoleID, pid).Error; err != nil {
			return fmt.Errorf("bind admin role to permission %s: %w", pid.String(), err)
		}
	}

	log.Printf("[RBAC-SEED] Completed RBAC seeding for tenant %s", tenantID.String())
	return nil
}
