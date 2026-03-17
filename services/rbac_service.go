package services

import (
	"fmt"
	"log"
	"strings"

	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RBACService struct {
	db *gorm.DB
}

func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{db: db}
}

// ListPermissions lists permissions filtering by resource and optionally by tenant (nil for global)
// Note: In the atomic model, we might list all.
func (s *RBACService) ListPermissions(resource string) ([]models.RBACPermission, error) {
	var permissions []models.RBACPermission
	query := s.db
	if resource != "" {
		query = query.Where("resource = ?", resource)
	}
	// This implementation assumes listing all permissions the context has access to.
	// If needed, we can add TenantID filter as an argument.
	if err := query.Find(&permissions).Error; err != nil {
		return nil, err
	}
	return permissions, nil
}

func (s *RBACService) DeletePermission(permID uuid.UUID) error {
	return s.db.Delete(&models.RBACPermission{}, "id = ?", permID).Error
}

func (s *RBACService) DeleteRole(roleID uuid.UUID) error {
	return s.db.Delete(&models.RBACRole{}, "id = ?", roleID).Error
}

func (s *RBACService) GetRole(roleID uuid.UUID) (*models.RBACRole, error) {
	var role models.RBACRole
	if err := s.db.Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		return nil, err
	}
	return &role, nil
}

// CreateRoleComposite creates a Role and links Permissions in a single transaction.
func (s *RBACService) CreateRoleComposite(role *models.RBACRole, permissionIDs []uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. Insert into 'roles'
		if err := tx.Create(role).Error; err != nil {
			return err
		}

		// 2. Insert into 'role_permissions' for each ID in array
		if len(permissionIDs) > 0 {
			var rolePermissions []models.RolePermission
			for _, permID := range permissionIDs {
				rolePermissions = append(rolePermissions, models.RolePermission{
					RoleID:       role.ID,
					PermissionID: permID,
				})
			}
			if err := tx.Create(&rolePermissions).Error; err != nil {
				return err
			}
		}

		// Reload role with count if needed, but the caller usually handles response
		return nil
	})
}

// AssignRoleScoped grants access by binding a User to a Role within a specific Scope.
func (s *RBACService) AssignRoleScoped(binding *models.RoleBinding) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Validates that 'user' and 'role' belong to the same tenant.
		// Note: The DB Foreign Key constraints (tenant_id, user_id) already enforce this.
		// We can add an explicit check here if we want friendlier error messages,
		// but standard DB constraints are robust.

		// Insert into 'role_bindings'
		if err := tx.Create(binding).Error; err != nil {
			// Retry omitting optional denormalized columns when schema lacks them
			// Check for PostgreSQL error codes and column name errors
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "username") || 
			   strings.Contains(errStr, "role_name") || 
			   strings.Contains(errStr, "42703") { // PostgreSQL error code for "column does not exist"
				log.Printf("[RBACService] Schema missing username/role_name columns, retrying with Omit: %v", err)
				if retryErr := tx.Omit("Username", "RoleName").Create(binding).Error; retryErr == nil {
					log.Printf("[RBACService] Successfully created role binding without denormalized columns")
					return nil
				} else {
					log.Printf("[RBACService] Retry with Omit failed: %v", retryErr)
					return retryErr
				}
			}
			return err
		}
		return nil
	})
}

// RegisterAtomicPermission defines a new capability in the system.
func (s *RBACService) RegisterAtomicPermission(perm *models.RBACPermission) error {
	// Insert into 'permissions'. Failure if resource+action pair exists (handled by DB unique constraint).
	return s.db.Create(perm).Error
}

// PolicyDecisionPointCheck verifies if a user can perform an action on a specific resource.
//
// Documentation:
// This function implements the Core Authorization Engine (Policy Decision Point).
// It verifies if a Principal (User or Service Account) has permission to perform an Action on a Resource.
//
// Logic:
// 1. Identifies all Role Bindings for the Principal.
// 2. Filters Bindings based on Scope:
//   - If scopeID is provided (e.g. Project UUID), checks for bindings with that scopeID OR Tenant-Wide bindings (scope_id IS NULL).
//   - If scopeID is nil (Tenant-Level check), checks only for Tenant-Wide bindings.
//
// 3. Joins Bindings -> Roles -> RolePermissions -> Permissions.
// 4. Checks if any Permission matches the requested Resource and Action.
//
// Usage for External Services:
// External services (e.g. OIDC Provider, API Gateway) should call the `/uflow/policy/check` endpoint
// which wraps this function.
// - Payload: { "principal_id": "...", "resource": "project", "action": "write", "scope_id": "..." }
// - Response: { "allowed": true, "trace": "..." }
type PolicyCheckResult struct {
	Allowed bool
	Trace   string
}

func (s *RBACService) PolicyDecisionPointCheck(principalID uuid.UUID, resource, action string, scopeID *uuid.UUID) (*PolicyCheckResult, error) {
	// Query role_bindings -> Join roles -> Join role_permissions -> Join permissions
	// Match on Scope ID or NULL (Tenant-Wide)

	var results []struct {
		RoleName  string
		BindingID uuid.UUID
	}

	// We need to check if there is ANY binding that grants this permission
	// The query logic:
	// Find bindings for this user OR service account
	//   Where binding.scope_id matches requested scopeID OR binding.scope_id IS NULL
	//   Join Role -> RolePermissions -> Permission
	//   Where Permission matches resource AND action

	// We handle the scope_id match carefully.
	// If the request has a scope_id (e.g. project-alpha), we allow bindings that are EITHER:
	// 1. Specific to project-alpha
	// 2. Tenant-wide or Global (NULL scope_id)

	query := s.db.Table("role_bindings rb").
		Select("r.name as role_name, rb.id as binding_id").
		Joins("JOIN roles r ON rb.role_id = r.id").
		Joins("JOIN role_permissions rp ON r.id = rp.role_id").
		Joins("JOIN permissions p ON rp.permission_id = p.id").
		Where("(rb.user_id = ? OR rb.service_account_id = ?)", principalID, principalID).
		Where("p.resource = ? AND p.action = ?", resource, action)

	if scopeID != nil {
		query = query.Where("(rb.scope_id = ? OR rb.scope_id IS NULL)", scopeID)
	} else {
		// If checking for general access (no specific scope requested), we might only accept tenant-wide/global bindings?
		// Or maybe any binding? usually PDP checks against a specific target.
		// If target is tenant-level, then we check for NULL scope_id.
		query = query.Where("rb.scope_id IS NULL")
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	if len(results) > 0 {
		return &PolicyCheckResult{
			Allowed: true,
			Trace:   fmt.Sprintf("Granted by Binding [%s] via Role [%s]", results[0].BindingID, results[0].RoleName),
		}, nil
	}

	return &PolicyCheckResult{
		Allowed: false,
		Trace:   "No matching binding found",
	}, nil
}
