package enduser

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type EndUserAuthController struct {
	tenantRepo        *database.TenantRepository
	otpRepo           *database.OTPRepository
	pendingRepo       *database.PendingRegistrationRepository
	tenantDBService   *database.TenantDBService
	antiReplayService *services.AntiReplayService
}

// NewEndUserAuthController creates a new end-user auth controller
func NewEndUserAuthController() (*EndUserAuthController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	redisClient := config.GetRedisClient()
	if redisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	// Get config for database parameters
	cfg := config.GetConfig()

	// Create tenant database service
	tenantDBService, err := database.NewTenantDBService(db, cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBPort)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenant DB service: %w", err)
	}

	return &EndUserAuthController{
		tenantRepo:        database.NewTenantRepository(db),
		otpRepo:           database.NewOTPRepository(db),
		pendingRepo:       database.NewPendingRegistrationRepository(db),
		tenantDBService:   tenantDBService,
		antiReplayService: services.NewAntiReplayService(redisClient),
	}, nil
}

// tenantMapping maps client ID to tenant ID
func (euac *EndUserAuthController) tenantMapping(clientID uuid.UUID) (uuid.UUID, error) {
	// Query tenant_mappings table in global database
	db := config.GetDatabase()
	if db == nil {
		return uuid.UUID{}, fmt.Errorf("database not initialized")
	}

	var tenantID uuid.UUID
	query := `SELECT tenant_id FROM tenant_mappings WHERE client_id = $1`
	err := db.QueryRow(query, clientID).Scan(&tenantID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.UUID{}, fmt.Errorf("client not found")
		}
		return uuid.UUID{}, fmt.Errorf("failed to lookup tenant mapping: %w", err)
	}

	return tenantID, nil
}

// InitiateRegistration handles end-user registration initiation
// @Summary Initiate end-user registration
// @Description Starts the registration process for end-users by sending an OTP to the provided email
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.CustomLoginRegister true "Registration data including email, password, name, and client ID"
// @Success 200 {object} models.InitiateRegistrationResponse "OTP sent successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 409 {object} map[string]string "Conflict - user already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/initiate-registration [post]
func (euac *EndUserAuthController) InitiateRegistration(c *gin.Context) {
	var input models.CustomLoginRegister
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euac.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	var existingUser models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, clientUUID, []string{"custom", "ad_sync", "entra_id"}).First(&existingUser).Error; err == nil {
		if (existingUser.Provider == "ad_sync" || existingUser.Provider == "entra_id") && existingUser.PasswordHash == "" {
			// Allow registration to proceed for synced users without passwords
		} else {
			c.JSON(http.StatusOK, gin.H{"response": "true", "message": "User already exists"})
			return
		}
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	var client models.Client
	if err := tenantDB.Where("client_id = ? AND tenant_id = ?", clientUUID, tenantUUID).First(&client).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to find client: %v", err)})
		return
	}

	newUser := models.ExtendedUser{
		User: sharedmodels.User{
			ID:           uuid.New(),
			ClientID:     clientUUID,
			TenantID:     tenantUUID,
			ProjectID:    client.ProjectID,
			Name:         input.Email, // Use email as name
			Email:        input.Email,
			PasswordHash: hashedPassword,
			TenantDomain: config.AppConfig.TenantDomainSuffix, // Use configured domain suffix (authsec.dev)
			Provider:     "custom",
			ProviderID:   input.Email,
			Active:       true,
		},
	}

	if err := tenantDB.Create(&newUser).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Audit log: End user registration completed
	middlewares.Audit(c, "enduser", newUser.ID.String(), "register", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":     newUser.Email,
			"client_id": clientUUID.String(),
			"tenant_id": tenantUUID.String(),
			"provider":  newUser.Provider,
		},
	})

	c.JSON(http.StatusCreated, gin.H{"message": "Registration completed successfully"})
}

