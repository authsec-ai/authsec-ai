-- Migration: 077_add_icp_fields_to_tenants.sql
-- Purpose: Add ICP-related fields to tenants table for PKI provisioning

DO $$
BEGIN
    RAISE NOTICE '=== ADDING ICP FIELDS TO TENANTS TABLE ===';

    -- Ensure tenants table exists
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'tenants'
    ) THEN
        RAISE NOTICE 'Tenants table does not exist, skipping ICP fields addition';
        RETURN;
    END IF;

    -- Add vault_mount column (stores the Vault PKI mount path for this tenant)
    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'vault_mount';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN vault_mount VARCHAR(255);
        RAISE NOTICE 'Added vault_mount column';
    ELSE
        RAISE NOTICE 'vault_mount column already exists';
    END IF;

    -- Add ca_cert column (stores the Root CA certificate PEM)
    PERFORM 1 FROM information_schema.columns WHERE table_name = 'tenants' AND column_name = 'ca_cert';
    IF NOT FOUND THEN
        ALTER TABLE tenants ADD COLUMN ca_cert TEXT;
        RAISE NOTICE 'Added ca_cert column';
    ELSE
        RAISE NOTICE 'ca_cert column already exists';
    END IF;

    -- Note: database_url is NOT stored - it's generated on-demand from tenant_db + config
    -- This avoids security risks and stale credential issues

    -- Add index on vault_mount for faster lookups
    IF NOT EXISTS (SELECT 1 FROM pg_indexes WHERE tablename='tenants' AND indexname='idx_tenants_vault_mount') THEN
        CREATE INDEX idx_tenants_vault_mount ON tenants(vault_mount);
        RAISE NOTICE 'Created index idx_tenants_vault_mount';
    ELSE
        RAISE NOTICE 'Index idx_tenants_vault_mount already exists';
    END IF;

    RAISE NOTICE '=== ICP FIELDS ADDITION COMPLETED ===';
END $$;
