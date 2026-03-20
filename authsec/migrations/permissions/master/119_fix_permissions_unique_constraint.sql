-- Migration 119: Fix permissions table unique constraint for ON CONFLICT
-- The admin_seed_repository uses ON CONFLICT (tenant_id, resource, action)
-- which requires a unique constraint on exactly those columns

-- Create the correct unique index (if not exists)
CREATE UNIQUE INDEX IF NOT EXISTS idx_permissions_tenant_resource_action_unique
ON permissions (tenant_id, resource, action);

-- Note: The old index idx_permissions_tenant_resource_action included 'id' column
-- which breaks ON CONFLICT matching. This new index enables proper upsert behavior.
