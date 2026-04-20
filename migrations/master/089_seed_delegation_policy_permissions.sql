-- Migration 144: Seed delegation-policies permissions for all tenants
-- Ensures the RBAC middleware allows access to the delegation-policies admin routes.

DO $$
DECLARE
    t RECORD;
    admin_role_id UUID;
    res TEXT;
    act TEXT;
BEGIN
    -- Only run on master DB (tenant_mappings exists only there)
    IF to_regclass('public.tenant_mappings') IS NULL THEN
        RAISE NOTICE 'Skipping delegation-policies permission seed (144) because this is not the master database';
        RETURN;
    END IF;

    FOR t IN SELECT COALESCE(tenant_id, id) AS tenant_id FROM tenants LOOP
        -- Get admin role for this tenant
        SELECT id INTO admin_role_id FROM roles WHERE tenant_id = t.tenant_id AND name = 'admin' LIMIT 1;

        -- Seed delegation-policies permissions
        FOR act IN SELECT UNNEST(ARRAY['create','read','update','delete','manage','access']) LOOP
            INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
            VALUES (gen_random_uuid(), t.tenant_id, 'delegation-policies', act, CONCAT('delegation-policies ', act), NOW())
            ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

            -- Bind to admin role if it exists
            IF admin_role_id IS NOT NULL THEN
                INSERT INTO role_permissions (role_id, permission_id)
                SELECT admin_role_id, p.id
                FROM permissions p
                WHERE p.tenant_id = t.tenant_id
                  AND p.resource = 'delegation-policies'
                  AND p.action = act
                ON CONFLICT DO NOTHING;
            END IF;
        END LOOP;
    END LOOP;
END
$$;
