package enduser

import (
	"fmt"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"

	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TOTPController handles TOTP authentication endpoints
type TOTPController struct {
	totpService *services.TOTPService
	userRepo    *database.UserRepository
	tenantRepo  *database.AdminTenantRepository
	deviceRepo  *database.DeviceAuthRepository
}

// NewTOTPController creates a new TOTP controller
func NewTOTPController() (*TOTPController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	return &TOTPController{
		totpService: services.NewTOTPService(database.NewTOTPRepository(db.DB)),
		userRepo:    database.NewUserRepository(db),
		tenantRepo:  database.NewAdminTenantRepository(db),
		deviceRepo:  database.NewDeviceAuthRepository(db),
	}, nil
}

// RegisterDevice initiates TOTP device registration
// @Summary Register TOTP device
// @Description Registers a new TOTP authenticator device. Returns QR code and secret for scanning.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TOTPRegistrationRequest true "Device registration request"
// @Success 200 {object} models.TOTPRegistrationResponse "QR code and secret returned"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/register [post]
func (ctrl *TOTPController) RegisterDevice(c *gin.Context) {
	var req models.TOTPRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Get email from token
	email := ""
	if emailVal, exists := c.Get("email"); exists && emailVal != nil {
		email = emailVal.(string)
	}

	// Set device type default
	deviceType := req.DeviceType
	if deviceType == "" {
		deviceType = "generic"
	}

	// Register device
	resp, err := ctrl.totpService.RegisterDevice(userID, tenantID, email, req.DeviceName, deviceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register device", "details": err.Error()})
		return
	}

	// Audit log: Device registered
	middlewares.Audit(c, "totp", resp.DeviceID, "register_device", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"device_name": req.DeviceName,
			"device_type": deviceType,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// ConfirmRegistration confirms TOTP device registration
// @Summary Confirm TOTP device registration
// @Description Confirms device registration by validating the TOTP code from the authenticator app.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TOTPRegistrationConfirmRequest true "Confirmation request"
// @Success 200 {object} models.TOTPRegistrationConfirmResponse "Device registered successfully"
// @Failure 400 {object} map[string]string "Bad request or invalid code"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/confirm [post]
func (ctrl *TOTPController) ConfirmRegistration(c *gin.Context) {
	var req models.TOTPRegistrationConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Parse device ID
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	// Confirm registration
	resp, err := ctrl.totpService.ConfirmRegistration(deviceID, userID, tenantID, req.TOTPCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to confirm registration", "details": err.Error()})
		return
	}

	// Audit log: Registration confirmed
	middlewares.Audit(c, "totp", req.DeviceID, "confirm_registration", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"device_id": req.DeviceID,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// VerifyTOTP validates TOTP code during login
// @Summary Verify TOTP code
// @Description Validates TOTP code for authentication. Returns JWT token if valid.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TOTPVerificationRequest true "TOTP verification request"
// @Success 200 {object} models.TOTPVerificationResponse "Verification successful"
// @Failure 400 {object} map[string]string "Bad request or invalid code"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/verify [post]
func (ctrl *TOTPController) VerifyTOTP(c *gin.Context) {
	var req models.TOTPVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Verify TOTP code
	valid, err := ctrl.totpService.VerifyTOTP(userID, tenantID, req.TOTPCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify TOTP", "details": err.Error()})
		return
	}

	if !valid {
		// Audit log: Failed verification
		middlewares.Audit(c, "totp", userIDStr, "verify_failed", &middlewares.AuditChanges{
			Before: map[string]interface{}{
				"totp_code": "****",
			},
		})

		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TOTP code"})
		return
	}

	// Get user details
	user, err := ctrl.userRepo.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	// Get tenant details
	tenant, err := ctrl.tenantRepo.GetTenantByID(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant not found"})
		return
	}

	// Generate new JWT token (fresh 24-hour token)
	token, err := ctrl.generateJWTToken(user, tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	// Audit log: Successful verification
	middlewares.Audit(c, "totp", userIDStr, "verify_success", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"method": "totp",
		},
	})

	c.JSON(http.StatusOK, models.TOTPVerificationResponse{
		Success: true,
		Token:   token,
		Message: "TOTP verification successful",
	})
}

// GetUserDevices retrieves user's registered TOTP devices
// @Summary Get TOTP devices
// @Description Returns list of registered TOTP devices for the authenticated user.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.TOTPDeviceListResponse "Device list returned"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/devices [get]
func (ctrl *TOTPController) GetUserDevices(c *gin.Context) {
	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Get devices
	devices, err := ctrl.totpService.GetUserDevices(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve devices", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, models.TOTPDeviceListResponse{
		Success: true,
		Devices: devices,
	})
}

// DeleteDevice deletes a TOTP device
// @Summary Delete TOTP device
// @Description Deletes a registered TOTP device.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TOTPDeviceDeleteRequest true "Delete request"
// @Success 200 {object} models.TOTPDeviceDeleteResponse "Device deleted successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/device/delete [post]
func (ctrl *TOTPController) DeleteDevice(c *gin.Context) {
	var req models.TOTPDeviceDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Parse device ID
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	// Delete device
	if err := ctrl.totpService.DeleteDevice(deviceID, userID, tenantID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to delete device", "details": err.Error()})
		return
	}

	// Audit log: Device deleted
	middlewares.Audit(c, "totp", req.DeviceID, "delete_device", &middlewares.AuditChanges{
		Before: map[string]interface{}{
			"device_id": req.DeviceID,
		},
	})

	c.JSON(http.StatusOK, models.TOTPDeviceDeleteResponse{
		Success: true,
		Message: "Device deleted successfully",
	})
}

// SetPrimaryDevice sets a device as primary
// @Summary Set primary TOTP device
// @Description Marks a device as the primary TOTP device for authentication.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body models.TOTPDeviceDeleteRequest true "Set primary request"
// @Success 200 {object} models.TOTPDeviceDeleteResponse "Device set as primary"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/device/primary [post]
func (ctrl *TOTPController) SetPrimaryDevice(c *gin.Context) {
	var req models.TOTPDeviceDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Parse device ID
	deviceID, err := uuid.Parse(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
		return
	}

	// Set as primary
	if err := ctrl.totpService.SetPrimaryDevice(deviceID, userID, tenantID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set primary device", "details": err.Error()})
		return
	}

	// Audit log: Primary device set
	middlewares.Audit(c, "totp", req.DeviceID, "set_primary", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"device_id": req.DeviceID,
		},
	})

	c.JSON(http.StatusOK, models.TOTPDeviceDeleteResponse{
		Success: true,
		Message: "Device set as primary",
	})
}

// RegenerateBackupCodes regenerates backup codes
// @Summary Regenerate backup codes
// @Description Generates new backup codes. Old codes will be invalidated.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} models.BackupCodeRegenerateResponse "New backup codes generated"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/backup/regenerate [post]
func (ctrl *TOTPController) RegenerateBackupCodes(c *gin.Context) {
	// Get user info from JWT token
	userIDStr, err := middlewares.ResolveUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated", "details": err.Error()})
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user ID"})
		return
	}

	// Get tenant_id from context
	tenantIDStr, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID not found in token"})
		return
	}

	tenantID, err := uuid.Parse(tenantIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Regenerate backup codes
	backupCodes, err := ctrl.totpService.RegenerateBackupCodes(userID, tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to regenerate backup codes", "details": err.Error()})
		return
	}

	// Audit log: Backup codes regenerated
	middlewares.Audit(c, "totp", userIDStr, "regenerate_backup_codes", nil)

	c.JSON(http.StatusOK, models.BackupCodeRegenerateResponse{
		Success:     true,
		BackupCodes: backupCodes,
		Message:     "Backup codes regenerated. Store them securely!",
	})
}

