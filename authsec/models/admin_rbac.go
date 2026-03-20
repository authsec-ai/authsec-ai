package models

import (
	"time"

	"github.com/google/uuid"
)

// AdminRole represents an admin role in the master database
type AdminRole struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description *string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for AdminRole
func (AdminRole) TableName() string {
	return "roles"
}

// AdminPermission represents an admin permission in the master database
type AdminPermission struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoleID     uuid.UUID `json:"role_id" gorm:"type:uuid;not null"`
	ScopeID    uuid.UUID `json:"scope_id" gorm:"type:uuid;not null"`
	ResourceID uuid.UUID `json:"resource_id" gorm:"type:uuid;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for AdminPermission
func (AdminPermission) TableName() string {
	return "permissions"
}

// AdminScope represents an admin scope in the master database
type AdminScope struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description *string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for AdminScope
func (AdminScope) TableName() string {
	return "scopes"
}

// AdminResource represents an admin resource in the master database
type AdminResource struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description *string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for AdminResource
func (AdminResource) TableName() string {
	return "resources"
}

// AdminResourceMethod represents an admin resource method in the master database
type AdminResourceMethod struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ResourceID    uuid.UUID  `json:"resource_id" gorm:"type:uuid;not null"`
	Method        string     `json:"method" gorm:"type:varchar(10);not null"`
	PathPattern   string     `json:"path_pattern" gorm:"type:varchar(255);not null"`
	RequiresAdmin bool       `json:"requires_admin" gorm:"default:true"`
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for AdminResourceMethod
func (AdminResourceMethod) TableName() string {
	return "resource_methods"
}

// AdminUserRole represents admin user-role assignment in the master database
type AdminUserRole struct {
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;primaryKey"`
	RoleID    uuid.UUID  `json:"role_id" gorm:"type:uuid;not null;primaryKey"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for AdminUserRole
func (AdminUserRole) TableName() string {
	return "user_roles"
}

// Request/Response DTOs for Admin RBAC

// CreateAdminRoleRequest represents the request to create an admin role
type CreateAdminRoleRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdateAdminRoleRequest represents the request to update an admin role
type UpdateAdminRoleRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// AssignRoleToUserRequest represents the request to assign role to admin user
type AssignRoleToUserRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// CreateAdminPermissionRequest represents the request to create an admin permission
type CreateAdminPermissionRequest struct {
	RoleID     uuid.UUID `json:"role_id" binding:"required"`
	ScopeID    uuid.UUID `json:"scope_id" binding:"required"`
	ResourceID uuid.UUID `json:"resource_id" binding:"required"`
}

// DeleteAdminPermissionRequest represents the request to delete admin permissions
type DeleteAdminPermissionRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids" binding:"required"`
}

// CreateAdminScopeRequest represents the request to create an admin scope
type CreateAdminScopeRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdateAdminScopeRequest represents the request to update an admin scope
type UpdateAdminScopeRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// CreateAdminResourceRequest represents the request to create an admin resource
type CreateAdminResourceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdateAdminResourceRequest represents the request to update an admin resource
type UpdateAdminResourceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// CreateAdminResourceMethodRequest represents the request to create an admin resource method
type CreateAdminResourceMethodRequest struct {
	ResourceID    uuid.UUID `json:"resource_id" binding:"required"`
	Method        string    `json:"method" binding:"required"`
	PathPattern   string    `json:"path_pattern" binding:"required"`
	RequiresAdmin bool      `json:"requires_admin"`
}

// UpdateAdminResourceMethodRequest represents the request to update an admin resource method
type UpdateAdminResourceMethodRequest struct {
	ResourceID    uuid.UUID `json:"resource_id,omitempty"`
	Method        string    `json:"method,omitempty"`
	PathPattern   string    `json:"path_pattern,omitempty"`
	RequiresAdmin *bool     `json:"requires_admin,omitempty"`
}
