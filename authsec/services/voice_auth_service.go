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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// VoiceAuthService handles voice authentication business logic
type VoiceAuthService struct {
	voiceRepo      *database.VoiceAuthRepository
	deviceRepo     *database.DeviceAuthRepository
	tenantRepo     *database.AdminTenantRepository
	userRepo       *database.UserRepository
	dbService      *database.TenantDBService
	deviceService  *DeviceAuthService
	sessionExpiry  time.Duration
	maxOTPAttempts int
}

// NewVoiceAuthService creates a new voice authentication service
func NewVoiceAuthService(db *database.DBConnection, dbService *database.TenantDBService, deviceService *DeviceAuthService) *VoiceAuthService {
	return &VoiceAuthService{
		voiceRepo:      database.NewVoiceAuthRepository(db),
		deviceRepo:     database.NewDeviceAuthRepository(db),
		tenantRepo:     database.NewAdminTenantRepository(db),
		userRepo:       database.NewUserRepository(db),
		dbService:      dbService,
		deviceService:  deviceService,
		sessionExpiry:  3 * time.Minute, // Voice sessions expire quickly (3 minutes)
		maxOTPAttempts: 5,
	}
}

// tenantMapping maps client ID to tenant ID using tenant_mappings table
func (s *VoiceAuthService) tenantMapping(clientID uuid.UUID) (uuid.UUID, error) {
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

// InitiateVoiceAuth creates a new voice authentication session
func (s *VoiceAuthService) InitiateVoiceAuth(req *models.VoiceInitiateRequest) (*models.VoiceInitiateResponse, error) {
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

	// Voice platform is optional and accepts any string (alexa, google, siri, web, custom, etc.)
	// No validation needed - store as-is for flexibility

	// Generate session token
	sessionToken, err := database.GenerateSessionToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session token: %w", err)
	}

	// Generate voice OTP (4-digit code)
	voiceOTP, err := database.GenerateVoiceOTP()
	if err != nil {
		return nil, fmt.Errorf("failed to generate voice OTP: %w", err)
	}

	// Default scopes if not provided
	scopes := req.Scopes
	if len(scopes) == 0 {
		scopes = []string{"openid", "email", "profile"}
	}

	// Create voice session
	session := &models.VoiceSession{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ClientID:      &clientID, // Store client_id from request
		SessionToken:  sessionToken,
		VoiceOTP:      voiceOTP,
		OTPAttempts:   0,
		VoicePlatform: req.VoicePlatform,
		VoiceUserID:   req.VoiceUserID,
		DeviceInfo:    req.DeviceInfo,
		Status:        "initiated",
		Scopes:        scopes,
		ExpiresAt:     time.Now().Add(s.sessionExpiry).Unix(),
	}

	// Save to database
	if err := s.voiceRepo.CreateVoiceSession(session); err != nil {
		return nil, fmt.Errorf("failed to create voice session: %w", err)
	}

	// Format OTP for voice output (e.g., "8532" -> "eight-five-three-two")
	voiceMessage := s.formatOTPForVoice(voiceOTP)

	return &models.VoiceInitiateResponse{
		SessionToken: sessionToken,
		VoiceOTP:     voiceOTP,
		ExpiresIn:    int(s.sessionExpiry.Seconds()),
		Message:      voiceMessage,
	}, nil
}

