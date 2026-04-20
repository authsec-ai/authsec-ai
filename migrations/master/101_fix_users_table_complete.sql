-- Migration: Fix users table to match sharedmodels.User schema
-- This adds all missing columns required by sharedmodels v0.5.0

-- BEGIN; (removed - app manages transactions)

-- Add missing columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS tenant_domain VARCHAR(255) DEFAULT 'authsec.dev';
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider VARCHAR(100) DEFAULT 'local';
ALTER TABLE users ADD COLUMN IF NOT EXISTS provider_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS active BOOLEAN DEFAULT true;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enabled BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_method TEXT[];
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_default_method VARCHAR(50);
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_enrolled_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS mfa_verified BOOLEAN DEFAULT false;
ALTER TABLE users ADD COLUMN IF NOT EXISTS external_id VARCHAR(255);
ALTER TABLE users ADD COLUMN IF NOT EXISTS sync_source VARCHAR(100);
ALTER TABLE users ADD COLUMN IF NOT EXISTS last_sync_at TIMESTAMP WITH TIME ZONE;
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_synced_user BOOLEAN DEFAULT false;

ALTER TABLE users ADD COLUMN IF NOT EXISTS is_active BOOLEAN DEFAULT true;

-- Drop old incompatible columns
ALTER TABLE users DROP COLUMN IF EXISTS mfa_data;

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_tenant_domain ON users(tenant_domain);
CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider, provider_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active);
CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);
CREATE INDEX IF NOT EXISTS idx_users_sync_source ON users(sync_source);
CREATE INDEX IF NOT EXISTS idx_users_provider_data ON users USING gin(provider_data);

-- Ensure uniqueness across provider + provider_id (allowing NULL values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_provider_id
    ON users(provider, provider_id)
    WHERE provider_id IS NOT NULL;

-- COMMIT; (removed - app manages transactions)
