-- Migration 079: Ensure users:active permission exists (modern RBAC schema)
-- Fixed to match production tenants table (tenant_id NOT NULL, email NOT NULL, tenant_domain NOT NULL)
-- Fixed: uses check-before-insert instead of ON CONFLICT ON CONSTRAINT (constraint may not exist yet)

DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists with all required NOT NULL fields
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (sys_tenant, sys_tenant, 'system@authsec.local', 'system.authsec.dev', 'System', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Ensure users:active permission exists (idempotent without requiring named constraint)
    IF NOT EXISTS (
        SELECT 1 FROM permissions
        WHERE tenant_id = sys_tenant AND resource = 'users' AND action = 'active'
    ) THEN
        INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
        VALUES (gen_random_uuid(), sys_tenant, 'users', 'active', 'Activate/deactivate users', NOW());
    END IF;

    -- Link to admin role if present
    WITH ar AS (
        SELECT id FROM roles WHERE tenant_id = sys_tenant AND name = 'admin' LIMIT 1
    ), perm AS (
        SELECT id FROM permissions WHERE tenant_id = sys_tenant AND resource = 'users' AND action = 'active' LIMIT 1
    )
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT ar.id, perm.id FROM ar, perm
    WHERE ar.id IS NOT NULL AND perm.id IS NOT NULL
    ON CONFLICT DO NOTHING;
END $$;
