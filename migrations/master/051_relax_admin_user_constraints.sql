-- Migration 078: Relax admin user constraints to allow optional names and default provider data
-- BEGIN; (removed - app manages transactions)

-- Allow NULL values in users.name to support invites without names
ALTER TABLE users
    ALTER COLUMN name DROP NOT NULL;

-- Ensure provider_data is always valid JSON and nullable
ALTER TABLE users
    ALTER COLUMN provider_data DROP NOT NULL,
    ALTER COLUMN provider_data SET DEFAULT '{}'::jsonb;

-- Backfill any existing rows with invalid or empty provider_data
UPDATE users
SET provider_data = '{}'::jsonb
WHERE provider_data IS NULL;

-- COMMIT; (removed - app manages transactions)
