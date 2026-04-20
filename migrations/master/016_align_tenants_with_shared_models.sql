-- Migration: 017_align_tenants_with_shared_models.sql
-- Purpose: Align tenants table with shared-models Tenant struct

DO $$
BEGIN
    RAISE NOTICE '=== ALIGNING TENANTS TABLE WITH SHARED MODELS ===';

    -- Ensure tenants table exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables 
        WHERE table_schema = 'public' AND table_name = 'tenants'
    ) THEN
        RAISE NOTICE 'Tenants table does not exist, skipping alignment';
        RETURN;
    END IF;

    -- Add missing columns
    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'username';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN username TEXT;
    END IF;

    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'provider_id';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN provider_id TEXT;
    END IF;

    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'avatar';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN avatar TEXT;
    END IF;

    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'source';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN source TEXT;
    END IF;

    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'last_login';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN last_login TIMESTAMPTZ;
    END IF;

    -- Safe type changes
    BEGIN
        ALTER TABLE tenants ALTER COLUMN email TYPE TEXT;
    EXCEPTION WHEN OTHERS THEN NULL; END;

    BEGIN
        ALTER TABLE tenants ALTER COLUMN provider TYPE TEXT;
    EXCEPTION WHEN OTHERS THEN NULL; END;

    -- Normalize nulls
    UPDATE tenants SET tenant_id = gen_random_uuid() WHERE tenant_id IS NULL;
    UPDATE tenants SET email = 'unknown-' || id || '@temp.authsec.dev' WHERE email IS NULL;
    UPDATE tenants SET tenant_domain = 'temp-' || substring(id::text,1,8) || '.authsec.dev'
      WHERE tenant_domain IS NULL;

    -- Enforce NOT NULL
    BEGIN ALTER TABLE tenants ALTER COLUMN tenant_id SET NOT NULL; EXCEPTION WHEN OTHERS THEN NULL; END;
    BEGIN ALTER TABLE tenants ALTER COLUMN email SET NOT NULL; EXCEPTION WHEN OTHERS THEN NULL; END;
    BEGIN ALTER TABLE tenants ALTER COLUMN tenant_domain SET NOT NULL; EXCEPTION WHEN OTHERS THEN NULL; END;

    -- Remove legacy columns
    PERFORM 1 FROM information_schema.columns WHERE table_name='tenants' AND column_name='domain';
    IF FOUND THEN ALTER TABLE tenants DROP COLUMN domain; END IF;

    PERFORM 1 FROM information_schema.columns WHERE table_name='tenants' AND column_name='active';
    IF FOUND THEN ALTER TABLE tenants DROP COLUMN active; END IF;

    -- Drop old indexes
    DROP INDEX IF EXISTS idx_tenants_active;
    DROP INDEX IF EXISTS idx_tenants_tenant_db;

    -- Create idx_tenants_provider
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename='tenants' AND indexname='idx_tenants_provider') THEN
        CREATE INDEX idx_tenants_provider ON tenants(provider);
    END IF;

    -- Add unique constraints only if not already in pg_constraint
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='uni_tenants_tenant_id') THEN
        BEGIN
            ALTER TABLE tenants ADD CONSTRAINT uni_tenants_tenant_id UNIQUE (tenant_id);
        EXCEPTION WHEN OTHERS THEN NULL; END;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='uni_tenants_email') THEN
        BEGIN
            ALTER TABLE tenants ADD CONSTRAINT uni_tenants_email UNIQUE (email);
        EXCEPTION WHEN OTHERS THEN NULL; END;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='uni_tenants_tenant_domain') THEN
        BEGIN
            ALTER TABLE tenants ADD CONSTRAINT uni_tenants_tenant_domain UNIQUE (tenant_domain);
        EXCEPTION WHEN OTHERS THEN NULL; END;
    END IF;

    RAISE NOTICE '=== TENANTS TABLE ALIGNMENT COMPLETED ===';
END $$;
