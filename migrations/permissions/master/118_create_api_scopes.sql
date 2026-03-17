-- Migration: 118_create_api_scopes.sql
-- Description: Creates API Scopes tables for OAuth scope-to-permission mapping
-- API Scopes allow external applications to request specific permissions via OAuth

-- 1. API Scopes (External OAuth Contracts)
-- These are high-level keys given to external apps (e.g., "files:read", "project:write")
CREATE TABLE IF NOT EXISTS api_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL, -- e.g., 'files:read', 'project:write'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),

    -- Unique scope name per tenant
    CONSTRAINT uq_api_scopes_tenant_name UNIQUE (tenant_id, name),
    -- Composite unique for tenant isolation
    CONSTRAINT uq_api_scopes_tenant_id UNIQUE (tenant_id, id)
);

-- 2. API Scope Permissions (M:N Linkage)
-- Maps OAuth scopes to internal RBAC permissions
-- When a client is authorized with scope "files:read", they get all mapped permissions
CREATE TABLE IF NOT EXISTS api_scope_permissions (
    scope_id UUID NOT NULL,
    permission_id UUID NOT NULL,
    PRIMARY KEY (scope_id, permission_id),
    FOREIGN KEY (scope_id) REFERENCES api_scopes(id) ON DELETE CASCADE,
    FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
);

-- Performance indexes
CREATE INDEX IF NOT EXISTS idx_api_scopes_tenant_id ON api_scopes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_scopes_name ON api_scopes(name);
CREATE INDEX IF NOT EXISTS idx_api_scope_permissions_scope_id ON api_scope_permissions(scope_id);
CREATE INDEX IF NOT EXISTS idx_api_scope_permissions_permission_id ON api_scope_permissions(permission_id);

-- Add comment for documentation
COMMENT ON TABLE api_scopes IS 'OAuth API Scopes - external contracts that map to internal RBAC permissions';
COMMENT ON TABLE api_scope_permissions IS 'Maps API Scopes to internal RBAC Permissions (M:N)';
COMMENT ON COLUMN api_scopes.name IS 'OAuth scope name, e.g., files:read, project:write';

