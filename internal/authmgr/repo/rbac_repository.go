// Package authmgrrepo provides RBAC repository operations for the authmgr sub-service.
// It is ported from auth-manager's internal/repo package and adapted to use
// authsec's config.GetTenantGORMDB for database connectivity.
package authmgrrepo

import (
	"context"
	"strings"
	"time"

	authmgrmodels "github.com/authsec-ai/authsec/internal/authmgr/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Perm represents a resource with its allowed actions (used in authz checks).
type Perm struct {
	R string   `json:"r"`
	A []string `json:"a"`
}

// FromScopes converts scope strings like ["invoice:read","invoice:write"] into
// []Perm like [{R:"invoice", A:["read","write"]}].
func FromScopes(scopeNames []string) []Perm {
	type key struct{ r string }
	m := map[key]map[string]struct{}{}
	for _, sc := range scopeNames {
		parts := strings.SplitN(sc, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, a := parts[0], parts[1]
		k := key{r: r}
		if _, ok := m[k]; !ok {
			m[k] = map[string]struct{}{}
		}
		m[k][a] = struct{}{}
	}
	out := make([]Perm, 0, len(m))
	for k, acts := range m {
		as := make([]string, 0, len(acts))
		for a := range acts {
			as = append(as, a)
		}
		out = append(out, Perm{R: k.r, A: as})
	}
	return out
}

// DBProvider is a function that returns a GORM DB for a given tenantID.
type DBProvider func(tenantID string) (*gorm.DB, error)

// RBACRepository defines runtime permission/role check operations.
type RBACRepository interface {
	CheckPermission(ctx context.Context, tenantID, userID uuid.UUID, resource, action string) (bool, error)
	CheckPermissionWithScope(ctx context.Context, tenantID, userID uuid.UUID, resource, action string, scopeType string, scopeID *uuid.UUID) (bool, error)
	CheckOAuthScope(ctx context.Context, tenantID uuid.UUID, scopeName, resource, action string) (bool, error)
	CheckRole(ctx context.Context, tenantID, userID uuid.UUID, roleName string) (bool, error)
	CheckRoleResource(ctx context.Context, tenantID, userID uuid.UUID, roleName, scopeType string, scopeID uuid.UUID) (bool, error)
	GetUserPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]authmgrmodels.Permission, error)
	GetUserRoles(ctx context.Context, tenantID, userID uuid.UUID) ([]authmgrmodels.Role, error)
}

type rbacRepository struct {
	dbProvider DBProvider
}

// NewRBACRepository creates a new RBAC repository backed by the given DBProvider.
func NewRBACRepository(dbProvider DBProvider) RBACRepository {
	return &rbacRepository{dbProvider: dbProvider}
}

func (r *rbacRepository) CheckPermission(ctx context.Context, tenantID, userID uuid.UUID, resource, action string) (bool, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return false, err
	}

	var count int64
	err = db.WithContext(ctx).
		Table("role_bindings").
		Joins("JOIN roles ON role_bindings.role_id = roles.id AND role_bindings.tenant_id = roles.tenant_id").
		Joins("JOIN role_permissions ON roles.id = role_permissions.role_id").
		Joins("JOIN permissions ON role_permissions.permission_id = permissions.id").
		Where("role_bindings.tenant_id = ?", tenantID).
		Where("role_bindings.user_id = ?", userID).
		Where("permissions.resource = ?", resource).
		Where("permissions.action = ?", action).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > ?", time.Now()).
		Count(&count).Error
	return count > 0, err
}

func (r *rbacRepository) CheckPermissionWithScope(ctx context.Context, tenantID, userID uuid.UUID, resource, action string, scopeType string, scopeID *uuid.UUID) (bool, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return false, err
	}

	query := db.WithContext(ctx).
		Table("role_bindings").
		Joins("JOIN roles ON role_bindings.role_id = roles.id AND role_bindings.tenant_id = roles.tenant_id").
		Joins("JOIN role_permissions ON roles.id = role_permissions.role_id").
		Joins("JOIN permissions ON role_permissions.permission_id = permissions.id").
		Where("role_bindings.tenant_id = ?", tenantID).
		Where("role_bindings.user_id = ?", userID).
		Where("permissions.resource = ?", resource).
		Where("permissions.action = ?", action).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > ?", time.Now())

	if scopeID == nil {
		query = query.Where("role_bindings.scope_id IS NULL")
	} else {
		query = query.Where("role_bindings.scope_id IS NULL OR (role_bindings.scope_id = ? AND role_bindings.scope_type = ?)", *scopeID, scopeType)
	}

	var count int64
	err = query.Count(&count).Error
	return count > 0, err
}

