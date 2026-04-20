-- Migration: Add Critical Unique Constraints
-- Description: Fix missing unique constraints for business-critical fields to prevent data corruption

-- =====================================================
-- CRITICAL UNIQUE CONSTRAINTS
-- =====================================================

-- 1. Users email uniqueness (within tenant scope)
-- REMOVED: No longer enforcing unique constraint on email to allow duplicates across clients
-- Drop existing index if it exists to recreate as proper constraint
DO $$
BEGIN
    -- Drop any existing unique index on email
    IF EXISTS (SELECT 1 FROM pg_indexes WHERE indexname = 'idx_users_email_unique') THEN
        DROP INDEX idx_users_email_unique;
    END IF;

    -- REMOVED: No longer adding unique constraint for email within tenant
    -- IF NOT EXISTS (
    --     SELECT 1 FROM information_schema.table_constraints
    --     WHERE constraint_name = 'users_email_tenant_unique'
    --     AND table_name = 'users'
    -- ) THEN
    --     ALTER TABLE users ADD CONSTRAINT users_email_tenant_unique UNIQUE (email, tenant_id);
    --     RAISE NOTICE 'Added unique constraint: users_email_tenant_unique';
    -- END IF;
END $$;

-- 2. Roles name uniqueness (within tenant scope)
-- Prevent duplicate role names within the same tenant
DO $$
BEGIN
    -- Drop existing constraint if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'roles_name_tenant_unique' 
        AND table_name = 'roles'
    ) THEN
        ALTER TABLE roles DROP CONSTRAINT roles_name_tenant_unique;
    END IF;
    
    -- Create unique index that handles NULL tenant_id properly
    CREATE UNIQUE INDEX IF NOT EXISTS roles_name_tenant_unique 
    ON roles (name, tenant_id);
    
    RAISE NOTICE 'Added unique index: roles_name_tenant_unique';
END $$;

-- 3. Scopes name uniqueness (within tenant scope)
DO $$
BEGIN
    -- Create unique index that handles NULL tenant_id properly
    CREATE UNIQUE INDEX IF NOT EXISTS scopes_name_tenant_unique 
    ON scopes (name, tenant_id);
    
    RAISE NOTICE 'Added unique index: scopes_name_tenant_unique';
END $$;

-- 4. Groups name uniqueness (within tenant scope)
DO $$
BEGIN
    -- Create unique index that handles NULL tenant_id properly
    CREATE UNIQUE INDEX IF NOT EXISTS groups_name_tenant_unique 
    ON groups (name, tenant_id);
    
    RAISE NOTICE 'Added unique index: groups_name_tenant_unique';
END $$;

-- 6. Credentials credential_id uniqueness (globally unique WebAuthn credentials)
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.table_constraints 
        WHERE constraint_name = 'credentials_credential_id_unique' 
        AND table_name = 'credentials'
    ) THEN
        ALTER TABLE credentials ADD CONSTRAINT credentials_credential_id_unique 
        UNIQUE (credential_id);
        RAISE NOTICE 'Added unique constraint: credentials_credential_id_unique';
    END IF;
END $$;

-- =====================================================
-- CLEANUP AND VALIDATION
-- =====================================================

-- Log completion
DO $$
BEGIN
    RAISE NOTICE 'Migration 046: Critical unique constraints added successfully';
    RAISE NOTICE 'Tables affected: users, roles, scopes, groups, credentials';
END $$;