package config

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/authsec-ai/authsec/monitoring"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// AuthManagerTokenService interface for token generation using auth-manager patterns
// This avoids import cycles while providing type-safe token generation
type AuthManagerTokenService interface {
	GenerateAdminToken(adminUserID uuid.UUID, email string, projectID uuid.UUID, tenantID *uuid.UUID, tenantDomain string, roles []string) (string, error)
	GenerateTenantUserToken(userID uuid.UUID, tenantID uuid.UUID, projectID uuid.UUID, email string, expiresIn time.Duration) (string, error)
	GenerateEndUserToken(userID uuid.UUID, tenantID string, clientID string, email string, scopes []string, expiresIn time.Duration) (string, error)
	GenerateVoiceAuthToken(userID uuid.UUID, tenantID uuid.UUID, email string, scopes []string, expiresIn time.Duration) (string, error)
	GenerateDeviceAuthToken(userID uuid.UUID, tenantID uuid.UUID, email string, scopes []string, expiresIn time.Duration) (string, error)
	GenerateCIBAToken(userID uuid.UUID, tenantID uuid.UUID, email string, scopes []string, expiresIn time.Duration) (string, error)
	GenerateTenantCIBAToken(userID uuid.UUID, tenantID uuid.UUID, clientID uuid.UUID, email string, scopes []string, expiresIn time.Duration) (string, error)
	GenerateTOTPToken(userID uuid.UUID, tenantID uuid.UUID, email string, expiresIn time.Duration) (string, error)
}

type Config struct {
	Port               string
	DBName             string
	DBUser             string
	DBPassword         string
	DBHost             string
	DBPort             string
	DBSchema           string
	DBSSLMode          string
	DatabaseURL        string
	JWTDefSecret       string
	JWTSdkSecret       string
	JWTSecret          string // Primary JWT secret (ext-service / hydra-service / SPIFFE delegate)
	OOCManagerURL      string
	VaultAddr          string
	VaultToken         string
	HydraAdminURL      string
	SMTPHost           string
	SMTPPort           string
	SMTPUser           string
	SMTPPassword       string
	TenantDomainSuffix string
	CorsAllowOrigin    string
	RedisURL           string
	ICPServiceURL      string // ICP service URL for PKI provisioning
	BaseURL            string // Base URL for callbacks (e.g., https://app.authsec.dev)

	// Runtime environment ("development" | "production" | "staging")
	Environment string

	// TOTP / encryption
	TotpEncryptionKey       string // 64-hex-char AES-256 key for TOTP secrets at rest
	SyncConfigEncryptionKey string // 64-hex-char AES-256 key for AD/Entra sync configs

	// Twilio (SMS MFA / voice)
	TwilioAccountSid string
	TwilioAuthToken  string
	TwilioFromNumber string

	// SPIFFE / SVID OIDC
	SpiffeOIDCIssuer       string
	SpiffeJWKSKeyID        string
	SpiffeRSAPrivateKeyB64 string
	SpiffeTrustDomain      string

	// Okta CIBA integration
	OktaDomain       string
	OktaClientID     string
	OktaClientSecret string
	OktaIssuer       string
	OktaAPIToken     string

	// OIDC token validation
	AuthExpectIss string // Expected issuer claim for incoming OIDC tokens
	AuthExpectAud string // Expected audience claim for incoming OIDC tokens

	// Server auth gate (maps to REQUIRE_SERVER_AUTH; default true)
	RequireServerAuth string

	// OIDC Provider credentials (fallback when Vault is not available)
	GoogleClientSecret    string
	GitHubClientSecret    string
	MicrosoftClientSecret string

	// HubSpot integration
	HubSpotAccessToken string

	// Hydra service fields
	HydraPublicURL      string // Hydra public endpoint (e.g., https://hydra.authsec.dev)
	ReactAppURL         string // Frontend app URL for redirects (e.g., https://app.authsec.dev)
	IdentityProviderURL string // Identity provider base URL for OIDC callbacks

	// SDK-Manager migration (all optional)
	OAuthAuthURL             string // OAuth authorization endpoint
	OAuthTokenURL            string // OAuth token exchange endpoint
	OAuthUserInfoURL         string // OAuth userinfo endpoint
	PKCEChallenge            string // Pre-computed PKCE challenge (if static)
	OAuthRedirectURI         string // Default OAuth redirect URI
	OAuthRedirectURITemplate string // Redirect URI template with {tenant_id}
	MCPToolTimeout           int    // MCP tool execution timeout in seconds (default 15)

	// Azure OpenAI (for playground + voice)
	AzureOpenAIKey           string
	AzureOpenAIEndpoint      string
	AzureOpenAIDeployment    string
	AzureOpenAIVersion       string
	AzureOpenAITTSDeployment string

	// SDK behavior flags
	SDKAlwaysExposeProtectedTools bool   // default true
	SDKHideUnauthorizedTools      bool   // default false
	SDKRequireSessionID           bool   // default false
	SDKRedirectSource             string // "db" or "env"
}

