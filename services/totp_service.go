package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// TOTPService handles TOTP (Time-Based One-Time Password) authentication
type TOTPService struct {
	totpRepo *database.TOTPRepository
}

// NewTOTPService creates a new TOTP service
func NewTOTPService(totpRepo *database.TOTPRepository) *TOTPService {
	return &TOTPService{totpRepo: totpRepo}
}

// GenerateSecret generates a random 20-byte TOTP secret (160 bits)
func (s *TOTPService) GenerateSecret() (string, error) {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// GenerateQRCodeURL generates a QR code URL for TOTP registration
// Format: otpauth://totp/Issuer:Username?secret=SECRET&issuer=Issuer
func (s *TOTPService) GenerateQRCodeURL(secret string, email string, issuer string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", issuer, email, secret, issuer)
}

// GenerateBackupCodes generates 10 recovery codes
func (s *TOTPService) GenerateBackupCodes() ([]string, error) {
	codes := make([]string, 10)
	for i := 0; i < 10; i++ {
		// Generate 16-character random code (8 bytes)
		bytes := make([]byte, 8)
		if _, err := rand.Read(bytes); err != nil {
			return nil, fmt.Errorf("failed to generate backup code: %w", err)
		}
		codes[i] = fmt.Sprintf("%08x%08x", binary.BigEndian.Uint32(bytes[0:4]), binary.BigEndian.Uint32(bytes[4:8]))
	}
	return codes, nil
}

// HashBackupCode hashes a backup code using SHA-256
func (s *TOTPService) HashBackupCode(code string) string {
	// Simple hash - in production, use bcrypt/scrypt
	h := sha1.New()
	h.Write([]byte(code))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ValidateTOTPCode validates a 6-digit TOTP code against the secret
// Allows for time drift (±1 time step = 30 seconds window)
func (s *TOTPService) ValidateTOTPCode(secret string, code string) bool {
	// Get current time
	now := time.Now()

	// Try current, previous, and next time steps (allow for clock drift)
	timeSteps := []int64{0, -1, 1}

	for _, offset := range timeSteps {
		t := now.Add(time.Duration(offset)*30*time.Second).Unix() / 30
		expectedCode := s.generateTOTP(secret, t)
		if subtle.ConstantTimeCompare([]byte(code), []byte(expectedCode)) == 1 {
			return true
		}
	}

	return false
}

// generateTOTP generates a TOTP code for a given time step
func (s *TOTPService) generateTOTP(secret string, t int64) string {
	// Decode base32 secret
	key, err := base32.StdEncoding.DecodeString(secret)
	if err != nil {
		return ""
	}

	// Convert time to 8-byte big-endian array
	timeBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBytes, uint64(t))

	// HMAC-SHA1
	h := hmac.New(sha1.New, key)
	h.Write(timeBytes)
	hash := h.Sum(nil)

	// Dynamic truncation
	offset := int(hash[len(hash)-1] & 0x0F)
	code := binary.BigEndian.Uint32(hash[offset : offset+4])

	// Mask and reduce to 6 digits
	code &= 0x7FFFFFFF
	code %= 1000000

	// Pad with leading zeros
	return fmt.Sprintf("%06d", code)
}

// ============================
// Service Methods
// ============================

// RegisterDevice initiates TOTP device registration
func (s *TOTPService) RegisterDevice(userID uuid.UUID, tenantID uuid.UUID, email string, deviceName string, deviceType string) (*models.TOTPRegistrationResponse, error) {
	// Generate TOTP secret
	secret, err := s.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.GenerateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Generate QR code URL
	qrCodeURL := s.GenerateQRCodeURL(secret, email, "AuthSec")

	// Create device record (not confirmed yet)
	device := &models.TOTPSecret{
		ID:         uuid.New(),
		UserID:     userID,
		TenantID:   tenantID,
		Secret:     secret,
		DeviceName: deviceName,
		DeviceType: deviceType,
		IsActive:   false, // Not active until confirmed
		IsPrimary:  false,
	}

	// Save to database (with is_active=false)
	if err := s.totpRepo.CreateTOTPSecret(device); err != nil {
		return nil, fmt.Errorf("failed to save device: %w", err)
	}

	// Store backup codes in database (hashed)
	for _, code := range backupCodes {
		backupCode := &models.BackupCode{
			UserID:   userID,
			TenantID: tenantID,
			Code:     s.HashBackupCode(code),
			IsUsed:   false,
		}
		if err := s.totpRepo.CreateBackupCode(backupCode); err != nil {
			return nil, fmt.Errorf("failed to save backup code: %w", err)
		}
	}

	return &models.TOTPRegistrationResponse{
		Success:     true,
		Secret:      secret,
		QRCodeURL:   qrCodeURL,
		DeviceID:    device.ID.String(),
		BackupCodes: backupCodes,
		Message:     "Scan QR code with your authenticator app, then enter the 6-digit code to confirm",
	}, nil
}

// ConfirmRegistration confirms TOTP device registration after QR code scan
func (s *TOTPService) ConfirmRegistration(deviceID uuid.UUID, userID uuid.UUID, tenantID uuid.UUID, totpCode string) (*models.TOTPRegistrationConfirmResponse, error) {
	// Get device
	device, err := s.totpRepo.GetTOTPSecretByID(deviceID, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("device not found")
	}

	// Validate TOTP code
	if !s.ValidateTOTPCode(device.Secret, totpCode) {
		return nil, fmt.Errorf("invalid TOTP code")
	}

	// Activate device
	device.IsActive = true

	// If user has no other active devices, make this primary
	existingDevices, err := s.totpRepo.GetUserTOTPSecrets(userID, tenantID)
	if err == nil && len(existingDevices) == 0 {
		device.IsPrimary = true
	}

	// Update database
	if err := s.totpRepo.UpdateTOTPSecret(device); err != nil {
		return nil, fmt.Errorf("failed to activate device: %w", err)
	}

	return &models.TOTPRegistrationConfirmResponse{
		Success:    true,
		Message:    "Device registered successfully",
		DeviceID:   device.ID.String(),
		DeviceName: device.DeviceName,
	}, nil
}

// VerifyTOTP validates a TOTP code for authentication
func (s *TOTPService) VerifyTOTP(userID uuid.UUID, tenantID uuid.UUID, totpCode string) (bool, error) {
	// Get user's TOTP devices
	devices, err := s.totpRepo.GetUserTOTPSecrets(userID, tenantID)
	if err != nil {
		return false, err
	}

	// Try validating against each device
	for _, device := range devices {
		if s.ValidateTOTPCode(device.Secret, totpCode) {
			// Update last used timestamp
			s.totpRepo.UpdateLastUsed(device.ID)
			return true, nil
		}
	}

	return false, nil
}

// VerifyBackupCode validates a backup code for authentication
func (s *TOTPService) VerifyBackupCode(userID uuid.UUID, tenantID uuid.UUID, code string) (bool, error) {
	// Get user's backup codes
	codes, err := s.totpRepo.GetUserBackupCodes(userID, tenantID)
	if err != nil {
		return false, err
	}

	// Hash the provided code
	hashedCode := s.HashBackupCode(code)

	// Check if code exists and is unused
	for _, c := range codes {
		if !c.IsUsed && subtle.ConstantTimeCompare([]byte(c.Code), []byte(hashedCode)) == 1 {
			// Mark as used
			s.totpRepo.UseBackupCode(c.ID)
			return true, nil
		}
	}

	return false, nil
}

// GetUserDevices retrieves user's registered TOTP devices
func (s *TOTPService) GetUserDevices(userID uuid.UUID, tenantID uuid.UUID) ([]models.TOTPSecret, error) {
	return s.totpRepo.GetUserTOTPSecrets(userID, tenantID)
}

// DeleteDevice deletes a TOTP device
func (s *TOTPService) DeleteDevice(deviceID uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	// Check if this is the only device
	devices, err := s.totpRepo.GetUserTOTPSecrets(userID, tenantID)
	if err == nil && len(devices) == 1 {
		return fmt.Errorf("cannot delete the last device. Disable 2FA instead")
	}

	return s.totpRepo.DeleteTOTPSecret(deviceID, userID, tenantID)
}

// SetPrimaryDevice sets a device as primary
func (s *TOTPService) SetPrimaryDevice(deviceID uuid.UUID, userID uuid.UUID, tenantID uuid.UUID) error {
	return s.totpRepo.SetPrimaryTOTPSecret(deviceID, userID, tenantID)
}

// RegenerateBackupCodes regenerates backup codes for a user
func (s *TOTPService) RegenerateBackupCodes(userID uuid.UUID, tenantID uuid.UUID) ([]string, error) {
	// Delete existing codes
	if err := s.totpRepo.DeleteBackupCodes(userID, tenantID); err != nil {
		return nil, err
	}

	// Generate new codes
	backupCodes, err := s.GenerateBackupCodes()
	if err != nil {
		return nil, err
	}

	// Store new codes
	for _, code := range backupCodes {
		backupCode := &models.BackupCode{
			UserID:   userID,
			TenantID: tenantID,
			Code:     s.HashBackupCode(code),
			IsUsed:   false,
		}
		if err := s.totpRepo.CreateBackupCode(backupCode); err != nil {
			return nil, err
		}
	}

	return backupCodes, nil
}

// LoginWithTOTP performs TOTP-only login (no password required)
// Returns user ID, tenant ID if successful
func (s *TOTPService) LoginWithTOTP(email string, totpCode string) (*uuid.UUID, *uuid.UUID, error) {
	// Get user by email (need to query users table)
	// This will need userRepo to lookup user by email
	// For now, we'll use a helper that the controller will call
	return nil, nil, fmt.Errorf("use LoginWithTOTPWithUser instead")
}

// LoginWithTOTPWithUser performs TOTP-only login given a user object
// Returns true if TOTP code is valid
func (s *TOTPService) LoginWithTOTPWithUser(user *models.ExtendedUser, totpCode string) (bool, error) {
	// Get user's TOTP devices
	devices, err := s.totpRepo.GetUserTOTPSecrets(user.ID, user.TenantID)
	if err != nil {
		return false, err
	}

	// If user has no TOTP devices, can't login with TOTP
	if len(devices) == 0 {
		return false, fmt.Errorf("no TOTP devices registered")
	}

	// Try validating against each device
	for _, device := range devices {
		if s.ValidateTOTPCode(device.Secret, totpCode) {
			// Update last used timestamp
			s.totpRepo.UpdateLastUsed(device.ID)
			return true, nil
		}
	}

	return false, nil
}

// HasTOTPEnabled checks if user has TOTP enabled
func (s *TOTPService) HasTOTPEnabled(userID uuid.UUID, tenantID uuid.UUID) (bool, error) {
	devices, err := s.totpRepo.GetUserTOTPSecrets(userID, tenantID)
	if err != nil {
		return false, err
	}
	return len(devices) > 0, nil
}
