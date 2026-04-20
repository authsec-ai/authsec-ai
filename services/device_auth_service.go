package services

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// DeviceAuthService handles device authorization business logic (RFC 8628)
type DeviceAuthService struct {
	deviceRepo      *database.DeviceAuthRepository
	tenantRepo      *database.AdminTenantRepository
	userRepo        *database.UserRepository
	dbService       *database.TenantDBService
	verificationURI string
	pollingInterval int // minimum seconds between polling attempts
	codeExpiry      time.Duration
}

// NewDeviceAuthService creates a new device authorization service
func NewDeviceAuthService(db *database.DBConnection, dbService *database.TenantDBService) *DeviceAuthService {
	verificationURI := os.Getenv("DEVICE_VERIFICATION_URI")
	if verificationURI == "" {
		verificationURI = "https://app.authsec.ai/activate"
	}

	return &DeviceAuthService{
		deviceRepo:      database.NewDeviceAuthRepository(db),
		tenantRepo:      database.NewAdminTenantRepository(db),
		userRepo:        database.NewUserRepository(db),
		dbService:       dbService,
		verificationURI: verificationURI,
		pollingInterval: 5,              // RFC 8628 recommended minimum
		codeExpiry:      5 * time.Minute, // spec: 5 min (300s)
	}
}

// InitiateDeviceFlow creates a new device authorization request.
// No client_id or tenant_domain is required — the CLI sends only scopes.
// Tenant context is bound during the /authorize step from the user's browser session.
func (s *DeviceAuthService) InitiateDeviceFlow(req *models.DeviceCodeRequest) (*models.DeviceCodeResponse, error) {
	deviceCode, err := database.GenerateDeviceCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate device code: %w", err)
	}

	userCode, err := database.GenerateUserCode()
	if err != nil {
		return nil, fmt.Errorf("failed to generate user code: %w", err)
	}

	verificationURIComplete := fmt.Sprintf("%s?user_code=%s", s.verificationURI, userCode)

	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	dc := &models.DeviceCode{
		ID:                      uuid.New(),
		TenantID:                nil, // filled on /authorize
		ClientID:                nil, // filled on /authorize
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         s.verificationURI,
		VerificationURIComplete: verificationURIComplete,
		Status:                  "pending",
		Scopes:                  scopes,
		DeviceInfo:              req.DeviceInfo,
		ExpiresAt:               time.Now().Add(s.codeExpiry).Unix(),
	}

	if err := s.deviceRepo.CreateDeviceCode(dc); err != nil {
		return nil, fmt.Errorf("failed to create device code: %w", err)
	}

	return &models.DeviceCodeResponse{
		DeviceCode:              deviceCode,
		UserCode:                userCode,
		VerificationURI:         s.verificationURI,
		VerificationURIComplete: verificationURIComplete,
		ExpiresIn:               int(s.codeExpiry.Seconds()),
		Interval:                s.pollingInterval,
	}, nil
}

// GetDeviceActivationInfo retrieves device information for the activation page
func (s *DeviceAuthService) GetDeviceActivationInfo(userCode string) (*models.DeviceActivationInfoResponse, error) {
	dc, err := s.deviceRepo.FindByUserCode(userCode)
	if err != nil {
		return nil, fmt.Errorf("invalid user code")
	}
	if dc.IsExpired() {
		return nil, fmt.Errorf("device code expired")
	}
	if dc.Status != "pending" {
		return nil, fmt.Errorf("device code already processed")
	}

	resp := &models.DeviceActivationInfoResponse{
		Success:    true,
		UserCode:   dc.UserCode,
		Scopes:     dc.Scopes,
		ExpiresAt:  time.Unix(dc.ExpiresAt, 0).Format(time.RFC3339),
		DeviceInfo: dc.DeviceInfo,
	}

	// Tenant info is unknown until /authorize — omit tenant_domain from info response
	return resp, nil
}

// AuthorizeDevice approves or denies a device code.
// When approved, generates the access token immediately and stores it in the DB.
// The CLI's next /token poll picks it up directly — no token re-generation needed.
func (s *DeviceAuthService) AuthorizeDevice(
	userCode string,
	userID uuid.UUID,
	userEmail string,
	tenantID uuid.UUID,
	tenantDomain string,
	clientID *uuid.UUID,
	approve bool,
) error {
	dc, err := s.deviceRepo.FindByUserCode(userCode)
	if err != nil {
		return fmt.Errorf("invalid user code")
	}
	if dc.IsExpired() {
		s.deviceRepo.UpdateStatus(dc.DeviceCode, "expired")
		return fmt.Errorf("device code expired")
	}
	if dc.Status != "pending" {
		return fmt.Errorf("device code already processed")
	}

	// Generate token at authorize time (empty string when denied)
	accessToken := ""
	if approve {
		tenant, tErr := s.tenantRepo.GetTenantByID(tenantID.String())
		if tErr != nil {
			return fmt.Errorf("tenant not found")
		}

		// Try end-user table first, then fall back to using JWT claims directly.
		// Admin users (who authenticated via SSO) are not in the end-user 'users' table,
		// but we already have their identity from the JWT — no lookup needed.
		user, uErr := s.userRepo.GetUserByID(userID)
		if uErr != nil {
			// User not in end-user table — use JWT claims directly
			user = &models.ExtendedUser{}
			user.ID = userID
			user.Email = userEmail
			user.TenantID = tenantID
		}

		accessToken, err = s.generateJWTToken(user, tenant, dc.Scopes, clientID)
		if err != nil {
			return fmt.Errorf("failed to generate access token: %w", err)
		}
	}

	return s.deviceRepo.AuthorizeDeviceCode(userCode, userID, userEmail, tenantID, tenantDomain, clientID, accessToken, approve)
}

