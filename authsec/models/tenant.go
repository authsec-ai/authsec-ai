package models

import (
	"fmt"
	"log"
	"strings"

	// "github.com/authsec-ai/authsec/services" // Temporarily commented out to resolve circular import
	sharedmodels "github.com/authsec-ai/sharedmodels"
	"gorm.io/gorm"
)

// TenantWithHooks extends the sharedmodels.Tenant with GORM hooks
type TenantWithHooks struct {
	sharedmodels.Tenant
}

// AfterCreate hook - automatically creates tenant database after tenant creation
// Temporarily disabled to resolve circular import - move to controller layer
/*
func (t *TenantWithHooks) AfterCreate(tx *gorm.DB) error {
	// Skip if TenantDB is already set (e.g., during migrations)
	if t.TenantDB != "" {
		log.Printf("Tenant %s already has database: %s, skipping creation", t.ID, t.TenantDB)
		return nil
	}

	// Generate tenant database name from tenant ID
	tenantID := t.ID.String()
	if t.TenantID != (uuid.UUID{}) {
		tenantID = t.TenantID.String()
	}

	log.Printf("Creating tenant database for tenant: %s", tenantID)

	// Create database service instance
	// dbService, err := services.NewTenantDBService(tx) // Temporarily disabled
	if err != nil {
		log.Printf("Failed to create tenant database service: %v", err)
		return fmt.Errorf("failed to create tenant database service: %w", err)
	}
	defer dbService.Close()

	// Create the tenant database
	dbName, err := dbService.CreateTenantDatabase(tenantID)
	if err != nil {
		log.Printf("Failed to create tenant database for %s: %v", tenantID, err)
		return fmt.Errorf("failed to create tenant database: %w", err)
	}

	// Update the tenant record with the database name
	if err := tx.Model(t).Update("tenant_db", dbName).Error; err != nil {
		log.Printf("Tenant database %s created, but update failed; manual cleanup may be required: %v", dbName, err)
		return fmt.Errorf("failed to update tenant_db field: %w", err)
	}

	log.Printf("Successfully created tenant database: %s for tenant: %s", dbName, tenantID)
	return nil
}
*/

// BeforeDelete hook - cleanup tenant database before tenant deletion
// BeforeDelete hook - cleanup tenant database before tenant deletion
func (t *TenantWithHooks) BeforeDelete(tx *gorm.DB) error {
	// Temporarily disabled to resolve circular import
	// TODO: Move this logic to controller layer
	log.Printf("Tenant database cleanup disabled due to circular import")
	return nil
}

// TableName ensures the model uses the correct table name
func (TenantWithHooks) TableName() string {
	return "tenants"
}

// Helper function to generate tenant database name
func generateTenantDBName(tenantID string) string {
	// Replace hyphens with underscores for valid database name
	cleanID := strings.ReplaceAll(tenantID, "-", "_")
	return fmt.Sprintf("tenant_%s", cleanID)
}
