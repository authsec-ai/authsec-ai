-- Migration 065: Ensure global scope names are unique for auth seed scripts
-- Some external services seed system scopes using `ON CONFLICT (name) DO NOTHING`.
-- Older databases may have been created before the global unique constraint on
-- scopes.name was introduced, which causes the seed to fail with
-- "no unique or exclusion constraint matching the ON CONFLICT specification".
-- This migration backfills the required uniqueness constraint safely.

DO $$
BEGIN
    -- Remove duplicate system scopes (tenant_id IS NULL) keeping the earliest record
    IF EXISTS (
        SELECT name
        FROM scopes
        WHERE tenant_id IS NULL
        GROUP BY name
        HAVING COUNT(*) > 1
    ) THEN
        DELETE FROM scopes s
        USING (
            SELECT id
            FROM (
                SELECT id,
                       ROW_NUMBER() OVER (PARTITION BY name ORDER BY created_at, id) AS rn
                FROM scopes
                WHERE tenant_id IS NULL
            ) ranked
            WHERE rn > 1
        ) dup
        WHERE s.id = dup.id;
    END IF;
END $$;

DO $$
BEGIN
    -- Ensure a unique constraint exists on scopes.name to satisfy ON CONFLICT (name)
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint c
        JOIN pg_class t ON t.oid = c.conrelid
        JOIN pg_namespace n ON n.oid = t.relnamespace
        WHERE c.contype = 'u'
          AND n.nspname = 'public'
          AND t.relname = 'scopes'
          AND (
                SELECT array_agg(att.attname::text ORDER BY att.attnum)
                FROM unnest(c.conkey) WITH ORDINALITY AS cols(attnum, ord)
                JOIN pg_attribute att ON att.attrelid = t.oid AND att.attnum = cols.attnum
          ) = ARRAY['name']::text[]
    ) THEN
        ALTER TABLE public.scopes ADD CONSTRAINT scopes_name_key UNIQUE (name) DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;

