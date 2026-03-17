package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/fxamacker/cbor/v2"

	// ✅ Use sharedmodels for DB structs
	"github.com/authsec-ai/authsec/config"
	adminCtrl "github.com/authsec-ai/authsec/controllers/admin"
	middleware "github.com/authsec-ai/authsec/middlewares"
	appmodels "github.com/authsec-ai/authsec/models"
	repositories "github.com/authsec-ai/authsec/repository"
	util "github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
)

// AttestationObject represents decoded CBOR structure
type AttestationObject struct {
	Fmt      string                 `cbor:"fmt"`
	AuthData []byte                 `cbor:"authData"`
	AttStmt  map[string]interface{} `cbor:"attStmt"`
}

type FinishRegistrationRequest struct {
	TenantID   string              `json:"tenant_id" binding:"required"`
	Email      string              `json:"email" binding:"required,email"`
	Credential CredentialContainer `json:"credential"`
}

// AttestationObject matches the WebAuthn CBOR structure

// AuthData layout per spec (simplified)
type AuthData struct {
	RPIDHash  []byte
	Flags     byte
	SignCount uint32
	AttData   *AttestedCredentialData
}

// AttestedCredentialData is part of AuthData
type AttestedCredentialData struct {
	AAGUID              []byte
	CredentialID        []byte
	CredentialPublicKey []byte
}

// parsePublicKeyFromAttestation decodes the attestationObject and extracts COSE PublicKey
func parsePublicKeyFromAttestation(attestationB64 string) ([]byte, error) {
	// Decode base64
	attBytes, err := base64.RawURLEncoding.DecodeString(attestationB64)
	if err != nil {
		attBytes, err = base64.StdEncoding.DecodeString(attestationB64)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode attestation: %w", err)
		}
	}

	// Unmarshal CBOR attestation object
	var attObj AttestationObject
	if err := cbor.Unmarshal(attBytes, &attObj); err != nil {
		return nil, fmt.Errorf("failed to CBOR unmarshal attestation: %w", err)
	}

	// Parse authData bytes manually
	if len(attObj.AuthData) < 55 {
		return nil, fmt.Errorf("authData too short")
	}

	aaguid := attObj.AuthData[37:53] // 16 bytes
	credIDLen := binary.BigEndian.Uint16(attObj.AuthData[53:55])
	credID := attObj.AuthData[55 : 55+credIDLen]
	pubKeyBytes := attObj.AuthData[55+credIDLen:]

	log.Printf("Extracted CredentialID=%x", credID)
	log.Printf("Extracted COSE PublicKey=%x", pubKeyBytes)
	log.Printf("Extracted AAGUID=%x", aaguid)

	return pubKeyBytes, nil
}

// CredentialContainer is shared across registration and authentication flows
type CredentialContainer struct {
	ID       string             `json:"id"`
	RawID    string             `json:"rawId"`
	Type     string             `json:"type"`
	Response CredentialResponse `json:"response"`
}

// CredentialResponse covers both registration and authentication payloads.
// Fields not used in one flow will just remain empty.
type CredentialResponse struct {
	// Registration-specific
	AttestationObject string `json:"attestationObject,omitempty"`
	// Authentication-specific
	AuthenticatorData string `json:"authenticatorData,omitempty"`
	Signature         string `json:"signature,omitempty"`
	UserHandle        string `json:"userHandle,omitempty"`
	// Shared
	ClientDataJSON string `json:"clientDataJSON"`
}

var reqBody struct {
	Email      string              `json:"email"`
	TenantID   string              `json:"tenant_id"`
	Credential CredentialContainer `json:"credential"`
}

type AuthenticatorResponse struct {
	AttestationObject string `json:"attestationObject"`
	ClientDataJSON    string `json:"clientDataJSON"`
}

// buildChallengeKey combines operation, email, and tenantID to index session maps
func buildChallengeKey(operation, email, tenantID string) string {
	return operation + ":" + email + ":" + tenantID
}

// convertUUIDToBytes converts UUID pointer to byte slice
func convertUUIDToBytes(aaguid *uuid.UUID) []byte {
	if aaguid == nil {
		return nil
	}
	return (*aaguid)[:]
}

// resolveDB provides simple database connection using direct config calls
func (h *WebAuthnHandler) resolveDB(tenantID string) *gorm.DB {
	// Try GlobalDB first
	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		log.Printf("resolveDB: Failed to connect to global DB - %v", err)
		return nil
	}
	return globalDB
}

// resolveDBWithError provides database connection with error handling
func (h *WebAuthnHandler) resolveDBWithError(c *gin.Context, tenantID, reqID string) (*gorm.DB, bool, error) {
	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		log.Printf("[%s] resolveDBWithError: Failed to connect to global DB - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Database connection failed"})
		return nil, false, err
	}
	return globalDB, true, nil
}

// validateOriginAndCreateWebAuthn validates the origin and creates a WebAuthn instance
func (h *WebAuthnHandler) validateOriginAndCreateWebAuthn(c *gin.Context, tenantID string) (*webauthn.WebAuthn, error) {
	// Get the origin from the request
	requestOrigin := c.Request.Header.Get("Origin")
	log.Printf("validateOriginAndCreateWebAuthn: Request origin=%s, tenantID=%s", requestOrigin, tenantID)

	// Check for custom domain FIRST before standard validation
	// Custom Domain Check - check global DB for verified custom domains
	u, err := url.Parse(requestOrigin)
	if err == nil && u.Scheme == "https" {
		host := u.Host
		// Strip port
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		// Connect to global DB to check tenant_domains table
		db, err := config.ConnectGlobalDB()
		if err == nil {
			var count int64
			// Check if domain exists and is verified
			whereClause := "domain = ? AND is_verified = true"
			args := []interface{}{host}

			// If tenantID is provided, also check tenant ownership
			if tenantID != "" {
				whereClause += " AND tenant_id = ?"
				args = append(args, tenantID)
			}

			if err := db.Table("tenant_domains").Where(whereClause, args...).Count(&count).Error; err == nil && count > 0 {
				log.Printf("validateOriginAndCreateWebAuthn: Valid custom domain %s (tenant: %s)", host, tenantID)
				return config.SetupWebAuthn("AuthSec MFA Service", host, requestOrigin), nil
			} else if err != nil {
				log.Printf("validateOriginAndCreateWebAuthn: Custom domain lookup failed for host=%s tenant=%s: %v", host, tenantID, err)
			} else {
				log.Printf("validateOriginAndCreateWebAuthn: Domain %s not found in tenant_domains or not verified", host)
			}
		} else {
			log.Printf("validateOriginAndCreateWebAuthn: Failed to connect to global DB: %v", err)
		}
	}

	// Second, validate origin using the proper subdomain validation function
	if config.ValidateSubdomainOrigin(requestOrigin) {
		log.Printf("validateOriginAndCreateWebAuthn: Creating dynamic WebAuthn for valid origin %s", requestOrigin)

		dynamicWebAuthn := config.SetupWebAuthn(
			"AuthSec MFA Service",
			"app.authsec.dev",
			requestOrigin, // Use the actual request origin
		)
		return dynamicWebAuthn, nil
	}

	// Origin validation failed
	log.Printf("validateOriginAndCreateWebAuthn: Origin validation failed for %s", requestOrigin)
	return nil, fmt.Errorf("invalid origin: %s not allowed", requestOrigin)
}

// readAndCloneBody reads the request body and allows it to be read again
func readAndCloneBody(c *gin.Context) ([]byte, error) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

