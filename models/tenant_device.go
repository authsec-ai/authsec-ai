package models

import (
	"time"

	"github.com/google/uuid"
)

// ========================================
// Tenant Device Token for Push Notifications
// ========================================

// TenantDeviceToken represents a registered device for push notifications in tenant DB
type TenantDeviceToken struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID      uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID    uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	DeviceToken string    `json:"device_token" gorm:"uniqueIndex;size:500;not null"`
	Platform    string    `json:"platform" gorm:"size:20;not null"` // ios, android

	// Device metadata
	DeviceName  string `json:"device_name,omitempty" gorm:"size:100"`
	DeviceModel string `json:"device_model,omitempty" gorm:"size:100"`
	AppVersion  string `json:"app_version,omitempty" gorm:"size:20"`
	OSVersion   string `json:"os_version,omitempty" gorm:"size:20"`

	// Status
	IsActive bool   `json:"is_active" gorm:"default:true;not null;index"`
	LastUsed *int64 `json:"last_used,omitempty"`

	// Timestamps (Unix epoch)
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// TableName specifies the table name for TenantDeviceToken
func (TenantDeviceToken) TableName() string {
	return "tenant_device_tokens"
}

// ========================================
// Tenant CIBA Authentication Request
// ========================================

// TenantCIBAAuthRequest represents a CIBA authentication request in tenant DB
type TenantCIBAAuthRequest struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	AuthReqID string    `json:"auth_req_id" gorm:"uniqueIndex;size:255;not null"`

	// User identification
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserEmail string    `json:"user_email" gorm:"size:255;not null"`

	// Client information
	ClientID *uuid.UUID `json:"client_id,omitempty" gorm:"type:uuid"`

	// Push notification
	DeviceTokenID  uuid.UUID `json:"device_token_id" gorm:"type:uuid;not null"`
	BindingMessage string    `json:"binding_message,omitempty" gorm:"size:255"`

	// OAuth scopes
	Scopes JSONStringArray `json:"scopes" gorm:"type:jsonb;default:'[]'"`

	// Status: pending, approved, denied, expired, consumed
	Status string `json:"status" gorm:"size:50;default:'pending';not null;index"`

	// Biometric verification
	BiometricVerified bool `json:"biometric_verified" gorm:"default:false"`

	// Timestamps (Unix epoch)
	ExpiresAt    int64  `json:"expires_at" gorm:"not null;index"`
	CreatedAt    int64  `json:"created_at"`
	RespondedAt  *int64 `json:"responded_at,omitempty"`
	LastPolledAt *int64 `json:"last_polled_at,omitempty"`
}

// TableName specifies the table name for TenantCIBAAuthRequest
func (TenantCIBAAuthRequest) TableName() string {
	return "tenant_ciba_auth_requests"
}

// IsExpired checks if the CIBA request has expired
func (c *TenantCIBAAuthRequest) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// IsPending checks if still waiting for user response
func (c *TenantCIBAAuthRequest) IsPending() bool {
	return c.Status == "pending" && !c.IsExpired()
}

// IsApproved checks if user approved the request
func (c *TenantCIBAAuthRequest) IsApproved() bool {
	return c.Status == "approved"
}

// ========================================
// Tenant TOTP Authentication
// ========================================

// TenantTOTPSecret represents a TOTP authenticator device in tenant DB
type TenantTOTPSecret struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID   uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// TOTP secret (base32 encoded)
	Secret string `json:"-" gorm:"size:64;not null"` // Never expose this in JSON

	// Device metadata
	DeviceName string `json:"device_name" gorm:"size:100"`
	DeviceType string `json:"device_type" gorm:"size:50;default:'generic'"` // generic, google_auth, microsoft_auth, authy
	LastUsed   *int64 `json:"last_used"`

	// Status
	IsActive  bool `json:"is_active" gorm:"default:true;index"`
	IsPrimary bool `json:"is_primary" gorm:"default:false"` // Preferred device for TOTP

	// Timestamps (Unix epoch)
	CreatedAt int64 `json:"created_at" gorm:"not null"`
	UpdatedAt int64 `json:"updated_at" gorm:"not null"`
}

