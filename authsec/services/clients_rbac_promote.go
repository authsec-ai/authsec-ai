package services

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PromoteExternalServicePermissions moves the system-tenant external-service resource/permissions
// to the given tenant (if that tenant doesn't already have them) and rebinds the role_permissions
// to the tenant's admin role. It performs only UPDATEs (no INSERTs) and is safe to call repeatedly.
func PromoteExternalServicePermissions(ctx context.Context, db *gorm.DB, tenantID uuid.UUID) error {
	if db == nil {
		return fmt.Errorf("db is required")
	}
	const sysTenant = "00000000-0000-0000-0000-000000000000"

	tx := db.WithContext(ctx)

	resources := []string{"external-service", "clients"}

	for _, resource := range resources {
		var count int64
		if err := tx.Table("permissions").
			Where("tenant_id = ? AND resource = ?", tenantID, resource).
			Count(&count).Error; err != nil {
			return fmt.Errorf("check tenant permissions for %s: %w", resource, err)
		}
		if count > 0 {
			continue
		}

		if err := tx.Exec(`
			UPDATE resources r
			SET tenant_id = ?
			WHERE r.name = ?
			  AND r.tenant_id = ?
			  AND NOT EXISTS (
				  SELECT 1 FROM resources r2
				  WHERE r2.name = ? AND r2.tenant_id = ?
			  )
		`, tenantID, resource, sysTenant, resource, tenantID).Error; err != nil {
			return fmt.Errorf("promote %s resource: %w", resource, err)
		}

		if err := tx.Exec(`
			UPDATE permissions p
			SET tenant_id = ?
			WHERE p.resource = ?
			  AND p.tenant_id = ?
			  AND NOT EXISTS (
				  SELECT 1 FROM permissions p2
				  WHERE p2.resource = ? AND p2.tenant_id = ?
			  )
		`, tenantID, resource, sysTenant, resource, tenantID).Error; err != nil {
			return fmt.Errorf("promote %s permissions: %w", resource, err)
		}
	}

	var adminRoleIDStr string
	if err := tx.Raw(`SELECT id FROM roles WHERE tenant_id = ? AND name = 'admin' LIMIT 1`, tenantID).
		Scan(&adminRoleIDStr).Error; err != nil {
		return fmt.Errorf("fetch tenant admin role: %w", err)
	}
	if adminRoleIDStr == "" {
		return fmt.Errorf("admin role not found for tenant %s", tenantID.String())
	}
	adminRoleID, err := uuid.Parse(adminRoleIDStr)
	if err != nil {
		return fmt.Errorf("parse admin role id: %w", err)
	}

	for _, resource := range resources {
		if err := tx.Exec(`
			UPDATE role_permissions rp
			SET role_id = ?
			WHERE rp.role_id IN (SELECT id FROM roles WHERE tenant_id = ? AND name = 'admin')
			  AND rp.permission_id IN (SELECT id FROM permissions WHERE tenant_id = ? AND resource = ?)
			  AND NOT EXISTS (
				  SELECT 1 FROM role_permissions rp2
				  WHERE rp2.role_id = ?
				    AND rp2.permission_id = rp.permission_id
			  )
		`, adminRoleID, sysTenant, tenantID, resource, adminRoleID).Error; err != nil {
			return fmt.Errorf("rebind role_permissions for %s to tenant admin: %w", resource, err)
		}
	}

	return nil
}
