package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DelegationToken stores an active delegated JWT-SVID for an AI agent client.
// The SDK/agent pulls this row to get its current token and permissions.
// Upserted by DelegateToken, keyed by (tenant_id, client_id).
type DelegationToken struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;primaryKey"`
	TenantID    uuid.UUID       `json:"tenant_id" gorm:"type:uuid;not null;uniqueIndex:uq_delegation_token_client"`
	ClientID    uuid.UUID       `json:"client_id" gorm:"type:uuid;not null;uniqueIndex:uq_delegation_token_client"`
	PolicyID    *uuid.UUID      `json:"policy_id,omitempty" gorm:"type:uuid"`
	Token       string          `json:"token" gorm:"type:text;not null"`
	SpiffeID    string          `json:"spiffe_id" gorm:"type:text;not null"`
	Permissions json.RawMessage `json:"permissions" gorm:"type:jsonb;default:'[]'"`
	Audience    json.RawMessage `json:"audience" gorm:"type:jsonb;default:'[]'"`
	ExpiresAt   time.Time       `json:"expires_at" gorm:"type:timestamptz;not null"`
	DelegatedBy uuid.UUID       `json:"delegated_by" gorm:"type:uuid;not null"`
	TTLSeconds  int             `json:"ttl_seconds" gorm:"not null"`
	Status      string          `json:"status" gorm:"type:text;not null;default:'active'"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

func (DelegationToken) TableName() string {
	return "delegation_tokens"
}

func (dt *DelegationToken) BeforeCreate(tx *gorm.DB) error {
	if dt.ID == uuid.Nil {
		dt.ID = uuid.New()
	}
	return nil
}

// GetPermissions parses the JSONB permissions field into a string slice.
func (dt *DelegationToken) GetPermissions() []string {
	if len(dt.Permissions) == 0 {
		return nil
	}
	var perms []string
	if err := json.Unmarshal(dt.Permissions, &perms); err != nil {
		return nil
	}
	return perms
}

// GetAudience parses the JSONB audience field into a string slice.
func (dt *DelegationToken) GetAudience() []string {
	if len(dt.Audience) == 0 {
		return nil
	}
	var aud []string
	if err := json.Unmarshal(dt.Audience, &aud); err != nil {
		return nil
	}
	return aud
}

// IsExpired returns true if the token has passed its expiry time.
func (dt *DelegationToken) IsExpired() bool {
	return time.Now().After(dt.ExpiresAt)
}
