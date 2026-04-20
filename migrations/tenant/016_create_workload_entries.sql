-- Migration 015: Create workload_entries table for SPIRE workload registration.
-- Note: No FK to tenants table as it doesn't exist in tenant databases.

CREATE TABLE IF NOT EXISTS workload_entries (
    id uuid NOT NULL DEFAULT gen_random_uuid(),
    spiffe_id varchar(512) NOT NULL,
    parent_id varchar(512) NOT NULL,
    selectors jsonb NOT NULL,
    ttl integer DEFAULT 3600,
    admin boolean DEFAULT false,
    federates_with text[],
    downstream boolean DEFAULT false,
    dns_names text[],
    spire_entry_id varchar(255),
    created_at timestamp without time zone DEFAULT now(),
    updated_at timestamp without time zone DEFAULT now(),
    tenant_id uuid NOT NULL,
    PRIMARY KEY(id)
);

CREATE UNIQUE INDEX IF NOT EXISTS workload_entries_spiffe_id_key ON workload_entries (spiffe_id);
CREATE INDEX IF NOT EXISTS idx_parent_id ON workload_entries (parent_id);
CREATE INDEX IF NOT EXISTS idx_spiffe_id ON workload_entries (spiffe_id);
CREATE INDEX IF NOT EXISTS idx_workload_entries_tenant_id ON workload_entries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_workload_entries_tenant_parent ON workload_entries (tenant_id, parent_id);

COMMENT ON COLUMN workload_entries.tenant_id IS 'Tenant that owns this workload entry (added for multi-tenancy)';
