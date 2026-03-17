package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RBACRole represents a role in the RBAC system
type RBACRole struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    *uuid.UUID `json:"tenant_id" gorm:"type:uuid;uniqueIndex:idx_roles_tenant_name;uniqueIndex:idx_roles_tenant_id"`
	Name        string     `json:"name" gorm:"type:text;not null;uniqueIndex:idx_roles_tenant_name"`
	Description string     `json:"description" gorm:"type:text"`
	IsSystem    bool       `json:"is_system" gorm:"default:false"`
	CreatedAt   time.Time  `json:"created_at"`

	// Relations
	Permissions []RBACPermission `json:"permissions" gorm:"many2many:role_permissions;joinForeignKey:RoleID;joinReferences:PermissionID"`
}

func (RBACRole) TableName() string {
	return "roles"
}

// BeforeCreate hook ensures ID is set before inserting into database
func (r *RBACRole) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// RBACPermission represents an atomic resource-action capability
type RBACPermission struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    *uuid.UUID `json:"tenant_id" gorm:"type:uuid;uniqueIndex:idx_permissions_tenant_resource_action"`
	Resource    string     `json:"resource" gorm:"type:text;not null;uniqueIndex:idx_permissions_tenant_resource_action"`
	Action      string     `json:"action" gorm:"type:text;not null;uniqueIndex:idx_permissions_tenant_resource_action"`
	Description string     `json:"description" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (RBACPermission) TableName() string {
	return "permissions"
}

// BeforeCreate hook ensures ID is set before inserting into database
func (p *RBACPermission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

// RolePermission represents the many-to-many link between Roles and Permissions
type RolePermission struct {
	RoleID       uuid.UUID `json:"role_id" gorm:"type:uuid;primaryKey"`
	PermissionID uuid.UUID `json:"permission_id" gorm:"type:uuid;primaryKey"`
}

func (RolePermission) TableName() string {
	return "role_permissions"
}

// ServiceAccount represents a non-human identity
type ServiceAccount struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id" gorm:"type:uuid;uniqueIndex:idx_sa_tenant_id"`
	Name        string     `json:"name" gorm:"type:text;not null"`
	Description string     `json:"description" gorm:"type:text"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// RoleBinding represents an assignment of a Role to a Principal (User or Service Account)
type RoleBinding struct {
	ID               uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID         *uuid.UUID      `json:"tenant_id" gorm:"type:uuid"`
	UserID           *uuid.UUID      `json:"user_id" gorm:"type:uuid"`
	Username         string          `json:"username" gorm:"type:text"`
	ServiceAccountID *uuid.UUID      `json:"service_account_id" gorm:"type:uuid"`
	RoleID           uuid.UUID       `json:"role_id" gorm:"type:uuid;not null"`
	RoleName         string          `json:"role_name" gorm:"type:text"`
	ScopeType        *string         `json:"scope_type" gorm:"type:text"`
	ScopeID          *uuid.UUID      `json:"scope_id" gorm:"type:uuid"`
	Conditions       json.RawMessage `json:"conditions" gorm:"type:jsonb;default:'{}'"`
	ExpiresAt        *time.Time      `json:"expires_at"`
	CreatedBy        *uuid.UUID      `json:"created_by" gorm:"type:uuid"`
	CreatedAt        time.Time       `json:"created_at"`

	// Relations
	// We need to be careful with GORM relationships and composite keys.
	// Since the DB enforces composite keys, we should reflect that here if we want GORM to handle preloading correctly.
	// However, for simple referencing, standard ID referencing works if the DB constraint handles the integrity.
	Role           *RBACRole       `json:"role,omitempty" gorm:"foreignKey:RoleID;references:ID"`
	User           *ExtendedUser   `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	ServiceAccount *ServiceAccount `json:"service_account,omitempty" gorm:"foreignKey:ServiceAccountID;references:ID"`
}

func (RoleBinding) TableName() string {
	return "role_bindings"
}

// GrantAudit represents the audit log for role assignments
// Note: This table was removed in the migration to ensure strict schema adherence to the prompt.
// If it's not in the prompt, I should probably remove it or keep it separate.
// The prompt didn't ask for it, but didn't forbid it. I dropped it in the migration.
// So I will remove the struct to avoid confusion.
