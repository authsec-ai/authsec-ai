package middlewares

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/authsec-ai/authsec/config"
	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// ConnectToTenantDB connects to tenant DB using either email or tenant ID (deprecated - use GetConnectionDynamically)
func ConnectToTenantDB(masterDB interface{}, userEmail *string, tenantID *string) (*gorm.DB, error) {
	// Delegate to the new function for consistency
	return GetConnectionDynamically(masterDB, userEmail, tenantID)
}

// Helper function to safely close tenant DB connection
func CloseTenantDB(tenantDB *gorm.DB) error {
	if tenantDB == nil {
		return nil
	}

	sqlDB, err := tenantDB.DB()
	if err != nil {
		return err
	}

	return sqlDB.Close()
}

// Connection pool manager for tenant databases
type TenantDBManager struct {
	connections map[string]*gorm.DB
}

var dbManager = &TenantDBManager{
	connections: make(map[string]*gorm.DB),
}

// GetOrCreateTenantDB gets existing connection or creates new one using email or tenant ID
func GetConnectionDynamically(masterDB interface{}, userEmail *string, tenantID *string) (*gorm.DB, error) {
	if userEmail == nil && tenantID == nil {
		return nil, fmt.Errorf("either userEmail or tenantID must be provided")
	}

	if userEmail != nil && tenantID != nil {
		return nil, fmt.Errorf("provide either userEmail or tenantID, not both")
	}

	// Get the database connection - use config.GetDatabase() instead
	dbConn := config.GetDatabase()
	if dbConn == nil {
		return nil, fmt.Errorf("database connection not initialized")
	}
	db := dbConn.DB

	var tenantDBName sql.NullString
	var err error

	// Query based on the provided parameter
	if userEmail != nil {
		err = db.QueryRow("SELECT tenant_db FROM tenants WHERE email = $1", *userEmail).Scan(&tenantDBName)
	} else {
		err = db.QueryRow("SELECT tenant_db FROM tenants WHERE tenant_id = $1", *tenantID).Scan(&tenantDBName)
	}

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found")
		}
		return nil, fmt.Errorf("tenant not found: %w", err)
	}

	if !tenantDBName.Valid || tenantDBName.String == "" {
		var tenantUUID string
		if userEmail != nil {
			db.QueryRow("SELECT id FROM tenants WHERE email = $1", *userEmail).Scan(&tenantUUID)
		} else {
			tenantUUID = *tenantID
		}
		return nil, fmt.Errorf("tenant database not configured for tenant %s", tenantUUID)
	}

	dbName := tenantDBName.String

	// ✅ Check if database exists before attempting connection
	if exists, err := databaseExists(dbName); err != nil {
		return nil, fmt.Errorf("failed to check if tenant database exists: %w", err)
	} else if !exists {
		return nil, fmt.Errorf("tenant database '%s' does not exist", dbName)
	}

	// Check if we already have a connection
	if gormDB, exists := dbManager.connections[dbName]; exists {
		// Test if connection is still alive
		if sqlDB, err := gormDB.DB(); err == nil {
			if err := sqlDB.Ping(); err == nil {
				return gormDB, nil
			}
		}
		// Connection is dead, remove it
		delete(dbManager.connections, dbName)
	}

	// Create new GORM connection for compatibility
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.AppConfig.DBHost, config.AppConfig.DBUser, config.AppConfig.DBPassword, dbName, config.AppConfig.DBPort,
	)

	tenantDB, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to tenant database '%s': %w", dbName, err)
	}

	// Configure connection pool settings
	sqlDB, err := tenantDB.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get raw DB: %w", err)
	}

	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	if err := sqlDB.Ping(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("failed to ping tenant database '%s': %w", dbName, err)
	}

	// Cache the connection
	dbManager.connections[dbName] = tenantDB

	return tenantDB, nil
}

// ✅ Helper function to check if database exists
func databaseExists(dbName string) (bool, error) {
	// Create connection to postgres database to check if target database exists
	adminDSN := fmt.Sprintf("host=%s user=%s password=%s dbname=postgres port=%s sslmode=disable",
		config.AppConfig.DBHost,
		config.AppConfig.DBUser,
		config.AppConfig.DBPassword,
		config.AppConfig.DBPort,
	)

	adminDB, err := sql.Open("postgres", adminDSN)
	if err != nil {
		return false, fmt.Errorf("failed to connect to admin database: %w", err)
	}
	defer adminDB.Close()

	var count int
	query := "SELECT 1 FROM pg_database WHERE datname = $1"
	err = adminDB.QueryRow(query, dbName).Scan(&count)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}
	return count > 0, nil
}
