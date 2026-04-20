package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// DeviceCode represents a device authorization grant request (RFC 8628)
// Used for authentication on devices without easy input (TVs, CLI tools, IoT devices)
// All timestamps are stored as Unix epoch (seconds)
type DeviceCode struct {
	ID       uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID *uuid.UUID `json:"tenant_id,omitempty" gorm:"type:uuid;index"` // nullable until /authorize step
	ClientID *uuid.UUID `json:"client_id,omitempty" gorm:"type:uuid;index"`

	// Device code: Long secret code for device polling
	DeviceCode string `json:"device_code" gorm:"uniqueIndex;size:128;not null"`

	// User code: Short human-readable code shown to user
	UserCode string `json:"user_code" gorm:"uniqueIndex;size:16;not null"`

	// Verification URIs
	VerificationURI         string `json:"verification_uri" gorm:"not null"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`

	// User information (populated after authorization)
	UserID       *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index"`
	UserEmail    string     `json:"user_email,omitempty"`
	TenantDomain string     `json:"tenant_domain,omitempty"` // cached at authorize time
	AccessToken  string     `json:"access_token,omitempty"`  // JWT stored at authorize time

	// Authorization state
	Status string `json:"status" gorm:"size:20;default:'pending';not null;index"` // pending, authorized, denied, expired, consumed

	// OAuth scopes
	Scopes JSONStringArray `json:"scopes" gorm:"type:jsonb;default:'[]'"`

	// Device information
	DeviceInfo JSONMap `json:"device_info,omitempty" gorm:"type:jsonb"`

	// Timing - Unix epoch timestamps (seconds)
	ExpiresAt    int64  `json:"expires_at" gorm:"not null;index"`
	LastPolledAt *int64 `json:"last_polled_at,omitempty"`
	AuthorizedAt *int64 `json:"authorized_at,omitempty"`
	CreatedAt    int64  `json:"created_at"`
	UpdatedAt    int64  `json:"updated_at"`
}

// TableName specifies the table name for DeviceCode
func (DeviceCode) TableName() string {
	return "device_codes"
}

// IsExpired checks if the device code has expired
func (d *DeviceCode) IsExpired() bool {
	return time.Now().Unix() > d.ExpiresAt
}

// IsPending checks if the device code is still waiting for authorization
func (d *DeviceCode) IsPending() bool {
	return d.Status == "pending" && !d.IsExpired()
}

// IsAuthorized checks if the device code has been authorized
func (d *DeviceCode) IsAuthorized() bool {
	return d.Status == "authorized"
}

// ========================================
// DTOs for Device Authorization Flow
// ========================================

// DeviceCodeRequest represents a request to initiate device authorization.
// client_id and tenant_domain are optional: the CLI (authsec-shield) sends neither.
// Tenant is resolved from the authenticated browser session during the /authorize step.
type DeviceCodeRequest struct {
	ClientID     string                 `json:"client_id,omitempty"`
	TenantDomain string                 `json:"tenant_domain,omitempty"`
	Scopes       []string               `json:"scopes,omitempty"`
	DeviceInfo   map[string]interface{} `json:"device_info,omitempty"`
}

// DeviceCodeResponse represents the response for device code request (RFC 8628)
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

// DeviceTokenRequest represents a request to poll for device token.
// grant_type must be "urn:ietf:params:oauth:grant-type:device_code".
// client_id is optional (not sent by authsec-shield CLI).
type DeviceTokenRequest struct {
	DeviceCode string `json:"device_code" binding:"required"`
	GrantType  string `json:"grant_type"`
	ClientID   string `json:"client_id,omitempty"`
}

