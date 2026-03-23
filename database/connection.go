package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// DBConnection wraps a SQL database connection
type DBConnection struct {
	DB *sql.DB
}

// ConnectionManager manages database connections
type ConnectionManager struct {
	masterDB          *DBConnection
	tenantConnections map[string]*DBConnection
}

var GlobalConnectionManager *ConnectionManager

// InitializeDatabase initializes the master database connection without GORM
func InitializeDatabase(dbHost, dbUser, dbPassword, dbName, dbPort string) (*DBConnection, error) {
	// First, try to connect to postgres database to create the target database if needed
	sslMode := getEnvOrDefault("DB_SSL_MODE", "disable")
	adminDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=%s TimeZone=UTC",
		dbHost, dbUser, dbPassword, dbPort, sslMode)

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres database for setup: %w", err)
	}
	defer adminDB.Close()

	// Test admin connection
	if err := adminDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping postgres database: %w", err)
	}

	// Create the target database if it doesn't exist
	_, err = adminDB.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		// Ignore error if database already exists
		if !strings.Contains(err.Error(), "already exists") {
			return nil, fmt.Errorf("failed to create database %s: %w", dbName, err)
		}
	}

	// Now connect to the target database
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC search_path=public",
		dbHost, dbUser, dbPassword, dbName, dbPort, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Configure connection pool
	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	connection := &DBConnection{DB: db}

	// Initialize global connection manager
	GlobalConnectionManager = &ConnectionManager{
		masterDB:          connection,
		tenantConnections: make(map[string]*DBConnection),
	}

	return connection, nil
}

// GetMasterDB returns the master database connection
func GetMasterDB() *DBConnection {
	if GlobalConnectionManager == nil {
		return nil
	}
	return GlobalConnectionManager.masterDB
}

// GetTenantDB gets or creates a tenant database connection
func GetTenantDB(tenantID string) (*DBConnection, error) {
	if GlobalConnectionManager == nil {
		return nil, fmt.Errorf("connection manager not initialized")
	}

	// Check if we already have a connection
	if conn, exists := GlobalConnectionManager.tenantConnections[tenantID]; exists {
		// Test if connection is still alive
		if err := conn.DB.Ping(); err == nil {
			return conn, nil
		}
		// Connection is dead, remove it
		delete(GlobalConnectionManager.tenantConnections, tenantID)
	}

	// Get tenant database name from master database
	masterDB := GlobalConnectionManager.masterDB

	var tenantDBName sql.NullString
	query := "SELECT tenant_db FROM tenants WHERE tenant_id::text = $1 OR id::text = $1"
	err := masterDB.DB.QueryRow(query, tenantID).Scan(&tenantDBName)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", tenantID)
		}
		return nil, fmt.Errorf("failed to query tenant: %w", err)
	}

	if !tenantDBName.Valid || tenantDBName.String == "" {
		return nil, fmt.Errorf("tenant database not configured for tenant %s", tenantID)
	}

	// Create connection to tenant database
	tenantConn, err := createTenantConnection(tenantDBName.String)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database '%s': %w", tenantDBName.String, err)
	}

	// Cache the connection
	GlobalConnectionManager.tenantConnections[tenantID] = tenantConn

	return tenantConn, nil
}

// createTenantConnection creates a connection to a specific tenant database
func createTenantConnection(dbName string) (*DBConnection, error) {
	// Get connection details from master connection
	// masterDB := GlobalConnectionManager.masterDB

	// Get database configuration (this should come from config)
	// For now, we'll use environment variables or default values
	dbHost := getEnvOrDefault("DB_HOST", "localhost")
	dbUser := getEnvOrDefault("DB_USER", "asiffinal")
	dbPassword := getEnvOrDefault("DB_PASSWORD", "test1")
	dbPort := getEnvOrDefault("DB_PORT", "5432")
	sslMode := getEnvOrDefault("DB_SSL_MODE", "disable")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC search_path=public",
		dbHost, dbUser, dbPassword, dbName, dbPort, sslMode)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database: %w", err)
	}

	// Configure connection pool
	db.SetMaxIdleConns(5)
	db.SetMaxOpenConns(50)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(10 * time.Minute)

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping tenant database: %w", err)
	}

	return &DBConnection{DB: db}, nil
}

// Close closes the database connection
func (conn *DBConnection) Close() error {
	if conn.DB != nil {
		return conn.DB.Close()
	}
	return nil
}

// CloseAll closes all database connections
func (cm *ConnectionManager) CloseAll() error {
	var lastErr error

	// Close tenant connections
	for _, conn := range cm.tenantConnections {
		if err := conn.Close(); err != nil {
			lastErr = err
		}
	}

	// Close master connection
	if cm.masterDB != nil {
		if err := cm.masterDB.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Helper functions

// QueryRow executes a query that returns a single row
func (conn *DBConnection) QueryRow(query string, args ...interface{}) *sql.Row {
	return conn.DB.QueryRow(query, args...)
}

// Query executes a query that returns multiple rows
func (conn *DBConnection) Query(query string, args ...interface{}) (*sql.Rows, error) {
	return conn.DB.Query(query, args...)
}

// Exec executes a query that doesn't return rows
func (conn *DBConnection) Exec(query string, args ...interface{}) (sql.Result, error) {
	return conn.DB.Exec(query, args...)
}

// Begin starts a transaction
func (conn *DBConnection) Begin() (*sql.Tx, error) {
	return conn.DB.Begin()
}

// getEnvOrDefault gets environment variable with fallback
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
