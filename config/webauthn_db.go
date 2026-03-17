package config

import (
	"fmt"
	"os"

	sharedmodels "github.com/authsec-ai/sharedmodels"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Global DB connection
func ConnectGlobalDB() (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SCHEMA"),
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

// Get tenant's DB name from global DB using tenant_id
func GetTenantDBName(globalDB *gorm.DB, tenantID string) (string, error) {
	var tenant sharedmodels.Tenant
	if err := globalDB.Where("tenant_id = ?", tenantID).First(&tenant).Error; err != nil {
		return "", err
	}
	return tenant.TenantDB, nil
}

// Connect to tenant's DB by db name
func ConnectTenantDB(tenantDB string) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable search_path=%s",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		tenantDB,
		os.Getenv("DB_PORT"),
		os.Getenv("DB_SCHEMA"),
	)
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}
