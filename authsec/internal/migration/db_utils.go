package migration

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

// TemplateDBName is the name of the golden tenant template database.
var TemplateDBName = "_authsec_tenant_template"

// TemplateReady is set to true once SetupTenantTemplate completes successfully.
var TemplateReady bool

// templateCreds holds the master DB credentials used by template and clone operations.
// Populated by InitTemplateCreds, called from main.go after config is loaded.
var templateCreds struct {
	host, port, user, password, sslMode string
}

// InitTemplateCreds stores the master DB credentials for template/clone operations.
// Must be called before SetupTenantTemplate or CloneTenantDatabase.
func InitTemplateCreds(host, port, user, password, sslMode string) {
	templateCreds.host = host
	templateCreds.port = port
	templateCreds.user = user
	templateCreds.password = password
	templateCreds.sslMode = sslMode
}

// connectToPostgresDB opens a short-lived connection to the postgres maintenance database.
func connectToPostgresDB() (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=%s",
		templateCreds.host, templateCreds.port,
		templateCreds.user, templateCreds.password,
		templateCreds.sslMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres database: %w", err)
	}
	db.SetConnMaxLifetime(30 * time.Second)
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}
	return db, nil
}

// terminateDBConnections kills all active connections to the given database.
func terminateDBConnections(db *sql.DB, dbName string) {
	_, err := db.Exec(
		"SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = $1 AND pid <> pg_backend_pid()",
		dbName,
	)
	if err != nil {
		log.Printf("[Migration] Warning: failed to terminate connections to %s: %v", dbName, err)
	}
}

// ConnectToNamedDB opens a raw sql.DB to the given database using template credentials.
func ConnectToNamedDB(dbName string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		templateCreds.host, templateCreds.port,
		templateCreds.user, templateCreds.password,
		dbName, templateCreds.sslMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(5)
	db.SetConnMaxLifetime(30 * time.Second)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

// CloneTenantDatabase creates a new database by cloning the golden template.
// Returns (true, nil) if created, (false, nil) if it already existed.
func CloneTenantDatabase(databaseName string) (bool, error) {
	if !TemplateReady {
		return false, fmt.Errorf("tenant template database is not ready")
	}
	if !IsValidDatabaseName(databaseName) {
		return false, fmt.Errorf("invalid database name: %s", databaseName)
	}

	db, err := connectToPostgresDB()
	if err != nil {
		return false, err
	}
	defer db.Close()

	var exists bool
	if err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check if database exists: %w", err)
	}
	if exists {
		log.Printf("[Migration] Database %s already exists, skipping clone", databaseName)
		return false, nil
	}

	terminateDBConnections(db, TemplateDBName)

	createQuery := fmt.Sprintf("CREATE DATABASE %s WITH TEMPLATE %s ENCODING 'UTF8'", databaseName, TemplateDBName)
	if _, err := db.Exec(createQuery); err != nil {
		return false, fmt.Errorf("failed to clone database from template: %w", err)
	}

	log.Printf("[Migration] Cloned database %s from template %s", databaseName, TemplateDBName)
	return true, nil
}

// ConnectToTenantDB opens a raw SQL connection to the named tenant database.
func ConnectToTenantDB(host, port, user, password, databaseName string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, databaseName,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection to %s: %w", databaseName, err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database %s: %w", databaseName, err)
	}

	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(20)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}

// CreateDatabase creates a new PostgreSQL database for a tenant.
// Returns (true, nil) if created, (false, nil) if it already existed.
func CreateDatabase(host, port, user, password, databaseName string) (bool, error) {
	if !IsValidDatabaseName(databaseName) {
		return false, fmt.Errorf("invalid database name: %s", databaseName)
	}

	adminDSN := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=postgres sslmode=disable",
		host, port, user, password,
	)
	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return false, fmt.Errorf("failed to open admin connection: %w", err)
	}
	defer adminDB.Close()
	adminDB.SetConnMaxLifetime(30 * time.Second)
	adminDB.SetMaxOpenConns(1)

	if err := adminDB.Ping(); err != nil {
		return false, fmt.Errorf("failed to ping admin database: %w", err)
	}

	var exists bool
	if err := adminDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", databaseName).Scan(&exists); err != nil {
		return false, fmt.Errorf("failed to check if database exists: %w", err)
	}

	if exists {
		log.Printf("[Migration] Database %s already exists", databaseName)
		return false, nil
	}

	if _, err := adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8'", databaseName)); err != nil {
		return false, fmt.Errorf("failed to create database %s: %w", databaseName, err)
	}

	log.Printf("[Migration] Created database: %s", databaseName)
	return true, nil
}

// IsValidDatabaseName validates that a name is safe for use in a CREATE DATABASE statement.
func IsValidDatabaseName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	first := rune(name[0])
	if !((first >= 'a' && first <= 'z') || first == '_') {
		return false
	}
	for _, c := range name {
		if !((c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

// GenerateTenantDBName produces a deterministic database name from a tenant UUID string.
func GenerateTenantDBName(tenantID string) string {
	clean := ""
	for _, c := range tenantID {
		if c == '-' {
			clean += "_"
		} else {
			clean += string(c)
		}
	}
	return fmt.Sprintf("tenant_%s", clean)
}

// AutoMigrateMigrationLogs ensures the migration_logs table exists in the master DB.
func AutoMigrateMigrationLogs(gormDB *gorm.DB) error {
	if gormDB == nil {
		return fmt.Errorf("GORM DB not initialized")
	}
	return gormDB.AutoMigrate(&MigrationLog{})
}

// RunTenantMigrationsInProcess runs tenant migrations directly in-process without an HTTP round-trip.
// masterDB is the raw master *sql.DB used for migration_logs tracking.
// migrationsDir is the path to the tenant SQL migration files; pass "" to use the default resolved path.
func RunTenantMigrationsInProcess(tenantID, host, port, user, password, dbName string, masterDB *sql.DB, migrationsDir string) error {
	if migrationsDir == "" {
		migrationsDir = MigrationsDir("tenant")
	}

	tenantDBConn, err := ConnectToTenantDB(host, port, user, password, dbName)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database %s: %w", dbName, err)
	}
	defer tenantDBConn.Close()

	runner := NewTenantMigrationRunner(tenantID, tenantDBConn, migrationsDir, masterDB)
	return runner.RunMigrations()
}
