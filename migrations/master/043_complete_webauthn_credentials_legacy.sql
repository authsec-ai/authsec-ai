-- Migration 063: Complete webauthn_credentials legacy table
-- Adds missing fields for full backward compatibility with migration 001
-- NOTE: This table only exists in tenant databases, not in admin DB

-- BEGIN; (removed - app manages transactions)

-- Only add columns if webauthn_credentials table exists (tenant DBs only)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_name = 'webauthn_credentials'
    ) THEN
        -- Add missing columns
        ALTER TABLE webauthn_credentials 
        ADD COLUMN IF NOT EXISTS attestation_type TEXT,
        ADD COLUMN IF NOT EXISTS transports TEXT[],
        ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN DEFAULT false,
        ADD COLUMN IF NOT EXISTS backup_state BOOLEAN DEFAULT false,
        ADD COLUMN IF NOT EXISTS sign_count BIGINT DEFAULT 0,
        ADD COLUMN IF NOT EXISTS user_present BOOLEAN DEFAULT false,
        ADD COLUMN IF NOT EXISTS user_verified BOOLEAN DEFAULT false,
        ADD COLUMN IF NOT EXISTS aaguid TEXT;

        -- Add index on user_id if not exists
        CREATE INDEX IF NOT EXISTS idx_webauthn_credentials_user_id ON webauthn_credentials(user_id);

        RAISE NOTICE 'webauthn_credentials table updated with missing fields';
    ELSE
        RAISE NOTICE 'webauthn_credentials table does not exist (admin DB) - skipping migration';
    END IF;
END $$;

-- COMMIT; (removed - app manages transactions)
