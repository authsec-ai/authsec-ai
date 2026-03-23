package models

import (
	"time"

	"github.com/google/uuid"
)

// VoiceSession represents a voice authentication session
// Used for voice assistant authentication (Alexa, Google Assistant, Siri, etc.)
// All timestamps are stored as Unix epoch (seconds)
type VoiceSession struct {
	ID            uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID      uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ClientID      *uuid.UUID `json:"client_id,omitempty" gorm:"type:uuid;index"`

	// Session identifier
	SessionToken string `json:"session_token" gorm:"uniqueIndex;size:128;not null"`

	// Voice OTP (spoken code)
	VoiceOTP    string `json:"voice_otp" gorm:"size:10;not null"`
	OTPAttempts int    `json:"otp_attempts" gorm:"default:0"`

	// Voice assistant information
	VoicePlatform string  `json:"voice_platform,omitempty" gorm:"size:50"` // 'alexa', 'google', 'siri', 'custom'
	VoiceUserID   string  `json:"voice_user_id,omitempty"`                 // Platform-specific user ID
	DeviceInfo    JSONMap `json:"device_info,omitempty" gorm:"type:jsonb"`

	// User information (populated after verification)
	UserID    *uuid.UUID `json:"user_id,omitempty" gorm:"type:uuid;index"`
	UserEmail string     `json:"user_email,omitempty"`

	// Session state
	Status string `json:"status" gorm:"size:20;default:'initiated';not null;index"` // initiated, verified, expired, failed

	// Link to device authorization flow (if voice initiates device flow)
	LinkedDeviceCode string `json:"linked_device_code,omitempty" gorm:"size:128"`

	// OAuth scopes (if requesting token directly)
	Scopes JSONStringArray `json:"scopes" gorm:"type:jsonb;default:'[]'"`

	// Timing - Unix epoch timestamps (seconds)
	ExpiresAt  int64  `json:"expires_at" gorm:"not null;index"`
	VerifiedAt *int64 `json:"verified_at,omitempty"`
	CreatedAt  int64  `json:"created_at"`
	UpdatedAt  int64  `json:"updated_at"`
}

// TableName specifies the table name for VoiceSession
func (VoiceSession) TableName() string {
	return "voice_sessions"
}

// IsExpired checks if the voice session has expired
func (v *VoiceSession) IsExpired() bool {
	return time.Now().Unix() > v.ExpiresAt
}

// CanRetryOTP checks if more OTP attempts are allowed
func (v *VoiceSession) CanRetryOTP() bool {
	return v.OTPAttempts < 5
}

// VoiceIdentityLink represents a permanent link between voice assistant account and user account
// Enables passwordless authentication via voice assistant
// All timestamps are stored as Unix epoch (seconds)
type VoiceIdentityLink struct {
	ID       uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`

	// Voice assistant identity
	VoicePlatform string `json:"voice_platform" gorm:"size:50;not null;index"` // 'alexa', 'google', 'siri'
	VoiceUserID   string `json:"voice_user_id" gorm:"not null;index"`          // Platform-specific user ID
	VoiceUserName string `json:"voice_user_name,omitempty"`                    // Optional display name from platform

	// Linked user
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	UserEmail string    `json:"user_email" gorm:"not null"`

	// Link metadata
	IsActive   bool   `json:"is_active" gorm:"default:true;index"`
	LinkMethod string `json:"link_method,omitempty" gorm:"size:50"` // 'browser_verification', 'voice_otp', 'admin_linked'

	// Security - Unix epoch timestamps (seconds)
	LastUsedAt *int64 `json:"last_used_at,omitempty"`
	LinkedAt   int64  `json:"linked_at"`

	// Standard timestamps - Unix epoch (seconds)
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// TableName specifies the table name for VoiceIdentityLink
func (VoiceIdentityLink) TableName() string {
	return "voice_identity_links"
}

// ========================================
// DTOs for Voice Authentication Flow
// ========================================

// VoiceInitiateRequest represents a request to initiate voice authentication
type VoiceInitiateRequest struct {
	ClientID      string                 `json:"client_id" binding:"required"` // Used to lookup tenant via tenant_mappings
	VoicePlatform string                 `json:"voice_platform,omitempty"`     // 'alexa', 'google', 'siri', 'web'
	VoiceUserID   string                 `json:"voice_user_id,omitempty"`      // Platform-specific user ID
	DeviceInfo    map[string]interface{} `json:"device_info,omitempty"`        // Optional device metadata
	Scopes        []string               `json:"scopes,omitempty"`             // Optional OAuth scopes
}

// VoiceInitiateResponse represents the response for voice initiation
type VoiceInitiateResponse struct {
	SessionToken string `json:"session_token"`        // Secret token for this session
	VoiceOTP     string `json:"voice_otp"`            // Numeric code to speak (e.g., "8532")
	ExpiresIn    int    `json:"expires_in"`           // Seconds until expiration
	Message      string `json:"message,omitempty"`    // Human-readable message for voice assistant
}

// VoiceVerifyRequest represents a request to verify voice OTP
type VoiceVerifyRequest struct {
	SessionToken       string `json:"session_token" binding:"required"`
	VoiceOTP           string `json:"voice_otp" binding:"required"`
	VoiceConfirmation  bool   `json:"voice_confirmation"` // User confirmed via voice
}

// VoiceVerifyResponse represents the response for voice verification
type VoiceVerifyResponse struct {
	Success          bool   `json:"success"`
	Status           string `json:"status"` // 'verified', 'failed', 'expired'
	Message          string `json:"message,omitempty"`

	// If linking to device flow
	DeviceCode       string `json:"device_code,omitempty"`
	UserCode         string `json:"user_code,omitempty"`
	VerificationURI  string `json:"verification_uri,omitempty"`

	// If direct token issuance (pre-linked voice identity)
	AccessToken      string `json:"access_token,omitempty"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	TokenType        string `json:"token_type,omitempty"`
	ExpiresIn        int    `json:"expires_in,omitempty"`
}

