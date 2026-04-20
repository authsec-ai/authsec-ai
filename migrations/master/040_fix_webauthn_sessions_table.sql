-- Migration 056: Fix WebAuthn Sessions Table Schema
-- This migration ensures the webauthn_sessions table has all required columns
-- and fixes any schema inconsistencies.
-- Note: Table creation is handled by migration 050

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Add missing columns if they don't exist
ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS session_key VARCHAR(255);

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS challenge TEXT;

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS user_id BYTEA;

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS user_verification VARCHAR(50);

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS extensions BYTEA;

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS cred_params BYTEA;

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS allowed_credential_ids BYTEA;

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

ALTER TABLE webauthn_sessions 
ADD COLUMN IF NOT EXISTS expires_at TIMESTAMP WITH TIME ZONE;

-- First, verify there are no NULL session keys so we do not silently remove data
DO $$
DECLARE
    null_session_keys BIGINT;
BEGIN
    SELECT COUNT(*) INTO null_session_keys FROM webauthn_sessions WHERE session_key IS NULL;
    IF null_session_keys > 0 THEN
        RAISE EXCEPTION 'Migration 056 aborted: Found % webauthn_sessions rows with NULL session_key. Please correct data manually before rerunning.', null_session_keys;
    END IF;
END $$;

-- Then add constraints
ALTER TABLE webauthn_sessions 
ALTER COLUMN session_key SET NOT NULL;

-- Add unique constraint if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'webauthn_sessions_session_key_key'
    ) THEN
        ALTER TABLE webauthn_sessions ADD CONSTRAINT webauthn_sessions_session_key_key UNIQUE (session_key);
    END IF;
END $$;

-- Ensure other NOT NULL constraints
DO $$
DECLARE
    missing_challenge BIGINT;
    missing_user_id BIGINT;
    missing_expires_at BIGINT;
BEGIN
    SELECT COUNT(*) INTO missing_challenge FROM webauthn_sessions WHERE challenge IS NULL;
    IF missing_challenge > 0 THEN
        RAISE EXCEPTION 'Migration 056 aborted: Found % webauthn_sessions rows with NULL challenge. Please correct data manually before rerunning.', missing_challenge;
    END IF;

    SELECT COUNT(*) INTO missing_user_id FROM webauthn_sessions WHERE user_id IS NULL;
    IF missing_user_id > 0 THEN
        RAISE EXCEPTION 'Migration 056 aborted: Found % webauthn_sessions rows with NULL user_id. Please correct data manually before rerunning.', missing_user_id;
    END IF;

    SELECT COUNT(*) INTO missing_expires_at FROM webauthn_sessions WHERE expires_at IS NULL;
    IF missing_expires_at > 0 THEN
        RAISE EXCEPTION 'Migration 056 aborted: Found % webauthn_sessions rows with NULL expires_at. Please correct data manually before rerunning.', missing_expires_at;
    END IF;
END $$;

ALTER TABLE webauthn_sessions 
ALTER COLUMN challenge SET NOT NULL;

ALTER TABLE webauthn_sessions 
ALTER COLUMN user_id SET NOT NULL;

ALTER TABLE webauthn_sessions 
ALTER COLUMN expires_at SET NOT NULL;

-- Create indexes for efficient queries (IF NOT EXISTS)
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_session_key ON webauthn_sessions(session_key);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires_at ON webauthn_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_created_at ON webauthn_sessions(created_at);

-- Add table and column comments
COMMENT ON TABLE webauthn_sessions IS 'Stores WebAuthn session data for registration and authentication ceremonies';
COMMENT ON COLUMN webauthn_sessions.session_key IS 'Unique session identifier in format: operation:email:tenant_id';
COMMENT ON COLUMN webauthn_sessions.challenge IS 'Base64-encoded WebAuthn challenge';
COMMENT ON COLUMN webauthn_sessions.user_id IS 'Binary user identifier for the WebAuthn ceremony';
COMMENT ON COLUMN webauthn_sessions.user_verification IS 'Required user verification level (required, preferred, discouraged)';
COMMENT ON COLUMN webauthn_sessions.extensions IS 'JSON-encoded WebAuthn extensions data';
COMMENT ON COLUMN webauthn_sessions.cred_params IS 'JSON-encoded credential parameters for registration';
COMMENT ON COLUMN webauthn_sessions.allowed_credential_ids IS 'JSON-encoded list of allowed credential IDs for authentication';
COMMENT ON COLUMN webauthn_sessions.expires_at IS 'Session expiration timestamp (typically 10 minutes from creation)';
