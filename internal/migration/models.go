package migration

import (
	"time"

	"github.com/google/uuid"
)

// MigrationLog tracks each migration execution in the master database.
type MigrationLog struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Version     int       `gorm:"not null;index"                                  json:"version"`
	Name        string    `gorm:"type:varchar(255);not null"                      json:"name"`
	ExecutedAt  time.Time `gorm:"not null;default:CURRENT_TIMESTAMP"              json:"executed_at"`
	Success     bool      `gorm:"not null;default:false"                          json:"success"`
	ErrorMsg    string    `gorm:"type:text"                                       json:"error_msg,omitempty"`
	DBType      string    `gorm:"type:varchar(50);not null"                       json:"db_type"`
	TenantID    *string   `gorm:"type:varchar(255);index"                         json:"tenant_id,omitempty"`
	ExecutionMS int64     `gorm:"not null;default:0"                              json:"execution_ms"`
}

func (MigrationLog) TableName() string { return "migration_logs" }

// MigrationStatusResponse is returned by GetMigrationStatus.
type MigrationStatusResponse struct {
	DBType          string    `json:"db_type"`
	TenantID        *string   `json:"tenant_id,omitempty"`
	LastMigration   int       `json:"last_migration"`
	TotalMigrations int       `json:"total_migrations"`
	Status          string    `json:"status"`
	LastExecuted    time.Time `json:"last_executed"`
}

// CreateTenantDBRequest is the payload for the create-tenant-db endpoint.
type CreateTenantDBRequest struct {
	TenantID     string `json:"tenant_id"     binding:"required"`
	DatabaseName string `json:"database_name,omitempty"`
	TenantDomain string `json:"tenant_domain,omitempty"`
}

// CreateTenantDBResponse is the response for the create-tenant-db endpoint.
type CreateTenantDBResponse struct {
	TenantID        string    `json:"tenant_id"`
	DatabaseName    string    `json:"database_name"`
	MigrationStatus string    `json:"migration_status"`
	CreatedAt       time.Time `json:"created_at"`
	Existed         bool      `json:"existed"`
}

// TenantInfo maps to the tenants table in the master DB.
// Not managed by GORM auto-migrate; the table already exists in production.
type TenantInfo struct {
	ID              uuid.UUID  `gorm:"type:uuid;primary_key"                  json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null;unique"              json:"tenant_id"`
	TenantDB        *string    `gorm:"type:text"                              json:"tenant_db"`
	Email           string     `gorm:"type:text;not null"                     json:"email"`
	TenantDomain    string     `gorm:"type:text;not null"                     json:"tenant_domain"`
	Status          *string    `gorm:"type:text"                              json:"status"`
	MigrationStatus *string    `gorm:"type:varchar(50);default:'pending'"     json:"migration_status"`
	LastMigration   *int       `gorm:"type:integer"                           json:"last_migration"`
	CreatedAt       *time.Time `gorm:"type:timestamptz"                       json:"created_at"`
	UpdatedAt       *time.Time `gorm:"type:timestamptz"                       json:"updated_at"`
}

func (TenantInfo) TableName() string { return "tenants" }

// TenantListItem is a lightweight tenant representation for list responses.
type TenantListItem struct {
	TenantID        string `json:"tenant_id"`
	Email           string `json:"email"`
	TenantDomain    string `json:"tenant_domain"`
	DatabaseName    string `json:"database_name"`
	MigrationStatus string `json:"migration_status"`
	LastMigration   *int   `json:"last_migration"`
}

// MigrateAllResponse summarises a bulk tenant migration run.
type MigrateAllResponse struct {
	Total     int                   `json:"total"`
	Succeeded int                   `json:"succeeded"`
	Failed    int                   `json:"failed"`
	Skipped   int                   `json:"skipped"`
	Results   []TenantMigrateResult `json:"results"`
}

// TenantMigrateResult is the per-tenant outcome within a MigrateAllResponse.
type TenantMigrateResult struct {
	TenantID     string `json:"tenant_id"`
	DatabaseName string `json:"database_name"`
	Status       string `json:"status"`
	Error        string `json:"error,omitempty"`
}
