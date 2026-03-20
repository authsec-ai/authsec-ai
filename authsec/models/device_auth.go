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
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID   uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ClientID   *uuid.UUID `json:"client_id,omitempty" gorm:"type:uuid;index"`

	// Device code: Long secret code for device polling
	DeviceCode string `json:"device_code" gorm:"uniqueIndex;size:128;not null"`

	// User code: Short human-readable code shown to user
	UserCode string `json:"user_code" gorm:"uniqueIndex;size:16;not null"`

	// Verification URIs
	VerificationURI         string `json:"verification_uri" gorm:"not null"`
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"`

	// User information (populated after authorization)
	UserID    *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index"`
	UserEmail string     `json:"user_email,omitempty"`

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

// DeviceCodeRequest represents a request to initiate device authorization
type DeviceCodeRequest struct {
	ClientID     string   `json:"client_id" binding:"required"`
	TenantDomain string   `json:"tenant_domain" binding:"required"`
	Scopes       []string `json:"scopes,omitempty"` // Optional OAuth scopes
	DeviceInfo   map[string]interface{} `json:"device_info,omitempty"` // Optional device metadata
}

// DeviceCodeResponse represents the response for device code request
// This follows RFC 8628 specification
type DeviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`                // Secret code for device to poll
	UserCode                string `json:"user_code"`                  // Short code for user to enter
	VerificationURI         string `json:"verification_uri"`           // URL where user activates device
	VerificationURIComplete string `json:"verification_uri_complete,omitempty"` // Optional pre-filled URL
	ExpiresIn               int    `json:"expires_in"`                 // Seconds until expiration
	Interval                int    `json:"interval"`                   // Minimum seconds between polling attempts
}

// DeviceTokenRequest represents a request to poll for device token
type DeviceTokenRequest struct {
	DeviceCode string `json:"device_code" binding:"required"`
	ClientID   string `json:"client_id" binding:"required"`
}

// DeviceTokenResponse represents the response for device token polling
type DeviceTokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope,omitempty"`
	Error        string `json:"error,omitempty"`        // RFC 8628 error codes
	ErrorDescription string `json:"error_description,omitempty"`
}

// RFC 8628 Error Codes
const (
	ErrorAuthorizationPending = "authorization_pending" // User hasn't authorized yet
	ErrorSlowDown             = "slow_down"             // Device is polling too frequently
	ErrorAccessDenied         = "access_denied"         // User denied authorization
	ErrorExpiredToken         = "expired_token"         // Device code expired
)

// DeviceVerificationRequest represents a request to verify/authorize a device
type DeviceVerificationRequest struct {
	UserCode string `json:"user_code" binding:"required"`
	Approve  bool   `json:"approve" binding:"required"`
}

// DeviceVerificationResponse represents the response for device verification
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
