package admin

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/internal/migration"
	"github.com/gin-gonic/gin"
)

// MigrationController handles HTTP endpoints for database migration management.
type MigrationController struct {
	masterMigrationsDir string
	tenantMigrationsDir string
}

// NewMigrationController creates a MigrationController using the canonical migration directories.
func NewMigrationController() *MigrationController {
	return &MigrationController{
		masterMigrationsDir: migration.MigrationsDir("master"),
		tenantMigrationsDir: migration.MigrationsDir("tenant"),
	}
}

// RunMasterMigrations POST /authsec-migration/migrations/master/run
func (mc *MigrationController) RunMasterMigrations(c *gin.Context) {
	log.Println("[MigrationController] Running master database migrations")

	rawDB := config.Database.DB
	runner := migration.NewMasterMigrationRunner(mc.masterMigrationsDir, rawDB, config.DB)
	if err := runner.RunMigrations(); err != nil {
		log.Printf("[MigrationController] Master migration error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute master migrations",
			"details": err.Error(),
		})
		return
	}

	status, err := runner.GetMigrationStatus()
	if err != nil || status == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Master migrations executed but status unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Master migrations executed successfully",
		"status":  status,
	})
}

// GetMasterMigrationStatus GET /authsec-migration/migrations/master/status
func (mc *MigrationController) GetMasterMigrationStatus(c *gin.Context) {
	rawDB := config.Database.DB
	runner := migration.NewMasterMigrationRunner(mc.masterMigrationsDir, rawDB, config.DB)
	status, err := runner.GetMigrationStatus()
	if err != nil {
		log.Printf("[MigrationController] Failed to get master migration status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get migration status"})
		return
	}
	c.JSON(http.StatusOK, status)
}

// CreateTenantDB POST /authsec-migration/tenants/create-db
func (mc *MigrationController) CreateTenantDB(c *gin.Context) {
	var req migration.CreateTenantDBRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload", "details": err.Error()})
		return
	}

	log.Printf("[MigrationController] Creating tenant DB for tenant: %s", req.TenantID)

	var tenant migration.TenantInfo
	if err := config.DB.Where("tenant_id = ?", req.TenantID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Tenant not found",
			"details": fmt.Sprintf("No tenant record found for: %s", req.TenantID),
		})
		return
	}

	dbName := req.DatabaseName
	if dbName == "" && tenant.TenantDB != nil && *tenant.TenantDB != "" {
		dbName = *tenant.TenantDB
	}
	if dbName == "" {
		dbName = migration.GenerateTenantDBName(req.TenantID)
	}

	if !migration.IsValidDatabaseName(dbName) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid database name format"})
		return
	}

	if tenant.MigrationStatus != nil && *tenant.MigrationStatus == "completed" {
		createdAt := time.Time{}
		if tenant.CreatedAt != nil {
			createdAt = *tenant.CreatedAt
		}
		c.JSON(http.StatusOK, migration.CreateTenantDBResponse{
			TenantID:        tenant.TenantID.String(),
			DatabaseName:    dbName,
			MigrationStatus: "completed",
			CreatedAt:       createdAt,
			Existed:         true,
		})
		return
	}

	cfg := config.AppConfig
	created, err := migration.CreateDatabase(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName)
	if err != nil {
		log.Printf("[MigrationController] Failed to create database %s: %v", dbName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create database", "details": err.Error()})
		return
	}

	config.DB.Model(&tenant).Updates(map[string]interface{}{
		"tenant_db":        dbName,
		"migration_status": "pending",
	})

	go mc.runTenantMigrationsAsync(tenant.TenantID.String(), dbName)

	createdAt := time.Time{}
	if tenant.CreatedAt != nil {
		createdAt = *tenant.CreatedAt
	}

	log.Printf("[MigrationController] Tenant DB setup initiated: %s (created=%v)", dbName, created)
	c.JSON(http.StatusCreated, migration.CreateTenantDBResponse{
		TenantID:        tenant.TenantID.String(),
		DatabaseName:    dbName,
		MigrationStatus: "pending",
		CreatedAt:       createdAt,
		Existed:         !created,
	})
}

