package admin

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	// authrepo "github.com/authsec-ai/auth-manager/pkg/repo"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/monitoring"
	"github.com/authsec-ai/authsec/services"
	"github.com/authsec-ai/authsec/utils"
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type AdminAuthController struct {
	adminUserRepo     *database.AdminUserRepository
	adminTenantRepo   *database.AdminTenantRepository
	otpRepo           *database.OTPRepository
	pendingRepo       *database.PendingRegistrationRepository
	antiReplayService *services.AntiReplayService
}

// RegisterInput represents the input for admin user registration
type RegisterInput struct {
	Email        string `json:"email" binding:"required,email"`
	Password     string `json:"password" binding:"required,min=10"`
	Name         string `json:"name" binding:"required"`
	TenantDomain string `json:"tenant_domain" binding:"required"`
}

// ForgotPasswordInput represents the input for forgot password
type ForgotPasswordInput struct {
	Email string `json:"email" binding:"required,email"`
}

// ResetPasswordInput represents the input for password reset
type ResetPasswordInput struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
} // NewAdminAuthController creates a new admin auth controller
func NewAdminAuthController() (*AdminAuthController, error) {
	db := config.GetDatabase()
	if db == nil {
		return nil, fmt.Errorf("database not initialized")
	}

	redisClient := config.GetRedisClient()
	if redisClient == nil {
		return nil, fmt.Errorf("redis client not initialized")
	}

	return &AdminAuthController{
		adminUserRepo:     database.NewAdminUserRepository(db),
		adminTenantRepo:   database.NewAdminTenantRepository(db),
		otpRepo:           database.NewOTPRepository(db),
		pendingRepo:       database.NewPendingRegistrationRepository(db),
		antiReplayService: services.NewAntiReplayService(redisClient),
	}, nil
}

