-- Migration: Add deleted_at column to users table for GORM soft deletes
-- This ensures compatibility with GORM soft delete functionality

-- Add deleted_at column if it doesn't exist
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP WITH TIME ZONE;

-- Add index for deleted_at
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);