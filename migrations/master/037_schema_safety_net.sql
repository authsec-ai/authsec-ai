-- Migration 051: Schema safety net for WebAuthn multi-tenancy
-- Ensures critical WebAuthn-related extensions and columns exist without clobbering production data

-- Required extensions for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Align core columns that earlier installs might be missing
ALTER TABLE users
    ADD COLUMN IF NOT EXISTS role_name VARCHAR(255);

ALTER TABLE credentials
    ADD COLUMN IF NOT EXISTS aaguid UUID;

-- Defensive index creation for role lookups when the column exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'user_roles'
          AND column_name = 'role_name'
    ) THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_user_roles_role_name ON user_roles(role_name)';
    END IF;
END $$;
