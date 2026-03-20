package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"

	"github.com/authsec-ai/authsec/config"
	middleware "github.com/authsec-ai/authsec/middlewares"
	repositories "github.com/authsec-ai/authsec/repository"
	sharedmodels "github.com/authsec-ai/sharedmodels"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// AdminWebAuthnHandler handles WebAuthn operations for admin users
// Uses global database for all operations
type AdminWebAuthnHandler struct {
	WebAuthn       *webauthn.WebAuthn
	SessionManager SessionManagerInterface
	RPDisplayName  string
	RPID           string
	RPOrigins      []string
}

// resolveDB provides database connection for admin operations (always global DB)
func (h *AdminWebAuthnHandler) resolveDB() (*gorm.DB, error) {
	return config.ConnectGlobalDB()
}

// validateOriginAndCreateWebAuthn validates the origin and creates a WebAuthn instance for admin operations
func (h *AdminWebAuthnHandler) validateOriginAndCreateWebAuthn(c *gin.Context) (*webauthn.WebAuthn, error) {
	// Get the origin from the request
	requestOrigin := c.Request.Header.Get("Origin")
	log.Printf("AdminWebAuthn validateOrigin: Request origin=%s", requestOrigin)

	// Check for custom domain FIRST before standard validation
	// This prevents custom domains from being overridden by standard validation
	originURL, err := url.Parse(requestOrigin)
	if err == nil && originURL.Scheme == "https" {
		domain := originURL.Host

		// Check if this domain is a verified custom domain in the database
		globalDB, err := h.resolveDB()
		if err == nil {
			var count int64
			err := globalDB.Table("tenant_domains").
				Where("domain = ? AND is_verified = true", domain).
				Count(&count).Error

			if err == nil && count > 0 {
				log.Printf("AdminWebAuthn validateOrigin: Verified custom domain: %s", domain)
				dynamicWebAuthn := config.SetupWebAuthn(
					"AuthSec MFA Service",
					domain, // Use custom domain as RP ID
					requestOrigin,
				)
				return dynamicWebAuthn, nil
			}
			log.Printf("AdminWebAuthn validateOrigin: Domain %s not found in tenant_domains or not verified", domain)
		} else {
			log.Printf("AdminWebAuthn validateOrigin: Failed to resolve DB: %v", err)
		}
	}

	// Second, try standard subdomain validation
	if config.ValidateSubdomainOrigin(requestOrigin) {
		log.Printf("AdminWebAuthn validateOrigin: Creating dynamic WebAuthn for valid origin %s", requestOrigin)

		dynamicWebAuthn := config.SetupWebAuthn(
			"AuthSec MFA Service",
			"app.authsec.dev",
			requestOrigin, // Use the actual request origin
		)
		return dynamicWebAuthn, nil
	}

	// Origin validation failed
	log.Printf("AdminWebAuthn validateOrigin: Origin validation failed for %s", requestOrigin)
	return nil, fmt.Errorf("invalid origin: %s not allowed", requestOrigin)
}

// GetMFAStatus returns the MFA status for admin users
func (h *AdminWebAuthnHandler) GetMFAStatus(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email is required"})
		return
	}

	globalDB, err := h.resolveDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Get admin user by email (no tenant context)
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if !user.MFAEnabled {
		c.JSON(http.StatusOK, gin.H{"mfa_required": false})
		return
	}

	mfaRepo := repositories.NewMFARepository(globalDB)
	methods, _ := mfaRepo.GetUserMethods(user.ID.String())

	c.JSON(http.StatusOK, gin.H{
		"mfa_required": true,
		"methods":      methods,
	})
}

