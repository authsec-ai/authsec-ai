package migration

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

const syntheticTenantID = "00000000-0000-0000-0000-000000000001"

// requiredTables is the list of tables that a fully-migrated tenant DB must contain.
var requiredTables = []string{
	"tenants", "users", "roles", "permissions", "clients",
	"role_bindings", "role_permissions", "service_accounts",
	"api_scopes", "scope_permissions", "api_scope_permissions",
	"groups", "user_groups", "resources", "resource_methods",
	"user_scopes", "client_roles", "credentials", "scopes",
	"services", "projects", "group_roles", "client_resources",
	"client_scopes", "client_groups", "user_roles",
	"delegation_policies", "delegation_tokens",
}

// requiredColumns maps table -> columns that must exist after incremental migrations.
var requiredColumns = map[string][]string{
	"users":         {"is_primary_admin"},
	"permissions":   {"resource", "action"},
	"role_bindings": {"username", "role_name"},
}

// requiredConstraints lists constraint names that must exist.
var requiredConstraints = []string{
	"users_tenant_id_id_unique",
}

// SetupTenantTemplate drops and recreates the golden template DB, runs all
// tenant migrations on it, then verifies the schema 3 times. If setup
// succeeds, TemplateReady is set to true.
// masterDB is an optional raw connection to the master database used to clear
// stale migration logs for the synthetic template tenant before rebuilding.
func SetupTenantTemplate(tenantMigrationsDir string, masterDB ...*sql.DB) error {
	start := time.Now()
	log.Printf("[Migration] Template setup: starting golden template build")

	// Phase 0: Clear stale migration logs for the synthetic tenant so that
	// all tenant migrations run fresh on the new template DB.
	if len(masterDB) > 0 && masterDB[0] != nil {
		if err := clearSyntheticTenantLogs(masterDB[0]); err != nil {
			log.Printf("[Migration] Template setup: could not clear stale migration logs: %v", err)
		}
	}

	if err := recreateTemplateDB(); err != nil {
		return fmt.Errorf("phase 1 (recreate): %w", err)
	}

	if err := runMigrationsOnTemplate(tenantMigrationsDir); err != nil {
		return fmt.Errorf("phase 2 (migrations): %w", err)
	}

	for attempt := 1; attempt <= 3; attempt++ {
		if err := verifyTemplateSchema(attempt); err != nil {
			return fmt.Errorf("phase 3 (verify round %d/3): %w", attempt, err)
		}
		log.Printf("[Migration] Template setup: verification %d/3 passed", attempt)
	}

	pgDB, err := connectToPostgresDB()
	if err == nil {
		terminateDBConnections(pgDB, TemplateDBName)
		pgDB.Close()
	}

	TemplateReady = true
	log.Printf("[Migration] Template setup: complete in %v — template ready for cloning",
		time.Since(start).Round(time.Millisecond))
	return nil
}

// recreateTemplateDB drops the old template (if any) and creates a fresh one.
func recreateTemplateDB() error {
	pgDB, err := connectToPostgresDB()
	if err != nil {
		return err
	}
	defer pgDB.Close()

	for attempt := 0; attempt < 3; attempt++ {
		terminateDBConnections(pgDB, TemplateDBName)
		_, err = pgDB.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s", TemplateDBName))
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("failed to drop template DB: %w", err)
	}

	time.Sleep(50 * time.Millisecond)

	if _, err := pgDB.Exec(fmt.Sprintf("CREATE DATABASE %s WITH ENCODING 'UTF8'", TemplateDBName)); err != nil {
		return fmt.Errorf("failed to create template DB: %w", err)
	}

	log.Printf("[Migration] Template setup: fresh template DB created")
	return nil
}

// runMigrationsOnTemplate connects to the template DB and executes all tenant migrations.
func runMigrationsOnTemplate(tenantMigrationsDir string) error {
	conn, err := ConnectToNamedDB(TemplateDBName)
	if err != nil {
		return fmt.Errorf("failed to connect to template DB: %w", err)
	}
	defer conn.Close()

	runner := NewTenantMigrationRunner(syntheticTenantID, conn, tenantMigrationsDir, nil)
	if err := runner.RunMigrations(); err != nil {
		return fmt.Errorf("tenant migrations failed on template: %w", err)
	}

	log.Printf("[Migration] Template setup: all tenant migrations applied successfully")
	return nil
}

// verifyTemplateSchema opens a fresh connection and checks tables, columns, and constraints.
func verifyTemplateSchema(attempt int) error {
	conn, err := ConnectToNamedDB(TemplateDBName)
	if err != nil {
		return fmt.Errorf("connect failed: %w", err)
	}
	defer conn.Close()

	for _, table := range requiredTables {
		var exists bool
		if err := conn.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)",
			table,
		).Scan(&exists); err != nil {
			return fmt.Errorf("query table %s: %w", table, err)
		}
		if !exists {
			return fmt.Errorf("required table missing: %s", table)
		}
	}

	for table, cols := range requiredColumns {
		for _, col := range cols {
			var exists bool
			if err := conn.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM information_schema.columns WHERE table_schema='public' AND table_name=$1 AND column_name=$2)",
				table, col,
			).Scan(&exists); err != nil {
				return fmt.Errorf("query column %s.%s: %w", table, col, err)
			}
			if !exists {
				return fmt.Errorf("required column missing: %s.%s", table, col)
			}
		}
	}

	for _, constraint := range requiredConstraints {
		var exists bool
		if err := conn.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM pg_constraint WHERE conname = $1)",
			constraint,
		).Scan(&exists); err != nil {
			return fmt.Errorf("query constraint %s: %w", constraint, err)
		}
		if !exists {
			return fmt.Errorf("required constraint missing: %s", constraint)
		}
	}

	var tableCount int
	if err := conn.QueryRow(
		"SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public'",
	).Scan(&tableCount); err != nil {
		return fmt.Errorf("count tables: %w", err)
	}
	if tableCount < len(requiredTables) {
		return fmt.Errorf("too few tables: got %d, need >= %d", tableCount, len(requiredTables))
	}

	var permCount int
	if err := conn.QueryRow("SELECT COUNT(*) FROM permissions").Scan(&permCount); err != nil {
		return fmt.Errorf("count permissions: %w", err)
	}
	if permCount < 5 {
		return fmt.Errorf("insufficient seed data: %d permissions (need >= 5)", permCount)
	}

	log.Printf("[Migration] Template verify %d: %d tables, %d permissions — OK", attempt, tableCount, permCount)
	return nil
}

// clearSyntheticTenantLogs removes migration_logs entries for the synthetic
// template tenant so that a fresh template build re-runs all migrations.
func clearSyntheticTenantLogs(masterDB *sql.DB) error {
	result, err := masterDB.Exec(
		"DELETE FROM migration_logs WHERE db_type = 'tenant' AND tenant_id = $1",
		syntheticTenantID,
	)
	if err != nil {
		return err
	}
	if n, _ := result.RowsAffected(); n > 0 {
		log.Printf("[Migration] Template setup: cleared %d stale migration log entries for synthetic tenant", n)
	}
	return nil
}
