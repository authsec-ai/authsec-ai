-- Migration: 018_add_tenant_db_column.sql
-- Purpose: Add tenant_db column to tenants table if it doesn't exist
-- Description: Safely adds the tenant_db column to the tenants table

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_name = 'tenants'
          AND table_schema = 'public'
    ) THEN
        ALTER TABLE tenants
        ADD COLUMN IF NOT EXISTS tenant_db VARCHAR(255);
    END IF;
END$$;