// AdminLogin handles admin user login
// @Summary Admin user login
// @Description Authenticates an admin user and returns a JWT token for admin operations
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body models.AdminLoginInput true "Admin login credentials"
// @Success 200 {object} models.LoginResponse "Successful login response with JWT token"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid credentials or account disabled"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/login [post]
func (aac *AdminAuthController) AdminLogin(c *gin.Context) {
	startTime := time.Now()
	requestID := c.GetString("request_id")
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var input models.AdminLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		// Audit: Failed login - invalid input
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", "", "admin_login", clientIP, userAgent, false, "invalid input: "+err.Error())
		}
		monitoring.RecordAuthFailure("admin", "invalid_input", "admin")

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if input.Email == "" || input.Password == "" {
		// Audit: Failed login - missing credentials
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", "", "admin_login", clientIP, userAgent, false, "missing email or password")
		}
		monitoring.RecordAuthFailure("admin", "missing_credentials", "admin")

		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	// ========== ANTI-REPLAY ATTACK VALIDATION ==========
	// Validate nonce and timestamp to prevent replay attacks
	if input.Nonce != "" || input.Timestamp != 0 {
		log.Printf("[AdminLogin][Anti-Replay] Validating request for email: %s, nonce: %s, timestamp: %d", input.Email, input.Nonce, input.Timestamp)

		secureReq := &models.SecureLoginRequest{
			Email:        input.Email,
			Password:     input.Password,
			TenantDomain: input.TenantDomain,
			Nonce:        input.Nonce,
			Timestamp:    input.Timestamp,
			Challenge:    input.Challenge,
			Signature:    input.Signature,
		}

		if err := aac.antiReplayService.ValidateLoginRequest(secureReq); err != nil {
			log.Printf("[AdminLogin][Anti-Replay] REPLAY ATTACK DETECTED for email %s: %v", input.Email, err)
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin", "", "admin_login", clientIP, userAgent, false, "replay attack detected: "+err.Error())
			}
			monitoring.RecordAuthFailure("admin", "replay_attack", "admin")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Request validation failed",
				"hint":  "Possible replay attack detected. Please generate a new request with fresh nonce and timestamp.",
			})
			return
		}
		log.Printf("[AdminLogin][Anti-Replay] Validation SUCCESS for email: %s", input.Email)
	} else {
		// Optional: Log warning for requests without anti-replay protection (legacy clients)
		log.Printf("[AdminLogin][Anti-Replay] WARNING: Login request without anti-replay protection for email: %s", input.Email)
	}
	// ========== END ANTI-REPLAY VALIDATION ==========

	// Get admin user by email - with optional tenant_domain filtering for tenant isolation
	fmt.Printf("[AdminLogin] Searching for admin user with email: %s, tenant_domain: %s in MAIN DATABASE\n", input.Email, input.TenantDomain)

	var adminUser *models.AdminUser
	var err error

	if input.TenantDomain != "" {
		// TENANT ISOLATION: When tenant_domain is provided, only find users belonging to that domain
		adminUser, err = aac.adminUserRepo.GetAdminUserByEmailAndTenantDomain(input.Email, input.TenantDomain)
	} else {
		// Global admin lookup (no tenant restriction) - for super admins
		adminUser, err = aac.adminUserRepo.GetAdminUserByEmail(input.Email)
	}

	if err != nil {
		// Audit: Failed login - user not found
		fmt.Printf("[AdminLogin] User NOT FOUND in main database: email=%s, tenant_domain=%s, error=%v\n", input.Email, input.TenantDomain, err)
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", "", "admin_login", clientIP, userAgent, false, "user not found in main database")
		}
		monitoring.RecordAuthFailure("admin", "user_not_found", "admin")

		errorMsg := "Invalid credentials"
		if input.TenantDomain != "" {
			errorMsg = "Invalid credentials for this tenant"
		}
		c.JSON(http.StatusUnauthorized, gin.H{"error": errorMsg, "hint": "User not found. Ensure you are using the correct tenant domain."})
		return
	}

	// TENANT ISOLATION: Verify user's tenant_domain matches the requested tenant_domain
	if input.TenantDomain != "" && adminUser.TenantDomain != "" && adminUser.TenantDomain != input.TenantDomain {
		fmt.Printf("[AdminLogin] TENANT ISOLATION VIOLATION: user=%s belongs to domain=%s but tried to login to domain=%s\n",
			adminUser.Email, adminUser.TenantDomain, input.TenantDomain)
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "tenant domain mismatch")
		}
		monitoring.RecordAuthFailure("admin", "tenant_domain_mismatch", "admin")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials for this tenant"})
		return
	}

	// Check if admin user is active
	if !adminUser.Active {
		// Audit: Failed login - account disabled
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "account disabled")
		}
		monitoring.RecordAuthFailure("admin", "account_disabled", "admin")

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	// Check if account is locked (lockout period: 30 minutes)
	if adminUser.AccountLockedAt != nil {
		lockoutDuration := 30 * time.Minute
		if time.Since(*adminUser.AccountLockedAt) < lockoutDuration {
			remainingTime := lockoutDuration - time.Since(*adminUser.AccountLockedAt)
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "account locked")
			}
			monitoring.RecordAuthFailure("admin", "account_locked", "admin")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Account is temporarily locked due to multiple failed login attempts",
				"message": fmt.Sprintf("Please try again in %d minutes or reset your password", int(remainingTime.Minutes())+1),
			})
			return
		}
		// Lockout period expired, reset fields
		if err := aac.adminUserRepo.UpdateAdminUser(adminUser.ID, map[string]interface{}{
			"failed_login_attempts": 0,
			"account_locked_at":     nil,
		}); err != nil {
			fmt.Printf("Failed to reset lockout fields: %v\n", err)
		}
		adminUser.FailedLoginAttempts = 0
		adminUser.AccountLockedAt = nil
	}

	// Check if password reset is required
	if adminUser.PasswordResetRequired {
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "password reset required")
		}
		monitoring.RecordAuthFailure("admin", "password_reset_required", "admin")

		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Password reset required",
			"message": "Your account requires a password reset. Please use the forgot password flow.",
		})
		return
	}

	// Verify password
	if !adminUser.CheckPassword(input.Password) {
		// Increment failed login attempts
		adminUser.FailedLoginAttempts++

		updates := map[string]interface{}{
			"failed_login_attempts": adminUser.FailedLoginAttempts,
		}

		// Lock account after 3 failed attempts and require password reset
		if adminUser.FailedLoginAttempts >= 3 {
			now := time.Now()
			updates["account_locked_at"] = now
			updates["password_reset_required"] = true
			if err := aac.adminUserRepo.UpdateAdminUser(adminUser.ID, updates); err != nil {
				fmt.Printf("Failed to lock account: %v\n", err)
			}

			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "account locked after 3 failed attempts")
			}
			monitoring.RecordAuthFailure("admin", "account_locked", "admin")

			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Account locked due to multiple failed login attempts",
				"message": "Your account has been locked for security. Please reset your password to unlock.",
			})
			return
		}

		if err := aac.adminUserRepo.UpdateAdminUser(adminUser.ID, updates); err != nil {
			fmt.Printf("Failed to update failed attempts: %v\n", err)
		}

		// Audit: Failed login - invalid password
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "invalid password")
		}
		monitoring.RecordAuthFailure("admin", "invalid_password", "admin")

		remainingAttempts := 3 - adminUser.FailedLoginAttempts
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "Invalid credentials",
			"message": fmt.Sprintf("Invalid password. %d attempt(s) remaining before account lockout.", remainingAttempts),
		})
		return
	}

	// Successful login - reset failed attempts
	if adminUser.FailedLoginAttempts > 0 {
		if err := aac.adminUserRepo.UpdateAdminUser(adminUser.ID, map[string]interface{}{
			"failed_login_attempts": 0,
			"account_locked_at":     nil,
		}); err != nil {
			fmt.Printf("Failed to reset failed attempts: %v\n", err)
		}
	}

	// Update last login
	if err := aac.adminUserRepo.UpdateLastLogin(adminUser.ID); err != nil {
		// Log error but don't fail the login
		fmt.Printf("Failed to update last login: %v\n", err)
	}

	// Check if temporary password has expired
	if adminUser.TemporaryPassword && adminUser.TemporaryPasswordExpiresAt != nil {
		if time.Now().After(*adminUser.TemporaryPasswordExpiresAt) {
			// Audit: Failed login - temporary password expired
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "temporary password expired")
			}
			monitoring.RecordAuthFailure("admin", "temporary_password_expired", "admin")

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Temporary password has expired. Please use forgot password to reset."})
			return
		}
	}

	// Determine MFA requirements
	mfaRequired := adminUser.MFAEnabled
	webauthnRequired := false
	otpRequired := false
	var mfaMethod string
	var methods []string

	if mfaRequired && len(adminUser.MFAMethod) > 0 {
		// Use default method if set, otherwise use first method
		if adminUser.MFADefaultMethod != "" {
			mfaMethod = adminUser.MFADefaultMethod
		} else {
			mfaMethod = adminUser.MFAMethod[0]
		}

		// Copy all available methods
		methods = make([]string, len(adminUser.MFAMethod))
		copy(methods, adminUser.MFAMethod)

		// Check what type of MFA is required
		for _, method := range adminUser.MFAMethod {
			if method == "webauthn" {
				webauthnRequired = true
			}
			if method == "otp" || method == "totp" {
				otpRequired = true
			}
		}
	}

	// Determine tenant_domain to return (custom domain support)
	// Prefer Origin header (where user is accessing from) over Host header (API backend)
	currentDomain := c.GetHeader("Origin")
	if currentDomain != "" {
		// Parse Origin to extract domain
		currentDomain = strings.TrimPrefix(currentDomain, "https://")
		currentDomain = strings.TrimPrefix(currentDomain, "http://")
		if idx := strings.Index(currentDomain, "/"); idx != -1 {
			currentDomain = currentDomain[:idx] // strip path
		}
	} else {
		// Fallback to Host header if Origin not present
		currentDomain = c.Request.Host
	}
	if idx := strings.Index(currentDomain, ":"); idx != -1 {
		currentDomain = currentDomain[:idx] // strip port
	}

	// Skip custom domain check for API backend domains (infrastructure domains)
	isAPIBackendDomain := strings.Contains(currentDomain, ".api.") ||
		currentDomain == "api.authsec.dev" ||
		strings.HasPrefix(currentDomain, "dev") && strings.Contains(currentDomain, ".authsec.dev")

	// Check tenant_domains table for primary domain
	var tenantDomainToReturn string
	if adminUser.TenantID != nil {
		tenantDomainRepo := database.NewTenantDomainsRepository(config.GetDatabase())
		primaryDomain, err := tenantDomainRepo.GetPrimaryDomainByTenantID(*adminUser.TenantID)
		if err != nil {
			// No custom domain configured, use stored tenant_domain
			tenantDomainToReturn = adminUser.TenantDomain
			log.Printf("DEBUG AdminLogin: Using stored tenant_domain=%s", tenantDomainToReturn)
		} else if primaryDomain != nil {
			tenantDomainToReturn = primaryDomain.Domain
			log.Printf("DEBUG AdminLogin: primaryTenantDomain=%s from tenant_domains table", tenantDomainToReturn)

			// Check if current domain is a verified custom domain for this tenant
			// Skip this check if the current domain is an API backend domain
			if !isAPIBackendDomain && currentDomain != primaryDomain.Domain {
				customDomain, err := tenantDomainRepo.GetDomainByHostname(currentDomain)
				if err == nil && customDomain != nil && customDomain.TenantID == *adminUser.TenantID && customDomain.IsVerified {
					// User is on a verified custom domain - use it
					tenantDomainToReturn = currentDomain
					log.Printf("DEBUG AdminLogin: Using custom domain %s", currentDomain)
				}
			} else if isAPIBackendDomain {
				log.Printf("DEBUG AdminLogin: Skipping custom domain check for API backend domain: %s", currentDomain)
			}
		}
	} else {
		tenantDomainToReturn = adminUser.TenantDomain
	}
	log.Printf("DEBUG AdminLogin: Returning tenantDomain=%s", tenantDomainToReturn)

	// Build response with MFA information
	response := gin.H{
		"tenant_id":         adminUser.TenantID,
		"tenant_domain":     tenantDomainToReturn,
		"email":             adminUser.Email,
		"first_login":       adminUser.LastLogin == nil,
		"otp_required":      otpRequired,
		"webauthn_required": webauthnRequired,
		"mfa_required":      mfaRequired,
		"mfa_method":        mfaMethod,
		"methods":           methods,
	}

	// If MFA is required, don't generate token yet
	// Client should call MFA verification endpoint next
	if !mfaRequired {
		// Generate JWT token for admin (only if MFA not required)
		token, err := aac.generateAdminJWTToken(adminUser)
		if err != nil {
			// Audit: Failed login - token generation error
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, false, "token generation failed: "+err.Error())
			}
			monitoring.RecordAuthFailure("admin", "token_generation_failed", "admin")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}
		response["token"] = token
	}

	// Include temporary password status if applicable
	if adminUser.TemporaryPassword {
		response["password_change_required"] = true
		response["temporary_password"] = true // Keep for backwards compatibility
		response["message"] = "Please change your password on first login"
	}

	// Audit: Successful login
	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "admin", adminUser.ID.String(), "admin_login", clientIP, userAgent, true, "")
	}
	monitoring.RecordAuthRequest("admin", "success", "admin")

	// Record metrics
	monitoring.RecordDBQuery("SELECT", "users", "admin", time.Since(startTime))

	c.JSON(http.StatusOK, response)
}

