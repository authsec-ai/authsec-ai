-- oauth_oidc_configurations: stores OIDC provider configurations per tenant
-- Used by oocmgr (oath_oidc_configuration_manager) service

CREATE TABLE IF NOT EXISTS oauth_oidc_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    org_id VARCHAR(255) NOT NULL,
    tenant_id VARCHAR(255) NOT NULL,
    config_type VARCHAR(255) NOT NULL,
    config_files JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    created_by VARCHAR(255),
    updated_by VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_oauth_oidc_configurations_name ON oauth_oidc_configurations(name);
CREATE INDEX IF NOT EXISTS idx_oauth_oidc_configurations_org_id ON oauth_oidc_configurations(org_id);
CREATE INDEX IF NOT EXISTS idx_oauth_oidc_configurations_tenant_id ON oauth_oidc_configurations(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oauth_oidc_configurations_deleted_at ON oauth_oidc_configurations(deleted_at);
