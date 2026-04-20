-- Migration 001: Create users table with sharedmodels compatibility

-- Ensure required extension for UUIDs
CREATE EXTENSION IF NOT EXISTS pgcrypto;


-- Create users table with correct UUID schema
-- REMOVED UNIQUE constraint on email to allow duplicate emails across different clients
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id UUID,
    tenant_id UUID,
    project_id UUID,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255) DEFAULT 'Not Provided',
    username VARCHAR(255) DEFAULT 'Not Provided',
    password_hash TEXT,
    tenant_domain VARCHAR(255) DEFAULT 'app.authsec.dev',
    provider VARCHAR(100) DEFAULT 'local',
    provider_id VARCHAR(255), -- Keep as VARCHAR for flexibility with different provider ID formats
    provider_data JSONB,
    avatar_url TEXT,
    active BOOLEAN DEFAULT true,
    mfa_enabled BOOLEAN DEFAULT false,
    mfa_method TEXT[],
    mfa_default_method VARCHAR(50),
    mfa_enrolled_at TIMESTAMP WITH TIME ZONE,
    mfa_verified BOOLEAN DEFAULT false,
    external_id VARCHAR(255),
    sync_source VARCHAR(100),
    last_sync_at TIMESTAMP WITH TIME ZONE,
    is_synced_user BOOLEAN DEFAULT false,
    last_login TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() NOT NULL
);

-- Indexes for performance




alter table users add column if not exists external_id VARCHAR(100) DEFAULT 'local';

alter table users add column if not exists provider VARCHAR(100) DEFAULT 'local';

alter table users add COLUMN if not exists password_hash text;

alter table users add COLUMN if not exists tenant_domain varchar(100) DEFAULT 'local';

alter table users add COLUMN if not exists provider VARCHAR(100);


CREATE INDEX IF NOT EXISTS idx_users_client_id ON users(client_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_project_id ON users(project_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_domain ON users(tenant_domain);
CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider, provider_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_external_id ON users(external_id);

-- Ensure uniqueness across provider + provider_id (allowing NULL values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_provider_provider_id
    ON users(provider, provider_id)
    WHERE provider_id IS NOT NULL;

-- JSONB indexes for performance
CREATE INDEX IF NOT EXISTS idx_users_provider_data ON users USING gin (provider_data);
CREATE INDEX IF NOT EXISTS idx_users_mfa_method ON users USING gin (mfa_method);