// GetMFAStatusForLogin returns MFA status for admin login flow
func (h *AdminWebAuthnHandler) GetMFAStatusForLogin(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email is required"})
		return
	}

	globalDB, err := h.resolveDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Get admin user by email (no tenant context)
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if !user.MFAEnabled {
		c.JSON(http.StatusOK, gin.H{"mfa_required": false})
		return
	}

	mfaRepo := repositories.NewMFARepository(globalDB)
	methods, _ := mfaRepo.GetUserMethods(user.ID.String())

	// Check for verified custom domain for this admin's tenant
	globalRepo := repositories.NewGlobalRepository(globalDB)
	customDomain, err := globalRepo.GetVerifiedCustomDomainForTenant(user.TenantID.String())
	if err != nil {
		log.Printf("GetMFAStatusForLogin: Error checking custom domain: %v", err)
	}

	response := gin.H{
		"mfa_required": true,
		"methods":      methods,
	}

	// If there's a verified custom domain, include it in the response
	if customDomain != "" {
		response["custom_domain"] = customDomain

		// Check if user has WebAuthn as an MFA method
		hasWebAuthnMethod := false
		for _, method := range methods {
			if method.MethodType == "webauthn" {
				hasWebAuthnMethod = true
				break
			}
		}

		// Only check for custom domain credentials if user has WebAuthn configured
		// If user has other MFA methods (TOTP, SMS, etc.), they don't need WebAuthn re-registration
		if hasWebAuthnMethod {
			// Check if user has credentials that are valid for this custom domain
			// Strategy: Check if credentials were created for this specific RP ID
			hasValidCreds, err := clientRepo.HasCredentialsForRPID(user.ID.String(), customDomain)
			if err != nil {
				log.Printf("GetMFAStatusForLogin: Error checking credentials for RP ID %s: %v", customDomain, err)
			} else if hasValidCreds {
				log.Printf("GetMFAStatusForLogin: User has credentials for custom domain RP ID: %s", customDomain)
			} else {
				log.Printf("GetMFAStatusForLogin: User has no credentials for custom domain RP ID: %s", customDomain)
				// User has WebAuthn method but no credentials for this domain - needs re-registration
				response["requires_registration"] = true
				response["message"] = "WebAuthn credentials required for custom domain. Please complete registration."
			}
		} else {
			// User doesn't have WebAuthn, they're using other MFA methods (TOTP, etc.)
			// No re-registration needed for custom domain
			log.Printf("GetMFAStatusForLogin: User has non-WebAuthn MFA methods, skipping custom domain credential check")
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetMFAStatusForLoginGET returns MFA status for admin login via GET
func (h *AdminWebAuthnHandler) GetMFAStatusForLoginGET(c *gin.Context) {
	email := c.Query("email")

	if email == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email query parameter is required"})
		return
	}

	globalDB, err := h.resolveDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Get admin user by email (no tenant context)
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	if !user.MFAEnabled {
		c.JSON(http.StatusOK, gin.H{"mfa_required": false})
		return
	}

	mfaRepo := repositories.NewMFARepository(globalDB)
	methods, _ := mfaRepo.GetUserMethods(user.ID.String())

	// Check for verified custom domain for this admin's tenant
	globalRepo := repositories.NewGlobalRepository(globalDB)
	customDomain, err := globalRepo.GetVerifiedCustomDomainForTenant(user.TenantID.String())
	if err != nil {
		log.Printf("GetMFAStatusForLoginGET: Error checking custom domain: %v", err)
	}

	response := gin.H{
		"mfa_required": true,
		"methods":      methods,
	}

	// If there's a verified custom domain, include it in the response
	if customDomain != "" {
		response["custom_domain"] = customDomain

		// Check if user has WebAuthn as an MFA method
		hasWebAuthnMethod := false
		for _, method := range methods {
			if method.MethodType == "webauthn" {
				hasWebAuthnMethod = true
				break
			}
		}

		// Only check for custom domain credentials if user has WebAuthn configured
		// If user has other MFA methods (TOTP, SMS, etc.), they don't need WebAuthn re-registration
		if hasWebAuthnMethod {
			// Check if user has credentials that are valid for this custom domain
			// Strategy: Check if credentials were created for this specific RP ID
			hasValidCreds, err := clientRepo.HasCredentialsForRPID(user.ID.String(), customDomain)
			if err != nil {
				log.Printf("GetMFAStatusForLoginGET: Error checking credentials for RP ID %s: %v", customDomain, err)
			} else if hasValidCreds {
				log.Printf("GetMFAStatusForLoginGET: User has credentials for custom domain RP ID: %s", customDomain)
			} else {
				log.Printf("GetMFAStatusForLoginGET: User has no credentials for custom domain RP ID: %s", customDomain)
				// User has WebAuthn method but no credentials for this domain - needs re-registration
				response["requires_registration"] = true
				response["message"] = "WebAuthn credentials required for custom domain. Please complete registration."
			}
		} else {
			// User doesn't have WebAuthn, they're using other MFA methods (TOTP, etc.)
			// No re-registration needed for custom domain
			log.Printf("GetMFAStatusForLoginGET: User has non-WebAuthn MFA methods, skipping custom domain credential check")
		}
	}

	c.JSON(http.StatusOK, response)
}

// BeginRegistration starts WebAuthn registration for admin users
// @Summary      Begin WebAuthn Registration for Admin Users
// @Description  Initiates WebAuthn credential registration for admin users using global database
// @Tags         WebAuthn, Admin
// @Accept       json
// @Produce      json
// @Param        request body object{email=string} true "Registration initiation data"
// @Success      200 {object} object{} "WebAuthn credential creation options"
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/admin/beginRegistration [post]
func (h *AdminWebAuthnHandler) BeginRegistration(c *gin.Context) {
	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email is required"})
		return
	}

	globalDB, err := h.resolveDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Get admin user by email
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	// Create WebAuthn user wrapper
	webAuthnUser := &WebAuthnUser{User: user}

	// Get existing credentials for this admin user
	credentials, err := clientRepo.GetCredentialsByClientID(user.ID.String())
	if err != nil {
		log.Printf("BeginRegistration: Error getting credentials - %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to load credentials"})
		return
	}

	// Convert to WebAuthn credentials
	webauthnCredentials := make([]webauthn.Credential, len(credentials))
	for i, cred := range credentials {
		webauthnCredentials[i] = webauthn.Credential{
			ID:              cred.CredentialID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Transport:       nil, // TODO: Add transport support
		}
	}

	webAuthnUser.SetCredentials(webauthnCredentials)

	// Validate origin and get appropriate WebAuthn instance
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c)
	if err != nil {
		log.Printf("BeginRegistration: Origin validation failed - %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid origin"})
		return
	}

	// Begin WebAuthn registration
	options, sessionData, err := dynamicWebAuthn.BeginRegistration(webAuthnUser)
	if err != nil {
		log.Printf("BeginRegistration: Error beginning registration - %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to begin registration"})
		return
	}

	// Store session data
	reqID := uuid.New().String()
	challengeKey := buildChallengeKey("registration", req.Email, "admin") // Use "admin" as tenant
	if err := h.SessionManager.Save(challengeKey, sessionData); err != nil {
		log.Printf("[%s] BeginRegistration: Failed to save session - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to save session"})
		return
	}

	c.JSON(http.StatusOK, options)
}

