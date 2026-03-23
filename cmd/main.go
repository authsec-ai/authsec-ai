// AuthSec – merged monolith combining user-flow and webauthn-service.
//
// Previously these were two separate microservices:
//
//   - user-flow      (port 7468) – admin/enduser auth, RBAC, OIDC, SCIM, TOTP, CIBA
//   - webauthn-service (port 8080) – WebAuthn passkeys, TOTP setup, SMS MFA
//
// They are now a single process. The only architectural change is that the HTTP
// call webauthn-service previously made to user-flow's /uflow/webauthn/register
// is now an in-process call via the bridge package.
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	authManagerConfig "github.com/authsec-ai/auth-manager/pkg/config"
	"github.com/authsec-ai/authsec/config"
	platformCtrl "github.com/authsec-ai/authsec/controllers/platform"
	"github.com/authsec-ai/authsec/handlers"
	"github.com/authsec-ai/authsec/internal/clients/icp"
	"github.com/authsec-ai/authsec/internal/migration"
	session "github.com/authsec-ai/authsec/internal/session"
	"github.com/authsec-ai/authsec/middlewares"
	"github.com/authsec-ai/authsec/monitoring"
	"github.com/authsec-ai/authsec/routes"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// @title           AuthSec API
// @version         5.0.0
// @description     Unified authentication and MFA service (user-flow + webauthn merged monolith)
// @contact.name   AuthSec AI
// @contact.url    https://authsec.ai
// @contact.email  support@authsec.ai
// @license.name  Apache 2.0
// @BasePath  /uflow
func main() {
	// Load .env file if present (optional, for development)
	godotenv.Load()

	// ─────────────────────────────────────────────────────────
	// Phase 1: user-flow initialisation
	// ─────────────────────────────────────────────────────────

	cfg := config.LoadConfig()

	monitoring.InitMetrics()

	// Initialise primary database
	config.InitDatabaseWithoutGORM(cfg)

	// Run database migrations via the authsec-migration system
	if err := migration.AutoMigrateMigrationLogs(config.DB); err != nil {
		log.Printf("Warning: failed to create migration_logs table: %v", err)
	}
	if err := config.DB.AutoMigrate(platformCtrl.SpireAllModels...); err != nil {
		log.Printf("Warning: failed to auto-migrate SPIRE tables: %v", err)
	}
	if os.Getenv("SKIP_MIGRATIONS") != "true" {
		masterRunner := migration.NewMasterMigrationRunner(
			migration.MigrationsDir("master"),
			config.Database.DB,
			config.DB,
		)
		if err := masterRunner.RunMigrations(); err != nil {
			log.Printf("Warning: master migrations encountered errors (service continuing): %v", err)
		} else {
			log.Println("Master migrations completed successfully")
		}

		// Build the golden tenant template in the background so it is ready for fast cloning.
		migration.InitTemplateCreds(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBSSLMode)
		go func() {
			if err := migration.SetupTenantTemplate(migration.MigrationsDir("tenant")); err != nil {
				log.Printf("Warning: tenant template setup failed (standard migration path remains available): %v", err)
			}
		}()
	}

	// Initialise auth-manager configuration
	authManagerConfig.LoadConfig()
	authManagerConfig.SetDB(config.DB)
	authManagerConfig.InitTenantDBResolver(config.DB, nil)

	// Initialise Vault (optional; logs warning if not configured)
	config.InitVault(cfg)

	// Centralised token service (used by controllers and the bridge)
	tokenService, err := services.NewAuthManagerTokenService()
	if err != nil {
		monitoring.GetLogger().WithError(err).Fatal("Failed to initialize auth-manager token service")
	}
	config.TokenService = tokenService
	monitoring.GetLogger().Info("Auth-manager token service initialized")

	// Redis cache (optional)
	var cacheManager *monitoring.CacheManager
	if cfg.RedisURL != "" {
		cacheManager, err = monitoring.NewCacheManager(cfg.RedisURL)
		if err != nil {
			monitoring.GetLogger().WithError(err).Warn("Failed to initialize Redis cache, continuing without cache")
		} else {
			monitoring.GetLogger().Info("Redis cache initialized")
		}
	}

	// Audit logger
	auditLogger := monitoring.NewAuditLogger(config.DB)
	if err := auditLogger.InitAuditTable(); err != nil {
		monitoring.GetLogger().WithError(err).Fatal("Failed to initialize audit table")
	}

	config.CacheManager = cacheManager
	config.AuditLogger = auditLogger

	// ─────────────────────────────────────────────────────────
	// Phase 2: webauthn-service initialisation
	// ─────────────────────────────────────────────────────────

	// Validate WebAuthn-specific environment variables
	if err := validateWebAuthnEnvVars(); err != nil {
		log.Fatal("WebAuthn environment validation failed:", err)
	}

	rpName := getEnv("WEBAUTHN_RP_NAME", "AuthSec")
	rpIDRaw := getEnv("WEBAUTHN_RP_ID", "localhost")
	rpID := config.NormalizeRPID(rpIDRaw)
	origin := getEnv("WEBAUTHN_ORIGIN", "http://localhost:3000")

	webAuthn := config.SetupWebAuthn(rpName, rpID, origin)

	// PostgreSQL session manager for WebAuthn challenges (uses the same global DB)
	pgSessionManager := session.NewPostgreSQLSessionManager(config.DB, "")
	if err := pgSessionManager.CleanupExpiredSessions(); err != nil {
		log.Printf("Warning: failed to cleanup expired WebAuthn sessions: %v", err)
	}

	sessionAdapter := handlers.NewSessionManagerAdapter(pgSessionManager)

	webAuthnHandler := &handlers.WebAuthnHandler{
		WebAuthn:       webAuthn,
		SessionManager: sessionAdapter,
	}

	adminWebAuthnHandler := &handlers.AdminWebAuthnHandler{
		WebAuthn:       webAuthn,
		SessionManager: sessionAdapter,
		RPDisplayName:  rpName,
		RPID:           rpID,
		RPOrigins:      []string{origin},
	}

	endUserWebAuthnHandler := &handlers.EndUserWebAuthnHandler{
		WebAuthn:       webAuthn,
		SessionManager: sessionAdapter,
		RPDisplayName:  rpName,
		RPID:           rpID,
		RPOrigins:      []string{origin},
	}

	log.Printf("WebAuthn handlers initialized (RP Name: %s, RP ID: %s, Origin: %s)", rpName, rpID, origin)

	// ─────────────────────────────────────────────────────────
	// Phase 3: HTTP router
	// ─────────────────────────────────────────────────────────

	r := gin.New()

	// Metrics (must be first)
	r.Use(monitoring.Middleware())

	// CORS (validates against CORS_ALLOW_ORIGIN config)
	r.Use(middlewares.CORSMiddleware())

	// Core middleware
	r.Use(middlewares.RequestIDMiddleware())
	r.Use(middlewares.AuthLoggingMiddleware("authsec"))
	r.Use(middlewares.SecurityHeadersMiddleware())
	r.Use(middlewares.RecoveryMiddleware())
	r.Use(middlewares.TimeoutMiddleware(120 * time.Second))
	r.Use(middlewares.MennovRateLimitMiddleware())
	r.Use(gzip.Gzip(gzip.DefaultCompression))

	// Prometheus metrics endpoint
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// All routes (user-flow + webauthn)
	routes.SetupRoutes(r, webAuthnHandler, adminWebAuthnHandler, endUserWebAuthnHandler)

	// ─────────────────────────────────────────────────────────
	// Phase 4: background workers
	// ─────────────────────────────────────────────────────────

	// Audit log cleanup (runs daily, removes events older than 90 days)
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			if err := auditLogger.CleanupOldEvents(90 * 24 * time.Hour); err != nil {
				monitoring.GetLogger().WithError(err).Error("Failed to cleanup old audit events")
			}
		}
	}()

	// System metrics update (every 30 seconds)
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			monitoring.UpdateSystemMetrics()
		}
	}()

	// PKI retry worker
	icpToken, err := services.GenerateOIDCServiceToken()
	if err != nil {
		log.Printf("Warning: failed to generate ICP service token for PKI retry worker: %v", err)
	} else {
		icpClient := icp.NewClient(cfg.ICPServiceURL, icpToken)
		icpService := services.NewICPProvisioningService(icpClient)
		pkiWorker := services.NewPKIRetryWorker(config.GetDatabase(), icpService, 5*time.Minute)
		pkiWorker.Start()
		log.Printf("PKI retry worker started")
	}

	// ─────────────────────────────────────────────────────────
	// Phase 5: start server
	// ─────────────────────────────────────────────────────────

	port := cfg.Port
	if port == "" {
		port = getEnv("PORT", "7468")
	}

	monitoring.GetLogger().
		WithField("port", port).
		WithField("webauthn_rp_id", rpID).
		Info("Starting AuthSec monolith")

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server (30s grace period)...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced shutdown: %v", err)
	}
	log.Println("Server exited cleanly")
}

// ─────────────────────────────────────────────────────────────────────────────
// Helpers (previously in webauthn-service cmd/main.go)
// ─────────────────────────────────────────────────────────────────────────────

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func validateWebAuthnEnvVars() error {
	required := []string{"WEBAUTHN_RP_NAME", "WEBAUTHN_RP_ID", "WEBAUTHN_ORIGIN"}
	for _, env := range required {
		if os.Getenv(env) == "" {
			return fmt.Errorf("required environment variable %s is not set", env)
		}
	}
	return nil
}

func splitAndTrim(csv string) []string {
	values := strings.Split(csv, ",")
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}
