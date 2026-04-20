
CREATE TABLE IF NOT EXISTS device_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    client_id UUID, -- Stored from tenant_mappings, no FK constraint
    device_code VARCHAR(128) UNIQUE NOT NULL,
    user_code VARCHAR(16) UNIQUE NOT NULL,
    verification_uri TEXT NOT NULL, -- e.g., https://authsec.dev/activate
    verification_uri_complete TEXT, -- Optional pre-filled URI with user_code
    user_id UUID, -- References users(id) in tenant DB
    user_email TEXT, -- Cached for convenience

    status VARCHAR(20) DEFAULT 'pending' NOT NULL, -- pending/authorized/denied/expired/consumed


    scopes JSONB DEFAULT '[]'::jsonb, -- Requested OAuth scopes

    device_info JSONB, -- Optional device metadata (name, type, IP, etc.)


    expires_at TIMESTAMP NOT NULL, -- Typically 10-15 minutes from creation
    last_polled_at TIMESTAMP, -- Last time device polled for token
    authorized_at TIMESTAMP, -- When user approved


    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),


    CONSTRAINT chk_device_codes_status CHECK (status IN ('pending', 'authorized', 'denied', 'expired', 'consumed'))
);

CREATE INDEX IF NOT EXISTS idx_device_codes_device_code ON device_codes(device_code);
CREATE INDEX IF NOT EXISTS idx_device_codes_user_code ON device_codes(user_code);
CREATE INDEX IF NOT EXISTS idx_device_codes_tenant_id ON device_codes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_device_codes_status ON device_codes(status);
CREATE INDEX IF NOT EXISTS idx_device_codes_expires_at ON device_codes(expires_at);
CREATE INDEX IF NOT EXISTS idx_device_codes_user_id ON device_codes(user_id);


CREATE OR REPLACE FUNCTION update_device_codes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trigger_device_codes_updated_at ON device_codes;
CREATE TRIGGER trigger_device_codes_updated_at
    BEFORE UPDATE ON device_codes
    FOR EACH ROW
    EXECUTE FUNCTION update_device_codes_updated_at();


COMMENT ON TABLE device_codes IS 'OAuth 2.0 Device Authorization Grant (RFC 8628) - stores device flow authorization requests';
COMMENT ON COLUMN device_codes.device_code IS 'Long secret code for device polling (128 chars)';
COMMENT ON COLUMN device_codes.user_code IS 'Short human-readable code shown to user (8-16 chars, e.g., WDJB-MJHT)';
COMMENT ON COLUMN device_codes.verification_uri IS 'URL where user activates device (e.g., https://authsec.dev/activate)';
COMMENT ON COLUMN device_codes.status IS 'Authorization state: pending (waiting), authorized (approved), denied (rejected), expired (timeout), consumed (token issued)';
COMMENT ON COLUMN device_codes.scopes IS 'JSON array of requested OAuth scopes (e.g., ["openid", "email", "profile"])';

CREATE OR REPLACE FUNCTION cleanup_expired_device_codes()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM device_codes
    WHERE expires_at < NOW() - INTERVAL '24 hours'
    AND status IN ('expired', 'consumed', 'denied');

    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

COMMENT ON FUNCTION cleanup_expired_device_codes IS 'Deletes device codes older than 24 hours that are expired/consumed/denied';
