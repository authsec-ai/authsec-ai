-- Tenant Migration: 004_create_api_scopes_tenant.sql
-- Description: Creates API Scopes tables for OAuth scope-to-permission mapping in tenant databases
-- API Scopes allow external applications to request specific permissions via OAuth

-- 1. API Scopes (External OAuth Contracts)
-- Note: tenant_id does NOT have FK to tenants table because tenant DBs don't have a tenants table
CREATE TABLE IF NOT EXISTS api_scopes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL, -- No FK - tenant DBs don't have tenants table
    name TEXT NOT NULL, -- e.g., 'files:read', 'project:write'
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    CONSTRAINT uq_api_scopes_tenant_name UNIQUE (tenant_id, name),
    CONSTRAINT uq_api_scopes_tenant_id UNIQUE (tenant_id, id)
);

-- 2. API Scope Permissions (M:N Linkage)
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
