package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"github.com/authsec-ai/authsec/config"
	middleware "github.com/authsec-ai/authsec/middlewares"
	appmodels "github.com/authsec-ai/authsec/models"
	repositories "github.com/authsec-ai/authsec/repository"
	"github.com/authsec-ai/authsec/services"
	util "github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
)

func NewTOTPHandler() *TOTPHandler {
	return &TOTPHandler{
		Service: services.NewWebAuthnTOTPService(),
	}
}

// @Summary      Begin TOTP Setup
// @Description  Start TOTP authenticator app setup process
// @Tags         TOTP
// @Accept       json
// @Produce      json
// @Param        request body TOTPSetupRequest true "User information"
// @Success      200 {object} TOTPSetupResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/totp/beginSetup [post]
// BeginTOTPSetup generates TOTP secret and QR code
func (h *TOTPHandler) BeginTOTPSetup(c *gin.Context) {
	var req TOTPSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Starting TOTP setup for email: %s, tenant: %s, client: %s", req.Email, req.TenantID, req.ClientID)

	// Get client from database
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Ensure client has a client_id
	if client.ClientID == uuid.Nil {
		client.ClientID = uuid.New()
		if err := tenantDB.Save(&client).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "could not update user ClientID"})
			return
		}
	}

	// Generate TOTP secret
	issuer := os.Getenv("WEBAUTHN_RP_NAME")
	if issuer == "" {
		issuer = "AuthSec"
	}

	key, err := h.Service.GenerateSecret(req.Email, issuer)
	if err != nil {
		log.Printf("Failed to generate TOTP secret: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate secret"})
		return
	}

	// Generate QR code
	qrCode, err := h.Service.GenerateQRCode(key, 256)
	if err != nil {
		log.Printf("Failed to generate QR code: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate QR code"})
		return
	}

	log.Printf("TOTP setup initiated for: %s", req.Email)

	// Return setup data (secret will be confirmed in next step)
	response := TOTPSetupResponse{
		Secret:      key.Secret(),
		QRCode:      base64.StdEncoding.EncodeToString(qrCode),
		ManualEntry: key.Secret(),
		Issuer:      issuer,
		Account:     req.Email,
		OTPAuthURL:  key.String(),
	}
	c.JSON(http.StatusOK, response)
}

func (h *TOTPHandler) BeginSetup(c *gin.Context) {
	var req TOTPSetupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Starting TOTP setup for email: %s, tenant: %s, client: %s", req.Email, req.TenantID, req.ClientID)

	// Get client from database
	tenantDB, client, err := fetchClientForLoginMFA(req.Email, req.TenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Ensure client has a client_id
	if client.ClientID == uuid.Nil {
		client.ClientID = uuid.New()
		if err := tenantDB.Save(&client).Error; err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "could not update user ClientID"})
			return
		}
	}

	// Generate TOTP secret
	issuer := os.Getenv("WEBAUTHN_RP_NAME")
	if issuer == "" {
		issuer = "AuthSec"
	}

	key, err := h.Service.GenerateSecret(req.Email, issuer)
	if err != nil {
		log.Printf("Failed to generate TOTP secret: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate secret"})
		return
	}

	// Generate QR code
	qrCode, err := h.Service.GenerateQRCode(key, 256)
	if err != nil {
		log.Printf("Failed to generate QR code: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate QR code"})
		return
	}

	log.Printf("TOTP setup initiated for: %s", req.Email)

	// Return setup data (secret will be confirmed in next step)
	response := TOTPSetupResponse{
		Secret:      key.Secret(),
		QRCode:      base64.StdEncoding.EncodeToString(qrCode),
		ManualEntry: key.Secret(),
		Issuer:      issuer,
		Account:     req.Email,
		OTPAuthURL:  key.String(),
	}
	c.JSON(http.StatusOK, response)
}

