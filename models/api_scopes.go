package models

import (
	"time"

	"github.com/google/uuid"
)

// APIScope represents an OAuth scope contract that maps to internal permissions.
// These are high-level keys given to external applications (e.g., "files:read")
// that translate to one or more internal RBAC permissions.
type APIScope struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id" gorm:"type:uuid;uniqueIndex:idx_api_scopes_tenant_name;uniqueIndex:idx_api_scopes_tenant_id"`
	Name        string     `json:"name" gorm:"type:text;not null;uniqueIndex:idx_api_scopes_tenant_name"` // e.g., "files:read", "project:write"
	Description string     `json:"description" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`

	// Relations - mapped internal permissions
	Permissions []RBACPermission `json:"permissions,omitempty" gorm:"many2many:api_scope_permissions;joinForeignKey:ScopeID;joinReferences:PermissionID"`
}

func (APIScope) TableName() string {
	return "api_scopes"
}

// APIScopePermission represents the many-to-many link between API Scopes and Permissions.
// This mapping defines which internal permissions are granted when an OAuth client
// is authorized with a particular scope.
type APIScopePermission struct {
	ScopeID      uuid.UUID `json:"scope_id" gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:uuid;primaryKey"`
}

func (APIScopePermission) TableName() string {
	return "api_scope_permissions"
}

// --- Request/Response DTOs ---

// CreateAPIScopeRequest represents the payload for creating an API scope.
// Example: {"name": "files:read", "description": "Read access to files", "mapped_permission_ids": ["uuid1", "uuid2"]}
type CreateAPIScopeRequest struct {
	Name                string   `json:"name" binding:"required"`
	Description         string   `json:"description"`
	MappedPermissionIDs []string `json:"mapped_permission_ids"` // UUIDs of permissions to link
}

// UpdateAPIScopeRequest represents the payload for updating an API scope.
type UpdateAPIScopeRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	MappedPermissionIDs []string `json:"mapped_permission_ids"` // Replaces all existing mappings
}

// APIScopeResponse represents the response for API scope operations.
type APIScopeResponse struct {
	ID                 string   `json:"id"`
	Name               string   `json:"name"`
	Description        string   `json:"description"`
	PermissionsLinked  int      `json:"permissions_linked"`
	PermissionIDs      []string `json:"permission_ids,omitempty"`
	PermissionStrings  []string `json:"permission_strings,omitempty"` // e.g., ["project:read", "invoice:read"]
	CreatedAt          string   `json:"created_at,omitempty"`
}

// APIScopeListItem represents a scope in list responses with summary info.
type APIScopeListItem struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	PermissionsLinked int      `json:"permissions_linked"`
	CreatedAt         string   `json:"created_at"`
}