// VerifyOTPAndCompleteRegistration handles OTP verification and registration completion
// @Summary Verify OTP and complete registration
// @Description Verifies the OTP sent during registration and creates the user account
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.VerifyOTPInput true "OTP verification data"
// @Success 200 {object} models.RegisterResponse "Registration completed successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid OTP"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/verify-otp [post]
func (euac *EndUserAuthController) VerifyOTPAndCompleteRegistration(c *gin.Context) {
	// For end-user registration, we create users directly without OTP verification
	// This method can be used for password reset OTP verification instead
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Use InitiateRegistration for direct registration"})
}

// EmailCheckInput represents the email validation request
type EmailCheckInput struct {
	Email    string    `json:"email" binding:"required,email"`
	ClientID uuid.UUID `json:"client_id"`
}

// EndUserLoginPrecheck validates email and returns tenant context
// @Summary End-user login precheck
// @Description Validates end-user email exists and returns tenant information before login
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body EmailCheckInput true "Email to check"
// @Success 200 {object} map[string]interface{} "Email validation result with tenant info"
// @Failure 400 {object} map[string]string "Bad request - invalid email"
// @Failure 404 {object} map[string]string "User not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/login/precheck [post]
func (euac *EndUserAuthController) EndUserLoginPrecheck(c *gin.Context) {
	var input EmailCheckInput

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid email format"})
		return
	}

	input.Email = strings.ToLower(input.Email)

	// Get tenant ID from client ID
	if input.ClientID == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id is required"})
		return
	}

	tenantID, err := euac.tenantMapping(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client_id"})
		return
	}

	// Get database connection
	db := config.GetDatabase()

	// Check if user exists in tenant database
	var user struct {
		ID           uuid.UUID
		Email        string
		MFAEnabled   bool
		MFAMethod    string
		Active       bool
		TenantDomain string
	}

	query := `SELECT id, email, mfa_enabled, COALESCE(mfa_method, '') as mfa_method, 
	                 active, tenant_domain 
	          FROM users WHERE LOWER(email) = LOWER($1) LIMIT 1`

	err = db.QueryRow(query, input.Email).Scan(&user.ID, &user.Email, &user.MFAEnabled,
		&user.MFAMethod, &user.Active, &user.TenantDomain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		log.Printf("Database error checking user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":         user.Email,
		"exists":        true,
		"mfa_enabled":   user.MFAEnabled,
		"mfa_method":    user.MFAMethod,
		"tenant_domain": user.TenantDomain,
		"tenant_id":     tenantID.String(),
	})
}

