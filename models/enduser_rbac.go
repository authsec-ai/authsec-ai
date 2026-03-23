package models

import (
	"time"

	"github.com/google/uuid"
)

// EndUserRole represents an end user role in the tenant database
type EndUserRole struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description *string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for EndUserRole
func (EndUserRole) TableName() string {
	return "roles"
}

// EndUserPermission represents an end user permission in the tenant database
type EndUserPermission struct {
	ID         uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	RoleID     uuid.UUID `json:"role_id" gorm:"type:uuid;not null"`
	ScopeID    uuid.UUID `json:"scope_id" gorm:"type:uuid;not null"`
	ResourceID uuid.UUID `json:"resource_id" gorm:"type:uuid;not null"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for EndUserPermission
func (EndUserPermission) TableName() string {
	return "permissions"
}

// EndUserScope represents an end user scope in the tenant database
type EndUserScope struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description *string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for EndUserScope
func (EndUserScope) TableName() string {
	return "scopes"
}

// EndUserResource represents an end user resource in the tenant database
type EndUserResource struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"type:varchar(100);not null"`
	Description *string    `json:"description,omitempty" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for EndUserResource
func (EndUserResource) TableName() string {
	return "resources"
}

// EndUserResourceMethod represents an end user resource method in the tenant database
type EndUserResourceMethod struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	ResourceID    uuid.UUID  `json:"resource_id" gorm:"type:uuid;not null"`
	Method        string     `json:"method" gorm:"type:varchar(10);not null"`
	PathPattern   string     `json:"path_pattern" gorm:"type:varchar(255);not null"`
	RequiresAdmin bool       `json:"requires_admin" gorm:"default:false"`
	CreatedAt     time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for EndUserResourceMethod
func (EndUserResourceMethod) TableName() string {
	return "resource_methods"
}

// EndUserUserRole represents end user-role assignment in the tenant database
type EndUserUserRole struct {
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;primaryKey"`
	RoleID    uuid.UUID  `json:"role_id" gorm:"type:uuid;not null;primaryKey"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
}

// TableName specifies the table name for EndUserUserRole
func (EndUserUserRole) TableName() string {
	return "user_roles"
}

// Request/Response DTOs for End User RBAC (Admin-facing)

// CreateEndUserRoleRequest represents the request to create an end user role (admin operation)
type CreateEndUserRoleRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdateEndUserRoleRequest represents the request to update an end user role (admin operation)
type UpdateEndUserRoleRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// AssignEndUserRoleRequest represents the request to assign role to end user (admin operation)
type AssignEndUserRoleRequest struct {
	UserID uuid.UUID `json:"user_id" binding:"required"`
}

// CreateEndUserPermissionRequest represents the request to create an end user permission (admin operation)
type CreateEndUserPermissionRequest struct {
	RoleID     uuid.UUID `json:"role_id" binding:"required"`
	ScopeID    uuid.UUID `json:"scope_id" binding:"required"`
	ResourceID uuid.UUID `json:"resource_id" binding:"required"`
}

// DeleteEndUserPermissionRequest represents the request to delete end user permissions (admin operation)
type DeleteEndUserPermissionRequest struct {
	PermissionIDs []uuid.UUID `json:"permission_ids" binding:"required"`
}

// CreateEndUserScopeRequest represents the request to create an end user scope (admin operation)
type CreateEndUserScopeRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   *string  `json:"description,omitempty"`
	PermissionIDs []string `json:"permission_ids,omitempty"`
}

// UpdateEndUserScopeRequest represents the request to update an end user scope (admin operation)
type UpdateEndUserScopeRequest struct {
	Name          string   `json:"name" binding:"required"`
	Description   *string  `json:"description,omitempty"`
	PermissionIDs []string `json:"permission_ids,omitempty"`
}

// EndUserScopeWithPermissions represents a scope and its mapped permission IDs
type EndUserScopeWithPermissions struct {
	ID            uuid.UUID   `json:"id"`
	TenantID      *uuid.UUID  `json:"tenant_id,omitempty"`
	Name          string      `json:"name"`
	Description   *string     `json:"description,omitempty"`
	PermissionIDs []uuid.UUID `json:"permission_ids"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     *time.Time  `json:"updated_at,omitempty"`
}

// CreateEndUserResourceRequest represents the request to create an end user resource (admin operation)
type CreateEndUserResourceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// UpdateEndUserResourceRequest represents the request to update an end user resource (admin operation)
type UpdateEndUserResourceRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description *string `json:"description,omitempty"`
}

// CreateEndUserResourceMethodRequest represents the request to create an end user resource method (admin operation)
type CreateEndUserResourceMethodRequest struct {
	ResourceID    uuid.UUID `json:"resource_id" binding:"required"`
	Method        string    `json:"method" binding:"required"`
	PathPattern   string    `json:"path_pattern" binding:"required"`
	RequiresAdmin bool      `json:"requires_admin"`
}

// UpdateEndUserResourceMethodRequest represents the request to update an end user resource method (admin operation)
type UpdateEndUserResourceMethodRequest struct {
	ResourceID    uuid.UUID `json:"resource_id,omitempty"`
	Method        string    `json:"method,omitempty"`
	PathPattern   string    `json:"path_pattern,omitempty"`
	RequiresAdmin *bool     `json:"requires_admin,omitempty"`
}
