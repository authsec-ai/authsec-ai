package models

import (
	"encoding/json"
	"time"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
)

// UserWithJSONMFAMethods loads users.mfa_method stored as JSONB and converts it
// to the sharedmodels.User text[] representation after queries.
type UserWithJSONMFAMethods struct {
	ID               uuid.UUID      `gorm:"column:id"`
	ClientID         uuid.UUID      `gorm:"column:client_id"`
	TenantID         uuid.UUID      `gorm:"column:tenant_id"`
	ProjectID        uuid.UUID      `gorm:"column:project_id"`
	Name             string         `gorm:"column:name"`
	Username         *string        `gorm:"column:username"`
	Email            string         `gorm:"column:email"`
	PasswordHash     string         `gorm:"column:password_hash"`
	TenantDomain     string         `gorm:"column:tenant_domain"`
	Provider         string         `gorm:"column:provider"`
	ProviderID       string         `gorm:"column:provider_id"`
	ProviderData     datatypes.JSON `gorm:"column:provider_data"`
	AvatarURL        *string        `gorm:"column:avatar_url"`
	Active           bool           `gorm:"column:active"`
	MFAEnabled       bool           `gorm:"column:mfa_enabled"`
	MFADefaultMethod *string        `gorm:"column:mfa_default_method"`
	MFAEnrolledAt    *time.Time     `gorm:"column:mfa_enrolled_at"`
	MFAVerified      bool           `gorm:"column:mfa_verified"`
	CreatedAt        time.Time      `gorm:"column:created_at"`
	UpdatedAt        time.Time      `gorm:"column:updated_at"`
	LastLogin        *time.Time     `gorm:"column:last_login"`
	ExternalID       *string        `gorm:"column:external_id"`
	SyncSource       *string        `gorm:"column:sync_source"`
	LastSyncAt       *time.Time     `gorm:"column:last_sync_at"`
	IsSyncedUser     bool           `gorm:"column:is_synced_user"`
	MFAMethodJSON    datatypes.JSON `gorm:"column:mfa_method"`
}

// TableName allows GORM to map the projection to the users table.
func (*UserWithJSONMFAMethods) TableName() string {
	return "users"
}

// ToShared converts the projection to the sharedmodels.User struct.
func (u UserWithJSONMFAMethods) ToShared() sharedmodels.User {
	user := sharedmodels.User{
		ID:               u.ID,
		ClientID:         u.ClientID,
		TenantID:         u.TenantID,
		ProjectID:        u.ProjectID,
		Name:             u.Name,
		Username:         u.Username,
		Email:            u.Email,
		PasswordHash:     u.PasswordHash,
		TenantDomain:     u.TenantDomain,
		Provider:         u.Provider,
		ProviderID:       u.ProviderID,
		ProviderData:     u.ProviderData,
		AvatarURL:        u.AvatarURL,
		Active:           u.Active,
		MFAEnabled:       u.MFAEnabled,
		MFADefaultMethod: u.MFADefaultMethod,
		MFAEnrolledAt:    u.MFAEnrolledAt,
		MFAVerified:      u.MFAVerified,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
		LastLogin:        u.LastLogin,
		ExternalID:       u.ExternalID,
		SyncSource:       u.SyncSource,
		LastSyncAt:       u.LastSyncAt,
		IsSyncedUser:     u.IsSyncedUser,
	}

	if len(u.MFAMethodJSON) > 0 {
		var methods []string
		if err := json.Unmarshal(u.MFAMethodJSON, &methods); err == nil {
			user.MFAMethod = pq.StringArray(methods)
		}
	}

	return user
}
