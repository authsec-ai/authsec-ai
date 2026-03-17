-- Migration: Add admin:access permission and map to admin roles
-- Run this on the TENANT database
-- This script is idempotent (safe to run multiple times)
-- NOTE: This migration is executed via authsec-migration service which provides tenant_id context
-- NOTE: The migration runner wraps each migration in its own transaction, so no explicit BEGIN/COMMIT needed.

-- ============================================
-- STEP 1: Create admin permissions if they don't exist
-- NOTE: tenant_id should be injected by migration runner or provided via environment
-- ============================================

-- Get tenant_id from tenants table (tenant DB should have reference to its tenant)
-- If tenants table doesn't exist in tenant DB, this migration should be modified
-- to accept tenant_id as parameter

DO $$
DECLARE
    v_tenant_id UUID;
BEGIN
    -- Try to get tenant_id from tenants table if it exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants') THEN
        SELECT id INTO v_tenant_id FROM tenants LIMIT 1;
    END IF;

    -- Skip if no tenant exists (permissions.tenant_id is NOT NULL)
    IF v_tenant_id IS NULL THEN
        RAISE NOTICE 'No tenant found, skipping admin permission seeding';
        RETURN;
    END IF;

    -- Create admin:access permission
    INSERT INTO permissions (id, tenant_id, resource, action, description)
    SELECT gen_random_uuid(), v_tenant_id, 'admin', 'access', 'Access admin panel'
    WHERE NOT EXISTS (
        SELECT 1 FROM permissions p 
        WHERE (v_tenant_id IS NULL OR p.tenant_id = v_tenant_id) 
        AND p.resource = 'admin' AND p.action = 'access'
    );

    -- Create admin:read permission
    INSERT INTO permissions (id, tenant_id, resource, action, description)
    SELECT gen_random_uuid(), v_tenant_id, 'admin', 'read', 'Read admin data'
    WHERE NOT EXISTS (
        SELECT 1 FROM permissions p 
        WHERE (v_tenant_id IS NULL OR p.tenant_id = v_tenant_id)
        AND p.resource = 'admin' AND p.action = 'read'
    );

    -- Create admin:write permission
    INSERT INTO permissions (id, tenant_id, resource, action, description)
    SELECT gen_random_uuid(), v_tenant_id, 'admin', 'write', 'Write admin data'
    WHERE NOT EXISTS (
        SELECT 1 FROM permissions p 
        WHERE (v_tenant_id IS NULL OR p.tenant_id = v_tenant_id)
        AND p.resource = 'admin' AND p.action = 'write'
    );

    -- Create admin:delete permission
    INSERT INTO permissions (id, tenant_id, resource, action, description)
    SELECT gen_random_uuid(), v_tenant_id, 'admin', 'delete', 'Delete admin data'
    WHERE NOT EXISTS (
        SELECT 1 FROM permissions p 
        WHERE (v_tenant_id IS NULL OR p.tenant_id = v_tenant_id)
        AND p.resource = 'admin' AND p.action = 'delete'
    );
END $$;

-- ============================================
-- STEP 2: Grant admin permissions to 'admin' role
-- ============================================
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name = 'admin'
  AND p.resource = 'admin'
  AND r.tenant_id = p.tenant_id
ON CONFLICT DO NOTHING;

-- ============================================
-- STEP 3: Grant admin permissions to any role containing 'admin' in name
-- ============================================
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.name ILIKE '%admin%'
  AND p.resource = 'admin'
  AND r.tenant_id = p.tenant_id
ON CONFLICT DO NOTHING;

-- ============================================
-- STEP 4: Grant admin:access to all existing roles (fallback for first-time users)
-- This ensures any user with ANY role can at least access the admin panel
-- Comment this out if you want stricter access control
-- ============================================
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON r.tenant_id = p.tenant_id
WHERE p.resource = 'admin' AND p.action = 'access'
ON CONFLICT DO NOTHING;

-- ============================================
-- VERIFICATION: Check that permissions are mapped
-- ============================================
-- Run this query to verify:
-- SELECT r.name as role_name, p.resource, p.action
-- FROM role_permissions rp
-- JOIN roles r ON rp.role_id = r.id
-- JOIN permissions p ON rp.permission_id = p.id
-- WHERE p.resource = 'admin'
-- ORDER BY r.name, p.action;
