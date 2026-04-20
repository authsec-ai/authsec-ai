-- Migration 037: Ensure tenant_mappings uses UUID column types
-- This migration upgrades the tenant_mappings table so that tenant_id and client_id
-- are stored as proper UUID values instead of VARCHAR, guaranteeing type safety
-- across the application codebase.

-- BEGIN; (removed - app manages transactions)

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'tenant_mappings'
          AND column_name = 'tenant_id'
          AND data_type <> 'uuid'
    ) THEN
        ALTER TABLE tenant_mappings
            ALTER COLUMN tenant_id TYPE UUID USING tenant_id::uuid;
        RAISE NOTICE 'Converted tenant_mappings.tenant_id to UUID';
    END IF;
END;
$$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'tenant_mappings'
          AND column_name = 'client_id'
          AND data_type <> 'uuid'
    ) THEN
        ALTER TABLE tenant_mappings
            ALTER COLUMN client_id TYPE UUID USING client_id::uuid;
        RAISE NOTICE 'Converted tenant_mappings.client_id to UUID';
    END IF;
END;
$$;

-- Reinforce NOT NULL constraints after type conversion
ALTER TABLE tenant_mappings
    ALTER COLUMN tenant_id SET NOT NULL,
    ALTER COLUMN client_id SET NOT NULL;

-- Ensure supporting indexes exist (they are no-ops if already present)
CREATE INDEX IF NOT EXISTS idx_tenant_mappings_tenant_id ON tenant_mappings(tenant_id);

-- COMMIT; (removed - app manages transactions)

