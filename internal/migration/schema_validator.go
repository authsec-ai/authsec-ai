package migration

import (
	"database/sql"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// SchemaValidator validates database schema against production
type SchemaValidator struct {
	db *sql.DB
}

// NewSchemaValidator creates a new schema validator
func NewSchemaValidator(db *sql.DB) *SchemaValidator {
	return &SchemaValidator{db: db}
}

// ValidateProduction compares current schema with production expectations
func (sv *SchemaValidator) ValidateProduction() error {
	log.Info("Validating database schema against production standards...")

	// Check required tables exist
	requiredTables := []string{
		"users",
		"roles",
		"permissions",
		"role_permissions",
		"role_bindings",
		"tenants",
		"clients",
		"resources",
		"migration_logs",
		"tenant_databases",
	}

	for _, table := range requiredTables {
		exists, err := sv.tableExists(table)
		if err != nil {
			return fmt.Errorf("failed to check table %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("required table missing: %s", table)
		}
		log.Debugf("Table %s exists", table)
	}

	// Check critical columns in users table
	userColumns := map[string]string{
		"id":                    "uuid",
		"email":                 "character varying",
		"password_hash":         "character varying",
		"tenant_id":             "character varying",
		"created_at":            "timestamp",
		"failed_login_attempts": "integer",
		"is_active":             "boolean",
	}

	for column := range userColumns {
		exists, actualType, err := sv.columnExists("users", column)
		if err != nil {
			return fmt.Errorf("failed to check column users.%s: %w", column, err)
		}
		if !exists {
			return fmt.Errorf("required column missing: users.%s", column)
		}
		log.Debugf("Column users.%s exists (type: %s)", column, actualType)
	}

	// Check indexes
	log.Info("Checking critical indexes...")
	indexes := []struct {
		table string
		index string
	}{
		{"users", "idx_users_email"},
		{"users", "idx_users_tenant_id"},
		{"role_bindings", "idx_role_bindings_user_id"},
		{"migration_logs", "idx_migration_logs_version"},
	}

	for _, idx := range indexes {
		exists, err := sv.indexExists(idx.table, idx.index)
		if err != nil {
			log.Warnf("Failed to check index %s.%s: %v", idx.table, idx.index, err)
			continue
		}
		if exists {
			log.Debugf("Index %s.%s exists", idx.table, idx.index)
		} else {
			log.Warnf("Index %s.%s missing (performance may be affected)", idx.table, idx.index)
		}
	}

	log.Info("Schema validation completed successfully")
	return nil
}

func (sv *SchemaValidator) tableExists(tableName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)
	`
	err := sv.db.QueryRow(query, tableName).Scan(&exists)
	return exists, err
}

func (sv *SchemaValidator) columnExists(tableName, columnName string) (bool, string, error) {
	var exists bool
	var dataType sql.NullString

	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = $1
			AND column_name = $2
		),
		(SELECT data_type FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = $1
			AND column_name = $2
		)
	`

	err := sv.db.QueryRow(query, tableName, columnName).Scan(&exists, &dataType)
	if err != nil {
		return false, "", err
	}

	return exists, dataType.String, nil
}

func (sv *SchemaValidator) indexExists(tableName, indexName string) (bool, error) {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM pg_indexes
			WHERE schemaname = 'public'
			AND tablename = $1
			AND indexname = $2
		)
	`
	err := sv.db.QueryRow(query, tableName, indexName).Scan(&exists)
	return exists, err
}

// GetSchemaVersion returns the highest successfully applied migration version
func (sv *SchemaValidator) GetSchemaVersion(dbType string) (int, error) {
	var version int
	query := `
		SELECT COALESCE(MAX(version), 0)
		FROM migration_logs
		WHERE db_type = $1 AND success = true
	`
	err := sv.db.QueryRow(query, dbType).Scan(&version)
	return version, err
}

// CompareWithMaster compares a tenant database schema with the master
func (sv *SchemaValidator) CompareWithMaster(tenantDB *sql.DB) error {
	log.Info("Comparing tenant schema with master...")

	// Get tables from both databases
	masterTables, err := getTablesFromDB(sv.db)
	if err != nil {
		return fmt.Errorf("failed to get master tables: %w", err)
	}

	tenantTables, err := getTablesFromDB(tenantDB)
	if err != nil {
		return fmt.Errorf("failed to get tenant tables: %w", err)
	}

	missingTables := []string{}
	extraTables := []string{}

	// Find missing tables (in master but not in tenant)
	for _, table := range masterTables {
		if !stringSliceContains(tenantTables, table) {
			missingTables = append(missingTables, table)
		}
	}

	// Find extra tables (in tenant but not in master)
	for _, table := range tenantTables {
		if !stringSliceContains(masterTables, table) {
			extraTables = append(extraTables, table)
		}
	}

	if len(missingTables) > 0 {
		log.Warnf("Tenant database missing tables: %v", missingTables)
	}
	if len(extraTables) > 0 {
		log.Infof("Tenant database has extra tables: %v", extraTables)
	}

	return nil
}

func getTablesFromDB(db *sql.DB) ([]string, error) {
	query := `
		SELECT table_name
		FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_type = 'BASE TABLE'
		ORDER BY table_name
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

func stringSliceContains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
