package enduser

import (
	"net/http"
	"strings"

	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantTOTPController handles TOTP authentication for tenant users
type TenantTOTPController struct {
	totpService *services.TenantTOTPService
}

// NewTenantTOTPController creates a new tenant TOTP controller
func NewTenantTOTPController() *TenantTOTPController {
	return &TenantTOTPController{
		totpService: services.NewTenantTOTPService(),
	}
}

// LoginWithTenantTOTP handles TOTP-only login for tenant users
// @Summary Login with TOTP for tenant users
// @Description Authenticates tenant users using TOTP codes (no password required)
// @Tags Tenant TOTP Authentication
// @Accept json
// @Produce json
// @Param request body models.TenantTOTPLoginRequest true "TOTP login request"
// @Success 200 {object} models.TenantTOTPLoginResponse "Login successful"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/totp/login [post]
func (ttc *TenantTOTPController) LoginWithTenantTOTP(c *gin.Context) {
	var req models.TenantTOTPLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.ClientID == "" || req.Email == "" || req.TOTPCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id, email, and totp_code are required"})
		return
	}

	// Normalize email
	req.Email = strings.ToLower(req.Email)

	// Validate TOTP code length
	if len(req.TOTPCode) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP code must be exactly 6 digits"})
		return
	}

	// Call TOTP service
	response, err := ttc.totpService.LoginWithTenantTOTP(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusUnauthorized, response)
	}
}

// RegisterTenantTOTPDevice registers a new TOTP device for tenant users
// @Summary Register TOTP device for tenant users
// @Description Registers a new TOTP device and returns QR code for setup
// @Tags Tenant TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TenantTOTPRegistrationRequest true "TOTP device registration request"
// @Success 200 {object} models.TenantTOTPRegistrationResponse "Device registered successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/totp/register [post]
func (ttc *TenantTOTPController) RegisterTenantTOTPDevice(c *gin.Context) {
	var req models.TenantTOTPRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.DeviceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_name is required"})
		return
	}

	// Set default device type if not provided
	if req.DeviceType == "" {
		req.DeviceType = "generic"
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

	emailStr, exists := c.Get("email")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email not found in token"})
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

	email := emailStr.(string)

	// Call TOTP service
	response, err := ttc.totpService.RegisterTenantTOTPDevice(&req, userID, tenantID, email)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ConfirmTenantTOTPDevice confirms TOTP device registration for tenant users
// @Summary Confirm TOTP device registration
// @Description Confirms TOTP device registration by validating the first TOTP code
// @Tags Tenant TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TenantTOTPRegistrationConfirmRequest true "TOTP device confirmation request"
// @Success 200 {object} models.TenantTOTPRegistrationConfirmResponse "Device confirmed successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/totp/confirm [post]
func (ttc *TenantTOTPController) ConfirmTenantTOTPDevice(c *gin.Context) {
	var req models.TenantTOTPRegistrationConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.DeviceID == "" || req.TOTPCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id and totp_code are required"})
		return
	}

	// Validate TOTP code length
	if len(req.TOTPCode) != 6 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP code must be exactly 6 digits"})
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

	// Call TOTP service
	response, err := ttc.totpService.ConfirmTenantTOTPDevice(&req, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetTenantTOTPDevices retrieves all TOTP devices for tenant user
// @Summary Get TOTP devices for tenant users
// @Description Retrieves all registered TOTP devices for the authenticated tenant user
// @Tags Tenant TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.TenantTOTPDeviceListResponse "Devices retrieved successfully"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/totp/devices [get]
func (ttc *TenantTOTPController) GetTenantTOTPDevices(c *gin.Context) {
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

	// Call TOTP service
	response, err := ttc.totpService.GetTenantTOTPDevices(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// DeleteTenantTOTPDevice deletes a TOTP device for tenant user
// @Summary Delete TOTP device for tenant users
// @Description Deletes a registered TOTP device for the authenticated tenant user
// @Tags Tenant TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TenantTOTPDeviceDeleteRequest true "TOTP device delete request"
// @Success 200 {object} models.TenantTOTPDeviceDeleteResponse "Device deleted successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/totp/devices/delete [post]
func (ttc *TenantTOTPController) DeleteTenantTOTPDevice(c *gin.Context) {
	var req models.TenantTOTPDeviceDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
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

	// Call TOTP service
	response, err := ttc.totpService.DeleteTenantTOTPDevice(&req, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// SetTenantPrimaryTOTPDevice sets a TOTP device as primary for tenant user
// @Summary Set primary TOTP device for tenant users
// @Description Sets a TOTP device as the primary device for the authenticated tenant user
// @Tags Tenant TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TenantTOTPDeviceDeleteRequest true "Set primary device request"
// @Success 200 {object} models.TenantTOTPDeviceDeleteResponse "Device set as primary successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/tenant/totp/devices/primary [post]
func (ttc *TenantTOTPController) SetTenantPrimaryTOTPDevice(c *gin.Context) {
	var req models.TenantTOTPDeviceDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate required fields
	if req.DeviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
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

	// Call TOTP service
	response, err := ttc.totpService.SetTenantPrimaryTOTPDevice(&req, userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}