var (
	AppConfig    *Config
	CacheManager *monitoring.CacheManager
	AuditLogger  *monitoring.AuditLogger
	TokenService AuthManagerTokenService // Global token service using auth-manager patterns

	// Redis client singleton
	redisClient *redis.Client
	redisOnce   sync.Once
)

func LoadConfig() *Config {
	// Return existing config if already loaded
	if AppConfig != nil {
		return AppConfig
	}

	// Load individual database variables
	dbName := getEnv("DB_NAME", "kloudone_db")
	dbUser := getEnv("DB_USER", "")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbHost := getEnv("DB_HOST", "postgres")
	dbPort := getEnv("DB_PORT", "5432")
	dbSchema := getEnv("DB_SCHEMA", "public")
	dbSSLMode := getEnv("DB_SSL_MODE", "disable")

	// Construct DatabaseURL
	databaseURL := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s search_path=%s",
		dbHost, dbUser, dbPassword, dbName, dbPort, dbSSLMode, dbSchema,
	)

	err := os.Setenv("DATABASE_URL", databaseURL)
	if err != nil {
		fmt.Println("Error setting env var:", err)
	}

	// Load other configuration variables
	port := getEnv("PORT", "7468")
	jwtSdkSecret := getEnv("JWT_SDK_SECRET", "")
	jwtDefSecret := getEnv("JWT_DEF_SECRET", "")
	jwtSecret := getEnv("JWT_SECRET", "")
	oocManagerURL := getEnv("OOC_MANAGER_URL", "http://localhost:7467")

	vaultAddr := getEnv("VAULT_ADDR", "http://localhost:8200")
	vaultToken := getEnv("VAULT_TOKEN", "")
	hydraAdminURL := getEnv("HYDRA_ADMIN_URL", "http://localhost:4445")

	corsAllowOrigin := getEnv("CORS_ALLOW_ORIGIN", "https://*.app.authsec.dev,https://app.authsec.dev,https://*.authsec.dev,https://authsec.dev,https://*.authsec.ai,https://200xx.app.authsec.dev")

	// Load Tenant Domain Suffix
	tenantDomainSuffix := getEnv("TENANT_DOMAIN_SUFFIX", "app.authsec.ai")

	// Load Redis configuration
	redisURL := getEnv("REDIS_URL", "")

	// Load ICP service configuration
	icpServiceURL := getEnv("ICP_SERVICE_URL", "http://localhost:7001")

	// Load SMTP configuration
	smtpHost := getEnv("SMTP_HOST", "")
	smtpPort := getEnv("SMTP_PORT", "")
	smtpUser := getEnv("SMTP_USER", "")
	smtpPassword := getEnv("SMTP_PASSWORD", "")

	// Load Base URL for OIDC callbacks
	baseURL := getEnv("BASE_URL", "https://app.authsec.dev")

	// Runtime environment
	environment := getEnv("ENVIRONMENT", "development")

	// TOTP / encryption keys
	totpEncryptionKey := getEnv("TOTP_ENCRYPTION_KEY", "")
	syncConfigEncryptionKey := getEnv("SYNC_CONFIG_ENCRYPTION_KEY", "")

	// Twilio (SMS MFA)
	twilioAccountSid := getEnv("TWILIO_ACCOUNT_SID", "")
	twilioAuthToken := getEnv("TWILIO_AUTH_TOKEN", "")
	twilioFromNumber := getEnv("TWILIO_FROM_NUMBER", "")

	// SPIFFE / SVID OIDC
	spiffeOIDCIssuer := getEnv("SPIFFE_OIDC_ISSUER", "")
	spiffeJWKSKeyID := getEnv("SPIFFE_JWKS_KEY_ID", "")
	spiffeRSAPrivateKeyB64 := getEnv("SPIFFE_RSA_PRIVATE_KEY_B64", "")
	spiffeTrustDomain := getEnv("SPIFFE_TRUST_DOMAIN", "")

	// Okta CIBA integration
	oktaDomain := getEnv("OKTA_DOMAIN", "")
	oktaClientID := getEnv("OKTA_CLIENT_ID", "")
	oktaClientSecret := getEnv("OKTA_CLIENT_SECRET", "")
	oktaIssuer := getEnv("OKTA_ISSUER", "")
	oktaAPIToken := getEnv("OKTA_API_TOKEN", "")

	// OIDC token validation expectations
	authExpectIss := getEnv("AUTH_EXPECT_ISS", "")
	authExpectAud := getEnv("AUTH_EXPECT_AUD", "")

	// Server auth gate
	requireServerAuth := getEnv("REQUIRE_SERVER_AUTH", "true")

	// Load OIDC provider credentials (fallback for when Vault is not available)
	googleClientSecret := getEnv("GOOGLE_CLIENT_SECRET", "")
	githubClientSecret := getEnv("GITHUB_CLIENT_SECRET", "")
	microsoftClientSecret := getEnv("MICROSOFT_CLIENT_SECRET", "")

	// Load HubSpot configuration
	hubSpotAccessToken := getEnv("HUBSPOT_ACCESS_TOKEN", "")

	// Load Hydra service configuration
	hydraPublicURL := getEnv("HYDRA_PUBLIC_URL", "http://localhost:4444")
	reactAppURL := getEnv("REACT_APP_URL", "https://app.authsec.dev")
	identityProviderURL := getEnv("IDENTITY_PROVIDER_URL", "https://app.authsec.dev")

	// SDK-Manager migration config (all optional)
	oauthAuthURL := getEnv("OAUTH_AUTH_URL", "")
	oauthTokenURL := getEnv("OAUTH_TOKEN_URL", "")
	oauthUserInfoURL := getEnv("OAUTH_USERINFO_URL", "")
	pkceChallenge := getEnv("PKCE_CHALLENGE", "")
	oauthRedirectURI := getEnv("OAUTH_REDIRECT_URI", "")
	oauthRedirectURITemplate := getEnv("OAUTH_REDIRECT_URI_TEMPLATE", "")
	mcpToolTimeout := 15
	if v := os.Getenv("MCP_TOOL_TIMEOUT"); v != "" {
		if parsed, err := fmt.Sscanf(v, "%d", &mcpToolTimeout); err != nil || parsed == 0 {
			mcpToolTimeout = 15
		}
	}

	// Azure OpenAI
	azureOpenAIKey := getEnv("AZURE_OPENAI_API_KEY", "")
	azureOpenAIEndpoint := getEnv("AZURE_OPENAI_ENDPOINT", "")
	azureOpenAIDeployment := getEnv("AZURE_OPENAI_DEPLOYMENT", "")
	azureOpenAIVersion := getEnv("AZURE_OPENAI_VERSION", "2024-02-15-preview")
	azureOpenAITTSDeployment := getEnv("AZURE_OPENAI_TTS_DEPLOYMENT", "tts")

	// SDK behavior flags
	sdkAlwaysExposeProtectedTools := getEnv("AUTHSEC_ALWAYS_EXPOSE_PROTECTED_TOOLS", "true") == "true"
	sdkHideUnauthorizedTools := getEnv("AUTHSEC_HIDE_UNAUTHORIZED_TOOLS", "false") == "true"
	sdkRequireSessionID := getEnv("AUTHSEC_REQUIRE_SESSION_ID", "false") == "true"
	sdkRedirectSource := getEnv("AUTHSEC_REDIRECT_SOURCE", "db")

	// Validate critical variables
	if dbName == "" || dbUser == "" || dbHost == "" || dbPort == "" {
		log.Fatal("DB_NAME, DB_USER, DB_HOST, and DB_PORT are required")
	}
	if port == "" {
		log.Fatal("PORT is required")
	}

	AppConfig = &Config{
		Port:                    port,
		DBName:                  dbName,
		DBUser:                  dbUser,
		DBPassword:              dbPassword,
		DBHost:                  dbHost,
		DBPort:                  dbPort,
		DBSchema:                dbSchema,
		DBSSLMode:               dbSSLMode,
		DatabaseURL:             databaseURL,
		JWTDefSecret:            jwtDefSecret,
		JWTSdkSecret:            jwtSdkSecret,
		JWTSecret:               jwtSecret,
		OOCManagerURL:           oocManagerURL,
		VaultAddr:               vaultAddr,
		VaultToken:              vaultToken,
		HydraAdminURL:           hydraAdminURL,
		SMTPHost:                smtpHost,
		SMTPPort:                smtpPort,
		SMTPUser:                smtpUser,
		SMTPPassword:            smtpPassword,
		TenantDomainSuffix:      tenantDomainSuffix,
		CorsAllowOrigin:         corsAllowOrigin,
		RedisURL:                redisURL,
		ICPServiceURL:           icpServiceURL,
		BaseURL:                 baseURL,
		Environment:             environment,
		TotpEncryptionKey:       totpEncryptionKey,
		SyncConfigEncryptionKey: syncConfigEncryptionKey,
		TwilioAccountSid:        twilioAccountSid,
		TwilioAuthToken:         twilioAuthToken,
		TwilioFromNumber:        twilioFromNumber,
		SpiffeOIDCIssuer:        spiffeOIDCIssuer,
		SpiffeJWKSKeyID:         spiffeJWKSKeyID,
		SpiffeRSAPrivateKeyB64:  spiffeRSAPrivateKeyB64,
		SpiffeTrustDomain:       spiffeTrustDomain,
		OktaDomain:              oktaDomain,
		OktaClientID:            oktaClientID,
		OktaClientSecret:        oktaClientSecret,
		OktaIssuer:              oktaIssuer,
		OktaAPIToken:            oktaAPIToken,
		AuthExpectIss:           authExpectIss,
		AuthExpectAud:           authExpectAud,
		RequireServerAuth:       requireServerAuth,
		GoogleClientSecret:      googleClientSecret,
		GitHubClientSecret:      githubClientSecret,
		MicrosoftClientSecret:   microsoftClientSecret,
		HubSpotAccessToken:      hubSpotAccessToken,
		HydraPublicURL:          hydraPublicURL,
		ReactAppURL:             reactAppURL,
		IdentityProviderURL:     identityProviderURL,

		// SDK-Manager migration
		OAuthAuthURL:                  oauthAuthURL,
		OAuthTokenURL:                 oauthTokenURL,
		OAuthUserInfoURL:              oauthUserInfoURL,
		PKCEChallenge:                 pkceChallenge,
		OAuthRedirectURI:              oauthRedirectURI,
		OAuthRedirectURITemplate:      oauthRedirectURITemplate,
		MCPToolTimeout:                mcpToolTimeout,
		AzureOpenAIKey:                azureOpenAIKey,
		AzureOpenAIEndpoint:           azureOpenAIEndpoint,
		AzureOpenAIDeployment:         azureOpenAIDeployment,
		AzureOpenAIVersion:            azureOpenAIVersion,
		AzureOpenAITTSDeployment:      azureOpenAITTSDeployment,
		SDKAlwaysExposeProtectedTools: sdkAlwaysExposeProtectedTools,
		SDKHideUnauthorizedTools:      sdkHideUnauthorizedTools,
		SDKRequireSessionID:           sdkRequireSessionID,
		SDKRedirectSource:             sdkRedirectSource,
	}

	// Validate required secrets are set — fail fast if missing (warn-only in test mode)
	requiredSecrets := map[string]string{
		"DB_USER":        dbUser,
		"DB_PASSWORD":    dbPassword,
		"JWT_SDK_SECRET": jwtSdkSecret,
		"JWT_DEF_SECRET": jwtDefSecret,
	}
	for name, val := range requiredSecrets {
		if val == "" {
			if testing.Testing() {
				log.Printf("WARNING: Required config %s is not set (test mode, continuing)", name)
			} else {
				log.Fatalf("CRITICAL: Required config %s is not set. Cannot start.", name)
			}
		}
	}

	return AppConfig
}

