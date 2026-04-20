-- Migration: Create RBAC (Role-Based Access Control) tables
-- This migration creates the necessary tables for role-based permissions

-- Migration 002: Add updated_at trigger to users table

-- Function to auto-update "updated_at"
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = NOW();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to call the function
DROP TRIGGER IF EXISTS users_set_updated_at ON users;
CREATE TRIGGER users_set_updated_at
BEFORE UPDATE ON users
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();


-- Create roles table
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(tenant_id, name)
);

-- Ensure roles table has proper DEFAULT for id column
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'roles'
        AND column_name = 'id'
        AND column_default LIKE '%gen_random_uuid%'
    ) THEN
        ALTER TABLE roles ALTER COLUMN id SET DEFAULT gen_random_uuid();
        RAISE NOTICE 'Added DEFAULT gen_random_uuid() to roles.id column';
    END IF;
END $$;

-- Create scopes table
CREATE TABLE IF NOT EXISTS scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(tenant_id, name)
);

-- Ensure scopes table has proper DEFAULT for id column
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'scopes'
        AND column_name = 'id'
        AND column_default LIKE '%gen_random_uuid%'
    ) THEN
        ALTER TABLE scopes ALTER COLUMN id SET DEFAULT gen_random_uuid();
        RAISE NOTICE 'Added DEFAULT gen_random_uuid() to scopes.id column';
    END IF;
END $$;

-- Create permissions table (many-to-many: roles <-> scopes <-> resources)
CREATE TABLE IF NOT EXISTS permissions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    scope_id UUID NOT NULL REFERENCES scopes(id) ON DELETE CASCADE,
    resource VARCHAR(255) NOT NULL,
    action VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(role_id, scope_id, resource, action)
);

-- Ensure permissions table has proper DEFAULT for id column
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'permissions'
        AND column_name = 'id'
        AND column_default LIKE '%gen_random_uuid%'
    ) THEN
        ALTER TABLE permissions ALTER COLUMN id SET DEFAULT gen_random_uuid();
        RAISE NOTICE 'Added DEFAULT gen_random_uuid() to permissions.id column';
    END IF;
END $$;

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scopes_tenant_id ON scopes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_permissions_role_id ON permissions(role_id);
CREATE INDEX IF NOT EXISTS idx_permissions_scope_id ON permissions(scope_id);

-- Global UNIQUE(name) on roles/scopes is intentionally omitted.
-- Per-tenant uniqueness is enforced by (tenant_id, name) constraints below.

-- Keep the composite unique constraints for tenant-specific entries
ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_tenant_id_name_key;
ALTER TABLE scopes DROP CONSTRAINT IF EXISTS scopes_tenant_id_name_key;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'UNIQUE'
        AND table_name = 'roles'
        AND constraint_name = 'roles_tenant_id_name_key'
    ) THEN
        ALTER TABLE roles ADD CONSTRAINT roles_tenant_id_name_key UNIQUE (tenant_id, name);
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'UNIQUE'
        AND table_name = 'scopes'
        AND constraint_name = 'scopes_tenant_id_name_key'
    ) THEN
        ALTER TABLE scopes ADD CONSTRAINT scopes_tenant_id_name_key UNIQUE (tenant_id, name);
    END IF;
END $$;

-- Insert default system-wide roles
INSERT INTO roles (name, description, tenant_id)
SELECT 'admin', 'Administrator with full system access', NULL
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'admin' AND tenant_id IS NULL);

INSERT INTO roles (name, description, tenant_id)
SELECT 'user', 'Regular user with limited access', NULL
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'user' AND tenant_id IS NULL);

INSERT INTO roles (name, description, tenant_id)
SELECT 'manager', 'Manager with elevated permissions', NULL
WHERE NOT EXISTS (SELECT 1 FROM roles WHERE name = 'manager' AND tenant_id IS NULL);

-- Insert default system-wide scopes
INSERT INTO scopes (name, description, tenant_id)
SELECT 'read', 'Read access to resources', NULL
WHERE NOT EXISTS (SELECT 1 FROM scopes WHERE name = 'read' AND tenant_id IS NULL);

INSERT INTO scopes (name, description, tenant_id)
SELECT 'write', 'Write access to resources', NULL
WHERE NOT EXISTS (SELECT 1 FROM scopes WHERE name = 'write' AND tenant_id IS NULL);

INSERT INTO scopes (name, description, tenant_id)
SELECT 'delete', 'Delete access to resources', NULL
WHERE NOT EXISTS (SELECT 1 FROM scopes WHERE name = 'delete' AND tenant_id IS NULL);

INSERT INTO scopes (name, description, tenant_id)
SELECT 'admin', 'Administrative access', NULL
WHERE NOT EXISTS (SELECT 1 FROM scopes WHERE name = 'admin' AND tenant_id IS NULL);

-- Insert default permissions only when old-style schema (role_id column) still present.
-- Migration 054 drops and recreates permissions with a tenant-scoped schema; after that
-- these inserts are irrelevant and must not run.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'permissions' AND column_name = 'role_id'
    ) THEN
        INSERT INTO permissions (role_id, scope_id, resource, action)
        SELECT r.id, s.id, 'all', 'all'
        FROM roles r CROSS JOIN scopes s
        WHERE r.name = 'admin' AND r.tenant_id IS NULL
        ON CONFLICT DO NOTHING;

        INSERT INTO permissions (role_id, scope_id, resource, action)
        SELECT r.id, s.id, 'users', 'read'
        FROM roles r CROSS JOIN scopes s
        WHERE r.name = 'user' AND r.tenant_id IS NULL AND s.name = 'read'
        ON CONFLICT DO NOTHING;

        INSERT INTO permissions (role_id, scope_id, resource, action)
        SELECT r.id, s.id, 'projects', 'read'
        FROM roles r CROSS JOIN scopes s
        WHERE r.name = 'user' AND r.tenant_id IS NULL AND s.name = 'read'
        ON CONFLICT DO NOTHING;
    END IF;
END $$;