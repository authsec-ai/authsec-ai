-- Migration: Ensure users table has all required columns for shared models compatibility
-- Description: Adds any missing columns to the users table to match the shared models User struct

DO $$
BEGIN
    -- Add client_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'client_id') THEN
        ALTER TABLE users ADD COLUMN client_id UUID;
        RAISE NOTICE 'Added client_id column to users table';
    ELSE
        RAISE NOTICE 'client_id column already exists in users table';
    END IF;

    -- Add tenant_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'tenant_id') THEN
        ALTER TABLE users ADD COLUMN tenant_id UUID;
        RAISE NOTICE 'Added tenant_id column to users table';
    ELSE
        RAISE NOTICE 'tenant_id column already exists in users table';
    END IF;

    -- Add project_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'project_id') THEN
        ALTER TABLE users ADD COLUMN project_id UUID;
        RAISE NOTICE 'Added project_id column to users table';
    ELSE
        RAISE NOTICE 'project_id column already exists in users table';
    END IF;

    -- Add name column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'name') THEN
        ALTER TABLE users ADD COLUMN name VARCHAR(255);
        RAISE NOTICE 'Added name column to users table';
    ELSE
        RAISE NOTICE 'name column already exists in users table';
    END IF;

    -- Add username column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'username') THEN
        ALTER TABLE users ADD COLUMN username VARCHAR(255);
        RAISE NOTICE 'Added username column to users table';
    ELSE
        RAISE NOTICE 'username column already exists in users table';
    END IF;

    -- Add password_hash column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'password_hash') THEN
        ALTER TABLE users ADD COLUMN password_hash TEXT;
        RAISE NOTICE 'Added password_hash column to users table';
    ELSE
        RAISE NOTICE 'password_hash column already exists in users table';
    END IF;

    -- Add tenant_domain column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'tenant_domain') THEN
        ALTER TABLE users ADD COLUMN tenant_domain VARCHAR(255);
        RAISE NOTICE 'Added tenant_domain column to users table';
    ELSE
        RAISE NOTICE 'tenant_domain column already exists in users table';
    END IF;

    -- Add provider column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'provider') THEN
        ALTER TABLE users ADD COLUMN provider VARCHAR(50) DEFAULT 'local';
        RAISE NOTICE 'Added provider column to users table';
    ELSE
        RAISE NOTICE 'provider column already exists in users table';
    END IF;

    -- Add provider_id column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'provider_id') THEN
        ALTER TABLE users ADD COLUMN provider_id VARCHAR(255);
        RAISE NOTICE 'Added provider_id column to users table';
    ELSE
        RAISE NOTICE 'provider_id column already exists in users table';
    END IF;

    -- Add provider_data column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'provider_data') THEN
        ALTER TABLE users ADD COLUMN provider_data JSONB;
        RAISE NOTICE 'Added provider_data column to users table';
    ELSE
        RAISE NOTICE 'provider_data column already exists in users table';
    END IF;

    -- Add avatar_url column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'avatar_url') THEN
        ALTER TABLE users ADD COLUMN avatar_url TEXT;
        RAISE NOTICE 'Added avatar_url column to users table';
    ELSE
        RAISE NOTICE 'avatar_url column already exists in users table';
    END IF;

    -- Add active column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'active') THEN
        ALTER TABLE users ADD COLUMN active BOOLEAN DEFAULT true;
        RAISE NOTICE 'Added active column to users table';
    ELSE
        RAISE NOTICE 'active column already exists in users table';
    END IF;

    -- Add MFA-related columns
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'mfa_enabled') THEN
        ALTER TABLE users ADD COLUMN mfa_enabled BOOLEAN DEFAULT false;
        RAISE NOTICE 'Added mfa_enabled column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'mfa_method') THEN
        ALTER TABLE users ADD COLUMN mfa_method JSONB;
        RAISE NOTICE 'Added mfa_method column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'mfa_default_method') THEN
        ALTER TABLE users ADD COLUMN mfa_default_method VARCHAR(50);
        RAISE NOTICE 'Added mfa_default_method column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'mfa_enrolled_at') THEN
        ALTER TABLE users ADD COLUMN mfa_enrolled_at TIMESTAMP WITH TIME ZONE;
        RAISE NOTICE 'Added mfa_enrolled_at column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'mfa_verified') THEN
        ALTER TABLE users ADD COLUMN mfa_verified BOOLEAN DEFAULT false;
        RAISE NOTICE 'Added mfa_verified column to users table';
    END IF;

    -- Add sync-related columns
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'external_id') THEN
        ALTER TABLE users ADD COLUMN external_id VARCHAR(255);
        RAISE NOTICE 'Added external_id column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'sync_source') THEN
        ALTER TABLE users ADD COLUMN sync_source VARCHAR(100);
        RAISE NOTICE 'Added sync_source column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'last_sync_at') THEN
        ALTER TABLE users ADD COLUMN last_sync_at TIMESTAMP WITH TIME ZONE;
        RAISE NOTICE 'Added last_sync_at column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'is_synced_user') THEN
        ALTER TABLE users ADD COLUMN is_synced_user BOOLEAN DEFAULT false;
        RAISE NOTICE 'Added is_synced_user column to users table';
    END IF;

    -- Add last_login column if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'last_login') THEN
        ALTER TABLE users ADD COLUMN last_login TIMESTAMP WITH TIME ZONE;
        RAISE NOTICE 'Added last_login column to users table';
    END IF;

    -- Ensure created_at and updated_at exist (they should from GORM)
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'created_at') THEN
        ALTER TABLE users ADD COLUMN created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
        RAISE NOTICE 'Added created_at column to users table';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'users' AND column_name = 'updated_at') THEN
        ALTER TABLE users ADD COLUMN updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();
        RAISE NOTICE 'Added updated_at column to users table';
    END IF;

    RAISE NOTICE 'Users table schema verification completed';
END $$;