// TableName specifies the table name for TenantTOTPSecret
func (TenantTOTPSecret) TableName() string {
	return "tenant_totp_secrets"
}

// TenantBackupCode represents a recovery backup code for TOTP in tenant DB
type TenantBackupCode struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Code      string    `json:"code" gorm:"size:32;not null;uniqueIndex"` // Hashed code
	IsUsed    bool      `json:"is_used" gorm:"default:false;index"`
	CreatedAt int64     `json:"created_at" gorm:"not null"`
	UsedAt    *int64    `json:"used_at,omitempty"`
}

// TableName specifies the table name for TenantBackupCode
func (TenantBackupCode) TableName() string {
	return "tenant_totp_backup_codes"
}

// ========================================
// DTOs for Tenant Device Authentication
// ========================================

// TenantDeviceTokenRegistrationRequest - Mobile app registers device for push notifications in tenant context
type TenantDeviceTokenRegistrationRequest struct {
	DeviceToken string `json:"device_token" binding:"required"` // Expo Push Token
	Platform    string `json:"platform" binding:"required"`     // ios, android
	DeviceName  string `json:"device_name,omitempty"`
	DeviceModel string `json:"device_model,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	OSVersion   string `json:"os_version,omitempty"`
}

// TenantDeviceTokenRegistrationResponse
type TenantDeviceTokenRegistrationResponse struct {
	Success  bool   `json:"success"`
	DeviceID string `json:"device_id,omitempty"`
	Message  string `json:"message"`
}

// TenantCIBAInitiateRequest - Initiate CIBA authentication for tenant user
type TenantCIBAInitiateRequest struct {
	ClientID       string   `json:"client_id" binding:"required"`   // Maps to tenant
	Email          string   `json:"email" binding:"required,email"` // User email
	TenantDomain   string   `json:"tenant_domain"`                  // Optional: For validation if provided
	BindingMessage string   `json:"binding_message,omitempty"`      // Message shown to user
	Scopes         []string `json:"scopes,omitempty"`               // OAuth scopes
}

// TenantCIBAInitiateResponse
type TenantCIBAInitiateResponse struct {
	AuthReqID        string `json:"auth_req_id"`                 // Request ID for polling
	ExpiresIn        int    `json:"expires_in"`                  // Seconds until expiration
	Interval         int    `json:"interval"`                    // Polling interval in seconds
	Message          string `json:"message,omitempty"`           // Status message
	Error            string `json:"error,omitempty"`             // Error code
	ErrorDescription string `json:"error_description,omitempty"` // Error description
}

// TenantCIBARespondRequest - User approves/denies via mobile app
type TenantCIBARespondRequest struct {
	AuthReqID         string `json:"auth_req_id" binding:"required"`
	Approved          bool   `json:"approved"`
	BiometricVerified bool   `json:"biometric_verified,omitempty"`
}

// TenantCIBARespondResponse
type TenantCIBARespondResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TenantCIBATokenRequest - Client polls for token
type TenantCIBATokenRequest struct {
	AuthReqID string `json:"auth_req_id" binding:"required"`
	ClientID  string `json:"client_id" binding:"required"`
}

// TenantCIBATokenResponse - Returns token or status
type TenantCIBATokenResponse struct {
	AccessToken      string `json:"access_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	ExpiresIn        int    `json:"expires_in,omitempty"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	Scope            string `json:"scope,omitempty"`
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// TenantTOTPLoginRequest represents a TOTP login for tenant user
type TenantTOTPLoginRequest struct {
	ClientID     string `json:"client_id" binding:"required"`       // Maps to tenant
	Email        string `json:"email" binding:"required,email"`     // User email
	TOTPCode     string `json:"totp_code" binding:"required,len=6"` // 6-digit TOTP code
	TenantDomain string `json:"tenant_domain"`                      // For validation
}

// TenantTOTPLoginResponse contains TOTP login result
type TenantTOTPLoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"` // JWT token if login successful
	Message string `json:"message,omitempty"`
}

// TenantTOTPRegistrationRequest represents a request to register a new TOTP device in tenant context
type TenantTOTPRegistrationRequest struct {
	DeviceName string `json:"device_name" binding:"required"` // e.g., "My iPhone", "Personal iPad"
	DeviceType string `json:"device_type"`                    // optional: generic, google_auth, microsoft_auth, authy
}

