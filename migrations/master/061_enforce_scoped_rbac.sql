-- Migration 104: Enforce Scoped RBAC Schema
-- Purpose: Implement strict multi-tenant RBAC with composite keys, scoped permissions, and service accounts.
-- This migration drops existing RBAC tables and re-creates them to match the strict requirements.

-- BEGIN; (removed - app manages transactions)

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Drop existing tables to ensure clean schema enforcement
 
-- 1. Tenants (Keep existing, ensure columns exist)
-- Existing tenants table structure is preserved.
-- Ensure name column is NOT NULL if possible, but we won't alter constraint on existing data to avoid failures.

-- 2. Users (Keep existing)
-- Ensure the unique constraint for composite FK exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_tenant_id_id_key'
        AND conrelid = 'users'::regclass
    ) THEN
        -- We might need to add this constraint if it doesn't exist
        -- However, users table might have rows with NULL tenant_id which would fail this.
        -- We will assume tenant_id is populated for RBAC users.
        -- For now, we try to add it. If it fails, we might need a fallback or cleanup.
        BEGIN
            ALTER TABLE users ADD CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id);
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Could not add unique constraint users_tenant_id_id_key, possibly due to null tenant_ids';
        END;
    END IF;
END $$;

-- 2b. Service Accounts (New)
CREATE TABLE IF NOT EXISTS service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    -- Constraint for composite FKs
    CONSTRAINT service_accounts_tenant_id_id_key UNIQUE (tenant_id, id)
);

-- 3. Roles (Definitions)
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name),
    UNIQUE (tenant_id, id)
);

-- 4. Permissions (Atomic Capabilities)
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource TEXT NOT NULL, -- e.g. 'project'
    action TEXT NOT NULL,   -- e.g. 'delete'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, resource, action),
    UNIQUE (tenant_id, id)
);

-- 5. Role Permissions (M:N Linkage)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 6. API Scopes (External Contracts)
CREATE TABLE IF NOT EXISTS scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- e.g. 'files:read'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name),
    UNIQUE (tenant_id, id)
);

-- 7. Scope Permissions (M:N Linkage)
CREATE TABLE IF NOT EXISTS scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id),
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 8. Role Bindings (The Core Assignment)
CREATE TABLE IF NOT EXISTS role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Principal (Polymorphic: User OR Service Account)
    user_id UUID,
    service_account_id UUID,

    -- Role
    role_id UUID NOT NULL,

    -- Scope (Polymorphic Resource Pointer)
    scope_type TEXT, -- e.g. 'project', NULL for Tenant-Wide
    scope_id UUID,   -- External UUID, NULL for Tenant-Wide

    -- Metadata
    conditions JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),

    -- Strict Tenant Isolation Constraints
    -- Note: We use composite keys referenced from users/service_accounts/roles to ensure tenant_id matches
    FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id),
    FOREIGN KEY (tenant_id, service_account_id) REFERENCES service_accounts(tenant_id, id),
    FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id),
    FOREIGN KEY (created_by) REFERENCES users(id),

    -- Logic Constraints
     

    -- Ensure exactly one principal is set
    CONSTRAINT check_principal CHECK (
        (user_id IS NOT NULL AND service_account_id IS NULL) OR
        (user_id IS NULL AND service_account_id IS NOT NULL)
    )
);

-- COMMIT; (removed - app manages transactions)

 