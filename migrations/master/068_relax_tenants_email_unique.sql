-- Migration 112: Relax tenants.email uniqueness
-- Drops unique index on tenants.email if present and recreates a non-unique index.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexname = 'idx_tenants_email'
          AND indexdef ILIKE '%UNIQUE%'
    ) THEN
        EXECUTE 'DROP INDEX IF EXISTS idx_tenants_email';
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email)';
    ELSIF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexname = 'idx_tenants_email'
    ) THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_tenants_email ON tenants(email)';
    END IF;
END $$;