// @Summary      Confirm TOTP Setup
// @Description  Confirm TOTP setup with verification code
// @Tags         TOTP
// @Accept       json
// @Produce      json
// @Param        request body TOTPConfirmRequest true "TOTP confirmation data"
// @Success      200 {object} TOTPConfirmResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/totp/confirmSetup [post]
// ConfirmTOTPSetup validates TOTP code and enables TOTP
func (h *TOTPHandler) ConfirmTOTPSetup(c *gin.Context) {
	var req TOTPConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Confirming TOTP setup for email: %s", req.Email)

	// Validate the TOTP code
	if !h.Service.ValidateCodeWithWindow(req.Secret, req.Code, 1) {
		log.Printf("Invalid TOTP code provided for: %s", req.Email)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid TOTP code"})
		return
	}

	// Get client from database
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Generate backup codes
	backupCodes, err := h.Service.GenerateBackupCodes(10)
	if err != nil {
		log.Printf("Failed to generate backup codes: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate backup codes"})
		return
	}

	// Encrypt secret and backup codes
	encryptedSecret, err := util.EncryptString(req.Secret)
	if err != nil {
		log.Printf("Failed to encrypt TOTP secret: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save TOTP data"})
		return
	}

	// Encrypt backup codes
	encryptedCodes := make(pq.StringArray, len(backupCodes))
	for i, code := range backupCodes {
		encrypted, err := util.EncryptString(code)
		if err != nil {
			log.Printf("Failed to encrypt backup code: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save backup codes"})
			return
		}
		encryptedCodes[i] = encrypted
	}

	// Create TOTP method data
	totpData := map[string]interface{}{
		"secret_encrypted": encryptedSecret,
		"issuer":           os.Getenv("WEBAUTHN_RP_NAME"),
		"algorithm":        "SHA1",
		"digits":           6,
		"period":           30,
		"setup_completed":  time.Now().UTC(),
	}

	// Save to MFA methods table
	mfaRepo := repositories.NewMFARepository(tenantDB)
	err = mfaRepo.EnableMethodWithBackupCodes(client.ID.String(), "totp", totpData, encryptedCodes, client.ID)
	if err != nil {
		log.Printf("Failed to save TOTP method: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to enable TOTP"})
		return
	}

	// Update client MFA settings in clients table
	updates := map[string]interface{}{
		"mfa_enabled": true,
		"updated_at":  time.Now(),
	}

	// Set as default method if no other method is set
	if client.MFADefaultMethod == nil || *client.MFADefaultMethod == "" {
		updates["mfa_default_method"] = "totp"
	}

	// Add TOTP to MFA methods array if not present
	newMethods := client.MFAMethod
	if !contains(newMethods, "totp") {
		newMethods = append(newMethods, "totp")
		methodsJSON, _ := json.Marshal(newMethods)
		updates["mfa_method"] = datatypes.JSON(methodsJSON)
	}

	// Update client MFA settings using raw SQL
	now := time.Now()
	newMethods = client.MFAMethod
	if !contains(newMethods, "totp") {
		newMethods = append(newMethods, "totp")
	}

	updateSQL := `
		UPDATE users 
		SET mfa_enabled = true, 
		    mfa_default_method = CASE WHEN mfa_default_method IS NULL OR mfa_default_method = '' THEN 'totp' ELSE mfa_default_method END,
		    mfa_method = $1,
		    updated_at = $2
		WHERE id = $3`

	if err := tenantDB.Exec(updateSQL, pq.StringArray(newMethods), now, client.ID).Error; err != nil {
		log.Printf("Failed to update client MFA settings: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update MFA settings"})
		return
	}

	// Format backup codes for display
	displayCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		displayCodes[i] = h.Service.FormatBackupCode(code)
	}

	log.Printf("TOTP setup completed successfully for: %s", req.Email)

	// Audit log for successful TOTP setup
	middleware.AuditAuthentication(c, client.ID.String(), "totp", "setup", true, map[string]interface{}{
		"tenant_id": req.TenantID,
		"email":     req.Email,
	})

	response := TOTPConfirmResponse{
		Success:     true,
		Message:     "TOTP enabled successfully",
		BackupCodes: displayCodes,
	}

	c.JSON(http.StatusOK, response)
}

