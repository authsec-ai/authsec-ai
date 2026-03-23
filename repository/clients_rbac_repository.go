package repositories

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ClientsRBACRepository defines the interface for clients RBAC-related operations
type ClientsRBACRepository interface {
	GrantUserClientsAccess(ctx context.Context, userID, tenantID uuid.UUID) error
}

// clientsRbacRepository implements ClientsRBACRepository
type clientsRbacRepository struct {
	db *gorm.DB
}

// NewClientsRBACRepository creates a new instance of ClientsRBACRepository
func NewClientsRBACRepository(db *gorm.DB) ClientsRBACRepository {
	return &clientsRbacRepository{db: db}
}

// GrantUserClientsAccess ensures the tenant has the necessary 'clients' permissions,
// an admin role exists, and the user is bound to that role.
func (r *clientsRbacRepository) GrantUserClientsAccess(ctx context.Context, userID, tenantID uuid.UUID) error {
	if r.db == nil {
		return fmt.Errorf("database connection is nil")
	}

	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		permissions := []struct {
			Action      string
			Description string
		}{
			{"create", "Create new clients"},
			{"read", "View client details"},
			{"update", "Modify client details"},
			{"delete", "Delete clients"},
			{"list", "List all clients"},
			{"activate", "Activate clients"},
			{"deactivate", "Deactivate clients"},
			{"admin", "Full administrative access to clients"},
		}

		for _, p := range permissions {
			var permID uuid.UUID
			err := tx.Table("permissions").Select("id").Where("resource = 'clients' AND action = ? AND tenant_id = ?", p.Action, tenantID).Scan(&permID).Error
			if err != nil {
				return err
			}
			if permID == uuid.Nil {
				if err := tx.Exec(`
					INSERT INTO permissions (id, tenant_id, resource, action, description, created_at, updated_at)
					VALUES (gen_random_uuid(), ?, 'clients', ?, ?, NOW(), NOW())
					ON CONFLICT (tenant_id, resource, action) DO NOTHING
				`, tenantID, p.Action, p.Description).Error; err != nil {
					return fmt.Errorf("insert permission (clients:%s): %w", p.Action, err)
				}
				if err := tx.Table("permissions").Select("id").Where("resource = 'clients' AND action = ? AND tenant_id = ?", p.Action, tenantID).Scan(&permID).Error; err != nil {
					return err
				}
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
				VALUES (?, ?, 'admin', 'Tenant admin', true, NOW(), NOW())
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
		if err := tx.Raw(`SELECT id FROM permissions WHERE tenant_id = ? AND resource = 'clients'`, tenantID).
			Scan(&permIDs).Error; err != nil {
			return fmt.Errorf("fetch clients permissions: %w", err)
		}
		for _, pid := range permIDs {
			if err := tx.Exec(`INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?) ON CONFLICT DO NOTHING`, adminRoleID, pid).Error; err != nil {
				return fmt.Errorf("bind permission %s to admin role: %w", pid, err)
			}
		}

		if err := tx.Exec(`
			INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, conditions, created_at, updated_at)
			VALUES (gen_random_uuid(), ?, ?, ?, NULL, NULL, '{}'::jsonb, NOW(), NOW())
			ON CONFLICT DO NOTHING
		`, tenantID, userID, adminRoleID).Error; err != nil {
			return fmt.Errorf("bind user to admin role: %w", err)
		}

		log.Printf("[ClientsRBAC] Granted clients access to user %s in tenant %s", userID, tenantID)
		return nil
	})
}
