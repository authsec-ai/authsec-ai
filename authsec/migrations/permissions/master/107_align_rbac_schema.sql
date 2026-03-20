-- Migration 107: Align RBAC tables with reference schema (tenant-scoped)
-- Fixed to backfill null values before setting NOT NULL

DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (sys_tenant, sys_tenant, 'system@authsec.local', 'system.authsec.dev', 'System', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Backfill NULL tenant_id values before setting NOT NULL
    UPDATE roles SET tenant_id = sys_tenant WHERE tenant_id IS NULL;
    UPDATE permissions SET tenant_id = sys_tenant WHERE tenant_id IS NULL;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'scopes') THEN
        EXECUTE 'UPDATE scopes SET tenant_id = $1 WHERE tenant_id IS NULL' USING sys_tenant;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'role_bindings') THEN
        EXECUTE 'UPDATE role_bindings SET tenant_id = $1 WHERE tenant_id IS NULL' USING sys_tenant;
    END IF;

    -- Now set NOT NULL on tenant_id columns
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'roles' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE roles ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'permissions' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE permissions ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'scopes' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE scopes ALTER COLUMN tenant_id SET NOT NULL;
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'role_bindings' AND column_name = 'tenant_id' AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE role_bindings ALTER COLUMN tenant_id SET NOT NULL;
    END IF;
END $$;

-- Ensure unique constraints
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_tenant_name_key') THEN
        ALTER TABLE roles ADD CONSTRAINT roles_tenant_name_key UNIQUE (tenant_id, name);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'permissions_tenant_resource_action_key') THEN
        ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_resource_action_key UNIQUE (tenant_id, resource, action);
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'scopes') THEN
        IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'scopes_tenant_name_key') THEN
            ALTER TABLE scopes ADD CONSTRAINT scopes_tenant_name_key UNIQUE (tenant_id, name);
        END IF;
    END IF;
END $$;