// LoginWithTOTP performs TOTP-only login (no password required)
// @Summary TOTP-only login
// @Description Login using TOTP code only (no password). Returns JWT token if valid.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Param request body models.TOTPLoginRequest true "TOTP login request"
// @Success 200 {object} models.TOTPLoginResponse "Login successful"
// @Failure 400 {object} map[string]string "Bad request or invalid TOTP"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/login [post]
func (ctrl *TOTPController) LoginWithTOTP(c *gin.Context) {
	var req models.TOTPLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user by email
	user, err := ctrl.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Validate TOTP code
	valid, err := ctrl.totpService.LoginWithTOTPWithUser(user, req.TOTPCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TOTP code"})
		return
	}

	// Get tenant details
	tenant, err := ctrl.tenantRepo.GetTenantByID(user.TenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant not found"})
		return
	}

	// Generate JWT token
	token, err := ctrl.generateJWTToken(user, tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	// Audit log: TOTP login
	middlewares.Audit(c, "totp", user.ID.String(), "login", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":  req.Email,
			"method": "totp_only",
		},
	})

	c.JSON(http.StatusOK, models.TOTPLoginResponse{
		Success: true,
		Token:   token,
		Message: "Login successful",
	})
}

// ApproveDeviceCodeWithTOTP approves device code using TOTP (no JWT token required)
// @Summary Approve device code with TOTP
// @Description Approves a pending device authorization using TOTP code instead of login.
// @Tags TOTP Authentication
// @Accept json
// @Produce json
// @Param request body models.TOTPDeviceApprovalRequest true "TOTP device approval request"
// @Success 200 {object} models.TOTPDeviceApprovalResponse "Device approved"
// @Failure 400 {object} map[string]string "Bad request or invalid TOTP"
// @Failure 404 {object} map[string]string "User or device code not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/totp/device-approve [post]
func (ctrl *TOTPController) ApproveDeviceCodeWithTOTP(c *gin.Context) {
	var req models.TOTPDeviceApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Get user by email
	user, err := ctrl.userRepo.GetUserByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Validate TOTP code
	valid, err := ctrl.totpService.LoginWithTOTPWithUser(user, req.TOTPCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid TOTP code"})
		return
	}

	// Find device code
	dc, err := ctrl.deviceRepo.FindByUserCode(req.UserCode)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device code not found"})
		return
	}

	// Check if expired
	if dc.IsExpired() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device code expired"})
		return
	}

	// Check if already processed
	if dc.Status != "pending" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Device code already processed"})
		return
	}

	// Authorize device
	if err := ctrl.deviceRepo.AuthorizeDeviceCode(req.UserCode, user.ID, user.Email, true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to approve device", "details": err.Error()})
		return
	}

	// Get tenant details
	tenant, err := ctrl.tenantRepo.GetTenantByID(user.TenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant not found"})
		return
	}

	// Generate JWT token
	token, err := ctrl.generateJWTToken(user, tenant)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token", "details": err.Error()})
		return
	}

	// Audit log: Device approved with TOTP
	middlewares.Audit(c, "totp", user.ID.String(), "approve_device", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_code": req.UserCode,
			"method":    "totp",
		},
	})

	c.JSON(http.StatusOK, models.TOTPDeviceApprovalResponse{
		Success: true,
		Message: "Device approved successfully",
		Token:   token,
	})
}

// generateJWTToken generates a JWT token
func (ctrl *TOTPController) generateJWTToken(user *models.ExtendedUser, tenant *models.Tenant) (string, error) {
	// Use centralized auth-manager token service
	return config.TokenService.GenerateTOTPToken(
		user.ID,
		tenant.ID,
		user.Email,
		24*time.Hour,
	)
}
