package services

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/google/uuid"
)

// TenantCIBAAuthService handles CIBA authentication for tenant users
type TenantCIBAAuthService struct {
	tenantRepo      *database.TenantDeviceRepository
	adminTenantRepo *database.AdminTenantRepository
	pushService     *PushNotificationService
	pollingInterval int
	requestExpiry   time.Duration
}

// NewTenantCIBAAuthService creates a new tenant CIBA authentication service
func NewTenantCIBAAuthService(
	pushService *PushNotificationService,
) *TenantCIBAAuthService {
	return &TenantCIBAAuthService{
		adminTenantRepo: database.NewAdminTenantRepository(config.GetDatabase()),
		pushService:     pushService,
		pollingInterval: 5,               // 5 seconds minimum between polls
		requestExpiry:   5 * time.Minute, // Requests expire in 5 minutes
	}
}

// generateAuthReqID generates a unique authentication request ID
func (s *TenantCIBAAuthService) generateAuthReqID() (string, error) {
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// tenantMapping maps client ID to tenant ID using tenant_mappings table
func (s *TenantCIBAAuthService) tenantMapping(clientID uuid.UUID) (uuid.UUID, error) {
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

// InitiateTenantCIBAAuth initiates CIBA authentication for tenant users
func (s *TenantCIBAAuthService) InitiateTenantCIBAAuth(req *models.TenantCIBAInitiateRequest) (*models.TenantCIBAInitiateResponse, error) {
	// Step 1: Parse and validate client_id
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return &models.TenantCIBAInitiateResponse{
			Error:            models.TenantCIBAErrorInvalidClient,
			ErrorDescription: "Invalid client ID format",
		}, nil
	}

	// Step 2: Map client_id to tenant_id
	tenantUUID, err := s.tenantMapping(clientUUID)
	if err != nil {
		return &models.TenantCIBAInitiateResponse{
			Error:            models.TenantCIBAErrorInvalidClient,
			ErrorDescription: "Client not found or not mapped to tenant",
		}, nil
	}

	// Step 3: Get tenant information (validate existence)
	// tenant, err := s.adminTenantRepo.GetTenantByUUID(tenantUUID)
	// if err != nil {
	// 	return &models.TenantCIBAInitiateResponse{
	// 		Error:            models.TenantCIBAErrorTenantNotFound,
	// 		ErrorDescription: "Tenant not found",
	// 	}, nil
	// }

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
		return &models.TenantCIBAInitiateResponse{
			Error:            models.TenantCIBAErrorUserNotFound,
			ErrorDescription: fmt.Sprintf("User not found: %s", req.Email),
		}, nil
	}

	// Step 6: Get user's registered push devices from tenant DB
	devices, err := tenantRepo.GetTenantDeviceTokensByUserID(user.ID, tenantUUID)
	if err != nil || len(devices) == 0 {
		return &models.TenantCIBAInitiateResponse{
			Error:            models.TenantCIBAErrorNoDevice,
			ErrorDescription: "User has no registered push notification devices",
		}, nil
	}

	// Use first active device
	device := devices[0]

	// Step 7: Generate auth_req_id
	authReqID, err := s.generateAuthReqID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate auth_req_id: %w", err)
	}

	// Default scopes if not provided
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	// Step 8: Create CIBA authentication request in tenant DB
	cibaRequest := &models.TenantCIBAAuthRequest{
		AuthReqID:      authReqID,
		UserID:         user.ID,
		TenantID:       tenantUUID,
		UserEmail:      strings.ToLower(req.Email),
		ClientID:       &clientUUID,
		DeviceTokenID:  device.ID,
		BindingMessage: req.BindingMessage,
		Scopes:         models.JSONStringArray(scopes),
		Status:         "pending",
	}

	if err := tenantRepo.CreateTenantCIBAAuthRequest(cibaRequest); err != nil {
		return nil, fmt.Errorf("failed to create CIBA request: %w", err)
	}

	// Step 9: Send push notification to user's device
	if s.pushService != nil {
		bindingMessage := req.BindingMessage
		if bindingMessage == "" {
			bindingMessage = "Tap to approve sign-in"
		}

		err = s.pushService.SendAuthRequest(
			device.DeviceToken,
			authReqID,
			bindingMessage,
			strings.ToLower(req.Email),
		)
		if err != nil {
			fmt.Printf("Warning: Failed to send push notification: %v\n", err)
		}
	}

	return &models.TenantCIBAInitiateResponse{
		AuthReqID: authReqID,
		ExpiresIn: int(s.requestExpiry.Seconds()),
		Interval:  s.pollingInterval,
		Message:   "Push notification sent to your registered device",
	}, nil
}

// RespondToTenantCIBA handles user response to CIBA authentication request
func (s *TenantCIBAAuthService) RespondToTenantCIBA(authReqID string, approved bool, biometricVerified bool, userID, tenantID uuid.UUID) (*models.TenantCIBARespondResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Step 1: Retrieve and validate the CIBA request
	request, err := tenantRepo.GetTenantCIBAAuthRequestByAuthReqID(authReqID, tenantID)
	if err != nil {
		return &models.TenantCIBARespondResponse{
			Success: false,
			Message: "Authentication request not found or expired",
		}, nil
	}

	// Step 2: Verify user ownership
	if request.UserID != userID {
		return &models.TenantCIBARespondResponse{
			Success: false,
			Message: "You are not authorized to respond to this request",
		}, nil
	}

	// Step 3: Check if request is still pending and not expired
	if !request.IsPending() {
		return &models.TenantCIBARespondResponse{
			Success: false,
			Message: "Request is no longer pending",
		}, nil
	}

	// Step 4: Update request status
	status := "approved"
	if !approved {
		status = "denied"
	}

	err = tenantRepo.UpdateTenantCIBAAuthRequestStatus(
		authReqID,
		tenantID,
		status,
		approved,
		biometricVerified,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update CIBA request: %w", err)
	}

	return &models.TenantCIBARespondResponse{
		Success: true,
		Message: fmt.Sprintf("Authentication %s", status),
	}, nil
}