// VerifyVoiceOTP verifies the voice OTP and returns device code or token
func (s *VoiceAuthService) VerifyVoiceOTP(req *models.VoiceVerifyRequest) (*models.VoiceVerifyResponse, error) {
	// Find voice session
	session, err := s.voiceRepo.FindVoiceSessionByToken(req.SessionToken)
	if err != nil {
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  "failed",
			Message: "Invalid session token",
		}, nil
	}

	// Check if expired
	if session.IsExpired() {
		s.voiceRepo.UpdateVoiceSessionStatus(req.SessionToken, "expired")
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  "expired",
			Message: "Voice session expired",
		}, nil
	}

	// Check if already verified
	if session.Status != "initiated" {
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  session.Status,
			Message: "Session already processed",
		}, nil
	}

	// Check OTP attempts
	if !session.CanRetryOTP() {
		s.voiceRepo.UpdateVoiceSessionStatus(req.SessionToken, "failed")
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  "failed",
			Message: "Maximum OTP attempts exceeded",
		}, nil
	}

	// Verify OTP
	if session.VoiceOTP != req.VoiceOTP {
		s.voiceRepo.IncrementOTPAttempts(req.SessionToken)
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  "initiated",
			Message: fmt.Sprintf("Incorrect OTP. %d attempts remaining", s.maxOTPAttempts-session.OTPAttempts-1),
		}, nil
	}

	// OTP verified successfully
	// Check if voice user is pre-linked to an account
	if session.VoicePlatform != "" && session.VoiceUserID != "" {
		link, err := s.voiceRepo.FindVoiceIdentityLink(session.TenantID, session.VoicePlatform, session.VoiceUserID)
		if err == nil && link.IsActive {
			// User is pre-linked - can issue token directly
			s.voiceRepo.UpdateVoiceIdentityLinkLastUsed(link.ID)

			// Get user from repository
			user, err := s.userRepo.GetUserByID(link.UserID)
			if err != nil {
				return &models.VoiceVerifyResponse{
					Success: false,
					Status:  "failed",
					Message: "User not found",
				}, nil
			}

			// Get tenant info
			tenant, err := s.tenantRepo.GetTenantByID(session.TenantID.String())
			if err != nil {
				return &models.VoiceVerifyResponse{
					Success: false,
					Status:  "failed",
					Message: "Tenant not found",
				}, nil
			}

			// Generate token with session tracking
			token, err := s.generateJWTTokenWithSession(
				user,
				tenant,
				session.Scopes,
				session.VoicePlatform,
				session.VoiceUserID,
				session.DeviceInfo,
			)
			if err != nil {
				return &models.VoiceVerifyResponse{
					Success: false,
					Status:  "failed",
					Message: "Failed to generate token",
				}, nil
			}

			// Mark session as verified
			s.voiceRepo.VerifyVoiceSession(req.SessionToken, &link.UserID, link.UserEmail)

			return &models.VoiceVerifyResponse{
				Success:     true,
				Status:      "verified",
				Message:     "Authentication successful",
				AccessToken: token,
				TokenType:   "Bearer",
				ExpiresIn:   86400, // 24 hours
			}, nil
		}
	}

	// User not pre-linked - initiate device authorization flow
	// This allows user to complete authentication in browser
	deviceReq := &models.DeviceCodeRequest{
		ClientID:     session.ClientID.String(),
		TenantDomain: "", // Will be populated from tenant_id
		Scopes:       session.Scopes,
		DeviceInfo: map[string]interface{}{
			"source":         "voice_authentication",
			"voice_platform": session.VoicePlatform,
			"voice_user_id":  session.VoiceUserID,
		},
	}

	// Get tenant info
	tenant, err := s.tenantRepo.GetTenantByID(session.TenantID.String())
	if err != nil {
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  "failed",
			Message: "Tenant not found",
		}, nil
	}
	deviceReq.TenantDomain = tenant.TenantDomain

	// Create device code
	deviceResp, err := s.deviceService.InitiateDeviceFlow(deviceReq)
	if err != nil {
		return &models.VoiceVerifyResponse{
			Success: false,
			Status:  "failed",
			Message: "Failed to create device authorization",
		}, nil
	}

	// Link device code to voice session
	s.voiceRepo.LinkDeviceCode(req.SessionToken, deviceResp.DeviceCode)

	// Mark session as verified
	s.voiceRepo.VerifyVoiceSession(req.SessionToken, nil, "")

	return &models.VoiceVerifyResponse{
		Success:         true,
		Status:          "verified",
		Message:         fmt.Sprintf("Voice verified. Complete authentication at %s with code %s", deviceResp.VerificationURI, deviceResp.UserCode),
		DeviceCode:      deviceResp.DeviceCode,
		UserCode:        deviceResp.UserCode,
		VerificationURI: deviceResp.VerificationURI,
	}, nil
}

