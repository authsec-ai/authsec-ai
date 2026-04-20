-- Migration: Add TOTP (Two-Factor Authentication) Tables
-- This adds support for TOTP-based 2FA using authenticator apps (Google Auth, Microsoft Auth, etc.)

-- TOTP Secrets Table: Stores registered authenticator devices
CREATE TABLE IF NOT EXISTS totp_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    secret VARCHAR(64) NOT NULL, -- Base32-encoded TOTP secret
    device_name VARCHAR(100) NOT NULL, -- e.g., "My iPhone", "Work Laptop"
    device_type VARCHAR(50) DEFAULT 'generic', -- generic, google_auth, microsoft_auth, authy
    last_used BIGINT, -- Unix epoch timestamp
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE NOT NULL, -- Preferred device for TOTP
    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_totp_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,

    CONSTRAINT fk_totp_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_totp_user ON totp_secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_totp_tenant ON totp_secrets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_totp_active ON totp_secrets(is_active, is_primary);

CREATE TABLE IF NOT EXISTS totp_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    code VARCHAR(64) NOT NULL UNIQUE, -- Hashed recovery code (SHA1 = 40 chars, SHA256 = 64 chars)
    is_used BOOLEAN DEFAULT FALSE NOT NULL,
    created_at BIGINT NOT NULL,
    used_at BIGINT,

    CONSTRAINT fk_backup_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,

    CONSTRAINT fk_backup_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_backup_user ON totp_backup_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_backup_tenant ON totp_backup_codes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_backup_used ON totp_backup_codes(is_used);

CREATE UNIQUE INDEX IF NOT EXISTS idx_totp_primary_device
    ON totp_secrets(user_id, tenant_id)
    WHERE is_primary = TRUE;

COMMENT ON TABLE totp_secrets IS 'Stores TOTP authenticator devices registered by users for 2FA';
COMMENT ON TABLE totp_backup_codes IS 'Stores recovery codes for TOTP 2FA';
COMMENT ON COLUMN totp_secrets.secret IS 'Base32-encoded TOTP secret (never exposed in API responses)';
COMMENT ON COLUMN totp_backup_codes.code IS 'SHA1-hashed recovery code (never exposed in plain)';
