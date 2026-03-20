package platform

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/database"
	"github.com/authsec-ai/authsec/middlewares"
	sharedmodels "github.com/authsec-ai/sharedmodels"

	// authrepo "github.com/authsec-ai/auth-manager/pkg/repo"

	icp "github.com/authsec-ai/authsec/internal/clients/icp"
	"github.com/authsec-ai/authsec/models"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// safeUUID safely converts an interface{} to uuid.UUID, handling both uuid.UUID and string types.
func safeUUID(v interface{}) uuid.UUID {
	switch val := v.(type) {
	case uuid.UUID:
		return val
	case string:
		parsed, err := uuid.Parse(val)
		if err != nil {
			return uuid.Nil
		}
		return parsed
	default:
		return uuid.Nil
	}
}

func (oc *OIDCController) generateAdminJWTToken(adminUser *models.AdminUser) (string, error) {
	if adminUser == nil {
		return "", errors.New("admin user is required")
	}

	defaultSecret := os.Getenv("JWT_DEF_SECRET")
	if defaultSecret == "" {
		panic("CRITICAL: JWT_DEF_SECRET environment variable is not set. Cannot generate secure tokens.")
	}

	expectedIssuer := os.Getenv("AUTH_EXPECT_ISS")
	if expectedIssuer == "" {
		expectedIssuer = "authsec-ai/auth-manager"
	}

	expectedAudience := os.Getenv("AUTH_EXPECT_AUD")
	if expectedAudience == "" {
		expectedAudience = "authsec-api"
	}

	roles, scopes, resources, dbPerms, err := oc.adminUserRepo.GetAdminUserAccessContext(adminUser.ID)
	if err != nil {
		log.Printf("WARN: failed to load admin RBAC context for user %s: %v", adminUser.ID.String(), err)
	}
	if len(roles) == 0 {
		roles = []string{"admin"}
	}
	if scopes == nil {
		scopes = []string{}
	}
	if resources == nil {
		resources = []string{}
	}

	// Start with permissions from DB
	perms := make([]string, 0)
	if dbPerms != nil {
		perms = append(perms, dbPerms...)
	}

	// Add permissions derived from scopes (if any)
	// scopePerms := authrepo.FromScopes(scopes)
	// if len(scopePerms) > 0 {
	// 	perms = append(perms, scopePerms...)
	// }

	scopeString := strings.Join(scopes, " ")
	if len(perms) > 0 {
		scopeString += " " + strings.Join(perms, " ")
	}
	now := time.Now()

	var tenantID string
	if adminUser.TenantID != nil && *adminUser.TenantID != uuid.Nil {
		tenantID = adminUser.TenantID.String()
	} else {
		// Fallback for global admins without a specific tenant ID
		tenantID = "admin"
	}

	// Ultra-minimal token: identity only
	// Auth-manager fetches roles/permissions from DB via GetAuthz() on every request
	claims := jwt.MapClaims{
		"sub":       adminUser.ID.String(),
		"email":     adminUser.Email,
		"tenant_id": tenantID,
		"aud":       expectedAudience,
		"iss":       expectedIssuer,
		"iat":       now.Unix(),
		"exp":       now.Add(24 * time.Hour).Unix(),
	}

	// project_id is required for auth-manager GetAuthz() - default to tenant_id
	if adminUser.ProjectID != nil && *adminUser.ProjectID != uuid.Nil {
		claims["project_id"] = adminUser.ProjectID.String()
	} else {
		claims["project_id"] = tenantID // Default project_id to tenant_id for GetAuthz()
	}
	// client_id is optional but useful for context
	if adminUser.ClientID != nil && *adminUser.ClientID != uuid.Nil {
		claims["client_id"] = adminUser.ClientID.String()
	} else {
		claims["client_id"] = tenantID // Default client_id to tenant_id
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	token.Header["kid"] = "default"

	tokenString, err := token.SignedString([]byte(defaultSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// OIDCController handles OIDC authentication flows
type OIDCController struct {
	oidcService            *services.OIDCService
	tenantRepo             *database.AdminTenantRepository
	userRepo               *database.UserRepository
	adminUserRepo          *database.AdminUserRepository
	pendingRepo            *database.PendingRegistrationRepository
	tenantDBService        *database.TenantDBService
	icpProvisioningService *services.ICPProvisioningService
}

// NewOIDCController creates a new OIDC controller
func NewOIDCController() (*OIDCController, error) {
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

	// Initialize ICP client and provisioning service
	// Generate service-to-service JWT token for ICP
	cfg := config.GetConfig()
	icpToken, err := services.GenerateOIDCServiceToken()
	if err != nil {
		log.Printf("Warning: Failed to generate ICP service token: %v", err)
		// Continue without ICP service - it's optional
		icpToken = ""
	}

	var icpProvisioningService *services.ICPProvisioningService
	if icpToken != "" {
		icpClient := icp.NewClient(cfg.ICPServiceURL, icpToken)
		icpProvisioningService = services.NewICPProvisioningService(icpClient)
	}

	return &OIDCController{
		oidcService:            services.NewOIDCService(db),
		tenantRepo:             database.NewAdminTenantRepository(db),
		userRepo:               database.NewUserRepository(db),
		adminUserRepo:          database.NewAdminUserRepository(db),
		pendingRepo:            database.NewPendingRegistrationRepository(db),
		tenantDBService:        tenantDBService,
		icpProvisioningService: icpProvisioningService,
	}, nil
}

// Initiate handles unified OIDC flow - automatically determines register vs login
// @Summary Initiate OIDC flow (unified)
// @Description Starts OIDC flow. If tenant_domain is empty, uses "discover" mode to find existing user.
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body models.OIDCInitiateInput true "OIDC initiation request"
// @Success 200 {object} models.OIDCInitiateResponse
// @Failure 400 {object} map[string]string
// @Router /uflow/oidc/initiate [post]
func (oc *OIDCController) Initiate(c *gin.Context) {
	var input models.OIDCInitiateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize tenant domain
	input.TenantDomain = strings.ToLower(strings.TrimSpace(input.TenantDomain))

	var action string
	var tenantID *uuid.UUID

	// Case 1: No tenant domain provided (from app.authsec.dev) - DISCOVER mode
	if input.TenantDomain == "" {
		action = "discover"
		tenantID = nil
		log.Printf("OIDC: No tenant domain, initiating DISCOVER flow for provider '%s'", input.Provider)
	} else {
		// Validate tenant domain format when provided
		// Allow full custom domains (test.auth-sec.org) in addition to subdomain prefixes (mycompany)
		if !isValidTenantDomainOrCustomDomain(input.TenantDomain) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant domain format"})
			return
		}

		// Check if tenant exists - this determines register vs login
		log.Printf("OIDC Initiate: Looking up tenant for domain: %s", input.TenantDomain)
		existingTenant, err := oc.tenantRepo.GetTenantByDomain(input.TenantDomain)

		if err == nil && existingTenant != nil {
			// Tenant exists → LOGIN flow (user must exist in this tenant)
			action = "login"
			tenantID = &existingTenant.TenantID
			log.Printf("OIDC: Tenant '%s' found (tenant_id=%s), initiating LOGIN flow", input.TenantDomain, existingTenant.TenantID)
		} else {
			// Tenant doesn't exist → REGISTER flow (create new tenant)
			action = "register"
			tenantID = nil
			log.Printf("OIDC: Tenant '%s' not found (error: %v), initiating REGISTER flow", input.TenantDomain, err)
		}
	}

	// Set request host for callback URL (use API domain, e.g., dev.authsec.dev)
	oc.oidcService.SetRequestHost(c.Request.Host)
	log.Printf("DEBUG Initiate: Set requestHost='%s' for OIDC callback", c.Request.Host)

	// Capture origin domain for post-auth redirect (where user came from)
	origin := c.GetHeader("Origin")
	if origin == "" {
		origin = c.GetHeader("Referer")
		if origin != "" {
			// Extract domain from referer URL
			if parsedURL, err := url.Parse(origin); err == nil {
				origin = parsedURL.Host
			}
		}
	}
	if origin != "" {
		// Clean up origin (remove https:// prefix if present)
		origin = strings.TrimPrefix(origin, "https://")
		origin = strings.TrimPrefix(origin, "http://")
		oc.oidcService.SetRequestOrigin(origin)
		log.Printf("DEBUG Initiate: Set requestOrigin='%s' for post-auth redirect", origin)
	}

	// Initiate OIDC flow
	response, err := oc.oidcService.InitiateOIDCFlow(&input, action, tenantID)
	if err != nil {
		log.Printf("Failed to initiate OIDC flow: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"redirect_url": response.RedirectURL,
		"state":        response.State,
		"action":       action, // Tell frontend what will happen
	})
}

// CheckTenantExists checks if a tenant domain is available or taken
// @Summary Check tenant domain availability
// @Description Returns whether a tenant domain exists (for UI to show login vs register)
// @Tags OIDC
// @Param domain query string true "Tenant domain to check"
// @Success 200 {object} map[string]interface{}
// @Router /uflow/oidc/check-tenant [get]
func (oc *OIDCController) CheckTenantExists(c *gin.Context) {
	domain := strings.ToLower(strings.TrimSpace(c.Query("domain")))

	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Domain parameter required"})
		return
	}

	existingTenant, err := oc.tenantRepo.GetTenantByDomain(domain)
	exists := err == nil && existingTenant != nil

	c.JSON(http.StatusOK, gin.H{
		"domain": domain,
		"exists": exists,
		"action": map[bool]string{true: "login", false: "register"}[exists],
	})
}

// GetProviders returns list of available OIDC providers for login UI
// @Summary Get available OIDC providers
// @Description Returns list of active OIDC providers for display on login page
// @Tags OIDC
// @Produce json
// @Success 200 {object} models.OIDCProviderListResponse
// @Router /uflow/oidc/providers [get]
func (oc *OIDCController) GetProviders(c *gin.Context) {
	providers, err := oc.oidcService.GetActiveProviders()
	if err != nil {
		log.Printf("Failed to get OIDC providers: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get providers"})
		return
	}

	c.JSON(http.StatusOK, models.OIDCProviderListResponse{
		Providers: providers,
	})
}

// GetAuthURL generates an OAuth URL based on client ID
// @Summary Generate OAuth URL
// @Description Generates an OAuth URL for a given client ID by finding the associated tenant domain
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body models.GetAuthURLInput true "Get Auth URL Input"
// @Success 200 {object} models.GetAuthURLResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /uflow/oidc/auth-url [post]
func (oc *OIDCController) GetAuthURL(c *gin.Context) {
	var input models.GetAuthURLInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Input validation
	if input.ClientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "client_id is required"})
		return
	}

	// Find tenant by client_id using the repository method
	tenant, err := oc.tenantRepo.GetTenantByClientID(input.ClientID)
	if err != nil {
		log.Printf("Failed to find tenant for client_id %s: %v", input.ClientID, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Client ID not found or associated with any tenant"})
		return
	}

	if tenant.TenantDomain == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Tenant found but has no domain configured"})
		return
	}

	// Construct the URL
	baseURL := "https://oauth.prod.authsec.ai/oauth2/auth"
	redirectURI := fmt.Sprintf("https://%s/oidc/auth/callback", tenant.TenantDomain)

	// Appending -main-client to the client_id for the OAuth provider as requested
	// Internal DB lookup used the raw UUID, but external provider seems to expect the suffix
	oauthClientID := input.ClientID
	if !strings.HasSuffix(oauthClientID, "-main-client") {
		oauthClientID = oauthClientID + "-main-client"
	}

	params := url.Values{}
	params.Add("client_id", oauthClientID)
	params.Add("response_type", "code")
	params.Add("scope", "openid profile email")
	params.Add("redirect_uri", redirectURI)
	params.Add("state", "test-state-123")                                       // Hardcoded as per example
	params.Add("code_challenge", "bqtF1ini4HEPUdQaIGqw1JVr7JgsO-y8Be1hxZUmedI") // Hardcoded as per example
	params.Add("code_challenge_method", "S256")

	authURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())

	c.JSON(http.StatusOK, models.GetAuthURLResponse{
		AuthURL: authURL,
	})
}

