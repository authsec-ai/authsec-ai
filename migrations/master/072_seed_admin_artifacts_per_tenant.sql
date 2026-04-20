-- Migration 116: Seed per-tenant admin role, permissions, scope, and bindings
-- This ensures every tenant has an admin role with CRUD permissions across core resources
-- and a tenant-wide role binding for an available admin user.

DO $$
DECLARE
    t RECORD;
    admin_role_id UUID;
    admin_scope_id UUID;
    admin_user_id UUID;
    res TEXT;
    act TEXT;
BEGIN
    -- Only run on master DB (tenant_mappings exists only there)
    IF to_regclass('public.tenant_mappings') IS NULL THEN
        RAISE NOTICE 'Skipping admin seed (116) because this is not the master database';
        RETURN;
    END IF;

    FOR t IN SELECT COALESCE(tenant_id, id) AS tenant_id FROM tenants LOOP
        -- Ensure admin role exists per tenant
        INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
        VALUES (gen_random_uuid(), t.tenant_id, 'admin', 'Tenant administrator', TRUE, NOW())
        ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO UPDATE SET description = EXCLUDED.description
        RETURNING id INTO admin_role_id;

        IF admin_role_id IS NULL THEN
            SELECT id INTO admin_role_id FROM roles WHERE tenant_id = t.tenant_id AND name = 'admin' LIMIT 1;
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
            'delegation-policies'
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

        -- Admin access guard for route-level checks
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

        -- Ensure admin scope and map all permissions to it
        INSERT INTO scopes (id, tenant_id, name, description, created_at)
        VALUES (gen_random_uuid(), t.tenant_id, 'admin', 'Administrator scope with full access', NOW())
        ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING
        RETURNING id INTO admin_scope_id;

        IF admin_scope_id IS NULL THEN
            SELECT id INTO admin_scope_id FROM scopes WHERE tenant_id = t.tenant_id AND name = 'admin' LIMIT 1;
        END IF;

        -- Note: scope_permissions table was dropped in migration 050
        -- Scope-permission mappings are now managed via scope_resource_mappings table

        -- Bind an available admin user to the admin role (tenant-wide)
        SELECT id INTO admin_user_id
        FROM users
        WHERE tenant_id = t.tenant_id
        ORDER BY created_at ASC
        LIMIT 1;

        IF admin_user_id IS NOT NULL THEN
            -- Note: user_roles table is deprecated, using role_bindings instead
            INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
            SELECT gen_random_uuid(), t.tenant_id, admin_user_id, admin_role_id, NULL, NULL, NOW(), NOW()
            WHERE NOT EXISTS (
                SELECT 1 FROM role_bindings
                WHERE tenant_id = t.tenant_id
                  AND user_id = admin_user_id
                  AND role_id = admin_role_id
                  AND scope_type IS NULL
                  AND scope_id IS NULL
            );
        END IF;
    END LOOP;
END
$$;
