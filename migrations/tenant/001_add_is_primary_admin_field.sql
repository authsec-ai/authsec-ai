-- Tenant Migration: Add is_primary_admin field to tenant users table
-- This migration should be applied to ALL tenant databases
-- Note: This migration is designed to run via psql which handles multi-statement transactions.

-- Add is_primary_admin column to users table in tenant database
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS is_primary_admin BOOLEAN DEFAULT false;

-- Create index for faster primary admin lookups
CREATE INDEX IF NOT EXISTS idx_users_is_primary_admin 
ON users(is_primary_admin) 
WHERE is_primary_admin = true;

-- For tenant databases, set the first created user as the primary user
-- This is typically the tenant creator/owner
UPDATE users u
SET is_primary_admin = true
WHERE u.id IN (
    SELECT id
    FROM users
    WHERE active = true
    ORDER BY created_at ASC
    LIMIT 1
)
AND u.is_primary_admin = false;

-- Add comment to document the column purpose
COMMENT ON COLUMN users.is_primary_admin IS 'Indicates if this user is the primary user who cannot be deleted. Each tenant should have at least one primary user.';
