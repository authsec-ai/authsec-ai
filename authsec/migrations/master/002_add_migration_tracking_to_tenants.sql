-- Master Migration 002: Add migration tracking columns to tenants table
-- Eliminates the need for a separate tenant_databases tracking table.

ALTER TABLE tenants ADD COLUMN IF NOT EXISTS migration_status VARCHAR(50) DEFAULT 'pending';
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS last_migration INTEGER;