// AdminLoginHybrid handles admin user login with hybrid database lookup
// @Summary Admin login (hybrid mode - checks both main and tenant databases)
// @Description Authenticates admin user by first checking main database, then tenant database if tenant_domain is provided
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body models.AdminLoginInput true "Admin login credentials with optional tenant_domain"
// @Success 200 {object} object "Login successful"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 401 {object} map[string]string "Unauthorized"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/login-hybrid [post]
func (aac *AdminAuthController) AdminLoginHybrid(c *gin.Context) {
	startTime := time.Now()
	requestID := c.GetString("request_id")
	clientIP := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")

	var input models.AdminLoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", "", "admin_login_hybrid", clientIP, userAgent, false, "invalid input: "+err.Error())
		}
		monitoring.RecordAuthFailure("admin-hybrid", "invalid_input", "admin")

		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if input.Email == "" || input.Password == "" {
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", "", "admin_login_hybrid", clientIP, userAgent, false, "missing email or password")
		}
		monitoring.RecordAuthFailure("admin-hybrid", "missing_credentials", "admin")

		c.JSON(http.StatusBadRequest, gin.H{"error": "Email and password are required"})
		return
	}

	// STEP 1: Try to find user in MAIN DATABASE as admin
	fmt.Printf("[AdminLoginHybrid] Step 1: Checking MAIN DATABASE for admin user: %s\n", input.Email)
	adminUser, err := aac.adminUserRepo.GetAdminUserByEmail(input.Email)

	if err == nil && adminUser != nil {
		// Found in main database as admin user
		fmt.Printf("[AdminLoginHybrid] User FOUND in MAIN DATABASE: email=%s, user_id=%s, active=%v\n",
			adminUser.Email, adminUser.ID, adminUser.Active)

		// Check if admin user is active
		if !adminUser.Active {
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", adminUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "account disabled (main DB)")
			}
			monitoring.RecordAuthFailure("admin-hybrid", "account_disabled", "admin")

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
			return
		}

		// Verify password
		if !adminUser.CheckPassword(input.Password) {
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", adminUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "invalid password (main DB)")
			}
			monitoring.RecordAuthFailure("admin-hybrid", "invalid_password", "admin")

			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Update last login
		if err := aac.adminUserRepo.UpdateLastLogin(adminUser.ID); err != nil {
			fmt.Printf("Failed to update last login: %v\n", err)
		}

		// Check if temporary password has expired
		if adminUser.TemporaryPassword && adminUser.TemporaryPasswordExpiresAt != nil {
			if time.Now().After(*adminUser.TemporaryPasswordExpiresAt) {
				if config.AuditLogger != nil {
					config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", adminUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "temporary password expired (main DB)")
				}
				monitoring.RecordAuthFailure("admin-hybrid", "temporary_password_expired", "admin")

				c.JSON(http.StatusUnauthorized, gin.H{"error": "Temporary password has expired. Please use forgot password to reset."})
				return
			}
		}

		// Generate JWT token for admin
		token, err := aac.generateAdminJWTToken(adminUser)
		if err != nil {
			if config.AuditLogger != nil {
				config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", adminUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "token generation failed (main DB): "+err.Error())
			}
			monitoring.RecordAuthFailure("admin-hybrid", "token_generation_failed", "admin")

			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		response := gin.H{
			"tenant_id":    "admin",
			"email":        adminUser.Email,
			"first_login":  adminUser.LastLogin == nil,
			"otp_required": false,
			"token":        token,
			"source":       "main_database",
			"user_type":    "global_admin",
		}

		if adminUser.TemporaryPassword {
			response["password_change_required"] = true
			response["temporary_password"] = true
			response["message"] = "Please change your password on first login"
		}

		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", adminUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, true, "authenticated from main database")
		}
		monitoring.RecordAuthRequest("admin-hybrid", "success", "admin")
		monitoring.RecordDBQuery("SELECT", "users", "admin", time.Since(startTime))

		c.JSON(http.StatusOK, response)
		return
	}

	// STEP 2: User not found in main database, try TENANT DATABASE if tenant_domain provided
	fmt.Printf("[AdminLoginHybrid] User NOT FOUND in main database: %s\n", input.Email)

	if input.TenantDomain == "" {
		fmt.Printf("[AdminLoginHybrid] No tenant_domain provided, cannot check tenant database\n")
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", "", "admin_login_hybrid", clientIP, userAgent, false, "user not found in main DB and no tenant_domain provided")
		}
		monitoring.RecordAuthFailure("admin-hybrid", "user_not_found", "admin")

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
			"hint":  "User not found in main database. Provide tenant_domain to search tenant database.",
		})
		return
	}

	fmt.Printf("[AdminLoginHybrid] Step 2: Checking TENANT DATABASE for tenant: %s\n", input.TenantDomain)

	// Get tenant by domain from database
	db := config.GetDatabase()
	var tenant models.Tenant
	query := `
		SELECT id, tenant_id, tenant_db, email, username, password_hash,
			provider, provider_id, avatar, name, source, status, last_login,
			created_at, updated_at, tenant_domain
		FROM tenants
		WHERE tenant_domain = $1
	`

	err = db.QueryRow(query, input.TenantDomain).Scan(
		&tenant.ID,
		&tenant.TenantID,
		&tenant.TenantDB,
		&tenant.Email,
		&tenant.Username,
		&tenant.PasswordHash,
		&tenant.Provider,
		&tenant.ProviderID,
		&tenant.Avatar,
		&tenant.Name,
		&tenant.Source,
		&tenant.Status,
		&tenant.LastLogin,
		&tenant.CreatedAt,
		&tenant.UpdatedAt,
		&tenant.TenantDomain,
	)

	if err != nil {
		fmt.Printf("[AdminLoginHybrid] Tenant NOT FOUND: domain=%s, error=%v\n", input.TenantDomain, err)
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", "", "admin_login_hybrid", clientIP, userAgent, false, "tenant not found: "+input.TenantDomain)
		}
		monitoring.RecordAuthFailure("admin-hybrid", "tenant_not_found", "admin")

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials", "hint": "Tenant not found"})
		return
	}

	fmt.Printf("[AdminLoginHybrid] Tenant FOUND: domain=%s, tenant_id=%s\n", input.TenantDomain, tenant.ID)

	// Get tenant database connection
	tenantIDStr := tenant.ID.String()
	tenantGormDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		fmt.Printf("[AdminLoginHybrid] Failed to connect to tenant database: tenant_id=%s, error=%v\n", tenant.ID, err)
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", "", "admin_login_hybrid", clientIP, userAgent, false, "tenant database connection failed")
		}
		monitoring.RecordAuthFailure("admin-hybrid", "tenant_db_connection_failed", "admin")

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}

	// Get raw SQL connection from GORM
	tenantSQLDB, err := tenantGormDB.DB()
	if err != nil {
		fmt.Printf("[AdminLoginHybrid] Failed to get SQL connection: error=%v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant database connection"})
		return
	}

	// Create DBConnection wrapper
	tenantDB := &database.DBConnection{DB: tenantSQLDB}

	// Create user repository for tenant database
	userRepo := database.NewUserRepository(tenantDB)

	// Search for user in tenant database
	fmt.Printf("[AdminLoginHybrid] Searching for user in tenant database: email=%s\n", input.Email)
	tenantUser, err := userRepo.GetUserByEmail(input.Email)
	if err != nil {
		fmt.Printf("[AdminLoginHybrid] User NOT FOUND in tenant database: email=%s, error=%v\n", input.Email, err)
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", "", "admin_login_hybrid", clientIP, userAgent, false, "user not found in tenant database")
		}
		monitoring.RecordAuthFailure("admin-hybrid", "user_not_found_tenant_db", "admin")

		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid credentials",
			"hint":  "User not found in main database or tenant database",
		})
		return
	}

	fmt.Printf("[AdminLoginHybrid] User FOUND in TENANT DATABASE: email=%s, user_id=%s, active=%v\n",
		tenantUser.Email, tenantUser.ID, tenantUser.Active)

	// Check if user is active
	if !tenantUser.Active {
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", tenantUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "account disabled (tenant DB)")
		}
		monitoring.RecordAuthFailure("admin-hybrid", "account_disabled", "tenant")

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Account is disabled"})
		return
	}

	// Verify password using tenant user's CheckPassword method
	if !tenantUser.CheckPassword(input.Password) {
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", tenantUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "invalid password (tenant DB)")
		}
		monitoring.RecordAuthFailure("admin-hybrid", "invalid_password", "tenant")

		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Check if user has admin role in tenant context
	// TODO: Implement role check in tenant database
	fmt.Printf("[AdminLoginHybrid] NOTE: Role verification for tenant users not yet implemented\n")

	// Generate JWT token for tenant user using centralized auth-manager token service
	token, err := config.TokenService.GenerateTenantUserToken(
		tenantUser.ID,
		tenant.ID,
		tenantUser.ProjectID,
		tenantUser.Email,
		24*time.Hour,
	)
	if err != nil {
		if config.AuditLogger != nil {
			config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", tenantUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, false, "token generation failed (tenant DB): "+err.Error())
		}
		monitoring.RecordAuthFailure("admin-hybrid", "token_generation_failed", "tenant")

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	// Determine tenant_domain to return (custom domain support)
	// Prefer Origin header (where user is accessing from) over Host header (API backend)
	currentDomain := c.GetHeader("Origin")
	if currentDomain != "" {
		// Parse Origin to extract domain
		currentDomain = strings.TrimPrefix(currentDomain, "https://")
		currentDomain = strings.TrimPrefix(currentDomain, "http://")
		if idx := strings.Index(currentDomain, "/"); idx != -1 {
			currentDomain = currentDomain[:idx] // strip path
		}
	} else {
		// Fallback to Host header if Origin not present
		currentDomain = c.Request.Host
	}
	if idx := strings.Index(currentDomain, ":"); idx != -1 {
		currentDomain = currentDomain[:idx] // strip port
	}

	// Skip custom domain check for API backend domains (infrastructure domains)
	isAPIBackendDomain := strings.Contains(currentDomain, ".api.") ||
		currentDomain == "api.authsec.dev" ||
		strings.HasPrefix(currentDomain, "dev") && strings.Contains(currentDomain, ".authsec.dev")

	tenantDomainToReturn := input.TenantDomain // default to input
	tenantDomainRepo := database.NewTenantDomainsRepository(config.GetDatabase())
	primaryDomain, err := tenantDomainRepo.GetPrimaryDomainByTenantID(tenant.ID)
	if err == nil && primaryDomain != nil {
		tenantDomainToReturn = primaryDomain.Domain
		// Check if current domain is a verified custom domain
		// Skip this check if the current domain is an API backend domain
		if !isAPIBackendDomain && currentDomain != primaryDomain.Domain {
			customDomain, err := tenantDomainRepo.GetDomainByHostname(currentDomain)
			if err == nil && customDomain != nil && customDomain.TenantID == tenant.ID && customDomain.IsVerified {
				tenantDomainToReturn = currentDomain
				log.Printf("DEBUG AdminLoginHybrid: Using custom domain %s", currentDomain)
			}
		} else if isAPIBackendDomain {
			log.Printf("DEBUG AdminLoginHybrid: Skipping custom domain check for API backend domain: %s", currentDomain)
		}
	}

	response := gin.H{
		"tenant_id":     tenant.ID.String(),
		"email":         tenantUser.Email,
		"first_login":   tenantUser.LastLogin == nil,
		"otp_required":  false,
		"token":         token,
		"source":        "tenant_database",
		"user_type":     "tenant_user",
		"tenant_domain": tenantDomainToReturn,
	}

	if config.AuditLogger != nil {
		config.AuditLogger.LogAuthentication(requestID, "admin-hybrid", tenantUser.ID.String(), "admin_login_hybrid", clientIP, userAgent, true, "authenticated from tenant database")
	}
	monitoring.RecordAuthRequest("admin-hybrid", "success", "tenant")
	monitoring.RecordDBQuery("SELECT", "users", tenant.ID.String(), time.Since(startTime))

	c.JSON(http.StatusOK, response)
}

