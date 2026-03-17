package migration

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MigrationFile represents a parsed SQL migration file.
type MigrationFile struct {
	Version  int
	Name     string
	FilePath string
	Content  string
}

// MigrationRunner executes versioned SQL migrations against a database.
type MigrationRunner struct {
	db            *sql.DB  // raw connection (target DB — master or tenant)
	gormDB        *gorm.DB // used only for master DB migration_logs writes
	migrationsDir string
	dbType        string  // "master" or "tenant"
	tenantID      *string // non-nil for tenant runners
	masterDB      *sql.DB // used by tenant runners to write migration_logs
}

// NewMasterMigrationRunner creates a runner for the master database.
func NewMasterMigrationRunner(migrationsDir string, rawDB *sql.DB, gormDB *gorm.DB) *MigrationRunner {
	return &MigrationRunner{
		db:            rawDB,
		gormDB:        gormDB,
		migrationsDir: migrationsDir,
		dbType:        "master",
	}
}

// NewTenantMigrationRunner creates a runner for a tenant database.
// masterDB is used solely for writing migration_logs (which live in the master DB).
func NewTenantMigrationRunner(tenantID string, tenantDBConn *sql.DB, migrationsDir string, masterDB *sql.DB) *MigrationRunner {
	return &MigrationRunner{
		db:            tenantDBConn,
		gormDB:        nil,
		migrationsDir: migrationsDir,
		dbType:        "tenant",
		tenantID:      &tenantID,
		masterDB:      masterDB,
	}
}

// LoadMigrationFiles loads and sorts all SQL files from the runner's migrations directory.
// It also loads from a sibling permissions/<dbType> subdirectory when present.
func (mr *MigrationRunner) LoadMigrationFiles() ([]MigrationFile, error) {
	var migrations []MigrationFile

	main, err := mr.loadMigrationsFromDir(mr.migrationsDir)
	if err != nil {
		return nil, err
	}
	migrations = append(migrations, main...)

	// Load from permissions/<dbType>/ sibling directory
	permDir := filepath.Join(filepath.Dir(mr.migrationsDir), "permissions", filepath.Base(mr.migrationsDir))
	if _, err := os.Stat(permDir); err == nil {
		perms, err := mr.loadMigrationsFromDir(permDir)
		if err != nil {
			log.Printf("[Migration] Warning: failed to load permission migrations from %s: %v", permDir, err)
		} else {
			log.Printf("[Migration] Loaded %d permission migrations from %s", len(perms), permDir)
			migrations = append(migrations, perms...)
		}
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	log.Printf("[Migration] Loaded %d total migration files for %s", len(migrations), mr.dbType)
	return migrations, nil
}

func (mr *MigrationRunner) loadMigrationsFromDir(dir string) ([]MigrationFile, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", dir, err)
	}

	var migrations []MigrationFile
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		version, name, err := parseMigrationFileName(entry.Name())
		if err != nil {
			log.Printf("[Migration] Skipping invalid migration file %s: %v", entry.Name(), err)
			continue
		}

		content, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			log.Printf("[Migration] Warning: failed to read %s: %v", entry.Name(), err)
			continue
		}

		migrations = append(migrations, MigrationFile{
			Version:  version,
			Name:     name,
			FilePath: filepath.Join(dir, entry.Name()),
			Content:  string(content),
		})
	}
	return migrations, nil
}

// parseMigrationFileName extracts the integer version and descriptive name from a filename.
// Expected format: 001_create_users_table.sql
func parseMigrationFileName(filename string) (int, string, error) {
	base := strings.TrimSuffix(filename, ".sql")
	parts := strings.SplitN(base, "_", 2)
	if len(parts) < 2 {
		return 0, "", fmt.Errorf("invalid migration filename format: %s", filename)
	}
	version, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", fmt.Errorf("invalid version number in %s: %w", filename, err)
	}
	return version, parts[1], nil
}

