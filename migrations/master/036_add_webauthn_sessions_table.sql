-- Migration 050: Add webauthn_sessions table
-- This migration adds the missing webauthn_sessions table that is required by the WebAuthn service

-- BEGIN; (removed - app manages transactions)

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create webauthn_sessions table for WebAuthn session management
CREATE TABLE IF NOT EXISTS webauthn_sessions (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    session_key VARCHAR(255) NOT NULL UNIQUE,
    challenge TEXT NOT NULL,
    user_id BYTEA NOT NULL,
    user_verification VARCHAR(50),
    extensions BYTEA,
    cred_params BYTEA,
    allowed_credential_ids BYTEA,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL
);
-- Add indexes for performance
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_session_key ON webauthn_sessions(session_key);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_user_id ON webauthn_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires_at ON webauthn_sessions(expires_at);

-- Clean up expired sessions (optional cleanup for existing installations)
-- This will be handled by the application, but we can set up the table ready for it
COMMENT ON TABLE webauthn_sessions IS 'Stores WebAuthn session data for authentication flows';

-- COMMIT; (removed - app manages transactions)