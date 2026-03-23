-- Tenant Migration 003: Enforce Scoped RBAC Schema
-- Purpose: Implement strict multi-tenant RBAC with composite keys, scoped permissions, and service accounts.
-- This migration ensures the tenant database matches the strict requirements.
-- Note: This migration is designed to run via psql which handles multi-statement transactions.

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Drop existing tables to ensure clean schema enforcement
DROP TABLE IF EXISTS grant_audit CASCADE;
DROP TABLE IF EXISTS role_bindings CASCADE;
DROP TABLE IF EXISTS scope_permissions CASCADE;
DROP TABLE IF EXISTS scopes CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS service_accounts CASCADE;

-- Ensure users table has a composite unique constraint on (tenant_id, id)
-- required for the composite FK in role_bindings below.
-- The tenant template may not have included this constraint for older tenants.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'users_tenant_id_id_unique'
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_tenant_id_id_unique UNIQUE (tenant_id, id);
    END IF;
END $$;

-- 1. Service Accounts (New)
CREATE TABLE service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    -- Constraint for composite FKs
    CONSTRAINT service_accounts_tenant_id_id_key UNIQUE (tenant_id, id)
);

-- 2. Roles (Definitions)
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name),
    UNIQUE (tenant_id, id)
);

-- 3. Permissions (Atomic Capabilities)
CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    resource TEXT NOT NULL, -- e.g. 'project'
    action TEXT NOT NULL,   -- e.g. 'delete'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, resource, action),
    UNIQUE (tenant_id, id)
);

-- 4. Role Permissions (M:N Linkage)
CREATE TABLE role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 5. API Scopes (External Contracts)
CREATE TABLE scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- e.g. 'files:read'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (tenant_id, name),
    UNIQUE (tenant_id, id)
);

-- 6. Scope Permissions (M:N Linkage)
CREATE TABLE scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id),
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 7. Role Bindings (The Core Assignment)
CREATE TABLE role_bindings (
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

    -- Strict Tenant Isolation Constraints
    -- Note: We use composite keys referenced from users/service_accounts/roles to ensure tenant_id matches
    FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id),
    FOREIGN KEY (tenant_id, service_account_id) REFERENCES service_accounts(tenant_id, id),
    FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id),
    FOREIGN KEY (created_by) REFERENCES users(id),

    -- Logic Constraints
    CONSTRAINT chk_scope_integrity CHECK (
        (scope_type IS NULL AND scope_id IS NULL) OR
        (scope_type IS NOT NULL AND scope_id IS NOT NULL)
    ),

    -- Ensure exactly one principal is set
    CONSTRAINT check_principal CHECK (
        (user_id IS NOT NULL AND service_account_id IS NULL) OR
        (user_id IS NULL AND service_account_id IS NOT NULL)
    )
);