// extractChallengeFromBody extracts the challenge from the credential response
// extractChallengeFromClientData ensures the challenge matches sessionData.Challenge exactly
func extractChallengeFromClientData(base64urlClientData string) (string, error) {
	if base64urlClientData == "" {
		return "", errors.New("clientDataJSON missing")
	}
	raw, err := base64.RawURLEncoding.DecodeString(base64urlClientData)
	if err != nil {
		return "", fmt.Errorf("failed to decode clientDataJSON: %w", err)
	}
	var clientData struct {
		Challenge string `json:"challenge"`
	}
	if err := json.Unmarshal(raw, &clientData); err != nil {
		return "", fmt.Errorf("failed to unmarshal clientDataJSON: %w", err)
	}
	// IMPORTANT: return challenge as-is, not decoded/re-encoded
	return clientData.Challenge, nil
}

// logProtocolError logs WebAuthn protocol errors with details
func logProtocolError(operation string, err error) {
	log.Printf("[WebAuthn] %s error: %v", operation, err)
}

// ---------- Finish Biometric Verify ----------
// @Summary      Finish Biometric Verification
// @Description  Completes biometric login verification using WebAuthn
// @Tags         Biometric
// @Accept       json
// @Produce      json
// @Param        request body FinishAuthenticationRequest true "Biometric verification data"
// @Success      200 {object} AuthenticationSuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      401 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/biometric/finishVerify [post]
// FinishBiometricVerify completes biometric login verification using WebAuthn

