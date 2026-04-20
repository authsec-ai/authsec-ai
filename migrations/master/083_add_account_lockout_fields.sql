-- Migration: Add account lockout and brute force protection fields
-- This migration adds fields to track failed login attempts and lock accounts
-- after 3 consecutive failures to prevent brute force attacks

-- Main database (users table for admin users)
ALTER TABLE users 
ADD COLUMN IF NOT EXISTS failed_login_attempts INTEGER DEFAULT 0,
ADD COLUMN IF NOT EXISTS account_locked_at TIMESTAMP WITH TIME ZONE,
ADD COLUMN IF NOT EXISTS password_reset_required BOOLEAN DEFAULT FALSE;

-- Add index for locked accounts to optimize queries
CREATE INDEX IF NOT EXISTS idx_users_account_locked ON users(account_locked_at) WHERE account_locked_at IS NOT NULL;

-- Add comment for documentation
COMMENT ON COLUMN users.failed_login_attempts IS 'Number of consecutive failed login attempts';
COMMENT ON COLUMN users.account_locked_at IS 'Timestamp when account was locked due to too many failed attempts';
COMMENT ON COLUMN users.password_reset_required IS 'Flag indicating user must reset password before next login';
