-- Migration 201: RBAC permission for template-based tenant DB creation
-- Requires JWT + admin role (not service token)

DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (sys_tenant, sys_tenant, 'system@authsec.local', 'system.authsec.dev', 'System', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Create template cloning permission
    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES
        (gen_random_uuid(), sys_tenant, 'migrations', 'create_tenant_from_template', 'Create tenant databases by cloning golden template', 'migrations:create_tenant_from_template', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    -- Assign to super_admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT ro.id, p.id
    FROM roles ro
    CROSS JOIN permissions p
    WHERE ro.name = 'super_admin' AND ro.tenant_id = sys_tenant
      AND p.tenant_id = sys_tenant AND p.resource = 'migrations'
      AND p.action = 'create_tenant_from_template'
    ON CONFLICT DO NOTHING;

    -- Assign to admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT ro.id, p.id
    FROM roles ro
    CROSS JOIN permissions p
    WHERE ro.name = 'admin' AND ro.tenant_id = sys_tenant
      AND p.tenant_id = sys_tenant AND p.resource = 'migrations'
      AND p.action = 'create_tenant_from_template'
    ON CONFLICT DO NOTHING;
END $$;
