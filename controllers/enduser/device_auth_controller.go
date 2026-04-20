package enduser

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeviceAuthController handles device authorization grant endpoints (RFC 8628)
type DeviceAuthController struct {
	deviceService *services.DeviceAuthService
	oidcService   *services.OIDCService
	tenantRepo    *database.AdminTenantRepository
	userRepo      *database.UserRepository
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
		oidcService:   services.NewOIDCService(db),
		tenantRepo:    database.NewAdminTenantRepository(db),
		userRepo:      database.NewUserRepository(db),
	}, nil
}

// RequestDeviceCode initiates the device authorization flow.
// No authentication required. CLI sends only scopes — no client_id or tenant_domain.
// @Router /uflow/auth/device/code [post]
func (ctrl *DeviceAuthController) RequestDeviceCode(c *gin.Context) {
	var req models.DeviceCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	resp, err := ctrl.deviceService.InitiateDeviceFlow(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create device code", "details": err.Error()})
		return
	}

	middlewares.Audit(c, "device_auth", resp.DeviceCode, "request_code", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_code":        resp.UserCode,
			"verification_uri": resp.VerificationURI,
			"expires_in":       resp.ExpiresIn,
		},
	})

	c.JSON(http.StatusOK, resp)
}

// PollDeviceToken is polled by the CLI to get the access token once authorized.
// Returns RFC 8628 error codes in the body; HTTP 400 for expired/slow_down, 403 for access_denied.
// @Router /uflow/auth/device/token [post]
func (ctrl *DeviceAuthController) PollDeviceToken(c *gin.Context) {
	var req models.DeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	resp, err := ctrl.deviceService.PollForToken(req.DeviceCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to poll device token", "details": err.Error()})
		return
	}

	if resp.Error != "" {
		statusCode := http.StatusOK
		switch resp.Error {
		case models.ErrorAccessDenied:
			statusCode = http.StatusForbidden
		case models.ErrorExpiredToken, models.ErrorSlowDown:
			statusCode = http.StatusBadRequest
		}
		c.JSON(statusCode, resp)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetActivationInfo returns device info for the activation page (public).
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

// VerifyUserCode validates a user_code without authentication.
// Called by app.authsec.ai/activate before the user logs in — confirms the code exists.
// @Router /uflow/auth/device/verify [post]
func (ctrl *DeviceAuthController) VerifyUserCode(c *gin.Context) {
	var req models.DeviceVerifyCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	dc, err := ctrl.deviceService.ValidateUserCode(req.UserCode)
	if err != nil {
		c.JSON(http.StatusOK, models.DeviceVerifyCodeResponse{
			Valid:  false,
			Error:  "invalid_code",
		})
		return
	}

	expiresIn := int(dc.ExpiresAt - time.Now().Unix())
	if expiresIn < 0 {
		expiresIn = 0
	}

	c.JSON(http.StatusOK, models.DeviceVerifyCodeResponse{
		Valid:        true,
		DeviceCodeID: dc.ID.String(),
		ExpiresIn:    expiresIn,
	})
}

// AuthorizeDevice is called by the web app (app.authsec.ai/activate) after the
// authenticated user enters their user_code and approves or denies the device.
// Requires: valid browser session (Bearer token from enduser login).
// @Router /uflow/auth/device/authorize [post]
func (ctrl *DeviceAuthController) AuthorizeDevice(c *gin.Context) {
	var req models.DeviceAuthorizeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

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

	// Email from token claims
	email := ""
	if v, ok := c.Get("email_id"); ok && v != nil {
		email, _ = v.(string)
	}
	if email == "" {
		if v, ok := c.Get("email"); ok && v != nil {
			email, _ = v.(string)
		}
	}

	// tenant_id from token claims (set by AuthMiddleware)
	tenantIDStr := ""
	if v, ok := c.Get("tenant_id"); ok && v != nil {
		tenantIDStr, _ = v.(string)
	}
	if tenantIDStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant context missing from session"})
		return
	}
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid tenant ID in token"})
		return
	}

	// Resolve tenant_domain via DB lookup (not in JWT claims)
	tenant, err := ctrl.tenantRepo.GetTenantByID(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found"})
		return
	}
	tenantDomain := tenant.TenantDomain

	// client_id from token claims
	var clientID *uuid.UUID
	if v, ok := c.Get("client_id"); ok && v != nil {
		if cidStr, ok := v.(string); ok && cidStr != "" {
			if parsed, err := uuid.Parse(cidStr); err == nil {
				clientID = &parsed
			}
		}
	}

	if err := ctrl.deviceService.AuthorizeDevice(
		req.UserCode, userID, email, tenantID, tenantDomain, clientID, req.Approved,
	); err != nil {
		statusCode := http.StatusBadRequest
		// 404 if code not found / expired
		if err.Error() == "invalid user code" {
			statusCode = http.StatusNotFound
		}
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}

	action := "authorize"
	status := "authorized"
	if !req.Approved {
		action = "deny"
		status = "denied"
	}

	middlewares.Audit(c, "device_auth", req.UserCode, action, &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_code": req.UserCode,
			"user_id":   userID.String(),
			"tenant_id": tenantID.String(),
			"approved":  req.Approved,
		},
	})

	c.JSON(http.StatusOK, models.DeviceAuthorizeResponse{Status: status})
}

