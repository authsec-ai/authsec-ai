// Package authmgrmodels contains GORM models for auth-manager RBAC tables.
// These models mirror the schema managed by auth-manager and are used by
// the authmgr controller for runtime permission and role checks.
package authmgrmodels

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ServiceAccount represents a service account principal.
type ServiceAccount struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null"`
	Name        string    `gorm:"not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Role represents a collection of permissions.
type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null"`
	Name        string    `gorm:"not null"`
	Description string
	IsSystem    bool      `gorm:"default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Permissions []Permission `gorm:"many2many:role_permissions;"`
}

// Permission represents an atomic resource+action pair.
type Permission struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null"`
	Resource    string    `gorm:"not null"`
	Action      string    `gorm:"not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// RolePermission is the many-to-many join between Role and Permission.
type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primary_key"`
	PermissionID uuid.UUID `gorm:"type:uuid;primary_key"`
}

// Scope represents a named collection of permissions (e.g. for OAuth).
type Scope struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    uuid.UUID `gorm:"type:uuid;not null"`
	Name        string    `gorm:"not null"`
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Permissions []Permission `gorm:"many2many:scope_permissions;"`
}

// ScopePermission is the many-to-many join between Scope and Permission.
type ScopePermission struct {
	ScopeID      uuid.UUID `gorm:"type:uuid;primary_key"`
	PermissionID uuid.UUID `gorm:"type:uuid;primary_key"`
}

// RoleBinding assigns a Role to a user (or service account) within a tenant,
// optionally scoped to a specific resource.
type RoleBinding struct {
	ID       uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `gorm:"type:uuid;not null"`

	UserID           *uuid.UUID `gorm:"type:uuid"`
	ServiceAccountID *uuid.UUID `gorm:"type:uuid"`

	RoleID uuid.UUID `gorm:"type:uuid;not null"`

	ScopeType *string    `gorm:"type:text"`
	ScopeID   *uuid.UUID `gorm:"type:uuid"`

	Conditions datatypes.JSON `gorm:"type:jsonb;default:'{}'"`
	ExpiresAt  *time.Time
	CreatedBy  *uuid.UUID `gorm:"type:uuid"`
	CreatedAt  time.Time
	UpdatedAt  time.Time

	Role Role `gorm:"foreignKey:RoleID"`
}

// GrantAudit logs permission grant/revoke changes.
type GrantAudit struct {
	ID          uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID    *uuid.UUID     `gorm:"type:uuid"`
	ActorUserID *uuid.UUID     `gorm:"type:uuid"`
	Action      string         `gorm:"type:text"`
	TargetType  string         `gorm:"type:text"`
	TargetID    *uuid.UUID     `gorm:"type:uuid"`
	Before      datatypes.JSON `gorm:"type:jsonb"`
	After       datatypes.JSON `gorm:"type:jsonb"`
	CreatedAt   time.Time
}

func (ServiceAccount) TableName() string  { return "service_accounts" }
func (Role) TableName() string            { return "roles" }
func (Permission) TableName() string      { return "permissions" }
func (RolePermission) TableName() string  { return "role_permissions" }
func (Scope) TableName() string           { return "scopes" }
func (ScopePermission) TableName() string { return "scope_permissions" }
func (RoleBinding) TableName() string     { return "role_bindings" }
func (GrantAudit) TableName() string      { return "grant_audit" }
