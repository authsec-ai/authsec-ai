package models

import (
	"time"

	"github.com/google/uuid"
)

// ScopeResourceMapping represents the mapping between a scope and a resource
type ScopeResourceMapping struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null"`
	ScopeName    string    `json:"scope_name" gorm:"not null;default:'*'"`
	ResourceName string    `json:"resource_name" gorm:"not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for ScopeResourceMapping
func (ScopeResourceMapping) TableName() string {
	return "scope_resource_mappings"
}
