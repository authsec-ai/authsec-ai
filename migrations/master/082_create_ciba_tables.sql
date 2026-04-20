
CREATE TABLE IF NOT EXISTS ciba_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    auth_req_id VARCHAR(255) NOT NULL UNIQUE, -- Okta's authentication request ID
    login_hint VARCHAR(255) NOT NULL, -- Email or user identifier
    binding_message VARCHAR(255), -- Message shown on user's device
    status VARCHAR(50) DEFAULT 'pending' NOT NULL, -- pending, approved, denied, expired
    expires_at BIGINT NOT NULL, -- Unix epoch timestamp
    created_at BIGINT NOT NULL,
    completed_at BIGINT, -- When user approved/denied

    CONSTRAINT fk_ciba_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,

    CONSTRAINT fk_ciba_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ciba_auth_req_id ON ciba_requests(auth_req_id);
CREATE INDEX IF NOT EXISTS idx_ciba_user ON ciba_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_ciba_status ON ciba_requests(status);
CREATE INDEX IF NOT EXISTS idx_ciba_expires_at ON ciba_requests(expires_at);

CREATE TABLE IF NOT EXISTS user_auth_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    preferred_method VARCHAR(50) DEFAULT 'device_code' NOT NULL, -- ciba, device_code, totp
    okta_verify_enrolled BOOLEAN DEFAULT FALSE NOT NULL, -- Has user enrolled Okta Verify?
    okta_user_id VARCHAR(255), -- Okta user ID for CIBA
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_auth_pref_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,

    CONSTRAINT fk_auth_pref_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,

    UNIQUE(user_id, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_auth_pref_user ON user_auth_preferences(user_id);
CREATE INDEX IF NOT EXISTS idx_auth_pref_method ON user_auth_preferences(preferred_method);

COMMENT ON TABLE ciba_requests IS 'Tracks CIBA backchannel authentication requests from voice agents';
COMMENT ON TABLE user_auth_preferences IS 'Stores user preferences for authentication methods';
COMMENT ON COLUMN ciba_requests.auth_req_id IS 'Okta authentication request ID returned from /bc/authorize';
COMMENT ON COLUMN ciba_requests.binding_message IS 'Message displayed on user device (e.g., "Voice Agent Login")';
COMMENT ON COLUMN user_auth_preferences.preferred_method IS 'User preferred auth: ciba (push), device_code, or totp';