// RunTenantMigrations POST /authsec-migration/tenants/:tenant_id/migrations/run
func (mc *MigrationController) RunTenantMigrations(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	log.Printf("[MigrationController] Running tenant migrations for: %s", tenantID)

	var tenant migration.TenantInfo
	if err := config.DB.Where("tenant_id = ?", tenantID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found", "details": fmt.Sprintf("No record for tenant: %s", tenantID)})
		return
	}

	dbName := ""
	if tenant.TenantDB != nil && *tenant.TenantDB != "" {
		dbName = *tenant.TenantDB
	} else {
		dbName = migration.GenerateTenantDBName(tenantID)
	}

	cfg := config.AppConfig
	if _, err := migration.CreateDatabase(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to ensure database exists", "details": err.Error()})
		return
	}

	config.DB.Model(&tenant).Updates(map[string]interface{}{
		"tenant_db":        dbName,
		"migration_status": "in_progress",
	})

	tenantDBConn, err := migration.ConnectToTenantDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName)
	if err != nil {
		config.DB.Model(&tenant).Update("migration_status", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to connect to tenant database", "details": err.Error()})
		return
	}
	defer tenantDBConn.Close()

	masterRaw := config.Database.DB
	runner := migration.NewTenantMigrationRunner(tenantID, tenantDBConn, mc.tenantMigrationsDir, masterRaw)
	if err := runner.RunMigrations(); err != nil {
		config.DB.Model(&tenant).Update("migration_status", "failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute tenant migrations", "details": err.Error()})
		return
	}

	status, err := runner.GetMigrationStatus()
	if err != nil || status == nil {
		config.DB.Model(&tenant).Update("migration_status", "completed")
		c.JSON(http.StatusOK, gin.H{"message": "Tenant migrations executed but status unavailable"})
		return
	}

	config.DB.Model(&tenant).Updates(map[string]interface{}{
		"migration_status": "completed",
		"last_migration":   status.LastMigration,
		"updated_at":       time.Now().UTC(),
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Tenant migrations executed successfully",
		"status":  status,
	})
}

// GetTenantMigrationStatus GET /authsec-migration/tenants/:tenant_id/migrations/status
func (mc *MigrationController) GetTenantMigrationStatus(c *gin.Context) {
	tenantID := c.Param("tenant_id")

	var tenant migration.TenantInfo
	if err := config.DB.Where("tenant_id = ?", tenantID).First(&tenant).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	dbName := ""
	if tenant.TenantDB != nil {
		dbName = *tenant.TenantDB
	}

	migStatus := "pending"
	if tenant.MigrationStatus != nil {
		migStatus = *tenant.MigrationStatus
	}

	if dbName == "" {
		c.JSON(http.StatusOK, gin.H{
			"tenant_id":        tenant.TenantID.String(),
			"migration_status": migStatus,
			"last_migration":   tenant.LastMigration,
		})
		return
	}

	cfg := config.AppConfig
	tenantDBConn, err := migration.ConnectToTenantDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"tenant_id":        tenant.TenantID.String(),
			"database_name":    dbName,
			"migration_status": migStatus,
			"last_migration":   tenant.LastMigration,
			"error":            "Unable to connect to tenant database",
		})
		return
	}
	defer tenantDBConn.Close()

	masterRaw := config.Database.DB
	runner := migration.NewTenantMigrationRunner(tenantID, tenantDBConn, mc.tenantMigrationsDir, masterRaw)
	status, err := runner.GetMigrationStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get migration status"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":        tenant.TenantID.String(),
		"database_name":    dbName,
		"migration_status": migStatus,
		"status":           status,
	})
}