// Login handles end-user login
// @Summary End-user login
// @Description Authenticates end-users and returns JWT tokens for tenant-specific operations
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.CustomLoginInput true "Login credentials with client ID"
// @Success 200 {object} models.CustomLoginStatus "Successful login with token"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid credentials"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/login [post]
func (euac *EndUserAuthController) Login(c *gin.Context) {
	var input models.CustomLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	// ========== ANTI-REPLAY ATTACK VALIDATION ==========
	// Check if request contains anti-replay protection fields
	// Note: CustomLoginInput from sharedmodels may not have these fields yet
	// This code demonstrates how to check if they exist in the JSON
	var rawInput map[string]interface{}
	if err := c.ShouldBindJSON(&rawInput); err == nil {
		nonce, hasNonce := rawInput["nonce"].(string)
		timestamp, hasTimestamp := rawInput["timestamp"].(float64)

		if hasNonce && hasTimestamp {
			log.Printf("[EndUserLogin][Anti-Replay] Validating request for email: %s, nonce: %s, timestamp: %.0f", input.Email, nonce, timestamp)

			secureReq := &models.SecureLoginRequest{
				Email:     input.Email,
				Password:  input.Password,
				Nonce:     nonce,
				Timestamp: int64(timestamp),
			}

			if challenge, hasChallenge := rawInput["challenge"].(string); hasChallenge {
				secureReq.Challenge = challenge
			}
			if signature, hasSignature := rawInput["signature"].(string); hasSignature {
				secureReq.Signature = signature
			}

			if err := euac.antiReplayService.ValidateLoginRequest(secureReq); err != nil {
				log.Printf("[EndUserLogin][Anti-Replay] REPLAY ATTACK DETECTED for email %s: %v", input.Email, err)
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Request validation failed",
					"hint":  "Possible replay attack detected. Please generate a new request with fresh nonce and timestamp.",
				})
				return
			}
			log.Printf("[EndUserLogin][Anti-Replay] Validation SUCCESS for email: %s", input.Email)
		} else {
			log.Printf("[EndUserLogin][Anti-Replay] WARNING: Login request without anti-replay protection for email: %s", input.Email)
		}
	}
	// ========== END ANTI-REPLAY VALIDATION ==========

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euac.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	var user models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider IN (?)", input.Email, clientUUID, []string{"custom", "ad_sync", "entra_id"}).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check account lockout status (using raw SQL since fields not in sharedmodels.User)
	var failedAttempts int
	var accountLockedAt *time.Time
	var passwordResetRequired bool
	if err := tenantDB.Raw(`
		SELECT failed_login_attempts, account_locked_at, password_reset_required 
		FROM users WHERE id = $1
	`, user.ID).Row().Scan(&failedAttempts, &accountLockedAt, &passwordResetRequired); err != nil {
		// Fields might not exist in old schemas, continue normally
		failedAttempts = 0
		accountLockedAt = nil
		passwordResetRequired = false
	}

	// Check if account is locked (lockout period: 30 minutes)
	if accountLockedAt != nil {
		lockoutDuration := 30 * time.Minute
		if time.Since(*accountLockedAt) < lockoutDuration {
			remainingTime := lockoutDuration - time.Since(*accountLockedAt)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Account is temporarily locked due to multiple failed login attempts",
				"message": fmt.Sprintf("Please try again in %d minutes or reset your password", int(remainingTime.Minutes())+1),
			})
			return
		}
		// Lockout period expired, reset fields
		tenantDB.Exec(`
			UPDATE users 
			SET failed_login_attempts = 0, account_locked_at = NULL 
			WHERE id = $1
		`, user.ID)
		failedAttempts = 0
		accountLockedAt = nil
	}

	// Check if password reset is required
	if passwordResetRequired {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Password reset required",
			"message": "Your account requires a password reset. Please use the forgot password flow.",
		})
		return
	}

	if !user.CheckPassword(input.Password) {
		// Increment failed login attempts
		failedAttempts++

		// Lock account after 3 failed attempts and require password reset
		if failedAttempts >= 3 {
			tenantDB.Exec(`
				UPDATE users 
				SET failed_login_attempts = $1, account_locked_at = $2, password_reset_required = true 
				WHERE id = $3
			`, failedAttempts, time.Now(), user.ID)

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Account locked due to multiple failed login attempts",
				"message": "Your account has been locked for security. Please reset your password to unlock.",
			})
			return
		}

		tenantDB.Exec(`
			UPDATE users 
			SET failed_login_attempts = $1 
			WHERE id = $2
		`, failedAttempts, user.ID)

		remainingAttempts := 3 - failedAttempts
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid credentials",
			"message": fmt.Sprintf("Invalid password. %d attempt(s) remaining before account lockout.", remainingAttempts),
		})
		return
	}

	// Successful login - reset failed attempts
	if failedAttempts > 0 {
		tenantDB.Exec(`
			UPDATE users 
			SET failed_login_attempts = 0, account_locked_at = NULL 
			WHERE id = $1
		`, user.ID)
	}

	// Generate JWT token
	token, err := euac.generateJWTToken(tenantUUID.String(), clientUUID.String(), input.Email, user.TenantDomain, &user.ID, tenantDB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Check if MFA is enabled
	var isFirstLogin bool
	if user.Provider == "ad_sync" || user.Provider == "entra_id" {
		isFirstLogin = true
	} else {
		isFirstLogin = user.LastLogin == nil
	}

	response := models.LoginResponse{
		TenantID:    user.TenantID.String(),
		Email:       user.Email,
		FirstLogin:  isFirstLogin,
		OTPRequired: false,
		Token:       token,
	}

	// Audit log: End user login successful
	middlewares.Audit(c, "enduser", user.ID.String(), "login", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":       user.Email,
			"client_id":   clientUUID.String(),
			"tenant_id":   tenantUUID.String(),
			"first_login": isFirstLogin,
			"provider":    user.Provider,
		},
	})

	c.JSON(http.StatusOK, response)
}