// InitiateRegistration starts OIDC registration flow for new tenant
// @Summary Initiate OIDC registration
// @Description Starts OIDC flow for registering a new tenant via social login
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body models.OIDCInitiateInput true "OIDC initiation request"
// @Success 200 {object} models.OIDCInitiateResponse
// @Failure 400 {object} map[string]string
// @Failure 409 {object} map[string]string "Tenant domain already exists"
// @Router /uflow/oidc/register/initiate [post]
func (oc *OIDCController) InitiateRegistration(c *gin.Context) {
	var input models.OIDCInitiateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize tenant domain (lowercase, no spaces)
	input.TenantDomain = strings.ToLower(strings.TrimSpace(input.TenantDomain))

	// Validate tenant domain format
	if !isValidTenantDomain(input.TenantDomain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant domain format. Use only lowercase letters, numbers, and hyphens."})
		return
	}

	// Check if tenant domain already exists
	existingTenant, err := oc.tenantRepo.GetTenantByDomain(input.TenantDomain)
	if err == nil && existingTenant != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Tenant domain already exists"})
		return
	}

	// Set request host for callback URL
	oc.oidcService.SetRequestHost(c.Request.Host)

	// Initiate OIDC flow with action "register"
	response, err := oc.oidcService.InitiateOIDCFlow(&input, "register", nil)
	if err != nil {
		log.Printf("Failed to initiate OIDC registration: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// InitiateLogin starts OIDC login flow for existing tenant
// @Summary Initiate OIDC login
// @Description Starts OIDC flow for logging into an existing tenant via social login
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body models.OIDCInitiateInput true "OIDC initiation request"
// @Success 200 {object} models.OIDCInitiateResponse
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string "Tenant not found"
// @Router /uflow/oidc/login/initiate [post]
func (oc *OIDCController) InitiateLogin(c *gin.Context) {
	var input models.OIDCInitiateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize tenant domain
	input.TenantDomain = strings.ToLower(strings.TrimSpace(input.TenantDomain))

	// Verify tenant exists
	tenant, err := oc.tenantRepo.GetTenantByDomain(input.TenantDomain)
	if err != nil || tenant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	// Set request host for callback URL
	oc.oidcService.SetRequestHost(c.Request.Host)

	// Initiate OIDC flow with action "login" and tenant ID
	response, err := oc.oidcService.InitiateOIDCFlow(&input, "login", &tenant.TenantID)
	if err != nil {
		log.Printf("Failed to initiate OIDC login: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// Callback handles the OIDC provider callback
// @Summary OIDC callback handler
// @Description Handles the callback from OIDC provider after authentication. This is part of the traditional redirect flow and is being replaced by the ExchangeCode endpoint for SPA flows.
// @Tags OIDC
// @Accept json
// @Produce json
// @Param code query string true "Authorization code"
// @Param state query string true "State token"
// @Success 200 {object} models.OIDCCallbackResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /authsec/uflow/oidc/callback [get]
func (oc *OIDCController) Callback(c *gin.Context) {
	// This endpoint receives OAuth callback and redirects to frontend SPA with code/state
	// The frontend will then call /exchange-code to complete the flow
	code := c.Query("code")
	state := c.Query("state")
	errorParam := c.Query("error")

	// Helper function to get tenant_domain from state token
	// Priority: OriginDomain > TenantDomain
	getTenantDomain := func(stateToken string) string {
		if stateToken != "" {
			oidcState, err := oc.oidcService.GetStateByToken(stateToken)
			if err == nil && oidcState != nil {
				if oidcState.OriginDomain != "" {
					return oidcState.OriginDomain
				}
				if oidcState.TenantDomain != "" {
					return oidcState.TenantDomain
				}
			}
		}
		return ""
	}

	// Check for error from provider
	if errorParam != "" {
		errorDesc := c.Query("error_description")
		log.Printf("OIDC provider error: %s - %s", errorParam, errorDesc)
		data := gin.H{"success": false, "error": errorParam, "description": errorDesc}
		if tenantDomain := getTenantDomain(state); tenantDomain != "" {
			data["tenant_domain"] = tenantDomain
		}
		renderOAuthCallbackHTML(c, data)
		return
	}

	if code == "" || state == "" {
		data := gin.H{"success": false, "error": "Missing code or state parameter"}
		if tenantDomain := getTenantDomain(state); tenantDomain != "" {
			data["tenant_domain"] = tenantDomain
		}
		renderOAuthCallbackHTML(c, data)
		return
	}

	// Retrieve the state from database to get tenant_domain for proper redirect
	oidcState, err := oc.oidcService.GetStateByToken(state)
	data := gin.H{
		"code":  code,
		"state": state,
	}

	// Add tenant_domain from state if available
	// Priority: OriginDomain (custom domain user came from) > TenantDomain (constructed subdomain)
	if err == nil && oidcState != nil {
		if oidcState.OriginDomain != "" {
			data["tenant_domain"] = oidcState.OriginDomain
			log.Printf("DEBUG Callback: Using origin_domain='%s' from state for redirect", oidcState.OriginDomain)
		} else if oidcState.TenantDomain != "" {
			data["tenant_domain"] = oidcState.TenantDomain
			log.Printf("DEBUG Callback: Using tenant_domain='%s' from state for redirect (fallback)", oidcState.TenantDomain)
		} else {
			log.Printf("DEBUG Callback: No domain found in state (origin_domain='%s', tenant_domain='%s'), will use default or Host header",
				oidcState.OriginDomain, oidcState.TenantDomain)
		}
	} else {
		log.Printf("DEBUG Callback: Failed to get state or state is nil, will use default or Host header")
	}

	// Pass the code and state to frontend for SPA flow
	// Frontend will call POST /uflow/oidc/exchange-code with these parameters
	renderOAuthCallbackHTML(c, data)
}

// ExchangeCode handles the code exchange for SPAs
// @Summary Exchange OIDC code for a JWT token
// @Description Receives the authorization code from a Single-Page Application and exchanges it for a session JWT.
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body models.OIDCCallbackInput true "OIDC code and state"
// @Success 200 {object} models.LoginResponse "Successful login response with JWT token"
// @Failure 400 {object} map[string]string "Bad request - invalid input"
// @Failure 401 {object} map[string]string "Unauthorized - invalid code or state"
// @Failure 500 {object} map[string]string "Internal server error"
// @Router /uflow/oidc/exchange-code [post]
func (oc *OIDCController) ExchangeCode(c *gin.Context) {
	var input models.OIDCCallbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if input.Code == "" || input.State == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing code or state parameter"})
		return
	}

	// Process callback using the same service function
	state, userInfo, err := oc.oidcService.HandleCallback(&input)
	if err != nil {
		log.Printf("OIDC code exchange error: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// The rest of the logic is similar to the original callback, but returns JSON instead of HTML.
	// We can reuse the handlers but they need to be adapted to return JSON.
	// For now, let's inline the logic for clarity.

	// Handle based on action (register, login, or discover)
	log.Printf("DEBUG ExchangeCode: Processing action='%s', state.OriginDomain='%s', state.TenantDomain='%s'",
		state.Action, state.OriginDomain, state.TenantDomain)

	switch state.Action {
	// For simplicity, we will focus on the "login" and "discover" which result in a token.
	// Registration is a more complex flow that creates a tenant and might not immediately result in a JWT.
	case "login":
		log.Printf("DEBUG ExchangeCode: Calling handleLoginAndGenerateToken")
		oc.handleLoginAndGenerateToken(c, state, userInfo)
	case "discover":
		log.Printf("DEBUG ExchangeCode: Calling handleDiscoverAndGenerateToken")
		oc.handleDiscoverAndGenerateToken(c, state, userInfo)
	case "register":
		// The registration flow is complex, creates a tenant, and might not return a token immediately.
		// For now, we will return a success message and let the user login separately.
		oc.handleRegistrationCallback(c, state, userInfo) // This renders HTML, needs to be changed.
		// A better approach would be to refactor handleRegistrationCallback to not write a response
		// and then decide here whether to return JSON or HTML.
		// c.JSON(http.StatusOK, gin.H{"message": "Registration successful. Please login."})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid action in state"})
	}
}

// handleLoginAndGenerateToken is a modified version of handleLoginCallback for the SPA flow
func (oc *OIDCController) handleLoginAndGenerateToken(c *gin.Context, state *models.OIDCState, userInfo *models.OIDCUserInfo) {
	if state.TenantID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID missing from state"})
		return
	}

	// Get tenant info first
	_, err := oc.tenantRepo.GetTenantByID(state.TenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant info"})
		return
	}

	// Check if user has OIDC identity in this tenant
	identity, _ := oc.oidcService.GetIdentityByTenantAndProviderUser(*state.TenantID, state.ProviderName, userInfo.Sub)

	var user *models.ExtendedUser

	if identity != nil {
		// User has OIDC identity - get user by ID
		user, err = oc.userRepo.GetUserByID(identity.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		// Update last login
		oc.oidcService.UpdateLastLogin(identity.ID)
	} else {
		// No OIDC identity - check if user exists by email in this tenant
		user, err = oc.userRepo.GetUserByEmailAndTenant(userInfo.Email, *state.TenantID)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in this workspace"})
			return
		}

		// Before linking, check if this OIDC identity is already in use globally.
		existingGlobalIdentity, _ := oc.oidcService.GetIdentityByProviderUser(state.ProviderName, userInfo.Sub)
		if existingGlobalIdentity != nil {
			c.JSON(http.StatusConflict, gin.H{"error": "This social login is already linked to another user account."})
			return
		}

		// User exists by email but no OIDC identity - link the OIDC provider
		profileDataJSON, _ := json.Marshal(map[string]interface{}{"name": userInfo.Name, "picture": userInfo.Picture})
		newIdentity := &models.OIDCUserIdentity{
			TenantID:       *state.TenantID,
			UserID:         user.ID,
			ProviderName:   state.ProviderName,
			ProviderUserID: userInfo.Sub,
			Email:          userInfo.Email,
			ProfileData:    string(profileDataJSON),
		}
		if err := oc.oidcService.CreateIdentity(newIdentity); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link social login."})
			return
		}
	}

	// Now that we have the user, generate a token for them - pass origin domain for correct redirect
	oc.generateAndRespondWithTokenAndOrigin(c, user, state.OriginDomain)
}

// handleDiscoverAndGenerateToken is a modified version of handleDiscoverCallback for the SPA flow
func (oc *OIDCController) handleDiscoverAndGenerateToken(c *gin.Context, state *models.OIDCState, userInfo *models.OIDCUserInfo) {
	existingUser, err := oc.userRepo.GetUserByEmail(userInfo.Email)

	if err == nil && existingUser != nil {
		// User EXISTS by email - auto-login to their tenant
		// Link identity if it doesn't exist
		existingIdentity, _ := oc.oidcService.GetIdentityByProviderUser(state.ProviderName, userInfo.Sub)
		if existingIdentity == nil {
			profileDataJSON, _ := json.Marshal(map[string]interface{}{"name": userInfo.Name, "picture": userInfo.Picture})
			identity := &models.OIDCUserIdentity{
				TenantID:       existingUser.TenantID,
				UserID:         existingUser.ID,
				ProviderName:   state.ProviderName,
				ProviderUserID: userInfo.Sub,
				Email:          userInfo.Email,
				ProfileData:    string(profileDataJSON),
			}
			oc.oidcService.CreateIdentity(identity)
		} else {
			oc.oidcService.UpdateLastLogin(existingIdentity.ID)
		}

		// Generate a token and respond - pass origin domain for correct redirect
		oc.generateAndRespondWithTokenAndOrigin(c, existingUser, state.OriginDomain)
		return
	}

	// User DOES NOT EXIST by email
	// Check if this is from app.authsec.dev (empty tenant_domain) or custom domain
	if state.TenantDomain == "" {
		// From app.authsec.dev or custom domain - allow registration with needs_domain
		c.JSON(http.StatusNotFound, gin.H{
			"error":         "User not found",
			"needs_domain":  true,
			"message":       "No existing account found. Please choose a workspace name to create your account.",
			"origin_domain": state.OriginDomain, // Pass origin for redirect after registration
			"provider_data": map[string]interface{}{
				"provider":         state.ProviderName,
				"email":            userInfo.Email,
				"name":             userInfo.Name,
				"picture":          userInfo.Picture,
				"provider_user_id": userInfo.Sub,
			},
		})
	} else {
		// From custom domain (e.g., ritam.app.authsec.com) - restrict registration
		c.JSON(http.StatusForbidden, gin.H{
			"error":   "User not found",
			"message": "No account found for this email in this workspace. Please contact your administrator.",
		})
	}
}

func (oc *OIDCController) generateAndRespondWithToken(c *gin.Context, user *models.ExtendedUser) {
	oc.generateAndRespondWithTokenAndOrigin(c, user, "")
}

func (oc *OIDCController) generateAndRespondWithTokenAndOrigin(c *gin.Context, user *models.ExtendedUser, originDomain string) {
	// user.TenantDomain is the authoritative workspace domain (e.g., papa.dev.authsec.dev).
	// originDomain is just where the request came from (e.g., dev.authsec.dev for generic login).
	// Always return the DB value so the frontend knows which workspace to redirect to.
	tenantDomain := user.TenantDomain
	log.Printf("DEBUG generateAndRespondWithTokenAndOrigin: tenantDomain='%s' (from DB), originDomain='%s'",
		tenantDomain, originDomain)

	// Look up the AdminUser to generate a properly-scoped JWT
	adminUser, err := oc.adminUserRepo.GetAdminUserByEmail(user.Email)
	if err != nil {
		log.Printf("ERROR generateAndRespondWithTokenAndOrigin: failed to look up admin user by email %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load user account"})
		return
	}

	tokenStr, err := oc.generateAdminJWTToken(adminUser)
	if err != nil {
		log.Printf("ERROR generateAndRespondWithTokenAndOrigin: failed to generate JWT for user %s: %v", user.Email, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session token"})
		return
	}

	c.JSON(http.StatusOK, models.LoginResponse{
		TenantID:     user.TenantID.String(),
		TenantDomain: tenantDomain,
		Email:        user.Email,
		FirstLogin:   user.LastLogin == nil,
		Token:        tokenStr,
	})
}

// handleRegistrationCallback processes registration after OIDC callback
func (oc *OIDCController) handleRegistrationCallback(c *gin.Context, state *models.OIDCState, userInfo *models.OIDCUserInfo) {
	// Check if user already exists with this OIDC identity
	existingIdentity, err := oc.oidcService.GetIdentityByProviderUser(state.ProviderName, userInfo.Sub)
	if err == nil && existingIdentity != nil {
		// User already registered with this provider - redirect to their tenant
		tenant, _ := oc.tenantRepo.GetTenantByID(existingIdentity.TenantID.String())
		if tenant != nil {
			c.JSON(http.StatusConflict, gin.H{
				"error":         "Account already exists",
				"message":       "You already have an account. Please login instead.",
				"tenant_domain": tenant.TenantDomain,
			})
			return
		}
	}

	// Create new tenant and user
	// Note: In admin registration pattern, tenant_id = client_id for the default client
	// tenantID is used for tenant.TenantID (business key)
	// tenant.ID (primary key) is auto-generated and used for FK references
	tenantID := uuid.New()
	projectID := uuid.New()
	clientID := tenantID // Client ID = Tenant ID for default client (matches admin registration)
	userID := uuid.New()

	// Start transaction
	db := config.GetDatabase()
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}
	defer tx.Rollback()

	// Create tenant
	fullDomain := fmt.Sprintf("%s.%s", state.TenantDomain, config.AppConfig.TenantDomainSuffix)
	tenantDBName := fmt.Sprintf("tenant_%s", strings.ReplaceAll(tenantID.String(), "-", "_"))
	username := userInfo.Email
	providerID := userInfo.Sub
	tenant := &models.Tenant{
		ID:           tenantID, // Use same ID for both id and tenant_id for simplicity
		TenantID:     tenantID,
		TenantDB:     tenantDBName,
		Email:        userInfo.Email,
		Username:     &username,
		Name:         userInfo.Name,
		TenantDomain: fullDomain,
		Provider:     state.ProviderName,
		ProviderID:   &providerID,
		Status:       "active",
		Source:       "oidc",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := oc.tenantRepo.CreateTenantTx(tx, tenant); err != nil {
		log.Printf("Failed to create tenant: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant"})
		return
	}

	// Create project - use tenant.ID for FK reference (projects.tenant_id -> tenants.id)
	if err := oc.tenantRepo.CreateProjectTx(tx, projectID, tenant.ID, userID, "Default Project"); err != nil {
		log.Printf("Failed to create project: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Create admin user in main DB (users table)
	usernameStr := userInfo.Email
	adminUser := &models.ExtendedUser{
		User: sharedmodels.User{
			ID:           userID,
			Email:        userInfo.Email,
			Name:         userInfo.Name,
			PasswordHash: "", // No password for OIDC users
			ClientID:     clientID,
			TenantID:     tenantID,
			ProjectID:    projectID,
			TenantDomain: fullDomain,
			Provider:     state.ProviderName,
			ProviderID:   userInfo.Sub,
			Username:     &usernameStr,
			ProviderData: datatypes.JSON("{}"),
			Active:       true,
		},
	}

	// Store avatar URL if available
	if userInfo.Picture != "" {
		adminUser.AvatarURL = &userInfo.Picture
	}

	if err := oc.userRepo.CreateUserTx(tx, adminUser); err != nil {
		log.Printf("Failed to create admin user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Use EnsureAdminRoleAndPermissionsTx to seed both role AND permissions (fix for OIDC registration bug)
	roleID, err := database.NewAdminSeedRepository(config.GetDatabase()).EnsureAdminRoleAndPermissionsTx(tx, tenantID)
	if err != nil {
		log.Printf("WARNING: Failed to ensure admin role and permissions for tenant %s: %v", tenantID, err)
	} else {
		// Insert into role_bindings (user_roles is deprecated)
		// scope_type and scope_id are NULL for tenant-wide role assignments
		if _, err := tx.Exec(`
			INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
			SELECT gen_random_uuid(), $1, $2, $3, NULL, NULL, NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM role_bindings
				WHERE tenant_id = $1 AND user_id = $2 AND role_id = $3 AND scope_type IS NULL AND scope_id IS NULL
			)
		`, tenantID, userID, roleID); err != nil {
			log.Printf("WARNING: Failed to assign admin role to OIDC user %s: %v", userID, err)
			// Non-fatal - user can still login via OIDC, just not via admin password login
		} else {
			log.Printf("INFO: Admin role assigned to OIDC user %s", userID)
		}
	}

	// Create default role bindings in MAIN DB for admin across core services
	var adminRoleID uuid.UUID
	if err := tx.QueryRow("SELECT id FROM roles WHERE LOWER(name) = 'admin' AND tenant_id = $1 LIMIT 1", tenantID).Scan(&adminRoleID); err != nil {
		log.Printf("Failed to resolve admin role id for default bindings: %v", err)
	} else {
		services := []string{"external-service", "clients", "user-flow", "ooc-manager", "log-service", "hydra-service", "sdk-manager"}
		usernameVal := userInfo.Email
		for _, svc := range services {
			if _, err := tx.Exec(`
				INSERT INTO role_bindings (id, tenant_id, user_id, role_id, role_name, username, scope_type, scope_id, created_at, updated_at)
				SELECT $1, $2, $3, $4, 'admin', $5, $6, $7, NOW(), NOW()
				WHERE NOT EXISTS (
					SELECT 1 FROM role_bindings
					WHERE tenant_id = $2 AND user_id = $3 AND role_id = $4 AND scope_type = $6 AND scope_id = $7
				)
			`, uuid.New(), tenantID, userID, adminRoleID, usernameVal, svc, tenantID); err != nil {
				log.Printf("WARNING: Failed to create role binding for service=%s tenant=%s: %v", svc, tenantID, err)
				// Non-fatal - continue with other bindings
			}
		}
		// Add a wildcard binding to grant full access for the admin user
		if _, err := tx.Exec(`
			INSERT INTO role_bindings (id, tenant_id, user_id, role_id, role_name, username, scope_type, scope_id, created_at, updated_at)
			SELECT $1, $2, $3, $4, 'admin', $5, '*', NULL, NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM role_bindings
				WHERE tenant_id = $2 AND user_id = $3 AND role_id = $4 AND scope_type = '*' AND scope_id IS NULL
			)
		`, uuid.New(), tenantID, userID, adminRoleID, usernameVal); err != nil {
			log.Printf("WARNING: Failed to create wildcard role binding tenant=%s: %v", tenantID, err)
		} else {
			log.Printf("INFO: Created role bindings for OIDC user %s across all services", userID)
		}
	}

	// Commit main DB transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	// Create tenant database
	dbName, err := oc.tenantDBService.CreateTenantDatabase(tenantID.String())
	if err != nil {
		log.Printf("Failed to create tenant database: %v", err)
		// Note: Main DB records created, tenant DB failed - may need cleanup
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant database"})
		return
	}
	log.Printf("Created tenant database: %s", dbName)

	// Update tenant record with database name
	mainDB := config.GetDatabase()
	if _, err := mainDB.Exec("UPDATE tenants SET tenant_db = $1, updated_at = NOW() WHERE tenant_id = $2", dbName, tenantID); err != nil {
		log.Printf("Warning: Failed to update tenant_db field: %v", err)
		// Non-fatal - continue with registration
	} else {
		log.Printf("Successfully updated tenant record with database name: %s", dbName)
	}

	// Provision PKI infrastructure via ICP service
	if oc.icpProvisioningService != nil {
		log.Printf("Provisioning PKI for tenant: %s", tenantID.String())

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		icpResp, err := oc.icpProvisioningService.ProvisionPKI(ctx, &icp.ProvisionPKIRequest{
			TenantID:   tenantID.String(),
			CommonName: fmt.Sprintf("%s Root CA", userInfo.Name),
			Domain:     fullDomain,
			TTL:        "87600h", // 10 years
			MaxTTL:     "24h",    // Max certificate TTL
		})
		if err != nil {
			log.Printf("Warning: PKI provisioning failed: %v", err)
			// Update tenant status to indicate PKI provisioning failure
			if _, updateErr := mainDB.Exec("UPDATE tenants SET status = 'pki_provisioning_failed' WHERE tenant_id = $1", tenantID); updateErr != nil {
				log.Printf("Failed to update tenant status: %v", updateErr)
			}
			// Continue - admin can retry PKI provisioning later
		} else {
			log.Printf("Successfully provisioned PKI - Mount: %s", icpResp.PKIMount)
			// Update tenant with PKI information (vault_mount and ca_cert only)
			if _, err := mainDB.Exec("UPDATE tenants SET vault_mount = $1, ca_cert = $2 WHERE tenant_id = $3", icpResp.PKIMount, icpResp.CACert, tenantID); err != nil {
				log.Printf("Warning: Failed to update tenant with PKI info: %v", err)
			}
		}
	} else {
		log.Printf("INFO: ICP provisioning service not configured, skipping PKI setup for tenant %s", tenantID.String())
	}

	// Connect to tenant database for additional setup
	tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, tenantDBName, config.AppConfig.DBPort)

	tenantDB, err := sql.Open("postgres", tenantDSN)
	if err != nil {
		log.Printf("Failed to connect to tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	defer tenantDB.Close()

	// Create default client in tenant DB with Hydra client ID
	hydraClientID := fmt.Sprintf("%s-main-client", clientID.String())
	clientInsert := `INSERT INTO clients (id, client_id, tenant_id, project_id, owner_id, org_id, name, description, hydra_client_id, active, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, $2, $5, $6, $7, true, NOW(), NOW())`
	if _, err := tenantDB.Exec(clientInsert, clientID, tenantID, projectID, tenantID, "Default Client", "Default client for OIDC user", hydraClientID); err != nil {
		log.Printf("Failed to create default client in tenant DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default client"})
		return
	}
	log.Printf("Created default client in tenant DB: %s", clientID)

	// Upsert tenant record in tenant database (migration may have seeded a minimal stub row)
	tenantInsert := `INSERT INTO tenants (id, tenant_id, email, password_hash, name, provider, source, status, tenant_domain, tenant_db, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, $5, 'oidc_registration', 'active', $6, $7, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email, name = EXCLUDED.name, provider = EXCLUDED.provider,
			source = EXCLUDED.source, status = EXCLUDED.status,
			tenant_domain = EXCLUDED.tenant_domain, tenant_db = EXCLUDED.tenant_db,
			updated_at = NOW()`
	if _, err := tenantDB.Exec(tenantInsert, tenantID, userInfo.Email, "", userInfo.Name, state.ProviderName, fullDomain, tenantDBName); err != nil {
		log.Printf("Failed to upsert tenant record in tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant record in tenant database"})
		return
	}
	log.Printf("Created tenant record in tenant DB for tenant: %s", tenantID)

	// Create default project in tenant database (project was already created in global database)
	projectInsert := `INSERT INTO projects (id, tenant_id, name, description, user_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())`
	if _, err := tenantDB.Exec(projectInsert, projectID, tenantID, "Default Project", "Default project for OIDC user", tenantID); err != nil {
		log.Printf("Failed to create default project in tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default project in tenant database"})
		return
	}
	log.Printf("Created default project in tenant DB: %s", projectID)

	// Create tenant_mappings entry in global database for client_id to tenant_id mapping
	globalDB := config.GetDatabase()
	tenantMappingInsert := `INSERT INTO tenant_mappings (tenant_id, client_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (client_id) DO NOTHING`
	if _, err := globalDB.Exec(tenantMappingInsert, tenantID, clientID); err != nil {
		log.Printf("Failed to create tenant mapping: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant mapping"})
		return
	}
	log.Printf("Created tenant_mappings entry: tenant_id=%s, client_id=%s", tenantID.String(), clientID.String())

	// Assign admin role to the created user in the tenant database
	if err := oc.assignAdminRoleToUser(tenantDB, userID, tenantID); err != nil {
		log.Printf("Warning: Failed to assign admin role to user %s in tenant DB: %v", userInfo.Email, err)
		// Non-fatal - continue with registration
	} else {
		log.Printf("Successfully assigned admin role to user in tenant DB: %s", userInfo.Email)
	}

	// Create OIDC identity link
	profileDataJSON, _ := json.Marshal(map[string]interface{}{
		"name":    userInfo.Name,
		"picture": userInfo.Picture,
	})

	identity := &models.OIDCUserIdentity{
		TenantID:       tenantID,
		UserID:         userID,
		ProviderName:   state.ProviderName,
		ProviderUserID: userInfo.Sub,
		Email:          userInfo.Email,
		ProfileData:    string(profileDataJSON),
	}

	if err := oc.oidcService.CreateIdentity(identity); err != nil {
		log.Printf("Failed to create OIDC identity: %v", err)
		// Non-fatal, user can still login
	}

	// Create user in tenant database
	if err := oc.createUserInTenantDB(tenantID, userID, clientID, fullDomain, state.ProviderName, userInfo); err != nil {
		log.Printf("Failed to create user in tenant DB: %v", err)
		// Non-fatal for registration response
	}

	// Save secret to Vault and register with Hydra
	secretID, err := config.SaveSecretToVault(tenantID.String(), projectID.String(), tenantID.String())
	if err != nil {
		log.Printf("Warning: Failed to save secret to vault: %v", err)
		log.Printf("OIDC registration will continue without Vault secret storage for tenant: %s", tenantID.String())
		// Don't block OIDC registration - they can still use the system without Vault integration
		secretID = "" // Clear secretID so we don't attempt Hydra registration
	}

	// Register client with Hydra only when we have a secret to use
	if secretID != "" {
		if err := services.RegisterClientWithHydra(clientID.String(), secretID, userInfo.Email, tenantID.String(), fullDomain); err != nil {
			log.Printf("Warning: Failed to register client with Hydra: %v", err)
			log.Printf("OIDC registration will continue without Hydra client registration for tenant: %s", tenantID.String())
			// Don't block OIDC registration - they can still use the system without OAuth integration
		}
	} else {
		log.Printf("Skipping Hydra registration for tenant %s because no Vault secret was stored", tenantID.String())
	}

	// Audit log: OIDC registration completed
	middlewares.Audit(c, "oidc", tenantID.String(), "register", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":     tenantID.String(),
			"tenant_domain": fullDomain,
			"user_id":       userID.String(),
			"email":         userInfo.Email,
			"provider":      state.ProviderName,
		},
	})

	// Use origin domain for redirect (where user came from) instead of fullDomain
	// The origin domain is stored in state and persists across the OAuth redirect
	redirectDomain := state.OriginDomain
	if redirectDomain == "" {
		redirectDomain = fullDomain // Fallback to constructed domain
	}
	log.Printf("DEBUG handleRegistrationCallback: Using redirectDomain='%s' (state.OriginDomain='%s', fullDomain='%s')", redirectDomain, state.OriginDomain, fullDomain)

	// Return HTML page that communicates with frontend
	renderOAuthCallbackHTML(c, map[string]interface{}{
		"success":       true,
		"message":       "Registration successful",
		"tenant_domain": redirectDomain,
		"tenant_id":     tenantID.String(),
		"client_id":     clientID.String(),
		"first_login":   true,
	})
}

// handleLoginCallback processes login after OIDC callback
// Login from tenant subdomain (e.g., ritam.app.authsec.dev) - user must exist in this tenant
func (oc *OIDCController) handleLoginCallback(c *gin.Context, state *models.OIDCState, userInfo *models.OIDCUserInfo) {
	log.Printf("DEBUG handleLoginCallback: state.TenantDomain='%s', state.TenantID=%v", state.TenantDomain, state.TenantID)

	if state.TenantID == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Tenant ID missing from state"})
		return
	}

	// Get tenant info first
	tenant, err := oc.tenantRepo.GetTenantByID(state.TenantID.String())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant info"})
		return
	}

	log.Printf("DEBUG handleLoginCallback: Found tenant with tenant_domain='%s' from database", tenant.TenantDomain)

	// Check if user has OIDC identity in this tenant
	identity, _ := oc.oidcService.GetIdentityByTenantAndProviderUser(*state.TenantID, state.ProviderName, userInfo.Sub)

	var user *models.ExtendedUser

	if identity != nil {
		// User has OIDC identity - get user by ID
		user, err = oc.userRepo.GetUserByID(identity.UserID)
		if err != nil {
			log.Printf("Failed to get user by identity: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user info"})
			return
		}
		// Update last login
		if err := oc.oidcService.UpdateLastLogin(identity.ID); err != nil {
			log.Printf("Failed to update last login: %v", err)
		}
	} else {
		// No OIDC identity in this tenant. Check if user exists by email in this tenant.
		user, err = oc.userRepo.GetUserByEmailAndTenant(userInfo.Email, *state.TenantID)
		if err != nil {
			// User doesn't exist in this tenant at all - REJECT
			log.Printf("OIDC Login: User %s not found in tenant %s - rejecting", userInfo.Email, state.TenantDomain)
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "User not found",
				"message": "No account found with this email in this workspace. Please contact your administrator.",
			})
			return
		}

		// Before linking, check if this OIDC identity is already in use globally.
		existingGlobalIdentity, _ := oc.oidcService.GetIdentityByProviderUser(state.ProviderName, userInfo.Sub)
		if existingGlobalIdentity != nil {
			log.Printf("OIDC Login Conflict: User %s in tenant %s tried to link an OIDC identity that is already linked to user %s in tenant %s.",
				user.ID, *state.TenantID, existingGlobalIdentity.UserID, existingGlobalIdentity.TenantID)
			c.JSON(http.StatusConflict, gin.H{
				"error":   "Social login already linked",
				"message": "This social login is already linked to another user account. Please use a different login method or contact support.",
			})
			return
		}

		// User exists by email but no OIDC identity - link the OIDC provider
		log.Printf("OIDC Login: Linking provider %s to existing user %s in tenant %s", state.ProviderName, userInfo.Email, *state.TenantID)
		profileDataJSON, _ := json.Marshal(map[string]interface{}{
			"name":    userInfo.Name,
			"picture": userInfo.Picture,
		})
		newIdentity := &models.OIDCUserIdentity{
			TenantID:       *state.TenantID,
			UserID:         user.ID,
			ProviderName:   state.ProviderName,
			ProviderUserID: userInfo.Sub,
			Email:          userInfo.Email,
			ProfileData:    string(profileDataJSON),
		}
		if err := oc.oidcService.CreateIdentity(newIdentity); err != nil {
			log.Printf("Failed to create OIDC identity during login link: %v", err)
			// Now that we have upsert, this failure is more serious.
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to link social login."})
			return
		}
	}

	// Check if first login (before last_login gets updated in the response or elsewhere)
	isFirstLogin := user.LastLogin == nil

	// Determine login domain - prioritize origin domain (where user came from) over tenant domain
	// This is important for custom domains (e.g., test.auth-sec.org vs test.app.authsec.dev)
	loginDomain := state.OriginDomain
	if loginDomain == "" {
		loginDomain = state.TenantDomain
	}
	if loginDomain == "" {
		loginDomain = tenant.TenantDomain
	}
	log.Printf("DEBUG handleLoginCallback: loginDomain='%s' (origin='%s', state.TenantDomain='%s', tenant.TenantDomain='%s')",
		loginDomain, state.OriginDomain, state.TenantDomain, tenant.TenantDomain)

	// Audit log: OIDC login successful
	middlewares.Audit(c, "oidc", user.ID.String(), "login", &middlewares.AuditChanges{
		After: map[string]interface{}{
			"tenant_id":     state.TenantID.String(),
			"tenant_domain": loginDomain,
			"user_id":       user.ID.String(),
			"email":         user.Email,
			"provider":      state.ProviderName,
			"first_login":   isFirstLogin,
		},
	})

	// Return HTML page that communicates with frontend
	log.Printf("DEBUG handleLoginCallback: Calling renderOAuthCallbackHTML with tenant_domain='%s'", loginDomain)
	renderOAuthCallbackHTML(c, map[string]interface{}{
		"success":       true,
		"message":       "Login successful",
		"tenant_domain": loginDomain,
		"tenant_id":     state.TenantID.String(),
		"client_id":     user.ClientID.String(),
		"first_login":   isFirstLogin,
	})
}

