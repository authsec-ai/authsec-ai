-- Migration 045: Drop legacy webauthn_credentials table
-- BEGIN; (removed - app manages transactions)

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'webauthn_credentials'
    ) THEN
        -- DROP TABLE webauthn_credentials; -- Disabled to preserve legacy data
    END IF;
END $$;

-- COMMIT; (removed - app manages transactions)