// SAMLLogin handles SAML-based login without password validation
// @Summary SAML login
// @Description Authenticates end-users via SAML provider without password requirement. The user's provider in the database must end with '-saml' (e.g., 'saml-okta', 'saml-azure')
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.SAMLLoginInput true "SAML login data (email and client_id only)"
// @Success 200 {object} models.LoginResponse "Login successful"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - SAML user not found or provider does not end with -saml"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/user/saml/login [post]
func (euac *EndUserAuthController) SAMLLogin(c *gin.Context) {
	var input models.SAMLLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euac.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to connect to tenant database: %v", err)})
		return
	}

	// Find user by email and client_id, and verify the provider starts with saml-
	var user models.User
	if err := tenantDB.Where("email = ? AND client_id = ? AND provider LIKE 'saml-%'", input.Email, clientUUID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "SAML user not found or provider does not start with saml-"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to authenticate user"})
		return
	}

	// No password validation for SAML users

	// Generate JWT token

	// Check if this is the first login
	isFirstLogin := user.LastLogin == nil

	response := models.LoginResponse{
		TenantID:    user.TenantID.String(),
		Email:       user.Email,
		FirstLogin:  isFirstLogin,
		OTPRequired: false,
	}

	// Audit log: SAML login successful
	middlewares.Audit(c, "enduser", user.ID.String(), "saml_login", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":       user.Email,
			"client_id":   clientUUID.String(),
			"tenant_id":   tenantUUID.String(),
			"first_login": isFirstLogin,
			"provider":    user.Provider,
		},
	})

	c.JSON(http.StatusOK, response)
}

// WebAuthnCallback handles WebAuthn authentication callback
// @Summary WebAuthn authentication callback
// @Description Processes WebAuthn authentication responses for passwordless login
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.WebAuthnCallbackInput true "WebAuthn callback data"
// @Success 200 {object} models.CustomLoginStatus "Successful WebAuthn authentication"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - authentication failed"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/webauthn-callback [post]
func (euac *EndUserAuthController) WebAuthnCallback(c *gin.Context) {
	var input models.WebAuthnCallbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	if input.MFAVerified == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "MFA verification status is required"})
		return
	}
	if !*input.MFAVerified {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "MFA verification failed"})
		return
	}

	tenant, err := euac.tenantRepo.GetTenantByTenantID(input.TenantID.String())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	tenantIDStr := tenant.TenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	var user models.User
	if err := tenantDB.Where("LOWER(email) = LOWER(?)", input.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	isFirstLogin := user.LastLogin == nil
	clientIDStr := ""
	if user.ClientID != uuid.Nil {
		clientIDStr = user.ClientID.String()
	}

	token, err := euac.generateJWTToken(tenantIDStr, clientIDStr, user.Email, user.TenantDomain, &user.ID, tenantDB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	now := time.Now()
	if err := tenantDB.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login", now).Error; err != nil {
		log.Printf("Failed to update user last login after WebAuthn callback: %v", err)
	}

	response := gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   365 * 24 * 60 * 60,
		"first_login":  isFirstLogin,
		"tenant_id":    tenantIDStr,
		"email":        user.Email,
	}

	// Audit log: WebAuthn callback login successful
	middlewares.Audit(c, "enduser", user.ID.String(), "webauthn_login", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":       user.Email,
			"tenant_id":   tenantIDStr,
			"first_login": isFirstLogin,
			"mfa_method":  "webauthn",
		},
	})

	c.JSON(http.StatusOK, response)
}