// handleDiscoverCallback handles the OIDC callback for discover mode
// This is used when user comes from app.authsec.dev without specifying a tenant
// Flow: Check if user email exists in main DB → if yes, auto-login; if no, prompt for domain to register
func (oc *OIDCController) handleDiscoverCallback(c *gin.Context, state *models.OIDCState, userInfo *models.OIDCUserInfo) {
	// First, check if user with this email already exists in main DB (users table)
	existingUser, err := oc.userRepo.GetUserByEmail(userInfo.Email)

	if err == nil && existingUser != nil {
		// User EXISTS by email - auto-login to their tenant
		log.Printf("OIDC Discover: Found existing user by email %s in tenant %s", userInfo.Email, existingUser.TenantID)

		// Get tenant info
		tenant, err := oc.tenantRepo.GetTenantByID(existingUser.TenantID.String())
		if err != nil {
			log.Printf("Failed to get tenant info: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant info"})
			return
		}

		// Check if OIDC identity exists globally (constraint is on provider_name + provider_user_id)
		existingIdentity, _ := oc.oidcService.GetIdentityByProviderUser(state.ProviderName, userInfo.Sub)
		if existingIdentity != nil {
			// Identity exists - just update last login
			if err := oc.oidcService.UpdateLastLogin(existingIdentity.ID); err != nil {
				log.Printf("Failed to update last login: %v", err)
			}
			log.Printf("Updated last login for existing OIDC identity: %s", userInfo.Email)
		} else {
			// Identity doesn't exist - create new one
			profileDataJSON, _ := json.Marshal(map[string]interface{}{
				"name":    userInfo.Name,
				"picture": userInfo.Picture,
			})
			identity := &models.OIDCUserIdentity{
				TenantID:       existingUser.TenantID,
				UserID:         existingUser.ID,
				ProviderName:   state.ProviderName,
				ProviderUserID: userInfo.Sub,
				Email:          userInfo.Email,
				ProfileData:    string(profileDataJSON),
			}
			if err := oc.oidcService.CreateIdentity(identity); err != nil {
				log.Printf("Failed to create OIDC identity for existing user: %v", err)
				// Non-fatal, continue with login
			} else {
				log.Printf("Created OIDC identity link for existing user %s", userInfo.Email)
			}
		}

		// Check if first login (before last_login gets updated)
		isFirstLogin := existingUser.LastLogin == nil

		// Determine redirect domain - prioritize origin domain over tenant domain
		redirectDomain := state.OriginDomain
		if redirectDomain == "" {
			redirectDomain = tenant.TenantDomain
		}
		log.Printf("DEBUG handleDiscoverCallback: redirectDomain='%s' (origin='%s', tenant='%s')",
			redirectDomain, state.OriginDomain, tenant.TenantDomain)

		// Return HTML page that communicates with frontend
		renderOAuthCallbackHTML(c, map[string]interface{}{
			"success":       true,
			"message":       "Login successful - redirecting to your workspace",
			"tenant_domain": redirectDomain,
			"tenant_id":     existingUser.TenantID.String(),
			"client_id":     existingUser.ClientID.String(),
			"first_login":   isFirstLogin,
		})
		return
	}

	// User DOES NOT EXIST by email - need to register with a new tenant domain
	// Return special response asking frontend to prompt for tenant domain
	log.Printf("OIDC Discover: No existing user found for email %s, prompting for tenant domain", userInfo.Email)
	log.Printf("DEBUG handleDiscoverCallback (new user): OriginDomain='%s' for redirect after registration", state.OriginDomain)

	// Return HTML page that communicates with frontend
	// Pass origin_domain so frontend knows where to redirect back after registration
	renderOAuthCallbackHTML(c, map[string]interface{}{
		"success":          false,
		"needs_domain":     true,
		"message":          "No existing account found. Please choose a workspace name to create your account.",
		"provider":         state.ProviderName,
		"email":            userInfo.Email,
		"name":             userInfo.Name,
		"picture":          userInfo.Picture,
		"provider_user_id": userInfo.Sub,
		"tenant_id":        nil,
		"client_id":        nil,
		"origin_domain":    state.OriginDomain, // Pass origin for redirect after registration
		// Frontend should call /uflow/oidc/complete-registration with tenant_domain
	})
}

