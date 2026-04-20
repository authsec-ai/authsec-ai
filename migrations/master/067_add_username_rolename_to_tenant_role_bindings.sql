-- Migration 111: Add username and role_name columns to role_bindings table
-- This migration ensures tenant databases have the denormalized columns that were added in migration 110
-- These columns are optional and used for performance optimization (avoiding joins)

-- Add username column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'role_bindings' 
        AND column_name = 'username'
    ) THEN
        ALTER TABLE role_bindings ADD COLUMN username TEXT;
        RAISE NOTICE 'Added username column to role_bindings';
    ELSE
        RAISE NOTICE 'username column already exists in role_bindings';
    END IF;
END$$;

-- Add role_name column if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'role_bindings' 
        AND column_name = 'role_name'
    ) THEN
        ALTER TABLE role_bindings ADD COLUMN role_name TEXT;
        RAISE NOTICE 'Added role_name column to role_bindings';
    ELSE
        RAISE NOTICE 'role_name column already exists in role_bindings';
    END IF;
END$$;

-- Backfill username from users table (if the column was just added)
UPDATE role_bindings rb
SET username = u.username
FROM users u
WHERE rb.user_id = u.id
  AND rb.username IS NULL
  AND u.username IS NOT NULL;

-- Backfill role_name from roles table (if the column was just added)
UPDATE role_bindings rb
SET role_name = r.name
FROM roles r
WHERE rb.role_id = r.id
  AND rb.role_name IS NULL
  AND r.name IS NOT NULL;
