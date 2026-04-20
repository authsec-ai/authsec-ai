package migration

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ----- Integration Test: Migration vs Template Parity -----
// Verifies that a database built by running all tenant migrations produces
// an identical schema to one cloned from the golden template.

func TestIntegration_MigrationVsTemplate_Parity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// --- Setup template infrastructure ---
	origTemplateName := TemplateDBName
	TemplateDBName = "_test_parity_template"
	defer func() { TemplateDBName = origTemplateName }()

	InitTemplateCreds(testDBHost, testDBPort, testDBUser, testDBPassword, testDBSSLMode)

	const masterDB = "test_parity_master"
	createTestDatabase(t, masterDB)
	defer dropTestDatabase(t, masterDB)
	defer dropTestDatabase(t, TemplateDBName)

	masterConn := connectTestDB(t, masterDB)
	defer masterConn.Close()
	createMigrationLogsTable(t, masterConn)

	mDir := testMigrationsDir(t)
	masterDir := filepath.Join(mDir, "master")
	tenantDir := filepath.Join(mDir, "tenant")

	// Run master migrations
	masterRunner := NewMasterMigrationRunner(masterDir, masterConn, nil)
	_ = masterRunner.RunMigrations()

	// --- Tenant A: created via RunMigrations ---
	const dbA = "test_parity_tenant_migration"
	createTestDatabase(t, dbA)
	defer dropTestDatabase(t, dbA)

	tenantConnA := connectTestDB(t, dbA)
	defer tenantConnA.Close()

	tenantRunnerA := NewTenantMigrationRunner("parity-tenant-a", tenantConnA, tenantDir, masterConn)
	err := tenantRunnerA.RunMigrations()
	require.NoError(t, err, "tenant A migrations should succeed")

	// --- Tenant B: created via template clone ---
	TemplateReady = false
	err = SetupTenantTemplate(tenantDir, masterConn)
	require.NoError(t, err, "template setup should succeed")
	require.True(t, TemplateReady)

	const dbB = "test_parity_tenant_template"
	defer dropTestDatabase(t, dbB)

	created, err := CloneTenantDatabase(dbB)
	require.NoError(t, err, "clone should succeed")
	require.True(t, created)

	// --- Compare schemas ---
	t.Run("tables_match", func(t *testing.T) {
		tablesA := getPublicTableNames(t, dbA)
		tablesB := getPublicTableNames(t, dbB)
		assert.Greater(t, len(tablesA), 5, "migration tenant should have meaningful tables")
		assert.Equal(t, tablesA, tablesB,
			"migration-created and template-cloned tenant DBs should have identical tables")
	})

	t.Run("column_counts_match", func(t *testing.T) {
		tables := getPublicTableNames(t, dbA)
		for _, table := range tables {
			colsA := getPublicColumnCount(t, dbA, table)
			colsB := getPublicColumnCount(t, dbB, table)
			assert.Equal(t, colsA, colsB,
				"table %s should have same column count (migration=%d, template=%d)", table, colsA, colsB)
		}
	})

	t.Run("column_details_match", func(t *testing.T) {
		keyTables := []string{
			"users", "roles", "permissions", "role_bindings",
			"clients", "api_scopes", "delegation_policies", "delegation_tokens",
		}
		for _, table := range keyTables {
			colsA := getPublicColumnDetails(t, dbA, table)
			colsB := getPublicColumnDetails(t, dbB, table)
			assert.Equal(t, colsA, colsB,
				"table %s should have identical column definitions", table)
		}
	})

	t.Run("constraints_match", func(t *testing.T) {
		constraintsA := getPublicConstraintNames(t, dbA)
		constraintsB := getPublicConstraintNames(t, dbB)
		assert.Equal(t, constraintsA, constraintsB,
			"both tenant DBs should have identical constraints")
	})

	t.Run("indexes_match", func(t *testing.T) {
		indexesA := getPublicIndexNames(t, dbA)
		indexesB := getPublicIndexNames(t, dbB)
		assert.Equal(t, indexesA, indexesB,
			"both tenant DBs should have identical indexes")
	})

	t.Run("seed_data_match", func(t *testing.T) {
		connA := connectTestDB(t, dbA)
		defer connA.Close()
		connB := connectTestDB(t, dbB)
		defer connB.Close()

		var permCountA, permCountB int
		connA.QueryRow("SELECT COUNT(*) FROM permissions").Scan(&permCountA)
		connB.QueryRow("SELECT COUNT(*) FROM permissions").Scan(&permCountB)
		assert.Equal(t, permCountA, permCountB,
			"both tenant DBs should have same number of permissions")

		var roleCountA, roleCountB int
		connA.QueryRow("SELECT COUNT(*) FROM roles").Scan(&roleCountA)
		connB.QueryRow("SELECT COUNT(*) FROM roles").Scan(&roleCountB)
		assert.Equal(t, roleCountA, roleCountB,
			"both tenant DBs should have same number of roles")
	})
}

// ----- Integration Test: Multi-Tenant Schema Consistency -----
// Verifies that creating multiple tenant databases produces identical schemas.

