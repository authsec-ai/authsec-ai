-- Migration: 022_add_tenant_id_to_groups.sql
-- Description: Add tenant_id column to groups table to support multi-tenancy
-- Date: 2025-10-03

DO $$
BEGIN
    -- Add tenant_id column to groups table if it doesn't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_name = 'groups' AND column_name = 'tenant_id') THEN
        ALTER TABLE groups ADD COLUMN tenant_id UUID;
        RAISE NOTICE 'Added tenant_id column to groups table';
    ELSE
        RAISE NOTICE 'tenant_id column already exists in groups table';
    END IF;
END $$;

-- Create index on tenant_id for performance
CREATE INDEX IF NOT EXISTS idx_groups_tenant_id ON groups(tenant_id);

-- Update existing groups to have a default tenant_id if needed
-- Note: This assumes groups without tenant_id should be assigned to a default tenant
-- In a real scenario, you might need to migrate existing data appropriately

-- Drop the old unique constraint on name only, if it exists
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.table_constraints
               WHERE table_name = 'groups' AND constraint_name = 'uni_groups_name') THEN
        ALTER TABLE groups DROP CONSTRAINT uni_groups_name;
        RAISE NOTICE 'Dropped old unique constraint uni_groups_name';
    END IF;
END $$;

-- Create new unique constraint on (tenant_id, name) combination
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.table_constraints
                   WHERE table_name = 'groups' AND constraint_name = 'uni_groups_tenant_name') THEN
        ALTER TABLE groups ADD CONSTRAINT uni_groups_tenant_name UNIQUE (tenant_id, name);
        RAISE NOTICE 'Created unique constraint uni_groups_tenant_name';
    END IF;
END $$;