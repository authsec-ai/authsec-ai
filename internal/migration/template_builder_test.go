package migration

import (
	"path/filepath"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testTemplateMasterDB = "test_template_master"
	testTemplateDBName   = "_test_tenant_template"
)

// setupTemplateMasterDB prepares a master database and configures template
// credentials so that template builder functions (SetupTenantTemplate,
// CloneTenantDatabase, etc.) can operate during tests.
func setupTemplateMasterDB(t *testing.T) func() {
	t.Helper()

	// Use a test-specific template DB name to avoid collisions with other tests
	origTemplateName := TemplateDBName
	TemplateDBName = testTemplateDBName

	// Pre-clean to handle leftover from previous runs
	dropTestDatabase(t, testTemplateDBName)
	createTestDatabase(t, testTemplateMasterDB)

	// Configure the template credentials used by connectToPostgresDB, ConnectToNamedDB, etc.
	InitTemplateCreds(testDBHost, testDBPort, testDBUser, testDBPassword, testDBSSLMode)

	// Set up migration_logs in the master DB
	masterConn := connectTestDB(t, testTemplateMasterDB)
	createMigrationLogsTable(t, masterConn)

	// Run master migrations so the master schema is in place
	mDir := testMigrationsDir(t)
	masterDir := filepath.Join(mDir, "master")
	runner := NewMasterMigrationRunner(masterDir, masterConn, nil)
	_ = runner.RunMigrations() // Partial failures OK for base schema conflicts

	return func() {
		masterConn.Close()
		dropTestDatabase(t, testTemplateMasterDB)
		dropTestDatabase(t, testTemplateDBName)
		TemplateDBName = origTemplateName
	}
}

// ----- Tests -----

func TestSetupTenantTemplate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanup := setupTemplateMasterDB(t)
	defer cleanup()

	TemplateReady = false

	mDir := testMigrationsDir(t)
	tenantDir := filepath.Join(mDir, "tenant")

	// Pass master DB connection for Phase 0 stale log cleanup
	masterConn := connectTestDB(t, testTemplateMasterDB)
	defer masterConn.Close()

	err := SetupTenantTemplate(tenantDir, masterConn)
	require.NoError(t, err)
	assert.True(t, TemplateReady, "TemplateReady should be true after successful setup")

	// Verify the template DB actually exists
	pgDB := connectTestDB(t, "postgres")
	defer pgDB.Close()
	var exists bool
	err = pgDB.QueryRow("SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)", TemplateDBName).Scan(&exists)
	require.NoError(t, err)
	assert.True(t, exists, "template database should exist")
}

func TestCloneTenantDatabase_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanup := setupTemplateMasterDB(t)
	defer cleanup()

	TemplateReady = false
	mDir := testMigrationsDir(t)
	tenantDir := filepath.Join(mDir, "tenant")

	masterConn := connectTestDB(t, testTemplateMasterDB)
	defer masterConn.Close()

	err := SetupTenantTemplate(tenantDir, masterConn)
	require.NoError(t, err)
	require.True(t, TemplateReady)

	clonedDB := "test_cloned_tenant_001"
	defer dropTestDatabase(t, clonedDB)

	created, err := CloneTenantDatabase(clonedDB)
	require.NoError(t, err)
	assert.True(t, created, "should have created new database")

	// Verify cloned DB has full schema
	conn := connectTestDB(t, clonedDB)
	defer conn.Close()

	for _, table := range requiredTables {
		var exists bool
		err := conn.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name=$1)",
			table,
		).Scan(&exists)
		require.NoError(t, err)
		assert.True(t, exists, "cloned DB should have table: %s", table)
	}

	// Verify permissions seed data was cloned
	var permCount int
	err = conn.QueryRow("SELECT COUNT(*) FROM permissions").Scan(&permCount)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, permCount, 5, "cloned DB should have seed permissions")
}

func TestCloneTenantDatabase_FailsWhenNotReady(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// Configure credentials so the function doesn't fail on missing config
	InitTemplateCreds(testDBHost, testDBPort, testDBUser, testDBPassword, testDBSSLMode)

	TemplateReady = false

	_, err := CloneTenantDatabase("some_db")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not ready")
}

func TestCloneTenantDatabase_Idempotent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	cleanup := setupTemplateMasterDB(t)
	defer cleanup()

	TemplateReady = false
	mDir := testMigrationsDir(t)
	tenantDir := filepath.Join(mDir, "tenant")

	masterConn := connectTestDB(t, testTemplateMasterDB)
	defer masterConn.Close()

	err := SetupTenantTemplate(tenantDir, masterConn)
	require.NoError(t, err)

	clonedDB := "test_cloned_tenant_idem"
	defer dropTestDatabase(t, clonedDB)

	// First clone
	created, err := CloneTenantDatabase(clonedDB)
	require.NoError(t, err)
	assert.True(t, created)

	// Second clone of same name — should return (false, nil)
	created2, err2 := CloneTenantDatabase(clonedDB)
	require.NoError(t, err2)
	assert.False(t, created2, "second clone should return false (already exists)")
}

func TestVerifyTemplateSchema_CatchesMissingTable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	origTemplateName := TemplateDBName
	TemplateDBName = testTemplateDBName
	defer func() { TemplateDBName = origTemplateName }()

	InitTemplateCreds(testDBHost, testDBPort, testDBUser, testDBPassword, testDBSSLMode)

	// Create a bare empty DB named like the template
	dropTestDatabase(t, testTemplateDBName)
	createTestDatabase(t, testTemplateDBName)
	defer dropTestDatabase(t, testTemplateDBName)

	err := verifyTemplateSchema(1)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "required table missing")
}
