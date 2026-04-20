-- Migration 043: Convert OTP and pending registration identifiers to UUID
-- NOTE: Removed explicit BEGIN/COMMIT as migrations run within app-managed transactions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Convert otp_entries.id to UUID if still integer-based
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'otp_entries'
          AND column_name = 'id'
          AND data_type <> 'uuid'
    ) THEN
        ALTER TABLE otp_entries
            ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
        UPDATE otp_entries
           SET id_uuid = gen_random_uuid()
         WHERE id_uuid IS NULL;
        ALTER TABLE otp_entries
            ALTER COLUMN id_uuid SET NOT NULL;
        ALTER TABLE otp_entries
            DROP CONSTRAINT IF EXISTS otp_entries_pkey;
        ALTER TABLE otp_entries
            DROP COLUMN id;
        ALTER TABLE otp_entries
            RENAME COLUMN id_uuid TO id;
        ALTER TABLE otp_entries
            ADD CONSTRAINT otp_entries_pkey PRIMARY KEY (id);
        DROP SEQUENCE IF EXISTS otp_entries_id_seq;
    END IF;
END $$;

-- Convert pending_registrations.id to UUID if still integer-based
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'pending_registrations'
          AND column_name = 'id'
          AND data_type <> 'uuid'
    ) THEN
        ALTER TABLE pending_registrations
            ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
        UPDATE pending_registrations
           SET id_uuid = gen_random_uuid()
         WHERE id_uuid IS NULL;
        ALTER TABLE pending_registrations
            ALTER COLUMN id_uuid SET NOT NULL;
        ALTER TABLE pending_registrations
            DROP CONSTRAINT IF EXISTS pending_registrations_pkey;
        ALTER TABLE pending_registrations
            DROP COLUMN id;
        ALTER TABLE pending_registrations
            RENAME COLUMN id_uuid TO id;
        ALTER TABLE pending_registrations
            ADD CONSTRAINT pending_registrations_pkey PRIMARY KEY (id);
        DROP SEQUENCE IF EXISTS pending_registrations_id_seq;
    END IF;
END $$;
