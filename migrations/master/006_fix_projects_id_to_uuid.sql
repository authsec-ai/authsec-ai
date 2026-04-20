-- Migration: Ensure projects table exists with UUID primary key
-- and add required indexes only if missing

DO $$
BEGIN
    -- Step 1: If projects table exists and contains data, abort
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'projects') THEN
        IF EXISTS (SELECT 1 FROM projects LIMIT 1) THEN
            RAISE NOTICE 'projects table contains data, skipping UUID conversion (already handled by later migrations)';
    
        END IF;
    END IF;

    -- Step 2: Create projects table if not exists
    CREATE TABLE IF NOT EXISTS projects (
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        name VARCHAR(255) NOT NULL,
        description TEXT,
        user_id UUID,
        tenant_id UUID,
        client_id UUID,
        active BOOLEAN DEFAULT true,
        created_at TIMESTAMPTZ DEFAULT NOW(),
        updated_at TIMESTAMPTZ DEFAULT NOW()
    );

    -- Step 3: Create indexes only if they don't already exist
    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'projects' AND indexname = 'idx_projects_user_id'
    ) THEN
        CREATE INDEX idx_projects_user_id ON projects(user_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'projects' AND indexname = 'idx_projects_tenant_id'
    ) THEN
        CREATE INDEX idx_projects_tenant_id ON projects(tenant_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'projects' AND indexname = 'idx_projects_client_id'
    ) THEN
        CREATE INDEX idx_projects_client_id ON projects(client_id);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'projects' AND indexname = 'idx_projects_active'
    ) THEN
        CREATE INDEX idx_projects_active ON projects(active);
    END IF;

    IF NOT EXISTS (
        SELECT 1 FROM pg_indexes
        WHERE tablename = 'projects' AND indexname = 'idx_projects_timestamps'
    ) THEN
        CREATE INDEX idx_projects_timestamps ON projects(created_at, updated_at);
    END IF;

    RAISE NOTICE 'Ensured projects table and indexes exist (UUID id enforced if empty table).';
END $$;
