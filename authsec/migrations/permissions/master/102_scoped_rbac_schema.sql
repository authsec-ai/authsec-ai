-- Migration 102: Scoped RBAC Schema Implementation
-- Purpose: Implement strict multi-tenant RBAC with composite keys and scoped permissions

-- BEGIN; (removed - app manages transactions)

-- Ensure extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Update Users Table for Composite FK Support
-- We need a unique constraint on (tenant_id, id) to allow composite foreign keys referencing it
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'users_tenant_id_id_key' 
        AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users ADD CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id);
    END IF;
END $$;

-- 2. Service Accounts Table (New)
CREATE TABLE IF NOT EXISTS service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT service_accounts_tenant_id_id_key UNIQUE (tenant_id, id)
);

-- 3. Roles Table (Update or Create)
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT roles_tenant_name_key UNIQUE (tenant_id, name),
    CONSTRAINT roles_tenant_id_id_key UNIQUE (tenant_id, id)
);

-- Ensure columns exist if table already existed
DO $$
BEGIN
    ALTER TABLE roles ADD COLUMN IF NOT EXISTS is_system BOOLEAN DEFAULT false;
    
    -- Ensure unique constraints exist
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_tenant_name_key' AND conrelid = 'roles'::regclass) THEN
        ALTER TABLE roles ADD CONSTRAINT roles_tenant_name_key UNIQUE (tenant_id, name);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'roles_tenant_id_id_key' AND conrelid = 'roles'::regclass) THEN
        ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_id_key UNIQUE (tenant_id, id);
    END IF;
END $$;

-- 4. Permissions Table (Replaces old structure)
-- Old structure had role_id, scope_id, resource_id. New structure is atomic resource+action.
DROP TABLE IF EXISTS permissions CASCADE;

CREATE TABLE permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT permissions_tenant_resource_action_key UNIQUE (tenant_id, resource, action)
);

-- 5. Role Permissions (Many-to-Many)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id),
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 6. Scopes (OAuth Scopes)
CREATE TABLE IF NOT EXISTS scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT scopes_tenant_name_key UNIQUE (tenant_id, name)
);

-- 7. Scope Permissions (Many-to-Many)
CREATE TABLE IF NOT EXISTS scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id),
    FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- 8. Role Bindings (The Assignment Core)
CREATE TABLE IF NOT EXISTS role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    
    user_id UUID,
    service_account_id UUID,
    role_id UUID NOT NULL,
    
    scope_type TEXT, -- NULL for Tenant-Wide
    scope_id UUID,   -- NULL for Tenant-Wide
    
    conditions JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Strict Multi-Tenant Isolation (Composite Foreign Keys)
    CONSTRAINT fk_rb_user FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_rb_sa FOREIGN KEY (tenant_id, service_account_id) REFERENCES service_accounts(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_rb_role FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_rb_creator FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL,
    
    -- Ensure either user_id or service_account_id is set, but not both (or handle as needed)
    CONSTRAINT check_principal CHECK (
        (user_id IS NOT NULL AND service_account_id IS NULL) OR 
        (user_id IS NULL AND service_account_id IS NOT NULL)
    )
);

-- 9. Grant Audit
CREATE TABLE IF NOT EXISTS grant_audit (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    actor_user_id UUID,
    action TEXT,
    target_type TEXT,
    target_id UUID,
    before JSONB,
    after JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- COMMIT; (removed - app manages transactions)
