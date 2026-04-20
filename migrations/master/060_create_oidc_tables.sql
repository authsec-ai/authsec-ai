-- Migration: 102_create_oidc_tables.sql
-- Description: Create OIDC provider tables for social login (Google, GitHub, Microsoft)
-- These tables are in the MAIN database, not tenant databases

-- =====================================================
-- Table: oidc_providers
-- Platform-level OIDC provider configurations
-- These are YOUR app's credentials for Google, GitHub, etc.
-- =====================================================
CREATE TABLE IF NOT EXISTS oidc_providers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_name VARCHAR(50) UNIQUE NOT NULL,          -- 'google', 'github', 'microsoft'
    display_name VARCHAR(100) NOT NULL,                  -- 'Google', 'GitHub', 'Microsoft'
    client_id VARCHAR(255) NOT NULL,                     -- OAuth client ID
    client_secret_vault_path VARCHAR(255) NOT NULL,      -- Vault path for secret
    authorization_url VARCHAR(500) NOT NULL,             -- OAuth authorize endpoint
    token_url VARCHAR(500) NOT NULL,                     -- OAuth token endpoint
    userinfo_url VARCHAR(500) NOT NULL,                  -- OAuth userinfo endpoint
    scopes TEXT DEFAULT 'openid email profile',          -- Space-separated scopes
    icon_url VARCHAR(500),                               -- Provider icon for UI
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE oidc_providers IS 'Platform-level OIDC provider configurations (Google, GitHub, Microsoft)';
COMMENT ON COLUMN oidc_providers.provider_name IS 'Unique identifier: google, github, microsoft';
COMMENT ON COLUMN oidc_providers.client_secret_vault_path IS 'HashiCorp Vault path where client_secret is stored';

-- =====================================================
-- Table: oidc_states
-- Short-lived state storage for OIDC flow security
-- Used to pass tenant context through OAuth redirects
-- =====================================================
CREATE TABLE IF NOT EXISTS oidc_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state_token VARCHAR(255) UNIQUE NOT NULL,            -- Random state for CSRF protection
    tenant_id UUID,                                      -- NULL for new registration
    tenant_domain VARCHAR(255) NOT NULL,                 -- e.g., 'ritam' for ritam.app.authsec.dev
    provider_name VARCHAR(50) NOT NULL,                  -- 'google', 'github', 'microsoft'
    action VARCHAR(20) NOT NULL,                         -- 'login' or 'register'
    code_verifier VARCHAR(128),                          -- For PKCE (optional but recommended)
    redirect_after VARCHAR(500),                         -- Where to redirect after success
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,        -- State expiry (usually 10 minutes)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE oidc_states IS 'Short-lived OIDC state storage for secure OAuth flow';
COMMENT ON COLUMN oidc_states.state_token IS 'Random token passed to OAuth provider and verified on callback';
COMMENT ON COLUMN oidc_states.code_verifier IS 'PKCE code verifier for enhanced security';

-- =====================================================
-- Table: oidc_user_identities
-- Links OIDC provider identities to users
-- Allows lookup: "Does this Google user exist in this tenant?"
-- =====================================================
CREATE TABLE IF NOT EXISTS oidc_user_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(tenant_id) ON DELETE CASCADE,
    user_id UUID NOT NULL,                               -- User ID in tenant database
    provider_name VARCHAR(50) NOT NULL,                  -- 'google', 'github', 'microsoft'
    provider_user_id VARCHAR(255) NOT NULL,              -- Provider's unique user ID (sub claim)
    email VARCHAR(255),                                  -- Email from provider
    profile_data JSONB,                                  -- Additional profile info (name, picture)
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,

    -- Each provider user ID can only link to one account globally
    CONSTRAINT oidc_user_identities_provider_unique UNIQUE (provider_name, provider_user_id),
    -- Each user can only have one identity per provider per tenant
    CONSTRAINT oidc_user_identities_tenant_user_provider_unique UNIQUE (tenant_id, user_id, provider_name)
);

COMMENT ON TABLE oidc_user_identities IS 'Links OIDC provider identities to tenant users';
COMMENT ON COLUMN oidc_user_identities.provider_user_id IS 'Unique user ID from provider (Google sub, GitHub id)';

-- =====================================================
-- Indexes for performance
-- =====================================================
CREATE INDEX IF NOT EXISTS idx_oidc_providers_active ON oidc_providers(is_active);
CREATE INDEX IF NOT EXISTS idx_oidc_states_token ON oidc_states(state_token);
CREATE INDEX IF NOT EXISTS idx_oidc_states_expires ON oidc_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_provider_user ON oidc_user_identities(provider_name, provider_user_id);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_tenant ON oidc_user_identities(tenant_id);
CREATE INDEX IF NOT EXISTS idx_oidc_identities_user ON oidc_user_identities(tenant_id, user_id);

-- =====================================================
-- Trigger for updated_at
-- =====================================================
CREATE OR REPLACE FUNCTION update_oidc_providers_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS oidc_providers_updated_at ON oidc_providers;
CREATE TRIGGER oidc_providers_updated_at
    BEFORE UPDATE ON oidc_providers
    FOR EACH ROW
    EXECUTE FUNCTION update_oidc_providers_updated_at();

CREATE OR REPLACE FUNCTION update_oidc_user_identities_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS oidc_user_identities_updated_at ON oidc_user_identities;
CREATE TRIGGER oidc_user_identities_updated_at
    BEFORE UPDATE ON oidc_user_identities
    FOR EACH ROW
    EXECUTE FUNCTION update_oidc_user_identities_updated_at();

-- =====================================================
-- Seed default OIDC providers (inactive by default)
-- Admin needs to configure client_id and enable
-- =====================================================
INSERT INTO oidc_providers (provider_name, display_name, client_id, client_secret_vault_path, authorization_url, token_url, userinfo_url, scopes, icon_url, is_active)
VALUES
    ('google', 'Google', 'CONFIGURE_ME', 'secret/oidc/google', 'https://accounts.google.com/o/oauth2/v2/auth', 'https://oauth2.googleapis.com/token', 'https://openidconnect.googleapis.com/v1/userinfo', 'openid email profile', 'https://www.google.com/favicon.ico', false),
    ('github', 'GitHub', 'CONFIGURE_ME', 'secret/oidc/github', 'https://github.com/login/oauth/authorize', 'https://github.com/login/oauth/access_token', 'https://api.github.com/user', 'read:user user:email', 'https://github.com/favicon.ico', false),
    ('microsoft', 'Microsoft', 'CONFIGURE_ME', 'secret/oidc/microsoft', 'https://login.microsoftonline.com/common/oauth2/v2.0/authorize', 'https://login.microsoftonline.com/common/oauth2/v2.0/token', 'https://graph.microsoft.com/oidc/userinfo', 'openid email profile', 'https://www.microsoft.com/favicon.ico', false)
ON CONFLICT (provider_name) DO NOTHING;

-- =====================================================
-- Cleanup job: Delete expired states (run via cron)
-- DELETE FROM oidc_states WHERE expires_at < NOW();
-- =====================================================