func (h *WebAuthnHandler) FinishBiometricVerify(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] FinishBiometricVerify: START", reqID)

	// Step 1: Parse request
	var req struct {
		Email      string              `json:"email"`
		TenantID   string              `json:"tenant_id"`
		Credential CredentialContainer `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] FinishBiometricVerify: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	if req.Email == "" || req.TenantID == "" || req.Credential.RawID == "" {
		log.Printf("[%s] FinishBiometricVerify: missing required fields", reqID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "email, tenant_id and credential required"})
		return
	}

	// Step 2: Resolve DB
	db := h.resolveDB(req.TenantID)
	if db == nil {
		log.Printf("[%s] FinishBiometricVerify: failed to resolve DB for tenant=%s", reqID, req.TenantID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "database resolution failed"})
		return
	}

	// Step 3: Load user using UserWithJSONMFAMethods to handle JSONB mfa_method field
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userWithJSONMFA).Error; err != nil {
		log.Printf("[%s] FinishBiometricVerify: user not found email=%s tenant=%s", reqID, req.Email, req.TenantID)
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "user not found"})
		return
	}
	user := userWithJSONMFA.ToShared()

	// Step 4: Build WebAuthn user
	client := &WebAuthnUser{User: &user}
	clientRepo := repositories.NewClientRepository(db)

	// Load existing credentials
	existingCreds, _ := clientRepo.GetCredentialsByClientID(user.ID.String())
	var creds []webauthn.Credential
	for _, dbCred := range existingCreds {
		var transports []protocol.AuthenticatorTransport
		for _, t := range dbCred.Transports {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              dbCred.CredentialID,
			PublicKey:       dbCred.PublicKey,
			AttestationType: dbCred.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    nil,
				SignCount: uint32(dbCred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: dbCred.BackupEligible,
				BackupState:    dbCred.BackupState,
			},
		})
	}
	client.SetCredentials(creds)

	// Step 5: Retrieve session
	challengeKey := buildChallengeKey("biometric", req.Email, req.TenantID)
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] FinishBiometricVerify: no session found for key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no biometric session"})
		return
	}

	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] FinishBiometricVerify: session data is not *webauthn.SessionData: %T", reqID, sessionData)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session data"})
		return
	}

	// Step 6: Marshal credential for go-webauthn
	credBody, _ := json.Marshal(req.Credential)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(credBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.ContentLength = int64(len(credBody))

	// Step 7: Get dynamic WebAuthn instance for this origin
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c, req.TenantID)
	if err != nil {
		log.Printf("[%s] FinishBiometricVerify: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid origin"})
		return
	}

	// Step 8: Run FinishLogin
	credential, err := dynamicWebAuthn.FinishLogin(client, *sessionDataTyped, c.Request)
	if err != nil {
		log.Printf("[%s] FinishBiometricVerify: FinishLogin failed: %v", reqID, err)
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "biometric verification failed"})
		return
	}

	// Step 9: Update credential sign count
	if err := clientRepo.UpdateCredentialSignCount(credential.ID, credential.Authenticator.SignCount); err != nil {
		log.Printf("[%s] FinishBiometricVerify: failed updating sign count: %v", reqID, err)
	}
	if err := clientRepo.UpdateCredentialFlags(credential.ID, credential.Flags.BackupEligible, credential.Flags.BackupState); err != nil {
		log.Printf("[%s] FinishBiometricVerify: failed updating credential flags: %v", reqID, err)
	}

	// Step 10: Cleanup session
	h.SessionManager.Delete(challengeKey)

	// Step 11: Update user MFA verification
	updates := map[string]interface{}{
		"mfa_verified": true,
		"updated_at":   time.Now().UTC(),
		"last_login":   time.Now().UTC(),
	}
	if user.MFAEnrolledAt == nil || user.MFAEnrolledAt.IsZero() {
		updates["mfa_enrolled_at"] = time.Now().UTC()
	}
	if err := db.Model(&user).Updates(updates).Error; err != nil {
		log.Printf("[%s] FinishBiometricVerify: failed to update MFA flags: %v", reqID, err)
	}

	// Audit log for successful biometric verification
	middleware.AuditAuthentication(c, user.ID.String(), "webauthn", "biometric_verify", true, map[string]interface{}{
		"credential_id": hex.EncodeToString(credential.ID),
		"tenant_id":     req.TenantID,
	})

	// Step 11: Success response
	log.Printf("[%s] FinishBiometricVerify: SUCCESS email=%s tenant=%s credentialID=%s",
		reqID, req.Email, req.TenantID, hex.EncodeToString(credential.ID))
	c.JSON(http.StatusOK, AuthenticationSuccessResponse{
		Success:      true,
		Message:      "Biometric verification successful",
		CredentialID: hex.EncodeToString(credential.ID),
	})
}

// --- Begin Registration ---

func (h *WebAuthnHandler) BeginRegistration(c *gin.Context) {
	var req BeginRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("BeginRegistration: Bad request - %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	log.Printf("BeginRegistration: Starting for email=%s, tenantID=%s", req.Email, req.TenantID)

	// Validate required fields
	if req.Email == "" || req.TenantID == "" {
		log.Printf("BeginRegistration: Missing required fields")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and tenantID are required"})
		return
	}

	// Try GlobalDB first
	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		log.Printf("BeginRegistration: Failed to connect to global DB - %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
		return
	}

	clientRepo := repositories.NewClientRepository(globalDB)
	client, err := clientRepo.GetClientByEmailAndTenantForLogin(req.Email, req.TenantID)

	// If not found in GlobalDB, try TenantDB
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("BeginRegistration: User not found in GlobalDB, trying TenantDB")

			// Get tenant DB name from GlobalDB
			globalRepo := repositories.NewGlobalRepository(globalDB)
			tenant, err := globalRepo.GetTenantByID(req.TenantID)
			if err != nil {
				log.Printf("BeginRegistration: Tenant not found - %v", err)
				c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
				return
			}

			// Connect to tenant DB
			tenantDB, err := config.ConnectTenantDB(tenant.TenantDB)
			if err != nil {
				log.Printf("BeginRegistration: Failed to connect to tenant DB - %v", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection failed"})
				return
			}

			// Try again with tenant DB
			clientRepo = repositories.NewClientRepository(tenantDB)
			client, err = clientRepo.GetClientByEmailAndTenantForLogin(req.Email, req.TenantID)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Printf("BeginRegistration: User not found in TenantDB either")
					c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
				} else {
					log.Printf("BeginRegistration: Database error - %v", err)
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
				}
				return
			}
		} else {
			log.Printf("BeginRegistration: Database error - %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
			return
		}
	}

	log.Printf("BeginRegistration: Found client - ID=%s, Email=%s", client.ID, client.Email)

	// Get existing credentials for this client using user.ID
	credentials, err := clientRepo.GetCredentialsByClientID(client.ID.String())
	if err != nil {
		log.Printf("BeginRegistration: Error getting credentials - %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load credentials"})
		return
	}

	log.Printf("BeginRegistration: Found %d existing credentials", len(credentials))

	// Convert to WebAuthn credentials
	webauthnCredentials := make([]webauthn.Credential, len(credentials))
	for i, cred := range credentials {
		webauthnCredentials[i] = webauthn.Credential{
			ID:              cred.CredentialID,
			PublicKey:       cred.PublicKey,
			AttestationType: cred.AttestationType,
			Authenticator: webauthn.Authenticator{
				AAGUID:    convertUUIDToBytes(cred.AAGUID),
				SignCount: uint32(cred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: cred.BackupEligible,
				BackupState:    cred.BackupState,
			},
		}

		// Handle transports if available
		if len(cred.Transports) > 0 {
			transports := make([]protocol.AuthenticatorTransport, len(cred.Transports))
			for j, transport := range cred.Transports {
				transports[j] = protocol.AuthenticatorTransport(transport)
			}
			webauthnCredentials[i].Transport = transports
		}
	}

	// Wrap client as WebAuthn user
	webauthnUser := &WebAuthnUser{User: client}
	webauthnUser.SetCredentials(webauthnCredentials)

	// Get dynamic WebAuthn instance for this origin
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c, req.TenantID)
	if err != nil {
		log.Printf("BeginRegistration: Origin validation failed - %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid origin"})
		return
	}

	// Create registration options - allow both platform and cross-platform authenticators
	options, sessionData, err := dynamicWebAuthn.BeginRegistration(webauthnUser,
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationPreferred, // Prefer user verification but don't require
			// Don't specify AuthenticatorAttachment to allow both platform and cross-platform
			ResidentKey: protocol.ResidentKeyRequirementDiscouraged, // Don't require resident keys
		}),
	)
	if err != nil {
		log.Printf("BeginRegistration: WebAuthn BeginRegistration error - %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin registration"})
		return
	}

	log.Printf("BeginRegistration: Generated challenge - %x", sessionData.Challenge)

	// Store session data
	challengeKey := buildChallengeKey("biometric_setup", req.Email, req.TenantID)
	if err := h.SessionManager.Save(challengeKey, sessionData); err != nil {
		log.Printf("BeginRegistration: Failed to save session - %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	log.Printf("BeginRegistration: Stored session with key=%s", challengeKey)

	// Return options to client
	c.JSON(http.StatusOK, options)
}

//
// --- Finish Registration ---
//
// FinishRegistrationRequest represents the frontend payload for WebAuthn registration.

// ---------- Finish WebAuthn Registration ----------
// @Summary      Finish WebAuthn Registration
// @Description  Complete WebAuthn credential registration
// @Tags         WebAuthn
// @Accept       json
// @Produce      json
// @Param        request body FinishRegistrationRequest true "Registration completion data"
// @Success      200 {object} RegistrationSuccessResponse
// @Failure      400 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/finishRegistration [post]
func safeBytes(in []byte) []byte {
	if in == nil {
		return []byte{}
	}
	return in
}

func (h *WebAuthnHandler) FinishRegistration(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] FinishRegistration: START", reqID)

	// Step 1: Validate origin dynamically
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c, "") // TenantID not parsed yet
	if err != nil {
		log.Printf("[%s] FinishRegistration: origin validation failed: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid origin"})
		return
	}

	// Step 2: Read and clone body
	body, err := readAndCloneBody(c)
	if err != nil {
		log.Printf("[%s] FinishRegistration: failed to read request body: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request body"})
		return
	}

	// Step 3: Parse typed request
	var reqBody FinishRegistrationRequest
	if err := json.Unmarshal(body, &reqBody); err != nil {
		log.Printf("[%s] FinishRegistration: JSON parse error: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request JSON"})
		return
	}

	tenantID := reqBody.TenantID
	email := reqBody.Email
	if tenantID == "" || email == "" {
		log.Printf("[%s] FinishRegistration: missing tenant_id or email", reqID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "tenant_id and email are required"})
		return
	}

	// Step 4: Resolve DB (global vs tenant)
	db, isGlobalDB, err := h.resolveDBWithError(c, tenantID, reqID)
	if err != nil {
		return
	}

	// Step 5: Load user using UserWithJSONMFAMethods to handle JSONB mfa_method field
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if isGlobalDB {
		if err := db.Scopes(util.WithUsersMFAMethodArray).Where("email = ?", email).First(&userWithJSONMFA).Error; err != nil {
			log.Printf("[%s] FinishRegistration: user not found in GlobalDB: %v", reqID, err)
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
	} else {
		if err := db.Scopes(util.WithUsersMFAMethodArray).
			Where("email = ? AND tenant_id = ?", email, tenantID).First(&userWithJSONMFA).Error; err != nil {
			log.Printf("[%s] FinishRegistration: user not found in TenantDB: %v", reqID, err)
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
			return
		}
	}

	// Use the embedded User for compatibility
	user := userWithJSONMFA.ToShared()

	// Step 6: Ensure ClientID exists
	if user.ClientID == uuid.Nil {
		user.ClientID = uuid.New()
		if err := db.Save(&user).Error; err != nil {
			log.Printf("[%s] FinishRegistration: failed to persist new ClientID: %v", reqID, err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to persist ClientID"})
			return
		}
	}

	client := &WebAuthnUser{User: &user}

	// Step 7: Recover session
	challengeKey := buildChallengeKey("biometric_setup", email, tenantID)
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] FinishRegistration: session not found key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "webauthn session not found"})
		return
	}
	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] FinishRegistration: invalid session data type: %T", reqID, sessionData)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session data"})
		return
	}

	// Step 8: Marshal credential into request for go-webauthn
	credBody, err := json.Marshal(reqBody.Credential)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid credential payload"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(credBody))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.ContentLength = int64(len(credBody))

	// Step 9: Try FinishRegistration
	credential, err := dynamicWebAuthn.FinishRegistration(client, *sessionDataTyped, c.Request)
	if err != nil {
		log.Printf("[%s] FinishRegistration: WebAuthn FinishRegistration failed: %v", reqID, err)

		// More comprehensive error handling - handle attestation, validation, and format errors
		if strings.Contains(err.Error(), "attestation") ||
			strings.Contains(err.Error(), "Invalid attestation") ||
			strings.Contains(err.Error(), "format") ||
			strings.Contains(err.Error(), "validation") {

			log.Printf("[%s] FinishRegistration: Attempting fallback credential creation", reqID)

			fallbackCred, fbErr := buildFallbackCredentialFromContainer(&reqBody.Credential)
			if fbErr != nil {
				log.Printf("[%s] FinishRegistration: Fallback credential creation failed: %v", reqID, fbErr)
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid attestation/public key"})
				return
			}

			log.Printf("[%s] FinishRegistration: Fallback credential created successfully", reqID)
			credential = fallbackCred
		} else {
			log.Printf("[%s] FinishRegistration: Non-attestation WebAuthn error: %v", reqID, err)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "WebAuthn registration failed"})
			return
		}
	}

	// Validate credential is not nil before proceeding
	if credential == nil {
		log.Printf("[%s] FinishRegistration: credential is nil after WebAuthn processing", reqID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "credential creation failed"})
		return
	}

	// Step 10: Persist credential
	var transports pq.StringArray
	for _, t := range credential.Transport {
		transports = append(transports, string(t))
	}

	// Validate credential data before saving
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

	// Ensure AttestationType is not empty (database constraint)
	attestationType := credential.AttestationType
	if attestationType == "" {
		attestationType = "none"
		log.Printf("[%s] FinishRegistration: Empty attestation type, defaulting to 'none'", reqID)
	}

	log.Printf("[%s] FinishRegistration: Attempting to save credential with ID=%x, AttestationType=%s",
		reqID, credential.ID, attestationType)

	// Use repository method to save credential with proper column mappings
	credRepo := repositories.NewCredentialRepository(db)
	if err := credRepo.AddCredential(client.ID.String(), credential); err != nil {
		log.Printf("[%s] FinishRegistration: failed to save credential: %v", reqID, err)
		log.Printf("[%s] FinishRegistration: credential details - ClientID=%s, CredentialID=%x, AttestationType=%s",
			reqID, client.ID, credential.ID, attestationType)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save credential"})
		return
	}

	log.Printf("[%s] FinishRegistration: Credential saved successfully", reqID)

	// Step 11: Enable MFA
	mfaRepo := repositories.NewMFARepository(db)
	_ = mfaRepo.EnableMethod(client.ID.String(), "webauthn",
		map[string]interface{}{"attestation_type": credential.AttestationType}, user.ID)

	// Step 12: Update MFA flags
	now := time.Now().UTC()
	methods := pq.StringArray{"webauthn"}

	// Use raw SQL to update to avoid GORM type conversion issues
	updateSQL := `
		UPDATE users 
		SET mfa_enabled = true, 
		    mfa_verified = true, 
		    mfa_default_method = 'webauthn',
		    mfa_method = $1,
		    updated_at = $2,
		    last_login = $2
		WHERE id = $3`
	if user.MFAEnrolledAt == nil || user.MFAEnrolledAt.IsZero() {
		updateSQL = `
			UPDATE users 
			SET mfa_enabled = true, 
			    mfa_verified = true, 
			    mfa_default_method = 'webauthn',
			    mfa_method = $1,
			    mfa_enrolled_at = $2,
			    updated_at = $2,
			    last_login = $2
			WHERE id = $3`
		if err := db.Exec(updateSQL, methods, now, user.ID).Error; err != nil {
			log.Printf("[%s] FinishRegistration: failed to update MFA flags: %v", reqID, err)
		}
	} else {
		if err := db.Exec(updateSQL, methods, now, user.ID).Error; err != nil {
			log.Printf("[%s] FinishRegistration: failed to update MFA flags: %v", reqID, err)
		}
	}

	// Step 13: Call userflow service to register user and get tokens
	accessToken, refreshToken, err := h.callUserflowService(client.ID.String(), user.Email, user.TenantID.String())
	if err != nil {
		log.Printf("[%s] FinishRegistration: userflow service call failed: %v", reqID, err)
		// Proceed anyway, but log the error
	} else {
		log.Printf("[%s] FinishRegistration: userflow service returned accessToken=%s, refreshToken=%s",
			reqID, accessToken, refreshToken)
	}

	// Step 14: Clean up session
	h.SessionManager.Delete(challengeKey)

	// Audit log for successful WebAuthn registration
	middleware.AuditAuthentication(c, user.ID.String(), "webauthn", "register", true, map[string]interface{}{
		"credential_id": hex.EncodeToString(credential.ID),
		"tenant_id":     tenantID,
	})

	// Success
	log.Printf("[%s] FinishRegistration: SUCCESS credentialID=%s", reqID, hex.EncodeToString(credential.ID))
	c.JSON(http.StatusOK, RegistrationSuccessResponse{
		Success:      true,
		Message:      "WebAuthn registration successful",
		CredentialID: hex.EncodeToString(credential.ID),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

//
// --- Userflow Service Integration ---
//

// callUserflowService obtains JWT tokens for a user after successful WebAuthn
// registration or authentication.
//
// In the merged authsec monolith this calls the bridge package which invokes
// the user-flow controller logic directly in-process, replacing the HTTP POST
// to {AUTH_MANAGER_URL}/uflow/webauthn/register that existed when user-flow
// and webauthn-service were separate microservices.
func (h *WebAuthnHandler) callUserflowService(clientID, email, tenantID string) (accessToken, refreshToken string, err error) {
	log.Printf("callUserflowService: invoking internal controller for email=%s, tenantID=%s", email, tenantID)
	return adminCtrl.WebAuthnRegisterInternal(clientID, email, tenantID)
}

//
// --- Begin Authentication ---
//

// BeginAuthentication starts the WebAuthn authentication (login) ceremony
func (h *WebAuthnHandler) BeginAuthentication(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] BeginAuthentication: START", reqID)

	// Step 1: Bind request JSON directly
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] BeginAuthentication: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Step 2: Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	// Step 3: Lookup user using UserWithJSONMFAMethods to handle JSONB mfa_method field
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userWithJSONMFA).Error; err != nil {
		log.Printf("[%s] BeginAuthentication: user not found email=%s tenant_id=%s err=%v",
			reqID, req.Email, req.TenantID, err)
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userWithJSONMFA.ToShared()

	// Step 4: Wrap user into WebAuthnUser
	webUser := &WebAuthnUser{User: &user}
	clientRepo := repositories.NewClientRepository(db)

	// Step 5: Load existing credentials
	existingCreds, err := clientRepo.GetCredentialsByClientID(user.ID.String())
	if err != nil {
		log.Printf("[%s] BeginAuthentication: failed to load credentials: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load credentials"})
		return
	}
	log.Printf("[%s] BeginAuthentication: loaded %d credentials for user %s",
		reqID, len(existingCreds), user.Email)

	// Check if user has no credentials - needs to register first
	if len(existingCreds) == 0 {
		log.Printf("[%s] BeginAuthentication: No credentials found for user %s - registration required", reqID, user.Email)
		c.JSON(http.StatusPreconditionRequired, gin.H{
			"error":                 "no_credentials",
			"message":               "WebAuthn not configured. Please register first.",
			"registration_required": true,
		})
		return
	}

	var creds []webauthn.Credential
	for _, dbCred := range existingCreds {
		var transports []protocol.AuthenticatorTransport
		for _, t := range dbCred.Transports {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}

		// Debug credential ID formats
		log.Printf("[%s] BeginAuthentication: Processing credential - DB ID (hex)=%x", reqID, dbCred.CredentialID)
		log.Printf("[%s] BeginAuthentication: Processing credential - DB ID (base64)=%s", reqID, base64.StdEncoding.EncodeToString(dbCred.CredentialID))
		log.Printf("[%s] BeginAuthentication: Processing credential - DB ID (base64url)=%s", reqID, base64.RawURLEncoding.EncodeToString(dbCred.CredentialID))

		creds = append(creds, webauthn.Credential{
			ID:              dbCred.CredentialID,
			PublicKey:       dbCred.PublicKey,
			AttestationType: dbCred.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    nil,
				SignCount: uint32(dbCred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: dbCred.BackupEligible,
				BackupState:    dbCred.BackupState,
			},
		})
	}
	webUser.SetCredentials(creds)

	// Step 6: Get dynamic WebAuthn instance for this origin
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c, req.TenantID)
	if err != nil {
		log.Printf("[%s] BeginAuthentication: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid origin"})
		return
	}

	// Step 7: Start WebAuthn login
	options, sessionData, err := dynamicWebAuthn.BeginLogin(webUser)
	if err != nil {
		log.Printf("[%s] BeginAuthentication: BeginLogin failed: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to begin authentication"})
		return
	}

	// Step 8: Save session data
	challengeKey := buildChallengeKey("biometric", req.Email, req.TenantID)
	if err := h.SessionManager.Save(challengeKey, sessionData); err != nil {
		log.Printf("[%s] BeginAuthentication: Failed to save session - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to save session"})
		return
	}

	// Step 9: Return response
	log.Printf("[%s] BeginAuthentication: session saved for email=%s tenant_id=%s",
		reqID, req.Email, req.TenantID)
	log.Printf("[%s] BeginAuthentication: returning options: %+v", reqID, options)

	// Additional debugging for 204 issue
	if options == nil {
		log.Printf("[%s] BeginAuthentication: ERROR - options is nil!", reqID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WebAuthn options are nil"})
		return
	}

	log.Printf("[%s] BeginAuthentication: options type: %T", reqID, options)
	c.JSON(http.StatusOK, options)
}

// --- Finish Authentication ---
//
// FinishAuthentication completes WebAuthn authentication
func (h *WebAuthnHandler) FinishAuthentication(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] FinishAuthentication: START", reqID)

	// Parse request
	var req struct {
		Email    string                               `json:"email"`
		TenantID string                               `json:"tenant_id"`
		Response protocol.CredentialAssertionResponse `json:"response"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] FinishAuthentication: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// ✅ Use helper for DB switching
	db := h.resolveDB(req.TenantID)

	// Load session
	challengeKey := buildChallengeKey("biometric", req.Email, req.TenantID)
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] FinishAuthentication: no session found for key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no authentication session"})
		return
	}

	// Load user from DB
	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		log.Printf("[%s] FinishAuthentication: user lookup failed for email=%s tenant=%s: %v", reqID, req.Email, req.TenantID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	user := userRecord.ToShared()
	// Load existing credentials - this is CRITICAL for authentication to work
	clientRepo := repositories.NewClientRepository(db)
	existingCreds, err := clientRepo.GetCredentialsByClientID(user.ID.String())
	if err != nil {
		log.Printf("[%s] FinishAuthentication: failed to load credentials: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to load credentials"})
		return
	}
	log.Printf("[%s] FinishAuthentication: loaded %d credentials for user %s", reqID, len(existingCreds), user.Email)

	var creds []webauthn.Credential
	for _, dbCred := range existingCreds {
		var transports []protocol.AuthenticatorTransport
		for _, t := range dbCred.Transports {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              dbCred.CredentialID,
			PublicKey:       dbCred.PublicKey,
			AttestationType: dbCred.AttestationType,
			Transport:       transports,
			Authenticator: webauthn.Authenticator{
				AAGUID:    nil,
				SignCount: uint32(dbCred.SignCount),
			},
			Flags: webauthn.CredentialFlags{
				BackupEligible: dbCred.BackupEligible,
				BackupState:    dbCred.BackupState,
			},
		})
	}

	// Type assert to webauthn.SessionData
	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] FinishAuthentication: session data is not *webauthn.SessionData: %T", reqID, sessionData)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session data"})
		return
	}

	// Complete authentication
	webauthnUser := &WebAuthnUser{User: &user}
	webauthnUser.SetCredentials(creds)
	credential, err := h.WebAuthn.FinishLogin(webauthnUser, *sessionDataTyped, c.Request)
	if err != nil {
		log.Printf("[%s] FinishAuthentication failed: %v", reqID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
		return
	}

	if err := clientRepo.UpdateCredentialSignCount(credential.ID, credential.Authenticator.SignCount); err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating sign count for user=%s: %v", reqID, user.Email, err)
	}
	if err := clientRepo.UpdateCredentialFlags(credential.ID, credential.Flags.BackupEligible, credential.Flags.BackupState); err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating credential flags for user=%s: %v", reqID, user.Email, err)
	}
	if err := db.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ?", user.ID, "webauthn").
		Updates(map[string]interface{}{
			"last_used_at": time.Now().UTC(),
			"updated_at":   time.Now().UTC(),
		}).Error; err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating MFA method for user=%s: %v", reqID, user.Email, err)
	}

	// Update user's last_login timestamp
	if err := db.Model(&user).Update("last_login", time.Now().UTC()).Error; err != nil {
		log.Printf("[%s] FinishAuthentication: failed updating last_login for user=%s: %v", reqID, user.Email, err)
	}

	// Call userflow service to get tokens
	accessToken, refreshToken, err := h.callUserflowService(user.ID.String(), user.Email, user.TenantID.String())
	if err != nil {
		log.Printf("[%s] FinishAuthentication: userflow service call failed: %v", reqID, err)
		// Proceed anyway, but log the error
	} else {
		log.Printf("[%s] FinishAuthentication: userflow service returned accessToken=%s, refreshToken=%s",
			reqID, accessToken, refreshToken)
	}

	// Cleanup session
	h.SessionManager.Delete(challengeKey)

	// Audit log for successful WebAuthn authentication
	middleware.AuditAuthentication(c, user.ID.String(), "webauthn", "authenticate", true, map[string]interface{}{
		"credential_id": fmt.Sprintf("%x", credential.ID),
		"tenant_id":     req.TenantID,
		"email":         user.Email,
	})

	log.Printf("[%s] FinishAuthentication: COMPLETED SUCCESSFULLY", reqID)
	c.JSON(http.StatusOK, AuthenticationSuccessResponse{
		Success:      true,
		Message:      "WebAuthn authentication successful",
		CredentialID: fmt.Sprintf("%x", credential.ID),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

//
// --- Begin Biometric Setup ---
//

func (h *WebAuthnHandler) BeginBiometricSetup(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] BeginBiometricSetup: START", reqID)

	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] BeginBiometricSetup: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	req.TenantID = strings.TrimSpace(req.TenantID)
	if req.Email == "" || req.TenantID == "" {
		log.Printf("[%s] BeginBiometricSetup: missing required fields email=%q tenant_id=%q", reqID, req.Email, req.TenantID)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request: email and tenant_id are required"})
		return
	}

	// Resolve database (global vs tenant)
	db, isGlobalDB, err := h.resolveDBWithError(c, req.TenantID, reqID)
	if err != nil {
		return
	}

	var userRecord appmodels.UserWithJSONMFAMethods
	var user sharedmodels.User
	var dbErr error
	if isGlobalDB {
		// For global DB, query by email only (user might be admin user)
		dbErr = db.Scopes(util.WithUsersMFAMethodArray).Where("email = ?", req.Email).First(&userRecord).Error
		log.Printf("[%s] BeginBiometricSetup: querying GlobalDB for email=%s", reqID, req.Email)
	} else {
		// For tenant DB, query with tenant_id
		dbErr = db.Scopes(util.WithUsersMFAMethodArray).
			Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error
		log.Printf("[%s] BeginBiometricSetup: querying TenantDB for email=%s tenant=%s", reqID, req.Email, req.TenantID)
	}

	if dbErr != nil {
		if dbErr == gorm.ErrRecordNotFound {
			// Auto-create user if not found
			tenantUUID, err := uuid.Parse(req.TenantID)
			if err != nil {
				log.Printf("[%s] BeginBiometricSetup: invalid tenant ID: %v", reqID, err)
				c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid tenant ID"})
				return
			}
			user = sharedmodels.User{
				ID:        uuid.New(),
				ClientID:  uuid.New(),
				Email:     req.Email,
				TenantID:  tenantUUID,
				Provider:  "local",
				CreatedAt: time.Now().UTC(),
				UpdatedAt: time.Now().UTC(),
			}
			if err := db.Create(&user).Error; err != nil {
				log.Printf("[%s] BeginBiometricSetup: failed to create user: %v", reqID, err)
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to create user"})
				return
			}
			log.Printf("[%s] BeginBiometricSetup: auto-created user ID=%s for email=%s tenant=%s", reqID, user.ID, req.Email, req.TenantID)
		} else {
			log.Printf("[%s] BeginBiometricSetup: database error: %v", reqID, dbErr)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "database error"})
			return
		}
	} else {
		user = userRecord.ToShared()
	}
	if h.WebAuthn == nil {
		log.Printf("[%s] BeginBiometricSetup: WebAuthn instance is nil", reqID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "WebAuthn service not initialized"})
		return
	}

	webUser := &WebAuthnUser{User: &user}

	// Get dynamic WebAuthn instance for this origin
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c, req.TenantID)
	if err != nil {
		log.Printf("[%s] BeginBiometricSetup: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid origin"})
		return
	}

	// Configure options for biometric authentication - prefer platform authenticators
	options, sessionData, err := dynamicWebAuthn.BeginRegistration(webUser,
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			// AuthenticatorAttachment: protocol.Platform,                          // Prefer built-in biometrics (Relaxed for Windows compatibility)
			UserVerification: protocol.VerificationPreferred,             // Prefer biometric verification but don't require it
			ResidentKey:      protocol.ResidentKeyRequirementDiscouraged, // Don't require resident keys
		}),
	)
	if err != nil {
		log.Printf("[%s] BeginBiometricSetup: WebAuthn.BeginRegistration failed: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to begin biometric setup"})
		return
	}

	if h.SessionManager == nil {
		log.Printf("[%s] BeginBiometricSetup: SessionManager is nil", reqID)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "session service not initialized"})
		return
	}

	// Debug: Log the challenge being generated
	log.Printf("[%s] BeginBiometricSetup: Generated challenge (hex)=%x", reqID, sessionData.Challenge)
	log.Printf("[%s] BeginBiometricSetup: Generated challenge (string)=%s", reqID, sessionData.Challenge)

	challengeKey := buildChallengeKey("biometric_setup", req.Email, req.TenantID)
	if err := h.SessionManager.Save(challengeKey, sessionData); err != nil {
		log.Printf("[%s] BeginBiometricSetup: Failed to save session - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to save session"})
		return
	}

	log.Printf("[%s] BeginBiometricSetup: SUCCESS - session saved with key=%s", reqID, challengeKey)
	c.JSON(http.StatusOK, options)
}