// MigrateAllTenants POST /authsec-migration/tenants/migrate-all
func (mc *MigrationController) MigrateAllTenants(c *gin.Context) {
	log.Println("[MigrationController] Migrate all tenants")

	var tenants []migration.TenantInfo
	if err := config.DB.Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read tenants", "details": err.Error()})
		return
	}

	response := migration.MigrateAllResponse{
		Total:   len(tenants),
		Results: make([]migration.TenantMigrateResult, 0, len(tenants)),
	}

	cfg := config.AppConfig
	masterRaw := config.Database.DB

	for _, tenant := range tenants {
		tenantID := tenant.TenantID.String()
		result := migration.TenantMigrateResult{TenantID: tenantID}

		if tenant.MigrationStatus != nil && *tenant.MigrationStatus == "completed" {
			if tenant.TenantDB != nil {
				result.DatabaseName = *tenant.TenantDB
			}
			result.Status = "skipped"
			response.Skipped++
			response.Results = append(response.Results, result)
			continue
		}

		dbName := ""
		if tenant.TenantDB != nil && *tenant.TenantDB != "" {
			dbName = *tenant.TenantDB
		} else {
			dbName = migration.GenerateTenantDBName(tenantID)
		}
		result.DatabaseName = dbName

		if _, err := migration.CreateDatabase(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName); err != nil {
			result.Status = "failed"
			result.Error = fmt.Sprintf("create db: %v", err)
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		config.DB.Model(&tenant).Updates(map[string]interface{}{
			"tenant_db":        dbName,
			"migration_status": "in_progress",
		})

		tenantDBConn, err := migration.ConnectToTenantDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName)
		if err != nil {
			config.DB.Model(&tenant).Update("migration_status", "failed")
			result.Status = "failed"
			result.Error = fmt.Sprintf("connect: %v", err)
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		runner := migration.NewTenantMigrationRunner(tenantID, tenantDBConn, mc.tenantMigrationsDir, masterRaw)
		err = runner.RunMigrations()
		tenantDBConn.Close()

		if err != nil {
			config.DB.Model(&tenant).Update("migration_status", "failed")
			result.Status = "failed"
			result.Error = fmt.Sprintf("migrations: %v", err)
			response.Failed++
			response.Results = append(response.Results, result)
			continue
		}

		status, _ := runner.GetMigrationStatus()
		if status != nil {
			config.DB.Model(&tenant).Updates(map[string]interface{}{
				"migration_status": "completed",
				"last_migration":   status.LastMigration,
				"updated_at":       time.Now().UTC(),
			})
		} else {
			config.DB.Model(&tenant).Update("migration_status", "completed")
		}

		result.Status = "completed"
		response.Succeeded++
		response.Results = append(response.Results, result)
	}

	log.Printf("[MigrationController] Migrate all: %d succeeded, %d failed, %d skipped of %d",
		response.Succeeded, response.Failed, response.Skipped, response.Total)
	c.JSON(http.StatusOK, response)
}

// ListTenants GET /authsec-migration/tenants
func (mc *MigrationController) ListTenants(c *gin.Context) {
	var tenants []migration.TenantInfo
	if err := config.DB.Find(&tenants).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read tenants", "details": err.Error()})
		return
	}

	items := make([]migration.TenantListItem, 0, len(tenants))
	for _, t := range tenants {
		item := migration.TenantListItem{
			TenantID:      t.TenantID.String(),
			Email:         t.Email,
			TenantDomain:  t.TenantDomain,
			LastMigration: t.LastMigration,
		}
		if t.TenantDB != nil {
			item.DatabaseName = *t.TenantDB
		}
		if t.MigrationStatus != nil {
			item.MigrationStatus = *t.MigrationStatus
		} else {
			item.MigrationStatus = "pending"
		}
		items = append(items, item)
	}

	c.JSON(http.StatusOK, gin.H{"total": len(items), "tenants": items})
}

// runTenantMigrationsAsync runs tenant migrations in the background.
func (mc *MigrationController) runTenantMigrationsAsync(tenantID, dbName string) {
	log.Printf("[MigrationController] Starting async tenant migrations for: %s", tenantID)

	config.DB.Model(&migration.TenantInfo{}).
		Where("tenant_id = ?", tenantID).
		Update("migration_status", "in_progress")

	cfg := config.AppConfig
	tenantDBConn, err := migration.ConnectToTenantDB(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, dbName)
	if err != nil {
		log.Printf("[MigrationController] Async: failed to connect to %s: %v", dbName, err)
		config.DB.Model(&migration.TenantInfo{}).
			Where("tenant_id = ?", tenantID).
			Update("migration_status", "failed")
		return
	}
	defer tenantDBConn.Close()

	masterRaw := config.Database.DB
	runner := migration.NewTenantMigrationRunner(tenantID, tenantDBConn, mc.tenantMigrationsDir, masterRaw)
	if err := runner.RunMigrations(); err != nil {
		log.Printf("[MigrationController] Async tenant migration failed for %s: %v", tenantID, err)
		config.DB.Model(&migration.TenantInfo{}).
			Where("tenant_id = ?", tenantID).
			Update("migration_status", "failed")
		return
	}

	status, _ := runner.GetMigrationStatus()
	if status != nil {
		config.DB.Model(&migration.TenantInfo{}).
			Where("tenant_id = ?", tenantID).
			Updates(map[string]interface{}{
				"migration_status": "completed",
				"last_migration":   status.LastMigration,
				"updated_at":       time.Now().UTC(),
			})
	} else {
		config.DB.Model(&migration.TenantInfo{}).
			Where("tenant_id = ?", tenantID).
			Update("migration_status", "completed")
	}

	log.Printf("[MigrationController] Async tenant migration completed for: %s", tenantID)
}

