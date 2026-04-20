-- Migration: Create scope_resource_mappings table in main DB
-- The admin scopes/mappings endpoint queries this table on the main DB connection.

CREATE TABLE IF NOT EXISTS scope_resource_mappings (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id uuid NOT NULL,
    scope_name text NOT NULL DEFAULT '*',
    resource_name text NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now(),
    CONSTRAINT scope_resource_mappings_tenant_scope_resource_key UNIQUE (tenant_id, scope_name, resource_name)
);

CREATE INDEX IF NOT EXISTS idx_scope_resource_mappings_tenant ON scope_resource_mappings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scope_resource_mappings_scope ON scope_resource_mappings(tenant_id, scope_name);
