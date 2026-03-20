package enduser

import (
	"fmt"
	"net/http"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VoiceAuthController handles voice authentication endpoints
type VoiceAuthController struct {
	voiceService  *services.VoiceAuthService
	deviceService *services.DeviceAuthService
	deviceRepo    *database.DeviceAuthRepository
}

// NewVoiceAuthController creates a new voice authentication controller
func NewVoiceAuthController() (*VoiceAuthController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	tenantDBService, err := database.NewTenantDBService(
		db,
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBPort,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant DB service: %w", err)
	}

	deviceService := services.NewDeviceAuthService(db, tenantDBService)
	deviceRepo := database.NewDeviceAuthRepository(db)

	return &VoiceAuthController{
		voiceService:  services.NewVoiceAuthService(db, tenantDBService, deviceService),
		deviceService: deviceService,
		deviceRepo:    deviceRepo,
	}, nil
}

// InitiateVoiceAuth initiates a voice authentication session
// @Summary Initiate voice authentication
// @Description Creates a voice authentication session and returns a session token and voice OTP for verification
// @Tags Voice Authentication
// @Accept json
// @Produce json
// @Param request body models.VoiceInitiateRequest true "Voice authentication initiation request"
// @Success 200 {object} models.VoiceInitiateResponse "Voice session created successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/voice/initiate [post]
func (ctrl *VoiceAuthController) InitiateVoiceAuth(c *gin.Context) {
	var req models.VoiceInitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	resp, err := ctrl.voiceService.InitiateVoiceAuth(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate voice authentication", "details": err.Error()})
		return
	}

	// Audit log: Voice authentication initiated
	middlewares.Audit(c, "voice_auth", resp.SessionToken, "initiate", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"client_id":      req.ClientID,
			"voice_platform": req.VoicePlatform,
			"session_token":  resp.SessionToken,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// VerifyVoiceOTP verifies the voice OTP and completes authentication
// @Summary Verify voice OTP
// @Description Verifies the voice OTP. Returns either a device code for browser completion or an access token if user is pre-linked.
// @Tags Voice Authentication
// @Accept json
// @Produce json
// @Param request body models.VoiceVerifyRequest true "Voice OTP verification request"
// @Success 200 {object} models.VoiceVerifyResponse "Voice OTP verified"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/voice/verify [post]
func (ctrl *VoiceAuthController) VerifyVoiceOTP(c *gin.Context) {
	var req models.VoiceVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	resp, err := ctrl.voiceService.VerifyVoiceOTP(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify voice OTP", "details": err.Error()})
		return
	}

	// Audit log: Voice OTP verified
	middlewares.Audit(c, "voice_auth", req.SessionToken, "verify_otp", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"session_token": req.SessionToken,
			"status":        resp.Status,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// GetTokenWithCredentials authenticates user with email/password via voice
// @Summary Get token with voice credentials (LESS SECURE)
// @Description Authenticates user with email/password spoken via voice assistant. WARNING: Credentials are spoken aloud.
// @Tags Voice Authentication
// @Accept json
// @Produce json
// @Param request body models.VoiceTokenRequest true "Voice token request with credentials"
// @Success 200 {object} models.VoiceTokenResponse "Token issued successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized - invalid credentials"
// @Router /uflow/auth/voice/token [post]
func (ctrl *VoiceAuthController) GetTokenWithCredentials(c *gin.Context) {
	var req models.VoiceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	resp, err := ctrl.voiceService.AuthenticateWithCredentials(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate", "details": err.Error()})
		return
	}

	if resp.Error != "" {
		statusCode := http.StatusUnauthorized
		if resp.Error == "invalid_grant" {
			statusCode = http.StatusUnauthorized
		}
		c.JSON(statusCode, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// LinkVoiceAssistant links a voice assistant to the authenticated user's account
// @Summary Link voice assistant to account
// @Description Links a voice assistant (Alexa, Google, Siri) to the authenticated user's account for passwordless authentication
// @Tags Voice Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param request body models.VoiceLinkRequest true "Voice link request"
// @Success 200 {object} models.VoiceLinkResponse "Voice assistant linked successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/voice/link [post]
func (ctrl *VoiceAuthController) LinkVoiceAssistant(c *gin.Context) {
	var req models.VoiceLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Extract user info from JWT token
	// Use ResolveUserID which handles both 'sub' and 'user_id' claims, and falls back to email lookup
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	// Extract tenant ID
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID missing in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Get user email (try email first, then email_id for backward compatibility)
	email := ""
	if emailVal, exists := c.Get("email"); exists && emailVal != nil {
		email = emailVal.(string)
	} else if emailID, exists := c.Get("email_id"); exists && emailID != nil {
		email = emailID.(string)
	}

	// Link voice assistant
	resp, err := ctrl.voiceService.LinkVoiceIdentity(tenantID, userID, email, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link voice assistant", "details": err.Error()})
		return
	}

	// Audit log: Voice assistant linked
	middlewares.Audit(c, "voice_auth", userID.String(), "link", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":        userID.String(),
			"tenant_id":      tenantID.String(),
			"voice_platform": req.VoicePlatform,
			"voice_user_id":  req.VoiceUserID,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// UnlinkVoiceAssistant removes a voice assistant link from the user's account
// @Summary Unlink voice assistant from account
// @Description Removes a voice assistant link from the authenticated user's account
// @Tags Voice Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param request body models.VoiceUnlinkRequest true "Voice unlink request"
// @Success 200 {object} models.VoiceUnlinkResponse "Voice assistant unlinked successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/voice/unlink [post]
func (ctrl *VoiceAuthController) UnlinkVoiceAssistant(c *gin.Context) {
	var req models.VoiceUnlinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Extract tenant ID from JWT token
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID missing in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Unlink voice assistant
	resp, err := ctrl.voiceService.UnlinkVoiceIdentity(tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlink voice assistant", "details": err.Error()})
		return
	}

	// Audit log: Voice assistant unlinked
	middlewares.Audit(c, "voice_auth", req.VoiceUserID, "unlink", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"tenant_id":      tenantID.String(),
			"voice_platform": req.VoicePlatform,
			"voice_user_id":  req.VoiceUserID,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// ListVoiceLinks lists all voice assistant links for the authenticated user
// @Summary List voice assistant links
// @Description Returns all voice assistant links for the authenticated user
// @Tags Voice Authentication
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {object} models.VoiceLinksListResponse "List of voice links"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/voice/links [get]
func (ctrl *VoiceAuthController) ListVoiceLinks(c *gin.Context) {
	// Extract user info from JWT token
	// Use ResolveUserID which handles both 'sub' and 'user_id' claims, and falls back to email lookup
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	// Extract tenant ID
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID missing in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// List voice links
	resp, err := ctrl.voiceService.ListVoiceLinks(tenantID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list voice links", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ========================================
// Device Code Pending Requests (uses existing device_codes table)
// ========================================

// GetPendingDeviceCodes returns pending device authorization requests for a tenant
// @Summary Get pending device auth requests
// @Description Returns all pending device authorization requests awaiting user approval
// @Tags Voice Authentication
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param client_id query string false "Optional client_id to filter by"
// @Success 200 {object} map[string]interface{} "List of pending device codes"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/voice/device-pending [get]
func (ctrl *VoiceAuthController) GetPendingDeviceCodes(c *gin.Context) {
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID missing in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Optional client_id filter
	var clientID *uuid.UUID
	if clientIDStr := c.Query("client_id"); clientIDStr != "" {
		cid, err := uuid.Parse(clientIDStr)
		if err == nil {
			clientID = &cid
		}
	}
	if clientIDStr := c.Query("client_id"); clientIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Get pending device codes from device_codes table
	codes, err := ctrl.deviceRepo.ListPendingDeviceCodes(tenantID, clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get pending codes", "details": err.Error()})
		return
	}

	// Convert to response format
	var pending []map[string]interface{}
	for _, code := range codes {
		item := map[string]interface{}{
			"id":               code.ID.String(),
			"user_code":        code.UserCode,
			"device_code":      code.DeviceCode,
			"verification_uri": code.VerificationURI,
			"device_info":      code.DeviceInfo,
			"scopes":           code.Scopes,
			"status":           code.Status,
			"expires_at":       code.ExpiresAt,
			"created_at":       code.CreatedAt,
		}
		if code.ClientID != nil {
			item["client_id"] = code.ClientID.String()
		}
		pending = append(pending, item)
	}

	c.JSON(http.StatusOK, gin.H{
		"requests": pending,
		"count":    len(pending),
	})
}

// ApproveDeviceCode approves or denies a pending device code
// @Summary Approve/deny device auth request
// @Description Approves or denies a pending device authorization request using the user_code
// @Tags Voice Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param request body map[string]interface{} true "Approval request with user_code and approve fields"
// @Success 200 {object} map[string]interface{} "Approval result"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/voice/device-approve [post]
func (ctrl *VoiceAuthController) ApproveDeviceCode(c *gin.Context) {
	var req struct {
		UserCode string `json:"user_code" binding:"required"`
		Approve  bool   `json:"approve"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user info from token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	// Get user email (try email first, then email_id for backward compatibility)
	email := ""
	if emailVal, exists := c.Get("email"); exists && emailVal != nil {
		email = emailVal.(string)
	} else if emailID, exists := c.Get("email_id"); exists && emailID != nil {
		email = emailID.(string)
	}

	// Use existing device service to verify (approve/deny)
	err = ctrl.deviceService.VerifyDeviceCode(req.UserCode, userID, email, req.Approve)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": err.Error(),
			"status":  "failed",
		})
		return
	}

	status := "denied"
	message := "Device authorization denied"
	if req.Approve {
		status = "authorized"
		message = "Device authorization approved"
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": message,
		"status":  status,
	})
}