// AdminRegister handles admin user registration
// @Summary Register a new admin user and tenant
// @Description Creates a new admin user account and associated tenant in the global database
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body RegisterInput true "Admin registration data"
// @Success 201 {object} object "Admin user and tenant created successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 409 {object} map[string]string "Conflict - admin user or tenant already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// AdminRegister handles admin user registration initiation
// @Summary Initiate admin user registration
// @Description Initiates admin user registration by creating a pending registration with OTP
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body RegisterInput true "Admin registration details"
// @Success 201 {object} map[string]string "Registration initiated successfully"
// @Failure 400 {object} map[string]string "Bad request"
// @Failure 409 {object} map[string]string "User or tenant already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/register [post]
func (aac *AdminAuthController) AdminRegister(c *gin.Context) {
	var input RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate input
	if input.Email == "" || input.Password == "" || input.TenantDomain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email, password, and tenant_domain are required"})
		return
	}

	// Check if admin user already exists
	_, err := aac.adminUserRepo.GetAdminUserByEmail(input.Email)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin user already exists"})
		return
	}

	// Check if tenant domain already exists
	tenantDomain := fmt.Sprintf("%s.%s", strings.ToLower(input.TenantDomain), config.AppConfig.TenantDomainSuffix)
	db := config.GetDatabase()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tenants WHERE tenant_domain = $1", tenantDomain).Scan(&count)
	if err != nil {
		log.Printf("Failed to check tenant domain existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tenant domain"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Tenant domain already exists"})
		return
	}

	// Check if pending registration already exists
	existingPending, err := aac.pendingRepo.GetPendingRegistration(input.Email)
	if err == nil && existingPending != nil {
		// Regenerate OTP for existing pending registration
		otp, err := utils.GenerateOTP()
		if err != nil {
			log.Printf("Failed to generate OTP: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
			return
		}

		// Delete existing OTPs
		if err := aac.otpRepo.DeleteOTPsByEmail(input.Email); err != nil {
			log.Printf("Warning - failed to delete old OTPs: %v", err)
		}

		// Create new OTP entry
		otpEntry := models.OTPEntry{
			Email:     input.Email,
			OTP:       otp,
			ExpiresAt: time.Now().Add(30 * time.Minute),
			Verified:  false,
		}

		if err := aac.otpRepo.CreateOTP(&otpEntry); err != nil {
			log.Printf("Failed to create OTP: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP"})
			return
		}

		// Send OTP via email (non-blocking)
		if err := utils.SendOTPEmail(input.Email, otp); err != nil {
			log.Printf("Failed to send OTP email: %v", err)
			// Don't fail - OTP is still created
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "OTP regenerated and sent to your email",
			"email":   input.Email,
			"otp":     otp, // Include OTP in response for testing (remove in production)
		})
		return
	}

	// Hash the password for storage in pending registration
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		log.Printf("Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate UUIDs for admin user
	adminUUID := uuid.New()

	// Create pending registration
	pendingReg := &models.PendingRegistration{
		Email:        input.Email,
		PasswordHash: hashedPassword,
		TenantID:     adminUUID,
		ProjectID:    uuid.Nil,  // Admin users don't belong to a specific project
		ClientID:     adminUUID, // Same as tenant for admin
		TenantDomain: tenantDomain,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // 24 hours for admin registration
	}

	if err := aac.pendingRepo.CreatePendingRegistration(pendingReg); err != nil {
		log.Printf("Failed to create pending registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pending registration"})
		return
	}

	// Generate OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Printf("Failed to generate OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Also allow a fixed OTP for testing purposes
	const fixedOTP = "1111111"
	const fixedOTP6 = "111111" // 6-digit version for backwards compatibility

	// Create OTP entry
	otpEntry := models.OTPEntry{
		Email:     input.Email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		Verified:  false,
	}

	if err := aac.otpRepo.CreateOTP(&otpEntry); err != nil {
		log.Printf("Failed to create OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP"})
		return
	}

	// Insert fixed OTP as alternative codes for testing
	if err := aac.otpRepo.CreateOTP(&models.OTPEntry{
		Email:     input.Email,
		OTP:       fixedOTP,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Verified:  false,
	}); err != nil {
		log.Printf("Failed to create fixed OTP override (7-digit): %v", err)
	}

	if err := aac.otpRepo.CreateOTP(&models.OTPEntry{
		Email:     input.Email,
		OTP:       fixedOTP6,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		Verified:  false,
	}); err != nil {
		log.Printf("Failed to create fixed OTP override (6-digit): %v", err)
	}

	// Send OTP via email (non-blocking - log warning but don't fail registration)
	if err := utils.SendOTPEmail(input.Email, otp); err != nil {
		log.Printf("Failed to send OTP email: %v", err)
		// Don't fail registration - OTP is still created and can be retrieved from DB for testing
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Admin registration initiated. Please check your email for OTP to complete registration.",
		"email":   input.Email,
		"otp":     otp, // Include OTP in response for testing purposes (remove in production)
	})
}

