-- Migration: Convert projects.id from bigint to UUID for shared-models compatibility
-- This ensures proper data type alignment with shared-models

-- Step 1: Drop foreign key constraints that reference projects.id
ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_project_id;

-- Step 2: Add a new UUID column temporarily
ALTER TABLE projects ADD COLUMN id_uuid UUID DEFAULT gen_random_uuid();

-- Step 3: Update the new column with UUID values for existing records
UPDATE projects SET id_uuid = gen_random_uuid() WHERE id_uuid IS NULL;

-- Step 4: Update referencing tables to match the new UUID values
-- First, set all foreign keys to NULL (they'll need to be re-established later)
UPDATE users SET project_id = NULL WHERE project_id IS NOT NULL;

-- Step 5: Drop the old primary key constraint and sequence
ALTER TABLE projects DROP CONSTRAINT projects_pkey;
DROP SEQUENCE IF EXISTS projects_id_seq CASCADE;

-- Step 4: Drop the old id column
ALTER TABLE projects DROP COLUMN id;

-- Step 5: Rename the new column to 'id'
ALTER TABLE projects RENAME COLUMN id_uuid TO id;

-- Step 6: Make the new id column NOT NULL and set as primary key
ALTER TABLE projects ALTER COLUMN id SET NOT NULL;
ALTER TABLE projects ADD PRIMARY KEY (id);

-- Step 7: Foreign key columns are already UUID type, no conversion needed

-- Step 8: Recreate foreign key constraints
ALTER TABLE users ADD CONSTRAINT fk_users_project_id
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE SET NULL;

-- Step 9: Recreate indexes
DROP INDEX IF EXISTS idx_projects_deleted_at;
CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);

-- Step 9: Add comment for documentation
COMMENT ON TABLE projects IS 'Projects table with UUID primary key for shared-models compatibility';
COMMENT ON COLUMN projects.id IS 'UUID primary key for shared-models compatibility';

