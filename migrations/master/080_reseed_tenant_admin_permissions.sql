-- Migration 125: Re-seed admin permissions for all tenants
-- This is a fix for tenants created via OIDC that didn't have permissions seeded
-- It mirrors the logic from admin_seed_repository.go EnsureAdminRoleAndPermissions

DO $$
DECLARE
    t RECORD;
    admin_role_id UUID;
    admin_scope_id UUID;
    res TEXT;
    act TEXT;
    perm_count INT;
    seeded_count INT := 0;
BEGIN
    -- Only run on master DB (tenant_mappings exists only there)
    IF to_regclass('public.tenant_mappings') IS NULL THEN
        RAISE NOTICE 'Skipping admin reseed (125) because this is not the master database';
        RETURN;
    END IF;

    RAISE NOTICE 'Starting admin permission reseed for all tenants...';

    FOR t IN SELECT COALESCE(tenant_id, id) AS tenant_id, name FROM tenants LOOP
        -- Check if this tenant has permissions seeded
        SELECT COUNT(*) INTO perm_count
        FROM role_permissions rp
        JOIN roles r ON rp.role_id = r.id
        WHERE r.tenant_id = t.tenant_id AND r.name = 'admin';

        IF perm_count >= 50 THEN
            RAISE NOTICE 'Tenant % (%) already has % admin permissions, skipping', t.name, t.tenant_id, perm_count;
            CONTINUE;
        END IF;

        RAISE NOTICE 'Tenant % (%) only has % admin permissions, seeding...', t.name, t.tenant_id, perm_count;

        -- Ensure admin role exists per tenant
        INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
        VALUES (gen_random_uuid(), t.tenant_id, 'admin', 'Tenant administrator', TRUE, NOW())
        ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO UPDATE SET description = EXCLUDED.description
        RETURNING id INTO admin_role_id;

        IF admin_role_id IS NULL THEN
            SELECT id INTO admin_role_id FROM roles WHERE tenant_id = t.tenant_id AND name = 'admin' LIMIT 1;
        END IF;

        IF admin_role_id IS NULL THEN
            RAISE WARNING 'Could not find or create admin role for tenant %', t.tenant_id;
            CONTINUE;
        END IF;

        -- Ensure permissions for all exposed controllers/resources
        FOR res IN SELECT UNNEST(ARRAY[
            'admin-access',
            'users',
            'tenants',
            'projects',
            'roles',
            'permissions',
            'scopes',
            'role-bindings',
            'policy',
            'groups',
            'sync',
            'sync-configs',
            'oidc',
            'endusers',
            'clients',
            'user-endusers',
            'user-rbac-roles',
            'user-rbac-permissions',
            'user-rbac-scopes',
            'user-permissions',
            'user-groups',
            'user-clients',
            'user-projects',
            'health',
            'api-scopes',
            'audit',
            'session',
            'mfa'
        ]) LOOP
            FOR act IN SELECT UNNEST(ARRAY['create','read','update','delete','manage','access']) LOOP
                INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
                VALUES (gen_random_uuid(), t.tenant_id, res, act, CONCAT(res, ' ', act), NOW())
                ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

                INSERT INTO role_permissions (role_id, permission_id)
                SELECT admin_role_id, p.id
                FROM permissions p
                WHERE p.tenant_id = t.tenant_id
                  AND p.resource = res
                  AND p.action = act
                ON CONFLICT DO NOTHING;
            END LOOP;
        END LOOP;

        -- Admin access guard for route-level checks (critical permission)
        INSERT INTO permissions (id, tenant_id, resource, action, description, created_at)
        VALUES (gen_random_uuid(), t.tenant_id, 'admin', 'access', 'Administrative access gate', NOW())
        ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

        INSERT INTO role_permissions (role_id, permission_id)
        SELECT admin_role_id, p.id
        FROM permissions p
        WHERE p.tenant_id = t.tenant_id
          AND p.resource = 'admin'
          AND p.action = 'access'
        ON CONFLICT DO NOTHING;

        -- Ensure admin scope exists
        INSERT INTO scopes (id, tenant_id, name, description, created_at)
        VALUES (gen_random_uuid(), t.tenant_id, 'admin', 'Administrator scope with full access', NOW())
        ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING
        RETURNING id INTO admin_scope_id;

        IF admin_scope_id IS NULL THEN
            SELECT id INTO admin_scope_id FROM scopes WHERE tenant_id = t.tenant_id AND name = 'admin' LIMIT 1;
        END IF;

        seeded_count := seeded_count + 1;
        RAISE NOTICE 'Successfully seeded admin permissions for tenant % (%)', t.name, t.tenant_id;
    END LOOP;

    RAISE NOTICE 'Admin permission reseed complete. Seeded % tenants.', seeded_count;
END
$$;