func (h *TOTPHandler) ConfirmSetup(c *gin.Context) {
	var req TOTPConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Confirming TOTP setup for email: %s", req.Email)

	// Validate the TOTP code
	if !h.Service.ValidateCodeWithWindow(req.Secret, req.Code, 1) {
		log.Printf("Invalid TOTP code provided for: %s", req.Email)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid TOTP code"})
		return
	}

	// Get client from database
	tenantDB, client, err := fetchClientForLoginMFA(req.Email, req.TenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Generate backup codes
	backupCodes, err := h.Service.GenerateBackupCodes(10)
	if err != nil {
		log.Printf("Failed to generate backup codes: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to generate backup codes"})
		return
	}

	// Encrypt secret and backup codes
	encryptedSecret, err := util.EncryptString(req.Secret)
	if err != nil {
		log.Printf("Failed to encrypt TOTP secret: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save TOTP data"})
		return
	}

	// Encrypt backup codes
	encryptedCodes := make(pq.StringArray, len(backupCodes))
	for i, code := range backupCodes {
		encrypted, err := util.EncryptString(code)
		if err != nil {
			log.Printf("Failed to encrypt backup code: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to save backup codes"})
			return
		}
		encryptedCodes[i] = encrypted
	}

	// Create TOTP method data
	totpData := map[string]interface{}{
		"secret_encrypted": encryptedSecret,
		"issuer":           os.Getenv("WEBAUTHN_RP_NAME"),
		"algorithm":        "SHA1",
		"digits":           6,
		"period":           30,
		"setup_completed":  time.Now().UTC(),
	}

	// Save to MFA methods table
	mfaRepo := repositories.NewMFARepository(tenantDB)
	err = mfaRepo.EnableMethodWithBackupCodes(client.ID.String(), "totp", totpData, encryptedCodes, client.ID)
	if err != nil {
		log.Printf("Failed to save TOTP method: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to enable TOTP"})
		return
	}

	// Update client MFA settings in clients table
	updates := map[string]interface{}{
		"mfa_enabled": true,
		"updated_at":  time.Now(),
	}

	// Set as default method if no other method is set
	if client.MFADefaultMethod == nil || *client.MFADefaultMethod == "" {
		updates["mfa_default_method"] = "totp"
	}

	// Add TOTP to MFA methods array if not present
	newMethods := client.MFAMethod
	if !contains(newMethods, "totp") {
		newMethods = append(newMethods, "totp")
		methodsJSON, _ := json.Marshal(newMethods)
		updates["mfa_method"] = datatypes.JSON(methodsJSON)
	}

	// Update client MFA settings using raw SQL
	now := time.Now()
	newMethods = client.MFAMethod
	if !contains(newMethods, "totp") {
		newMethods = append(newMethods, "totp")
	}

	updateSQL := `
		UPDATE users 
		SET mfa_enabled = true, 
		    mfa_default_method = CASE WHEN mfa_default_method IS NULL OR mfa_default_method = '' THEN 'totp' ELSE mfa_default_method END,
		    mfa_method = $1,
		    updated_at = $2
		WHERE id = $3`

	if err := tenantDB.Exec(updateSQL, pq.StringArray(newMethods), now, client.ID).Error; err != nil {
		log.Printf("Failed to update client MFA settings: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to update MFA settings"})
		return
	}

	// Format backup codes for display
	displayCodes := make([]string, len(backupCodes))
	for i, code := range backupCodes {
		displayCodes[i] = h.Service.FormatBackupCode(code)
	}

	log.Printf("TOTP setup completed successfully for: %s", req.Email)

	// Audit log for successful TOTP setup (login flow)
	middleware.AuditAuthentication(c, client.ID.String(), "totp", "setup", true, map[string]interface{}{
		"tenant_id": req.TenantID,
		"email":     req.Email,
	})

	response := TOTPConfirmResponse{
		Success:     true,
		Message:     "TOTP enabled successfully",
		BackupCodes: displayCodes,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Verify TOTP Code
// @Description  Verify TOTP code for authentication
// @Tags         TOTP
// @Accept       json
// @Produce      json
// @Param        request body TOTPVerifyRequest true "TOTP verification data"
// @Success      200 {object} AuthenticationResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/totp/verify [post]
func (h *TOTPHandler) VerifyTOTP(c *gin.Context) {
	var req TOTPVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Verifying TOTP code for email: %s", req.Email)

	// Get client from database
	tenantDB, client, err := fetchClientForMFA(req.Email, req.TenantID, req.ClientID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Check if backup code was provided
	if req.BackupCode != "" {
		// Verify backup code
		mfaRepo := repositories.NewMFARepository(tenantDB)
		method, err := mfaRepo.GetMethod(client.ID.String(), "totp")
		if err != nil {
			log.Printf("Failed to get TOTP method: %v", err)
			c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to verify backup code"})
			return
		}

		// Decrypt and check backup codes
		var encryptedCodes pq.StringArray
		if method.BackupCodes != nil {
			if err := json.Unmarshal([]byte(*method.BackupCodes), &encryptedCodes); err != nil {
				log.Printf("Failed to parse backup codes: %v", err)
				c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "invalid backup codes"})
				return
			}
		}

		for i, encryptedCode := range encryptedCodes {
			decryptedCode, err := util.DecryptString(encryptedCode)
			if err != nil {
				log.Printf("Failed to decrypt backup code: %v", err)
				continue
			}
			if decryptedCode == req.BackupCode {
				// Valid backup code - remove it from the list
				newCodes := make(pq.StringArray, 0, len(encryptedCodes)-1)
				newCodes = append(newCodes, encryptedCodes[:i]...)
				newCodes = append(newCodes, encryptedCodes[i+1:]...)

				// Update backup codes
				if err := mfaRepo.UpdateBackupCodes(client.ID.String(), "totp", newCodes); err != nil {
					log.Printf("Failed to update backup codes: %v", err)
				}

				// Update user's last login
				if err := tenantDB.Model(&client).Update("last_login", time.Now()).Error; err != nil {
					log.Printf("Failed to update last login: %v", err)
				}

				response := AuthenticationResponse{
					Success:  true,
					Message:  "Backup code accepted",
					Method:   "backup_code",
					TenantID: req.TenantID,
					Email:    client.Email,
				}
				c.JSON(http.StatusOK, response)
				return
			}
		}

		log.Printf("Invalid backup code provided for: %s", req.Email)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid backup code"})
		return
	}

	// Verify regular TOTP code
	mfaRepo := repositories.NewMFARepository(tenantDB)
	method, err := mfaRepo.GetMethod(client.ID.String(), "totp")
	if err != nil {
		log.Printf("Failed to get TOTP method: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to verify TOTP"})
		return
	}

	var totpData struct {
		SecretEncrypted string `json:"secret_encrypted"`
	}

	if err := json.Unmarshal(method.MethodData, &totpData); err != nil {
		log.Printf("Failed to parse TOTP method data: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "invalid TOTP configuration"})
		return
	}

	// Decrypt secret
	secret, err := util.DecryptString(totpData.SecretEncrypted)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to verify TOTP"})
		return
	}

	// Validate TOTP code
	if !h.Service.ValidateCode(secret, req.Code) {
		log.Printf("Invalid TOTP code provided for: %s", req.Email)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid TOTP code"})
		return
	}

	// Update user's last login
	if err := tenantDB.Model(&client).Update("last_login", time.Now()).Error; err != nil {
		log.Printf("Failed to update last login: %v", err)
	}

	// Update MFA method last used
	if err := tenantDB.Model(&sharedmodels.MFAMethod{}).
		Where("client_id = ? AND method_type = ?", client.ID, "totp").
		Update("last_used_at", time.Now()).Error; err != nil {
		log.Printf("Failed to update MFA method last used: %v", err)
	}

	log.Printf("TOTP verification successful for: %s", req.Email)

	// Audit log for successful TOTP verification
	middleware.AuditAuthentication(c, client.ID.String(), "totp", "verify", true, map[string]interface{}{
		"tenant_id": req.TenantID,
		"email":     req.Email,
	})

	response := AuthenticationResponse{
		Success:  true,
		Message:  "TOTP verification successful",
		Method:   "totp",
		TenantID: req.TenantID,
		Email:    client.Email,
	}

	c.JSON(http.StatusOK, response)
}