// TenantTOTPRegistrationResponse contains the QR code data for device registration in tenant context
type TenantTOTPRegistrationResponse struct {
	Success     bool     `json:"success"`
	Secret      string   `json:"secret"`       // TOTP secret (shown only once)
	QRCodeURL   string   `json:"qr_code_url"`  // URL for QR code generation
	DeviceID    string   `json:"device_id"`    // Device ID for confirmation
	BackupCodes []string `json:"backup_codes"` // Recovery codes (shown only once)
	Message     string   `json:"message"`
}

// TenantTOTPRegistrationConfirmRequest confirms device registration after QR code scan
type TenantTOTPRegistrationConfirmRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	TOTPCode string `json:"totp_code" binding:"required,len=6"` // 6-digit code to verify setup
}

// TenantTOTPRegistrationConfirmResponse confirms successful registration
type TenantTOTPRegistrationConfirmResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

// TenantTOTPDeviceListResponse lists user's registered TOTP devices in tenant context
type TenantTOTPDeviceListResponse struct {
	Success bool               `json:"success"`
	Devices []TenantTOTPSecret `json:"devices"`
	Message string             `json:"message,omitempty"`
}

// TenantTOTPDeviceDeleteRequest deletes a registered device
type TenantTOTPDeviceDeleteRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// TenantTOTPDeviceDeleteResponse confirms deletion
type TenantTOTPDeviceDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TenantTOTPDeviceApprovalRequest approves device code using TOTP in tenant context
type TenantTOTPDeviceApprovalRequest struct {
	UserCode     string `json:"user_code" binding:"required"`       // Device user code to approve
	ClientID     string `json:"client_id" binding:"required"`       // Maps to tenant
	Email        string `json:"email" binding:"required,email"`     // User email (to find user and validate TOTP)
	TOTPCode     string `json:"totp_code" binding:"required,len=6"` // 6-digit TOTP code from authenticator app
	TenantDomain string `json:"tenant_domain" binding:"required"`   // For validation
}

// TenantTOTPDeviceApprovalResponse contains approval result
type TenantTOTPDeviceApprovalResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"` // JWT token if device approved
}

// TenantBackupCodeRegenerateRequest regenerates backup codes
type TenantBackupCodeRegenerateRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// TenantBackupCodeRegenerateResponse contains new backup codes
type TenantBackupCodeRegenerateResponse struct {
	Success     bool     `json:"success"`
	BackupCodes []string `json:"backup_codes"`
	Message     string   `json:"message"`
}

// Tenant CIBA Error Codes (RFC 8628 + custom)
// TenantDeviceSummary for API response (omits sensitive device_token)
type TenantDeviceSummary struct {
	ID          string  `json:"id"`
	DeviceName  string  `json:"device_name,omitempty"`
	Platform    string  `json:"platform"`
	DeviceModel string  `json:"device_model,omitempty"`
	AppVersion  string  `json:"app_version,omitempty"`
	OSVersion   string  `json:"os_version,omitempty"`
	IsActive    bool    `json:"is_active"`
	LastUsed    *int64  `json:"last_used,omitempty"`
	CreatedAt   int64   `json:"created_at"`
}

type TenantDeviceListResponse struct {
	Success bool                  `json:"success"`
	Devices []TenantDeviceSummary `json:"devices"`
	Message string                `json:"message,omitempty"`
}

type TenantDeviceDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

const (
	TenantCIBAErrorAuthorizationPending = "authorization_pending" // User hasn't responded yet
	TenantCIBAErrorAccessDenied         = "access_denied"         // User denied
	TenantCIBAErrorExpiredToken         = "expired_token"         // Request expired
	TenantCIBAErrorUserNotFound         = "user_not_found"        // Email not found
	TenantCIBAErrorNoDevice             = "no_device_registered"  // User has no push device
	TenantCIBAErrorTenantNotFound       = "tenant_not_found"      // Tenant not found
	TenantCIBAErrorInvalidClient        = "invalid_client"        // Client ID invalid
)
