-- Migration to create sync_configurations table for storing AD/Entra sync credentials
-- This table stores directory sync configurations in the main authsec database

CREATE TABLE IF NOT EXISTS sync_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    client_id UUID NOT NULL,
    project_id UUID NOT NULL,

    -- Sync type: 'active_directory' or 'entra_id'
    sync_type VARCHAR(50) NOT NULL CHECK (sync_type IN ('active_directory', 'entra_id')),

    -- Configuration name for easy identification
    config_name VARCHAR(255) NOT NULL,
    description TEXT,

    -- Common fields
    is_active BOOLEAN NOT NULL DEFAULT true,

    -- AD-specific fields (encrypted)
    ad_server VARCHAR(500),
    ad_username VARCHAR(500),
    ad_password TEXT, -- encrypted
    ad_base_dn VARCHAR(500),
    ad_filter TEXT,
    ad_use_ssl BOOLEAN DEFAULT true,
    ad_skip_verify BOOLEAN DEFAULT false,

    -- Entra ID-specific fields (encrypted)
    entra_tenant_id VARCHAR(500),
    entra_client_id VARCHAR(500),
    entra_client_secret TEXT, -- encrypted
    entra_scopes TEXT, -- JSON array stored as text
    entra_skip_verify BOOLEAN DEFAULT false,

    -- Metadata
    last_sync_at TIMESTAMP WITH TIME ZONE,
    last_sync_status VARCHAR(50), -- 'success', 'failed', 'in_progress'
    last_sync_error TEXT,
    last_sync_users_count INTEGER DEFAULT 0,

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by UUID, -- admin user who created this config

    -- Ensure unique config names per tenant
    UNIQUE(tenant_id, config_name)
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_sync_configs_tenant_id ON sync_configurations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sync_configs_client_id ON sync_configurations(client_id);
CREATE INDEX IF NOT EXISTS idx_sync_configs_sync_type ON sync_configurations(sync_type);
CREATE INDEX IF NOT EXISTS idx_sync_configs_tenant_type ON sync_configurations(tenant_id, sync_type);
CREATE INDEX IF NOT EXISTS idx_sync_configs_active ON sync_configurations(is_active);

-- Add comment to table
COMMENT ON TABLE sync_configurations IS 'Stores Active Directory and Entra ID sync configurations with encrypted credentials';
COMMENT ON COLUMN sync_configurations.sync_type IS 'Type of directory sync: active_directory or entra_id';
COMMENT ON COLUMN sync_configurations.ad_password IS 'Encrypted AD service account password';
COMMENT ON COLUMN sync_configurations.entra_client_secret IS 'Encrypted Entra ID client secret';
