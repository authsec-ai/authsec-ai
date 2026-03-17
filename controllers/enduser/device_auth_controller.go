package enduser

import (
	"fmt"
	"net/http"
	"os"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeviceAuthController handles device authorization grant endpoints
type DeviceAuthController struct {
	deviceService *services.DeviceAuthService
	tenantRepo    *database.AdminTenantRepository
}

// NewDeviceAuthController creates a new device authorization controller
func NewDeviceAuthController() (*DeviceAuthController, error) {
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

	return &DeviceAuthController{
		deviceService: services.NewDeviceAuthService(db, tenantDBService),
		tenantRepo:    database.NewAdminTenantRepository(db),
	}, nil
}

// RequestDeviceCode initiates the device authorization flow
// @Summary Request device code
// @Description Initiates device authorization grant flow (RFC 8628). Returns device_code for polling and user_code for user activation.
// @Tags Device Authorization
// @Accept json
// @Produce json
// @Param request body models.DeviceCodeRequest true "Device code request"
// @Success 200 {object} models.DeviceCodeResponse "Device code created successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/device/code [post]
func (ctrl *DeviceAuthController) RequestDeviceCode(c *gin.Context) {
	var req models.DeviceCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Initiate device flow
	resp, err := ctrl.deviceService.InitiateDeviceFlow(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device code", "details": err.Error()})
		return
	}

	// Audit log: Device code requested
	middlewares.Audit(c, "device_auth", resp.DeviceCode, "request_code", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"client_id":        req.ClientID,
			"user_code":        resp.UserCode,
			"verification_uri": resp.VerificationURI,
			"expires_in":       resp.ExpiresIn,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// PollDeviceToken polls for device authorization status
// @Summary Poll for device token
// @Description Device polls this endpoint to check if user has authorized. Returns token when authorized, or error codes per RFC 8628.
// @Tags Device Authorization
// @Accept json
// @Produce json
// @Param request body models.DeviceTokenRequest true "Device token request"
// @Success 200 {object} models.DeviceTokenResponse "Token issued or status returned"
// @Failure 400 {object} map[string]string "Bad request"
// @Router /uflow/auth/device/token [post]
func (ctrl *DeviceAuthController) PollDeviceToken(c *gin.Context) {
	var req models.DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Poll for token
	resp, err := ctrl.deviceService.PollForToken(req.DeviceCode, req.ClientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to poll device token", "details": err.Error()})
		return
	}

	// RFC 8628 specifies that pending/slow_down/expired_token return HTTP 400
	// access_denied returns HTTP 403, but we simplify to 200 with error field
	if resp.Error != "" {
		statusCode := http.StatusOK
		if resp.Error == models.ErrorAccessDenied {
			statusCode = http.StatusForbidden
		} else if resp.Error == models.ErrorExpiredToken {
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetActivationInfo retrieves device information for activation page
// @Summary Get device activation info
// @Description Returns device information to display on the activation page (user_code, tenant, scopes, etc.)
// @Tags Device Authorization
// @Accept json
// @Produce json
// @Param user_code query string true "User code from device"
// @Success 200 {object} models.DeviceActivationInfoResponse "Device activation information"
// @Failure 400 {object} map[string]string "Bad request - invalid or expired code"
// @Router /uflow/auth/device/activate/info [get]
func (ctrl *DeviceAuthController) GetActivationInfo(c *gin.Context) {
	userCode := c.Query("user_code")
	if userCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code query parameter required"})
		return
	}

	info, err := ctrl.deviceService.GetDeviceActivationInfo(userCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired user code", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// VerifyDeviceCode verifies and authorizes/denies a device code
// @Summary Verify device code
// @Description User authorizes or denies the device after authenticating. Requires JWT token in Authorization header.
// @Tags Device Authorization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param Authorization header string true "Bearer JWT token"
// @Param request body models.DeviceVerificationRequest true "Device verification request"
// @Success 200 {object} models.DeviceVerificationResponse "Device verified successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized - invalid or missing token"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/device/verify [post]
func (ctrl *DeviceAuthController) VerifyDeviceCode(c *gin.Context) {
	var req models.DeviceVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Extract user info from JWT token (set by auth middleware)
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

	// Get user email from token (try both email_id and email)
	email := ""
	if emailID, exists := c.Get("email_id"); exists && emailID != nil {
		email = emailID.(string)
	} else if emailVal, exists := c.Get("email"); exists && emailVal != nil {
		email = emailVal.(string)
	}

	// Verify device code
	if err := ctrl.deviceService.VerifyDeviceCode(req.UserCode, userID, email, req.Approve); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify device code", "details": err.Error()})
		return
	}

	message := "Device authorized successfully"
	action := "authorize"
	if !req.Approve {
		message = "Device authorization denied"
		action = "deny"
	}

	// Audit log: Device code verified
	middlewares.Audit(c, "device_auth", req.UserCode, action, &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_code": req.UserCode,
			"user_id":   userID.String(),
			"email":     email,
			"approved":  req.Approve,
		},
	})

	c.JSON(http.StatusOK, models.DeviceVerificationResponse{
		Success: true,
		Message: message,
	})
}

// ShowActivationPage serves the HTML page for device activation
// @Summary Show device activation page
// @Description Displays the HTML page where users activate their devices
// @Tags Device Authorization
// @Produce html
// @Param user_code query string false "User code (optional - can be entered on page)"
// @Success 200 {string} string "HTML page"
// @Router /activate [get]
func (ctrl *DeviceAuthController) ShowActivationPage(c *gin.Context) {
	// Read HTML template from file
	htmlContent, err := os.ReadFile("templates/device_activation.html")
	if err != nil {
		// Fallback to inline HTML if file not found
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(getActivationPageHTML()))
		return
	}

	c.Data(http.StatusOK, "text/html; charset=utf-8", htmlContent)
}

// getActivationPageHTML returns inline HTML if template file is not available
func getActivationPageHTML() string {
	return `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><title>Device Activation</title></head>
<body style="font-family: Arial, sans-serif; max-width: 500px; margin: 50px auto; padding: 20px;">
<h1>Device Activation</h1>
<p>Enter the code shown on your device:</p>
<form id="codeForm">
<input type="text" id="userCode" placeholder="ABCD-1234" style="padding: 10px; font-size: 16px; width: 100%;" maxlength="9">
<button type="submit" style="margin-top: 10px; padding: 10px 20px; background: #667eea; color: white; border: none; border-radius: 5px; cursor: pointer;">Continue</button>
</form>
<div id="result" style="margin-top: 20px;"></div>
<script>
document.getElementById('codeForm').addEventListener('submit', async (e) => {
e.preventDefault();
const code = document.getElementById('userCode').value.trim().toUpperCase();
if (!code) return;
try {
const res = await fetch('/uflow/auth/device/activate/info?user_code=' + code);
const data = await res.json();
if (!res.ok) throw new Error(data.error);
window.location.href = '/uflow/auth/enduser/login?redirect_after=/activate/approve?user_code=' + code;
} catch (err) {
document.getElementById('result').innerHTML = '<p style="color: red;">Error: ' + err.message + '</p>';
}
});
</script>
</body></html>`
}
