package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/authsec-ai/sharedmodels"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// TenantMapping represents the mapping between tenants and clients
type TenantMapping struct {
	ID        uuid.UUID `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ClientID  uuid.UUID `json:"client_id" gorm:"type:uuid;uniqueIndex;not null"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginResponse represents the response for login operations
type LoginResponse struct {
	TenantID         string   `json:"tenant_id"`
	TenantDomain     string   `json:"tenant_domain,omitempty"`
	Email            string   `json:"email"`
	FirstLogin       bool     `json:"first_login"`
	OTPRequired      bool     `json:"otp_required"`
	WebAuthnRequired bool     `json:"webauthn_required,omitempty"`
	MFARequired      bool     `json:"mfa_required"`
	MFAMethod        string   `json:"mfa_method,omitempty"`
	Methods          []string `json:"methods,omitempty"`
	Token            string   `json:"token,omitempty"`
}

// LoginVerifyOTPInput represents the input for login OTP verification
type LoginVerifyOTPInput struct {
	Email    string `json:"email" binding:"required,email"`
	OTP      string `json:"otp" binding:"required,len=6"`
	TenantID string `json:"tenant_id" binding:"required"`
}

// LoginVerifyOTPResponse represents the response for login OTP verification
type LoginVerifyOTPResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
}

// ExtendedUser extends sharedmodels.User with additional fields used by user-flow
type ExtendedUser struct {
	sharedmodels.User

	// AD/Entra sync specific fields
	ExternalID   *string    `json:"external_id,omitempty" gorm:"column:external_id"`
	SyncSource   *string    `json:"sync_source,omitempty" gorm:"column:sync_source"`
	LastSyncAt   *time.Time `json:"last_sync_at,omitempty" gorm:"column:last_sync_at"`
	IsSyncedUser bool       `json:"is_synced_user" gorm:"column:is_synced_user;default:false"`

	// Brute force protection fields
	FailedLoginAttempts int        `json:"failed_login_attempts" gorm:"column:failed_login_attempts;default:0"`
	AccountLockedAt     *time.Time `json:"account_locked_at,omitempty" gorm:"column:account_locked_at"`
	PasswordResetRequired bool     `json:"password_reset_required" gorm:"column:password_reset_required;default:false"`
}

// TableName specifies the table name for ExtendedUser
func (ExtendedUser) TableName() string {
	return "users"
}

// HashPassword hashes the user's password before saving
func (u *ExtendedUser) HashPassword() error {
	if u.PasswordHash == "" {
		return fmt.Errorf("password cannot be empty")
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(u.PasswordHash), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedBytes)
	return nil
}