// ADD THIS: New getter function
func GetConfig() *Config {
	if AppConfig == nil {
		log.Fatal("Configuration not loaded. Call LoadConfig() first.")
	}
	return AppConfig
}

// sensitiveKeywords lists substrings that indicate a key holds a secret value.
var sensitiveKeywords = []string{"SECRET", "PASSWORD", "TOKEN", "KEY", "CREDENTIAL"}

func isSensitiveKey(key string) bool {
	upper := strings.ToUpper(key)
	for _, s := range sensitiveKeywords {
		if strings.Contains(upper, s) {
			return true
		}
	}
	return false
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		if isSensitiveKey(key) {
			log.Printf("Loaded %s: ***", key)
		} else {
			log.Printf("Loaded %s: %s", key, value)
		}
		return value
	}
	if isSensitiveKey(key) {
		log.Printf("Using fallback for %s: ***", key)
	} else {
		log.Printf("Using fallback for %s: %s", key, fallback)
	}
	return fallback
}

// GetEnv is a public wrapper for getEnv
func GetEnv(key, fallback string) string {
	return getEnv(key, fallback)
}

// GetRedisClient returns a singleton Redis client instance
func GetRedisClient() *redis.Client {
	redisOnce.Do(func() {
		if AppConfig == nil {
			log.Println("Warning: Config not loaded, Redis client may use default configuration")
		}

		// Get Redis configuration
		redisURL := ""
		if AppConfig != nil {
			redisURL = AppConfig.RedisURL
		}
		if redisURL == "" {
			redisURL = getEnv("REDIS_URL", "redis://localhost:6379")
		}

		// Parse Redis URL or use default options
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			// Fallback to default Redis configuration
			log.Printf("Failed to parse Redis URL, using default: %v", err)
			opt = &redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			}
		}

		redisClient = redis.NewClient(opt)

		// Test connection
		ctx := context.Background()
		_, err = redisClient.Ping(ctx).Result()
		if err != nil {
			log.Printf("Warning: Failed to connect to Redis: %v", err)
			log.Println("Token revocation and refresh features will not be available")
		} else {
			log.Println("Successfully connected to Redis")
		}
	})

	return redisClient
}
