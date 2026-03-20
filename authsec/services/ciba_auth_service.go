package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// CIBAAuthService handles CIBA authentication business logic
// Mirrors DeviceAuthService but uses push notifications instead of device codes
type CIBAAuthService struct {
	cibaRepo      *database.CIBAAuthRepository
	userRepo      *database.UserRepository
	tenantRepo    *database.AdminTenantRepository
	pushService   *PushNotificationService
	dbService     *database.TenantDBService
	pollingInterval int
	requestExpiry time.Duration
}

// NewCIBAAuthService creates a new CIBA authentication service
func NewCIBAAuthService(
	db *database.DBConnection,
	dbService *database.TenantDBService,
	pushService *PushNotificationService,
) *CIBAAuthService {
	return &CIBAAuthService{
		cibaRepo:      database.NewCIBAAuthRepository(db),
		userRepo:      database.NewUserRepository(db),
		tenantRepo:    database.NewAdminTenantRepository(db),
		pushService:   pushService,
		dbService:     dbService,
		pollingInterval: 5,                // 5 seconds minimum between polls
		requestExpiry:   5 * time.Minute,  // Requests expire in 5 minutes
	}
}

// generateAuthReqID generates a unique authentication request ID
func (s *CIBAAuthService) generateAuthReqID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// InitiateCIBAAuth initiates CIBA authentication flow
// This is like InitiateDeviceFlow but sends push notification instead of returning user_code
func (s *CIBAAuthService) InitiateCIBAAuth(req *models.CIBAInitiateRequest) (*models.CIBAInitiateResponse, error) {
	// Step 1: Look up user by email (cross-tenant lookup, just like TOTP flow)
	user, err := s.lookupUserByEmail(req.LoginHint)
	if err != nil {
		return &models.CIBAInitiateResponse{
			Error:            models.CIBAErrorUserNotFound,
			ErrorDescription: fmt.Sprintf("User not found: %s", req.LoginHint),
		}, nil
	}

	// Step 2: Get user's registered push devices
	devices, err := s.cibaRepo.GetDeviceTokensByUserID(user.ID, user.TenantID)
	if err != nil || len(devices) == 0 {
		return &models.CIBAInitiateResponse{
			Error:            models.CIBAErrorNoDevice,
			ErrorDescription: "User has no registered push notification devices",
		}, nil
	}

	// Use first active device
	device := devices[0]

	// Step 3: Generate auth_req_id
	authReqID, err := s.generateAuthReqID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth_req_id: %w", err)
	}

	// Step 4: Parse client_id if provided
	var clientID *uuid.UUID
	if req.ClientID != "" {
		cid, err := uuid.Parse(req.ClientID)
		if err == nil {
			clientID = &cid
		}
	}

	// Default scopes if not provided
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	// Step 5: Create CIBA auth request in database
	authRequest := &models.CIBAAuthRequest{
		ID:             uuid.New(),
		AuthReqID:      authReqID,
		UserID:         user.ID,
		TenantID:       user.TenantID,
		UserEmail:      user.Email,
		ClientID:       clientID,
		DeviceTokenID:  device.ID,
		BindingMessage: req.BindingMessage,
		Scopes:         scopes,
		Status:         "pending",
		ExpiresAt:      time.Now().Add(s.requestExpiry).Unix(),
	}

	if err := s.cibaRepo.CreateCIBAAuthRequest(authRequest); err != nil {
		return nil, fmt.Errorf("failed to create CIBA request: %w", err)
	}

	// Step 6: Send push notification
	bindingMsg := req.BindingMessage
	if bindingMsg == "" {
		bindingMsg = "Authentication request"
	}

	if s.pushService != nil {
		err = s.pushService.SendAuthRequest(
			device.DeviceToken,
			authReqID,
			bindingMsg,
			user.Email,
		)
		if err != nil {
			// Log error but don't fail - request is still valid
			fmt.Printf("Failed to send push notification: %v\n", err)
		} else {
			// Update device last_used
			s.cibaRepo.UpdateDeviceTokenLastUsed(device.ID)
		}
	}

	return &models.CIBAInitiateResponse{
		AuthReqID: authReqID,
		ExpiresIn: int(s.requestExpiry.Seconds()),
		Interval:  s.pollingInterval,
		Message:   "Push notification sent to user's device",
	}, nil
}