// assignAdminRoleToUser assigns the admin role to a user with the tenant scope populated
func (aac *AdminAuthController) assignAdminRoleToUser(userID uuid.UUID, tenantID uuid.UUID) error {
	db := config.GetDatabase()
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	roleID, err := aac.adminUserRepo.EnsureAdminRole(tenantID)
	if err != nil {
		return fmt.Errorf("failed to ensure admin role: %w", err)
	}

	// Insert into role_bindings (user_roles is deprecated)
	// scope_type and scope_id are NULL for tenant-wide role assignments
	_, err = db.Exec(`
		INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
		SELECT $1, $2, $3, $4, NULL, NULL, $5, $5
		WHERE NOT EXISTS (
			SELECT 1 FROM role_bindings
			WHERE tenant_id = $2 AND user_id = $3 AND role_id = $4 AND scope_type IS NULL AND scope_id IS NULL
		)
	`, uuid.New(), tenantID, userID, roleID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to assign admin role: %w", err)
	}

	return nil
}

// AdminForgotPassword handles admin password reset request
// @Summary Request admin password reset
// @Description Initiates a password reset process for admin users
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body ForgotPasswordInput true "Admin email for password reset"
// @Success 200 {object} map[string]string "Reset code sent if email exists"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Router /uflow/auth/admin/forgot-password [post]
func (aac *AdminAuthController) AdminForgotPassword(c *gin.Context) {
	var input ForgotPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	start := time.Now()

	// Anti-enumeration: normalize response timing
	defer func() {
		time.Sleep(time.Until(start.Add(1 * time.Second)))
	}()

	// ✅ Check if admin user exists
	adminUser, err := aac.adminUserRepo.GetAdminUserByEmail(input.Email)
	if err != nil {
		// Differentiate between "not found" vs real DB error
		if errors.Is(err, sql.ErrNoRows) {
			// Don't reveal user existence
			c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset code has been sent"})
			return
		}
		// Real DB failure — stop early
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	log.Printf("AdminForgotPassword: located admin user %s for %s", adminUser.ID, input.Email)

	// ✅ Optional: Add rate limiting to prevent abuse
	// if tooManyRequests(input.Email, c.ClientIP()) {
	//     c.JSON(http.StatusTooManyRequests, gin.H{"message": "Please wait before requesting another code"})
	//     return
	// }

	// ✅ Generate OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// ✅ Delete existing OTPs
	if err := aac.otpRepo.DeleteOTPsByEmail(input.Email); err != nil {
		log.Printf("AdminForgotPassword: failed to delete existing OTPs for %s: %v", input.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// ✅ Create OTP entry
	otpEntry := models.OTPEntry{
		Email:     input.Email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		Verified:  false,
	}

	if err := aac.otpRepo.CreateOTP(&otpEntry); err != nil {
		log.Printf("AdminForgotPassword: failed to persist OTP for %s: %v", input.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	// ✅ Send OTP email — silently handle failures
	if err := utils.SendOTPEmail(input.Email, otp); err != nil {
		log.Printf("AdminForgotPassword: failed to send OTP email to %s: %v", input.Email, err)
		// FIX: Don't delete OTP on email failure - the OTP is still valid
		// and the email might still be delivered despite the error
		log.Printf("AdminForgotPassword: OTP remains valid in database despite email error for %s", input.Email)
		// Don't expose internal error to avoid leaking user existence
		c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset code has been sent"})
		return
	}

	log.Printf("AdminForgotPassword: OTP dispatched successfully to %s", input.Email)

	// ✅ Generic success response
	c.JSON(http.StatusOK, gin.H{"message": "If the email exists, a reset code has been sent"})
}

// AdminVerifyOTP handles OTP verification for admin password reset
// @Summary Verify OTP for admin password reset
// @Description Verifies the OTP code sent for admin password reset
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body models.VerifyOTPInput true "OTP verification data"
// @Success 200 {object} map[string]string "OTP verified successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid OTP"
// @Router /uflow/auth/admin/forgot-password/verify-otp [post]
func (aac *AdminAuthController) AdminVerifyOTP(c *gin.Context) {
	var input models.VerifyOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	// Check if using hardcoded test OTP
	isHardcodedOTP := input.OTP == "1111111" || input.OTP == "111111"

	// Log for debugging
	if isHardcodedOTP {
		log.Printf("AdminVerifyOTP: Using hardcoded OTP %s for email %s", input.OTP, input.Email)
	}

	// Verify OTP (allow fixed override as well)
	otpEntry, err := aac.otpRepo.GetValidOTP(input.Email, input.OTP)
	if err != nil && !isHardcodedOTP {
		log.Printf("AdminVerifyOTP: OTP validation failed for %s: %v", input.Email, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	// Mark OTP as verified
	// For hardcoded OTP: create a database entry and mark it verified so password reset flow works
	if isHardcodedOTP {
		// Delete any existing OTPs for this email first
		aac.otpRepo.DeleteOTPsByEmail(input.Email)

		// Create a verified OTP entry in the database
		verifiedEntry := &models.OTPEntry{
			Email:     input.Email,
			OTP:       input.OTP,
			ExpiresAt: time.Now().Add(30 * time.Minute),
			Verified:  true, // Mark as verified immediately
		}
		if err := aac.otpRepo.CreateOTP(verifiedEntry); err != nil {
			log.Printf("AdminVerifyOTP: Failed to create verified hardcoded OTP entry: %v", err)
			// Continue anyway - this is just for password reset flow
		}
		log.Printf("AdminVerifyOTP: Created verified hardcoded OTP entry for %s", input.Email)
	} else if err == nil {
		// Normal OTP - mark existing entry as verified
		if err := aac.otpRepo.VerifyOTP(otpEntry.ID); err != nil {
			log.Printf("AdminVerifyOTP: Failed to mark OTP as verified for %s: %v", input.Email, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
			return
		}
	}

	log.Printf("AdminVerifyOTP: OTP verified successfully for %s (hardcoded: %v)", input.Email, isHardcodedOTP)
	c.JSON(http.StatusOK, gin.H{"message": "OTP verified successfully"})
}

// AdminResetPassword handles admin password reset
// @Summary Reset admin password
// @Description Completes the password reset process for admin users
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body ResetPasswordInput true "New password and verification data"
// @Success 200 {object} map[string]string "Password reset successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid verification"
// @Router /uflow/auth/admin/forgot-password/reset [post]
func (aac *AdminAuthController) AdminResetPassword(c *gin.Context) {
	var input ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	// Check if there's a verified OTP for this email
	_, err := aac.otpRepo.GetVerifiedOTP(input.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No verified reset code found. Please request a new password reset"})
		return
	}

	// Get admin user
	adminUser, err := aac.adminUserRepo.GetAdminUserByEmail(input.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Admin user not found"})
		return
	}

	// Update password
	adminUser.Password = input.NewPassword
	if err := adminUser.HashPassword(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Update the user with new password hash
	updates := map[string]interface{}{
		"password_hash":                 adminUser.PasswordHash,
		"temporary_password":            false,
		"temporary_password_expires_at": nil,
		"failed_login_attempts":         0,
		"account_locked_at":             nil,
		"password_reset_required":       false,
		"updated_at":                    time.Now(),
	}

	if err := aac.adminUserRepo.UpdateAdminUser(adminUser.ID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// Clean up the used OTP
	aac.otpRepo.DeleteOTPsByEmail(input.Email)

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// AdminCompleteRegistration handles admin registration completion after OTP verification
// @Summary Complete admin registration
// @Description Completes admin registration by verifying OTP and creating admin user and tenant
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body models.VerifyOTPInput true "OTP verification data"
// @Success 200 {object} map[string]string "Admin registration completed successfully"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid OTP"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/complete-registration [post]
func (aac *AdminAuthController) AdminCompleteRegistration(c *gin.Context) {
	var input models.VerifyOTPInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	input.Email = strings.ToLower(input.Email)

	// Verify OTP (allow fixed override)
	fixedOTP := "1111111"
	fixedOTP2 := "111111" // Backwards compatibility
	fixedOverride := false
	otpEntry, err := aac.otpRepo.GetValidOTP(input.Email, input.OTP)
	if err != nil {
		if input.OTP != fixedOTP && input.OTP != fixedOTP2 {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired OTP"})
			return
		}
		fixedOverride = true
	} else if input.OTP == fixedOTP || input.OTP == fixedOTP2 {
		// If using hardwired OTP, skip VerifyOTPTx since the entry is synthetic
		fixedOverride = true
	}

	// Get pending registration
	pendingReg, err := aac.pendingRepo.GetPendingRegistration(input.Email)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Registration session expired. Please initiate registration again"})
		return
	}

	// Begin transaction for registration completion
	db := config.GetDatabase()
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to begin registration transaction"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Mark OTP as verified unless using fixed override
	if !fixedOverride {
		if err := aac.otpRepo.VerifyOTPTx(tx, otpEntry.ID); err != nil {
			tx.Rollback()
			log.Printf("Failed to mark OTP as verified: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify OTP"})
			return
		}
	}

	// Step 1: Create tenant first (required for FK constraints on projects table)
	tenantRepo := database.NewTenantRepository(db)
	tenantDBName := fmt.Sprintf("tenant_%s", strings.ReplaceAll(pendingReg.TenantID.String(), "-", "_"))
	tenant := models.Tenant{
		ID:           pendingReg.TenantID,
		TenantID:     pendingReg.TenantID,
		Email:        pendingReg.Email,
		PasswordHash: pendingReg.PasswordHash,
		Name:         fmt.Sprintf("%s %s", pendingReg.FirstName, pendingReg.LastName),
		Provider:     "local",
		Source:       "admin_registration",
		Status:       "active",
		TenantDomain: pendingReg.TenantDomain,
		TenantDB:     tenantDBName, // Set the tenant database name
	}

	if err := tenantRepo.CreateTenantTx(tx, &tenant); err != nil {
		tx.Rollback()
		log.Printf("Failed to create tenant: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant"})
		return
	}

	// Step 2: Generate project ID and create project (now tenant exists for FK)
	defaultProjectID := uuid.New()
	projectInsertGlobal := `INSERT INTO projects (id, tenant_id, name, description, user_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())`
	if _, err := tx.Exec(projectInsertGlobal, defaultProjectID, pendingReg.TenantID, "Default Project", "Default project for admin user", pendingReg.TenantID); err != nil {
		tx.Rollback()
		log.Printf("Failed to create project in global database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Step 3: Create admin user (stored in regular users table)
	userRepo := database.NewUserRepository(db)
	username := pendingReg.Email
	adminUser := models.ExtendedUser{
		User: sharedmodels.User{
			ID:           pendingReg.TenantID, // Use tenant ID as user ID for admin
			Email:        pendingReg.Email,
			Name:         fmt.Sprintf("%s %s", pendingReg.FirstName, pendingReg.LastName),
			PasswordHash: pendingReg.PasswordHash,
			ClientID:     pendingReg.ClientID,
			TenantID:     pendingReg.TenantID,
			ProjectID:    defaultProjectID, // Use the newly created project ID
			TenantDomain: pendingReg.TenantDomain,
			Provider:     "local",
			ProviderID:   pendingReg.Email,
			Username:     &username,
			ProviderData: datatypes.JSON("{}"),
			Active:       true,
		},
	}

	if err := userRepo.CreateUserTx(tx, &adminUser); err != nil {
		tx.Rollback()
		log.Printf("Failed to create admin user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin user"})
		return
	}

	// Create tenant_mappings entry in global database for client_id to tenant_id mapping
	tenantMappingInsert := `INSERT INTO tenant_mappings (tenant_id, client_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (client_id) DO NOTHING`
	if _, err := tx.Exec(tenantMappingInsert, pendingReg.TenantID, pendingReg.TenantID); err != nil {
		tx.Rollback()
		log.Printf("Failed to create tenant mapping: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant mapping"})
		return
	}

	log.Printf("Created tenant_mappings entry: tenant_id=%s, client_id=%s", pendingReg.TenantID.String(), pendingReg.ClientID.String())

	// Assign admin role BEFORE committing transaction (tenant-scoped)
	roleID, err := database.NewAdminSeedRepository(config.GetDatabase()).EnsureAdminRoleAndPermissionsTx(tx, pendingReg.TenantID)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to ensure admin role/permissions: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create admin role"})
		return
	}

	// Insert into role_bindings within transaction (user_roles is deprecated)
	// scope_type and scope_id are NULL for tenant-wide role assignments
	bindingID := uuid.New()
	_, err = tx.Exec(`
		INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
		SELECT $1, $2, $3, $4, NULL, NULL, $5, $5
		WHERE NOT EXISTS (
			SELECT 1 FROM role_bindings
			WHERE tenant_id = $2 AND user_id = $3 AND role_id = $4 AND scope_type IS NULL AND scope_id IS NULL
		)
	`, bindingID, pendingReg.TenantID, adminUser.ID, roleID, time.Now())
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to assign admin role: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign admin role"})
		return
	}
	// Delete pending registration
	if err := aac.pendingRepo.DeletePendingRegistrationsByEmailTx(tx, input.Email); err != nil {
		tx.Rollback()
		log.Printf("Failed to delete pending registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete registration"})
		return
	}

	// Commit global transaction so the migration service can see the tenant record
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to complete registration"})
		return
	}

	// Create tenant database and run migrations (must happen after commit so migration service can query the tenant)
	// These are post-commit operations — global state is already committed, so failures here are
	// logged as warnings and don't block the registration response.
	log.Printf("Creating tenant database for: %s", pendingReg.TenantID.String())

	defaultClientID := pendingReg.ClientID
	tenantDBReady := false
	tenantDBService, err := database.NewTenantDBService(db, config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, config.AppConfig.DBPort)
	if err != nil {
		log.Printf("Warning: Failed to create tenant DB service: %v", err)
		log.Printf("Tenant database %s will need to be provisioned manually or on next login", tenantDBName)
	} else {
		if _, err := tenantDBService.CreateTenantDatabase(pendingReg.TenantID.String()); err != nil {
			log.Printf("Warning: Failed to create tenant database: %v", err)
			log.Printf("Tenant database %s will need to be provisioned manually or on next login", tenantDBName)
		} else {
			tenantDBReady = true
		}
	}

	// Seed tenant database with default records (only if DB + migrations succeeded)
	if tenantDBReady {
		tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
			config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, tenantDBName, config.AppConfig.DBPort)

		tenantDB, err := sql.Open("postgres", tenantDSN)
		if err != nil {
			log.Printf("Warning: Failed to connect to tenant database: %v", err)
		} else {
			defer tenantDB.Close()

			// Create default client with Hydra client ID
			hydraClientID := fmt.Sprintf("%s-main-client", defaultClientID.String())
			clientInsert := `INSERT INTO clients (id, client_id, tenant_id, project_id, owner_id, org_id, name, description, hydra_client_id, active, created_at, updated_at)
				VALUES ($1, $1, $2, $3, $4, $2, $5, $6, $7, true, NOW(), NOW())`
			if _, err := tenantDB.Exec(clientInsert, defaultClientID, pendingReg.TenantID, defaultProjectID, pendingReg.TenantID, "Default Client", "Default client for admin user", hydraClientID); err != nil {
				log.Printf("Warning: Failed to create default client in tenant database: %v", err)
			}

			// Create tenant record in tenant database (required for FK constraint on projects table)
			tenantInsert := `INSERT INTO tenants (id, tenant_id, email, password_hash, name, provider, source, status, tenant_domain, tenant_db, created_at, updated_at)
				VALUES ($1, $1, $2, $3, $4, 'local', 'admin_registration', 'active', $5, $6, NOW(), NOW())`
			if _, err := tenantDB.Exec(tenantInsert, pendingReg.TenantID, pendingReg.Email, pendingReg.PasswordHash,
				fmt.Sprintf("%s %s", pendingReg.FirstName, pendingReg.LastName), pendingReg.TenantDomain, tenantDBName); err != nil {
				log.Printf("Warning: Failed to create tenant record in tenant database: %v", err)
			}

			// Create default project in tenant database
			projectInsert := `INSERT INTO projects (id, tenant_id, name, description, user_id, active, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())`
			if _, err := tenantDB.Exec(projectInsert, defaultProjectID, pendingReg.TenantID, "Default Project", "Default project for admin user", pendingReg.TenantID); err != nil {
				log.Printf("Warning: Failed to create default project in tenant database: %v", err)
			}

			// Create corresponding end user account in tenant database
			if err := aac.createEndUserInTenantDB(tenantDB, &adminUser, defaultClientID, defaultProjectID, pendingReg); err != nil {
				log.Printf("Warning: Failed to create end user account in tenant database: %v", err)
			} else {
				log.Printf("Successfully created end user account in tenant database for admin: %s", pendingReg.Email)
			}
		}
	}

	// Save secret to Vault and register with Hydra
	secretID, err := config.SaveSecretToVault(pendingReg.TenantID.String(), pendingReg.ProjectID.String(), pendingReg.TenantID.String())
	if err != nil {
		log.Printf("Warning: Failed to save secret to vault: %v", err)
		log.Printf("Admin registration will continue without Vault secret storage for tenant: %s", pendingReg.TenantID.String())
		// Don't block admin registration - they can still use the system without Vault integration
		secretID = "" // Clear secretID so we don't attempt Hydra registration
	}

	// Register user with Hydra only when we have a secret to use
	if secretID != "" {
		if err := services.RegisterClientWithHydra(pendingReg.ClientID.String(), secretID, pendingReg.Email, pendingReg.TenantID.String(), pendingReg.TenantDomain); err != nil {
			log.Printf("Warning: Failed to register client with Hydra: %v", err)
			log.Printf("Admin registration will continue without Hydra client registration for tenant: %s", pendingReg.TenantID.String())
			// Don't block admin registration - they can still use the system without OAuth integration
		}
	} else {
		log.Printf("Skipping Hydra registration for tenant %s because no Vault secret was stored", pendingReg.TenantID.String())
	}

	log.Printf("Admin registration completed for email: %s", input.Email)

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Admin registration completed successfully",
		"email":     input.Email,
		"tenant_id": pendingReg.TenantID.String(),
		"user_id":   adminUser.ID.String(),
	})
}

// generateAdminJWTToken generates JWT token for admin users using auth-manager token service
func (aac *AdminAuthController) generateAdminJWTToken(adminUser *models.AdminUser) (string, error) {
	if adminUser == nil {
		return "", errors.New("admin user is required")
	}

	// Determine project_id for token
	var projectID uuid.UUID
	if adminUser.ProjectID != nil && *adminUser.ProjectID != uuid.Nil {
		projectID = *adminUser.ProjectID
	} else {
		// Default to a zero UUID for admin users without project
		projectID = uuid.Nil
	}

	// Fetch admin roles from database
	var roles []string
	if adminUser.TenantID != nil && *adminUser.TenantID != uuid.Nil {
		// Get roles for this admin user in their tenant
		rolesFromDB, err := aac.adminUserRepo.GetAdminRoles(adminUser.ID, *adminUser.TenantID)
		if err == nil {
			roles = rolesFromDB
		} else {
			log.Printf("Warning: Failed to fetch admin roles for user %s: %v", adminUser.ID, err)
			// Continue with empty roles - auth-manager will fetch from DB on each request
		}
	}

	// If no roles found in DB, default to "admin" role for backward compatibility
	if len(roles) == 0 {
		roles = []string{"admin"}
	}

	// Use centralized auth-manager token service with actual tenant data
	token, err := config.TokenService.GenerateAdminToken(
		adminUser.ID,
		adminUser.Email,
		projectID,
		adminUser.TenantID,     // Pass actual tenant_id
		adminUser.TenantDomain, // Pass tenant_domain
		roles,                  // Pass admin roles
	)
	if err != nil {
		return "", fmt.Errorf("failed to generate admin token: %w", err)
	}

	return token, nil
}

// AdminLoginPrecheck validates admin email and returns tenant context
// @Summary Precheck admin email before login
// @Description Validates if admin user exists and returns tenant context for login flow
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body models.AdminPrecheckInput true "Admin email for precheck"
// @Success 200 {object} models.AdminPrecheckResponse "Email validation result with tenant context"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/login/precheck [post]
func (aac *AdminAuthController) AdminLoginPrecheck(c *gin.Context) {
	var input models.AdminPrecheckInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get admin user with configured providers
	user, providers, err := aac.adminUserRepo.GetAdminUserWithProviders(input.Email)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("ERROR: Failed to get admin user for precheck: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate email"})
		return
	}

	// If user doesn't exist, return bootstrap flow
	if user == nil {
		c.JSON(http.StatusOK, models.AdminPrecheckResponse{
			Exists:           false,
			NextStep:         "bootstrap",
			RequiresPassword: false,
		})
		return
	}

	// User exists - determine which domain to return
	// Return the CURRENT domain they're accessing from (preserves custom domain UX)
	// For WebAuthn RP ID coordination, the actual login will return original tenant_domain
	currentDomain := input.CurrentDomain
	if currentDomain == "" {
		// Try Origin header (where user is accessing from)
		origin := c.GetHeader("Origin")
		if origin != "" {
			currentDomain = strings.TrimPrefix(origin, "https://")
			currentDomain = strings.TrimPrefix(currentDomain, "http://")
			if idx := strings.Index(currentDomain, "/"); idx != -1 {
				currentDomain = currentDomain[:idx] // strip path
			}
		} else {
			// Fallback to Host header if Origin not present (backwards compatibility)
			currentDomain = c.Request.Host
		}
	}
	log.Printf("DEBUG AdminLoginPrecheck: currentDomain=%s (from Origin or Host), email=%s", currentDomain, input.Email)

	// Get tenant's primary domain from tenant_domains table (source of truth)
	// If not found, fall back to tenants.tenant_domain for backward compatibility
	var primaryTenantDomain string
	if user.TenantID != nil {
		tenantDomainRepo := database.NewTenantDomainsRepository(config.GetDatabase())
		primaryDomain, err := tenantDomainRepo.GetPrimaryDomainByTenantID(*user.TenantID)
		if err != nil {
			log.Printf("WARN: No primary domain in tenant_domains table, falling back to tenants.tenant_domain: %v", err)
			// Fall back to querying tenants table
			tenantRepo := database.NewTenantRepository(config.GetDatabase())
			tenant, tenantErr := tenantRepo.GetTenantByTenantID(user.TenantID.String())
			if tenantErr != nil {
				log.Printf("ERROR: Failed to get tenant for precheck fallback: %v", tenantErr)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tenant domain information"})
				return
			}
			primaryTenantDomain = tenant.TenantDomain
			log.Printf("DEBUG AdminLoginPrecheck: Using fallback tenant_domain=%s", primaryTenantDomain)
		} else if primaryDomain != nil {
			primaryTenantDomain = primaryDomain.Domain
			log.Printf("DEBUG AdminLoginPrecheck: primaryTenantDomain=%s from tenant_domains table", primaryTenantDomain)
		}
	}

	// Check if current domain is a verified custom domain for this tenant
	tenantDomainToReturn := primaryTenantDomain // default to primary domain

	// Skip custom domain check for API backend domains (infrastructure domains)
	isAPIBackendDomain := strings.Contains(currentDomain, ".api.") ||
		currentDomain == "api.authsec.dev" ||
		strings.HasPrefix(currentDomain, "dev") && strings.Contains(currentDomain, ".authsec.dev")

	if !isAPIBackendDomain && currentDomain != primaryTenantDomain && user.TenantID != nil {
		log.Printf("DEBUG AdminLoginPrecheck: Checking if %s is a verified custom domain", currentDomain)
		// Check if current domain is a verified custom domain
		tenantDomainRepo := database.NewTenantDomainsRepository(config.GetDatabase())
		customDomain, err := tenantDomainRepo.GetDomainByHostname(currentDomain)
		if err != nil {
			log.Printf("DEBUG AdminLoginPrecheck: GetDomainByHostname error: %v", err)
		} else if customDomain == nil {
			log.Printf("DEBUG AdminLoginPrecheck: No custom domain found for %s", currentDomain)
		} else {
			log.Printf("DEBUG AdminLoginPrecheck: Found domain - TenantID=%s, IsVerified=%v", customDomain.TenantID, customDomain.IsVerified)
		}
		if err == nil && customDomain != nil && customDomain.TenantID == *user.TenantID && customDomain.IsVerified {
			// User is on a verified custom domain - return it to prevent redirect
			tenantDomainToReturn = currentDomain
			log.Printf("DEBUG AdminLoginPrecheck: Using custom domain %s", currentDomain)
		}
	} else if isAPIBackendDomain {
		log.Printf("DEBUG AdminLoginPrecheck: Skipping custom domain check for API backend domain: %s", currentDomain)
	}
	log.Printf("DEBUG AdminLoginPrecheck: Returning tenantDomain=%s", tenantDomainToReturn)

	// Prepare login response
	response := models.AdminPrecheckResponse{
		Exists:             true,
		DisplayName:        user.Name,
		TenantDomain:       tenantDomainToReturn,
		NextStep:           "login",
		RequiresPassword:   true,
		AvailableProviders: providers,
	}

	if user.TenantID != nil {
		response.TenantID = user.TenantID.String()
	}

	c.JSON(http.StatusOK, response)
}