// @Summary      Verify TOTP for Login
// @Description  Verify TOTP code during login process
// @Tags         TOTP
// @Accept       json
// @Produce      json
// @Param        request body TOTPLoginVerifyRequest true "TOTP login verification data"
// @Success      200 {object} AuthenticationResponse
// @Failure      400 {object} ErrorResponse
// @Failure      404 {object} ErrorResponse
// @Failure      500 {object} ErrorResponse
// @Router       /webauthn/totp/verifyLogin [post]
func (h *TOTPHandler) VerifyLoginTOTP(c *gin.Context) {
	var req TOTPLoginVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	log.Printf("Verifying TOTP code for email: %s", req.Email)

	// Get client from database
	tenantDB, client, err := fetchClientForLoginMFA(req.Email, req.TenantID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "client not found"})
		return
	}

	// Get TOTP method from MFA methods table
	mfaRepo := repositories.NewMFARepository(tenantDB)
	method, err := mfaRepo.GetMethod(client.ID.String(), "totp")
	if err != nil || !method.Enabled {
		// Check if user has other MFA methods available
		availableMethods, _ := mfaRepo.GetUserMethods(client.ID.String())
		if len(availableMethods) > 0 {
			methodType := availableMethods[0].MethodType
			log.Printf("TOTP not enabled for client: %s, but %s is available", req.Email, methodType)
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error: fmt.Sprintf("TOTP not enabled for this account. Use %s method instead.", methodType),
			})
			return
		}
		log.Printf("TOTP not enabled for client: %s", req.Email)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "TOTP not enabled for this account"})
		return
	}

	// First, check if it's a backup code
	if len(req.Code) == 8 || len(req.Code) == 9 { // 8 chars or with dash
		if err := h.verifyBackupCode(mfaRepo, client.ID.String(), req.Code, method.BackupCodes); err == nil {
			log.Printf("Backup code verified for: %s", req.Email)

			// Update MFA verified status for backup code
			h.updateMFAVerifiedStatus(tenantDB, req.TenantID)

			response := AuthenticationResponse{
				Success:  true,
				Message:  "Backup code accepted",
				Method:   "backup_code",
				TenantID: req.TenantID,
				Email:    client.Email,
			}
			c.JSON(http.StatusOK, response)
			return
		}
	}

	// Verify regular TOTP code
	var totpData struct {
		SecretEncrypted string `json:"secret_encrypted"`
	}

	if err := json.Unmarshal(method.MethodData, &totpData); err != nil {
		log.Printf("Failed to parse TOTP method data: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "invalid TOTP configuration"})
		return
	}

	// Decrypt secret
	secret, err := util.DecryptString(totpData.SecretEncrypted)
	if err != nil {
		log.Printf("Failed to decrypt TOTP secret: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "failed to verify TOTP"})
		return
	}

	// Validate TOTP code
	if h.Service.ValidateCodeWithWindow(secret, req.Code, 1) {
		// Update last used timestamp
		mfaRepo.UpdateLastUsed(client.ID.String(), "totp")

		log.Printf("TOTP code verified for: %s", req.Email)

		// Update MFA verified status
		h.updateMFAVerifiedStatus(tenantDB, client.ID.String())

		// Audit log for successful TOTP login verification
		middleware.AuditAuthentication(c, client.ID.String(), "totp", "login_verify", true, map[string]interface{}{
			"tenant_id": req.TenantID,
			"email":     req.Email,
		})

		response := AuthenticationResponse{
			Success:  true,
			Message:  "TOTP code valid",
			Method:   "totp",
			TenantID: req.TenantID,
			Email:    client.Email,
		}
		c.JSON(http.StatusOK, response)
		return
	}

	// Audit log for failed TOTP login verification
	middleware.AuditAuthentication(c, client.ID.String(), "totp", "login_verify", false, map[string]interface{}{
		"tenant_id": req.TenantID,
		"email":     req.Email,
		"reason":    "invalid_code",
	})

	log.Printf("Invalid TOTP code for: %s", req.Email)
	c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid TOTP code"})
}

// Helper function to safely update MFA verified status
func (h *TOTPHandler) updateMFAVerifiedStatus(tenantDB *gorm.DB, ID string) {
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	if err := tenantDB.Scopes(util.WithUsersMFAMethodArray).Where("id = ?", ID).First(&userWithJSONMFA).Error; err != nil {
		log.Printf("Failed to find client for MFA verification update: %v", err)
		return // Don't fail the authentication for this
	}

	// Safe way to update MFAVerified
	updates := map[string]interface{}{
		"mfa_verified": true,
		"last_login":   time.Now(),
	}

	if err := tenantDB.Model(&userWithJSONMFA).Updates(updates).Error; err != nil {
		log.Printf("Failed to update client mfa_verified status: %v", err)
		// Don't fail the authentication for this, just log the error
	} else {
		log.Printf("Successfully updated MFA verified status for client: %s", ID)
	}
}

// Helper function to verify backup codes
func (h *TOTPHandler) verifyBackupCode(mfaRepo *repositories.MFARepository, clientID, code string, encryptedCodes *string) error {
	// Convert backup codes from JSON string to pq.StringArray
	var backupCodes pq.StringArray
	if encryptedCodes != nil {
		if err := json.Unmarshal([]byte(*encryptedCodes), &backupCodes); err != nil {
			return fmt.Errorf("failed to unmarshal backup codes: %w", err)
		}
	}

	// Decrypt and check backup codes
	for i, encryptedCode := range backupCodes {
		decryptedCode, err := util.DecryptString(encryptedCode)
		if err != nil {
			continue
		}

		// Compare codes (case insensitive, ignore dashes)
		normalizedInput := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
		normalizedStored := strings.ToUpper(strings.ReplaceAll(decryptedCode, "-", ""))

		if normalizedInput == normalizedStored {
			// Remove used backup code
			newCodes := make(pq.StringArray, 0, len(backupCodes)-1)
			for j, c := range backupCodes {
				if j != i {
					newCodes = append(newCodes, c)
				}
			}

			// Update backup codes in database
			return mfaRepo.UpdateBackupCodes(clientID, "totp", newCodes)
		}
	}

	return fmt.Errorf("backup code not found")
}

// Helper function to fetch client (shared across TOTP handlers)
func fetchClientForMFA(email, tenantID, clientID string) (*gorm.DB, *sharedmodels.User, error) {
	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		return nil, nil, err
	}

	// First check if user exists in global DB (for admin users)
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	globalErr := globalDB.Scopes(util.WithUsersMFAMethodArray).Where("email = ? AND tenant_id = ?", email, tenantID).First(&userWithJSONMFA).Error
	user := userWithJSONMFA.ToShared()

	var tenantDB *gorm.DB
	if globalErr == nil && user.ID != uuid.Nil {
		// Found admin user in global DB - no need for tenant DB
		tenantDB = nil
		log.Printf("fetchClientForMFA: Found admin user in global DB: %s", user.Email)
		return globalDB, &user, nil
	} else {
		// Not an admin user - proceed with tenant DB lookup
		tenantDBName, err := config.GetTenantDBName(globalDB, tenantID)
		if err != nil {
			return nil, nil, err
		}

		tenantDB, err = config.ConnectTenantDB(tenantDBName)
		if err != nil {
			return nil, nil, err
		}

		clientRepo := repositories.NewClientRepository(tenantDB)
		// Use the new TOTP-specific function
		client, err := clientRepo.GetClientForTOTP(email, tenantID, clientID)
		if err != nil {
			return nil, nil, err
		}

		return tenantDB, client, nil
	}
}

