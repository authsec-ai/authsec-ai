package handlers

import (
	"crypto/sha256"
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	middleware "github.com/authsec-ai/authsec/middlewares"
	appmodels "github.com/authsec-ai/authsec/models"
	repositories "github.com/authsec-ai/authsec/repository"
	"github.com/authsec-ai/authsec/services"
	util "github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

func NewSMSHandler() *SMSHandler {
	return &SMSHandler{
		Service: services.NewSMSService(),
	}
}

// extractRPIDFromOrigin extracts and normalizes the RP ID from an origin
// For WebAuthn, the RP ID is typically the registrable domain without scheme/port
func extractRPIDFromOrigin(origin string) string {
	if origin == "" {
		return ""
	}

	// Parse the origin URL
	u, err := url.Parse(origin)
	if err != nil {
		log.Printf("extractRPIDFromOrigin: failed to parse origin %s: %v", origin, err)
		return ""
	}

	// Get the host (which may include port)
	host := u.Host

	// Remove port if present
	if colonIdx := strings.LastIndex(host, ":"); colonIdx > 0 {
		// Check if this is actually a port (not IPv6)
		if !strings.Contains(host[colonIdx:], "]") {
			host = host[:colonIdx]
		}
	}

	// Normalize to lowercase
	host = strings.ToLower(host)

	// For subdomains of authsec.dev or authsec.ai, use the base domain as RP ID
	// e.g., tenant123.app.authsec.dev -> app.authsec.dev
	basedomains := []string{"app.authsec.dev", "stage.authsec.dev", "dev.authsec.dev", "app.authsec.ai", "stage.authsec.ai", "dev.authsec.ai"}

	for _, basedomain := range basedomains {
		if host == basedomain || strings.HasSuffix(host, "."+basedomain) {
			return basedomain
		}
	}

	// For custom tenant domains (e.g., test.auth-sec.org, tenant.example.com)
	// Use the full domain as RP ID
	// This allows each custom domain to have its own RP ID scope
	log.Printf("extractRPIDFromOrigin: Using custom domain RP ID: %s for origin: %s", host, origin)
	return host
}

// computeRPIDHash computes SHA-256 hash of RP ID (needed to match against credentials)
func computeRPIDHash(rpID string) []byte {
	hash := sha256.Sum256([]byte(rpID))
	return hash[:]
}

// hasWebAuthnCredentialsForRPID checks if a user has any WebAuthn credentials
// Note: The credentials table doesn't store RP ID, so we can only check if ANY credentials exist.
// The browser will filter credentials by RP ID during actual authentication.
// This is a best-effort approach given the current schema limitations.
func hasWebAuthnCredentialsForRPID(db *gorm.DB, userID, rpID string) bool {
	if rpID == "" {
		return false
	}

	// Check credentials table for this user (using client_id column per schema)
	var count int64
	err := db.Table("credentials").
		Where("client_id = ?", userID).
		Count(&count).Error

	if err != nil {
		log.Printf("hasWebAuthnCredentialsForRPID: error checking credentials for user %s: %v", userID, err)
		return false
	}

	// If user has any credentials, we return true and let the browser filter by RP ID
	// During actual authentication:
	// - Browser will only show credentials that match the current RP ID
	// - If no credentials match, authentication will fail and user needs to register
	return count > 0
}

// determineWebAuthnStatus determines if user should register or authenticate with WebAuthn
// Returns: "register" (no creds for this domain), "authenticate" (creds exist), or "unavailable" (webauthn not in methods)
func determineWebAuthnStatus(db *gorm.DB, userID, requestOrigin string, methods []sharedmodels.MFAMethod) string {
	// Check if webauthn is in user's enabled methods
	hasWebAuthn := false
	for _, method := range methods {
		if method.MethodType == "webauthn" && method.Enabled {
			hasWebAuthn = true
			break
		}
	}

	if !hasWebAuthn {
		return "unavailable"
	}

	// Extract RP ID from request origin
	rpID := extractRPIDFromOrigin(requestOrigin)
	if rpID == "" {
		log.Printf("determineWebAuthnStatus: Could not extract RP ID from origin %s", requestOrigin)
		// Default to "register" if we can't determine origin
		return "register"
	}

	// Check if this is a known base domain (authsec.dev/authsec.ai) or a custom domain
	knownBaseDomains := []string{"app.authsec.dev", "stage.authsec.dev", "dev.authsec.dev",
		"app.authsec.ai", "stage.authsec.ai", "dev.authsec.ai"}
	isKnownDomain := false
	for _, baseDomain := range knownBaseDomains {
		if rpID == baseDomain {
			isKnownDomain = true
			break
		}
	}

	// For custom domains: Always return "register"
	// Credentials are RP ID-specific - even if user has creds for app.authsec.dev,
	// they won't work on custom-domain.com. Browser filters by RP ID.
	if !isKnownDomain {
		log.Printf("determineWebAuthnStatus: Custom domain %s detected, returning 'register' (credentials from other domains won't work)", rpID)
		return "register"
	}

	// For known domains: Check if credentials exist
	// Credentials registered on app.authsec.dev work for *.app.authsec.dev
	hasCredentials := hasWebAuthnCredentialsForRPID(db, userID, rpID)

	if hasCredentials {
		return "authenticate"
	}

	return "register"
}

// GetMFAStatus returns the MFA status for a given user (generic check)
func (h *WebAuthnHandler) GetMFAStatus(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		TenantID string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := globalDB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).
		First(&userWithJSONMFA).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userWithJSONMFA.ToShared()

	mfaRepo := repositories.NewMFARepository(globalDB)
	methods, _ := mfaRepo.GetUserMethods(user.ID.String())

	c.JSON(http.StatusOK, gin.H{
		"user_id":     user.ID,
		"email":       user.Email,
		"mfa_enabled": user.MFAEnabled,
		"methods":     methods,
	})
}

