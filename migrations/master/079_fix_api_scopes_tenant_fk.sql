-- Migration: 124_fix_api_scopes_tenant_fk.sql
-- Description: Removes the tenant_id FK constraint from api_scopes table
-- Reason: Tenant databases don't have a 'tenants' table, so the FK constraint
--         causes insert failures. The tenant_id column is still kept for data isolation.

-- Drop the FK constraint if it exists (tenant databases)
-- This allows api_scopes to be used in tenant databases without referencing tenants table
ALTER TABLE IF EXISTS api_scopes DROP CONSTRAINT IF EXISTS api_scopes_tenant_id_fkey;

-- Note: The tenant_id column remains NOT NULL and is used for filtering,
-- but without the FK reference to a non-existent tenants table.
