-- Migration: Convert projects.id from SERIAL to UUID with data preservation
-- This migration safely converts the projects table primary key from integer to UUID

DO $$
DECLARE
    project_record RECORD;
    new_uuid UUID;
BEGIN
    -- Check if projects table exists
    IF NOT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'projects' AND table_schema = 'public') THEN
        -- Create table with UUID if it doesn't exist
        CREATE TABLE projects (
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
        RAISE NOTICE 'Created projects table with UUID primary key';
        RETURN;
    END IF;

    -- Check current column type
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'projects'
          AND column_name = 'id'
          AND data_type = 'uuid'
    ) THEN
        RAISE NOTICE 'projects.id is already UUID, skipping conversion';
        RETURN;
    END IF;

    -- Create temporary table to hold data
    CREATE TEMP TABLE projects_backup AS SELECT * FROM projects;

    -- Drop and recreate with UUID primary key
    DROP TABLE projects CASCADE;

    CREATE TABLE projects (
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

    -- Restore data with new UUIDs
    FOR project_record IN SELECT * FROM projects_backup LOOP
        -- Generate a UUID based on the old integer ID for consistency
        new_uuid := ('00000000-0000-0000-0000-' || LPAD(project_record.id::text, 12, '0'))::uuid;

        INSERT INTO projects (
            id, name, description, user_id, tenant_id, client_id,
            active, created_at, updated_at
        ) VALUES (
            new_uuid,
            project_record.name,
            project_record.description,
            project_record.user_id,
            project_record.tenant_id,
            project_record.client_id,
            project_record.active,
            project_record.created_at,
            project_record.updated_at
        );
    END LOOP;

    -- Recreate indexes
    CREATE INDEX IF NOT EXISTS idx_projects_user_id ON projects(user_id);
    CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
    CREATE INDEX IF NOT EXISTS idx_projects_client_id ON projects(client_id);
    CREATE INDEX IF NOT EXISTS idx_projects_active ON projects(active);
    CREATE INDEX IF NOT EXISTS idx_projects_timestamps ON projects(created_at, updated_at);

    -- Clean up
    -- DROP TABLE projects_backup; -- Commented out to preserve backup

    RAISE NOTICE 'Successfully converted projects.id from SERIAL to UUID with data preservation';
END $$;