// GetMFAStatusForLogin returns MFA status specifically for login flow
func (h *WebAuthnHandler) GetMFAStatusForLogin(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		TenantID string `json:"tenant_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := globalDB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).
		First(&userWithJSONMFA).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userWithJSONMFA.ToShared()

	if !user.MFAEnabled {
		c.JSON(http.StatusOK, gin.H{"mfa_required": false})
		return
	}

	mfaRepo := repositories.NewMFARepository(globalDB)
	methods, _ := mfaRepo.GetUserMethods(user.ID.String())

	// Check WebAuthn availability for current domain
	// WebAuthn credentials are RP ID-specific, so we need to check if user has
	// credentials registered for the current domain's RP ID
	requestOrigin := c.Request.Header.Get("Origin")
	webauthnStatus := determineWebAuthnStatus(globalDB, user.ID.String(), requestOrigin, methods)

	c.JSON(http.StatusOK, gin.H{
		"mfa_required":    true,
		"methods":         methods,
		"webauthn_status": webauthnStatus, // "register", "authenticate", or "unavailable"
	})
}

// GetMFAStatusForLoginGET returns MFA status for login flow via GET request with query parameters
func (h *WebAuthnHandler) GetMFAStatusForLoginGET(c *gin.Context) {
	email := c.Query("email")
	tenantID := c.Query("tenant_id")

	if email == "" || tenantID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email and tenant_id query parameters are required"})
		return
	}

	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := globalDB.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", email, tenantID).
		First(&userWithJSONMFA).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userWithJSONMFA.ToShared()

	if !user.MFAEnabled {
		c.JSON(http.StatusOK, gin.H{"mfa_required": false})
		return
	}

	mfaRepo := repositories.NewMFARepository(globalDB)
	methods, _ := mfaRepo.GetUserMethods(user.ID.String())

	// Check WebAuthn availability for current domain
	// WebAuthn credentials are RP ID-specific, so we need to check if user has
	// credentials registered for the current domain's RP ID
	requestOrigin := c.Request.Header.Get("Origin")
	webauthnStatus := determineWebAuthnStatus(globalDB, user.ID.String(), requestOrigin, methods)

	c.JSON(http.StatusOK, gin.H{
		"mfa_required":    true,
		"methods":         methods,
		"webauthn_status": webauthnStatus, // "register", "authenticate", or "unavailable"
	})
}

// @Summary      Begin SMS Setup
// @Description  Start SMS MFA setup process
// @Tags         SMS
// @Accept       json
// @Produce      json
// @Param        request body SMSSetupRequest true "SMS setup information"
// @Success      200 {object} SMSSetupResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/sms/beginSetup [post]
// BeginSMSSetup initiates SMS MFA setup

func (h *SMSHandler) BeginSMSSetup(c *gin.Context) {
	var req SMSSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Starting SMS setup for email: %s, phone: %s", req.Email, req.PhoneNumber)

	// Validate phone number
	if !h.Service.ValidatePhoneNumber(req.PhoneNumber) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid phone number format (use +1234567890)"})
		return
	}

	// Get client from database
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Generate verification code
	code, err := h.Service.GenerateCode()
	if err != nil {
		log.Printf("Failed to generate SMS code: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate verification code"})
		return
	}

	// Send SMS
	if err := h.Service.SendCode(req.PhoneNumber, code); err != nil {
		log.Printf("Failed to send SMS: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to send SMS code"})
		return
	}

	// Store verification data temporarily (encrypt the code)
	encryptedCode, _ := util.EncryptString(code)

	smsData := map[string]interface{}{
		"phone_number":       req.PhoneNumber,
		"phone_verified":     false,
		"verification_code":  encryptedCode,
		"code_expires_at":    time.Now().Add(5 * time.Minute).UTC(),
		"setup_initiated_at": time.Now().UTC(),
		"attempts_remaining": 3,
	}

	// Save as disabled method until confirmation
	mfaRepo := repositories.NewMFARepository(tenantDB)
	err = mfaRepo.EnableMethodWithExpiry(client.ID.String(), "sms", smsData, false, time.Now().Add(10*time.Minute), client.ID)
	if err != nil {
		log.Printf("Failed to save SMS setup data: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save SMS setup"})
		return
	}

	log.Printf("SMS verification code sent to: %s", h.Service.FormatPhoneForDisplay(req.PhoneNumber))

	response := SMSSetupResponse{
		Success:           true,
		Message:           "SMS verification code sent",
		PhoneDisplay:      h.Service.FormatPhoneForDisplay(req.PhoneNumber),
		ExpiresInMinutes:  5,
		AttemptsRemaining: 3,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Confirm SMS Setup
// @Description  Confirm SMS setup with verification code
// @Tags         SMS
// @Accept       json
// @Produce      json
// @Param        request body SMSConfirmRequest true "SMS confirmation data"
// @Success      200 {object} SMSConfirmResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/sms/confirmSetup [post]
// ConfirmSMSSetup verifies SMS code and enables SMS MFA
func (h *SMSHandler) ConfirmSMSSetup(c *gin.Context) {
	var req SMSConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Confirming SMS setup for email: %s", req.Email)

	// Get client and SMS method data
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	mfaRepo := repositories.NewMFARepository(tenantDB)
	method, err := mfaRepo.GetMethod(client.ID.String(), "sms")
	if err != nil {
		log.Printf("SMS setup not found: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "SMS setup not initiated"})
		return
	}

	// Parse SMS data
	var smsData struct {
		PhoneNumber       string    `json:"phone_number"`
		PhoneVerified     bool      `json:"phone_verified"`
		VerificationCode  string    `json:"verification_code"`
		CodeExpiresAt     time.Time `json:"code_expires_at"`
		AttemptsRemaining int       `json:"attempts_remaining"`
	}

	if err := json.Unmarshal(method.MethodData, &smsData); err != nil {
		log.Printf("Failed to parse SMS data: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "invalid SMS setup data"})
		return
	}

	// Check if code expired
	if time.Now().After(smsData.CodeExpiresAt) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "verification code expired"})
		return
	}

	// Check attempts remaining
	if smsData.AttemptsRemaining <= 0 {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "too many verification attempts"})
		return
	}

	// Decrypt and verify code
	storedCode, err := util.DecryptString(smsData.VerificationCode)
	if err != nil {
		log.Printf("Failed to decrypt verification code: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "verification failed"})
		return
	}

	if req.Code != storedCode {
		// Decrement attempts
		smsData.AttemptsRemaining--
		updatedData, _ := json.Marshal(smsData)
		mfaRepo.UpdateMethodData(client.ID.String(), "sms", updatedData)

		c.JSON(http.StatusBadRequest, gin.H{
			"error":              "invalid verification code",
			"attempts_remaining": smsData.AttemptsRemaining,
		})
		return
	}

	// Code is valid - enable SMS method
	confirmedSMSData := map[string]interface{}{
		"phone_number":   smsData.PhoneNumber,
		"phone_verified": true,
		"confirmed_at":   time.Now().UTC(),
		"country_code":   smsData.PhoneNumber[:3], // Extract country code
	}

	err = mfaRepo.EnableMethod(client.ID.String(), "sms", confirmedSMSData, client.ID)
	if err != nil {
		log.Printf("Failed to enable SMS method: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to enable SMS MFA"})
		return
	}

	// Load user for update using UserWithJSONMFAMethods to handle JSONB mfa_method field
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := tenantDB.Scopes(util.WithUsersMFAMethodArray).Where("id = ?", client.ID).First(&userWithJSONMFA).Error; err != nil {
		log.Printf("Failed to load user for MFA update: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load user for update"})
		return
	}

	// Update client MFA settings using raw SQL
	now := time.Now()
	newMethods := client.MFAMethod
	if !contains(newMethods, "sms") {
		newMethods = append(newMethods, "sms")
	}

	updateSQL := `
		UPDATE users 
		SET mfa_enabled = true, 
		    mfa_default_method = CASE WHEN mfa_default_method IS NULL OR mfa_default_method = '' THEN 'sms' ELSE mfa_default_method END,
		    mfa_method = $1,
		    updated_at = $2
		WHERE id = $3`

	if err := tenantDB.Exec(updateSQL, pq.StringArray(newMethods), now, client.ID).Error; err != nil {
		log.Printf("Failed to update client MFA settings: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update MFA settings"})
		return
	}

	log.Printf("SMS MFA enabled for: %s", req.Email)

	// Audit log for successful SMS setup
	middleware.AuditAuthentication(c, client.ID.String(), "sms", "setup", true, map[string]interface{}{
		"tenant_id": req.TenantID,
		"email":     req.Email,
	})

	response := SMSConfirmResponse{
		Success:      true,
		Message:      "SMS MFA enabled successfully",
		PhoneDisplay: h.Service.FormatPhoneForDisplay(smsData.PhoneNumber),
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Request SMS Code
// @Description  Request SMS code for authentication
// @Tags         SMS
// @Accept       json
// @Produce      json
// @Param        request body RequestSMSCodeRequest true "SMS code request"
// @Success      200 {object} SMSCodeResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/sms/requestCode [post]
// RequestSMSCode sends a new SMS code for authentication
func (h *SMSHandler) RequestSMSCode(c *gin.Context) {
	var req RequestSMSCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Get client and SMS method
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	mfaRepo := repositories.NewMFARepository(tenantDB)
	method, err := mfaRepo.GetMethod(client.ID.String(), "sms")
	if err != nil || !method.Enabled {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "SMS MFA not enabled"})
		return
	}

	// Parse existing SMS data
	var smsData map[string]interface{}
	json.Unmarshal(method.MethodData, &smsData)

	phoneNumber := smsData["phone_number"].(string)

	// Generate new code
	code, _ := h.Service.GenerateCode()
	encryptedCode, _ := util.EncryptString(code)

	// Send SMS
	if err := h.Service.SendCode(phoneNumber, code); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to send SMS"})
		return
	}

	// Update method data with new code
	smsData["verification_code"] = encryptedCode
	smsData["code_expires_at"] = time.Now().Add(5 * time.Minute).UTC()
	smsData["attempts_remaining"] = 3

	updatedData, _ := json.Marshal(smsData)
	mfaRepo.UpdateMethodData(client.ID.String(), "sms", updatedData)

	response := SMSCodeResponse{
		Success:           true,
		Message:           "SMS code sent",
		PhoneDisplay:      h.Service.FormatPhoneForDisplay(phoneNumber),
		ExpiresInMinutes:  5,
		AttemptsRemaining: 3,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Verify SMS Code
// @Description  Verify SMS code for authentication
// @Tags         SMS
// @Accept       json
// @Produce      json
// @Param        request body VerifySMSRequest true "SMS verification data"
// @Success      200 {object} AuthenticationResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/sms/verify [post]
// VerifySMS validates SMS code for authentication
func (h *SMSHandler) VerifySMS(c *gin.Context) {
	var req VerifySMSRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Verifying SMS code for email: %s", req.Email)

	// Get client and SMS method
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	mfaRepo := repositories.NewMFARepository(tenantDB)
	method, err := mfaRepo.GetMethod(client.ID.String(), "sms")
	if err != nil || !method.Enabled {
		log.Printf("SMS not enabled for client: %s", req.Email)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "SMS MFA not enabled for this account"})
		return
	}

	// For authentication, we need to check if this is a recent code
	// In production, you'd store authentication codes separately from setup codes
	var smsData struct {
		PhoneNumber       string    `json:"phone_number"`
		VerificationCode  string    `json:"verification_code,omitempty"`
		CodeExpiresAt     time.Time `json:"code_expires_at,omitempty"`
		AttemptsRemaining int       `json:"attempts_remaining,omitempty"`
	}

	if err := json.Unmarshal(method.MethodData, &smsData); err != nil {
		log.Printf("Failed to parse SMS data: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "invalid SMS configuration"})
		return
	}

	// Check if there's a pending verification code
	if smsData.VerificationCode == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no SMS code requested. Please request a new code."})
		return
	}

	// Verify the code (similar logic as confirm setup)
	if time.Now().After(smsData.CodeExpiresAt) {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "SMS code expired"})
		return
	}

	storedCode, err := util.DecryptString(smsData.VerificationCode)
	if err != nil || req.Code != storedCode {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid SMS code"})
		return
	}

	// Update last used timestamp
	mfaRepo.UpdateLastUsed(client.ID.String(), "sms")

	log.Printf("SMS code verified for: %s", req.Email)
	// Fetch the client for this user's tenant
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := tenantDB.Scopes(util.WithUsersMFAMethodArray).Where("tenant_id = ?", req.TenantID).First(&userWithJSONMFA).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to find client and project"})
		return
	}
	user := userWithJSONMFA.ToShared()
	user.MFAVerified = true
	if err := tenantDB.Model(&userWithJSONMFA).Update("mfa_verified", true).Error; err != nil {
		log.Printf("Failed to update user mfa_verified status: %v", err)
	}

	// Audit log for successful SMS verification
	middleware.AuditAuthentication(c, client.ID.String(), "sms", "verify", true, map[string]interface{}{
		"tenant_id": req.TenantID,
		"email":     req.Email,
	})

	response := AuthenticationResponse{
		Success:  true,
		Message:  "SMS code valid",
		Method:   "sms",
		TenantID: req.TenantID,
		Email:    client.Email,
	}

	c.JSON(http.StatusOK, response)
}