// CheckPassword compares the provided password with the stored hash
func (u *ExtendedUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// AdminUser represents an administrative user in the global database
type AdminUser struct {
	ID                         uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Email                      string          `json:"email" gorm:"uniqueIndex;not null"`
	Username                   string          `json:"username" gorm:"uniqueIndex;not null"`
	Password                   string          `json:"-" gorm:"-"` // Plain password for input
	PasswordHash               string          `json:"-" gorm:"column:password_hash;not null"`
	Name                       string          `json:"name"`
	ClientID                   *uuid.UUID      `json:"client_id,omitempty" gorm:"column:client_id"`
	TenantID                   *uuid.UUID      `json:"tenant_id,omitempty" gorm:"column:tenant_id"`
	ProjectID                  *uuid.UUID      `json:"project_id,omitempty" gorm:"column:project_id"`
	TenantDomain               string          `json:"tenant_domain,omitempty" gorm:"column:tenant_domain"`
	Provider                   string          `json:"provider,omitempty" gorm:"column:provider;default:'local'"`
	ProviderID                 string          `json:"provider_id,omitempty" gorm:"column:provider_id"`
	ProviderData               json.RawMessage `json:"provider_data,omitempty" gorm:"column:provider_data;type:jsonb"`
	AvatarURL                  string          `json:"avatar_url,omitempty" gorm:"column:avatar_url"`
	Active                     bool            `json:"active" gorm:"default:true"`
	MFAEnabled                 bool            `json:"mfa_enabled,omitempty" gorm:"column:mfa_enabled;default:false"`
	MFAMethod                  []string        `json:"mfa_method,omitempty" gorm:"column:mfa_method;type:text[]"`
	MFADefaultMethod           string          `json:"mfa_default_method,omitempty" gorm:"column:mfa_default_method"`
	MFAEnrolledAt              *time.Time      `json:"mfa_enrolled_at,omitempty" gorm:"column:mfa_enrolled_at"`
	MFAVerified                bool            `json:"mfa_verified,omitempty" gorm:"column:mfa_verified;default:false"`
	ExternalID                 string          `json:"external_id,omitempty" gorm:"column:external_id"`
	SyncSource                 string          `json:"sync_source,omitempty" gorm:"column:sync_source"`
	LastSyncAt                 *time.Time      `json:"last_sync_at,omitempty" gorm:"column:last_sync_at"`
	IsSyncedUser               bool            `json:"is_synced_user" gorm:"column:is_synced_user;default:false"`
	LastLogin                  *time.Time      `json:"last_login,omitempty" gorm:"column:last_login"`
	TemporaryPassword          bool            `json:"temporary_password" gorm:"column:temporary_password;default:false"`
	TemporaryPasswordExpiresAt *time.Time      `json:"temporary_password_expires_at,omitempty" gorm:"column:temporary_password_expires_at"`
	IsPrimaryAdmin             bool            `json:"is_primary_admin" gorm:"column:is_primary_admin;default:false"`
	FailedLoginAttempts        int             `json:"failed_login_attempts" gorm:"column:failed_login_attempts;default:0"`
	AccountLockedAt            *time.Time      `json:"account_locked_at,omitempty" gorm:"column:account_locked_at"`
	PasswordResetRequired      bool            `json:"password_reset_required" gorm:"column:password_reset_required;default:false"`
	CreatedAt                  time.Time       `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt                  time.Time       `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for AdminUser
func (AdminUser) TableName() string {
	return "users"
}

// HashPassword hashes the admin user's password before saving
func (u *AdminUser) HashPassword() error {
	if u.Password == "" {
		return fmt.Errorf("password cannot be empty")
	}
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedBytes)
	u.Password = "" // Clear plain password
	return nil
}

// CheckPassword compares the provided password with the stored hash
func (u *AdminUser) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// RoleAssignmentRequest represents a request for role assignment
type RoleAssignmentRequest struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null"`
	TenantID  string    `json:"tenant_id" gorm:"type:varchar(255);not null"`
	RoleID    uuid.UUID `json:"role_id" gorm:"type:uuid;not null"`
	Reason    string    `json:"reason" gorm:"type:text"`
	Status    string    `json:"status" gorm:"type:varchar(50);default:'pending'"` // pending, approved, rejected
	CreatedAt time.Time `json:"created_at" gorm:"default:CURRENT_TIMESTAMP"`
	UpdatedAt time.Time `json:"updated_at" gorm:"default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for RoleAssignmentRequest
func (RoleAssignmentRequest) TableName() string {
	return "role_assignment_requests"
}

// SAMLLoginInput represents the input for SAML login (no password required)
type SAMLLoginInput struct {
	ClientID string `json:"client_id" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	// Provider is validated in the database to ensure it ends with "-saml"
}

// UpdateUserRequest represents the request payload for updating user profile
type UpdateUserRequest struct {
	Name         *string `json:"name,omitempty"`
	Username     *string `json:"username,omitempty"`
	Email        *string `json:"email,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	TenantDomain *string `json:"tenant_domain,omitempty"`
}

// GetAuthURLInput represents input for constructing the Auth URL
type GetAuthURLInput struct {
	ClientID string `json:"client_id" binding:"required"`
}

// GetAuthURLResponse represents the output containing the constructed Auth URL
type GetAuthURLResponse struct {
	AuthURL string `json:"auth_url"`
	State   string `json:"state"`
}
