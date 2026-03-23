-- Migration 114: Ensure roles.created_at and roles.updated_at default to NOW()
-- Safe to run multiple times; checks column existence before altering.

DO $$
BEGIN
    -- Add created_at if missing
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'roles' AND column_name = 'created_at'
    ) THEN
        ALTER TABLE roles ADD COLUMN created_at TIMESTAMPTZ DEFAULT NOW();
    ELSE
        ALTER TABLE roles ALTER COLUMN created_at SET DEFAULT NOW();
    END IF;

    -- Add updated_at if missing
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'roles' AND column_name = 'updated_at'
    ) THEN
        ALTER TABLE roles ADD COLUMN updated_at TIMESTAMPTZ DEFAULT NOW();
    ELSE
        ALTER TABLE roles ALTER COLUMN updated_at SET DEFAULT NOW();
    END IF;
END$$;