// AuthenticateWithCredentials authenticates user with email/password via voice
// WARNING: Less secure - credentials spoken aloud
func (s *VoiceAuthService) AuthenticateWithCredentials(req *models.VoiceTokenRequest) (*models.VoiceTokenResponse, error) {
	// Find voice session
	session, err := s.voiceRepo.FindVoiceSessionByToken(req.SessionToken)
	if err != nil {
		return &models.VoiceTokenResponse{
			Error:            "invalid_request",
			ErrorDescription: "Invalid session token",
		}, nil
	}

	// Check if expired
	if session.IsExpired() {
		return &models.VoiceTokenResponse{
			Error:            "expired_token",
			ErrorDescription: "Voice session expired",
		}, nil
	}

	// Validate tenant
	tenant, err := s.tenantRepo.GetTenantByDomain(req.TenantDomain)
	if err != nil {
		return &models.VoiceTokenResponse{
			Error:            "invalid_request",
			ErrorDescription: "Tenant not found",
		}, nil
	}

	// Find user by email and tenant
	user, err := s.userRepo.GetUserByEmailAndTenant(req.Email, tenant.ID)
	if err != nil {
		return &models.VoiceTokenResponse{
			Error:            "invalid_grant",
			ErrorDescription: "Invalid credentials",
		}, nil
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return &models.VoiceTokenResponse{
			Error:            "invalid_grant",
			ErrorDescription: "Invalid credentials",
		}, nil
	}

	// Generate token with session tracking
	token, err := s.generateJWTTokenWithSession(
		user,
		tenant,
		session.Scopes,
		session.VoicePlatform,
		session.VoiceUserID,
		session.DeviceInfo,
	)
	if err != nil {
		return &models.VoiceTokenResponse{
			Error:            "server_error",
			ErrorDescription: "Failed to generate token",
		}, nil
	}

	// Mark session as verified
	s.voiceRepo.VerifyVoiceSession(req.SessionToken, &user.ID, user.Email)

	return &models.VoiceTokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   86400, // 24 hours
	}, nil
}

// LinkVoiceIdentity links a voice assistant to a user account
func (s *VoiceAuthService) LinkVoiceIdentity(tenantID uuid.UUID, userID uuid.UUID, userEmail string, req *models.VoiceLinkRequest) (*models.VoiceLinkResponse, error) {
	// Voice platform accepts any string for flexibility (alexa, google, siri, web, custom, etc.)
	if req.VoicePlatform == "" {
		return &models.VoiceLinkResponse{
			Success: false,
			Message: "Voice platform is required",
		}, nil
	}

	// Check if already linked
	existing, err := s.voiceRepo.FindVoiceIdentityLink(tenantID, req.VoicePlatform, req.VoiceUserID)
	if err == nil && existing.IsActive {
		return &models.VoiceLinkResponse{
			Success: false,
			Message: "Voice assistant already linked to this account",
			LinkID:  existing.ID.String(),
		}, nil
	}

	// Create link
	link := &models.VoiceIdentityLink{
		ID:            uuid.New(),
		TenantID:      tenantID,
		VoicePlatform: req.VoicePlatform,
		VoiceUserID:   req.VoiceUserID,
		VoiceUserName: req.VoiceUserName,
		UserID:        userID,
		UserEmail:     userEmail,
		IsActive:      true,
		LinkMethod:    req.LinkMethod,
	}

	if err := s.voiceRepo.CreateVoiceIdentityLink(link); err != nil {
		return &models.VoiceLinkResponse{
			Success: false,
			Message: "Failed to link voice assistant",
		}, nil
	}

	return &models.VoiceLinkResponse{
		Success: true,
		Message: fmt.Sprintf("Voice assistant (%s) linked successfully", req.VoicePlatform),
		LinkID:  link.ID.String(),
	}, nil
}

// UnlinkVoiceIdentity removes a voice assistant link
func (s *VoiceAuthService) UnlinkVoiceIdentity(tenantID uuid.UUID, req *models.VoiceUnlinkRequest) (*models.VoiceUnlinkResponse, error) {
	err := s.voiceRepo.DeactivateVoiceIdentityLink(tenantID, req.VoicePlatform, req.VoiceUserID)
	if err != nil {
		return &models.VoiceUnlinkResponse{
			Success: false,
			Message: "Voice assistant link not found",
		}, nil
	}

	return &models.VoiceUnlinkResponse{
		Success: true,
		Message: "Voice assistant unlinked successfully",
	}, nil
}