//
// --- Confirm Biometric Setup ---
//

// ConfirmBiometricSetup completes biometric MFA registration (WebAuthn under the hood)
func (h *WebAuthnHandler) ConfirmBiometricSetup(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] ConfirmBiometricSetup: START", reqID)

	// Read and clone body (so go-webauthn can re-read it)
	body, err := readAndCloneBody(c)
	if err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: failed to read request body: %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	// Parse tenant_id, email from body (similar to FinishRegistration)
	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: JSON parse error: %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request JSON"})
		return
	}
	tenantID, _ := raw["tenant_id"].(string)
	email, _ := raw["email"].(string)
	if tenantID == "" || email == "" {
		log.Printf("[%s] ConfirmBiometricSetup: missing tenant_id or email", reqID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and email are required"})
		return
	}

	req := struct {
		Email    string
		TenantID string
	}{
		Email:    strings.TrimSpace(email),
		TenantID: strings.TrimSpace(tenantID),
	}
	if req.Email == "" || req.TenantID == "" {
		log.Printf("[%s] ConfirmBiometricSetup: trimmed email or tenant_id empty", reqID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id and email are required"})
		return
	}

	// ✅ Resolve DB dynamically
	db := h.resolveDB(req.TenantID)

	// Load session - use same key that BeginBiometricSetup used
	challengeKey := buildChallengeKey("biometric_setup", req.Email, req.TenantID)
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] ConfirmBiometricSetup: no session found for key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no registration session"})
		return
	}

	// Type assert to webauthn.SessionData
	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] ConfirmBiometricSetup: session data is not *webauthn.SessionData: %T", reqID, sessionData)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session data"})
		return
	}

	// Load the actual user from database (same as BeginBiometricSetup did)
	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: user not found email=%s tenant=%s: %v", reqID, req.Email, req.TenantID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	user := userRecord.ToShared()

	// Ensure ClientID exists (same as FinishRegistration does)
	if user.ClientID == uuid.Nil {
		user.ClientID = uuid.New()
		if err := db.Save(&user).Error; err != nil {
			log.Printf("[%s] ConfirmBiometricSetup: failed to persist new ClientID: %v", reqID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist ClientID"})
			return
		}
		log.Printf("[%s] ConfirmBiometricSetup: generated ClientID=%s", reqID, user.ClientID)
	}

	// Complete registration
	webauthnUser := &WebAuthnUser{User: &user}

	// Get dynamic WebAuthn instance for this origin
	dynamicWebAuthn, err := h.validateOriginAndCreateWebAuthn(c, req.TenantID)
	if err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: Origin validation failed - %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid origin"})
		return
	}

	// Debug: Log critical data for comparison
	log.Printf("[%s] ConfirmBiometricSetup: User ID=%s", reqID, webauthnUser.User.ID.String())
	log.Printf("[%s] ConfirmBiometricSetup: WebAuthn User ID (hex)=%x", reqID, webauthnUser.WebAuthnID())
	log.Printf("[%s] ConfirmBiometricSetup: Stored challenge (hex)=%x", reqID, sessionDataTyped.Challenge)
	log.Printf("[%s] ConfirmBiometricSetup: Challenge (string)=%s", reqID, sessionDataTyped.Challenge)

	// Extract the credential portion from the request
	credentialData, ok := raw["credential"]
	if !ok {
		log.Printf("[%s] ConfirmBiometricSetup: no credential in request", reqID)
		c.JSON(http.StatusBadRequest, gin.H{"error": "credential is required"})
		return
	}

	// Convert credential to JSON for WebAuthn library
	credentialJSON, err := json.Marshal(credentialData)
	if err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: failed to marshal credential: %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid credential format"})
		return
	}

	// Debug: Log the actual credential data being sent to WebAuthn
	log.Printf("[%s] ConfirmBiometricSetup: Credential JSON being sent to WebAuthn: %s", reqID, string(credentialJSON))

	// Try to extract and compare the challenge from the client data
	var clientCredential map[string]interface{}
	if err := json.Unmarshal(credentialJSON, &clientCredential); err == nil {
		if response, ok := clientCredential["response"].(map[string]interface{}); ok {
			if clientDataJSON, ok := response["clientDataJSON"].(string); ok {
				if clientChallenge, err := extractChallengeFromClientData(clientDataJSON); err == nil {
					log.Printf("[%s] ConfirmBiometricSetup: Client challenge=%s", reqID, clientChallenge)
					log.Printf("[%s] ConfirmBiometricSetup: Server challenge=%s", reqID, sessionDataTyped.Challenge)
					if clientChallenge != sessionDataTyped.Challenge {
						log.Printf("[%s] ConfirmBiometricSetup: ⚠️  CHALLENGE MISMATCH!", reqID)
					} else {
						log.Printf("[%s] ConfirmBiometricSetup: ✅ Challenges match", reqID)
					}
				} else {
					log.Printf("[%s] ConfirmBiometricSetup: Failed to extract client challenge: %v", reqID, err)
				}
			}
		}
	}

	// Set up the request body with just the credential data for WebAuthn library
	c.Request.Body = io.NopCloser(bytes.NewBuffer(credentialJSON))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Request.ContentLength = int64(len(credentialJSON))

	// Complete registration with WebAuthn library parsing the body directly
	log.Printf("[%s] ConfirmBiometricSetup: About to call FinishRegistration with session challenge=%s", reqID, sessionDataTyped.Challenge)
	log.Printf("[%s] ConfirmBiometricSetup: WebAuthn config - AttestationPreference: %s, Debug: %t",
		reqID, dynamicWebAuthn.Config.AttestationPreference, dynamicWebAuthn.Config.Debug)

	credential, err := dynamicWebAuthn.FinishRegistration(webauthnUser, *sessionDataTyped, c.Request)

	if err != nil {
		log.Printf("[%s] ConfirmBiometricSetup failed with detailed error: %v", reqID, err)
		log.Printf("[%s] ConfirmBiometricSetup: Error type: %T", reqID, err)

		// Check if it's an attestation-related error
		if strings.Contains(err.Error(), "attestation") || strings.Contains(err.Error(), "Invalid attestation") {
			log.Printf("[%s] ConfirmBiometricSetup: Attestation error detected. WebAuthn config: %+v", reqID, dynamicWebAuthn.Config)
			log.Printf("[%s] ConfirmBiometricSetup: Session challenge: %s", reqID, sessionDataTyped.Challenge)
			log.Printf("[%s] ConfirmBiometricSetup: RP ID: %s, Origins: %v", reqID, dynamicWebAuthn.Config.RPID, dynamicWebAuthn.Config.RPOrigins)

			// Try to provide a more helpful error message
			if strings.Contains(err.Error(), "Invalid attestation format") {
				log.Printf("[%s] ConfirmBiometricSetup: This appears to be an attestation format validation issue", reqID)
				log.Printf("[%s] ConfirmBiometricSetup: The authenticator sent an attestation format that is not supported by the WebAuthn library", reqID)
				log.Printf("[%s] ConfirmBiometricSetup: Since AttestationPreference is set to PreferNoAttestation, this should not happen", reqID)
			}
		}

		// Return more specific error information
		errorDetails := map[string]interface{}{
			"error":            "registration failed",
			"details":          err.Error(),
			"config_debug":     dynamicWebAuthn.Config.Debug,
			"attestation_pref": string(dynamicWebAuthn.Config.AttestationPreference),
			"rp_id":            dynamicWebAuthn.Config.RPID,
		}

		c.JSON(http.StatusBadRequest, errorDetails)
		return
	}
	log.Printf("[%s] ConfirmBiometricSetup: ✅ WebAuthn FinishRegistration succeeded", reqID)

	// Persist credential + MFA method
	now := time.Now().UTC()

	// Add debug logging for credential data
	log.Printf("[%s] ConfirmBiometricSetup: Credential data - ID=%x, AttestationType='%s', SignCount=%d",
		reqID, credential.ID, credential.AttestationType, credential.Authenticator.SignCount)

	// Ensure AttestationType is not empty (database constraint)
	attestationType := credential.AttestationType
	if attestationType == "" {
		attestationType = "none"
		log.Printf("[%s] ConfirmBiometricSetup: Empty attestation type, setting to 'none'", reqID)
	}

	cred := repositories.Credential{
		ID:              uuid.New(),
		ClientID:        webauthnUser.ID, // Use webauthnUser.ID to match FinishRegistration pattern
		CredentialID:    credential.ID,
		PublicKey:       credential.PublicKey,
		AttestationType: attestationType,
		SignCount:       int64(credential.Authenticator.SignCount),
		BackupEligible:  credential.Flags.BackupEligible,
		BackupState:     credential.Flags.BackupState,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	// Save credential using repository with detailed logging
	clientRepo := repositories.NewClientRepository(db)
	log.Printf("[%s] ConfirmBiometricSetup: About to save credential with ClientID=%s", reqID, webauthnUser.ID.String())
	if err := clientRepo.SaveCredential(&cred); err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: failed to save credential: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist biometric credential"})
		return
	}
	log.Printf("[%s] ConfirmBiometricSetup: Credential saved successfully with ID=%s", reqID, cred.ID.String())

	// Enable MFA method
	mfaRepo := repositories.NewMFARepository(db)
	if err := mfaRepo.EnableMethod(webauthnUser.ID.String(), "webauthn", map[string]interface{}{
		"credential_id":    fmt.Sprintf("%x", credential.ID),
		"attestation_type": credential.AttestationType,
	}, webauthnUser.ID); err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: failed to enable MFA method: %v", reqID, err)
	}

	// Update user MFA settings
	methods := pq.StringArray{"webauthn"}

	updateSQL := `
		UPDATE users 
		SET mfa_enabled = true, 
		    mfa_verified = true, 
		    mfa_default_method = CASE WHEN mfa_default_method IS NULL OR mfa_default_method = '' THEN 'webauthn' ELSE mfa_default_method END,
		    mfa_method = $1,
		    updated_at = $2
		WHERE id = $3`

	if err := db.Exec(updateSQL, methods, now, webauthnUser.ID).Error; err != nil {
		log.Printf("[%s] ConfirmBiometricSetup: failed to update user MFA flags: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update user MFA settings"})
		return
	}

	// Cleanup session
	h.SessionManager.Delete(challengeKey)

	// Audit log for successful biometric setup
	middleware.AuditAuthentication(c, webauthnUser.ID.String(), "webauthn", "biometric_setup", true, map[string]interface{}{
		"credential_id": fmt.Sprintf("%x", credential.ID),
		"tenant_id":     req.TenantID,
	})

	log.Printf("[%s] ConfirmBiometricSetup: COMPLETED SUCCESSFULLY", reqID)
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"credential_id": fmt.Sprintf("%x", credential.ID),
	})
}

