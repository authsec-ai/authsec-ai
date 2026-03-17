-- Migration 104: Enforce Scoped RBAC Schema
-- Purpose: Implement strict multi-tenant RBAC with composite keys, scoped permissions, and service accounts.
-- Fixed: All CREATE TABLE use IF NOT EXISTS; constraints wrapped in DO blocks for idempotency.

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 1. Tenants (Keep existing, ensure columns exist)
-- Existing tenants table structure is preserved.

-- 2. Users (Keep existing)
-- Ensure the unique constraint for composite FK exists
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'users_tenant_id_id_key'
        AND conrelid = 'users'::regclass
    ) THEN
        BEGIN
            ALTER TABLE users ADD CONSTRAINT users_tenant_id_id_key UNIQUE (tenant_id, id);
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'Could not add unique constraint users_tenant_id_id_key, possibly due to null tenant_ids';
        END;
    END IF;
END $$;

-- 2b. Service Accounts
CREATE TABLE IF NOT EXISTS service_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    ALTER TABLE service_accounts ADD CONSTRAINT service_accounts_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE service_accounts ADD CONSTRAINT service_accounts_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

-- 3. Roles (Definitions)
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    is_system BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_name_key UNIQUE (tenant_id, name);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

-- 4. Permissions (Atomic Capabilities)
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    resource TEXT NOT NULL,
    action TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_resource_action_key UNIQUE (tenant_id, resource, action);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE permissions ADD CONSTRAINT permissions_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

-- 5. Role Permissions (M:N Linkage)
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (role_id, permission_id)
);

DO $$
BEGIN
    ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_role_id_fkey
        FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE role_permissions ADD CONSTRAINT role_permissions_permission_id_fkey
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

-- 6. API Scopes (External Contracts)
CREATE TABLE IF NOT EXISTS scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    ALTER TABLE scopes ADD CONSTRAINT scopes_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE scopes ADD CONSTRAINT scopes_tenant_id_name_key UNIQUE (tenant_id, name);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE scopes ADD CONSTRAINT scopes_tenant_id_id_key UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

-- 7. Scope Permissions (M:N Linkage)
CREATE TABLE IF NOT EXISTS scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id)
);

DO $$
BEGIN
    ALTER TABLE scope_permissions ADD CONSTRAINT scope_permissions_scope_id_fkey
        FOREIGN KEY (scope_id) REFERENCES scopes(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE scope_permissions ADD CONSTRAINT scope_permissions_permission_id_fkey
        FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

-- 8. Role Bindings (The Core Assignment)
CREATE TABLE IF NOT EXISTS role_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID,
    service_account_id UUID,
    role_id UUID NOT NULL,
    scope_type TEXT,
    scope_id UUID,
    conditions JSONB DEFAULT '{}',
    expires_at TIMESTAMPTZ,
    created_by UUID,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_tenant_id_fkey
        FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_user_fkey
        FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_sa_fkey
        FOREIGN KEY (tenant_id, service_account_id) REFERENCES service_accounts(tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_role_fkey
        FOREIGN KEY (tenant_id, role_id) REFERENCES roles(tenant_id, id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT role_bindings_created_by_fkey
        FOREIGN KEY (created_by) REFERENCES users(id);
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;

DO $$
BEGIN
    ALTER TABLE role_bindings ADD CONSTRAINT check_principal CHECK (
        (user_id IS NOT NULL AND service_account_id IS NULL) OR
        (user_id IS NULL AND service_account_id IS NOT NULL)
    );
EXCEPTION WHEN duplicate_object OR duplicate_table THEN NULL;
END $$;