// FinishRegistration completes WebAuthn registration for admin users
// @Summary      Finish WebAuthn Registration for Admin Users
// @Description  Completes WebAuthn credential registration for admin users and stores in global database
// @Tags         WebAuthn, Admin
// @Accept       json
// @Produce      json
// @Param        request body object{email=string,credential=string} true "Registration completion data"
// @Success      200 {object} object{success=bool,credential_id=string,message=string} "Registration completion response"
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/admin/finishRegistration [post]
func (h *AdminWebAuthnHandler) FinishRegistration(c *gin.Context) {
	var req struct {
		Email      string          `json:"email" binding:"required"`
		Credential json.RawMessage `json:"credential" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email and credential are required"})
		return
	}

	reqID := uuid.New().String()
	log.Printf("[%s] FinishRegistration: Starting for email=%s", reqID, req.Email)
	log.Printf("[%s] FinishRegistration: Received credential data length: %d bytes", reqID, len(req.Credential))

	// Log credential data preview (first 200 characters or less)
	previewLen := len(req.Credential)
	if previewLen > 200 {
		previewLen = 200
	}
	log.Printf("[%s] FinishRegistration: Credential data preview: %s", reqID, string(req.Credential[:previewLen]))

	globalDB, err := h.resolveDB()
	if err != nil {
		log.Printf("[%s] FinishRegistration: DB connect failed - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Get admin user
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		log.Printf("[%s] FinishRegistration: User not found - %v", reqID, err)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	// Get session data
	challengeKey := buildChallengeKey("registration", req.Email, "admin")
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] FinishRegistration: No session found for key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no registration session"})
		return
	}

	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] FinishRegistration: Invalid session data type", reqID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session"})
		return
	}

	// Create WebAuthn user wrapper
	webAuthnUser := &WebAuthnUser{User: user}

	// Load existing credentials
	existingCreds, _ := clientRepo.GetCredentialsByClientID(user.ID.String())
	webauthnCredentials := make([]webauthn.Credential, len(existingCreds))
	for i, cred := range existingCreds {
		webauthnCredentials[i] = webauthn.Credential{
			ID:              cred.CredentialID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
		}
	}
	webAuthnUser.SetCredentials(webauthnCredentials)

	// Validate origin and get appropriate WebAuthn instance
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c)
	if err != nil {
		log.Printf("[%s] FinishRegistration: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid origin"})
		return
	}

	// Set up request body for go-webauthn library
	c.Request.Body = io.NopCloser(bytes.NewReader(req.Credential))
	c.Request.ContentLength = int64(len(req.Credential))
	c.Request.Header.Set("Content-Type", "application/json")

	// Log request details for debugging
	log.Printf("[%s] FinishRegistration: About to call WebAuthn FinishRegistration", reqID)
	log.Printf("[%s] FinishRegistration: Request Content-Type: %s", reqID, c.Request.Header.Get("Content-Type"))
	log.Printf("[%s] FinishRegistration: Request Content-Length: %d", reqID, c.Request.ContentLength)
	log.Printf("[%s] FinishRegistration: WebAuthn user ID: %x", reqID, webAuthnUser.WebAuthnID())
	log.Printf("[%s] FinishRegistration: Session challenge: %x", reqID, sessionDataTyped.Challenge)

	// Finish WebAuthn registration
	credential, err := dynamicWebAuthn.FinishRegistration(webAuthnUser, *sessionDataTyped, c.Request)
	if err != nil {
		log.Printf("[%s] FinishRegistration: WebAuthn finish failed - %v", reqID, err)
		log.Printf("[%s] FinishRegistration: Error type: %T", reqID, err)

		if strings.Contains(err.Error(), "attestation") ||
			strings.Contains(err.Error(), "Invalid attestation") ||
			strings.Contains(err.Error(), "format") ||
			strings.Contains(err.Error(), "validation") {

			log.Printf("[%s] FinishRegistration: Attempting fallback credential creation", reqID)

			var container CredentialContainer
			if unmarshalErr := json.Unmarshal(req.Credential, &container); unmarshalErr != nil {
				log.Printf("[%s] FinishRegistration: Failed to parse credential payload for fallback: %v", reqID, unmarshalErr)
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid credential payload"})
				return
			}

			fallbackCred, fbErr := buildFallbackCredentialFromContainer(&container)
			if fbErr != nil {
				log.Printf("[%s] FinishRegistration: Fallback credential creation failed: %v", reqID, fbErr)
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid attestation/public key"})
				return
			}

			log.Printf("[%s] FinishRegistration: Fallback credential created successfully", reqID)
			credential = fallbackCred
		} else {
			if protoErr, ok := err.(*protocol.Error); ok {
				log.Printf("[%s] FinishRegistration: Protocol error type: %s, details: %s", reqID, protoErr.Type, protoErr.Details)
			}
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "registration failed"})
			return
		}
	}

	if credential == nil {
		log.Printf("[%s] FinishRegistration: credential is nil after WebAuthn processing", reqID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "credential creation failed"})
		return
	}

	if len(credential.ID) == 0 {
		log.Printf("[%s] FinishRegistration: credential ID is empty", reqID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "credential ID cannot be empty"})
		return
	}
	if len(credential.PublicKey) == 0 {
		log.Printf("[%s] FinishRegistration: credential public key is empty", reqID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "credential public key cannot be empty"})
		return
	}

	attestationType := credential.AttestationType
	if attestationType == "" {
		log.Printf("[%s] FinishRegistration: Empty attestation type, defaulting to 'none'", reqID)
		attestationType = "none"
	}

	// Extract RP ID from the WebAuthn config to store with credential
	rpID := dynamicWebAuthn.Config.RPID
	log.Printf("[%s] FinishRegistration: Storing credential with RP ID: %s", reqID, rpID)

	// Save credential
	cred := repositories.Credential{
		ID:              uuid.New(),
		ClientID:        user.ID, // Use user.ID for admin users
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: attestationType,
		SignCount:       int64(credential.Authenticator.SignCount),
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		RPID:            &rpID, // Store the RP ID
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Handle AAGUID
	if len(credential.Authenticator.AAGUID) == 16 {
		if parsed, err := uuid.FromBytes(credential.Authenticator.AAGUID); err == nil {
			cred.AAGUID = &parsed
		}
	}

	if err := clientRepo.SaveCredential(&cred); err != nil {
		log.Printf("[%s] FinishRegistration: Failed to save credential - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save credential"})
		return
	}

	// Enable MFA method
	mfaRepo := repositories.NewMFARepository(globalDB)
	if err := mfaRepo.EnableMethod(user.ID.String(), "webauthn", map[string]interface{}{
		"credential_id": fmt.Sprintf("%x", credential.ID),
	}, user.ID); err != nil {
		log.Printf("[%s] FinishRegistration: Failed to enable MFA method - %v", reqID, err)
	}

	// Update user MFA settings
	methods := pq.StringArray{"webauthn"}
	updateSQL := `
		UPDATE users
		SET mfa_enabled = true,
		    mfa_verified = true,
		    mfa_default_method = 'webauthn',
		    mfa_method = $1,
		    updated_at = $2
		WHERE id = $3`

	if err := globalDB.Exec(updateSQL, methods, time.Now(), user.ID).Error; err != nil {
		log.Printf("[%s] FinishRegistration: Failed to update MFA flags - %v", reqID, err)
	}

	// Cleanup session
	h.SessionManager.Delete(challengeKey)

	// Audit log for successful WebAuthn registration
	middleware.AuditAuthentication(c, user.ID.String(), "webauthn", "register", true, map[string]interface{}{
		"credential_id": fmt.Sprintf("%x", credential.ID),
		"user_type":     "admin",
	})

	log.Printf("[%s] FinishRegistration: Completed successfully", reqID)
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"credential_id": fmt.Sprintf("%x", credential.ID),
	})
}

// BeginAuthentication starts WebAuthn authentication for admin users
// @Summary      Begin WebAuthn Authentication for Admin Users
// @Description  Initiates WebAuthn authentication (login) for admin users using global database
// @Tags         WebAuthn, Admin
// @Accept       json
// @Produce      json
// @Param        request body object{email=string} true "Authentication initiation data"
// @Success      200 {object} object{} "WebAuthn assertion options"
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/admin/beginAuthentication [post]
func (h *AdminWebAuthnHandler) BeginAuthentication(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] BeginAuthentication: START", reqID)

	var req struct {
		Email string `json:"email" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] BeginAuthentication: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email is required"})
		return
	}

	globalDB, err := h.resolveDB()
	if err != nil {
		log.Printf("[%s] BeginAuthentication: DB connect failed - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Get admin user by email
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		log.Printf("[%s] BeginAuthentication: user not found email=%s err=%v", reqID, req.Email, err)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}

	// Create WebAuthn user wrapper
	webAuthnUser := &WebAuthnUser{User: user}

	// Load existing credentials
	existingCreds, err := clientRepo.GetCredentialsByClientID(user.ID.String())
	if err != nil {
		log.Printf("[%s] BeginAuthentication: failed to load credentials: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load credentials"})
		return
	}
	log.Printf("[%s] BeginAuthentication: loaded %d credentials for user %s", reqID, len(existingCreds), user.Email)

	// Convert to WebAuthn credentials
	var creds []webauthn.Credential
	for _, dbCred := range existingCreds {
		creds = append(creds, webauthn.Credential{
			ID:              dbCred.CredentialID,
			PublicKey:       dbCred.PublicKey,
			AttestationType: dbCred.AttestationType,
			Authenticator: webauthn.Authenticator{
				SignCount: uint32(dbCred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: dbCred.BackupEligible,
				BackupState:    dbCred.BackupState,
			},
		})
	}
	webAuthnUser.SetCredentials(creds)

	// Validate origin and get appropriate WebAuthn instance
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c)
	if err != nil {
		log.Printf("[%s] BeginAuthentication: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid origin"})
		return
	}

	// Begin WebAuthn authentication
	options, sessionData, err := dynamicWebAuthn.BeginLogin(webAuthnUser)
	if err != nil {
		log.Printf("[%s] BeginAuthentication: BeginLogin failed: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to begin authentication"})
		return
	}

	// Save session data
	challengeKey := buildChallengeKey("authentication", req.Email, "admin")
	if err := h.SessionManager.Save(challengeKey, sessionData); err != nil {
		log.Printf("[%s] BeginAuthentication: Failed to save session - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to save session"})
		return
	}

	log.Printf("[%s] BeginAuthentication: session saved for email=%s", reqID, req.Email)
	c.JSON(http.StatusOK, options)
}

