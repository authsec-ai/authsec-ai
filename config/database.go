package config

import (
	"fmt"
	"log"
	"os"

	"github.com/authsec-ai/authsec/database"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Database connection (raw SQL)
var Database *database.DBConnection

// DB is the GORM instance for controllers (migrations disabled)
var DB *gorm.DB

// InitDatabaseWithoutGORM initializes the database connection using the native SQL driver.
// Migrations are NOT run here; call RunStartupMigrations (in main.go) separately.
func InitDatabaseWithoutGORM(cfg *Config) {
	if os.Getenv("SKIP_DB_INIT") == "true" {
		log.Println("Skipping database initialization (SKIP_DB_INIT=true)")
		return
	}

	var err error

	Database, err = database.InitializeDatabase(
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		cfg.DBName,
		cfg.DBPort,
	)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	log.Println("Database connected successfully")

	// Initialize GORM instance for controllers (auto-migration disabled)
	sslMode := cfg.DBSSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, sslMode)

	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:               logger.Default.LogMode(logger.Silent),
		DisableAutomaticPing: false,
	})
	if err != nil {
		log.Fatalf("Failed to initialize GORM: %v", err)
	}

	log.Println("GORM initialized for controllers (AutoMigrate disabled)")
}

// GetDatabase returns the current raw database connection.
func GetDatabase() *database.DBConnection {
	return Database
}

// GetTenantDatabase returns a raw DB connection for the given tenant.
func GetTenantDatabase(tenantID string) (*database.DBConnection, error) {
	return database.GetTenantDB(tenantID)
}

// GetTenantGORMDB returns a GORM connection for a specific tenant.
func GetTenantGORMDB(tenantID string) (*gorm.DB, error) {
	conn, err := database.GetTenantDB(tenantID)
	if err != nil {
		return nil, err
	}

	return gorm.Open(postgres.New(postgres.Config{
		Conn: conn.DB,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
}
