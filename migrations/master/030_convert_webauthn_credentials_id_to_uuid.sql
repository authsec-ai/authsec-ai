-- Migration 044: Convert webauthn_credentials.id to UUID
-- BEGIN; (removed - app manages transactions)
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'webauthn_credentials'
          AND column_name = 'id'
          AND data_type <> 'uuid'
    ) THEN
        ALTER TABLE webauthn_credentials
            ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
        UPDATE webauthn_credentials
           SET id_uuid = gen_random_uuid()
         WHERE id_uuid IS NULL;
        ALTER TABLE webauthn_credentials
            ALTER COLUMN id_uuid SET NOT NULL;
        ALTER TABLE webauthn_credentials
            DROP CONSTRAINT IF EXISTS webauthn_credentials_pkey;
        ALTER TABLE webauthn_credentials
            DROP COLUMN id;
        ALTER TABLE webauthn_credentials
            RENAME COLUMN id_uuid TO id;
        ALTER TABLE webauthn_credentials
            ADD CONSTRAINT webauthn_credentials_pkey PRIMARY KEY (id);
        DROP SEQUENCE IF EXISTS webauthn_credentials_id_seq;
    END IF;
END $$;

-- COMMIT; (removed - app manages transactions)
