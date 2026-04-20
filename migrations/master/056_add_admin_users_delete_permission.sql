-- Migration: Add DELETE permission for users resource (modern RBAC schema)
DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists
    INSERT INTO tenants (id, tenant_id, name, email, tenant_domain, created_at)
    VALUES (sys_tenant, sys_tenant, 'System', 'system@authsec.internal', 'system', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Ensure delete permission exists
    INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
    VALUES (
        gen_random_uuid(),
        sys_tenant,
        'users',
        'delete',
        'Delete a user',
        NOW()
    )
    ON CONFLICT (tenant_id, resource, action) DO NOTHING;

    -- Link to admin role
    WITH ar AS (
        SELECT id FROM roles WHERE tenant_id = sys_tenant AND name = 'admin' LIMIT 1
    ), perm AS (
        SELECT id FROM permissions WHERE tenant_id = sys_tenant AND resource = 'users' AND action = 'delete' LIMIT 1
    )
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT ar.id, perm.id FROM ar, perm
    ON CONFLICT DO NOTHING;
END $$;
