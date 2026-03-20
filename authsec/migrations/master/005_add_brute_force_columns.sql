-- Migration 005: Add brute force protection columns to users table
-- Idempotent: uses DO blocks with exception handling for duplicate columns

DO $$ BEGIN
    ALTER TABLE users ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0;
EXCEPTION WHEN duplicate_column THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE users ADD COLUMN IF NOT EXISTS account_locked_at TIMESTAMP WITH TIME ZONE;
EXCEPTION WHEN duplicate_column THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_required BOOLEAN DEFAULT false;
EXCEPTION WHEN duplicate_column THEN NULL;
END $$;

-- Partial index for locked accounts (matches 000_comprehensive_base_schema.sql)
CREATE INDEX IF NOT EXISTS idx_users_account_locked ON users(account_locked_at) WHERE (account_locked_at IS NOT NULL);
