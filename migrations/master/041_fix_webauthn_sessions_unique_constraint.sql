-- Migration 058: Fix WebAuthn Sessions Table Unique Constraint
-- This migration ensures the webauthn_sessions table in tenant databases 
-- has the proper unique constraint on session_key column
-- Note: Table creation is handled by migration 050

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Validate data instead of deleting rows so operators can correct issues explicitly
DO $$
DECLARE
    missing_session_keys BIGINT;
    missing_challenges BIGINT;
    missing_user_ids BIGINT;
    missing_expiration BIGINT;
BEGIN
    SELECT COUNT(*) INTO missing_session_keys FROM webauthn_sessions WHERE session_key IS NULL OR session_key = '';
    IF missing_session_keys > 0 THEN
        RAISE EXCEPTION 'Migration 058 aborted: Found % webauthn_sessions rows without a valid session_key. Please correct data manually before rerunning.', missing_session_keys;
    END IF;

    SELECT COUNT(*) INTO missing_challenges FROM webauthn_sessions WHERE challenge IS NULL OR challenge = '';
    IF missing_challenges > 0 THEN
        RAISE EXCEPTION 'Migration 058 aborted: Found % webauthn_sessions rows without a valid challenge. Please correct data manually before rerunning.', missing_challenges;
    END IF;

    SELECT COUNT(*) INTO missing_user_ids FROM webauthn_sessions WHERE user_id IS NULL;
    IF missing_user_ids > 0 THEN
        RAISE EXCEPTION 'Migration 058 aborted: Found % webauthn_sessions rows without a user_id. Please correct data manually before rerunning.', missing_user_ids;
    END IF;

    SELECT COUNT(*) INTO missing_expiration FROM webauthn_sessions WHERE expires_at IS NULL;
    IF missing_expiration > 0 THEN
        RAISE EXCEPTION 'Migration 058 aborted: Found % webauthn_sessions rows without an expires_at timestamp. Please correct data manually before rerunning.', missing_expiration;
    END IF;
END $$;

-- Set columns to NOT NULL if they aren't already
DO $$
BEGIN
    -- Add NOT NULL constraint to session_key if it doesn't have it
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'webauthn_sessions' 
        AND column_name = 'session_key' 
        AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE webauthn_sessions ALTER COLUMN session_key SET NOT NULL;
    END IF;

    -- Add NOT NULL constraint to challenge if it doesn't have it
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'webauthn_sessions' 
        AND column_name = 'challenge' 
        AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE webauthn_sessions ALTER COLUMN challenge SET NOT NULL;
    END IF;

    -- Add NOT NULL constraint to user_id if it doesn't have it
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'webauthn_sessions' 
        AND column_name = 'user_id' 
        AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE webauthn_sessions ALTER COLUMN user_id SET NOT NULL;
    END IF;

    -- Add NOT NULL constraint to expires_at if it doesn't have it
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'webauthn_sessions' 
        AND column_name = 'expires_at' 
        AND is_nullable = 'YES'
    ) THEN
        ALTER TABLE webauthn_sessions ALTER COLUMN expires_at SET NOT NULL;
    END IF;
END $$;

-- Add unique constraint on session_key if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'webauthn_sessions_session_key_key'
        AND conrelid = 'webauthn_sessions'::regclass
    ) THEN
        IF EXISTS (
            SELECT 1
            FROM webauthn_sessions
            GROUP BY session_key
            HAVING COUNT(*) > 1
        ) THEN
            RAISE EXCEPTION 'Migration 058 aborted: Duplicate session_key values detected in webauthn_sessions. Please de-duplicate data manually before rerunning.';
        END IF;

        ALTER TABLE webauthn_sessions ADD CONSTRAINT webauthn_sessions_session_key_key UNIQUE (session_key);
    END IF;
END $$;

-- Create indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_session_key ON webauthn_sessions(session_key);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires_at ON webauthn_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_created_at ON webauthn_sessions(created_at);