// CompleteRegistration completes registration after discover mode found no existing user
// @Summary Complete OIDC registration after discover
// @Description Completes registration for a new user after discover mode, with chosen tenant domain
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body OIDCCompleteRegistrationInput true "Registration completion input"
// @Success 200 {object} models.OIDCCallbackResponse
// @Failure 400 {object} map[string]string
// @Router /uflow/oidc/complete-registration [post]
func (oc *OIDCController) CompleteRegistration(c *gin.Context) {
	var input struct {
		TenantDomain   string `json:"tenant_domain" binding:"required"`
		Provider       string `json:"provider" binding:"required"`
		Email          string `json:"email" binding:"required"`
		Name           string `json:"name"`
		Picture        string `json:"picture"`
		ProviderUserID string `json:"provider_user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Normalize and validate tenant domain
	tenantDomain := strings.ToLower(strings.TrimSpace(input.TenantDomain))
	if !isValidTenantDomain(tenantDomain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant domain format. Use only lowercase letters, numbers, and hyphens."})
		return
	}

	// Check if tenant domain already exists
	existingTenant, err := oc.tenantRepo.GetTenantByDomain(tenantDomain)
	if err == nil && existingTenant != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Tenant domain already exists. Please choose a different name."})
		return
	}

	// Check if this OIDC identity is already registered (double check)
	existingIdentity, err := oc.oidcService.GetIdentityByProviderUser(input.Provider, input.ProviderUserID)
	if err == nil && existingIdentity != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "This social login is already registered with another account."})
		return
	}

	// Create new tenant and user (similar to handleRegistrationCallback)
	// Note: In admin registration pattern, tenant_id = client_id for the default client
	tenantID := uuid.New()
	projectID := uuid.New()
	clientID := tenantID // Client ID = Tenant ID for default client (matches admin registration)
	userID := uuid.New()

	// Start transaction
	db := config.GetDatabase()
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to start transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}
	defer tx.Rollback()

	// Create tenant
	fullDomain := fmt.Sprintf("%s.%s", tenantDomain, config.AppConfig.TenantDomainSuffix)
	tenantDBName := fmt.Sprintf("tenant_%s", strings.ReplaceAll(tenantID.String(), "-", "_"))
	username := input.Email
	providerIDPtr := input.ProviderUserID
	tenant := &models.Tenant{
		ID:           tenantID, // Use same ID for both id and tenant_id for simplicity
		TenantID:     tenantID,
		TenantDB:     tenantDBName,
		Email:        input.Email,
		Username:     &username,
		Name:         input.Name,
		TenantDomain: fullDomain,
		Provider:     input.Provider,
		ProviderID:   &providerIDPtr,
		Status:       "active",
		Source:       "oidc",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := oc.tenantRepo.CreateTenantTx(tx, tenant); err != nil {
		log.Printf("Failed to create tenant: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant"})
		return
	}

	// Create project - use tenant.ID for FK reference (projects.tenant_id -> tenants.id)
	if err := oc.tenantRepo.CreateProjectTx(tx, projectID, tenant.ID, userID, "Default Project"); err != nil {
		log.Printf("Failed to create project: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create project"})
		return
	}

	// Create user in main DB (users table)
	usernameStr := input.Email
	adminUser := &models.ExtendedUser{
		User: sharedmodels.User{
			ID:           userID,
			Email:        input.Email,
			Name:         input.Name,
			PasswordHash: "", // No password for OIDC users
			ClientID:     clientID,
			TenantID:     tenantID,
			ProjectID:    projectID,
			TenantDomain: fullDomain,
			Provider:     input.Provider,
			ProviderID:   input.ProviderUserID,
			Username:     &usernameStr,
			ProviderData: datatypes.JSON("{}"),
			Active:       true,
		},
	}

	// Store avatar URL if available
	if input.Picture != "" {
		adminUser.AvatarURL = &input.Picture
	}

	if err := oc.userRepo.CreateUserTx(tx, adminUser); err != nil {
		log.Printf("Failed to create user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	// Use EnsureAdminRoleAndPermissionsTx to seed both role AND permissions (fix for OIDC registration bug)
	roleID, err := database.NewAdminSeedRepository(config.GetDatabase()).EnsureAdminRoleAndPermissionsTx(tx, tenantID)
	if err != nil {
		log.Printf("WARNING: Failed to ensure admin role and permissions for tenant %s: %v", tenantID, err)
	} else {
		// Insert into role_bindings (user_roles is deprecated)
		// scope_type and scope_id are NULL for tenant-wide role assignments
		if _, err := tx.Exec(`
			INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
			SELECT gen_random_uuid(), $1, $2, $3, NULL, NULL, NOW(), NOW()
			WHERE NOT EXISTS (
				SELECT 1 FROM role_bindings
				WHERE tenant_id = $1 AND user_id = $2 AND role_id = $3 AND scope_type IS NULL AND scope_id IS NULL
			)
		`, tenantID, userID, roleID); err != nil {
			log.Printf("WARNING: Failed to assign admin role to OIDC user %s: %v", userID, err)
		} else {
			log.Printf("INFO: Admin role assigned to OIDC user %s", userID)
		}
	}

	// Commit main DB transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	// Create tenant database
	dbName, err := oc.tenantDBService.CreateTenantDatabase(tenantID.String())
	if err != nil {
		log.Printf("Failed to create tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant database"})
		return
	}
	log.Printf("Created tenant database: %s", dbName)

	// Provision PKI infrastructure via ICP service
	mainDB := config.GetDatabase()
	if oc.icpProvisioningService != nil {
		log.Printf("Provisioning PKI for tenant: %s", tenantID.String())

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		icpResp, err := oc.icpProvisioningService.ProvisionPKI(ctx, &icp.ProvisionPKIRequest{
			TenantID:   tenantID.String(),
			CommonName: fmt.Sprintf("%s Root CA", input.Name),
			Domain:     fullDomain,
			TTL:        "87600h", // 10 years
			MaxTTL:     "24h",    // Max certificate TTL
		})

		if err != nil {
			log.Printf("Warning: PKI provisioning failed: %v", err)
			// Update tenant status to indicate PKI provisioning failure
			if _, updateErr := mainDB.Exec("UPDATE tenants SET status = 'pki_provisioning_failed' WHERE tenant_id = $1", tenantID); updateErr != nil {
				log.Printf("Failed to update tenant status: %v", updateErr)
			}
			// Continue - admin can retry PKI provisioning later
		} else {
			log.Printf("Successfully provisioned PKI - Mount: %s", icpResp.PKIMount)
			// Update tenant with PKI information (vault_mount and ca_cert only)
			if _, err := mainDB.Exec("UPDATE tenants SET vault_mount = $1, ca_cert = $2 WHERE tenant_id = $3", icpResp.PKIMount, icpResp.CACert, tenantID); err != nil {
				log.Printf("Warning: Failed to update tenant with PKI info: %v", err)
			}
		}
	} else {
		log.Printf("INFO: ICP provisioning service not configured, skipping PKI setup for tenant %s", tenantID.String())
	}

	// Connect to tenant database for additional setup
	tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, tenantDBName, config.AppConfig.DBPort)

	tenantDB, err := sql.Open("postgres", tenantDSN)
	if err != nil {
		log.Printf("Failed to connect to tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database"})
		return
	}
	defer tenantDB.Close()

	// Create default client in tenant DB with Hydra client ID
	hydraClientID := fmt.Sprintf("%s-main-client", clientID.String())
	clientInsert := `INSERT INTO clients (id, client_id, tenant_id, project_id, owner_id, org_id, name, description, hydra_client_id, active, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, $2, $5, $6, $7, true, NOW(), NOW())`
	if _, err := tenantDB.Exec(clientInsert, clientID, tenantID, projectID, tenantID, "Default Client", "Default client for OIDC user", hydraClientID); err != nil {
		log.Printf("Failed to create default client in tenant DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default client"})
		return
	}
	log.Printf("Created default client in tenant DB: %s", clientID)

	// Upsert tenant record in tenant database (migration may have seeded a minimal stub row)
	tenantInsert := `INSERT INTO tenants (id, tenant_id, email, password_hash, name, provider, source, status, tenant_domain, tenant_db, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, $5, 'oidc_registration', 'active', $6, $7, NOW(), NOW())
		ON CONFLICT (id) DO UPDATE SET
			email = EXCLUDED.email, name = EXCLUDED.name, provider = EXCLUDED.provider,
			source = EXCLUDED.source, status = EXCLUDED.status,
			tenant_domain = EXCLUDED.tenant_domain, tenant_db = EXCLUDED.tenant_db,
			updated_at = NOW()`
	if _, err := tenantDB.Exec(tenantInsert, tenantID, input.Email, "", input.Name, input.Provider, fullDomain, tenantDBName); err != nil {
		log.Printf("Failed to upsert tenant record in tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant record in tenant database"})
		return
	}
	log.Printf("Created tenant record in tenant DB for tenant: %s", tenantID)

	// Create default project in tenant database (project was already created in global database)
	projectInsert := `INSERT INTO projects (id, tenant_id, name, description, user_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, true, NOW(), NOW())`
	if _, err := tenantDB.Exec(projectInsert, projectID, tenantID, "Default Project", "Default project for OIDC user", tenantID); err != nil {
		log.Printf("Failed to create default project in tenant database: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create default project in tenant database"})
		return
	}
	log.Printf("Created default project in tenant DB: %s", projectID)

	// Create tenant_mappings entry in global database for client_id to tenant_id mapping
	globalDB := config.GetDatabase()
	tenantMappingInsert := `INSERT INTO tenant_mappings (tenant_id, client_id, created_at, updated_at)
		VALUES ($1, $2, NOW(), NOW())
		ON CONFLICT (client_id) DO NOTHING`
	if _, err := globalDB.Exec(tenantMappingInsert, tenantID, clientID); err != nil {
		log.Printf("Failed to create tenant mapping: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create tenant mapping"})
		return
	}
	log.Printf("Created tenant_mappings entry: tenant_id=%s, client_id=%s", tenantID.String(), clientID.String())

	// Assign admin role to the created user in the tenant database
	if err := oc.assignAdminRoleToUser(tenantDB, userID, tenantID); err != nil {
		log.Printf("Warning: Failed to assign admin role to user %s in tenant DB: %v", input.Email, err)
		// Non-fatal - continue with registration
	} else {
		log.Printf("Successfully assigned admin role to user in tenant DB: %s", input.Email)
	}

	// Create OIDC identity link
	profileDataJSON, _ := json.Marshal(map[string]interface{}{
		"name":    input.Name,
		"picture": input.Picture,
	})

	identity := &models.OIDCUserIdentity{
		TenantID:       tenantID,
		UserID:         userID,
		ProviderName:   input.Provider,
		ProviderUserID: input.ProviderUserID,
		Email:          input.Email,
		ProfileData:    string(profileDataJSON),
	}

	if err := oc.oidcService.CreateIdentity(identity); err != nil {
		log.Printf("Failed to create OIDC identity: %v", err)
	}

	// Create user in tenant database
	userInfo := &models.OIDCUserInfo{
		Sub:       input.ProviderUserID,
		Email:     input.Email,
		Name:      input.Name,
		GivenName: strings.SplitN(input.Name, " ", 2)[0],
	}
	if len(strings.SplitN(input.Name, " ", 2)) > 1 {
		userInfo.FamilyName = strings.SplitN(input.Name, " ", 2)[1]
	}
	if err := oc.createUserInTenantDB(tenantID, userID, clientID, fullDomain, input.Provider, userInfo); err != nil {
		log.Printf("Failed to create user in tenant DB: %v", err)
	}

	// Save secret to Vault and register with Hydra
	secretID, err := config.SaveSecretToVault(tenantID.String(), projectID.String(), tenantID.String())
	if err != nil {
		log.Printf("Warning: Failed to save secret to vault: %v", err)
		log.Printf("OIDC registration will continue without Vault secret storage for tenant: %s", tenantID.String())
		// Don't block OIDC registration - they can still use the system without Vault integration
		secretID = "" // Clear secretID so we don't attempt Hydra registration
	}

	// Register client with Hydra only when we have a secret to use
	if secretID != "" {
		if err := services.RegisterClientWithHydra(clientID.String(), secretID, input.Email, tenantID.String(), fullDomain); err != nil {
			log.Printf("Warning: Failed to register client with Hydra: %v", err)
			log.Printf("OIDC registration will continue without Hydra client registration for tenant: %s", tenantID.String())
			// Don't block OIDC registration - they can still use the system without OAuth integration
		}
	} else {
		log.Printf("Skipping Hydra registration for tenant %s because no Vault secret was stored", tenantID.String())
	}

	// Return JSON response without token (frontend should login separately)
	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"message":       "Registration successful - welcome to your new workspace!",
		"tenant_domain": fullDomain,
		"tenant_id":     tenantID.String(),
		"client_id":     clientID.String(),
		"first_login":   true, // Always true for new registrations
	})
}

// createUserInTenantDB creates the user record in the tenant's database
func (oc *OIDCController) createUserInTenantDB(tenantID, userID, clientID uuid.UUID, tenantDomain, provider string, userInfo *models.OIDCUserInfo) error {
	// Get tenant database connection
	tenantIDStr := tenantID.String()
	tenantDB, err := middlewares.GetConnectionDynamically(config.DB, nil, &tenantIDStr)
	if err != nil {
		return fmt.Errorf("failed to get tenant DB connection: %w", err)
	}

	// Create user in tenant DB - schema has: name (not first_name/last_name), active (not is_active)
	query := `
		INSERT INTO users (id, email, name, tenant_id, client_id, tenant_domain, provider, provider_id, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, true, $9, $10)
		ON CONFLICT (id) DO NOTHING
	`

	// Get full name
	name := userInfo.Name
	if name == "" {
		name = userInfo.GivenName
		if userInfo.FamilyName != "" {
			name = name + " " + userInfo.FamilyName
		}
	}

	now := time.Now()
	result := tenantDB.Exec(query, userID, userInfo.Email, name, tenantID, clientID, tenantDomain, provider, userInfo.Sub, now, now)
	return result.Error
}

// LinkIdentity links an OIDC provider to an existing user account
// @Summary Link OIDC provider to account
// @Description Links a social login provider to an existing user account
// @Tags OIDC
// @Accept json
// @Produce json
// @Param input body models.LinkOIDCIdentityInput true "Provider to link"
// @Success 200 {object} models.OIDCInitiateResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /uflow/oidc/link [post]
func (oc *OIDCController) LinkIdentity(c *gin.Context) {
	// Get user from context (requires auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found"})
		return
	}

	var input models.LinkOIDCIdentityInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Safely extract tenant ID string
	var tenantIDStr string
	switch v := tenantID.(type) {
	case uuid.UUID:
		tenantIDStr = v.String()
	case string:
		tenantIDStr = v
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID format"})
		return
	}

	tid, err := uuid.Parse(tenantIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	// Get tenant domain
	tenant, tErr := oc.tenantRepo.GetTenantByID(tenantIDStr)
	if tErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get tenant"})
		return
	}

	// Initiate OIDC flow with action "link"
	oidcInput := &models.OIDCInitiateInput{
		TenantDomain: tenant.TenantDomain,
		Provider:     input.Provider,
	}
	response, err := oc.oidcService.InitiateOIDCFlow(oidcInput, "link", &tid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Store user ID in state for linking
	// TODO: Update state with user_id for linking

	c.JSON(http.StatusOK, gin.H{
		"redirect_url": response.RedirectURL,
		"message":      "Redirecting to provider for authorization",
	})
	_ = userID // Will be used when implementing link callback
}

// GetLinkedIdentities returns all OIDC identities linked to current user
// @Summary Get linked OIDC identities
// @Description Returns list of social login providers linked to user account
// @Tags OIDC
// @Produce json
// @Success 200 {array} models.OIDCUserIdentity
// @Failure 401 {object} map[string]string
// @Router /uflow/oidc/identities [get]
func (oc *OIDCController) GetLinkedIdentities(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found"})
		return
	}

	identities, err := oc.oidcService.GetIdentitiesByUser(safeUUID(tenantID), safeUUID(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get identities"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"identities": identities})
}

// UnlinkIdentity removes an OIDC provider from user account
// @Summary Unlink OIDC provider
// @Description Removes a social login provider from user account
// @Tags OIDC
// @Param provider path string true "Provider name (google, github, microsoft)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /uflow/oidc/unlink/{provider} [delete]
func (oc *OIDCController) UnlinkIdentity(c *gin.Context) {
	provider := c.Param("provider")
	if provider == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider name required"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Tenant not found"})
		return
	}

	if err := oc.oidcService.UnlinkIdentity(safeUUID(tenantID), safeUUID(userID), provider); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider unlinked successfully"})
}

// ========================================
// Admin endpoints for managing OIDC providers
// ========================================

// GetAllProviders returns all OIDC providers (admin)
func (oc *OIDCController) GetAllProviders(c *gin.Context) {
	providers, err := oc.oidcService.GetAllProviders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get providers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"providers": providers})
}

// UpdateProvider updates an OIDC provider configuration (admin)
func (oc *OIDCController) UpdateProvider(c *gin.Context) {
	providerName := c.Param("provider")
	if providerName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Provider name required"})
		return
	}

	var input models.OIDCProviderUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := oc.oidcService.UpdateProvider(providerName, &input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Provider updated successfully"})
}

// ========================================
// Helper functions
// ========================================

// renderOAuthCallbackHTML returns HTML page that communicates OAuth results to frontend
func renderOAuthCallbackHTML(c *gin.Context, data map[string]interface{}) {
	// Convert data to JSON string for embedding in JavaScript
	dataJSON, _ := json.Marshal(data)

	log.Printf("DEBUG renderOAuthCallbackHTML: Received data with tenant_domain='%v'", data["tenant_domain"])

	// Determine the frontend redirect URL
	// Priority: 1. tenant_domain from data (preserves user's login domain), 2. Host header, 3. Default
	defaultBaseURL := config.AppConfig.BaseURL
	if defaultBaseURL == "" {
		defaultBaseURL = "https://app.authsec.dev"
	}
	redirectURL := defaultBaseURL + "/authsec/uflow/oidc/callback"

	// Try to use tenant_domain from data first (this preserves the domain the user logged in from)
	if tenantDomain, ok := data["tenant_domain"].(string); ok && tenantDomain != "" {
		// Use the tenant domain that was passed in (from state or database)
		// No validation needed - trust the domain from the state/database
		redirectURL = "https://" + tenantDomain + "/authsec/uflow/oidc/callback"
		log.Printf("DEBUG renderOAuthCallbackHTML: Using tenant_domain from data, redirectURL='%s'", redirectURL)
	} else {
		// Fallback: Try to extract from Host or X-Forwarded-Host header
		host := c.GetHeader("X-Forwarded-Host")
		if host == "" {
			host = c.Request.Host
		}

		// Strip port if present
		if idx := strings.Index(host, ":"); idx != -1 {
			host = host[:idx]
		}

		// Convert API domain to frontend domain
		// dev2.api.authsec.dev -> dev.authsec.dev
		// api.authsec.dev -> app.authsec.dev
		frontendHost := convertAPIToFrontendDomain(host)

		// For platform domains, validate against allowlist
		// For custom domains, trust them (they came from tenant_domains table via state)
		if isAllowedFrontendDomain(frontendHost) || (!strings.HasSuffix(frontendHost, ".authsec.dev") && !strings.HasSuffix(frontendHost, ".authsec.ai")) {
			redirectURL = "https://" + frontendHost + "/authsec/uflow/oidc/callback"
		}
	}

	html := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <title>Authentication</title>
    <script>
        window.onload = function() {
            const data = %s;
            const redirectURL = '%s';

            // If opened in popup window, send message to opener
            if (window.opener && !window.opener.closed) {
                window.opener.postMessage({ type: 'oauth-callback', data: data }, '*');
                setTimeout(() => window.close(), 100);
            } else {
                // If opened in same window, redirect to frontend with query params
                const params = new URLSearchParams();
                for (const [key, value] of Object.entries(data)) {
                    if (value !== null && value !== undefined) {
                        params.append(key, String(value));
                    }
                }
                window.location.href = redirectURL + '?' + params.toString();
            }
        };
    </script>
</head>
<body>
    <p>Processing authentication... Please wait.</p>
</body>
</html>
	`, string(dataJSON), redirectURL)

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// convertAPIToFrontendDomain converts API domain to frontend domain
// Examples:
//   - dev2.authsec.dev -> dev.authsec.dev
//   - dev2.api.authsec.dev -> dev.authsec.dev
//   - api.authsec.dev -> app.authsec.dev
//   - localhost:8080 -> localhost:8080 (unchanged)
func convertAPIToFrontendDomain(host string) string {
	// Specific API to Frontend mappings
	apiToFrontendMap := map[string]string{
		// Dev environment (.authsec.dev)
		"dev2.authsec.dev":     "dev.authsec.dev",     // Dev API -> Dev Frontend
		"dev2.api.authsec.dev": "dev.authsec.dev",     // Dev API (with api subdomain) -> Dev Frontend
		"api.authsec.dev":      "app.authsec.dev",     // Dev Prod API -> Dev Prod Frontend
		"staging.authsec.dev":  "staging.authsec.dev", // Staging (if same)
		// Prod environment (.authsec.ai)
		"prod.api.authsec.ai": "app.authsec.ai", // Prod API -> Prod Frontend
	}

	// Check for exact match
	if frontend, ok := apiToFrontendMap[host]; ok {
		return frontend
	}

	// Handle pattern {env}.api.authsec.dev -> {env}.authsec.dev
	if strings.Contains(host, ".api.authsec.dev") {
		return strings.Replace(host, ".api.authsec.dev", ".authsec.dev", 1)
	}

	// Handle pattern {env}.api.authsec.ai -> {env}.authsec.ai
	if strings.Contains(host, ".api.authsec.ai") {
		return strings.Replace(host, ".api.authsec.ai", ".authsec.ai", 1)
	}

	// No conversion needed (already frontend domain or custom domain)
	return host
}

// isAllowedFrontendDomain checks if the domain is an allowed frontend domain
func isAllowedFrontendDomain(host string) bool {
	// Strip port if present
	if idx := strings.Index(host, ":"); idx != -1 {
		host = host[:idx]
	}

	allowedDomains := []string{
		// Dev environment
		"app.authsec.dev",
		"dev.authsec.dev",
		"dev2.authsec.dev",
		"staging.authsec.dev",
		// Prod environment
		"app.authsec.ai",
		// Local
		"localhost",
	}

	// Check exact match
	for _, allowed := range allowedDomains {
		if host == allowed {
			return true
		}
	}

	// Check if it's a subdomain of authsec.dev or authsec.ai
	if strings.HasSuffix(host, ".authsec.dev") || strings.HasSuffix(host, ".authsec.ai") {
		return true
	}

	return false
}

// isValidTenantDomain validates tenant domain format (subdomain prefix only)
func isValidTenantDomain(domain string) bool {
	if len(domain) < 3 || len(domain) > 63 {
		return false
	}

	// Must start with letter
	if domain[0] < 'a' || domain[0] > 'z' {
		return false
	}

	// Only lowercase letters, numbers, and hyphens
	for _, char := range domain {
		if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
			return false
		}
	}

	// Cannot end with hyphen
	if domain[len(domain)-1] == '-' {
		return false
	}

	return true
}

