package enduser

import (
	"net/http"
	"strings"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantCIBAController handles CIBA authentication for tenant users
type TenantCIBAController struct {
	cibaService *services.TenantCIBAAuthService
}

// NewTenantCIBAController creates a new tenant CIBA controller
func NewTenantCIBAController() (*TenantCIBAController, error) {
	// Initialize push notification service (optional)
	pushService, err := services.NewPushNotificationService()
	if err != nil {
		// Continue without push service - will show warning in service
		pushService = nil
	}

	return &TenantCIBAController{
		cibaService: services.NewTenantCIBAAuthService(pushService),
	}, nil
}

// InitiateTenantCIBA initiates CIBA authentication for tenant users
// @Summary Initiate CIBA authentication for tenant users
// @Description Initiates CIBA (push notification) authentication flow for tenant users
// @Tags Tenant CIBA Authentication
// @Accept json
// @Produce json
// @Param request body models.TenantCIBAInitiateRequest true "CIBA initiation request"
// @Success 200 {object} models.TenantCIBAInitiateResponse "CIBA initiation successful"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/initiate [post]
func (tcc *TenantCIBAController) InitiateTenantCIBA(c *gin.Context) {
	var req models.TenantCIBAInitiateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.ClientID == "" || req.Email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id and email are required"})
		return
	}

	// Normalize email
	req.Email = strings.ToLower(req.Email)

	// Call CIBA service
	response, err := tcc.cibaService.InitiateTenantCIBAAuth(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response
	if response.Error != "" {
		c.JSON(http.StatusBadRequest, response)
	} else {
		c.JSON(http.StatusOK, response)
	}
}

// RespondToTenantCIBA handles user response to CIBA authentication request
// @Summary Respond to CIBA authentication request
// @Description User approves or denies CIBA authentication request from mobile app
// @Tags Tenant CIBA Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TenantCIBARespondRequest true "CIBA response request"
// @Success 200 {object} models.TenantCIBARespondResponse "CIBA response processed"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/respond [post]
func (tcc *TenantCIBAController) RespondToTenantCIBA(c *gin.Context) {
	var req models.TenantCIBARespondRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user and tenant info from JWT token
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Call CIBA service
	response, err := tcc.cibaService.RespondToTenantCIBA(req.AuthReqID, req.Approved, req.BiometricVerified, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// PollTenantCIBAToken polls for CIBA authentication token
// @Summary Poll for CIBA authentication token
// @Description Polls for completion of CIBA authentication and returns token if approved
// @Tags Tenant CIBA Authentication
// @Accept json
// @Produce json
// @Param request body models.TenantCIBATokenRequest true "CIBA token request"
// @Success 200 {object} models.TenantCIBATokenResponse "Token response"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/token [post]
func (tcc *TenantCIBAController) PollTenantCIBAToken(c *gin.Context) {
	var req models.TenantCIBATokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.AuthReqID == "" || req.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "auth_req_id and client_id are required"})
		return
	}

	// Call CIBA service
	response, err := tcc.cibaService.PollTenantCIBAToken(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Return response with appropriate status code
	if response.Error != "" {
		switch response.Error {
		case models.TenantCIBAErrorAuthorizationPending:
			c.JSON(http.StatusAccepted, response) // 202 - Still pending
		case models.TenantCIBAErrorAccessDenied, models.TenantCIBAErrorExpiredToken:
			c.JSON(http.StatusBadRequest, response) // 400 - Failed
		default:
			c.JSON(http.StatusBadRequest, response) // 400 - Other errors
		}
	} else {
		c.JSON(http.StatusOK, response) // 200 - Success
	}
}

// RegisterTenantDevice registers a device for push notifications in tenant context
// @Summary Register device for tenant push notifications
// @Description Registers a mobile device for receiving push notifications in tenant context
// @Tags Tenant CIBA Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TenantDeviceTokenRegistrationRequest true "Device registration request"
// @Success 200 {object} models.TenantDeviceTokenRegistrationResponse "Device registered"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/register-device [post]
func (tcc *TenantCIBAController) RegisterTenantDevice(c *gin.Context) {
	var req models.TenantDeviceTokenRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.DeviceToken == "" || req.Platform == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_token and platform are required"})
		return
	}

	// Validate platform
	if req.Platform != "ios" && req.Platform != "android" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "platform must be 'ios' or 'android'"})
		return
	}

	// Get user and tenant info from JWT token
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Call CIBA service
	response, err := tcc.cibaService.RegisterTenantDevice(&req, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetTenantCIBARequests gets pending CIBA requests for the authenticated user
// @Summary Get pending CIBA requests
// @Description Retrieves all pending CIBA authentication requests for the authenticated tenant user
// @Tags Tenant CIBA Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]interface{} "Pending requests retrieved"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/requests [get]
func (tcc *TenantCIBAController) GetTenantCIBARequests(c *gin.Context) {
	// Get user and tenant info from JWT token
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Connect to tenant database
	tenantIDStr2 := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	// Get pending requests
	tenantRepo := database.NewTenantDeviceRepository(tenantDB)
	requests, err := tenantRepo.GetPendingTenantCIBAAuthRequests(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve pending requests"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"requests": requests,
	})
}

// ListTenantDevices lists registered push devices for authenticated tenant user
// @Summary List registered CIBA push devices
// @Description Retrieves all active push notification devices registered by the authenticated user
// @Tags Tenant CIBA Authentication
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.TenantDeviceListResponse "Devices retrieved successfully"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/devices [get]
func (tcc *TenantCIBAController) ListTenantDevices(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	tenantIDStr2 := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)
	devices, err := tenantRepo.GetTenantDeviceTokensByUserID(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retrieve devices"})
		return
	}

	// Convert to summaries (omit device_token)
	summaries := make([]models.TenantDeviceSummary, len(devices))
	for i, device := range devices {
		summaries[i] = models.TenantDeviceSummary{
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

	c.JSON(http.StatusOK, models.TenantDeviceListResponse{
		Success: true,
		Devices: summaries,
		Message: "Devices retrieved successfully",
	})
}

// DeleteTenantDevice deactivates a registered push device
// @Summary Delete registered CIBA push device
// @Description Deactivates (soft deletes) a specific push notification device
// @Tags Tenant CIBA Authentication
// @Produce json
// @Security BearerAuth
// @Param device_id path string true "Device ID"
// @Success 200 {object} models.TenantDeviceDeleteResponse "Device deleted successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid device ID"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 404 {object} map[string]string "Device not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/ciba/devices/{device_id} [delete]
func (tcc *TenantCIBAController) DeleteTenantDevice(c *gin.Context) {
	deviceIDStr := c.Param("device_id")
	deviceID, err := uuid.Parse(deviceIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID format"})
		return
	}

	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in token"})
		return
	}

	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	tenantIDStr2 := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr2)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	tenantRepo := database.NewTenantDeviceRepository(tenantDB)
	err = tenantRepo.DeactivateTenantDeviceToken(deviceID, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to deactivate device"})
		return
	}

	c.JSON(http.StatusOK, models.TenantDeviceDeleteResponse{
		Success: true,
		Message: "Device deactivated successfully",
	})
}
