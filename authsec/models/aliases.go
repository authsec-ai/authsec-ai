package models

import (
	"time"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/google/uuid"
)
//aa
type (
	Tenant                               = sharedmodels.Tenant
	RegisterResponse                     = sharedmodels.RegisterResponse
	// InitiateRegistrationInput is overridden locally to make FirstName and LastName optional
	// Original: InitiateRegistrationInput = sharedmodels.InitiateRegistrationInput
	InitiateRegistrationResponse         = sharedmodels.InitiateRegistrationResponse
	VerifyOTPInput                       = sharedmodels.VerifyOTPInput
	ResendOTPInput                       = sharedmodels.ResendOTPInput
	LoginInput                           = sharedmodels.LoginInput
	WebAuthnCallbackInput                = sharedmodels.WebAuthnCallbackInput
	RegisterClientsRequest               = sharedmodels.RegisterClientsRequest
	RegisterClientsResponse              = sharedmodels.RegisterClientsResponse
	PaginatedEndUsersResponse            = sharedmodels.PaginatedEndUsersResponse
	UpdateEndUserStatusInput             = sharedmodels.UpdateEndUserStatusInput
	UpdateEndUserStatusResponse          = sharedmodels.UpdateEndUserStatusResponse
	OIDCLoginInput                       = sharedmodels.OIDCLoginInput
	OIDCLoginResponse                    = sharedmodels.OIDCLoginResponse
	CustomLoginInput                     = sharedmodels.CustomLoginInput
	CustomLoginStatus                    = sharedmodels.CustomLoginStatus
	CustomLoginRegister                  = sharedmodels.CustomLoginRegister
	CustomForgotPasswordInput            = sharedmodels.CustomForgotPasswordInput
	CustomForgotPasswordResponse         = sharedmodels.CustomForgotPasswordResponse
	CustomVerifyPasswordResetOTPInput    = sharedmodels.CustomVerifyPasswordResetOTPInput
	CustomVerifyPasswordResetOTPResponse = sharedmodels.CustomVerifyPasswordResetOTPResponse
	CustomResetPasswordInput             = sharedmodels.CustomResetPasswordInput
	CustomResetPasswordResponse          = sharedmodels.CustomResetPasswordResponse
	AdminChangePasswordInput             = sharedmodels.AdminChangePasswordInput
	AdminChangePasswordResponse          = sharedmodels.AdminChangePasswordResponse
	AdminResetPasswordInput              = sharedmodels.AdminResetPasswordInput
	AdminResetPasswordResponse           = sharedmodels.AdminResetPasswordResponse
	AdminForgotPasswordInput             = sharedmodels.AdminForgotPasswordInput
	AdminForgotPasswordResponse          = sharedmodels.AdminForgotPasswordResponse
	AdminVerifyPasswordResetOTPInput     = sharedmodels.AdminVerifyPasswordResetOTPInput
	AdminVerifyPasswordResetOTPResponse  = sharedmodels.AdminVerifyPasswordResetOTPResponse
	AdminResetPasswordInput2             = sharedmodels.AdminResetPasswordInput2
	AdminResetPasswordResponse2          = sharedmodels.AdminResetPasswordResponse2
	TokenVerifyRequest                   = sharedmodels.TokenVerifyRequest
	TokenVerifyResponse                  = sharedmodels.TokenVerifyResponse
	TokenRequest                         = sharedmodels.TokenRequest
	TokenResponse                        = sharedmodels.TokenResponse
	User                                 = sharedmodels.User
	Client                               = sharedmodels.Client
	Role                                 = sharedmodels.Role
	Scope                                = sharedmodels.Scope
	Resource                             = sharedmodels.Resource
	MFAMethod                            = sharedmodels.MFAMethod
	Credential                           = sharedmodels.Credential
	Introspection                        = sharedmodels.Introspection
	Project                              = sharedmodels.Project
	ProjectInput                         = sharedmodels.ProjectInput
	ProjectResponse                      = sharedmodels.ProjectResponse
	UserDefinedGroupsRequest             = sharedmodels.UserDefinedGroupsRequest
	MapGroupsRequest                     = sharedmodels.MapGroupsRequest
	DeleteGroupsRequest                  = sharedmodels.DeleteGroupsRequest
	UserDefinedResourcesRequest          = sharedmodels.UserDefinedResourcesRequest
	MapResourcesRequest                  = sharedmodels.MapResourcesRequest
	DeleteResourcesRequest               = sharedmodels.DeleteResourcesRequest
	UserDefinedRolesRequest              = sharedmodels.UserDefinedRolesRequest
	MapRolesRequest                      = sharedmodels.MapRolesRequest
	// DeleteRolesRequest is overridden locally to support both "roles" and "role_ids" field names
	// Original: DeleteRolesRequest = sharedmodels.DeleteRolesRequest
	UserDefinedScopesRequest             = sharedmodels.UserDefinedScopesRequest
	MapScopesRequest                     = sharedmodels.MapScopesRequest
	DeleteScopesRequest                  = sharedmodels.DeleteScopesRequest
	Service                              = sharedmodels.Service
	Permission                           = sharedmodels.Permission
	ResourceMethod                       = sharedmodels.ResourceMethod
)

