-- Seed permissions for external-service using the action-based permission schema
DO $$
DECLARE
    role_permissions_has_tenant BOOLEAN := FALSE;
    role_permissions_has_created_at BOOLEAN := FALSE;
    t RECORD;
BEGIN
    -- Only run on master DB (tenant_mappings exists only there)
    IF to_regclass('public.tenant_mappings') IS NULL THEN
        RAISE NOTICE 'Skipping external-service permission seed (117) because this is not the master database';
        RETURN;
    END IF;

    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'role_permissions' AND column_name = 'tenant_id'
    ) INTO role_permissions_has_tenant;
    SELECT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'role_permissions' AND column_name = 'created_at'
    ) INTO role_permissions_has_created_at;

    FOR t IN SELECT COALESCE(tenant_id, id) AS tenant_id FROM tenants LOOP
        -- Seed action-based permissions

        -- Seed action-based permissions
        IF to_regclass('public.permissions') IS NOT NULL THEN
            INSERT INTO permissions (id, resource, action, description, tenant_id, created_at)
            SELECT gen_random_uuid(), 'external-service', 'create', 'Create an external service', t.tenant_id, now()
            WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'external-service' AND action = 'create' AND tenant_id = t.tenant_id);

            INSERT INTO permissions (id, resource, action, description, tenant_id, created_at)
            SELECT gen_random_uuid(), 'external-service', 'read', 'Read/list external services', t.tenant_id, now()
            WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'external-service' AND action = 'read' AND tenant_id = t.tenant_id);

            INSERT INTO permissions (id, resource, action, description, tenant_id, created_at)
            SELECT gen_random_uuid(), 'external-service', 'update', 'Update an external service', t.tenant_id, now()
            WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'external-service' AND action = 'update' AND tenant_id = t.tenant_id);

            INSERT INTO permissions (id, resource, action, description, tenant_id, created_at)
            SELECT gen_random_uuid(), 'external-service', 'delete', 'Delete an external service', t.tenant_id, now()
            WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'external-service' AND action = 'delete' AND tenant_id = t.tenant_id);

            INSERT INTO permissions (id, resource, action, description, tenant_id, created_at)
            SELECT gen_random_uuid(), 'external-service', 'credentials', 'Read stored credentials for a service', t.tenant_id, now()
            WHERE NOT EXISTS (SELECT 1 FROM permissions WHERE resource = 'external-service' AND action = 'credentials' AND tenant_id = t.tenant_id);
        ELSE
            RAISE NOTICE 'permissions table not found; skipping permission seed';
            CONTINUE;
        END IF;

        -- Map permissions to roles when available
        IF to_regclass('public.roles') IS NOT NULL AND to_regclass('public.role_permissions') IS NOT NULL THEN
            -- Ensure base roles exist
            INSERT INTO roles (id, tenant_id, name, description, created_at)
            SELECT gen_random_uuid(), t.tenant_id, 'admin', 'Administrator role with full access', now()
            WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'admin' AND tenant_id = t.tenant_id);

            INSERT INTO roles (id, tenant_id, name, description, created_at)
            SELECT gen_random_uuid(), t.tenant_id, 'user', 'Standard user role', now()
            WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'user' AND tenant_id = t.tenant_id);

            -- Grant all external-service permissions to admin
            IF role_permissions_has_tenant THEN
                IF role_permissions_has_created_at THEN
                    INSERT INTO role_permissions (role_id, permission_id, tenant_id, created_at)
                    SELECT r.id, p.id, t.tenant_id, now()
                    FROM roles r, permissions p
                    WHERE r.name = 'admin'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                ELSE
                    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
                    SELECT r.id, p.id, t.tenant_id
                    FROM roles r, permissions p
                    WHERE r.name = 'admin'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                END IF;
            ELSE
                IF role_permissions_has_created_at THEN
                    INSERT INTO role_permissions (role_id, permission_id, created_at)
                    SELECT r.id, p.id, now()
                    FROM roles r, permissions p
                    WHERE r.name = 'admin'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                ELSE
                    INSERT INTO role_permissions (role_id, permission_id)
                    SELECT r.id, p.id
                    FROM roles r, permissions p
                    WHERE r.name = 'admin'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                END IF;
            END IF;

            -- Grant read-only access to standard users
            IF role_permissions_has_tenant THEN
                IF role_permissions_has_created_at THEN
                    INSERT INTO role_permissions (role_id, permission_id, tenant_id, created_at)
                    SELECT r.id, p.id, t.tenant_id, now()
                    FROM roles r, permissions p
                    WHERE r.name = 'user'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.action = 'read'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                ELSE
                    INSERT INTO role_permissions (role_id, permission_id, tenant_id)
                    SELECT r.id, p.id, t.tenant_id
                    FROM roles r, permissions p
                    WHERE r.name = 'user'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.action = 'read'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                END IF;
            ELSE
                IF role_permissions_has_created_at THEN
                    INSERT INTO role_permissions (role_id, permission_id, created_at)
                    SELECT r.id, p.id, now()
                    FROM roles r, permissions p
                    WHERE r.name = 'user'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.action = 'read'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                ELSE
                    INSERT INTO role_permissions (role_id, permission_id)
                    SELECT r.id, p.id
                    FROM roles r, permissions p
                    WHERE r.name = 'user'
                      AND r.tenant_id = t.tenant_id
                      AND p.resource = 'external-service'
                      AND p.action = 'read'
                      AND p.tenant_id = t.tenant_id
                      AND NOT EXISTS (
                          SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
                      );
                END IF;
            END IF;
        ELSE
            RAISE NOTICE 'role permissions tables not found; skipping role mappings';
        END IF;
    END LOOP;
END
$$;