func (r *rbacRepository) CheckOAuthScope(ctx context.Context, tenantID uuid.UUID, scopeName, resource, action string) (bool, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return false, err
	}

	var count int64
	err = db.WithContext(ctx).
		Table("scopes").
		Joins("JOIN scope_permissions ON scopes.id = scope_permissions.scope_id").
		Joins("JOIN permissions ON scope_permissions.permission_id = permissions.id").
		Where("scopes.tenant_id = ?", tenantID).
		Where("scopes.name = ?", scopeName).
		Where("permissions.resource = ?", resource).
		Where("permissions.action = ?", action).
		Count(&count).Error
	return count > 0, err
}

func (r *rbacRepository) CheckRole(ctx context.Context, tenantID, userID uuid.UUID, roleName string) (bool, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return false, err
	}

	var count int64
	err = db.WithContext(ctx).
		Table("role_bindings").
		Joins("JOIN roles ON role_bindings.role_id = roles.id AND role_bindings.tenant_id = roles.tenant_id").
		Where("role_bindings.tenant_id = ?", tenantID).
		Where("role_bindings.user_id = ?", userID).
		Where("roles.name = ?", roleName).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > ?", time.Now()).
		Count(&count).Error
	return count > 0, err
}

func (r *rbacRepository) CheckRoleResource(ctx context.Context, tenantID, userID uuid.UUID, roleName, scopeType string, scopeID uuid.UUID) (bool, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return false, err
	}

	var count int64
	err = db.WithContext(ctx).
		Table("role_bindings").
		Joins("JOIN roles ON role_bindings.role_id = roles.id AND role_bindings.tenant_id = roles.tenant_id").
		Where("role_bindings.tenant_id = ?", tenantID).
		Where("role_bindings.user_id = ?", userID).
		Where("roles.name = ?", roleName).
		Where("role_bindings.scope_type = ?", scopeType).
		Where("role_bindings.scope_id = ?", scopeID).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > ?", time.Now()).
		Count(&count).Error
	return count > 0, err
}

func (r *rbacRepository) GetUserPermissions(ctx context.Context, tenantID, userID uuid.UUID) ([]authmgrmodels.Permission, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return nil, err
	}

	var permissions []authmgrmodels.Permission
	err = db.WithContext(ctx).
		Table("permissions").
		Distinct("permissions.*").
		Joins("JOIN role_permissions ON permissions.id = role_permissions.permission_id").
		Joins("JOIN roles ON role_permissions.role_id = roles.id").
		Joins("JOIN role_bindings ON roles.id = role_bindings.role_id AND roles.tenant_id = role_bindings.tenant_id").
		Where("role_bindings.tenant_id = ?", tenantID).
		Where("role_bindings.user_id = ?", userID).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > ?", time.Now()).
		Find(&permissions).Error
	return permissions, err
}

func (r *rbacRepository) GetUserRoles(ctx context.Context, tenantID, userID uuid.UUID) ([]authmgrmodels.Role, error) {
	db, err := r.dbProvider(tenantID.String())
	if err != nil {
		return nil, err
	}

	var roles []authmgrmodels.Role
	err = db.WithContext(ctx).
		Table("roles").
		Distinct("roles.*").
		Joins("JOIN role_bindings ON roles.id = role_bindings.role_id AND roles.tenant_id = role_bindings.tenant_id").
		Where("role_bindings.tenant_id = ?", tenantID).
		Where("role_bindings.user_id = ?", userID).
		Where("role_bindings.expires_at IS NULL OR role_bindings.expires_at > ?", time.Now()).
		Find(&roles).Error
	return roles, err
}