// FinishAuthentication completes WebAuthn authentication for admin users
// @Summary      Finish WebAuthn Authentication for Admin Users
// @Description  Completes WebAuthn authentication (login) for admin users and validates credentials
// @Tags         WebAuthn, Admin
// @Accept       json
// @Produce      json
// @Param        request body object{email=string,response=object} true "Authentication completion data"
// @Success      200 {object} object{success=bool,user_id=string} "Authentication success response"
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/admin/finishAuthentication [post]
func (h *AdminWebAuthnHandler) FinishAuthentication(c *gin.Context) {
	var req struct {
		Email      string          `json:"email" binding:"required"`
		Credential json.RawMessage `json:"credential" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email and credential are required"})
		return
	}

	reqID := uuid.New().String()
	log.Printf("[%s] FinishAuthentication: START", reqID)
	log.Printf("[%s] FinishAuthentication: Received credential data length: %d bytes", reqID, len(req.Credential))

	globalDB, err := h.resolveDB()
	if err != nil {
		log.Printf("[%s] FinishAuthentication: DB connect failed - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "db connect failed"})
		return
	}

	// Load session
	challengeKey := buildChallengeKey("authentication", req.Email, "admin")
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] FinishAuthentication: no session found for key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no authentication session"})
		return
	}

	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] FinishAuthentication: invalid session data type", reqID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session"})
		return
	}

	// Load user from DB
	clientRepo := repositories.NewClientRepository(globalDB)
	user, err := clientRepo.GetClientByEmail(req.Email)
	if err != nil {
		log.Printf("[%s] FinishAuthentication: user lookup failed for email=%s: %v", reqID, req.Email, err)
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "user not found"})
		return
	}

	// Load existing credentials
	existingCreds, err := clientRepo.GetCredentialsByClientID(user.ID.String())
	if err != nil {
		log.Printf("[%s] FinishAuthentication: failed to load credentials: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load credentials"})
		return
	}
	log.Printf("[%s] FinishAuthentication: loaded %d credentials for user %s", reqID, len(existingCreds), user.Email)

	var creds []webauthn.Credential
	for _, dbCred := range existingCreds {
		creds = append(creds, webauthn.Credential{
			ID:              dbCred.CredentialID,
			PublicKey:       dbCred.PublicKey,
			AttestationType: dbCred.AttestationType,
			Authenticator: webauthn.Authenticator{
				SignCount: uint32(dbCred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: dbCred.BackupEligible,
				BackupState:    dbCred.BackupState,
			},
		})
	}

	// Complete authentication
	webauthnUser := &WebAuthnUser{User: user}
	webauthnUser.SetCredentials(creds)

	// Validate origin and get appropriate WebAuthn instance
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c)
	if err != nil {
		log.Printf("[%s] FinishAuthentication: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid origin"})
		return
	}

	// Set up request body for go-webauthn library
	c.Request.Body = io.NopCloser(bytes.NewReader(req.Credential))
	c.Request.ContentLength = int64(len(req.Credential))
	c.Request.Header.Set("Content-Type", "application/json")

	// Add detailed logging for debugging parse errors
	log.Printf("[%s] FinishAuthentication: About to call FinishLogin for user %s", reqID, user.Email)
	log.Printf("[%s] FinishAuthentication: User has %d credentials", reqID, len(creds))
	log.Printf("[%s] FinishAuthentication: Request Content-Type: %s", reqID, c.Request.Header.Get("Content-Type"))
	log.Printf("[%s] FinishAuthentication: Request Content-Length: %d", reqID, c.Request.ContentLength)
	log.Printf("[%s] FinishAuthentication: Session challenge: %x", reqID, sessionDataTyped.Challenge)
	log.Printf("[%s] FinishAuthentication: Session UserID: %x", reqID, sessionDataTyped.UserID)

	credential, err := dynamicWebAuthn.FinishLogin(webauthnUser, *sessionDataTyped, c.Request)
	if err != nil {
		log.Printf("[%s] FinishAuthentication failed: %v", reqID, err)

		// Check for specific protocol error types
		if protocolErr, ok := err.(*protocol.Error); ok {
			log.Printf("[%s] FinishAuthentication: Protocol error type: %s, details: %s", reqID, protocolErr.Type, protocolErr.Details)
		}

		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "authentication failed"})
		return
	}

	// Update credential metadata
	if err := clientRepo.UpdateCredentialSignCount(credential.ID, credential.Authenticator.SignCount); err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating sign count for user=%s: %v", reqID, user.Email, err)
	}
	if err := clientRepo.UpdateCredentialFlags(credential.ID, credential.Flags.BackupEligible, credential.Flags.BackupState); err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating credential flags for user=%s: %v", reqID, user.Email, err)
	}

	// Update MFA last_used_at
	mfaRepo := repositories.NewMFARepository(globalDB)
	if err := globalDB.Table("mfa_methods").
		Where("client_id = ? AND method_type = ?", user.ID, "webauthn").
		Updates(map[string]interface{}{
			"last_used_at": time.Now().UTC(),
			"updated_at":   time.Now().UTC(),
		}).Error; err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating MFA method for user=%s: %v", reqID, user.Email, err)
	}
	_ = mfaRepo // Keep reference to avoid unused variable warning

	// Update user's last_login and first-login flags in global DB
	now := time.Now().UTC()
	userUpdates := map[string]interface{}{
		"last_login":   now,
		"mfa_verified": true,
		"updated_at":   now,
	}
	if user.MFAEnrolledAt == nil || user.MFAEnrolledAt.IsZero() {
		userUpdates["mfa_enrolled_at"] = now
	}
	migrator := globalDB.Migrator()
	if migrator.HasColumn(&sharedmodels.User{}, "first_login") {
		userUpdates["first_login"] = false
	}
	if migrator.HasColumn(&sharedmodels.User{}, "is_first_login") {
		userUpdates["is_first_login"] = false
	}
	if err := globalDB.Model(&user).Updates(userUpdates).Error; err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating user state for user=%s: %v", reqID, user.Email, err)
	}

	// Cleanup session
	h.SessionManager.Delete(challengeKey)

	// Audit log for successful WebAuthn authentication
	middleware.AuditAuthentication(c, user.ID.String(), "webauthn", "authenticate", true, map[string]interface{}{
		"credential_id": fmt.Sprintf("%x", credential.ID),
		"user_type":     "admin",
		"email":         user.Email,
	})

	log.Printf("[%s] FinishAuthentication: COMPLETED SUCCESSFULLY", reqID)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"user_id": user.ID.String(),
		"email":   user.Email,
	})
}
