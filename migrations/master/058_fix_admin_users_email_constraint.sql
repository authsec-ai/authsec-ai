-- Migration: Legacy admin user constraint fix (deprecated)
-- The platform now uses the shared users table; no action required.
DO $$
BEGIN
    RAISE NOTICE 'Skipping migration 100: legacy admin-only user table deprecated in favor of users.';
END $$;