func TestIntegration_MultiTenant_SchemaConsistency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const masterDB = "test_consistency_master"
	const tenantDBOne = "test_consistency_tenant_one"
	const tenantDBTwo = "test_consistency_tenant_two"

	createTestDatabase(t, masterDB)
	createTestDatabase(t, tenantDBOne)
	createTestDatabase(t, tenantDBTwo)
	defer dropTestDatabase(t, masterDB)
	defer dropTestDatabase(t, tenantDBOne)
	defer dropTestDatabase(t, tenantDBTwo)

	masterConn := connectTestDB(t, masterDB)
	defer masterConn.Close()
	createMigrationLogsTable(t, masterConn)

	mDir := testMigrationsDir(t)
	tenantDir := filepath.Join(mDir, "tenant")

	// Create tenant one
	connOne := connectTestDB(t, tenantDBOne)
	defer connOne.Close()
	runnerOne := NewTenantMigrationRunner("tenant-one", connOne, tenantDir, masterConn)
	err := runnerOne.RunMigrations()
	require.NoError(t, err, "tenant one migrations should succeed")

	// Create tenant two
	connTwo := connectTestDB(t, tenantDBTwo)
	defer connTwo.Close()
	runnerTwo := NewTenantMigrationRunner("tenant-two", connTwo, tenantDir, masterConn)
	err = runnerTwo.RunMigrations()
	require.NoError(t, err, "tenant two migrations should succeed")

	t.Run("tables_match", func(t *testing.T) {
		tablesOne := getPublicTableNames(t, tenantDBOne)
		tablesTwo := getPublicTableNames(t, tenantDBTwo)
		assert.Equal(t, tablesOne, tablesTwo,
			"both tenant databases should have identical table sets")
		assert.Greater(t, len(tablesOne), 5, "should have meaningful number of tables")
	})

	t.Run("column_counts_match", func(t *testing.T) {
		for _, table := range []string{"users", "roles", "permissions", "role_bindings", "clients"} {
			colsOne := getPublicColumnCount(t, tenantDBOne, table)
			colsTwo := getPublicColumnCount(t, tenantDBTwo, table)
			assert.Equal(t, colsOne, colsTwo,
				"table %s should have same number of columns in both tenant DBs", table)
		}
	})
}

// ----- Integration Test: RunTenantMigrationsInProcess -----
// Tests the in-process tenant migration helper exposed in db_utils.go.

func TestIntegration_RunTenantMigrationsInProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	const masterDB = "test_inprocess_master"
	const tenantDB = "test_inprocess_tenant"

	createTestDatabase(t, masterDB)
	createTestDatabase(t, tenantDB)
	defer dropTestDatabase(t, masterDB)
	defer dropTestDatabase(t, tenantDB)

	masterConn := connectTestDB(t, masterDB)
	defer masterConn.Close()
	createMigrationLogsTable(t, masterConn)

	mDir := testMigrationsDir(t)
	tenantDir := filepath.Join(mDir, "tenant")

	err := RunTenantMigrationsInProcess(
		"test-inprocess-tenant",
		testDBHost, testDBPort, testDBUser, testDBPassword,
		tenantDB, masterConn, tenantDir,
	)
	require.NoError(t, err, "in-process migrations should succeed")

	// Verify schema
	conn := connectTestDB(t, tenantDB)
	defer conn.Close()

	for _, table := range requiredTables {
		var exists bool
		err := conn.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)",
			table,
		).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "tenant DB should have table: %s", table)
	}
}

// ----- Schema comparison helpers -----

func getPublicTableNames(t *testing.T, dbName string) []string {
	t.Helper()
	db := connectTestDB(t, dbName)
	defer db.Close()
	return queryStringSlice(t, db, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
		ORDER BY table_name
	`)
}

func getPublicColumnCount(t *testing.T, dbName, tableName string) int {
	t.Helper()
	db := connectTestDB(t, dbName)
	defer db.Close()
	var count int
	err := db.QueryRow(`
		SELECT COUNT(*) FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
	`, tableName).Scan(&count)
	require.NoError(t, err)
	return count
}

func getPublicColumnDetails(t *testing.T, dbName, tableName string) []string {
	t.Helper()
	db := connectTestDB(t, dbName)
	defer db.Close()
	return queryStringSlice(t, db, `
		SELECT column_name || ':' || data_type || ':' || COALESCE(character_maximum_length::text, '') || ':' || is_nullable
		FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY column_name
	`, tableName)
}

func getPublicConstraintNames(t *testing.T, dbName string) []string {
	t.Helper()
	db := connectTestDB(t, dbName)
	defer db.Close()
	return queryStringSlice(t, db, `
		SELECT conname FROM pg_constraint
		JOIN pg_namespace ON pg_namespace.oid = connamespace
		WHERE nspname = 'public'
		ORDER BY conname
	`)
}

func getPublicIndexNames(t *testing.T, dbName string) []string {
	t.Helper()
	db := connectTestDB(t, dbName)
	defer db.Close()
	return queryStringSlice(t, db, `
		SELECT indexname FROM pg_indexes
		WHERE schemaname = 'public'
		ORDER BY indexname
	`)
}

func queryStringSlice(t *testing.T, db *sql.DB, query string, args ...interface{}) []string {
	t.Helper()
	rows, err := db.Query(query, args...)
	require.NoError(t, err)
	defer rows.Close()

	var result []string
	for rows.Next() {
		var s string
		require.NoError(t, rows.Scan(&s))
		result = append(result, s)
	}
	return result
}
