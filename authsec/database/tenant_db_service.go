package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/authsec-ai/authsec/internal/migration"
	_ "github.com/lib/pq"
)

// TenantDBService handles tenant database operations without GORM
type TenantDBService struct {
	masterDB   *DBConnection
	adminDB    *sql.DB
	dbHost     string
	dbUser     string
	dbPassword string
	dbPort     string
}

// NewTenantDBService creates a new tenant database service instance
func NewTenantDBService(masterDB *DBConnection, dbHost, dbUser, dbPassword, dbPort string) (*TenantDBService, error) {
	// Create admin database connection for CREATE DATABASE operations
	adminDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbPort,
	)

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to admin database: %w", err)
	}

	// Test the connection
	if err := adminDB.Ping(); err != nil {
		adminDB.Close()
		return nil, fmt.Errorf("failed to ping admin database: %w", err)
	}

	return &TenantDBService{
		masterDB:   masterDB,
		adminDB:    adminDB,
		dbHost:     dbHost,
		dbUser:     dbUser,
		dbPassword: dbPassword,
		dbPort:     dbPort,
	}, nil
}

// Close closes the admin database connection
func (s *TenantDBService) Close() error {
	if s.adminDB != nil {
		return s.adminDB.Close()
	}
	return nil
}

// CreateTenantDatabase creates a new tenant database with schema
func (s *TenantDBService) CreateTenantDatabase(tenantID string) (string, error) {
	// Generate database name with tenant_ prefix
	dbName := s.generateTenantDBName(tenantID)

	log.Printf("Creating tenant database: %s for tenant: %s", dbName, tenantID)

	// Check if database already exists
	if exists, err := s.databaseExists(dbName); err != nil {
		return "", fmt.Errorf("failed to check if database exists: %w", err)
	} else if !exists {
		// Create the database
		if err := s.createDatabase(dbName); err != nil {
			return "", fmt.Errorf("failed to create database: %w", err)
		}
	} else {
		log.Printf("Database %s already exists, will run migrations on it", dbName)
	}

	// Always run tenant migrations (idempotent) - handles retry case where DB exists but migrations failed
	if err := s.RunTenantMigrations(tenantID); err != nil {
		log.Printf("Failed to run tenant migrations for %s: %v", dbName, err)
		return "", fmt.Errorf("failed to run tenant migrations: %w", err)
	}

	log.Printf("Successfully created tenant database: %s", dbName)
	return dbName, nil
}

// RunTenantMigrations runs tenant migrations in-process by calling the migration runner directly.
func (s *TenantDBService) RunTenantMigrations(tenantID string) error {
	dbName := s.generateTenantDBName(tenantID)
	log.Printf("Running tenant migrations in-process for tenant %s (db: %s)", tenantID, dbName)
	return migration.RunTenantMigrationsInProcess(tenantID, s.dbHost, s.dbPort, s.dbUser, s.dbPassword, dbName, s.masterDB.DB, "")
}

// generateTenantDBName creates a database name from tenant ID
func (s *TenantDBService) generateTenantDBName(tenantID string) string {
	// Replace hyphens with underscores for valid database name
	cleanID := strings.ReplaceAll(tenantID, "-", "_")
	return fmt.Sprintf("tenant_%s", cleanID)
}

// databaseExists checks if a database exists
func (s *TenantDBService) databaseExists(dbName string) (bool, error) {
	var count int
	query := "SELECT 1 FROM pg_database WHERE datname = $1"
	err := s.adminDB.QueryRow(query, dbName).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}

// createDatabase creates a new PostgreSQL database
func (s *TenantDBService) createDatabase(dbName string) error {
	// Ensure database name is safe (alphanumeric and underscores only)
	if !isValidDatabaseName(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	query := fmt.Sprintf(`CREATE DATABASE "%s" WITH ENCODING 'UTF8'`, dbName)
	_, err := s.adminDB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to execute CREATE DATABASE: %w", err)
	}

	return nil
}

