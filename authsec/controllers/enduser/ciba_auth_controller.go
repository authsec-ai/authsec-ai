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

// CIBAAuthController handles CIBA (Client Initiated Backchannel Authentication) endpoints
// This is like DeviceAuthController but uses push notifications instead of device codes
type CIBAAuthController struct {
	cibaService *services.CIBAAuthService
}

// NewCIBAAuthController creates a new CIBA authentication controller
func NewCIBAAuthController() (*CIBAAuthController, error) {
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

	// Initialize push notification service (optional - if not configured, will return error in response)
	pushService, err := services.NewPushNotificationService()
	if err != nil {
		fmt.Printf("Warning: Push notification service not initialized: %v\n", err)
		pushService = nil // Continue without push service
	}

	return &CIBAAuthController{
		cibaService: services.NewCIBAAuthService(db, tenantDBService, pushService),
	}, nil
}

// InitiateCIBAAuth initiates CIBA authentication flow
// @Summary Initiate CIBA authentication
// @Description Initiates CIBA flow by looking up user by email and sending push notification to their registered device
// @Tags CIBA Authentication
// @Accept json
// @Produce json
// @Param request body models.CIBAInitiateRequest true "CIBA initiate request"
// @Success 200 {object} models.CIBAInitiateResponse "CIBA request created, push notification sent"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/ciba/initiate [post]
func (ctrl *CIBAAuthController) InitiateCIBAAuth(c *gin.Context) {
	var req models.CIBAInitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Initiate CIBA flow
	resp, err := ctrl.cibaService.InitiateCIBAAuth(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to initiate CIBA", "details": err.Error()})
		return
	}

	// Check if there was an error in response (user not found, no device, etc.)
	if resp.Error != "" {
		statusCode := http.StatusOK // RFC 8628 style - return 200 with error field
		if resp.Error == models.CIBAErrorUserNotFound {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, resp)
		return
	}

	// Audit log
	middlewares.Audit(c, "ciba_auth", resp.AuthReqID, "initiate", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"login_hint":      req.LoginHint,
			"binding_message": req.BindingMessage,
			"auth_req_id":     resp.AuthReqID,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// RespondToCIBA handles user's approve/deny response from mobile app
// @Summary Respond to CIBA request
// @Description User approves or denies the CIBA authentication request from mobile app. Requires JWT token.
// @Tags CIBA Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param request body models.CIBARespondRequest true "CIBA respond request"
// @Success 200 {object} models.CIBARespondResponse "Response recorded"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/ciba/respond [post]
func (ctrl *CIBAAuthController) RespondToCIBA(c *gin.Context) {
	var req models.CIBARespondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Respond to CIBA request
	resp, err := ctrl.cibaService.RespondToCIBA(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to respond", "details": err.Error()})
		return
	}

	// Audit log
	action := "approve"
	if !req.Approved {
		action = "deny"
	}
	middlewares.Audit(c, "ciba_auth", req.AuthReqID, action, &middlewares.AuditChanges{
		After: map[string]interface{}{
			"auth_req_id":        req.AuthReqID,
			"approved":           req.Approved,
			"biometric_verified": req.BiometricVerified,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// PollCIBAToken polls for CIBA authentication status
// @Summary Poll for CIBA token
// @Description Voice agent polls this endpoint to check if user has approved. Returns token when approved.
// @Tags CIBA Authentication
// @Accept json
// @Produce json
// @Param request body models.CIBATokenRequest true "CIBA token request"
// @Success 200 {object} models.CIBATokenResponse "Token issued or status returned"
// @Failure 400 {object} map[string]string "Bad request"
// @Router /uflow/auth/ciba/token [post]
func (ctrl *CIBAAuthController) PollCIBAToken(c *gin.Context) {
	var req models.CIBATokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Poll for token
	resp, err := ctrl.cibaService.PollForToken(req.AuthReqID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to poll token", "details": err.Error()})
		return
	}

	// Return appropriate status code based on error
	if resp.Error != "" {
		statusCode := http.StatusOK
		if resp.Error == models.CIBAErrorAccessDenied {
			statusCode = http.StatusForbidden
		} else if resp.Error == models.CIBAErrorExpiredToken {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// RegisterDevice registers a new device for push notifications
// @Summary Register device for push notifications
// @Description Mobile app registers device token (Expo Push Token) for push notifications. Requires JWT token.
// @Tags CIBA Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param request body models.DeviceTokenRegistrationRequest true "Device registration request"
// @Success 200 {object} models.DeviceTokenRegistrationResponse "Device registered"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Router /uflow/auth/ciba/register-device [post]
func (ctrl *CIBAAuthController) RegisterDevice(c *gin.Context) {
	var req models.DeviceTokenRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user ID from JWT token
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

	// Get tenant_id from token
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant_id in token"})
		return
	}

	// Register device
	resp, err := ctrl.cibaService.RegisterDevice(userID, tenantID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device", "details": err.Error()})
		return
	}

	// Audit log
	middlewares.Audit(c, "ciba_device", resp.DeviceID, "register", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_id":     userID.String(),
			"tenant_id":   tenantID.String(),
			"platform":    req.Platform,
			"device_name": req.DeviceName,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// GetDevices retrieves registered push notification devices for the authenticated user
// @Summary Get registered CIBA push devices
// @Description Retrieves all active push notification devices registered by the authenticated user
// @Tags CIBA Authentication
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Success 200 {object} models.DeviceListResponse "Devices retrieved successfully"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/ciba/devices [get]
func (ctrl *CIBAAuthController) GetDevices(c *gin.Context) {
	// Get user ID from JWT token
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

	// Get tenant_id from token
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant_id in token"})
		return
	}

	// Get devices
	devices, err := ctrl.cibaService.GetUserDevices(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve devices", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.DeviceListResponse{
		Success: true,
		Devices: devices,
		Message: "Devices retrieved successfully",
	})
}

// DeleteDevice deactivates a registered push notification device
// @Summary Delete registered CIBA push device
// @Description Deactivates (soft deletes) a specific push notification device
// @Tags CIBA Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param device_id path string true "Device ID"
// @Success 200 {object} models.DeviceDeleteResponse "Device deleted successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid device ID"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 404 {object} map[string]string "Device not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/ciba/devices/{device_id} [delete]
func (ctrl *CIBAAuthController) DeleteDevice(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID format"})
		return
	}

	// Get user ID from JWT token
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

	// Get tenant_id from token
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "tenant_id not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant_id in token"})
		return
	}

	// Delete device
	if err := ctrl.cibaService.DeleteDevice(deviceID, userID, tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to delete device", "details": err.Error()})
		return
	}

	// Audit log
	middlewares.Audit(c, "ciba_device", deviceIDStr, "delete", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"device_id": deviceIDStr,
			"user_id":   userID.String(),
			"tenant_id": tenantID.String(),
		},
	})

	c.JSON(http.StatusOK, models.DeviceDeleteResponse{
		Success: true,
		Message: "Device deactivated successfully",
	})
}