// VerifyLoginOTP handles OTP verification for login
// @Summary Verify login OTP
// @Description Verifies OTP for multi-factor authentication during login
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.LoginVerifyOTPInput true "OTP verification data"
// @Success 200 {object} models.LoginVerifyOTPResponse "OTP verified successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid OTP"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/verify-login-otp [post]
func (euac *EndUserAuthController) VerifyLoginOTP(c *gin.Context) {
	var input models.LoginVerifyOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	input.Email = strings.ToLower(input.Email)

	otpEntry, err := euac.otpRepo.GetValidOTP(input.Email, input.OTP)
	if err != nil {
		log.Printf("End-user login OTP invalid for %s: %v", input.Email, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	if err := euac.otpRepo.VerifyOTP(otpEntry.ID); err != nil {
		log.Printf("Failed to mark end-user OTP as verified: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
		return
	}

	tenantUUID, err := uuid.Parse(input.TenantID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	var user models.User
	if err := tenantDB.Where("LOWER(email) = LOWER(?)", input.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	clientIDStr := ""
	if user.ClientID != uuid.Nil {
		clientIDStr = user.ClientID.String()
	}

	token, err := euac.generateJWTToken(tenantIDStr, clientIDStr, user.Email, user.TenantDomain, &user.ID, tenantDB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	now := time.Now()
	if err := tenantDB.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login", now).Error; err != nil {
		log.Printf("Failed to update user last login after OTP verification: %v", err)
	}

	// Audit log: OTP verification successful
	middlewares.Audit(c, "enduser", user.ID.String(), "otp_verify", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":     user.Email,
			"tenant_id": tenantIDStr,
		},
	})

	c.JSON(http.StatusOK, models.LoginVerifyOTPResponse{
		Message: "OTP verified successfully",
		Token:   token,
	})
}

// ResendOTP handles OTP resend requests
// @Summary Resend OTP
// @Description Resends OTP for registration or login verification
// @Tags End-User Authentication
// @Accept json
// @Produce json
// @Param input body models.ResendOTPInput true "Resend OTP request data"
// @Success 200 {object} map[string]string "OTP resent successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/resend-otp [post]
func (euac *EndUserAuthController) ResendOTP(c *gin.Context) {
	var input models.ResendOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	pendingReg, err := euac.pendingRepo.GetPendingRegistration(input.Email)
	if err != nil {
		log.Printf("ResendOTP: no pending registration for %s: %v", input.Email, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "No pending registration found. Please initiate registration again."})
		return
	}

	if err := euac.generateAndSendOTP(input.Email); err != nil {
		log.Printf("ResendOTP: failed to send OTP to %s: %v", input.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resend OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    "OTP resent successfully",
		"email":      input.Email,
		"expires_at": pendingReg.ExpiresAt,
	})
}

// WebAuthnRegister handles WebAuthn registration for end-users
func (euac *EndUserAuthController) WebAuthnRegister(c *gin.Context) {
	var input models.WebAuthnRegistrationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	tenant, err := euac.tenantRepo.GetTenantByTenantID(input.TenantID.String())
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	tenantIDStr := tenant.TenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	var user models.User
	if err := tenantDB.Where("LOWER(email) = LOWER(?)", input.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to access tenant database"})
		return
	}

	if len(input.CredentialID) > 0 && len(input.PublicKey) > 0 {
		credentialQuery := `
			INSERT INTO credentials (
				client_id, credential_id, public_key, attestation_type,
				aaguid, sign_count, transports, backup_eligible, backup_state
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (credential_id) DO UPDATE SET
				public_key = EXCLUDED.public_key,
				attestation_type = EXCLUDED.attestation_type,
				aaguid = EXCLUDED.aaguid,
				sign_count = EXCLUDED.sign_count,
				transports = EXCLUDED.transports,
				backup_eligible = EXCLUDED.backup_eligible,
				backup_state = EXCLUDED.backup_state,
				updated_at = NOW()
		`

		clientID := user.ClientID
		if clientID == uuid.Nil {
			clientID = tenant.TenantID
		}

		if _, err := sqlDB.Exec(
			credentialQuery,
			clientID,
			input.CredentialID,
			input.PublicKey,
			input.AttestationType,
			input.AAGUID,
			input.SignCount,
			pq.Array(input.Transports),
			input.BackupEligible,
			input.BackupState,
		); err != nil {
			log.Printf("Failed to store WebAuthn credential: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store WebAuthn credential"})
			return
		}

		now := time.Now()
		methodData, _ := json.Marshal(map[string]interface{}{
			"type":        "webauthn",
			"registered":  now.UTC(),
			"backup":      input.BackupEligible,
			"transports":  input.Transports,
			"attestation": input.AttestationType,
		})

		userID := user.ID
		if _, err := sqlDB.Exec(
			`INSERT INTO mfa_methods (user_id, client_id, method_type, is_primary, verified, enabled, method_data, created_at, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (user_id, client_id, method_type) DO UPDATE SET
			 	is_primary = EXCLUDED.is_primary,
				verified = EXCLUDED.verified,
				enabled = EXCLUDED.enabled,
				method_data = EXCLUDED.method_data,
				updated_at = EXCLUDED.updated_at`,
			userID,
			clientID,
			"webauthn",
			true,
			true,
			true,
			datatypes.JSON(methodData),
			now,
			now,
		); err != nil {
			log.Printf("Failed to upsert WebAuthn MFA method: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store WebAuthn MFA method"})
			return
		}

		if err := tenantDB.Model(&models.User{}).Where("id = ?", user.ID).Updates(map[string]interface{}{
			"mfa_enabled": true,
			"mfa_method":  pq.StringArray([]string{"webauthn"}),
			"updated_at":  now,
		}).Error; err != nil {
			log.Printf("Failed to update user MFA flags: %v", err)
		}
	}

	clientIDStr := ""
	if user.ClientID != uuid.Nil {
		clientIDStr = user.ClientID.String()
	}

	isFirstLogin := user.LastLogin == nil

	token, err := euac.generateJWTToken(tenantIDStr, clientIDStr, user.Email, user.TenantDomain, &user.ID, tenantDB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	now := time.Now()
	if err := tenantDB.Model(&models.User{}).Where("id = ?", user.ID).Update("last_login", now).Error; err != nil {
		log.Printf("Failed to update user last login after WebAuthn registration: %v", err)
	}

	// Audit log: WebAuthn registration successful
	middlewares.Audit(c, "enduser", user.ID.String(), "webauthn_register", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"email":       user.Email,
			"tenant_id":   tenantIDStr,
			"mfa_enabled": true,
			"mfa_methods": []string{"webauthn"},
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"access_token": token,
		"token_type":   "Bearer",
		"expires_in":   365 * 24 * 60 * 60,
		"first_login":  isFirstLogin,
		"tenant_id":    tenantIDStr,
		"email":        user.Email,
		"mfa_enabled":  true,
		"mfa_methods":  []string{"webauthn"},
	})
}