// dropDatabase drops a PostgreSQL database
func (s *TenantDBService) dropDatabase(dbName string) error {
	// Ensure database name is safe
	if !isValidDatabaseName(dbName) {
		return fmt.Errorf("invalid database name: %s", dbName)
	}

	// Terminate active connections to the database
	terminateQuery := fmt.Sprintf(`
		SELECT pg_terminate_backend(pg_stat_activity.pid)
		FROM pg_stat_activity
		WHERE pg_stat_activity.datname = '%s'
		AND pid <> pg_backend_pid()`, dbName)

	if _, err := s.adminDB.Exec(terminateQuery); err != nil {
		log.Printf("Warning: failed to terminate connections to database %s: %v", dbName, err)
	}

	// Drop the database
	query := fmt.Sprintf(`DROP DATABASE IF EXISTS "%s"`, dbName)
	_, err := s.adminDB.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to execute DROP DATABASE: %w", err)
	}

	return nil
}

// applyTenantSchema applies the tenant schema template to a new database using psql
func (s *TenantDBService) applyTenantSchema(dbName string) error {
	schemaFile, source, err := resolveTenantSchemaFiles()
	if err != nil {
		return err
	}

	log.Printf("Applying tenant schema from %s to database %s", source, dbName)

	// Find psql executable
	psqlPath, err := exec.LookPath("psql")
	if err != nil {
		log.Printf("ERROR: psql not found in PATH: %v. PATH=%s", err, os.Getenv("PATH"))
		return fmt.Errorf("psql not found: %w", err)
	}
	log.Printf("Found psql at: %s", psqlPath)

	// Use psql command to apply schema - it properly handles PL/pgSQL functions and complex SQL
	cmd := exec.Command(psqlPath, "-h", s.dbHost, "-p", s.dbPort, "-U", s.dbUser, "-d", dbName, "-f", schemaFile, "-v", "ON_ERROR_STOP=1")
	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", s.dbPassword))

	// Capture output directly for better error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("psql error output: %s", string(output))
		return fmt.Errorf("failed to apply tenant schema using psql: %w", err)
	}

	log.Printf("Successfully applied tenant schema to database: %s", dbName)
	return nil
}

func resolveTenantSchemaFiles() (string, string, error) {
	// Try paths relative to current directory and parent directory (for tests)
	basePaths := []string{".", ".."}
	candidates := []string{
		"schema/runtime_master_schema.sql",
		"schema/generated_tenant_template.sql",
		"templates/tenant_schema_template.sql",
	}

	for _, base := range basePaths {
		for _, candidate := range candidates {
			fullPath := filepath.Join(base, candidate)

			// Check if file exists and is not empty
			info, err := os.Stat(fullPath)
			if err != nil {
				continue
			}
			if info.Size() == 0 {
				continue
			}

			// Return absolute path for reliability
			absPath, err := filepath.Abs(fullPath)
			if err != nil {
				absPath = fullPath
			}
			return absPath, fullPath, nil
		}
	}

	allPaths := make([]string, 0, len(basePaths)*len(candidates))
	for _, base := range basePaths {
		for _, candidate := range candidates {
			allPaths = append(allPaths, filepath.Join(base, candidate))
		}
	}

	return "", "", fmt.Errorf("no tenant schema SQL available (checked %s)", strings.Join(allPaths, ", "))
}

// isValidDatabaseName checks if database name contains only safe characters
func isValidDatabaseName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return false
		}
	}

	// Database name cannot start with a number
	firstChar := name[0]
	return (firstChar >= 'a' && firstChar <= 'z') || (firstChar >= 'A' && firstChar <= 'Z') || firstChar == '_'
}

// HealthCheck verifies that a tenant database is accessible
func (s *TenantDBService) HealthCheck(dbName string) error {
	tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		s.dbHost,
		s.dbUser,
		s.dbPassword,
		dbName,
		s.dbPort,
	)

	tenantDB, err := sql.Open("postgres", tenantDSN)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database %s: %w", dbName, err)
	}
	defer tenantDB.Close()

	if err := tenantDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping tenant database %s: %w", dbName, err)
	}

	return nil
}

// DropTenantDatabase drops a tenant database after terminating all connections.
// This is a destructive operation that permanently removes the database.
func (s *TenantDBService) DropTenantDatabase(dbName string) error {
	// Check if database exists first
	exists, err := s.databaseExists(dbName)
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %w", err)
	}
	if !exists {
		log.Printf("Database %s does not exist, skipping drop", dbName)
		return nil
	}

	log.Printf("Dropping tenant database: %s", dbName)
	return s.dropDatabase(dbName)
}
