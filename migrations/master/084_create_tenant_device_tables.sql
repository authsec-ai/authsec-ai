-- Migration: Add tenant device tables for CIBA and TOTP authentication
-- This migration adds device-specific tables to tenant databases
-- Version: 131_create_tenant_device_tables.sql

-- ========================================
-- Tenant Device Tokens for Push Notifications
-- ========================================

CREATE TABLE IF NOT EXISTS tenant_device_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    device_token VARCHAR(500) NOT NULL UNIQUE,
    platform VARCHAR(20) NOT NULL, -- ios, android

    device_name VARCHAR(100),
    device_model VARCHAR(100),
    app_version VARCHAR(20),
    os_version VARCHAR(20),

    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    last_used BIGINT,

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_tenant_device_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,
    CONSTRAINT fk_tenant_device_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_tenant_device_token UNIQUE (device_token, tenant_id),
    CONSTRAINT fk_tenant_device_token UNIQUE (device_token, tenant_id)
);

CREATE INDEX IF NOT EXISTS idx_tenant_device_token_user ON tenant_device_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_device_token_tenant ON tenant_device_tokens(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_device_token_active ON tenant_device_tokens(is_active);
CREATE INDEX IF NOT EXISTS idx_tenant_device_token_device_token ON tenant_device_tokens(device_token);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'uq_tenant_device_id_tenant'
          AND conrelid = 'tenant_device_tokens'::regclass
    ) THEN
        ALTER TABLE tenant_device_tokens ADD CONSTRAINT uq_tenant_device_id_tenant UNIQUE (id, tenant_id);
    END IF;
END $$;


CREATE TABLE IF NOT EXISTS tenant_ciba_auth_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auth_req_id VARCHAR(255) NOT NULL UNIQUE,

    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    user_email VARCHAR(255) NOT NULL,

    client_id UUID,

    device_token_id UUID NOT NULL,
    binding_message VARCHAR(255),

    scopes JSONB DEFAULT '[]',

    status VARCHAR(50) DEFAULT 'pending' NOT NULL,

    biometric_verified BOOLEAN DEFAULT FALSE,
   
    expires_at BIGINT NOT NULL,
    created_at BIGINT NOT NULL,
    responded_at BIGINT,
    last_polled_at BIGINT,

    CONSTRAINT fk_tenant_ciba_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,
    
    CONSTRAINT fk_tenant_ciba_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,
    
    CONSTRAINT fk_tenant_ciba_device FOREIGN KEY (device_token_id, tenant_id)
        REFERENCES tenant_device_tokens(id, tenant_id) ON DELETE CASCADE
);


CREATE INDEX IF NOT EXISTS idx_tenant_ciba_auth_req_id ON tenant_ciba_auth_requests(auth_req_id);
CREATE INDEX IF NOT EXISTS idx_tenant_ciba_user ON tenant_ciba_auth_requests(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_ciba_tenant ON tenant_ciba_auth_requests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_ciba_status ON tenant_ciba_auth_requests(status);
CREATE INDEX IF NOT EXISTS idx_tenant_ciba_expires_at ON tenant_ciba_auth_requests(expires_at);
CREATE INDEX IF NOT EXISTS idx_tenant_ciba_created_at ON tenant_ciba_auth_requests(created_at);



CREATE TABLE IF NOT EXISTS tenant_totp_secrets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,

    secret VARCHAR(64) NOT NULL,

    device_name VARCHAR(100),
    device_type VARCHAR(50) DEFAULT 'generic', -- generic, google_auth, microsoft_auth, authy
    last_used BIGINT,
   
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    is_primary BOOLEAN DEFAULT FALSE, -- Preferred device for TOTP

    created_at BIGINT NOT NULL,
    updated_at BIGINT NOT NULL,

    CONSTRAINT fk_tenant_totp_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,
    
    CONSTRAINT fk_tenant_totp_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE,

    UNIQUE(user_id, tenant_id, is_primary) -- Only one primary device per user
);


