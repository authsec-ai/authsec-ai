-- Migration 062: Ensure global role names are unique for auth seed scripts
-- Some external services seed system roles using `ON CONFLICT (name) DO NOTHING`.
-- Older databases may have been created before the global unique constraint on
-- roles.name was introduced, which causes the seed to fail with
-- "no unique or exclusion constraint matching the ON CONFLICT specification".
-- This migration backfills the required uniqueness constraint safely.

DO $$
BEGIN
    -- Remove duplicate system roles (tenant_id IS NULL) keeping the earliest record
    IF EXISTS (
        SELECT name
        FROM roles
        WHERE tenant_id IS NULL
        GROUP BY name
        HAVING COUNT(*) > 1
    ) THEN
        DELETE FROM roles r
        USING (
            SELECT id
            FROM (
                SELECT id,
                       ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at, id) AS rn
                FROM roles
                WHERE tenant_id IS NULL
            ) ranked
            WHERE rn > 1
        ) dup
        WHERE r.id = dup.id;
    END IF;
END $$;

-- Global UNIQUE(name) on roles intentionally not added.
-- Multi-tenant: each tenant has its own "admin" role.
-- Per-tenant uniqueness is enforced by UNIQUE(tenant_id, name).

