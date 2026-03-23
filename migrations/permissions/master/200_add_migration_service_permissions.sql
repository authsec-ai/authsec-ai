-- Migration 200: RBAC permissions for authsec-migration service
-- Fixed to use production permissions schema (tenant_id, resource, action) instead of old (resource_id, action)

DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (sys_tenant, sys_tenant, 'system@authsec.local', 'system.authsec.dev', 'System', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Create migrations permissions
    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES
        (gen_random_uuid(), sys_tenant, 'migrations', 'admin', 'Full admin access to migration operations', 'migrations:admin', NOW()),
        (gen_random_uuid(), sys_tenant, 'migrations', 'run', 'Execute database migrations', 'migrations:run', NOW()),
        (gen_random_uuid(), sys_tenant, 'migrations', 'view', 'View migration status and history', 'migrations:view', NOW()),
        (gen_random_uuid(), sys_tenant, 'migrations', 'create_tenant_db', 'Create new tenant databases', 'migrations:create_tenant_db', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    -- Assign migration admin permissions to super_admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT ro.id, p.id
    FROM roles ro
    CROSS JOIN permissions p
    WHERE ro.name = 'super_admin' AND ro.tenant_id = sys_tenant
      AND p.tenant_id = sys_tenant AND p.resource = 'migrations'
    ON CONFLICT DO NOTHING;

    -- Assign migration permissions to admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT ro.id, p.id
    FROM roles ro
    CROSS JOIN permissions p
    WHERE ro.name = 'admin' AND ro.tenant_id = sys_tenant
      AND p.tenant_id = sys_tenant AND p.resource = 'migrations'
      AND p.action IN ('admin', 'run', 'create_tenant_db')
    ON CONFLICT DO NOTHING;
END $$;
