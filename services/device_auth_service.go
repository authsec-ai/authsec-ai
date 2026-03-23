package services

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// DeviceAuthService handles device authorization business logic
type DeviceAuthService struct {
	deviceRepo      *database.DeviceAuthRepository
	tenantRepo      *database.AdminTenantRepository
	userRepo        *database.UserRepository
	dbService       *database.TenantDBService
	baseURL         string
	pollingInterval int // Minimum seconds between polling attempts
	codeExpiry      time.Duration
}

// NewDeviceAuthService creates a new device authorization service
func NewDeviceAuthService(db *database.DBConnection, dbService *database.TenantDBService) *DeviceAuthService {
	baseURL := os.Getenv("BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	return &DeviceAuthService{
		deviceRepo:      database.NewDeviceAuthRepository(db),
		tenantRepo:      database.NewAdminTenantRepository(db),
		userRepo:        database.NewUserRepository(db),
		dbService:       dbService,
		baseURL:         baseURL,
		pollingInterval: 5,                // 5 seconds minimum between polls
		codeExpiry:      15 * time.Minute, // Device codes expire in 15 minutes
	}
}

// tenantMapping maps client ID to tenant ID using tenant_mappings table
func (s *DeviceAuthService) tenantMapping(clientID uuid.UUID) (uuid.UUID, error) {
	db := config.GetDatabase()
	if db == nil {
		return uuid.UUID{}, fmt.Errorf("database not initialized")
	}

	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM tenant_mappings WHERE client_id = $1`
	err := db.QueryRow(query, clientID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.UUID{}, fmt.Errorf("client not found in tenant_mappings")
		}
		return uuid.UUID{}, fmt.Errorf("failed to lookup tenant mapping: %w", err)
	}

	return tenantID, nil
}

// InitiateDeviceFlow creates a new device authorization request
func (s *DeviceAuthService) InitiateDeviceFlow(req *models.DeviceCodeRequest) (*models.DeviceCodeResponse, error) {
	// Parse and validate client_id
	clientID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return nil, fmt.Errorf("invalid client_id format")
	}

	// Look up tenant_id from tenant_mappings table using client_id
	tenantID, err := s.tenantMapping(clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tenant from client_id: %w", err)
	}

	// Generate device code (long secret)
	deviceCode, err := database.GenerateDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device code: %w", err)
	}

	// Generate user code (short human-readable)
	userCode, err := database.GenerateUserCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user code: %w", err)
	}

	// Build verification URIs
	verificationURI := fmt.Sprintf("%s/activate", s.baseURL)
	verificationURIComplete := fmt.Sprintf("%s/activate?user_code=%s", s.baseURL, userCode)

	// Default scopes if not provided
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	// Create device code record
	dc := &models.DeviceCode{
		ID:                      uuid.New(),
		TenantID:                tenantID,
		ClientID:                &clientID, // Store client_id from request
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		Status:                  "pending",
		Scopes:                  scopes,
		DeviceInfo:              req.DeviceInfo,
		ExpiresAt:               time.Now().Add(s.codeExpiry).Unix(),
	}
	// You can add client validation later if needed

	// Save to database
	if err := s.deviceRepo.CreateDeviceCode(dc); err != nil {
		return nil, fmt.Errorf("failed to create device code: %w", err)
	}

	return &models.DeviceCodeResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresIn:               int(s.codeExpiry.Seconds()),
		Interval:                s.pollingInterval,
	}, nil
}

// GetDeviceActivationInfo retrieves device information for activation page
func (s *DeviceAuthService) GetDeviceActivationInfo(userCode string) (*models.DeviceActivationInfoResponse, error) {
	dc, err := s.deviceRepo.FindByUserCode(userCode)
	if err != nil {
		return nil, fmt.Errorf("invalid user code")
	}

	// Check if expired
	if dc.IsExpired() {
		return nil, fmt.Errorf("device code expired")
	}

	// Check if already processed
	if dc.Status != "pending" {
		return nil, fmt.Errorf("device code already processed")
	}

	// Get tenant info
	tenant, err := s.tenantRepo.GetTenantByID(dc.TenantID.String())
	if err != nil {
		return nil, fmt.Errorf("tenant not found")
	}

	return &models.DeviceActivationInfoResponse{
		Success:      true,
		UserCode:     dc.UserCode,
		TenantDomain: tenant.TenantDomain,
		DeviceInfo:   dc.DeviceInfo,
		Scopes:       dc.Scopes,
		ExpiresAt:    time.Unix(dc.ExpiresAt, 0).Format(time.RFC3339),
	}, nil
}

// VerifyDeviceCode authorizes or denies a device code
func (s *DeviceAuthService) VerifyDeviceCode(userCode string, userID uuid.UUID, userEmail string, approve bool) error {
	// Verify device code exists and is pending
	dc, err := s.deviceRepo.FindByUserCode(userCode)
	if err != nil {
		return fmt.Errorf("invalid user code")
	}

	// Check if expired
	if dc.IsExpired() {
		s.deviceRepo.UpdateStatus(dc.DeviceCode, "expired")
		return fmt.Errorf("device code expired")
	}

	// Check if already processed
	if dc.Status != "pending" {
		return fmt.Errorf("device code already processed")
	}

	// Authorize or deny
	return s.deviceRepo.AuthorizeDeviceCode(userCode, userID, userEmail, approve)
}

// PollForToken polls for device authorization and returns token if authorized
func (s *DeviceAuthService) PollForToken(deviceCode string, clientID string) (*models.DeviceTokenResponse, error) {
	// Find device code
	dc, err := s.deviceRepo.FindByDeviceCode(deviceCode)
	if err != nil {
		return &models.DeviceTokenResponse{
			Error:            models.ErrorExpiredToken,
			ErrorDescription: "Device code not found or expired",
		}, nil
	}

	// Update last polled timestamp
	s.deviceRepo.UpdateLastPolled(deviceCode)

	// Check if expired
	if dc.IsExpired() {
		s.deviceRepo.UpdateStatus(deviceCode, "expired")
		return &models.DeviceTokenResponse{
			Error:            models.ErrorExpiredToken,
			ErrorDescription: "Device code expired",
		}, nil
	}

	// Check status
	switch dc.Status {
	case "pending":
		// Still waiting for user authorization
		return &models.DeviceTokenResponse{
			Error:            models.ErrorAuthorizationPending,
			ErrorDescription: "User has not authorized the device yet",
		}, nil

	case "denied":
		// User denied authorization
		return &models.DeviceTokenResponse{
			Error:            models.ErrorAccessDenied,
			ErrorDescription: "User denied the authorization request",
		}, nil

	case "authorized":
		// User authorized - generate token
		if dc.UserID == nil {
			return &models.DeviceTokenResponse{
				Error:            "server_error",
				ErrorDescription: "Device authorized but user ID missing",
			}, nil
		}

		// Get user from tenant database using existing repository method
		user, err := s.userRepo.GetUserByID(*dc.UserID)
		if err != nil {
			return &models.DeviceTokenResponse{
				Error:            "server_error",
				ErrorDescription: "User not found in tenant database",
			}, nil
		}

		// Get tenant info for token claims
		tenant, err := s.tenantRepo.GetTenantByID(dc.TenantID.String())
		if err != nil {
			return &models.DeviceTokenResponse{
				Error:            "server_error",
				ErrorDescription: "Tenant not found",
			}, nil
		}

		// Generate JWT token
		token, err := s.generateJWTToken(user, tenant, dc.Scopes)
		if err != nil {
			return &models.DeviceTokenResponse{
				Error:            "server_error",
				ErrorDescription: "Failed to generate token",
			}, nil
		}

		// Mark device code as consumed
		s.deviceRepo.MarkAsConsumed(deviceCode)

		return &models.DeviceTokenResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   86400, // 24 hours
			Scope:       strings.Join(dc.Scopes, " "),
		}, nil

	case "consumed":
		// Token already issued - prevent replay
		return &models.DeviceTokenResponse{
			Error:            models.ErrorExpiredToken,
			ErrorDescription: "Device code already used",
		}, nil

	case "expired":
		return &models.DeviceTokenResponse{
			Error:            models.ErrorExpiredToken,
			ErrorDescription: "Device code expired",
		}, nil

	default:
		return &models.DeviceTokenResponse{
			Error:            "server_error",
			ErrorDescription: "Unknown device code status",
		}, nil
	}
}

// generateJWTToken generates a JWT token using THE SAME logic as enduser login
// This EXACTLY mirrors the token generation in controllers/enduser_auth_controller.go
func (s *DeviceAuthService) generateJWTToken(user *models.ExtendedUser, tenant *models.Tenant, scopes []string) (string, error) {
	// Use centralized auth-manager token service
	return config.TokenService.GenerateDeviceAuthToken(
		user.ID,
		tenant.ID,
		user.Email,
		scopes,
		24*time.Hour,
	)
}

// CleanupExpiredCodes runs periodic cleanup of expired device codes
func (s *DeviceAuthService) CleanupExpiredCodes() (int64, error) {
	// Mark expired codes
	expired, err := s.deviceRepo.ExpireOldDeviceCodes()
	if err != nil {
		return 0, fmt.Errorf("failed to expire old codes: %w", err)
	}

	// Delete old expired/consumed/denied codes (older than 24 hours)
	deleted, err := s.deviceRepo.DeleteExpiredDeviceCodes(24 * time.Hour)
	if err != nil {
		return expired, fmt.Errorf("failed to delete expired codes: %w", err)
	}

	return expired + deleted, nil
}