func fetchClientForLoginMFA(email, tenantID string) (*gorm.DB, *sharedmodels.User, error) {
	globalDB, err := config.ConnectGlobalDB()
	if err != nil {
		return nil, nil, err
	}

	// First check if user exists in global DB (for admin users)
	var userWithJSONMFA appmodels.UserWithJSONMFAMethods
	globalErr := globalDB.Scopes(util.WithUsersMFAMethodArray).Where("email = ? AND tenant_id = ?", email, tenantID).First(&userWithJSONMFA).Error
	user := userWithJSONMFA.ToShared()

	if globalErr == nil && user.ID != uuid.Nil {
		// Found admin user in global DB - no need for tenant DB
		log.Printf("fetchClientForLoginMFA: Found admin user in global DB: %s", user.Email)
		return globalDB, &user, nil
	} else {
		// Not an admin user - proceed with tenant DB lookup
		tenantDBName, err := config.GetTenantDBName(globalDB, tenantID)
		if err != nil {
			return nil, nil, err
		}

		tenantDB, err := config.ConnectTenantDB(tenantDBName)
		if err != nil {
			return nil, nil, err
		}

		clientRepo := repositories.NewClientRepository(tenantDB)
		// Use the new TOTP-specific login function
		client, err := clientRepo.GetClientForTOTPLogin(email, tenantID)
		if err != nil {
			return nil, nil, err
		}

		return tenantDB, client, nil
	}
}
