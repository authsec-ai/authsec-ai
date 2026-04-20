-- Migration 073: Add temporary password tracking to users table
-- This supports admin invite functionality where users get temporary passwords

-- Add temporary_password flag to users table
ALTER TABLE users ADD COLUMN IF NOT EXISTS temporary_password BOOLEAN DEFAULT false;

-- Add temporary_password_expires_at timestamp
ALTER TABLE users ADD COLUMN IF NOT EXISTS temporary_password_expires_at TIMESTAMP WITH TIME ZONE;

-- Add password_change_required flag
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_change_required BOOLEAN DEFAULT false;

-- Add invited_by to track who invited the user
ALTER TABLE users ADD COLUMN IF NOT EXISTS invited_by UUID;

-- Add invited_at timestamp
ALTER TABLE users ADD COLUMN IF NOT EXISTS invited_at TIMESTAMP WITH TIME ZONE;

-- Create index for finding users with temporary passwords
CREATE INDEX IF NOT EXISTS idx_users_temporary_password ON users(temporary_password) WHERE temporary_password = true;

-- Create index for finding users requiring password change
CREATE INDEX IF NOT EXISTS idx_users_password_change_required ON users(password_change_required) WHERE password_change_required = true;

-- Create index for finding expired temporary passwords
CREATE INDEX IF NOT EXISTS idx_users_temp_password_expires_at ON users(temporary_password_expires_at) WHERE temporary_password_expires_at IS NOT NULL;

COMMENT ON COLUMN users.temporary_password IS 'Indicates if user is using a temporary password from admin invite';
COMMENT ON COLUMN users.temporary_password_expires_at IS 'Timestamp when temporary password expires';
COMMENT ON COLUMN users.password_change_required IS 'Forces user to change password on next login';
COMMENT ON COLUMN users.invited_by IS 'UUID of admin who invited this user';
COMMENT ON COLUMN users.invited_at IS 'Timestamp when user was invited';