// AdminBootstrap creates a new tenant with admin user
// @Summary Bootstrap new tenant with admin user
// @Description Creates new tenant and admin user, sends OTP for verification
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Param input body models.AdminBootstrapInput true "Bootstrap details"
// @Success 201 {object} models.AdminBootstrapResponse "Bootstrap initiated, OTP sent"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 409 {object} map[string]string "Conflict - tenant or user already exists"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/login/bootstrap [post]
func (aac *AdminAuthController) AdminBootstrap(c *gin.Context) {
	var input models.AdminBootstrapInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate password confirmation if provided
	if input.ConfirmPassword != "" && input.Password != input.ConfirmPassword {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Passwords do not match"})
		return
	}

	// Validate tenant domain format
	tenantDomain := strings.ToLower(input.TenantDomain)
	if !strings.HasSuffix(tenantDomain, config.AppConfig.TenantDomainSuffix) {
		tenantDomain = fmt.Sprintf("%s.%s", tenantDomain, config.AppConfig.TenantDomainSuffix)
	}

	// Check if admin user already exists
	existingUser, err := aac.adminUserRepo.GetAdminUserByEmail(input.Email)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("ERROR: Failed to check existing admin user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate user"})
		return
	}
	if existingUser != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Admin user already exists"})
		return
	}

	// Check if tenant domain already exists
	db := config.GetDatabase()
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM tenants WHERE tenant_domain = $1", tenantDomain).Scan(&count)
	if err != nil {
		log.Printf("ERROR: Failed to check tenant domain: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check tenant domain"})
		return
	}
	if count > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "Tenant domain already exists"})
		return
	}

	// Check for existing pending registration
	existingPending, err := aac.pendingRepo.GetPendingRegistration(input.Email)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("ERROR: Failed to check pending registration: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check pending registration"})
		return
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		log.Printf("ERROR: Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	// Generate UUIDs
	tenantID := uuid.New()
	clientID := tenantID // Same as tenant for admin

	// If pending registration exists, update it; otherwise create new
	if existingPending != nil {
		// Update existing pending registration
		existingPending.PasswordHash = hashedPassword
		existingPending.TenantDomain = tenantDomain
		existingPending.TenantID = tenantID
		existingPending.ClientID = clientID
		existingPending.ExpiresAt = time.Now().Add(24 * time.Hour)

		if err := aac.pendingRepo.UpdatePendingRegistration(existingPending); err != nil {
			log.Printf("ERROR: Failed to update pending registration: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update pending registration"})
			return
		}
	} else {
		// Create new pending registration
		pendingReg := &models.PendingRegistration{
			Email:        input.Email,
			PasswordHash: hashedPassword,
			TenantID:     tenantID,
			ProjectID:    uuid.Nil,
			ClientID:     clientID,
			TenantDomain: tenantDomain,
			ExpiresAt:    time.Now().Add(24 * time.Hour),
		}

		if err := aac.pendingRepo.CreatePendingRegistration(pendingReg); err != nil {
			log.Printf("ERROR: Failed to create pending registration: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create pending registration"})
			return
		}
	}

	// Generate OTP
	otp, err := utils.GenerateOTP()
	if err != nil {
		log.Printf("ERROR: Failed to generate OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate OTP"})
		return
	}

	// Delete old OTPs for this email
	if err := aac.otpRepo.DeleteOTPsByEmail(input.Email); err != nil {
		log.Printf("WARN: Failed to delete old OTPs: %v", err)
	}

	// Create OTP entry
	otpEntry := models.OTPEntry{
		Email:     input.Email,
		OTP:       otp,
		ExpiresAt: time.Now().Add(30 * time.Minute),
		Verified:  false,
	}

	if err := aac.otpRepo.CreateOTP(&otpEntry); err != nil {
		log.Printf("ERROR: Failed to create OTP: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create OTP"})
		return
	}

	// Send OTP via email
	if err := utils.SendOTPEmail(input.Email, otp); err != nil {
		log.Printf("WARN: Failed to send OTP email: %v", err)
	} else {
		log.Printf("INFO: OTP email sent successfully to: %s", input.Email)
	}

	log.Printf("INFO: Bootstrap initiated for: %s, tenant: %s, OTP: %s", input.Email, tenantDomain, otp)

	c.JSON(http.StatusCreated, models.AdminBootstrapResponse{
		Message:      "Bootstrap initiated. Please check your email for OTP to complete registration.",
		Status:       "pending_verification",
		TenantID:     tenantID.String(),
		TenantDomain: tenantDomain,
	})
}

