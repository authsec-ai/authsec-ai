-- Migration 064: Remove client_id foreign keys from MFA and credentials tables
-- Ensures application-side ownership of client associations without deleting data

-- BEGIN; (removed - app manages transactions)

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'mfa_methods'
    ) THEN
        ALTER TABLE mfa_methods
            DROP CONSTRAINT IF EXISTS fk_mfa_methods_client_id;
        RAISE NOTICE 'Dropped fk_mfa_methods_client_id from mfa_methods (if present)';
    ELSE
        RAISE NOTICE 'mfa_methods table not found - skipping client_id foreign key drop';
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'credentials'
    ) THEN
        ALTER TABLE credentials
            DROP CONSTRAINT IF EXISTS fk_credentials_client_id;
        RAISE NOTICE 'Dropped fk_credentials_client_id from credentials (if present)';
    ELSE
        RAISE NOTICE 'credentials table not found - skipping client_id foreign key drop';
    END IF;
END $$;

-- COMMIT; (removed - app manages transactions)
