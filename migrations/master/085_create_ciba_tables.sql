-- CIBA (Client Initiated Backchannel Authentication) Tables
-- Similar to device_codes, but for push notification based auth

-- Device Tokens: Store FCM/APNS tokens for push notifications
CREATE TABLE IF NOT EXISTS device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,

    device_token VARCHAR(500) NOT NULL UNIQUE,
    platform VARCHAR(20) NOT NULL, -- 'ios' or 'android'
    device_name VARCHAR(100), -- "iPhone 13 Pro", "Samsung Galaxy S21"
    device_model VARCHAR(100), -- "iPhone14,2", "SM-G991B"
    app_version VARCHAR(20), -- "1.0.0"
    os_version VARCHAR(20), -- "iOS 16.2", "Android 13"
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    last_used BIGINT, -- Unix timestamp when last push was sent
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_device_token_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,

    CONSTRAINT fk_device_token_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_device_tokens_user ON device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_tenant ON device_tokens(tenant_id);
CREATE INDEX IF NOT EXISTS idx_device_tokens_token ON device_tokens(device_token);
CREATE INDEX IF NOT EXISTS idx_device_tokens_active ON device_tokens(is_active);

-- CIBA Auth Requests: Track push notification authentication requests
CREATE TABLE IF NOT EXISTS ciba_auth_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- Request identifier (returned to client for polling)
    auth_req_id VARCHAR(255) NOT NULL UNIQUE,

    -- User identification
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    user_email VARCHAR(255) NOT NULL,

    -- Client information
    client_id UUID,

    -- Push notification
    device_token_id UUID NOT NULL, -- Which device we sent push to
    binding_message VARCHAR(255), -- "Voice Agent Login", "API Access Request"

    -- OAuth scopes
    scopes JSONB DEFAULT '[]'::jsonb,

    -- Status: pending, approved, denied, expired, consumed
    status VARCHAR(50) DEFAULT 'pending' NOT NULL,

    -- Biometric verification
    biometric_verified BOOLEAN DEFAULT FALSE,

    -- Timestamps (Unix epoch)
    expires_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    responded_at BIGINT, -- When user approved/denied
    last_polled_at BIGINT, -- When client last polled

    CONSTRAINT fk_ciba_auth_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,

    CONSTRAINT fk_ciba_auth_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,

    CONSTRAINT fk_ciba_auth_device FOREIGN KEY (device_token_id)
        REFERENCES device_tokens(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_ciba_auth_req_id ON ciba_auth_requests(auth_req_id);
CREATE INDEX IF NOT EXISTS idx_ciba_auth_user ON ciba_auth_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_ciba_auth_tenant ON ciba_auth_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ciba_auth_status ON ciba_auth_requests(status);
CREATE INDEX IF NOT EXISTS idx_ciba_auth_expires ON ciba_auth_requests(expires_at);

COMMENT ON TABLE device_tokens IS 'FCM/APNS device tokens for push notifications';
COMMENT ON TABLE ciba_auth_requests IS 'CIBA authentication requests (push notification based auth)';

COMMENT ON COLUMN device_tokens.device_token IS 'FCM token (Android) or APNS token (iOS)';
COMMENT ON COLUMN ciba_auth_requests.auth_req_id IS 'Unique request ID returned to client for polling';
COMMENT ON COLUMN ciba_auth_requests.binding_message IS 'Message shown to user in push notification';
