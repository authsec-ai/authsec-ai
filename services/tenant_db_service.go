package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/authsec-ai/authsec/config"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TenantDBService handles tenant database operations
type TenantDBService struct {
	masterDB *gorm.DB
	adminDB  *sql.DB
}

// NewTenantDBService creates a new tenant database service instance
func NewTenantDBService(masterDB *gorm.DB) (*TenantDBService, error) {
	// Create admin database connection for CREATE DATABASE operations
	adminDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBPort,
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
		masterDB: masterDB,
		adminDB:  adminDB,
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
	} else if exists {
		log.Printf("Database %s already exists, skipping creation", dbName)
		return dbName, nil
	}

	// Create the database
	if err := s.createDatabase(dbName); err != nil {
		return "", fmt.Errorf("failed to create database: %w", err)
	}

	// Apply tenant schema template
	if err := s.applyTenantSchema(dbName); err != nil {
		log.Printf("tenant database %s may require manual cleanup after schema error: %v", dbName, err)
		return "", fmt.Errorf("failed to apply tenant schema: %w", err)
	}

	log.Printf("Successfully created tenant database: %s", dbName)
	return dbName, nil
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

// applyTenantSchema applies the tenant schema template to a new database
func (s *TenantDBService) applyTenantSchema(dbName string) error {
	// Connect to the new tenant database
	tenantDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		dbName,
		config.AppConfig.DBPort,
	)

	tenantDB, err := gorm.Open(postgres.Open(tenantDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database %s: %w", dbName, err)
	}

	// Get the raw database connection for executing SQL
	sqlDB, err := tenantDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw database connection: %w", err)
	}
	defer sqlDB.Close()

	schemaSQL, source, err := resolveTenantSchemaSQL()
	if err != nil {
		return err
	}

	log.Printf("Applying tenant schema from %s", source)

	// Execute the schema SQL
	if _, err := sqlDB.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed to execute tenant schema: %w", err)
	}

	log.Printf("Successfully applied tenant schema to database: %s", dbName)
	return nil
}

func resolveTenantSchemaSQL() (string, string, error) {
	if cached := strings.TrimSpace(config.MasterSchemaSQL()); cached != "" {
		return cached, "runtime master cache", nil
	}

	paths := []string{
		config.RuntimeMasterSchemaPath(),
		filepath.Join("schema", "generated_tenant_template.sql"),
		filepath.Join("templates", "tenant_schema_template.sql"),
	}

	for _, candidate := range paths {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(data)) == "" {
			continue
		}

		return string(data), candidate, nil
	}

	return "", "", fmt.Errorf("no tenant schema SQL available (checked %s)", strings.Join(paths, ", "))
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
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		dbName,
		config.AppConfig.DBPort,
	)

	tenantDB, err := gorm.Open(postgres.Open(tenantDSN), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to tenant database %s: %w", dbName, err)
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get raw database connection: %w", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("failed to ping tenant database %s: %w", dbName, err)
	}

	return nil
}
