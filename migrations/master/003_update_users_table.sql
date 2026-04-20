-- Migration to add missing columns to existing users table
-- This migration is for existing installations that already have a basic users table
-- New installations should use 000_create_users_table.sql instead

-- Add missing columns (for existing installations with old users table)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS client_id UUID,
ADD COLUMN IF NOT EXISTS project_id UUID,
ADD COLUMN IF NOT EXISTS name VARCHAR(255),
ADD COLUMN IF NOT EXISTS username VARCHAR(255),
ADD COLUMN IF NOT EXISTS password_hash TEXT,
ADD COLUMN IF NOT EXISTS tenant_domain VARCHAR(255),
ADD COLUMN IF NOT EXISTS provider_id VARCHAR(255),
ADD COLUMN IF NOT EXISTS provider_data JSONB,
ADD COLUMN IF NOT EXISTS avatar_url TEXT,
ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS mfa_method JSONB,
ADD COLUMN IF NOT EXISTS mfa_default_method VARCHAR(50),
ADD COLUMN IF NOT EXISTS mfa_enrolled_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS mfa_verified BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS sync_source VARCHAR(100),
ADD COLUMN IF NOT EXISTS last_sync_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS is_synced_user BOOLEAN DEFAULT false,
ADD COLUMN IF NOT EXISTS last_login TIMESTAMP WITH TIME ZONE;

-- Update provider default value for existing records
UPDATE users SET provider = 'local' WHERE provider IS NULL OR provider = 'custom';

-- Add indexes for performance (safe to run multiple times)
CREATE INDEX IF NOT EXISTS idx_users_client_id ON users(client_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_project_id ON users(project_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);
CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider);
CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);