// VoiceTokenRequest represents a request for token using voice credentials
// WARNING: Less secure - user speaks credentials
type VoiceTokenRequest struct {
	SessionToken string `json:"session_token" binding:"required"`
	Email        string `json:"email" binding:"required"`
	Password     string `json:"password" binding:"required"`
	TenantDomain string `json:"tenant_domain" binding:"required"`
}

// VoiceTokenResponse represents the response for voice token request
type VoiceTokenResponse struct {
	AccessToken  string `json:"access_token,omitempty"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type,omitempty"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
	Error        string `json:"error,omitempty"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// ========================================
// Voice Identity Link DTOs
// ========================================

// VoiceLinkRequest represents a request to link voice assistant to user account
type VoiceLinkRequest struct {
	VoicePlatform string `json:"voice_platform" binding:"required"` // 'alexa', 'google', 'siri'
	VoiceUserID   string `json:"voice_user_id" binding:"required"`  // Platform-specific user ID
	VoiceUserName string `json:"voice_user_name,omitempty"`         // Optional display name
	LinkMethod    string `json:"link_method,omitempty"`             // How link was established
}

// VoiceLinkResponse represents the response for voice link request
type VoiceLinkResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	LinkID  string `json:"link_id,omitempty"`
}

// VoiceUnlinkRequest represents a request to unlink voice assistant
type VoiceUnlinkRequest struct {
	VoicePlatform string `json:"voice_platform" binding:"required"`
	VoiceUserID   string `json:"voice_user_id,omitempty"` // Optional - if empty, unlinks all for platform
}

// VoiceUnlinkResponse represents the response for voice unlink request
type VoiceUnlinkResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VoiceLinksListResponse represents list of linked voice assistants for a user
type VoiceLinksListResponse struct {
	Links []VoiceIdentityLinkPublic `json:"links"`
}

// VoiceIdentityLinkPublic represents public info about a voice link (for user UI)
type VoiceIdentityLinkPublic struct {
	ID            string `json:"id"`
	VoicePlatform string `json:"voice_platform"`
	VoiceUserName string `json:"voice_user_name,omitempty"`
	IsActive      bool   `json:"is_active"`
	LastUsedAt    *int64 `json:"last_used_at,omitempty"`
	LinkedAt      int64  `json:"linked_at"`
}

// ========================================
// Voice Platform Constants
// ========================================

const (
	VoicePlatformAlexa  = "alexa"
	VoicePlatformGoogle = "google"
	VoicePlatformSiri   = "siri"
	VoicePlatformCustom = "custom"
)

// IsValidVoicePlatform checks if the voice platform is supported
func IsValidVoicePlatform(platform string) bool {
	switch platform {
	case VoicePlatformAlexa, VoicePlatformGoogle, VoicePlatformSiri, VoicePlatformCustom:
		return true
	default:
		return false
	}
}

// ========================================
// Voice Active Session (Device Tracking)
// ========================================