// OTPEntry represents a stored one-time password record.
type OTPEntry struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email     string    `json:"email"`
	OTP       string    `json:"-"` // never expose OTP value
	ExpiresAt time.Time `json:"expires_at"`
	Verified  bool      `json:"verified" gorm:"default:false"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PendingRegistration represents a staged registration waiting for OTP confirmation.
type PendingRegistration struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FirstName    string    `json:"first_name"`
	LastName     string    `json:"last_name"`
	TenantID     uuid.UUID `json:"tenant_id" gorm:"type:uuid"`
	ProjectID    uuid.UUID `json:"project_id" gorm:"type:uuid"`
	ClientID     uuid.UUID `json:"client_id" gorm:"type:uuid"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	TenantDomain string    `json:"tenant_domain"`
}

// TenantGroup represents a user-defined group within a tenant
type TenantGroup struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Name        string    `json:"name" gorm:"not null"`
	Description *string   `json:"description"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Group struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID    *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	Name        string     `json:"name" gorm:"uniqueIndex;not null"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   *time.Time `json:"updated_at,omitempty"`
}

func (Group) TableName() string {
	return "groups"
}

type RemoveGroupsRequest struct {
	TenantID string   `json:"tenant_id" binding:"required"`
	ClientID string   `json:"client_id" binding:"required"`
	Groups   []string `json:"groups" binding:"required"`
}

// UserGroup represents the many-to-many relationship between users and groups
type UserGroup struct {
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;primaryKey;not null"`
	GroupID   uuid.UUID  `json:"group_id" gorm:"type:uuid;primaryKey;not null"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt *time.Time `json:"updated_at,omitempty" gorm:"autoUpdateTime"`
}

// TableName specifies the table name for UserGroup
func (UserGroup) TableName() string {
	return "user_groups"
}

// TableName specifies the table name for TenantGroup
func (TenantGroup) TableName() string {
	return "groups"
}

// AddUsersToGroupRequest represents a request to add multiple users to a group
type AddUsersToGroupRequest struct {
	GroupID uuid.UUID   `json:"group_id" binding:"required"`
	UserIDs []uuid.UUID `json:"user_ids" binding:"required,min=1"`
}

// RemoveUsersFromGroupRequest represents a request to remove multiple users from a group
type RemoveUsersFromGroupRequest struct {
	GroupID uuid.UUID   `json:"group_id" binding:"required"`
	UserIDs []uuid.UUID `json:"user_ids" binding:"required,min=1"`
}

// InitiateRegistrationInput with optional name fields (FirstName and LastName are not required)
// This overrides sharedmodels.InitiateRegistrationInput to match our database schema
type InitiateRegistrationInput struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=10"`
	FirstName    string `json:"first_name"` // Optional - no binding:"required"
	LastName     string `json:"last_name"`  // Optional - no binding:"required"
	TenantDomain string `json:"tenant_domain" binding:"required"`
}

// DeleteRolesRequest with support for both "roles" and "role_ids" field names
// This overrides sharedmodels.DeleteRolesRequest to handle legacy clients sending "role_ids"
type DeleteRolesRequest struct {
	TenantID  string   `json:"tenant_id" binding:"required"`
	ProjectID string   `json:"project_id" binding:"required"`
	Roles     []string `json:"roles" binding:"required"`
	RoleIDs   []string `json:"role_ids"` // Alternative field name for backward compatibility
}

// GetRoles returns the roles list, preferring Roles but falling back to RoleIDs if Roles is empty
func (r *DeleteRolesRequest) GetRoles() []string {
	if len(r.Roles) > 0 {
		return r.Roles
	}
	return r.RoleIDs
}


// RegisterClientWithHydra is now implemented in hydra.go
// No need to delegate to sharedmodels
