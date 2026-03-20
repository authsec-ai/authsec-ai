package shared

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/authsec-ai/authsec/config"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

var seededTenantID uuid.UUID

// createTempDB creates a throwaway Postgres database for tests and returns a cleanup fn.
func createTempDB() (string, func(), error) {
	host := getenvDefault("DB_HOST", "localhost")
	port := getenvDefault("DB_PORT", "5432")
	user := getenvDefault("DB_USER", "postgres")
	pass := getenvDefault("DB_PASSWORD", "postgres")

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

	cleanup := func() {
		// terminate any connections to the test DB before dropping
		_, _ = adminDB.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1`, dbName)
		_, _ = adminDB.Exec(`DROP DATABASE IF EXISTS "` + dbName + `"`)
		adminDB.Close()
	}

	return dbName, cleanup, nil
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// TestMain sets up a clean database per test run when RUN_INTEGRATION=1.
func TestMain(m *testing.M) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		log.Println("Skipping controllers integration tests (set RUN_INTEGRATION=1 to enable)")
		os.Exit(0)
	}

	dbName, cleanup, err := createTempDB()
	if err != nil {
		log.Fatalf("Failed to create temp test database: %v", err)
	}
	defer cleanup()

	// Point the app at the temp DB and allow migrations/DB init
	os.Setenv("DB_NAME", dbName)
	os.Unsetenv("SKIP_DB_INIT")
	os.Unsetenv("SKIP_MIGRATIONS")
	os.Unsetenv("SKIP_CONTROLLER_DB_SETUP")
	// Disable server-auth gate in tests and align JWT secrets
	os.Setenv("REQUIRE_SERVER_AUTH", "false")
	os.Setenv("JWT_DEF_SECRET", "test-jwt-secret-key-for-testing-purposes-only")
	os.Setenv("JWT_SDK_SECRET", "test-jwt-secret-key-for-testing-purposes-only")

	cfg := config.LoadConfig()
	config.InitDatabaseWithoutGORM(cfg)

	// Seed minimal admin role/user for tests
	if err := seedTestAdmin(dbName, cfg); err != nil {
		log.Fatalf("Failed to seed test admin: %v", err)
	}

	code := m.Run()
	os.Exit(code)
}

func seedTestAdmin(dbName string, cfg *config.Config) error {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, dbName, cfg.DBPort)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	tenantID := uuid.New()
	seededTenantID = tenantID
	userID := tenantID
	clientID := tenantID

	// ensure tenants table
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tenants (tenant_id uuid PRIMARY KEY, email text, tenant_domain text, tenant_db text);`); err != nil {
		return err
	}
	_, _ = db.Exec(`ALTER TABLE tenants ADD COLUMN IF NOT EXISTS tenant_db text;`)
	// ensure users table columns exist
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS client_id uuid;`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_id uuid;`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_domain text;`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash text;`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS active boolean DEFAULT true;`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS temporary_password boolean DEFAULT false;`)
	_, _ = db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS temporary_password_expires_at timestamp with time zone;`)

	// ensure roles table
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS roles (id uuid PRIMARY KEY, tenant_id uuid, name text NOT NULL, description text, created_at timestamptz default now(), updated_at timestamptz default now(), UNIQUE(tenant_id,name));`)
	// ensure role_bindings table (user_roles is deprecated)
	_, _ = db.Exec(`CREATE TABLE IF NOT EXISTS role_bindings (id uuid PRIMARY KEY, tenant_id uuid, user_id uuid, role_id uuid, scope_type text, scope_id uuid, created_at timestamptz default now(), updated_at timestamptz default now());`)

	// upsert tenant
	_, _ = db.Exec(`INSERT INTO tenants (tenant_id, email, tenant_domain, tenant_db) VALUES ($1,$2,$3,$4) ON CONFLICT (tenant_id) DO UPDATE SET tenant_db=EXCLUDED.tenant_db`, tenantID, "admin@test.local", "test.local", dbName)

	// upsert admin user
	_, _ = db.Exec(`INSERT INTO users (id, email, password_hash, client_id, tenant_id, tenant_domain, active) VALUES ($1,$2,'', $3, $4, $5, true) ON CONFLICT (id) DO NOTHING`,
		userID, "admin@test.local", clientID, tenantID, "test.local")

	// upsert admin role - use constraint name to avoid ambiguity with multiple unique indexes
	var roleID uuid.UUID
	if err := db.QueryRow(`INSERT INTO roles (id, tenant_id, name, description) VALUES ($1, $2, 'admin', 'admin role') ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO UPDATE SET name=EXCLUDED.name RETURNING id`,
		uuid.New(), tenantID).Scan(&roleID); err != nil {
		return err
	}
	// assign role via role_bindings (user_roles is deprecated)
	bindingID := uuid.New()
	_, _ = db.Exec(`INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at) SELECT $1,$2,$3,$4,NULL,NULL,now(),now() WHERE NOT EXISTS (SELECT 1 FROM role_bindings WHERE tenant_id=$2 AND user_id=$3 AND role_id=$4)`, bindingID, tenantID, userID, roleID)

	return nil
}