CREATE INDEX IF NOT EXISTS idx_tenant_totp_user ON tenant_totp_secrets(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_totp_tenant ON tenant_totp_secrets(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_totp_active ON tenant_totp_secrets(is_active);
CREATE INDEX IF NOT EXISTS idx_tenant_totp_primary ON tenant_totp_secrets(is_primary);
CREATE INDEX IF NOT EXISTS idx_tenant_totp_created_at ON tenant_totp_secrets(created_at);



CREATE TABLE IF NOT EXISTS tenant_totp_backup_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    tenant_id UUID NOT NULL,
    code VARCHAR(64) NOT NULL UNIQUE, -- Hashed code
    is_used BOOLEAN DEFAULT FALSE NOT NULL,
    created_at BIGINT NOT NULL,
    used_at BIGINT,

    CONSTRAINT fk_tenant_backup_user FOREIGN KEY (user_id, tenant_id)
        REFERENCES users(id, tenant_id) ON DELETE CASCADE,
    
    CONSTRAINT fk_tenant_backup_tenant FOREIGN KEY (tenant_id)
        REFERENCES tenants(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_tenant_backup_user ON tenant_totp_backup_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_tenant_backup_tenant ON tenant_totp_backup_codes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_backup_used ON tenant_totp_backup_codes(is_used);
CREATE INDEX IF NOT EXISTS idx_tenant_backup_code ON tenant_totp_backup_codes(code);
CREATE INDEX IF NOT EXISTS idx_tenant_backup_created_at ON tenant_totp_backup_codes(created_at);


COMMENT ON TABLE tenant_device_tokens IS 'Stores push notification device tokens for tenant users';
COMMENT ON TABLE tenant_ciba_auth_requests IS 'Tracks CIBA push notification authentication requests for tenant users';
COMMENT ON TABLE tenant_totp_secrets IS 'Stores TOTP authenticator secrets for tenant users';
COMMENT ON TABLE tenant_totp_backup_codes IS 'Stores backup recovery codes for tenant user TOTP devices';


COMMENT ON COLUMN tenant_device_tokens.device_token IS 'FCM/APNS push notification token from mobile device';
COMMENT ON COLUMN tenant_device_tokens.platform IS 'Mobile platform: ios or android';
COMMENT ON COLUMN tenant_device_tokens.device_name IS 'User-friendly device name (e.g., "John''s iPhone")';
COMMENT ON COLUMN tenant_device_tokens.device_model IS 'Device model (e.g., "iPhone 14 Pro")';
COMMENT ON COLUMN tenant_device_tokens.app_version IS 'AuthSec Mobile app version';
COMMENT ON COLUMN tenant_device_tokens.os_version IS 'Operating system version';
COMMENT ON COLUMN tenant_device_tokens.last_used IS 'Unix timestamp of last authentication using this device';


COMMENT ON COLUMN tenant_ciba_auth_requests.auth_req_id IS 'Unique request ID for CIBA authentication flow';
COMMENT ON COLUMN tenant_ciba_auth_requests.device_token_id IS 'Device to send push notification to';
COMMENT ON COLUMN tenant_ciba_auth_requests.binding_message IS 'Message displayed on user device during approval';
COMMENT ON COLUMN tenant_ciba_auth_requests.scopes IS 'OAuth scopes requested by client';
COMMENT ON COLUMN tenant_ciba_auth_requests.status IS 'Request status: pending, approved, denied, expired, consumed';
COMMENT ON COLUMN tenant_ciba_auth_requests.biometric_verified IS 'Whether user used biometric verification';
COMMENT ON COLUMN tenant_ciba_auth_requests.responded_at IS 'When user approved/denied the request';
COMMENT ON COLUMN tenant_ciba_auth_requests.last_polled_at IS 'When client last polled for token';

COMMENT ON COLUMN tenant_totp_secrets.secret IS 'Base32 encoded TOTP secret (never exposed in API responses)';
COMMENT ON COLUMN tenant_totp_secrets.device_name IS 'User-friendly device name for TOTP authenticator';
COMMENT ON COLUMN tenant_totp_secrets.device_type IS 'Type of TOTP authenticator app';
COMMENT ON COLUMN tenant_totp_secrets.is_primary IS 'Primary device for TOTP login';
COMMENT ON COLUMN tenant_totp_secrets.last_used IS 'Unix timestamp of last TOTP verification';

COMMENT ON COLUMN tenant_totp_backup_codes.code IS 'SHA-1 hash of backup recovery code';
COMMENT ON COLUMN tenant_totp_backup_codes.is_used IS 'Whether backup code has been used';
COMMENT ON COLUMN tenant_totp_backup_codes.used_at IS 'When backup code was used';

CREATE INDEX IF NOT EXISTS idx_tenant_ciba_user_status ON tenant_ciba_auth_requests(user_id, status);
CREATE INDEX IF NOT EXISTS idx_tenant_totp_user_active ON tenant_totp_secrets(user_id, is_active);
CREATE INDEX IF NOT EXISTS idx_tenant_backup_user_unused ON tenant_totp_backup_codes(user_id, is_used);