// GetAuthChallenge generates a challenge for anti-replay protection
// @Summary Get authentication challenge
// @Description Generates a server-issued challenge for use in login requests to prevent replay attacks
// @Tags Admin Authentication
// @Accept json
// @Produce json
// @Success 200 {object} models.AuthChallenge "Challenge generated successfully"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/auth/admin/challenge [get]
func (aac *AdminAuthController) GetAuthChallenge(c *gin.Context) {
	challenge, err := aac.antiReplayService.GenerateChallenge()
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

// createEndUserInTenantDB creates a corresponding end user account in the tenant database
// This allows admins to also authenticate as end users within their tenant
func (aac *AdminAuthController) createEndUserInTenantDB(tenantDB *sql.DB, adminUser *models.ExtendedUser, clientID, projectID uuid.UUID, pendingReg *models.PendingRegistration) error {
	// Create end user with same credentials as admin
	endUserInsert := `
		INSERT INTO users (id, client_id, tenant_id, project_id, email, name, username, 
			password_hash, tenant_domain, provider, provider_id, active, 
			created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, true, NOW(), NOW())
		ON CONFLICT (email, client_id) DO NOTHING
	`

	username := pendingReg.Email
	_, err := tenantDB.Exec(endUserInsert,
		adminUser.ID,        // Use same ID as admin user for consistency
		clientID,            // client_id
		pendingReg.TenantID, // tenant_id
		projectID,           // project_id
		pendingReg.Email,    // email
		fmt.Sprintf("%s %s", pendingReg.FirstName, pendingReg.LastName), // name
		username,                // username
		pendingReg.PasswordHash, // password_hash (same as admin)
		pendingReg.TenantDomain, // tenant_domain
		"local",                 // provider
		pendingReg.Email,        // provider_id
	)

	if err != nil {
		return fmt.Errorf("failed to insert end user in tenant database: %w", err)
	}

	log.Printf("Created end user account in tenant database: email=%s, user_id=%s", pendingReg.Email, adminUser.ID.String())
	return nil
}