// RespondToCIBA handles user's approve/deny response from mobile app
func (s *CIBAAuthService) RespondToCIBA(req *models.CIBARespondRequest) (*models.CIBARespondResponse, error) {
	// Get CIBA request
	authReq, err := s.cibaRepo.GetCIBAAuthRequestByID(req.AuthReqID)
	if err != nil {
		return nil, fmt.Errorf("invalid auth_req_id")
	}

	// Check if expired
	if authReq.IsExpired() {
		s.cibaRepo.UpdateCIBAAuthRequestStatus(req.AuthReqID, "expired", false)
		return nil, fmt.Errorf("request expired")
	}

	// Check if already processed
	if authReq.Status != "pending" {
		return nil, fmt.Errorf("request already processed")
	}

	// Update status
	status := "denied"
	message := "Authentication denied"
	if req.Approved {
		status = "approved"
		message = "Authentication approved"
	}

	if err := s.cibaRepo.UpdateCIBAAuthRequestStatus(req.AuthReqID, status, req.BiometricVerified); err != nil {
		return nil, fmt.Errorf("failed to update status: %w", err)
	}

	return &models.CIBARespondResponse{
		Success: true,
		Message: message,
	}, nil
}

// PollForToken polls for CIBA authentication status and returns token if approved
// This is like PollForToken in DeviceAuthService
func (s *CIBAAuthService) PollForToken(authReqID string) (*models.CIBATokenResponse, error) {
	// Get CIBA request
	authReq, err := s.cibaRepo.GetCIBAAuthRequestByID(authReqID)
	if err != nil {
		return &models.CIBATokenResponse{
			Error:            models.CIBAErrorExpiredToken,
			ErrorDescription: "Request not found or expired",
		}, nil
	}

	// Update last polled timestamp
	s.cibaRepo.UpdateLastPolled(authReqID)

	// Check if expired
	if authReq.IsExpired() {
		s.cibaRepo.UpdateCIBAAuthRequestStatus(authReqID, "expired", false)
		return &models.CIBATokenResponse{
			Error:            models.CIBAErrorExpiredToken,
			ErrorDescription: "Request expired",
		}, nil
	}

	// Check status
	switch authReq.Status {
	case "pending":
		// Still waiting for user response
		return &models.CIBATokenResponse{
			Error:            models.CIBAErrorAuthorizationPending,
			ErrorDescription: "User has not responded yet",
		}, nil

	case "denied":
		// User denied
		return &models.CIBATokenResponse{
			Error:            models.CIBAErrorAccessDenied,
			ErrorDescription: "User denied the authentication request",
		}, nil

	case "approved":
		// User approved - generate token
		// Get user from tenant database
		user, err := s.userRepo.GetUserByID(authReq.UserID)
		if err != nil {
			return &models.CIBATokenResponse{
				Error:            "server_error",
				ErrorDescription: "User not found",
			}, nil
		}

		// Get tenant info
		tenant, err := s.tenantRepo.GetTenantByID(authReq.TenantID.String())
		if err != nil {
			return &models.CIBATokenResponse{
				Error:            "server_error",
				ErrorDescription: "Tenant not found",
			}, nil
		}

		// Generate JWT token (same logic as device flow)
		token, err := s.generateJWTToken(user, tenant, authReq.Scopes)
		if err != nil {
			return &models.CIBATokenResponse{
				Error:            "server_error",
				ErrorDescription: "Failed to generate token",
			}, nil
		}

		// Mark as consumed
		s.cibaRepo.MarkAsConsumed(authReqID)

		return &models.CIBATokenResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   365 * 24 * 3600, // 365 days
			Scope:       strings.Join(authReq.Scopes, " "),
		}, nil

	case "consumed":
		// Token already issued
		return &models.CIBATokenResponse{
			Error:            models.CIBAErrorExpiredToken,
			ErrorDescription: "Request already used",
		}, nil

	case "expired":
		return &models.CIBATokenResponse{
			Error:            models.CIBAErrorExpiredToken,
			ErrorDescription: "Request expired",
		}, nil

	default:
		return &models.CIBATokenResponse{
			Error:            "server_error",
			ErrorDescription: "Unknown status",
		}, nil
	}
}