// VerifyDeviceCode is the legacy endpoint (kept for backwards compatibility).
// New integrations should use POST /device/authorize.
// @Router /uflow/auth/device/verify [post]
func (ctrl *DeviceAuthController) VerifyDeviceCode(c *gin.Context) {
	var req models.DeviceVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

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

	email := ""
	if v, ok := c.Get("email_id"); ok && v != nil {
		email, _ = v.(string)
	}
	if email == "" {
		if v, ok := c.Get("email"); ok && v != nil {
			email, _ = v.(string)
		}
	}

	tenantIDStr := ""
	if v, ok := c.Get("tenant_id"); ok && v != nil {
		tenantIDStr, _ = v.(string)
	}
	tenantID := uuid.Nil
	if tenantIDStr != "" {
		if parsed, parseErr := uuid.Parse(tenantIDStr); parseErr == nil {
			tenantID = parsed
		}
	}

	tenantDomain := ""
	if tenantID != uuid.Nil {
		if t, tErr := ctrl.tenantRepo.GetTenantByID(tenantIDStr); tErr == nil {
			tenantDomain = t.TenantDomain
		}
	}

	var clientID *uuid.UUID
	if v, ok := c.Get("client_id"); ok && v != nil {
		if cidStr, ok := v.(string); ok && cidStr != "" {
			if parsed, parseErr := uuid.Parse(cidStr); parseErr == nil {
				clientID = &parsed
			}
		}
	}

	if err := ctrl.deviceService.VerifyDeviceCode(req.UserCode, userID, email, tenantID, tenantDomain, clientID, req.Approve); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to verify device code", "details": err.Error()})
		return
	}

	message := "Device authorized successfully"
	action := "authorize"
	if !req.Approve {
		message = "Device authorization denied"
		action = "deny"
	}

	middlewares.Audit(c, "device_auth", req.UserCode, action, &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_code": req.UserCode,
			"user_id":   userID.String(),
			"approved":  req.Approve,
		},
	})

	c.JSON(http.StatusOK, models.DeviceVerificationResponse{
		Success: true,
		Message: message,
	})
}

// ShowActivationPage serves the HTML page for device activation
// @Router /activate [get]
func (ctrl *DeviceAuthController) ShowActivationPage(c *gin.Context) {
	htmlContent, err := os.ReadFile("templates/device_activation.html")
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(getActivationPageHTML()))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", htmlContent)
}

// AuthorizeDeviceWithOIDC is the public endpoint for shield end-user login.
// After the user authenticates via OIDC (SSO), the callback page sends:
//   {user_code, oidc_code, state}
// This endpoint exchanges the OIDC code for user identity, then authorizes
// the device code — so the shield's poll gets the token.
// No JWT required — the OIDC code exchange itself proves authentication.
// @Router /uflow/auth/device/authorize-oidc [post]
func (ctrl *DeviceAuthController) AuthorizeDeviceWithOIDC(c *gin.Context) {
	var req models.DeviceAuthorizeOIDCRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	if req.UserCode == "" || req.OIDCCode == "" || req.State == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "user_code, oidc_code, and state are required"})
		return
	}

	// Step 1: Validate the user_code is still pending
	dc, err := ctrl.deviceService.ValidateUserCode(req.UserCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired device code"})
		return
	}
	_ = dc

	// Step 2: Exchange the OIDC code for user identity
	callbackInput := &models.OIDCCallbackInput{
		Code:  req.OIDCCode,
		State: req.State,
	}
	state, userInfo, err := ctrl.oidcService.HandleCallback(callbackInput)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "OIDC authentication failed", "details": err.Error()})
		return
	}

	if userInfo == nil || userInfo.Email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Could not retrieve user identity from OIDC provider"})
		return
	}

	// Step 3: Resolve user in the tenant
	var tenantID uuid.UUID
	if state.TenantID != nil {
		tenantID = *state.TenantID
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Could not determine tenant from OIDC state"})
		return
	}

	user, err := ctrl.userRepo.GetUserByEmailAndTenant(userInfo.Email, tenantID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in this workspace", "details": err.Error()})
		return
	}

	// Step 4: Get tenant info
	tenant, err := ctrl.tenantRepo.GetTenantByID(tenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant not found"})
		return
	}

	// Step 5: client_id — resolve from tenant mappings if available
	var clientID *uuid.UUID
	// clientID will be nil here; AuthorizeDevice handles this gracefully

	// Step 6: Authorize the device code with the user's identity
	if err := ctrl.deviceService.AuthorizeDevice(
		req.UserCode,
		user.ID,
		user.Email,
		tenantID,
		tenant.TenantDomain,
		clientID,
		true, // approved
	); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to authorize device", "details": err.Error()})
		return
	}

	middlewares.Audit(c, "device_auth", req.UserCode, "authorize_oidc", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"user_code":  req.UserCode,
			"user_email": user.Email,
			"tenant_id":  tenantID.String(),
			"method":     "oidc",
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"status":  "authorized",
		"message": "Device authorized. The CLI will receive credentials shortly.",
	})
}

// getActivationPageHTML returns inline HTML fallback if template file is not available
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