//
// --- Verify Biometric Finish ---
//

func (h *WebAuthnHandler) VerifyBiometricFinish(c *gin.Context) {
	var req struct {
		Email      string          `json:"email"`
		TenantID   string          `json:"tenant_id"`
		Credential json.RawMessage `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userRecord.ToShared()
	webUser := &WebAuthnUser{User: &user}

	challengeKey := buildChallengeKey("biometric_login", req.Email, req.TenantID)
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "no biometric session"})
		return
	}

	// Type assert to webauthn.SessionData
	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("session data is not *webauthn.SessionData: %T", sessionData)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid session data"})
		return
	}

	c.Request.Body = io.NopCloser(bytes.NewBuffer(req.Credential))
	c.Request.ContentLength = int64(len(req.Credential))

	// This should be FinishLogin for authentication, not FinishRegistration!
	credential, err := h.WebAuthn.FinishLogin(webUser, *sessionDataTyped, c.Request)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "biometric verification failed"})
		return
	}

	// Update credential sign count after successful authentication
	clientRepo := repositories.NewClientRepository(db)
	if err := clientRepo.UpdateCredentialSignCount(credential.ID, credential.Authenticator.SignCount); err != nil {
		log.Printf("VerifyBiometricFinish: failed to update sign count: %v", err)
	}

	h.SessionManager.Delete(challengeKey)

	c.JSON(http.StatusOK, gin.H{"success": true})
}

//
// --- OTP Setup ---
//

func (h *WebAuthnHandler) BeginOTPSetup(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userRecord.ToShared()

	// Generate OTP secret (TOTP)
	secret := uuid.New().String() // replace with proper TOTP secret generator
	now := time.Now()

	mfa := sharedmodels.MFAMethod{
		ID:         uuid.New(),
		ClientID:   user.ClientID,
		UserID:     &user.ID,
		MethodType: "otp",
		MethodData: datatypes.JSON(json.RawMessage(fmt.Sprintf(`{"secret":"%s"}`, secret))),
		Verified:   false,
		EnrolledAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(&mfa).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save otp secret"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"secret": secret})
}

func (h *WebAuthnHandler) VerifyOTP(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var mfa sharedmodels.MFAMethod
	if err := db.Where("method_type = ? AND enabled = true", "otp").First(&mfa).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "otp method not found"})
		return
	}

	// TODO: verify TOTP code properly
	if req.Code == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid otp"})
		return
	}

	if err := db.Model(&mfa).Update("verified", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update otp"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

//
// --- SMS Setup ---
//

func (h *WebAuthnHandler) BeginSMSSetup(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userRecord.ToShared()

	now := time.Now()
	mfa := sharedmodels.MFAMethod{
		ID:         uuid.New(),
		ClientID:   user.ClientID,
		UserID:     &user.ID,
		MethodType: "sms",
		MethodData: datatypes.JSON(json.RawMessage(fmt.Sprintf(`{"phone":"%s"}`, req.Phone))),
		Verified:   false,
		EnrolledAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(&mfa).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save sms"})
		return
	}

	// TODO: send SMS code
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "sms code sent"})
}

func (h *WebAuthnHandler) VerifySMS(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var mfa sharedmodels.MFAMethod
	if err := db.Where("method_type = ? AND enabled = true", "sms").First(&mfa).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "sms method not found"})
		return
	}

	// TODO: verify SMS code properly
	if req.Code == "" {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid sms code"})
		return
	}

	if err := db.Model(&mfa).Update("verified", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update sms"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

//
// --- Backup Codes ---
//

func (h *WebAuthnHandler) GenerateBackupCodes(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "user not found"})
		return
	}
	user := userRecord.ToShared()

	// generate backup codes
	codes := []string{
		uuid.New().String()[:8],
		uuid.New().String()[:8],
		uuid.New().String()[:8],
		uuid.New().String()[:8],
		uuid.New().String()[:8],
	}
	codesJSON, _ := json.Marshal(codes)
	now := time.Now()

	mfa := sharedmodels.MFAMethod{
		ID:         uuid.New(),
		ClientID:   user.ClientID,
		UserID:     &user.ID,
		MethodType: "backup",
		MethodData: datatypes.JSON(codesJSON),
		Enabled:    true,
		Verified:   true,
		EnrolledAt: now,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := db.Create(&mfa).Error; err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save backup codes"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"backup_codes": codes})
}

// BeginWebAuthnRegistration is a wrapper for backward compatibility with routes
func (h *WebAuthnHandler) BeginWebAuthnRegistration(c *gin.Context) {
	h.BeginRegistration(c)
}

// BeginBiometricVerify wraps BeginAuthentication for biometric flows
func (h *WebAuthnHandler) BeginBiometricVerify(c *gin.Context) {
	h.BeginAuthentication(c)
}

// FinishBiometricVerify wraps FinishAuthentication for biometric flows
// FinishBiometricLoginVerify verifies a biometric (WebAuthn) login attempt

func (h *WebAuthnHandler) FinishBiometricLoginVerify(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] FinishBiometricLoginVerify: START", reqID)

	// Parse request
	var req struct {
		Email      string                               `json:"email"`
		TenantID   string                               `json:"tenant_id"`
		Credential protocol.CredentialAssertionResponse `json:"credential"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] FinishBiometricLoginVerify: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	// ✅ Resolve DB dynamically (tenant vs global)
	db := h.resolveDB(req.TenantID)

	// Load session
	challengeKey := buildChallengeKey("biometric_login", req.Email, req.TenantID)
	sessionData, found := h.SessionManager.Get(challengeKey)
	if !found {
		log.Printf("[%s] FinishBiometricLoginVerify: no session found for key=%s", reqID, challengeKey)
		c.JSON(http.StatusBadRequest, gin.H{"error": "no login session"})
		return
	}

	// Type assert to webauthn.SessionData
	sessionDataTyped, ok := sessionData.(*webauthn.SessionData)
	if !ok {
		log.Printf("[%s] FinishBiometricLoginVerify: session data is not *webauthn.SessionData: %T", reqID, sessionData)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session data"})
		return
	}

	// Lookup user in DB
	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		log.Printf("[%s] FinishBiometricLoginVerify: user not found email=%s tenant=%s err=%v", reqID, req.Email, req.TenantID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	user := userRecord.ToShared()

	// Wrap user as WebAuthnUser
	webauthnUser := &WebAuthnUser{User: &user}

	// Perform WebAuthn verification
	credential, err := h.WebAuthn.FinishLogin(webauthnUser, *sessionDataTyped, c.Request)
	if err != nil {
		log.Printf("[%s] FinishBiometricLoginVerify: login verification failed: %v", reqID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "biometric login failed"})
		return
	}

	// ✅ Update credential usage
	clientRepo := repositories.NewClientRepository(db)
	if err := clientRepo.UpdateCredentialSignCount(credential.ID, credential.Authenticator.SignCount); err != nil {
		log.Printf("[%s] FinishBiometricLoginVerify: failed updating credential sign_count: %v", reqID, err)
	}
	if err := clientRepo.UpdateCredentialFlags(credential.ID, credential.Flags.BackupEligible, credential.Flags.BackupState); err != nil {
		log.Printf("[%s] FinishBiometricLoginVerify: failed updating credential flags: %v", reqID, err)
	}

	// Cleanup session
	h.SessionManager.Delete(challengeKey)

	log.Printf("[%s] FinishBiometricLoginVerify: COMPLETED SUCCESSFULLY", reqID)
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Biometric login verified",
		"credential_id": fmt.Sprintf("%x", credential.ID),
	})
}

