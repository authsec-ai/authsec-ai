-- Migration: Add permissions for external-service
-- Executed per-tenant, assigning permissions to the existing 'admin' role.

DO $$
DECLARE
    v_tenant_id UUID;
BEGIN
    -- Get tenant_id from tenants table if it exists
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants') THEN
        SELECT id INTO v_tenant_id FROM tenants LIMIT 1;
    END IF;

    -- Skip if no tenant exists (permissions.tenant_id is NOT NULL)
    IF v_tenant_id IS NULL THEN
        RAISE NOTICE 'No tenant found, skipping external-service permission seeding';
        RETURN;
    END IF;

    -- STEP 1: Define Permissions for external-service
    INSERT INTO permissions (id, tenant_id, resource, action, description)
    VALUES
        (gen_random_uuid(), v_tenant_id, 'external-service', 'create',      'Create new external services'),
        (gen_random_uuid(), v_tenant_id, 'external-service', 'read',        'View external service details'),
        (gen_random_uuid(), v_tenant_id, 'external-service', 'update',      'Modify external service details'),
        (gen_random_uuid(), v_tenant_id, 'external-service', 'delete',      'Delete external services'),
        (gen_random_uuid(), v_tenant_id, 'external-service', 'credentials', 'View external service credentials')
    ON CONFLICT (tenant_id, resource, action) DO NOTHING;

    -- STEP 2: Grant permissions to Admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r CROSS JOIN permissions p
    WHERE r.name = 'admin' AND r.tenant_id = v_tenant_id
      AND p.resource = 'external-service' AND p.tenant_id = v_tenant_id
    ON CONFLICT DO NOTHING;
END $$;
