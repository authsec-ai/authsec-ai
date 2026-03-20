package models

import (
	"github.com/google/uuid"
)

// TOTPSecret represents a TOTP authenticator device
type TOTPSecret struct {
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

// TableName specifies the table name for TOTPSecret
func (TOTPSecret) TableName() string {
	return "totp_secrets"
}

// BackupCode represents a recovery backup code for TOTP
type BackupCode struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	UserID    uuid.UUID `json:"user_id" gorm:"type:uuid;not null;index"`
	TenantID  uuid.UUID `json:"tenant_id" gorm:"type:uuid;not null;index"`
	Code      string    `json:"code" gorm:"size:32;not null;uniqueIndex"` // Hashed code
	IsUsed    bool      `json:"is_used" gorm:"default:false;index"`
	CreatedAt int64     `json:"created_at" gorm:"not null"`
	UsedAt    *int64    `json:"used_at,omitempty"`
}

// TableName specifies the table name for BackupCode
func (BackupCode) TableName() string {
	return "totp_backup_codes"
}

// ========================================
// DTOs for TOTP Authentication
// ========================================

// TOTPRegistrationRequest represents a request to register a new TOTP device
type TOTPRegistrationRequest struct {
	DeviceName string `json:"device_name" binding:"required"` // e.g., "My iPhone", "Personal iPad"
	DeviceType string `json:"device_type"`                    // optional: generic, google_auth, microsoft_auth, authy
}

// TOTPRegistrationResponse contains the QR code data for device registration
type TOTPRegistrationResponse struct {
	Success     bool     `json:"success"`
	Secret      string   `json:"secret"`       // TOTP secret (shown only once)
	QRCodeURL   string   `json:"qr_code_url"`  // URL for QR code generation
	DeviceID    string   `json:"device_id"`    // Device ID for confirmation
	BackupCodes []string `json:"backup_codes"` // Recovery codes (shown only once)
	Message     string   `json:"message"`
}

// TOTPRegistrationConfirmRequest confirms device registration after QR code scan
type TOTPRegistrationConfirmRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	TOTPCode string `json:"totp_code" binding:"required,len=6"` // 6-digit code to verify setup
}

// TOTPRegistrationConfirmResponse confirms successful registration
type TOTPRegistrationConfirmResponse struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	DeviceID   string `json:"device_id"`
	DeviceName string `json:"device_name"`
}

// TOTPVerificationRequest represents a request to verify TOTP during login
type TOTPVerificationRequest struct {
	TOTPCode string `json:"totp_code" binding:"required,len=6"` // 6-digit TOTP code
}

// TOTPVerificationResponse contains verification result
type TOTPVerificationResponse struct {
	Success    bool   `json:"success"`
	Token      string `json:"token,omitempty"` // JWT token if verification successful
	Message    string `json:"message,omitempty"`
	RequireOTP bool   `json:"require_otp"` // Whether TOTP is required for this user
}

// TOTPLoginRequest represents a TOTP-only login (no password)
type TOTPLoginRequest struct {
	Email    string `json:"email" binding:"required,email"`     // User email
	TOTPCode string `json:"totp_code" binding:"required,len=6"` // 6-digit TOTP code
}

// TOTPLoginResponse contains TOTP-only login result
type TOTPLoginResponse struct {
	Success bool   `json:"success"`
	Token   string `json:"token,omitempty"` // JWT token if login successful
	Message string `json:"message,omitempty"`
}

// TOTPDeviceListResponse lists user's registered TOTP devices
type TOTPDeviceListResponse struct {
	Success bool         `json:"success"`
	Devices []TOTPSecret `json:"devices"`
	Message string       `json:"message,omitempty"`
}

// TOTPDeviceDeleteRequest deletes a registered device
type TOTPDeviceDeleteRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// TOTPDeviceDeleteResponse confirms deletion
type TOTPDeviceDeleteResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// TOTPDeviceApprovalRequest approves device code using TOTP
type TOTPDeviceApprovalRequest struct {
	UserCode string `json:"user_code" binding:"required"`       // Device user code to approve
	Email    string `json:"email" binding:"required,email"`     // User email (to find user and validate TOTP)
	TOTPCode string `json:"totp_code" binding:"required,len=6"` // 6-digit TOTP code from authenticator app
}

// TOTPDeviceApprovalResponse contains approval result
type TOTPDeviceApprovalResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Token   string `json:"token,omitempty"` // JWT token if device approved
}

// BackupCodeRegenerateRequest regenerates backup codes
type BackupCodeRegenerateRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// BackupCodeRegenerateResponse contains new backup codes
type BackupCodeRegenerateResponse struct {
	Success     bool     `json:"success"`
	BackupCodes []string `json:"backup_codes"`
	Message     string   `json:"message"`
}