// isValidTenantDomainOrCustomDomain validates both subdomain prefixes and full custom domains
func isValidTenantDomainOrCustomDomain(domain string) bool {
	if len(domain) == 0 || len(domain) > 253 {
		return false
	}

	// If it contains a dot, it's a full domain (e.g., test.auth-sec.org)
	if strings.Contains(domain, ".") {
		// Basic full domain validation
		// Must not start or end with dot
		if domain[0] == '.' || domain[len(domain)-1] == '.' {
			return false
		}
		// Must not have consecutive dots
		if strings.Contains(domain, "..") {
			return false
		}
		// Each label must be valid
		labels := strings.Split(domain, ".")
		for _, label := range labels {
			if len(label) == 0 || len(label) > 63 {
				return false
			}
			// Label must start with alphanumeric
			if !((label[0] >= 'a' && label[0] <= 'z') || (label[0] >= '0' && label[0] <= '9')) {
				return false
			}
			// Label can contain alphanumeric and hyphens
			for _, char := range label {
				if !((char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-') {
					return false
				}
			}
			// Cannot end with hyphen
			if label[len(label)-1] == '-' {
				return false
			}
		}
		return true
	}

	// If no dot, treat as subdomain prefix - use original validation
	return isValidTenantDomain(domain)
}

// assignAdminRoleToUser assigns admin role to a user in the tenant database
func (oc *OIDCController) assignAdminRoleToUser(tenantDB *sql.DB, userID uuid.UUID, tenantID uuid.UUID) error {
	// Insert admin role if it doesn't exist (check first to avoid deferrable constraint issues)
	var adminRoleID uuid.UUID
	checkRoleExistsQuery := `SELECT id FROM roles WHERE name = 'admin' AND tenant_id = $1`
	err := tenantDB.QueryRow(checkRoleExistsQuery, tenantID).Scan(&adminRoleID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Role doesn't exist, insert it
			insertRoleQuery := `INSERT INTO roles (name, description, tenant_id) VALUES ('admin', 'Administrator role with full access', $1) RETURNING id`
			err = tenantDB.QueryRow(insertRoleQuery, tenantID).Scan(&adminRoleID)
			if err != nil {
				return fmt.Errorf("failed to insert admin role: %w", err)
			}
		} else {
			return fmt.Errorf("failed to check existing admin role: %w", err)
		}
	}

	// Assign admin role via role_bindings (user_roles is deprecated)
	checkBindingQuery := `SELECT 1 FROM role_bindings WHERE user_id = $1 AND role_id = $2 AND tenant_id = $3 AND scope_type IS NULL`
	var exists int
	err = tenantDB.QueryRow(checkBindingQuery, userID, adminRoleID, tenantID).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing role binding: %w", err)
	}

	// Only insert if the role binding doesn't already exist
	if err == sql.ErrNoRows {
		assignBindingQuery := `INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at) VALUES ($1, $2, $3, $4, NULL, NULL, NOW(), NOW())`
		if _, err := tenantDB.Exec(assignBindingQuery, uuid.New(), tenantID, userID, adminRoleID); err != nil {
			return fmt.Errorf("failed to create admin role binding: %w", err)
		}
	}

	return nil
}
