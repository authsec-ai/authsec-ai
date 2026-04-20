-- Migration 081: Make name fields optional in pending_registrations table
-- BEGIN; (removed - app manages transactions)

-- Allow NULL values in pending_registrations first_name and last_name
ALTER TABLE pending_registrations
    ALTER COLUMN first_name DROP NOT NULL,
    ALTER COLUMN last_name DROP NOT NULL;

-- Set default values for backwards compatibility
ALTER TABLE pending_registrations
    ALTER COLUMN first_name SET DEFAULT '',
    ALTER COLUMN last_name SET DEFAULT '';

-- COMMIT; (removed - app manages transactions)
