-- Migration: Fix users table email constraints in tenant databases
-- Date: 2025-11-17
-- Purpose: Change from UNIQUE(email) to UNIQUE(email, tenant_id)
-- This allows same email across different tenants while maintaining uniqueness within a tenant

-- BEGIN; (removed - app manages transactions)

-- Step 1: Drop old unique constraints on email (if exist)
DO $$ 
BEGIN
    -- Drop various possible email constraints
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'users_email_key' 
        AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users DROP CONSTRAINT users_email_key;
        RAISE NOTICE 'Dropped constraint: users_email_key';
    END IF;

    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'uni_users_email'
        AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users DROP CONSTRAINT uni_users_email;
        RAISE NOTICE 'Dropped constraint: uni_users_email';
    END IF;
END $$;

-- Step 2: Drop old unique indexes on email (if exist)
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS users_email_unique;

-- Step 3: Add composite unique constraint on (email, tenant_id)
DO $$
BEGIN
    -- Check if constraint already exists
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'users_email_tenant_unique'
        AND conrelid = 'users'::regclass
    ) THEN
        ALTER TABLE users 
        ADD CONSTRAINT users_email_tenant_unique UNIQUE (email, tenant_id);
        RAISE NOTICE 'Added constraint: users_email_tenant_unique';
    ELSE
        RAISE NOTICE 'Constraint users_email_tenant_unique already exists';
    END IF;
    
    -- Alternative: If using client_id instead of tenant_id
    -- Check if users_client_email_unique exists, if so, keep it
    IF EXISTS (
        SELECT 1 FROM pg_constraint 
        WHERE conname = 'users_client_email_unique'
        AND conrelid = 'users'::regclass
    ) THEN
        RAISE NOTICE 'Found existing users_client_email_unique constraint - keeping it';
    END IF;
END $$;

-- Step 4: Create indexes for performance
-- Index for email lookups (non-unique, as email can appear multiple times across tenants)
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);

-- Index for tenant-scoped email lookups
CREATE INDEX IF NOT EXISTS idx_users_email_tenant ON users(email, tenant_id);

-- Index for client-scoped email lookups (if client_id is used)
CREATE INDEX IF NOT EXISTS idx_users_email_client ON users(email, client_id);

-- COMMIT; (removed - app manages transactions)

-- Verification queries
DO $$
DECLARE
    constraint_count INTEGER;
    index_count INTEGER;
BEGIN
    -- Check constraints
    SELECT COUNT(*) INTO constraint_count
    FROM pg_constraint
    WHERE conrelid = 'users'::regclass
      AND conname IN ('users_email_tenant_unique', 'users_client_email_unique');
    
    -- Check indexes
    SELECT COUNT(*) INTO index_count
    FROM pg_indexes
    WHERE tablename = 'users'
      AND indexname LIKE '%email%';
    
    IF constraint_count >= 1 AND index_count >= 2 THEN
        RAISE NOTICE '✓ Migration successful: users email constraint updated';
        RAISE NOTICE '✓ Found % email-related constraint(s)', constraint_count;
        RAISE NOTICE '✓ Found % email-related index(es)', index_count;
    ELSE
        RAISE WARNING 'Migration verification: constraint_count=%, index_count=%', constraint_count, index_count;
    END IF;
END $$;

-- Display final constraint state
SELECT 
    conname as constraint_name,
    pg_get_constraintdef(oid) as constraint_definition
FROM pg_constraint
WHERE conrelid = 'users'::regclass
  AND conname LIKE '%email%'
ORDER BY conname;

-- Display final index state
SELECT 
    indexname,
    indexdef
FROM pg_indexes
WHERE tablename = 'users'
  AND indexname LIKE '%email%'
ORDER BY indexname;
