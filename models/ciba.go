package models

import (
	"time"

	"github.com/google/uuid"
)

// ========================================
// Device Token for Push Notifications
// ========================================

// DeviceToken represents a registered device for push notifications
type DeviceToken struct {
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

// TableName specifies the table name for DeviceToken
func (DeviceToken) TableName() string {
	return "device_tokens"
}

// ========================================
// CIBA Authentication Request
// ========================================

// CIBAAuthRequest represents a CIBA authentication request (push notification based)
type CIBAAuthRequest struct {
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

// TableName specifies the table name for CIBAAuthRequest
func (CIBAAuthRequest) TableName() string {
	return "ciba_auth_requests"
}

// IsExpired checks if the CIBA request has expired
func (c *CIBAAuthRequest) IsExpired() bool {
	return time.Now().Unix() > c.ExpiresAt
}

// IsPending checks if still waiting for user response
func (c *CIBAAuthRequest) IsPending() bool {
	return c.Status == "pending" && !c.IsExpired()
}

// IsApproved checks if user approved the request
func (c *CIBAAuthRequest) IsApproved() bool {
	return c.Status == "approved"
}

// ========================================
// DTOs for CIBA Flow
// ========================================

// DeviceTokenRegistrationRequest - Mobile app registers device for push notifications
type DeviceTokenRegistrationRequest struct {
	DeviceToken string `json:"device_token" binding:"required"` // Expo Push Token
	Platform    string `json:"platform" binding:"required"`     // ios, android
	DeviceName  string `json:"device_name,omitempty"`
	DeviceModel string `json:"device_model,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	OSVersion   string `json:"os_version,omitempty"`
}

// DeviceTokenRegistrationResponse
type DeviceTokenRegistrationResponse struct {
	Success  bool   `json:"success"`
	DeviceID string `json:"device_id,omitempty"`
	Message  string `json:"message"`
}

// CIBAInitiateRequest - Initiate CIBA authentication (voice agent calls this)
type CIBAInitiateRequest struct {
	LoginHint      string   `json:"login_hint" binding:"required"`      // User email
	BindingMessage string   `json:"binding_message,omitempty"`          // Message shown to user
	ClientID       string   `json:"client_id,omitempty"`                // Optional client ID
	Scopes         []string `json:"scopes,omitempty"`                   // OAuth scopes
}

// CIBAInitiateResponse
type CIBAInitiateResponse struct {
	AuthReqID string `json:"auth_req_id"`           // Request ID for polling
	ExpiresIn int    `json:"expires_in"`            // Seconds until expiration
	Interval  int    `json:"interval"`              // Polling interval in seconds
	Message   string `json:"message,omitempty"`     // Status message
	Error     string `json:"error,omitempty"`       // Error code
	ErrorDescription string `json:"error_description,omitempty"` // Error description
}

// CIBARespondRequest - User approves/denies via mobile app
type CIBARespondRequest struct {
	AuthReqID         string `json:"auth_req_id" binding:"required"`
	Approved          bool   `json:"approved"`
	BiometricVerified bool   `json:"biometric_verified,omitempty"`
}

// CIBARespondResponse
type CIBARespondResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CIBATokenRequest - Voice agent polls for token
type CIBATokenRequest struct {
	AuthReqID string `json:"auth_req_id" binding:"required"`
	ClientID  string `json:"client_id,omitempty"`
}

// CIBATokenResponse - Returns token or status
type CIBATokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// CIBA Error Codes (RFC 8628 + custom)
const (
	CIBAErrorAuthorizationPending = "authorization_pending" // User hasn't responded yet
	CIBAErrorAccessDenied         = "access_denied"         // User denied
	CIBAErrorExpiredToken         = "expired_token"         // Request expired
	CIBAErrorUserNotFound         = "user_not_found"        // Email not found
	CIBAErrorNoDevice             = "no_device_registered"  // User has no push device
)

// ========================================
// Device Management DTOs (Admin CIBA)
// ========================================

// DeviceSummary represents a device without sensitive token data
type DeviceSummary struct {
	ID          string `json:"id"`
	DeviceName  string `json:"device_name,omitempty"`
	Platform    string `json:"platform"`
	DeviceModel string `json:"device_model,omitempty"`
	AppVersion  string `json:"app_version,omitempty"`
	OSVersion   string `json:"os_version,omitempty"`
	IsActive    bool   `json:"is_active"`
	LastUsed    *int64 `json:"last_used,omitempty"`
	CreatedAt   int64  `json:"created_at"`
}

// DeviceListResponse response for listing devices
type DeviceListResponse struct {
	Success bool            `json:"success"`
	Devices []DeviceSummary `json:"devices"`
	Message string          `json:"message,omitempty"`
}

// DeviceDeleteRequest request to delete a device
type DeviceDeleteRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// DeviceDeleteResponse response after deleting a device
type DeviceDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