// PollTenantCIBAToken polls for token completion
func (s *TenantCIBAAuthService) PollTenantCIBAToken(req *models.TenantCIBATokenRequest) (*models.TenantCIBATokenResponse, error) {
	// Parse and validate client_id
	clientUUID, err := uuid.Parse(req.ClientID)
	if err != nil {
		return &models.TenantCIBATokenResponse{
			Error:            models.TenantCIBAErrorInvalidClient,
			ErrorDescription: "Invalid client ID format",
		}, nil
	}

	// Map client_id to tenant_id
	tenantUUID, err := s.tenantMapping(clientUUID)
	if err != nil {
		return &models.TenantCIBATokenResponse{
			Error:            models.TenantCIBAErrorInvalidClient,
			ErrorDescription: "Client not found or not mapped to tenant",
		}, nil
	}

	// Connect to tenant database
	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Update last polled timestamp (do this asynchronously to avoid slowing down response)
	go func() {
		tenantRepo.UpdateTenantCIBAAuthRequestLastPolled(req.AuthReqID, tenantUUID)
	}()

	// Retrieve the CIBA request
	request, err := tenantRepo.GetTenantCIBAAuthRequestByAuthReqID(req.AuthReqID, tenantUUID)
	if err != nil {
		return &models.TenantCIBATokenResponse{
			Error:            models.TenantCIBAErrorExpiredToken,
			ErrorDescription: "Authentication request not found",
		}, nil
	}

	// Check if request has expired
	if request.IsExpired() {
		return &models.TenantCIBATokenResponse{
			Error:            models.TenantCIBAErrorExpiredToken,
			ErrorDescription: "Authentication request has expired",
		}, nil
	}

	// Check request status
	if request.Status == "pending" {
		return &models.TenantCIBATokenResponse{
			Error:            models.TenantCIBAErrorAuthorizationPending,
			ErrorDescription: "User has not responded to authentication request",
		}, nil
	}

	if request.Status == "denied" {
		return &models.TenantCIBATokenResponse{
			Error:            models.TenantCIBAErrorAccessDenied,
			ErrorDescription: "User denied the authentication request",
		}, nil
	}

	if request.Status == "approved" {
		// Generate JWT token with actual client_id from the CIBA request
		clientID := clientUUID
		if request.ClientID != nil {
			clientID = *request.ClientID
		}
		token, err := s.generateJWTToken(request.UserID, tenantUUID, clientID, request.UserEmail, request.Scopes)
		if err != nil {
			return nil, fmt.Errorf("failed to generate JWT token: %w", err)
		}

		// Mark request as consumed to prevent reuse
		tenantRepo.UpdateTenantCIBAAuthRequestStatus(
			req.AuthReqID,
			tenantUUID,
			"consumed",
			true,
			request.BiometricVerified,
		)

		// Update user's last login timestamp
		tenantRepo.UpdateTenantUserLastLogin(request.UserID)

		return &models.TenantCIBATokenResponse{
			AccessToken: token,
			TokenType:   "Bearer",
			ExpiresIn:   24 * 60 * 60, // 24 hours
			Scope:       strings.Join(request.Scopes, " "),
		}, nil
	}

	return &models.TenantCIBATokenResponse{
		Error:            models.TenantCIBAErrorExpiredToken,
		ErrorDescription: "Invalid request state",
	}, nil
}

// generateJWTToken generates a JWT token for authenticated user
func (s *TenantCIBAAuthService) generateJWTToken(userID, tenantID, clientID uuid.UUID, email string, scopes []string) (string, error) {
	// Use centralized auth-manager token service with correct client_id
	return config.TokenService.GenerateTenantCIBAToken(
		userID,
		tenantID,
		clientID,
		email,
		scopes,
		24*time.Hour,
	)
}

// RegisterTenantDevice registers a device for push notifications in tenant context
func (s *TenantCIBAAuthService) RegisterTenantDevice(req *models.TenantDeviceTokenRegistrationRequest, userID, tenantID uuid.UUID) (*models.TenantDeviceTokenRegistrationResponse, error) {
	// Connect to tenant database
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)

	// Check if device token already exists
	existingDevice, err := tenantRepo.GetTenantDeviceTokenByToken(req.DeviceToken, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to check device token: %w", err)
	}

	if existingDevice != nil {
		// Device already registered, update it
		existingDevice.DeviceName = req.DeviceName
		existingDevice.DeviceModel = req.DeviceModel
		existingDevice.AppVersion = req.AppVersion
		existingDevice.OSVersion = req.OSVersion
		existingDevice.IsActive = true

		if err := tenantRepo.UpdateTenantDeviceToken(existingDevice); err != nil {
			return nil, fmt.Errorf("failed to update device token: %w", err)
		}

		return &models.TenantDeviceTokenRegistrationResponse{
			Success:  true,
			DeviceID: existingDevice.ID.String(),
			Message:  "Device updated successfully",
		}, nil
	}

	// Create new device token
	deviceToken := &models.TenantDeviceToken{
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

	if err := tenantRepo.CreateTenantDeviceToken(deviceToken); err != nil {
		return nil, fmt.Errorf("failed to register device: %w", err)
	}

	return &models.TenantDeviceTokenRegistrationResponse{
		Success:  true,
		DeviceID: deviceToken.ID.String(),
		Message:  "Device registered successfully",
	}, nil
}
