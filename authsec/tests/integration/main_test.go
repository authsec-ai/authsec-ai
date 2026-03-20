//go:build integration

package integration

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	authManagerConfig "github.com/authsec-ai/auth-manager/pkg/config"
	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/handlers"
	"github.com/authsec-ai/authsec/internal/migration"
	session "github.com/authsec-ai/authsec/internal/session"
	"github.com/authsec-ai/authsec/routes"
	"github.com/authsec-ai/authsec/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// Package-level test state
var (
	testRouter *gin.Engine
	testDBName string

	testTenantID     uuid.UUID
	testAdminUserID  uuid.UUID
	testEndUserID    uuid.UUID
	testClientID     uuid.UUID
	testProjectID    uuid.UUID
	testAdminRoleID  uuid.UUID
	testAdminEmail   = "admin@test.authsec.local"
	testEndUserEmail = "enduser@test.authsec.local"
	testTenantDomain = "test.authsec.local"
	testPassword     = "TestPassword123!"

	jwtDefSecret = "test-integration-jwt-def-secret-32chars!"
	jwtSdkSecret = "test-integration-jwt-sdk-secret-32chars!"
)

func TestMain(m *testing.M) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		log.Println("Skipping integration tests (set RUN_INTEGRATION=1 to enable)")
		os.Exit(0)
	}

	// Create temporary test database
	dbName, cleanup, err := createTempDB()
	if err != nil {
		log.Fatalf("Failed to create temp test database: %v", err)
	}
	defer cleanup()
	testDBName = dbName

	// Set environment variables
	setTestEnv(dbName)

	// Initialize database (mirrors cmd/main.go)
	cfg := config.LoadConfig()
	config.InitDatabaseWithoutGORM(cfg)

	// Run migrations
	if err := migration.AutoMigrateMigrationLogs(config.DB); err != nil {
		log.Printf("Warning: failed to create migration_logs table: %v", err)
	}

	masterRunner := migration.NewMasterMigrationRunner(
		migration.MigrationsDir("master"),
		config.Database.DB,
		config.DB,
	)
	if err := masterRunner.RunMigrations(); err != nil {
		log.Printf("Warning: master migrations encountered errors: %v", err)
	}

	// Initialize auth-manager config
	authManagerConfig.LoadConfig()
	authManagerConfig.SetDB(config.DB)
	authManagerConfig.InitTenantDBResolver(config.DB, nil)

	// Initialize token service
	tokenService, err := services.NewAuthManagerTokenService()
	if err != nil {
		log.Printf("Warning: Failed to initialize token service: %v", err)
	} else {
		config.TokenService = tokenService
	}

	// Seed test data
	if err := seedTestData(); err != nil {
		log.Fatalf("Failed to seed test data: %v", err)
	}

	// Setup WebAuthn handlers
	rpName := "AuthSec Test"
	rpID := "localhost"
	origin := "http://localhost:3000"

	wa := config.SetupWebAuthn(rpName, rpID, origin)
	pgSessionManager := session.NewPostgreSQLSessionManager(config.DB, "")
	sessionAdapter := handlers.NewSessionManagerAdapter(pgSessionManager)

	webAuthnHandler := &handlers.WebAuthnHandler{
		WebAuthn:       wa,
		SessionManager: sessionAdapter,
		RPDisplayName:  rpName,
		RPID:           rpID,
		RPOrigins:      []string{origin},
	}
	adminWebAuthnHandler := &handlers.AdminWebAuthnHandler{
		WebAuthn:       wa,
		SessionManager: sessionAdapter,
		RPDisplayName:  rpName,
		RPID:           rpID,
		RPOrigins:      []string{origin},
	}
	endUserWebAuthnHandler := &handlers.EndUserWebAuthnHandler{
		WebAuthn:       wa,
		SessionManager: sessionAdapter,
		RPDisplayName:  rpName,
		RPID:           rpID,
		RPOrigins:      []string{origin},
	}

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	testRouter = gin.New()
	testRouter.Use(gin.Recovery())
	routes.SetupRoutes(testRouter, webAuthnHandler, adminWebAuthnHandler, endUserWebAuthnHandler)

	// Run tests
	code := m.Run()

	// Close connections before cleanup drops the database
	if config.Database != nil && config.Database.DB != nil {
		config.Database.DB.Close()
	}
	if sqlDB, err := config.DB.DB(); err == nil {
		sqlDB.Close()
	}

	os.Exit(code)
}

func setTestEnv(dbName string) {
	os.Setenv("DB_HOST", "localhost")
	os.Setenv("DB_PORT", "5432")
	os.Setenv("DB_USER", "kloudone")
	os.Setenv("DB_PASSWORD", "kloudone")
	os.Setenv("DB_NAME", dbName)
	os.Setenv("DB_SCHEMA", "public")

	os.Setenv("JWT_DEF_SECRET", jwtDefSecret)
	os.Setenv("JWT_SDK_SECRET", jwtSdkSecret)
	os.Setenv("JWT_SECRET", jwtDefSecret)

	os.Setenv("REQUIRE_SERVER_AUTH", "false")
	os.Setenv("ENVIRONMENT", "development")

	os.Setenv("WEBAUTHN_RP_NAME", "AuthSec Test")
	os.Setenv("WEBAUTHN_RP_ID", "localhost")
	os.Setenv("WEBAUTHN_ORIGIN", "http://localhost:3000")

	os.Setenv("TOTP_ENCRYPTION_KEY", "6AB33320B8A8E177655F72CEDDAE56593D045BE5A47416FDE7C7CF983D5B80D6")

	os.Setenv("HYDRA_ADMIN_URL", "http://localhost:4445")
	os.Setenv("OOC_MANAGER_URL", "http://localhost:7467")

	os.Unsetenv("SKIP_DB_INIT")
	os.Unsetenv("SKIP_MIGRATIONS")
	os.Unsetenv("SKIP_CONTROLLER_DB_SETUP")
}

func createTempDB() (string, func(), error) {
	host := envDefault("DB_HOST", "localhost")
	port := envDefault("DB_PORT", "5432")
	user := envDefault("DB_USER", "kloudone")
	pass := envDefault("DB_PASSWORD", "kloudone")

	dbName := "test_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	adminDSN := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable", host, port, user, pass)

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return "", nil, fmt.Errorf("open admin DB: %w", err)
	}
	if err := adminDB.Ping(); err != nil {
		adminDB.Close()
		return "", nil, fmt.Errorf("ping admin DB: %w", err)
	}

	if _, err := adminDB.Exec(`CREATE DATABASE "` + dbName + `"`); err != nil {
		adminDB.Close()
		return "", nil, fmt.Errorf("create test db: %w", err)
	}
	log.Printf("Created temp test database: %s", dbName)

	cleanup := func() {
		_, _ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, dbName)
		_, _ = adminDB.Exec(`DROP DATABASE IF EXISTS "` + dbName + `"`)
		adminDB.Close()
		log.Printf("Cleaned up temp test database: %s", dbName)
	}

	return dbName, cleanup, nil
}

func envDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
