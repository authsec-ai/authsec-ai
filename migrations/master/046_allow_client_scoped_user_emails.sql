-- Migration 068: Allow duplicate tenant user emails across clients
-- Drops legacy tenant-scoped unique constraints and replaces them with client-scoped uniqueness.

-- Remove historical unique constraints on users.email
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_email_tenant_unique'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_tenant_unique;
        RAISE NOTICE 'Dropped constraint users_email_tenant_unique';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'uni_users_email'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT uni_users_email;
        RAISE NOTICE 'Dropped constraint uni_users_email';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_email_key'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_key;
        RAISE NOTICE 'Dropped constraint users_email_key';
    END IF;
END $$;

-- Remove legacy indexes that enforced email uniqueness
DROP INDEX IF EXISTS idx_users_email_unique;
DROP INDEX IF EXISTS users_email_unique_idx;
DROP INDEX IF EXISTS users_email_tenant_unique_idx;

-- REMOVED: No longer adding client-scoped unique constraint to allow duplicate emails even within same client
-- This allows multiple user records with the same email across different contexts
DO $$
BEGIN
    -- Drop the constraint if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_client_email_unique'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_client_email_unique;
        RAISE NOTICE 'Dropped constraint users_client_email_unique';
    END IF;
END $$;

-- Ensure supporting index exists for lookup performance (non-unique)
CREATE INDEX IF NOT EXISTS idx_users_client_email_lower
    ON users (client_id, LOWER(email));
