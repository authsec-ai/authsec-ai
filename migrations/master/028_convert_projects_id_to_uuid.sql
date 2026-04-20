-- Migration 040: Convert projects.id from BIGINT to UUID
-- Ensures project records use UUID identifiers consistent with shared models.
-- NOTE: Removed explicit BEGIN/COMMIT as migrations run within app-managed transactions

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Only run conversion if id column is not already UUID
DO $$
BEGIN
    -- Check if projects table exists and id column is not UUID
    IF EXISTS (
        SELECT 1 FROM information_schema.columns 
        WHERE table_name = 'projects' 
        AND column_name = 'id' 
        AND data_type != 'uuid'
    ) THEN
        -- Add UUID column
        ALTER TABLE projects ADD COLUMN IF NOT EXISTS id_uuid UUID DEFAULT gen_random_uuid();
        UPDATE projects SET id_uuid = gen_random_uuid() WHERE id_uuid IS NULL;
        ALTER TABLE projects ALTER COLUMN id_uuid SET NOT NULL;

        -- Drop PK to allow column swap
        ALTER TABLE projects DROP CONSTRAINT IF EXISTS projects_pkey;

        -- Swap columns
        ALTER TABLE projects DROP COLUMN id;
        ALTER TABLE projects RENAME COLUMN id_uuid TO id;

        -- Recreate primary key and default
        ALTER TABLE projects ADD CONSTRAINT projects_pkey PRIMARY KEY (id);
        ALTER TABLE projects ALTER COLUMN id SET DEFAULT gen_random_uuid();

        -- Remove legacy sequence
        DROP SEQUENCE IF EXISTS projects_id_seq;
        
        RAISE NOTICE 'Converted projects.id to UUID';
    ELSE
        RAISE NOTICE 'projects.id is already UUID or table does not exist, skipping';
    END IF;
END $$;

