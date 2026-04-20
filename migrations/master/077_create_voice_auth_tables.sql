CREATE TABLE IF NOT EXISTS voice_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id UUID, -- Stored from tenant_mappings, no FK constraint


    session_token VARCHAR(128) UNIQUE NOT NULL, -- Secret token for this voice session


    voice_otp VARCHAR(10) NOT NULL, -- Numeric code spoken by assistant (e.g., "8532")
    otp_attempts INTEGER DEFAULT 0, -- Failed verification attempts

    voice_platform VARCHAR(50), -- 'alexa', 'google', 'siri', 'custom'
    voice_user_id TEXT, -- Platform-specific user ID (e.g., Alexa user ID)
    device_info JSONB, -- Additional device metadata

    user_id UUID, -- References users(id) in tenant DB
    user_email TEXT, -- Cached for convenience

    status VARCHAR(20) DEFAULT 'initiated' NOT NULL, -- initiated/verified/expired/failed

    linked_device_code VARCHAR(128), -- References device_codes.device_code


    scopes JSONB DEFAULT '[]'::jsonb,

    expires_at TIMESTAMP NOT NULL, -- Short expiry: typically 3-5 minutes
    verified_at TIMESTAMP, -- When OTP was verified

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    CONSTRAINT chk_voice_sessions_status CHECK (status IN ('initiated', 'verified', 'expired', 'failed')),
    CONSTRAINT chk_voice_otp_attempts CHECK (otp_attempts >= 0 AND otp_attempts <= 5)
);


CREATE TABLE IF NOT EXISTS voice_identity_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,

    -- Voice assistant identity
    voice_platform VARCHAR(50) NOT NULL, -- 'alexa', 'google', 'siri'
    voice_user_id TEXT NOT NULL, -- Platform-specific user ID
    voice_user_name TEXT, -- Optional display name from platform

    user_id UUID NOT NULL, -- References users(id) in tenant DB
    user_email TEXT NOT NULL, -- Cached for convenience


    is_active BOOLEAN DEFAULT true,
    link_method VARCHAR(50), -- 'browser_verification', 'voice_otp', 'admin_linked'

    last_used_at TIMESTAMP,
    linked_at TIMESTAMP DEFAULT NOW(),


    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),

    
    CONSTRAINT uq_voice_identity_tenant_platform_user UNIQUE (tenant_id, voice_platform, voice_user_id)
);

CREATE INDEX IF NOT EXISTS idx_voice_sessions_session_token ON voice_sessions(session_token);
CREATE INDEX IF NOT EXISTS idx_voice_sessions_tenant_id ON voice_sessions(tenant_id);
CREATE INDEX IF NOT EXISTS idx_voice_sessions_status ON voice_sessions(status);
CREATE INDEX IF NOT EXISTS idx_voice_sessions_expires_at ON voice_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_voice_sessions_voice_user_id ON voice_sessions(voice_user_id);

CREATE INDEX IF NOT EXISTS idx_voice_identity_links_tenant_id ON voice_identity_links(tenant_id);
CREATE INDEX IF NOT EXISTS idx_voice_identity_links_user_id ON voice_identity_links(user_id);
CREATE INDEX IF NOT EXISTS idx_voice_identity_links_voice_platform_user ON voice_identity_links(voice_platform, voice_user_id);
CREATE INDEX IF NOT EXISTS idx_voice_identity_links_is_active ON voice_identity_links(is_active);

CREATE OR REPLACE FUNCTION update_voice_sessions_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_voice_sessions_updated_at ON voice_sessions;
CREATE TRIGGER trigger_voice_sessions_updated_at
    BEFORE UPDATE ON voice_sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_voice_sessions_updated_at();

CREATE OR REPLACE FUNCTION update_voice_identity_links_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_voice_identity_links_updated_at ON voice_identity_links;
CREATE TRIGGER trigger_voice_identity_links_updated_at
    BEFORE UPDATE ON voice_identity_links
    FOR EACH ROW
    EXECUTE FUNCTION update_voice_identity_links_updated_at();

COMMENT ON TABLE voice_sessions IS 'Voice authentication sessions for voice assistant integration (Alexa, Google, Siri)';
COMMENT ON COLUMN voice_sessions.session_token IS 'Secret token identifying this voice session';
COMMENT ON COLUMN voice_sessions.voice_otp IS 'Numeric code spoken to user for verification (e.g., 8532)';
COMMENT ON COLUMN voice_sessions.linked_device_code IS 'Optional link to device authorization flow';
COMMENT ON COLUMN voice_sessions.status IS 'Session state: initiated, verified, expired, failed';

COMMENT ON TABLE voice_identity_links IS 'Permanent links between voice assistant accounts and user accounts for passwordless auth';
COMMENT ON COLUMN voice_identity_links.voice_user_id IS 'Platform-specific user ID (e.g., Alexa user amzn1.account.xxx)';
COMMENT ON COLUMN voice_identity_links.is_active IS 'Whether link is active (user can deactivate)';


CREATE OR REPLACE FUNCTION cleanup_expired_voice_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM voice_sessions
    WHERE expires_at < NOW() - INTERVAL '1 hour'
    AND status IN ('expired', 'failed', 'verified');

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_voice_sessions IS 'Deletes voice sessions older than 1 hour that are expired/failed/verified';
