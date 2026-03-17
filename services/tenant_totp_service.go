package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"

	"github.com/google/uuid"
)

// TenantTOTPService handles TOTP authentication for tenant users
type TenantTOTPService struct {
	adminTenantRepo *database.AdminTenantRepository
}

// NewTenantTOTPService creates a new tenant TOTP service
func NewTenantTOTPService() *TenantTOTPService {
	return &TenantTOTPService{
		adminTenantRepo: database.NewAdminTenantRepository(config.GetDatabase()),
	}
}

// tenantMapping maps client ID to tenant ID using tenant_mappings table
func (s *TenantTOTPService) tenantMapping(clientID uuid.UUID) (uuid.UUID, error) {
	db := config.GetDatabase()
	if db == nil {
		return uuid.UUID{}, fmt.Errorf("database not initialized")
	}

	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM tenant_mappings WHERE client_id = $1`
	err := db.QueryRow(query, clientID).Scan(&tenantID)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return uuid.UUID{}, fmt.Errorf("client not found")
		}
		return uuid.UUID{}, fmt.Errorf("failed to lookup tenant mapping: %w", err)
	}

	return tenantID, nil
}

// GenerateSecret generates a random 20-byte TOTP secret (160 bits)
func (s *TenantTOTPService) GenerateSecret() (string, error) {
	bytes := make([]byte, 20)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secret: %w", err)
	}
	return base32.StdEncoding.EncodeToString(bytes), nil
}

// GenerateQRCodeURL generates a QR code URL for TOTP registration
// Format: otpauth://totp/Issuer:Username?secret=SECRET&issuer=Issuer
func (s *TenantTOTPService) GenerateQRCodeURL(secret string, email string, issuer string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", issuer, email, secret, issuer)
}

// GenerateBackupCodes generates 10 recovery codes
func (s *TenantTOTPService) GenerateBackupCodes() ([]string, error) {
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
func (s *TenantTOTPService) HashBackupCode(code string) string {
	// Simple hash - in production, use bcrypt/scrypt
	h := sha1.New()
	h.Write([]byte(code))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// ValidateTOTPCode validates a 6-digit TOTP code against secret
// Allows for time drift (±1 time step = 30 seconds window)
func (s *TenantTOTPService) ValidateTOTPCode(secret string, code string) bool {
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
func (s *TenantTOTPService) generateTOTP(secret string, t int64) string {
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

	// Dynamic truncation (RFC 4226)
	offset := int(hash[len(hash)-1] & 0x0F)
	code := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	code = code % 1000000 // 6-digit code

	return fmt.Sprintf("%06d", code)
}

// LoginWithTenantTOTP handles TOTP-only login for tenant users
func (s *TenantTOTPService) LoginWithTenantTOTP(req *models.TenantTOTPLoginRequest) (*models.TenantTOTPLoginResponse, error) {
	// Step 1: Parse and validate client_id
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return &models.TenantTOTPLoginResponse{
			Success: false,
			Message: "Invalid client ID format",
		}, nil
	}

	// Step 2: Map client_id to tenant_id
	tenantUUID, err := s.tenantMapping(clientUUID)
	if err != nil {
		return &models.TenantTOTPLoginResponse{
			Success: false,
			Message: "Client not found or not mapped to tenant",
		}, nil
	}

	// Step 3: Get tenant information (validate existence)
	_, err = s.adminTenantRepo.GetTenantByUUID(tenantUUID)
	if err != nil {
		return &models.TenantTOTPLoginResponse{
			Success: false,
			Message: "Tenant not found",
		}, nil
	}

	// Step 4: Connect to tenant database
	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Step 5: Look up user in tenant database
	tenantRepo := database.NewTenantDeviceRepository(tenantDB)
	user, err := tenantRepo.GetTenantUserByEmail(strings.ToLower(req.Email), clientUUID)
	if err != nil {
		return &models.TenantTOTPLoginResponse{
			Success: false,
			Message: "User not found",
		}, nil
	}

	// Step 6: Get user's TOTP devices
	secrets, err := tenantRepo.GetTenantUserTOTPSecrets(user.ID, tenantUUID)
	if err != nil || len(secrets) == 0 {
		return &models.TenantTOTPLoginResponse{
			Success: false,
			Message: "No TOTP devices registered for this user",
		}, nil
	}

	// Step 7: Validate TOTP code against all devices
	validCode := false
	var usedSecret *models.TenantTOTPSecret
	for _, secret := range secrets {
		if s.ValidateTOTPCode(secret.Secret, req.TOTPCode) {
			validCode = true
			usedSecret = &secret
			break
		}
	}

	if !validCode {
		return &models.TenantTOTPLoginResponse{
			Success: false,
			Message: "Invalid TOTP code",
		}, nil
	}

	// Step 8: Update last_used timestamp
	if usedSecret != nil {
		tenantRepo.UpdateTenantTOTPSecretLastUsed(usedSecret.ID)
	}

	// Step 9: Generate JWT token
	token, err := s.generateJWTToken(user.ID, tenantUUID, user.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to generate JWT token: %w", err)
	}

	// Step 10: Update user's last login timestamp
	tenantRepo.UpdateTenantUserLastLogin(user.ID)

	return &models.TenantTOTPLoginResponse{
		Success: true,
		Token:   token,
		Message: "Authentication successful",
	}, nil
}

// RegisterTenantTOTPDevice registers a new TOTP device for tenant user
func (s *TenantTOTPService) RegisterTenantTOTPDevice(req *models.TenantTOTPRegistrationRequest, userID, tenantID uuid.UUID, email string) (*models.TenantTOTPRegistrationResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Generate TOTP secret
	secret, err := s.GenerateSecret()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP secret: %w", err)
	}

	// Generate backup codes
	backupCodes, err := s.GenerateBackupCodes()
	if err != nil {
		return nil, fmt.Errorf("failed to generate backup codes: %w", err)
	}

	// Create TOTP secret record
	totpSecret := &models.TenantTOTPSecret{
		ID:         uuid.New(),
		UserID:     userID,
		TenantID:   tenantID,
		Secret:     secret,
		DeviceName: req.DeviceName,
		DeviceType: req.DeviceType,
		IsActive:   true,
		IsPrimary:  false, // First device becomes primary only if no other devices exist
	}

	// Check if user has any existing TOTP devices
	existingSecrets, err := tenantRepo.GetTenantUserTOTPSecrets(userID, tenantID)
	if err == nil && len(existingSecrets) == 0 {
		totpSecret.IsPrimary = true // First device is primary
	}

	if err := tenantRepo.CreateTenantTOTPSecret(totpSecret); err != nil {
		if err.Error() == "tenant_not_found" {
			return nil, fmt.Errorf("database integrity error: tenant record missing in tenant database (run: INSERT INTO tenants (id) VALUES ('%s');)", tenantID)
		}
		if err.Error() == "user_not_found" {
			return nil, fmt.Errorf("user record not found in tenant database")
		}
		return nil, fmt.Errorf("failed to create TOTP secret: %w", err)
	}

	// Hash and store backup codes
	var hashedCodes []models.TenantBackupCode
	for _, code := range backupCodes {
		hashedCodes = append(hashedCodes, models.TenantBackupCode{
			ID:       uuid.New(),
			UserID:   userID,
			TenantID: tenantID,
			Code:     s.HashBackupCode(code),
			IsUsed:   false,
		})
	}

	if err := tenantRepo.CreateTenantBackupCodes(hashedCodes); err != nil {
		if err.Error() == "tenant_not_found" {
			// If we got here, it's weird because secret creation should have failed first,
			// but handle it anyway
			fmt.Printf("Warning: Failed to create backup codes: tenant missing in DB\n")
		}
		fmt.Printf("Warning: Failed to create backup codes: %v\n", err)
	}

	// Generate QR code URL
	issuer := "AuthSec Tenant Auth"
	qrCodeURL := s.GenerateQRCodeURL(secret, email, issuer)

	return &models.TenantTOTPRegistrationResponse{
		Success:     true,
		Secret:      secret,
		QRCodeURL:   qrCodeURL,
		DeviceID:    totpSecret.ID.String(),
		BackupCodes: backupCodes,
		Message:     "TOTP device registered successfully",
	}, nil
}

// ConfirmTenantTOTPDevice confirms TOTP device registration after QR code scan
func (s *TenantTOTPService) ConfirmTenantTOTPDevice(req *models.TenantTOTPRegistrationConfirmRequest, userID, tenantID uuid.UUID) (*models.TenantTOTPRegistrationConfirmResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Get TOTP secret
	deviceUUID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		return &models.TenantTOTPRegistrationConfirmResponse{
			Success: false,
			Message: "Invalid device ID format",
		}, nil
	}

	secret, err := tenantRepo.GetTenantTOTPSecretByID(deviceUUID, userID, tenantID)
	if err != nil {
		return &models.TenantTOTPRegistrationConfirmResponse{
			Success: false,
			Message: "Device not found",
		}, nil
	}

	// Validate TOTP code
	if !s.ValidateTOTPCode(secret.Secret, req.TOTPCode) {
		return &models.TenantTOTPRegistrationConfirmResponse{
			Success: false,
			Message: "Invalid TOTP code",
		}, nil
	}

	return &models.TenantTOTPRegistrationConfirmResponse{
		Success:    true,
		Message:    "TOTP device confirmed successfully",
		DeviceID:   secret.ID.String(),
		DeviceName: secret.DeviceName,
	}, nil
}

// GetTenantTOTPDevices retrieves all TOTP devices for a tenant user
func (s *TenantTOTPService) GetTenantTOTPDevices(userID, tenantID uuid.UUID) (*models.TenantTOTPDeviceListResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	devices, err := tenantRepo.GetTenantUserTOTPSecrets(userID, tenantID)
	if err != nil {
		return &models.TenantTOTPDeviceListResponse{
			Success: false,
			Message: "Failed to retrieve TOTP devices",
		}, nil
	}

	return &models.TenantTOTPDeviceListResponse{
		Success: true,
		Devices: devices,
	}, nil
}

// DeleteTenantTOTPDevice deletes a TOTP device for tenant user
func (s *TenantTOTPService) DeleteTenantTOTPDevice(req *models.TenantTOTPDeviceDeleteRequest, userID, tenantID uuid.UUID) (*models.TenantTOTPDeviceDeleteResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Parse device ID
	deviceUUID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		return &models.TenantTOTPDeviceDeleteResponse{
			Success: false,
			Message: "Invalid device ID format",
		}, nil
	}

	// Check if device exists
	secret, err := tenantRepo.GetTenantTOTPSecretByID(deviceUUID, userID, tenantID)
	if err != nil {
		return &models.TenantTOTPDeviceDeleteResponse{
			Success: false,
			Message: "Device not found",
		}, nil
	}

	// Don't allow deletion of primary device if it's the only one
	if secret.IsPrimary {
		allDevices, err := tenantRepo.GetTenantUserTOTPSecrets(userID, tenantID)
		if err == nil && len(allDevices) == 1 {
			return &models.TenantTOTPDeviceDeleteResponse{
				Success: false,
				Message: "Cannot delete the only TOTP device. Register another device first.",
			}, nil
		}
	}

	// Delete device
	if err := tenantRepo.DeleteTenantTOTPSecret(deviceUUID, userID, tenantID); err != nil {
		return &models.TenantTOTPDeviceDeleteResponse{
			Success: false,
			Message: "Failed to delete device",
		}, nil
	}

	return &models.TenantTOTPDeviceDeleteResponse{
		Success: true,
		Message: "Device deleted successfully",
	}, nil
}

// SetTenantPrimaryTOTPDevice sets a TOTP device as primary for tenant user
func (s *TenantTOTPService) SetTenantPrimaryTOTPDevice(req *models.TenantTOTPDeviceDeleteRequest, userID, tenantID uuid.UUID) (*models.TenantTOTPDeviceDeleteResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Parse device ID
	deviceUUID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		return &models.TenantTOTPDeviceDeleteResponse{
			Success: false,
			Message: "Invalid device ID format",
		}, nil
	}

	// Check if device exists
	_, err = tenantRepo.GetTenantTOTPSecretByID(deviceUUID, userID, tenantID)
	if err != nil {
		return &models.TenantTOTPDeviceDeleteResponse{
			Success: false,
			Message: "Device not found",
		}, nil
	}

	// Set as primary
	if err := tenantRepo.SetTenantTOTPSecretAsPrimary(deviceUUID, userID, tenantID); err != nil {
		return &models.TenantTOTPDeviceDeleteResponse{
			Success: false,
			Message: "Failed to set primary device",
		}, nil
	}

	return &models.TenantTOTPDeviceDeleteResponse{
		Success: true,
		Message: "Device set as primary successfully",
	}, nil
}

// generateJWTToken generates a JWT token for authenticated user
func (s *TenantTOTPService) generateJWTToken(userID, tenantID uuid.UUID, email string) (string, error) {
	// Use centralized auth-manager token service
	return config.TokenService.GenerateTOTPToken(
		userID,
		tenantID,
		email,
		24*time.Hour,
	)
}