// RunMigrations executes all pending migrations with retry logic.
func (mr *MigrationRunner) RunMigrations() error {
	log.Printf("[Migration] Starting %s database migrations", mr.dbType)

	// For tenant databases, execute the base template first if the schema doesn't exist yet.
	if mr.dbType == "tenant" {
		templatePath := filepath.Join(mr.migrationsDir, "000_tenant_template.sql")
		if _, err := os.Stat(templatePath); err == nil {
			var schemaExists bool
			mr.db.QueryRow(
				"SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_schema='public' AND table_name='users')",
			).Scan(&schemaExists)

			if schemaExists {
				log.Printf("[Migration] Tenant schema already exists, skipping template")
			} else {
				log.Printf("[Migration] Executing tenant base template")
				content, err := os.ReadFile(templatePath)
				if err != nil {
					return fmt.Errorf("failed to read tenant template: %w", err)
				}
				if err := mr.executeSQLContent(string(content)); err != nil {
					return fmt.Errorf("tenant template execution failed: %w", err)
				}
				log.Printf("[Migration] Tenant base template executed successfully")
			}
		}
	}

	// Seed tenant self-reference row so DML migrations can resolve tenant_id.
	if mr.dbType == "tenant" && mr.tenantID != nil {
		seedSQL := `INSERT INTO tenants (id, tenant_id, status, created_at, updated_at)
		            VALUES ($1::uuid, $1::uuid, 'active', NOW(), NOW())
		            ON CONFLICT (id) DO NOTHING`
		if _, err := mr.db.Exec(seedSQL, *mr.tenantID); err != nil {
			log.Printf("[Migration] Warning: failed to seed tenant self-reference row (non-fatal): %v", err)
		} else {
			log.Printf("[Migration] Seeded tenant self-reference row for tenant %s", *mr.tenantID)
		}
	}

	allMigrations, err := mr.LoadMigrationFiles()
	if err != nil {
		return err
	}

	// Filter version-0 template for tenant (already handled above)
	var migrations []MigrationFile
	for _, m := range allMigrations {
		if mr.dbType == "tenant" && m.Version == 0 {
			continue
		}
		migrations = append(migrations, m)
	}

	if len(migrations) == 0 {
		log.Printf("[Migration] No migration files found for %s", mr.dbType)
		return nil
	}

	const maxRetries = 3
	executedCount, failedCount := 0, 0

	for _, m := range migrations {
		if mr.isMigrationExecuted(m.Version) {
			log.Printf("[Migration] %s v%d (%s) already applied, skipping", mr.dbType, m.Version, m.Name)
			continue
		}

		log.Printf("[Migration] Applying %s v%d: %s", mr.dbType, m.Version, m.Name)

		var lastErr error
		var executionMS int64
		succeeded := false

		for attempt := 1; attempt <= maxRetries; attempt++ {
			start := time.Now()
			err := mr.executeSQLContent(m.Content)
			executionMS = time.Since(start).Milliseconds()

			if err == nil {
				succeeded = true
				executedCount++
				log.Printf("[Migration] %s v%d completed in %dms", mr.dbType, m.Version, executionMS)
				break
			}

			lastErr = err
			log.Printf("[Migration] %s v%d attempt %d/%d failed: %v", mr.dbType, m.Version, attempt, maxRetries, err)
			if attempt < maxRetries {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			}
		}

		if succeeded {
			mr.logMigration(m.Version, m.Name, true, "", executionMS)
		} else {
			failedCount++
			errMsg := fmt.Sprintf("FAILED after %d attempts: %v", maxRetries, lastErr)
			log.Printf("[Migration] ERROR: %s v%d %s", mr.dbType, m.Version, errMsg)
			mr.logMigration(m.Version, m.Name, false, errMsg, executionMS)
		}
	}

	log.Printf("[Migration] %s done: %d applied, %d failed", mr.dbType, executedCount, failedCount)

	if failedCount > 0 {
		return fmt.Errorf("%d out of %d %s migrations failed", failedCount, len(migrations), mr.dbType)
	}
	return nil
}

