package services

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/authsec-ai/authsec/config"
	"github.com/authsec-ai/authsec/internal/clients/icp"
	_ "github.com/lib/pq"
)

// ICPProvisioningService handles ICP-specific provisioning tasks
type ICPProvisioningService struct {
	icpClient *icp.Client
}

// NewICPProvisioningService creates a new ICP provisioning service
func NewICPProvisioningService(icpClient *icp.Client) *ICPProvisioningService {
	return &ICPProvisioningService{
		icpClient: icpClient,
	}
}

// ApplyICPTenantMigrations applies ICP tenant schema to a tenant database with proper migration tracking
func (s *ICPProvisioningService) ApplyICPTenantMigrations(tenantDBURL string) error {
	log.Printf("Applying ICP tenant migrations to tenant database")

	// Connect to tenant database
	db, err := sql.Open("postgres", tenantDBURL)
	if err != nil {
		return fmt.Errorf("failed to connect to tenant db: %w", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping tenant db: %w", err)
	}

	// Create ICP migrations tracking table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS icp_tenant_migrations (
			version INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create icp_tenant_migrations table: %w", err)
	}

	// Resolve migrations path
	migrationsPath, err := s.resolveICPMigrationsPath()
	if err != nil {
		return err
	}

	log.Printf("Using ICP migrations from: %s", migrationsPath)

	// Read migration files
	migrationFiles, err := filepath.Glob(filepath.Join(migrationsPath, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("failed to list migration files: %w", err)
	}

	if len(migrationFiles) == 0 {
		log.Printf("No migration files found in %s", migrationsPath)
		return nil
	}

	// Sort migration files to ensure they're applied in order
	sort.Strings(migrationFiles)

	log.Printf("Found %d migration file(s) to process", len(migrationFiles))

	// Apply each migration with tracking
	for _, migrationFile := range migrationFiles {
		filename := filepath.Base(migrationFile)

		// Extract version number from filename (e.g., 000005_create_agents_table.up.sql -> 5)
		parts := strings.SplitN(filename, "_", 2)
		versionStr := parts[0]
		version, err := strconv.Atoi(versionStr)
		if err != nil {
			log.Printf("Warning: invalid migration version in file %s: %v", filename, err)
			continue
		}

		// Extract migration name
		var name string
		if len(parts) > 1 {
			name = strings.TrimSuffix(parts[1], ".up.sql")
		} else {
			name = versionStr
		}

		// Check if migration already applied
		var count int
		err = db.QueryRow("SELECT COUNT(*) FROM icp_tenant_migrations WHERE version = $1", version).Scan(&count)
		if err != nil {
			log.Printf("Warning: failed to check migration status for %d: %v", version, err)
			continue
		}

		if count > 0 {
			log.Printf("ICP migration %d (%s) already applied, skipping", version, name)
			continue
		}

		// Read migration file
		migrationSQL, err := os.ReadFile(migrationFile)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", migrationFile, err)
		}

		// Execute migration in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %d: %w", version, err)
		}

		log.Printf("Applying ICP migration %d: %s", version, name)

		// Execute migration SQL
		_, err = tx.Exec(string(migrationSQL))
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d (%s): %w", version, name, err)
		}

		// Record migration as applied
		_, err = tx.Exec("INSERT INTO icp_tenant_migrations (version, name) VALUES ($1, $2)", version, name)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", version, err)
		}

		// Commit transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", version, err)
		}

		log.Printf("Successfully applied ICP migration %d: %s", version, name)
	}

	log.Printf("Successfully applied all ICP tenant migrations")
	return nil
}

// resolveICPMigrationsPath finds the ICP tenant migrations directory
func (s *ICPProvisioningService) resolveICPMigrationsPath() (string, error) {
	// Try multiple possible paths
	candidates := []string{
		filepath.Join("migrations", "icp-tenant"),
		filepath.Join("..", "migrations", "icp-tenant"),
		"/app/migrations/icp-tenant", // For containerized environments
	}

	for _, path := range candidates {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		// Check if directory exists by trying to list it
		if _, err := filepath.Glob(filepath.Join(absPath, "*.sql")); err == nil {
			return absPath, nil
		}
	}

	return "", fmt.Errorf("ICP tenant migrations not found (checked: %v)", candidates)
}

// ProvisionPKI calls ICP service to provision PKI for a tenant
func (s *ICPProvisioningService) ProvisionPKI(ctx context.Context, req *icp.ProvisionPKIRequest) (*icp.ProvisionPKIResponse, error) {
	log.Printf("Checking ICP service health before provisioning for tenant: %s", req.TenantID)

	// Create a fresh context with its own timeout
	icpCtx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Step 1: Check ICP health first
	if err := s.icpClient.HealthCheck(icpCtx); err != nil {
		return nil, fmt.Errorf("ICP health check failed: %w", err)
	}

	log.Printf("ICP service is healthy, proceeding with PKI provisioning for tenant: %s", req.TenantID)

	// Step 2: Provision PKI
	resp, err := s.icpClient.ProvisionPKI(icpCtx, req)
	fmt.Println(resp)
	if err != nil {
		return nil, fmt.Errorf("ICP provisioning failed: %w", err)
	}

	log.Printf("ICP provisioning successful - PKI Mount: %s", resp.PKIMount)
	return resp, nil
}

// RetryPKIProvisioning retries PKI provisioning for a failed tenant
func (s *ICPProvisioningService) RetryPKIProvisioning(ctx context.Context, tenantID, commonName, domain string) (*icp.ProvisionPKIResponse, error) {
	log.Printf("Retrying PKI provisioning for tenant: %s", tenantID)

	req := &icp.ProvisionPKIRequest{
		TenantID:   tenantID,
		CommonName: commonName,
		Domain:     domain,
		TTL:        "87600h", // 10 years
		MaxTTL:     "24h",
	}

	return s.ProvisionPKI(ctx, req)
}

// UpdateTenantStatusInICP updates tenant status in ICP service
func (s *ICPProvisioningService) UpdateTenantStatusInICP(ctx context.Context, tenantID, status string) error {
	log.Printf("Updating tenant status in ICP: %s -> %s", tenantID, status)

	if err := s.icpClient.UpdateTenantStatus(ctx, tenantID, status); err != nil {
		log.Printf("Warning: Failed to update tenant status in ICP: %v", err)
		// Don't fail the operation - can be retried later
		return nil
	}

	return nil
}

// HealthCheck checks if ICP service is reachable
func (s *ICPProvisioningService) HealthCheck(ctx context.Context) error {
	return s.icpClient.HealthCheck(ctx)
}

// GenerateTenantDatabaseURL generates a database URL for a tenant database
func GenerateTenantDatabaseURL(tenantDBName string) string {
	cfg := config.GetConfig()
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		cfg.DBHost,
		cfg.DBUser,
		cfg.DBPassword,
		tenantDBName,
		cfg.DBPort,
	)
}