// VerifyDeviceCode is the legacy alias for AuthorizeDevice (used by legacy /verify endpoint).
func (s *DeviceAuthService) VerifyDeviceCode(
	userCode string,
	userID uuid.UUID,
	userEmail string,
	tenantID uuid.UUID,
	tenantDomain string,
	clientID *uuid.UUID,
	approve bool,
) error {
	return s.AuthorizeDevice(userCode, userID, userEmail, tenantID, tenantDomain, clientID, approve)
}

// ValidateUserCode looks up a user_code and returns its record ID and remaining TTL.
// Used by the public /verify endpoint (activation page pre-login check).
func (s *DeviceAuthService) ValidateUserCode(userCode string) (*models.DeviceCode, error) {
	dc, err := s.deviceRepo.FindByUserCode(userCode)
	if err != nil {
		return nil, fmt.Errorf("invalid user code")
	}
	if dc.IsExpired() {
		s.deviceRepo.UpdateStatus(dc.DeviceCode, "expired")
		return nil, fmt.Errorf("expired")
	}
	if dc.Status != "pending" {
		return nil, fmt.Errorf("device code already processed")
	}
	return dc, nil
}

// PollForToken polls for device authorization status and returns a token when authorized.
// Enforces a minimum polling interval per device_code (DB-level rate limit).
func (s *DeviceAuthService) PollForToken(deviceCode string) (*models.DeviceTokenResponse, error) {
	dc, err := s.deviceRepo.FindByDeviceCode(deviceCode)
	if err != nil {
		return &models.DeviceTokenResponse{
			Error:            models.ErrorExpiredToken,
			ErrorDescription: "Device code not found or expired",
		}, nil
	}

	// Check expiry before rate-limit update
	if dc.IsExpired() {
		s.deviceRepo.UpdateStatus(deviceCode, "expired")
		return &models.DeviceTokenResponse{
			Error:            models.ErrorExpiredToken,
			ErrorDescription: "Device code expired",
		}, nil
	}

	// Rate limit: max 1 poll per pollingInterval seconds (DB atomic check-and-update)
	tooSoon, err := s.deviceRepo.UpdateLastPolled(deviceCode, int64(s.pollingInterval))
	if err != nil {
		return nil, fmt.Errorf("failed to update poll timestamp: %w", err)
	}
	if tooSoon {
		return &models.DeviceTokenResponse{
			Error:            models.ErrorSlowDown,
			ErrorDescription: fmt.Sprintf("Polling too frequently. Minimum interval is %d seconds.", s.pollingInterval),
		}, nil
	}

	switch dc.Status {
	case "pending":
		return &models.DeviceTokenResponse{
			Error:            models.ErrorAuthorizationPending,
			ErrorDescription: "User has not authorized the device yet",
		}, nil

	case "denied":
		return &models.DeviceTokenResponse{
			Error:            models.ErrorAccessDenied,
			ErrorDescription: "User denied the authorization request",
		}, nil

	case "authorized":
		// Token was generated and stored at /authorize time — return it directly.
		if dc.AccessToken == "" {
			return &models.DeviceTokenResponse{
				Error:            "server_error",
				ErrorDescription: "Device authorized but access token missing",
			}, nil
		}

		clientIDStr := ""
		if dc.ClientID != nil {
			clientIDStr = dc.ClientID.String()
		}
		userIDStr := ""
		if dc.UserID != nil {
			userIDStr = dc.UserID.String()
		}
		tenantIDStr := ""
		if dc.TenantID != nil {
			tenantIDStr = dc.TenantID.String()
		}

		// Invalidate device code (one-time use)
		s.deviceRepo.MarkAsConsumed(deviceCode)

		return &models.DeviceTokenResponse{
			AccessToken:  dc.AccessToken,
			TokenType:    "Bearer",
			ExpiresIn:    86400,
			Scope:        strings.Join(dc.Scopes, " "),
			Email:        dc.UserEmail,
			UserID:       userIDStr,
			TenantID:     tenantIDStr,
			TenantDomain: dc.TenantDomain,
			ClientID:     clientIDStr,
		}, nil

	case "consumed":
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

// generateJWTToken generates a JWT using the centralized token service.
func (s *DeviceAuthService) generateJWTToken(
	user *models.ExtendedUser,
	tenant *models.Tenant,
	scopes []string,
	_ *uuid.UUID,
) (string, error) {
	return config.TokenService.GenerateDeviceAuthToken(
		user.ID,
		tenant.TenantID,
		user.Email,
		scopes,
		24*time.Hour,
	)
}

// CleanupExpiredCodes runs periodic cleanup of expired device codes
func (s *DeviceAuthService) CleanupExpiredCodes() (int64, error) {
	expired, err := s.deviceRepo.ExpireOldDeviceCodes()
	if err != nil {
		return 0, fmt.Errorf("failed to expire old codes: %w", err)
	}

	deleted, err := s.deviceRepo.DeleteExpiredDeviceCodes(24 * time.Hour)
	if err != nil {
		return expired, fmt.Errorf("failed to delete expired codes: %w", err)
	}

	return expired + deleted, nil
}
