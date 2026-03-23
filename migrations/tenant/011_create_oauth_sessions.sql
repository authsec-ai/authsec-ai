-- SDK-Manager migration: OAuth sessions table in tenant databases.
-- The base template already creates a minimal oauth_sessions table, so this
-- migration adds the SDK-Manager-specific columns if they are not yet present.

ALTER TABLE oauth_sessions
    ADD COLUMN IF NOT EXISTS user_email        VARCHAR(255),
    ADD COLUMN IF NOT EXISTS user_info         JSONB,
    ADD COLUMN IF NOT EXISTS authorization_code TEXT,
    ADD COLUMN IF NOT EXISTS token_expires_at  BIGINT,
    ADD COLUMN IF NOT EXISTS last_activity     BIGINT,
    ADD COLUMN IF NOT EXISTS oauth_state       VARCHAR(255),
    ADD COLUMN IF NOT EXISTS pkce_verifier     TEXT,
    ADD COLUMN IF NOT EXISTS pkce_challenge    TEXT,
    ADD COLUMN IF NOT EXISTS is_active         BOOLEAN DEFAULT true,
    ADD COLUMN IF NOT EXISTS client_identifier VARCHAR(255),
    ADD COLUMN IF NOT EXISTS org_id            VARCHAR(255),
    ADD COLUMN IF NOT EXISTS provider          VARCHAR(100),
    ADD COLUMN IF NOT EXISTS provider_id       VARCHAR(255),
    ADD COLUMN IF NOT EXISTS accessible_tools  JSONB;

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_org_id
    ON oauth_sessions(org_id) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_client
    ON oauth_sessions(client_identifier) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_oauth_sessions_state
    ON oauth_sessions(oauth_state) WHERE is_active = true;
