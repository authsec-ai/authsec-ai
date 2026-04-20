-- Migration: Add foreign keys and constraints to users table
-- This migration adds proper foreign key relationships for users table

-- Add foreign key constraint for tenant_id (if tenants table exists and constraint doesn't exist)
-- REMOVED: fk_users_tenant_id - User model has no FK relationship tags in shared-models
-- DO $$
-- BEGIN
--     IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'tenants') THEN
--         IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints
--                       WHERE constraint_name = 'fk_users_tenant_id'
--                       AND table_name = 'users') THEN
--             ALTER TABLE users ADD CONSTRAINT fk_users_tenant_id
--                 FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE;
--         END IF;
--     END IF;
-- END $$;

-- Add check constraints (skip if they already exist)
-- Note: Using simple ALTER statements since the constraints should not exist yet

-- Add indexes (these are safe to run multiple times)
CREATE INDEX IF NOT EXISTS idx_users_tenant_project ON users(tenant_id, project_id);
CREATE INDEX IF NOT EXISTS idx_users_provider_status ON users(provider, active);
CREATE INDEX IF NOT EXISTS idx_users_sync_info ON users(sync_source, is_synced_user);
CREATE INDEX IF NOT EXISTS idx_users_mfa ON users(mfa_enabled, mfa_verified);
CREATE INDEX IF NOT EXISTS idx_users_timestamps ON users(created_at, updated_at);