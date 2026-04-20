-- Migration 077: Remove all unique constraints on users table
-- Description: Removes all unique constraints on users.email to allow duplicate emails
-- This allows the same email to be used across different clients, tenants, and contexts

-- BEGIN; (removed - app manages transactions)

-- Drop all possible unique constraints on users.email
DO $$
BEGIN
    -- Drop users_client_email_unique constraint
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_client_email_unique'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_client_email_unique;
        RAISE NOTICE 'Dropped constraint: users_client_email_unique';
    END IF;

    -- Drop users_email_tenant_unique constraint
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_email_tenant_unique'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_tenant_unique;
        RAISE NOTICE 'Dropped constraint: users_email_tenant_unique';
    END IF;

    -- Drop uni_users_email constraint
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'uni_users_email'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT uni_users_email;
        RAISE NOTICE 'Dropped constraint: uni_users_email';
    END IF;

    -- Drop users_email_key constraint (from original table creation)
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE table_name = 'users' AND constraint_name = 'users_email_key'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_key;
        RAISE NOTICE 'Dropped constraint: users_email_key';
    END IF;

    RAISE NOTICE 'All unique constraints on users.email have been removed';
END $$;

-- Drop all unique indexes on users.email
DO $$
BEGIN
    -- Drop idx_users_email if it's a unique index
    IF EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'users' AND indexname = 'idx_users_email'
    ) THEN
        DROP INDEX IF EXISTS idx_users_email;
        RAISE NOTICE 'Dropped unique index: idx_users_email';
    END IF;

    -- Drop idx_users_email_unique
    DROP INDEX IF EXISTS idx_users_email_unique;

    -- Drop users_email_unique_idx
    DROP INDEX IF EXISTS users_email_unique_idx;

    -- Drop users_email_tenant_unique_idx
    DROP INDEX IF EXISTS users_email_tenant_unique_idx;

    RAISE NOTICE 'All unique indexes on users.email have been removed';
END $$;

-- Recreate idx_users_email as a non-unique index for performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- Create supporting indexes for common query patterns (non-unique)
CREATE INDEX IF NOT EXISTS idx_users_client_email ON users (client_id, email);
CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users (tenant_id, email);
CREATE INDEX IF NOT EXISTS idx_users_client_email_lower ON users (client_id, LOWER(email));

-- Log completion
DO $$
BEGIN
    RAISE NOTICE '=== Migration 077 Completed ===';
    RAISE NOTICE 'All unique constraints and indexes on users.email have been removed';
    RAISE NOTICE 'Non-unique indexes created for performance';
    RAISE NOTICE 'Users table now allows duplicate emails across all contexts';
END $$;

-- COMMIT; (removed - app manages transactions)
