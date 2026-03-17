-- Migration 105: Allow Global RBAC (Nullable Tenant ID)
-- Purpose: Update RBAC tables to support Global/System entities by allowing NULL tenant_id.
-- This unifies the Admin/Global RBAC with the Tenant RBAC schema.
--
-- NOTE: Migration 107 reverts tenant_id back to NOT NULL. This migration exists for 
-- historical compatibility with databases that were migrated during the intermediate period.
-- For new deployments, the net effect of 105→107 is: tenant_id remains NOT NULL with 
-- additional partial indexes for global entities.

-- BEGIN; (removed - app manages transactions)

-- 1. Roles - Allow NULL tenant_id for global roles
-- Using DROP NOT NULL only if column has NOT NULL constraint
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'roles' AND column_name = 'tenant_id' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE roles ALTER COLUMN tenant_id DROP NOT NULL;
    END IF;
END $$;

-- Partial indexes for global (NULL tenant_id) uniqueness
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_global_name ON roles (name) WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_global_id ON roles (id) WHERE tenant_id IS NULL;

-- 2. Permissions - Allow NULL tenant_id for global permissions
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'permissions' AND column_name = 'tenant_id' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE permissions ALTER COLUMN tenant_id DROP NOT NULL;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_global_resource_action ON permissions (resource, action) WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_global_id ON permissions (id) WHERE tenant_id IS NULL;

-- 3. Scopes - Allow NULL tenant_id for global scopes
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'scopes' AND column_name = 'tenant_id' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE scopes ALTER COLUMN tenant_id DROP NOT NULL;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_scopes_global_name ON scopes (name) WHERE tenant_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_scopes_global_id ON scopes (id) WHERE tenant_id IS NULL;

-- 4. Service Accounts - Allow NULL tenant_id for global service accounts
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'service_accounts' AND column_name = 'tenant_id' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE service_accounts ALTER COLUMN tenant_id DROP NOT NULL;
    END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS idx_service_accounts_global_id ON service_accounts (id) WHERE tenant_id IS NULL;

-- 5. Role Bindings - Allow NULL tenant_id for global bindings
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'role_bindings' AND column_name = 'tenant_id' AND is_nullable = 'NO'
    ) THEN
        ALTER TABLE role_bindings ALTER COLUMN tenant_id DROP NOT NULL;
    END IF;
END $$;

-- COMMIT; (removed - app manages transactions)