// WebAuthnMFALoginStatus handles WebAuthn MFA login status
func (euac *EndUserAuthController) WebAuthnMFALoginStatus(c *gin.Context) {
	var input models.CustomLoginStatus
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	clientUUID, err := uuid.Parse(input.ClientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid client ID format"})
		return
	}

	tenantUUID, err := euac.tenantMapping(clientUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to map tenant: %v", err)})
		return
	}

	tenantIDStr := tenantUUID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to connect to tenant database"})
		return
	}

	var user models.User
	if err := tenantDB.Where("LOWER(email) = LOWER(?) AND client_id = ?", input.Email, clientUUID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch user"})
		return
	}

	if !user.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	isFirstLogin := user.LastLogin == nil

	sqlDB, err := tenantDB.DB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to inspect MFA state"})
		return
	}

	rows, err := sqlDB.Query(`
		SELECT method_type, is_primary
		FROM mfa_methods
		WHERE user_id = $1 AND client_id = $2 AND verified = true
		ORDER BY is_primary DESC, created_at ASC`,
		user.ID, clientUUID)
	if err != nil {
		log.Printf("Failed to query MFA methods for user %s: %v", input.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to inspect MFA state"})
		return
	}
	defer rows.Close()

	var mfaMethods []map[string]interface{}
	defaultMethod := ""

	for rows.Next() {
		var methodType string
		var isPrimary bool
		if err := rows.Scan(&methodType, &isPrimary); err == nil {
			mfaMethods = append(mfaMethods, map[string]interface{}{
				"method_type": methodType,
				"is_primary":  isPrimary,
			})
			if defaultMethod == "" || isPrimary {
				defaultMethod = methodType
			}
		}
	}

	// Track if re-registration is needed for current domain
	requiresRegistration := false

	// If no MFA methods found in mfa_methods table, check legacy tables as fallback
	if len(mfaMethods) == 0 {
		log.Printf("DEBUG EndUser: No MFA methods found in mfa_methods table for user %s, checking legacy tables", input.Email)

		// First, check for TOTP secrets (TOTP has priority because it's domain-independent)
		totpQuery := `SELECT COUNT(*) FROM totp_secrets WHERE user_id = $1 AND tenant_id = $2 AND is_verified = true`
		var totpCount int
		if err := sqlDB.QueryRow(totpQuery, user.ID, tenantUUID).Scan(&totpCount); err == nil && totpCount > 0 {
			log.Printf("DEBUG EndUser: Found %d TOTP secrets for user %s", totpCount, input.Email)
			// User has TOTP configured but not in mfa_methods table
			mfaMethods = append(mfaMethods, map[string]interface{}{
				"method_type": "totp",
				"is_primary":  true,
			})
			defaultMethod = "totp"
		} else {
			// No TOTP found, check for WebAuthn credentials
			// Get current domain from request to check RP ID specific credentials
			currentDomain := c.Request.Host
			if idx := strings.Index(currentDomain, ":"); idx != -1 {
				currentDomain = currentDomain[:idx] // strip port
			}
			log.Printf("DEBUG EndUser: No TOTP found, checking credentials for current domain/RP ID: %s", currentDomain)

			// Check for credentials matching the current RP ID (domain)
			credQuery := `SELECT COUNT(*) FROM credentials
			              WHERE (user_id = $1 OR client_id = $2)
			              AND (rp_id = $3 OR rp_id IS NULL)`
			var credCount int
			if err := sqlDB.QueryRow(credQuery, user.ID, user.ClientID, currentDomain).Scan(&credCount); err == nil && credCount > 0 {
				log.Printf("DEBUG EndUser: Found %d credentials for RP ID '%s' for user %s", credCount, currentDomain, input.Email)
				// User has credentials registered but not in mfa_methods table
				mfaMethods = append(mfaMethods, map[string]interface{}{
					"method_type": "webauthn",
					"is_primary":  true,
				})
				defaultMethod = "webauthn"
			} else {
				// Check if user has credentials on other domains
				otherDomainsQuery := `SELECT COUNT(*) FROM credentials
				                      WHERE (user_id = $1 OR client_id = $2)
				                      AND rp_id IS NOT NULL
				                      AND rp_id != $3`
				var otherCredCount int
				if err := sqlDB.QueryRow(otherDomainsQuery, user.ID, user.ClientID, currentDomain).Scan(&otherCredCount); err == nil && otherCredCount > 0 {
					log.Printf("DEBUG EndUser: User %s has %d credentials on other domains but not on %s - requires re-registration",
						input.Email, otherCredCount, currentDomain)
					// User has credentials on other domains but not this one
					requiresRegistration = true
				}
			}
		}
	}

	// Build response
	response := gin.H{
		"email":        input.Email,
		"tenant_id":    tenantIDStr,
		"client_id":    clientUUID.String(),
		"first_login":  isFirstLogin,
		"mfa_required": len(mfaMethods) > 0,
		"mfa_method":   defaultMethod,
		"mfa_methods":  mfaMethods,
	}

	// Add requires_registration flag if user needs to re-register on this domain
	if requiresRegistration {
		response["requires_registration"] = true
		response["message"] = "WebAuthn credentials required for this domain. Please complete registration."
		log.Printf("DEBUG EndUser: Returning requires_registration=true for user %s on domain %s", input.Email, c.Request.Host)
	}

	c.JSON(http.StatusOK, response)
}