// DeviceTokenResponse represents the response for device token polling.
// When authorized, all fields are populated. On error only Error/ErrorDescription are set.
type DeviceTokenResponse struct {
	// Success fields
	AccessToken string `json:"access_token,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
	ExpiresIn   int    `json:"expires_in,omitempty"`
	Scope       string `json:"scope,omitempty"`
	// Identity fields returned alongside token
	Email        string `json:"email,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`
	TenantDomain string `json:"tenant_domain,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	// RFC 8628 error fields
	Error            string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// RFC 8628 Error Codes
const (
	ErrorAuthorizationPending = "authorization_pending"
	ErrorSlowDown             = "slow_down"
	ErrorAccessDenied         = "access_denied"
	ErrorExpiredToken         = "expired_token"
)

// DeviceAuthorizeRequest is sent by the web app (app.authsec.ai/activate) after the
// authenticated user enters their user_code and approves or denies the device.
type DeviceAuthorizeRequest struct {
	UserCode string `json:"user_code" binding:"required"`
	Approved bool   `json:"approved"` // true = approve, false = deny
}

// DeviceAuthorizeOIDCRequest is used by the shield end-user login flow.
// After OIDC authentication, the callback page sends the user_code + OIDC code.
type DeviceAuthorizeOIDCRequest struct {
	UserCode string `json:"user_code" binding:"required"` // Code shown in shield terminal
	OIDCCode string `json:"oidc_code" binding:"required"` // Authorization code from OIDC callback
	State    string `json:"state" binding:"required"`      // OIDC state parameter
}

// DeviceAuthorizeResponse is returned by POST /device/authorize.
type DeviceAuthorizeResponse struct {
	Status string `json:"status"` // "authorized" or "denied"
}

// DeviceVerifyCodeRequest is sent by the public /verify endpoint to check a user_code.
type DeviceVerifyCodeRequest struct {
	UserCode string `json:"user_code" binding:"required"`
}

// DeviceVerifyCodeResponse is returned by /verify — used by the activation page before login.
type DeviceVerifyCodeResponse struct {
	Valid        bool   `json:"valid"`
	DeviceCodeID string `json:"device_code_id,omitempty"` // internal ID for the record
	ExpiresIn    int    `json:"expires_in,omitempty"`     // seconds until code expires
	Error        string `json:"error,omitempty"`          // "invalid_code" when not found/expired
}

// DeviceVerificationRequest is the legacy form (kept for backwards compat with /verify POST).
type DeviceVerificationRequest struct {
	UserCode string `json:"user_code" binding:"required"`
	Approve  bool   `json:"approve"`
}

// DeviceVerificationResponse is the legacy response (kept for backwards compat).
type DeviceVerificationResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// DeviceActivationInfoRequest represents a request to get device activation info
type DeviceActivationInfoRequest struct {
	UserCode string `json:"user_code" form:"user_code" binding:"required"`
}

// DeviceActivationInfoResponse represents device info shown on activation page
type DeviceActivationInfoResponse struct {
	Success      bool     `json:"success"`
	UserCode     string   `json:"user_code"`
	TenantDomain string   `json:"tenant_domain"`
	DeviceInfo   JSONMap  `json:"device_info,omitempty"`
	Scopes       []string `json:"scopes,omitempty"`
	ExpiresAt    string   `json:"expires_at"`
	Message      string   `json:"message,omitempty"`
}

// ========================================
// Helper Types for JSON
// ========================================

// JSONStringArray is a custom type for storing string arrays as JSON
type JSONStringArray []string

// Value implements driver.Valuer for database storage
func (j JSONStringArray) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal([]string{})
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner for database retrieval
func (j *JSONStringArray) Scan(value interface{}) error {
	if value == nil {
		*j = []string{}
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), j)
	}
	return json.Unmarshal(bytes, j)
}

// JSONMap is a custom type for storing maps as JSON
type JSONMap map[string]interface{}

// Value implements driver.Valuer for database storage
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return json.Marshal(map[string]interface{}{})
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner for database retrieval
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(map[string]interface{})
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return json.Unmarshal([]byte(value.(string)), j)
	}
	return json.Unmarshal(bytes, j)
}
