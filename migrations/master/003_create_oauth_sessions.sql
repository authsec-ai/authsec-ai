-- SDK-Manager migration: OAuth sessions table for MCP Auth service.
-- Pre-auth sessions live in the master DB; post-auth sessions migrate to tenant DBs.

CREATE TABLE IF NOT EXISTS oauth_sessions (
    session_id       VARCHAR(36) PRIMARY KEY,
    user_email       VARCHAR(255),
    user_info        JSONB,
    access_token     TEXT,
    refresh_token    TEXT,
    authorization_code TEXT,
    token_expires_at BIGINT,
    created_at       BIGINT NOT NULL,
    last_activity    BIGINT NOT NULL,
    oauth_state      VARCHAR(255),
    pkce_verifier    TEXT,
    pkce_challenge   TEXT,
    is_active        BOOLEAN DEFAULT true,
    client_identifier VARCHAR(255),
    org_id           VARCHAR(255),
    tenant_id        VARCHAR(255),
    user_id          VARCHAR(255),
    provider         VARCHAR(100),
    provider_id      VARCHAR(255),
    accessible_tools JSONB
);

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_org_id
    ON oauth_sessions(org_id) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_client
    ON oauth_sessions(client_identifier) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_state
    ON oauth_sessions(oauth_state) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_tenant
    ON oauth_sessions(tenant_id) WHERE is_active = true;
