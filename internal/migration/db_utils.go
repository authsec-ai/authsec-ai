package migration

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/gorm"
)

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
