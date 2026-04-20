-- Migration 061: Harden webauthn_sessions schema (tenant alignment helper)
-- This migration adds missing columns and constraints to existing webauthn_sessions tables
-- Note: Table creation is handled by migration 050

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Add missing columns if they don't exist
DO $$
BEGIN
    -- Add missing columns if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'session_key') THEN
        ALTER TABLE webauthn_sessions ADD COLUMN session_key VARCHAR(255);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'user_verification') THEN
        ALTER TABLE webauthn_sessions ADD COLUMN user_verification VARCHAR(50);
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'extensions') THEN
        ALTER TABLE webauthn_sessions ADD COLUMN extensions BYTEA;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'cred_params') THEN
        ALTER TABLE webauthn_sessions ADD COLUMN cred_params BYTEA;
    END IF;
    
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'allowed_credential_ids') THEN
        ALTER TABLE webauthn_sessions ADD COLUMN allowed_credential_ids BYTEA;
    END IF;
    
    -- Change user_id column type from uuid to bytea if needed
    IF (SELECT data_type FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'user_id') = 'uuid' THEN
        ALTER TABLE webauthn_sessions ALTER COLUMN user_id TYPE BYTEA USING user_id::text::bytea;
    END IF;
    
    -- Make session_key NOT NULL if it exists and is not already NOT NULL
    IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'webauthn_sessions' AND column_name = 'session_key' AND is_nullable = 'YES') THEN
        ALTER TABLE webauthn_sessions ALTER COLUMN session_key SET NOT NULL;
    END IF;
    
    -- Add unique constraint on session_key if it doesn't exist
    IF NOT EXISTS (
        SELECT 1 
        FROM pg_constraint 
        WHERE conrelid = 'webauthn_sessions'::regclass 
          AND contype = 'u'
    ) THEN
        ALTER TABLE webauthn_sessions ADD CONSTRAINT webauthn_sessions_session_key_unique UNIQUE (session_key);
    END IF;
END $$;

-- Create missing indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_session_key ON webauthn_sessions(session_key);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_user_id ON webauthn_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_expires_at ON webauthn_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_webauthn_sessions_created_at ON webauthn_sessions(created_at);