// RegisterDevice registers a new device for push notifications
func (s *CIBAAuthService) RegisterDevice(userID uuid.UUID, tenantID uuid.UUID, req *models.DeviceTokenRegistrationRequest) (*models.DeviceTokenRegistrationResponse, error) {
	deviceToken := &models.DeviceToken{
		ID:          uuid.New(),
		UserID:      userID,
		TenantID:    tenantID,
		DeviceToken: req.DeviceToken,
		Platform:    req.Platform,
		DeviceName:  req.DeviceName,
		DeviceModel: req.DeviceModel,
		AppVersion:  req.AppVersion,
		OSVersion:   req.OSVersion,
		IsActive:    true,
	}

	if err := s.cibaRepo.CreateDeviceToken(deviceToken); err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return &models.DeviceTokenRegistrationResponse{
		Success:  true,
		DeviceID: deviceToken.ID.String(),
		Message:  "Device registered successfully for push notifications",
	}, nil
}

// lookupUserByEmail looks up user by email across tenant databases
// This is exactly like TOTP flow - email is unique identifier
func (s *CIBAAuthService) lookupUserByEmail(email string) (*models.ExtendedUser, error) {
	// Get all tenants
	tenants, err := s.tenantRepo.GetAllTenants()
	if err != nil {
		return nil, fmt.Errorf("failed to get tenants: %w", err)
	}

	// Search each tenant database for user with this email
	for _, tenant := range tenants {
		// Try to find user by email in this tenant
		user, err := s.userRepo.GetUserByEmailAndTenant(email, tenant.ID)
		if err == nil && user != nil {
			// Found user!
			return user, nil
		}
	}

	return nil, fmt.Errorf("user not found: %s", email)
}

// generateJWTToken generates JWT token (same as DeviceAuthService)
func (s *CIBAAuthService) generateJWTToken(user *models.ExtendedUser, tenant *models.Tenant, scopes []string) (string, error) {
	// Use centralized auth-manager token service
	return config.TokenService.GenerateCIBAToken(
		user.ID,
		tenant.ID,
		user.Email,
		scopes,
		365 * 24 * time.Hour,
	)
}

// CleanupExpiredRequests runs periodic cleanup
func (s *CIBAAuthService) CleanupExpiredRequests() (int64, error) {
	// Mark expired
	expired, err := s.cibaRepo.ExpireOldRequests()
	if err != nil {
		return 0, fmt.Errorf("failed to expire old requests: %w", err)
	}

	// Delete old ones (older than 24 hours)
	deleted, err := s.cibaRepo.DeleteExpiredRequests(24 * time.Hour)
	if err != nil {
		return expired, fmt.Errorf("failed to delete expired requests: %w", err)
	}

	return expired + deleted, nil
}

// ========================================
// Device Management (Admin APIs)
// ========================================

// GetUserDevices retrieves all registered push devices for a user
func (s *CIBAAuthService) GetUserDevices(userID, tenantID uuid.UUID) ([]models.DeviceSummary, error) {
	devices, err := s.cibaRepo.GetDeviceTokensByUserID(userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve devices: %w", err)
	}

	// Convert to summaries (omit sensitive device token)
	summaries := make([]models.DeviceSummary, len(devices))
	for i, device := range devices {
		summaries[i] = models.DeviceSummary{
			ID:          device.ID.String(),
			DeviceName:  device.DeviceName,
			Platform:    device.Platform,
			DeviceModel: device.DeviceModel,
			AppVersion:  device.AppVersion,
			OSVersion:   device.OSVersion,
			IsActive:    device.IsActive,
			LastUsed:    device.LastUsed,
			CreatedAt:   device.CreatedAt,
		}
	}

	return summaries, nil
}

// DeleteDevice deactivates a user's push notification device
func (s *CIBAAuthService) DeleteDevice(deviceID, userID, tenantID uuid.UUID) error {
	// Verify device belongs to user and tenant
	device, err := s.cibaRepo.GetDeviceTokenByID(deviceID)
	if err != nil {
		return fmt.Errorf("device not found: %w", err)
	}

	if device.UserID != userID || device.TenantID != tenantID {
		return fmt.Errorf("device not found or unauthorized")
	}

	// Deactivate device
	if err := s.cibaRepo.DeactivateDeviceToken(deviceID, userID, tenantID); err != nil {
		return fmt.Errorf("failed to deactivate device: %w", err)
	}

	return nil
}
