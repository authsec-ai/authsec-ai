-- Migration 103: Add permissions for User Flow Service
-- Fixed to use production schema (no resources table, permissions uses tenant_id/resource/action)
-- Fixed: uses check-before-insert instead of ON CONFLICT ON CONSTRAINT (constraint may not exist yet)
-- Fixed: removed full_permission_string column (added by migration 109, not available yet)

DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (sys_tenant, sys_tenant, 'system@authsec.local', 'system.authsec.dev', 'System', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Ensure users:delete permission exists
    IF NOT EXISTS (
        SELECT 1 FROM permissions
        WHERE tenant_id = sys_tenant AND resource = 'users' AND action = 'delete'
    ) THEN
        INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
        VALUES (gen_random_uuid(), sys_tenant, 'users', 'delete', 'Delete a user', NOW());
    END IF;

    -- Ensure users:read permission exists
    IF NOT EXISTS (
        SELECT 1 FROM permissions
        WHERE tenant_id = sys_tenant AND resource = 'users' AND action = 'read'
    ) THEN
        INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
        VALUES (gen_random_uuid(), sys_tenant, 'users', 'read', 'Read user information', NOW());
    END IF;

    -- Ensure users:write permission exists
    IF NOT EXISTS (
        SELECT 1 FROM permissions
        WHERE tenant_id = sys_tenant AND resource = 'users' AND action = 'write'
    ) THEN
        INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
        VALUES (gen_random_uuid(), sys_tenant, 'users', 'write', 'Create and update users', NOW());
    END IF;

    -- Assign permissions to admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r, permissions p
    WHERE r.name = 'admin' AND r.tenant_id = sys_tenant
      AND p.tenant_id = sys_tenant
      AND p.resource = 'users'
      AND p.action IN ('delete', 'read', 'write')
    ON CONFLICT DO NOTHING;
END $$;
