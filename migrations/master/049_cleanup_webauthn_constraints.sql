-- Migration: Remove redundant WebAuthn session constraints/index
-- Purpose: Avoid duplicate unique constraints and extra btree index on session_key

DO $$
BEGIN
    RAISE NOTICE '=== MIGRATION 075: Cleaning up WebAuthn session constraints ===';

    IF EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE constraint_name = 'webauthn_sessions_session_key_unique'
          AND table_name = 'webauthn_sessions'
          AND constraint_type = 'UNIQUE'
    ) THEN
        ALTER TABLE webauthn_sessions DROP CONSTRAINT webauthn_sessions_session_key_unique;
        RAISE NOTICE 'Dropped duplicate unique constraint webauthn_sessions_session_key_unique';
    ELSE
        RAISE NOTICE 'Constraint webauthn_sessions_session_key_unique not found, skipping';
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND tablename = 'webauthn_sessions'
          AND indexname = 'idx_webauthn_sessions_session_key'
    ) THEN
        DROP INDEX public.idx_webauthn_sessions_session_key;
        RAISE NOTICE 'Dropped redundant index idx_webauthn_sessions_session_key';
    ELSE
        RAISE NOTICE 'Index idx_webauthn_sessions_session_key not found, skipping';
    END IF;

    RAISE NOTICE '=== MIGRATION 075 COMPLETED ===';
END $$;