// generateJWTToken generates JWT token for end-users
func (euac *EndUserAuthController) generateAndSendOTP(email string) error {
	otp, err := utils.GenerateOTP()
	if err != nil {
		return fmt.Errorf("failed to generate OTP: %w", err)
	}

	if err := euac.otpRepo.DeleteOTPsByEmail(email); err != nil {
		log.Printf("generateAndSendOTP: failed to delete existing OTPs for %s: %v", email, err)
	}

	entry := models.OTPEntry{
		Email:     email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		Verified:  false,
	}

	if err := euac.otpRepo.CreateOTP(&entry); err != nil {
		return fmt.Errorf("failed to save OTP: %w", err)
	}

	if err := utils.SendOTPEmail(email, otp); err != nil {
		// FIX: Don't delete OTP on email failure - the OTP is still valid
		// and the email might still be delivered despite the error
		log.Printf("generateAndSendOTP: failed to send email to %s, but OTP remains valid: %v", email, err)
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}

func (euac *EndUserAuthController) generateJWTToken(tenantID, clientID, emailID, tenantDomain string, userID *uuid.UUID, tenantDB interface{}) (string, error) {
	// Collect scopes for potential inclusion in token (though auth-manager fetches from DB)
	scopes := []string{"read", "write"}

	if userID != nil {
		userIDStr := userID.String()

		if tenantDB != nil {
			switch db := tenantDB.(type) {
			case *gorm.DB:
				if sqlDB, err := db.DB(); err == nil && sqlDB != nil {
					permSvc := services.NewPermissionService(sqlDB)
					_, dbScopes := permSvc.GetUserPermissions(userIDStr, tenantID), permSvc.GetUserScopes(userIDStr, tenantID)
					if len(dbScopes) > 0 {
						scopes = dbScopes
					}
				}
			case *sql.DB:
				permSvc := services.NewPermissionService(db)
				_, dbScopes := permSvc.GetUserPermissions(userIDStr, tenantID), permSvc.GetUserScopes(userIDStr, tenantID)
				if len(dbScopes) > 0 {
					scopes = dbScopes
				}
			}
		}
	}

	// Use centralized auth-manager token service
	if userID == nil {
		// For cases without userID, create a temporary one
		tempID := uuid.New()
		userID = &tempID
	}

	return config.TokenService.GenerateEndUserToken(
		*userID,
		tenantID,
		clientID,
		emailID,
		scopes,
		365*24*time.Hour,
	)
}

// GetAuthChallenge generates a challenge for anti-replay protection
// @Summary Get authentication challenge
// @Description Generates a server-issued challenge for use in login requests to prevent replay attacks
// @Tags End User Authentication
// @Accept json
// @Produce json
// @Success 200 {object} models.AuthChallenge "Challenge generated successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/enduser/challenge [get]
func (euac *EndUserAuthController) GetAuthChallenge(c *gin.Context) {
	challenge, err := euac.antiReplayService.GenerateChallenge()
	if err != nil {
		log.Printf("ERROR: Failed to generate auth challenge: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate challenge"})
		return
	}

	log.Printf("INFO: Auth challenge generated: %s, expires at: %v", challenge.Challenge, challenge.ExpiresAt)

	c.JSON(http.StatusOK, gin.H{
		"challenge":  challenge.Challenge,
		"expires_at": challenge.ExpiresAt.Unix(),
		"created_at": challenge.CreatedAt.Unix(),
	})
}
