-- Migration 055: Synchronize backup flags across credential tables
-- This migration ensures the legacy webauthn_credentials table carries the same
-- backup flag values as the primary credentials table so WebAuthn validation
-- does not fail when credentials are loaded from either source.

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public'
          AND table_name = 'webauthn_credentials'
    ) THEN
        EXECUTE '
            ALTER TABLE webauthn_credentials
                ADD COLUMN IF NOT EXISTS backup_eligible BOOLEAN DEFAULT FALSE,
                ADD COLUMN IF NOT EXISTS backup_state BOOLEAN DEFAULT FALSE
        ';

        EXECUTE $sync$
            WITH matched AS (
                SELECT
                    wc.id,
                    c.backup_eligible,
                    c.backup_state
                FROM webauthn_credentials wc
                INNER JOIN credentials c
                    ON LOWER(wc.credential_id) = LOWER(encode(c.credential_id, 'hex'))
            )
            UPDATE webauthn_credentials AS wc
            SET
                backup_eligible = matched.backup_eligible,
                backup_state = matched.backup_state
            FROM matched
            WHERE wc.id = matched.id
        $sync$;
    ELSE
        RAISE NOTICE 'Skipping backup flag sync: webauthn_credentials table not present';
    END IF;
END $$;
