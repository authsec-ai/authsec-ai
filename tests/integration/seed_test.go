//go:build integration

package integration

import (
	"fmt"
	"log"

	"github.com/authsec-ai/authsec/config"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func seedTestData() error {
	db := config.Database.DB

	testTenantID = uuid.New()
	testAdminUserID = uuid.New()
	testEndUserID = uuid.New()
	testClientID = uuid.New()
	testProjectID = uuid.New()
	testAdminRoleID = uuid.New()

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}

	// 1. Insert test tenant
	_, err = db.Exec(`
		INSERT INTO tenants (id, tenant_id, email, tenant_domain, tenant_db, name, status, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, 'Test Tenant', 'active', NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`,
		testTenantID, testAdminEmail, testTenantDomain, testDBName)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}

	// 2. Insert admin user
	_, err = db.Exec(`
		INSERT INTO users (id, client_id, tenant_id, project_id, email, password_hash,
			tenant_domain, provider, active, created_at, updated_at, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'local', true, NOW(), NOW(), 'Test Admin')
		ON CONFLICT (id) DO NOTHING`,
		testAdminUserID, testClientID, testTenantID, testProjectID,
		testAdminEmail, string(hashedPassword), testTenantDomain)
	if err != nil {
		return fmt.Errorf("insert admin user: %w", err)
	}

	// 3. Insert end user
	_, err = db.Exec(`
		INSERT INTO users (id, client_id, tenant_id, project_id, email, password_hash,
			tenant_domain, provider, active, created_at, updated_at, name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, 'local', true, NOW(), NOW(), 'Test EndUser')
		ON CONFLICT (id) DO NOTHING`,
		testEndUserID, testClientID, testTenantID, testProjectID,
		testEndUserEmail, string(hashedPassword), testTenantDomain)
	if err != nil {
		return fmt.Errorf("insert end user: %w", err)
	}

	// 4. Insert project
	_, _ = db.Exec(`
		INSERT INTO projects (id, name, description, user_id, tenant_id, client_id, active, created_at)
		VALUES ($1, 'Test Project', 'Integration test project', $2, $3, $4, true, NOW())
		ON CONFLICT (id) DO NOTHING`,
		testProjectID, testAdminUserID, testTenantID, testClientID)

	// 5. Insert client
	_, _ = db.Exec(`
		INSERT INTO clients (id, client_id, tenant_id, project_id, owner_id, org_id, name, email,
			status, active, created_at, updated_at)
		VALUES ($1, $1, $2, $3, $4, $2, 'Test Client', $5, 'Active', true, NOW(), NOW())
		ON CONFLICT (id) DO NOTHING`,
		testClientID, testTenantID, testProjectID, testAdminUserID, testAdminEmail)

	// 6. Create admin role
	_, _ = db.Exec(`
		INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
		VALUES ($1, $2, 'admin', 'Admin role', true, NOW())
		ON CONFLICT DO NOTHING`,
		testAdminRoleID, testTenantID)

	// 7. Create user role for end users
	userRoleID := uuid.New()
	_, _ = db.Exec(`
		INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
		VALUES ($1, $2, 'user', 'User role', false, NOW())
		ON CONFLICT DO NOTHING`,
		userRoleID, testTenantID)

	// 8. Create role binding for admin user
	_, _ = db.Exec(`
		INSERT INTO role_bindings (id, tenant_id, user_id, role_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT DO NOTHING`,
		uuid.New(), testTenantID, testAdminUserID, testAdminRoleID)

	// 9. Create role binding for end user
	_, _ = db.Exec(`
		INSERT INTO role_bindings (id, tenant_id, user_id, role_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		ON CONFLICT DO NOTHING`,
		uuid.New(), testTenantID, testEndUserID, userRoleID)

	// 10. Create permissions for test tenant
	permResources := []struct{ resource, action string }{
		{"admin", "access"}, {"admin", "read"}, {"admin", "write"}, {"admin", "manage"}, {"admin", "delete"},
		{"users", "read"}, {"users", "write"}, {"users", "delete"}, {"users", "manage"}, {"users", "active"},
		{"tenants", "read"}, {"tenants", "write"}, {"tenants", "delete"}, {"tenants", "manage"},
		{"clients", "read"}, {"clients", "write"}, {"clients", "admin"},
		{"roles", "manage"}, {"permissions", "manage"},
		{"external-service", "create"}, {"external-service", "read"},
		{"external-service", "update"}, {"external-service", "delete"},
		{"external-service", "credentials"},
		{"migrations", "admin"},
	}
	for _, p := range permResources {
		permID := uuid.New()
		_, _ = db.Exec(`
			INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT DO NOTHING`,
			permID, testTenantID, p.resource, p.action,
			p.resource+":"+p.action, p.resource+":"+p.action)
		// Link to admin role
		_, _ = db.Exec(`INSERT INTO role_permissions (role_id, permission_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			testAdminRoleID, permID)
	}

	// 11. Create tenant_mappings entry
	_, _ = db.Exec(`
		INSERT INTO tenant_mappings (id, tenant_id, client_id, created_at)
		VALUES ($1, $2, $3, NOW())
		ON CONFLICT DO NOTHING`,
		uuid.New(), testTenantID, testClientID)

	log.Printf("Seeded test data: tenant=%s admin=%s enduser=%s", testTenantID, testAdminUserID, testEndUserID)
	return nil
}
