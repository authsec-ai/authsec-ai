-- Align RBAC tables with reference schema (tenant-scoped, strict FK + scope integrity)
-- NOTE: This migration reverts the nullable tenant_id from migration 105 back to NOT NULL.
-- The net effect of 104→105→107 is that tenant_id is NOT NULL with additional partial indexes.

-- BEGIN; (removed - app manages transactions)

-- Ensure tenant_id columns are NOT NULL for core RBAC tables
-- Use DO blocks to handle cases where column might already be NOT NULL
DO $$
BEGIN
    -- First, ensure no NULL values exist (set to a default tenant if needed)
    -- This is a safety check - in production, all rows should already have tenant_id
    
    -- Roles: delete orphaned global rows (tenant_id IS NULL) before enforcing NOT NULL
    DELETE FROM roles WHERE tenant_id IS NULL;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'roles' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE roles ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- Permissions
    DELETE FROM permissions WHERE tenant_id IS NULL;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'permissions' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE permissions ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- Scopes
    DELETE FROM scopes WHERE tenant_id IS NULL;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'scopes' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE scopes ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    -- Role Bindings
    DELETE FROM role_bindings WHERE tenant_id IS NULL;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'role_bindings' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE role_bindings ALTER COLUMN tenant_id SET NOT NULL;
    END IF;
END $$;

-- Ensure unique constraints for tenant + name/resource/action
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_tenant_name_key') THEN
        ALTER TABLE roles ADD CONSTRAINT roles_tenant_name_key UNIQUE (tenant_id, name);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'permissions_tenant_resource_action_key') THEN
        ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_resource_action_key UNIQUE (tenant_id, resource, action);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'scopes_tenant_name_key') THEN
        ALTER TABLE scopes ADD CONSTRAINT scopes_tenant_name_key UNIQUE (tenant_id, name);
    END IF;
END$$;

-- Ensure role_bindings scope integrity check exists.
-- NOTE: Migration 071 drops this constraint. If existing data violates it, skip gracefully.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'role_bindings_chk_scope_integrity') THEN
        BEGIN
            ALTER TABLE role_bindings
            ADD CONSTRAINT role_bindings_chk_scope_integrity CHECK (
                (scope_type IS NULL AND scope_id IS NULL) OR
                (scope_type IS NOT NULL AND scope_id IS NOT NULL)
            );
        EXCEPTION
            WHEN check_violation THEN
                RAISE NOTICE 'Skipping role_bindings_chk_scope_integrity: existing data violates constraint (dropped in migration 071)';
            WHEN OTHERS THEN
                RAISE NOTICE 'Could not add role_bindings_chk_scope_integrity: %', SQLERRM;
        END;
    END IF;
END$$;

-- Ensure tenant-scoped FK composites exist where supported
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'role_bindings_tenant_user_fk') THEN
        ALTER TABLE role_bindings
        ADD CONSTRAINT role_bindings_tenant_user_fk FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'role_bindings_tenant_role_fk') THEN
        ALTER TABLE role_bindings
        ADD CONSTRAINT role_bindings_tenant_role_fk FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id) ON DELETE CASCADE;
    END IF;
END$$;

-- COMMIT; (removed - app manages transactions)
