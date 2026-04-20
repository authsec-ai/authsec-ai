-- Migration 080: Add is_primary_admin field to users table for admin delete protection
-- This migration adds a boolean field to track the primary admin who cannot be deleted

-- BEGIN; (removed - app manages transactions)

-- Add is_primary_admin column to users table in master database
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS is_primary_admin BOOLEAN DEFAULT false;

-- Create index for faster primary admin lookups
CREATE INDEX IF NOT EXISTS idx_users_is_primary_admin 
ON users(is_primary_admin) 
WHERE is_primary_admin = true;

-- Set the first admin for each tenant as the primary admin
-- This ensures each tenant has at least one protected admin
UPDATE users u
SET is_primary_admin = true
WHERE u.id IN (
    SELECT DISTINCT ON (tenant_id) id
    FROM users
    WHERE tenant_id IS NOT NULL
    AND active = true
    ORDER BY tenant_id, created_at ASC
)
AND u.is_primary_admin = false;

-- Add comment to document the column purpose
COMMENT ON COLUMN users.is_primary_admin IS 'Indicates if this user is the primary admin who cannot be deleted. Each tenant should have at least one primary admin.';

-- COMMIT; (removed - app manages transactions)