// executeSQLContent executes arbitrary SQL content in a single transaction.
func (mr *MigrationRunner) executeSQLContent(content string) error {
	tx, err := mr.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	for _, stmt := range splitSQLStatements(content) {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := tx.Exec(stmt); err != nil {
			tx.Rollback()
			return fmt.Errorf("execute statement: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// isMigrationExecuted returns true if the given version is already recorded as successful.
func (mr *MigrationRunner) isMigrationExecuted(version int) bool {
	query := `SELECT COUNT(*) FROM migration_logs WHERE version = $1 AND db_type = $2 AND success = true`
	args := []interface{}{version, mr.dbType}

	var queryDB *sql.DB
	if mr.dbType == "tenant" && mr.masterDB != nil {
		queryDB = mr.masterDB
		query += ` AND tenant_id = $3`
		args = append(args, *mr.tenantID)
	} else {
		queryDB = mr.db
		query += ` AND tenant_id IS NULL`
	}

	var count int64
	if err := queryDB.QueryRow(query, args...).Scan(&count); err != nil {
		return false // table may not exist yet
	}
	return count > 0
}

// logMigration records a migration execution in migration_logs.
func (mr *MigrationRunner) logMigration(version int, name string, success bool, errorMsg string, executionMS int64) {
	if mr.gormDB != nil {
		mr.gormDB.Create(&MigrationLog{
			ID:          uuid.New(),
			Version:     version,
			Name:        name,
			ExecutedAt:  time.Now().UTC(),
			Success:     success,
			ErrorMsg:    errorMsg,
			DBType:      mr.dbType,
			TenantID:    mr.tenantID,
			ExecutionMS: executionMS,
		})
		return
	}

	// Fallback: write directly via raw SQL (used by tenant runners)
	logDB := mr.db
	if mr.masterDB != nil {
		logDB = mr.masterDB
	}

	tenantIDVal := sql.NullString{}
	if mr.tenantID != nil {
		tenantIDVal = sql.NullString{String: *mr.tenantID, Valid: true}
	}

	_, err := logDB.Exec(
		`INSERT INTO migration_logs (id, version, name, executed_at, success, error_msg, db_type, tenant_id, execution_ms)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		uuid.New().String(), version, name, time.Now().UTC(), success, errorMsg, mr.dbType, tenantIDVal, executionMS,
	)
	if err != nil {
		log.Printf("[Migration] Warning: failed to log migration v%d: %v", version, err)
	}
}

// GetMigrationStatus returns a summary of migration progress.
func (mr *MigrationRunner) GetMigrationStatus() (*MigrationStatusResponse, error) {
	migrations, err := mr.LoadMigrationFiles()
	if err != nil {
		return nil, err
	}

	queryDB := mr.db
	if mr.dbType == "tenant" && mr.masterDB != nil {
		queryDB = mr.masterDB
	}

	baseQuery := `SELECT version, executed_at FROM migration_logs WHERE db_type = $1 AND success = true`
	args := []interface{}{mr.dbType}
	if mr.tenantID != nil {
		baseQuery += ` AND tenant_id = $2`
		args = append(args, *mr.tenantID)
	} else {
		baseQuery += ` AND tenant_id IS NULL`
	}

	var lastMigration int
	var lastExecuted time.Time
	queryDB.QueryRow(baseQuery+` ORDER BY version DESC LIMIT 1`, args...).Scan(&lastMigration, &lastExecuted)

	var executedCount int
	countQuery := strings.Replace(baseQuery, "version, executed_at", "COUNT(*)", 1)
	queryDB.QueryRow(countQuery, args...).Scan(&executedCount)

	status := "pending"
	if executedCount == len(migrations) {
		status = "completed"
	} else if executedCount > 0 {
		status = "in_progress"
	}

	return &MigrationStatusResponse{
		DBType:          mr.dbType,
		TenantID:        mr.tenantID,
		LastMigration:   lastMigration,
		TotalMigrations: len(migrations),
		Status:          status,
		LastExecuted:    lastExecuted,
	}, nil
}

// MigrationsDir resolves the canonical migrations directory path at runtime.
func MigrationsDir(dbType string) string {
	execPath, err := os.Executable()
	if err == nil {
		p := filepath.Join(filepath.Dir(execPath), "migrations", dbType)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, "migrations", dbType)
	if _, err := os.Stat(p); err == nil {
		return p
	}
	return filepath.Join("migrations", dbType)
}

// splitSQLStatements intelligently splits SQL into individual statements,
// handling dollar-quoted strings, single-quoted strings, and comments.
func splitSQLStatements(content string) []string {
	var statements []string
	var current strings.Builder
	runes := []rune(content)
	i := 0

	for i < len(runes) {
		// Dollar-quoted string
		if runes[i] == '$' {
			if tag := extractDollarTag(runes, i); tag != "" {
				for j := 0; j < len(tag); j++ {
					current.WriteRune(runes[i])
					i++
				}
				for i < len(runes) {
					current.WriteRune(runes[i])
					if i+len(tag) <= len(runes) && string(runes[i:i+len(tag)]) == tag {
						for j := 1; j < len(tag); j++ {
							i++
							if i < len(runes) {
								current.WriteRune(runes[i])
							}
						}
						i++
						break
					}
					i++
				}
				continue
			}
		}

		// Single-line comment
		if i+1 < len(runes) && runes[i] == '-' && runes[i+1] == '-' {
			for i < len(runes) && runes[i] != '\n' {
				current.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				current.WriteRune(runes[i])
				i++
			}
			continue
		}

		// Multi-line comment
		if i+1 < len(runes) && runes[i] == '/' && runes[i+1] == '*' {
			current.WriteRune(runes[i])
			current.WriteRune(runes[i+1])
			i += 2
			for i+1 < len(runes) {
				current.WriteRune(runes[i])
				if runes[i] == '*' && runes[i+1] == '/' {
					i++
					current.WriteRune(runes[i])
					i++
					break
				}
				i++
			}
			continue
		}

		// Single-quoted string
		if runes[i] == '\'' {
			current.WriteRune(runes[i])
			i++
			for i < len(runes) {
				current.WriteRune(runes[i])
				if runes[i] == '\'' {
					if i+1 < len(runes) && runes[i+1] == '\'' {
						i++
						current.WriteRune(runes[i])
					} else {
						i++
						break
					}
				}
				i++
			}
			continue
		}

		// Statement terminator
		if runes[i] == ';' {
			if stmt := strings.TrimSpace(current.String()); stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
			i++
			continue
		}

		current.WriteRune(runes[i])
		i++
	}

	if stmt := strings.TrimSpace(current.String()); stmt != "" {
		statements = append(statements, stmt)
	}
	return statements
}

func extractDollarTag(runes []rune, i int) string {
	if i >= len(runes) || runes[i] != '$' {
		return ""
	}
	j := i + 1
	for j < len(runes) && j < i+100 {
		if runes[j] == '$' {
			return string(runes[i : j+1])
		}
		if !((runes[j] >= 'a' && runes[j] <= 'z') ||
			(runes[j] >= 'A' && runes[j] <= 'Z') ||
			(runes[j] >= '0' && runes[j] <= '9') ||
			runes[j] == '_') {
			return ""
		}
		j++
	}
	return ""
}