// VoiceActiveSession represents an active JWT session from voice authentication
// Used for device tracking, session management, and remote logout
// All timestamps are stored as Unix epoch (seconds)
type VoiceActiveSession struct {
	ID       uuid.UUID  `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	TenantID uuid.UUID  `json:"tenant_id" gorm:"type:uuid;not null;index"`
	ClientID *uuid.UUID `json:"client_id,omitempty" gorm:"type:uuid"`

	// User identity
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	UserEmail string    `json:"user_email" gorm:"not null"`

	// Session identification
	SessionID string `json:"session_id" gorm:"uniqueIndex;size:128;not null"` // Used as jti claim

	// Device/Platform information
	VoicePlatform string  `json:"voice_platform,omitempty" gorm:"size:50"`
	VoiceUserID   string  `json:"voice_user_id,omitempty"`
	DeviceInfo    JSONMap `json:"device_info,omitempty" gorm:"type:jsonb;default:'{}'"`
	DeviceName    string  `json:"device_name,omitempty"` // User-friendly name

	// Token information
	AccessTokenHash  string `json:"-" gorm:"size:64"` // SHA256 hash for revocation
	RefreshTokenHash string `json:"-" gorm:"size:64"`

	// Timestamps - Unix epoch (seconds)
	LoginAt        int64 `json:"login_at"`
	LastActivityAt int64 `json:"last_activity_at"`
	ExpiresAt      int64 `json:"expires_at" gorm:"not null;index"`

	// Session state
	IsActive      bool   `json:"is_active" gorm:"default:true;index"`
	RevokedAt     *int64 `json:"revoked_at,omitempty"`
	RevokedReason string `json:"revoked_reason,omitempty" gorm:"size:100"`

	// Standard timestamps - Unix epoch (seconds)
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
}

// TableName specifies the table name for VoiceActiveSession
func (VoiceActiveSession) TableName() string {
	return "voice_active_sessions"
}

// IsExpired checks if the session has expired
func (v *VoiceActiveSession) IsExpired() bool {
	return time.Now().Unix() > v.ExpiresAt
}

// ========================================
// Voice Active Session DTOs
// ========================================

// VoiceActiveSessionPublic represents public info about an active session (for UI)
type VoiceActiveSessionPublic struct {
	ID             string  `json:"id"`
	SessionID      string  `json:"session_id"`
	VoicePlatform  string  `json:"voice_platform,omitempty"`
	DeviceName     string  `json:"device_name,omitempty"`
	DeviceInfo     JSONMap `json:"device_info,omitempty"`
	LoginAt        int64   `json:"login_at"`
	LastActivityAt int64   `json:"last_activity_at"`
	ExpiresAt      int64   `json:"expires_at"`
	IsActive       bool    `json:"is_active"`
	IsCurrent      bool    `json:"is_current"` // True if this is the current session
}

// VoiceActiveSessionsListResponse represents the list of active sessions
type VoiceActiveSessionsListResponse struct {
	Sessions     []VoiceActiveSessionPublic `json:"sessions"`
	TotalCount   int                        `json:"total_count"`
	ActiveCount  int                        `json:"active_count"`
}

// VoiceSessionLogoutRequest represents a request to logout a specific session
type VoiceSessionLogoutRequest struct {
	SessionID string `json:"session_id" binding:"required"` // The session to logout
}

// VoiceSessionLogoutResponse represents the response for session logout
type VoiceSessionLogoutResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// VoiceSessionLogoutAllResponse represents the response for logout all sessions
type VoiceSessionLogoutAllResponse struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	LoggedOutCount int   `json:"logged_out_count"`
}

// ========================================
// Pending Voice Auth Request DTOs
// ========================================

// VoicePendingRequest represents a pending voice auth request awaiting approval
type VoicePendingRequest struct {
	ID            string  `json:"id"`
	SessionToken  string  `json:"session_token"`
	VoicePlatform string  `json:"voice_platform,omitempty"`
	VoiceUserID   string  `json:"voice_user_id,omitempty"`
	DeviceInfo    JSONMap `json:"device_info,omitempty"`
	UserCode      string  `json:"user_code,omitempty"` // The code user needs to enter
	ExpiresAt     int64   `json:"expires_at"`
	CreatedAt     int64   `json:"created_at"`
	Status        string  `json:"status"` // 'pending', 'approved', 'denied', 'expired'
}

// VoicePendingRequestsResponse represents the list of pending voice auth requests
type VoicePendingRequestsResponse struct {
	Requests []VoicePendingRequest `json:"requests"`
	Count    int                   `json:"count"`
}

// VoiceCheckPendingRequest represents request to check for pending voice auths
type VoiceCheckPendingRequest struct {
	ClientID string `json:"client_id" binding:"required"` // Client ID to check for pending requests
}

// VoiceApproveRequest represents a request to approve/deny a voice auth
type VoiceApproveRequest struct {
	SessionToken string `json:"session_token" binding:"required"` // Session to approve
	UserCode     string `json:"user_code" binding:"required"`     // Code from voice assistant
	Approve      bool   `json:"approve"`                          // true = approve, false = deny
}

// VoiceApproveResponse represents the response for approve/deny
type VoiceApproveResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Status  string `json:"status"` // 'approved', 'denied', 'expired', 'invalid'
}

// ========================================
// WebSocket Message Types
// ========================================

// VoiceWSMessage represents a WebSocket message for real-time notifications
type VoiceWSMessage struct {
	Type      string      `json:"type"`      // 'pending_request', 'approved', 'denied', 'expired'
	Payload   interface{} `json:"payload"`   // The actual data
	Timestamp int64       `json:"timestamp"` // Unix epoch when the message was sent
}

// VoiceWSPendingPayload is the payload for a pending request notification
type VoiceWSPendingPayload struct {
	SessionToken  string  `json:"session_token"`
	VoicePlatform string  `json:"voice_platform,omitempty"`
	DeviceInfo    JSONMap `json:"device_info,omitempty"`
	UserCode      string  `json:"user_code"`
	ExpiresIn     int     `json:"expires_in"` // Seconds until expiration
}

// VoiceWSApprovalPayload is the payload for approval status updates
type VoiceWSApprovalPayload struct {
	SessionToken string `json:"session_token"`
	Status       string `json:"status"` // 'approved', 'denied'
	Message      string `json:"message,omitempty"`
}