// BeginBiometricLoginSetup wraps BeginRegistration for biometric login setup
func (h *WebAuthnHandler) BeginBiometricLoginSetup(c *gin.Context) {
	h.BeginRegistration(c)
}

// ConfirmBiometricLoginSetup wraps FinishRegistration for biometric login setup
// ConfirmBiometricLoginSetup confirms that biometric login registration is completed
func (h *WebAuthnHandler) ConfirmBiometricLoginSetup(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] ConfirmBiometricLoginSetup: redirecting to FinishRegistration", reqID)

	// Instead of duplicating FinishRegistration logic, delegate
	h.FinishRegistration(c)
}

// BeginBiometricLoginVerify wraps BeginAuthentication for biometric login
// BeginBiometricLoginVerify starts biometric login verification (assertion)
func (h *WebAuthnHandler) BeginBiometricLoginVerify(c *gin.Context) {
	reqID := uuid.New().String()
	log.Printf("[%s] BeginBiometricLoginVerify: START", reqID)

	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("[%s] BeginBiometricLoginVerify: invalid request: %v", reqID, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	db := h.resolveDB(req.TenantID)

	var userRecord appmodels.UserWithJSONMFAMethods
	if err := db.Scopes(util.WithUsersMFAMethodArray).
		Where("email = ? AND tenant_id = ?", req.Email, req.TenantID).First(&userRecord).Error; err != nil {
		log.Printf("[%s] BeginBiometricLoginVerify: user not found email=%s tenant=%s err=%v", reqID, req.Email, req.TenantID, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
		return
	}
	user := userRecord.ToShared()

	webauthnUser := &WebAuthnUser{User: &user}
	options, sessionData, err := h.WebAuthn.BeginLogin(webauthnUser)
	if err != nil {
		log.Printf("[%s] BeginBiometricLoginVerify: begin login failed: %v", reqID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin login"})
		return
	}

	// Save session
	challengeKey := buildChallengeKey("biometric_login", req.Email, req.TenantID)
	if err := h.SessionManager.Save(challengeKey, sessionData); err != nil {
		log.Printf("[%s] BeginBiometricLoginVerify: Failed to save session - %v", reqID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save session"})
		return
	}

	log.Printf("[%s] BeginBiometricLoginVerify: session saved for key=%s", reqID, challengeKey)
	c.JSON(http.StatusOK, options)
}

func (h *WebAuthnHandler) VerifyBackupCode(c *gin.Context) {
	var req struct {
		Email    string `json:"email"`
		TenantID string `json:"tenant_id"`
		Code     string `json:"code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	// Resolve database (global vs tenant)
	db := h.resolveDB(req.TenantID)

	var mfa sharedmodels.MFAMethod
	if err := db.Where("method_type = ? AND enabled = true", "backup").First(&mfa).Error; err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "backup codes not found"})
		return
	}

	var codes []string
	if err := json.Unmarshal(mfa.MethodData, &codes); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to parse backup codes"})
		return
	}

	valid := false
	for _, code := range codes {
		if code == req.Code {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid backup code"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func extractChallengeFromBody(body []byte) (string, error) {
	var root map[string]interface{}
	if err := json.Unmarshal(body, &root); err != nil {
		return "", fmt.Errorf("bad json: %w", err)
	}
	credVal, ok := root["credential"]
	var cred map[string]interface{}
	if ok {
		cred, _ = credVal.(map[string]interface{})
	} else {
		cred = root // maybe the credential is the root
	}
	respVal, ok := cred["response"]
	if !ok {
		return "", errors.New("response missing")
	}
	resp, _ := respVal.(map[string]interface{})
	cdj, _ := resp["clientDataJSON"].(string)
	if cdj == "" {
		return "", errors.New("clientDataJSON missing")
	}
	return extractChallengeFromClientData(cdj)
}