// ListVoiceLinks lists all voice assistant links for a user
func (s *VoiceAuthService) ListVoiceLinks(tenantID uuid.UUID, userID uuid.UUID) (*models.VoiceLinksListResponse, error) {
	links, err := s.voiceRepo.ListVoiceIdentityLinks(tenantID, userID)
	if err != nil {
		return &models.VoiceLinksListResponse{Links: []models.VoiceIdentityLinkPublic{}}, nil
	}

	var publicLinks []models.VoiceIdentityLinkPublic
	for _, link := range links {
		publicLinks = append(publicLinks, models.VoiceIdentityLinkPublic{
			ID:            link.ID.String(),
			VoicePlatform: link.VoicePlatform,
			VoiceUserName: link.VoiceUserName,
			IsActive:      link.IsActive,
			LastUsedAt:    link.LastUsedAt,
			LinkedAt:      link.LinkedAt,
		})
	}

	return &models.VoiceLinksListResponse{Links: publicLinks}, nil
}

// generateJWTToken generates a JWT token using THE SAME logic as enduser login
// This EXACTLY mirrors the token generation in controllers/enduser_auth_controller.go
func (s *VoiceAuthService) generateJWTToken(user *models.ExtendedUser, tenant *models.Tenant, scopes []string) (string, error) {
	// Use centralized auth-manager token service
	return config.TokenService.GenerateVoiceAuthToken(
		user.ID,
		tenant.ID,
		user.Email,
		scopes,
		24*time.Hour,
	)
}

// generateJWTTokenWithSession generates a JWT token with additional session tracking claims
func (s *VoiceAuthService) generateJWTTokenWithSession(
	user *models.ExtendedUser,
	tenant *models.Tenant,
	scopes []string,
	voicePlatform string,
	voiceUserID string,
	deviceInfo map[string]interface{},
) (string, error) {
	jwtSecret := os.Getenv("JWT_DEF_SECRET")
	if jwtSecret == "" {
		panic("CRITICAL: JWT_DEF_SECRET environment variable is not set. Cannot generate secure tokens.")
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"tenant_id":      tenant.ID.String(),
		"project_id":     tenant.ID.String(),
		"client_id":      user.ClientID.String(),
		"email":          user.Email,
		"sub":            user.ID.String(),
		"aud":            "authsec-api",
		"iss":            "authsec-ai/auth-manager",
		"iat":            now.Unix(),
		"exp":            now.Add(24 * time.Hour).Unix(),
		"scopes":         scopes,
		"voice_platform": voicePlatform,
		"voice_user_id":  voiceUserID,
		"device_info":    deviceInfo,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "default"

	tokenString, err := token.SignedString([]byte(jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// formatOTPForVoice formats OTP for voice output
// "8532" -> "Your code is eight, five, three, two"
func (s *VoiceAuthService) formatOTPForVoice(otp string) string {
	digits := []string{"zero", "one", "two", "three", "four", "five", "six", "seven", "eight", "nine"}
	var words []string

	for _, char := range otp {
		digit := int(char - '0')
		if digit >= 0 && digit <= 9 {
			words = append(words, digits[digit])
		}
	}

	return fmt.Sprintf("Your authentication code is: %s. Say 'confirm' to proceed, or visit the activation page to complete authentication.", strings.Join(words, ", "))
}

// CleanupExpiredSessions runs periodic cleanup of expired voice sessions
func (s *VoiceAuthService) CleanupExpiredSessions() (int64, error) {
	// Mark expired sessions
	expired, err := s.voiceRepo.ExpireOldVoiceSessions()
	if err != nil {
		return 0, fmt.Errorf("failed to expire old sessions: %w", err)
	}

	// Delete old expired/failed/verified sessions (older than 1 hour)
	deleted, err := s.voiceRepo.DeleteExpiredVoiceSessions(1 * time.Hour)
	if err != nil {
		return expired, fmt.Errorf("failed to delete expired sessions: %w", err)
	}

	return expired + deleted, nil
}
