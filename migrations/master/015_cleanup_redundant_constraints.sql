-- Migration: 016_cleanup_redundant_constraints.sql
-- Purpose: Remove redundant foreign key constraints that GORM auto-migration handles
-- Description: Based on shared-models analysis, these FK constraints are redundant

-- Remove redundant FK constraints that GORM handles automatically
DO $$
BEGIN
    RAISE NOTICE '=== CLEANUP REDUNDANT FOREIGN KEY CONSTRAINTS ===';

    -- Remove fk_users_project_id if it exists
    -- Reason: User model has no FK relationship tags for project_id in shared-models
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'FOREIGN KEY'
        AND table_name = 'users'
        AND constraint_name = 'fk_users_project_id'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_project_id;
        RAISE NOTICE 'Dropped redundant constraint fk_users_project_id from users table';
    ELSE
        RAISE NOTICE 'Constraint fk_users_project_id not found, skipping';
    END IF;

    -- Remove fk_projects_user_id if it exists
    -- Reason: Project model has no FK relationship tags for user_id in shared-models
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'FOREIGN KEY'
        AND table_name = 'projects'
        AND constraint_name = 'fk_projects_user_id'
    ) THEN
        ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_user_id;
        RAISE NOTICE 'Dropped redundant constraint fk_projects_user_id from projects table';
    ELSE
        RAISE NOTICE 'Constraint fk_projects_user_id not found, skipping';
    END IF;

    -- Remove fk_projects_tenant_id if it exists
    -- Reason: Project model has FK relationship tags for tenant_id in shared-models
    -- GORM will create this automatically
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'FOREIGN KEY'
        AND table_name = 'projects'
        AND constraint_name = 'fk_projects_tenant_id'
    ) THEN
        ALTER TABLE projects DROP CONSTRAINT IF EXISTS fk_projects_tenant_id;
        RAISE NOTICE 'Dropped redundant constraint fk_projects_tenant_id from projects table';
    ELSE
        RAISE NOTICE 'Constraint fk_projects_tenant_id not found, skipping';
    END IF;

    -- Remove fk_users_project_id if it exists
    -- Reason: User model may not have FK relationship tags for project_id in shared-models
    -- GORM will create this automatically if needed
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'FOREIGN KEY'
        AND table_name = 'users'
        AND constraint_name = 'fk_users_project_id'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_project_id;
        RAISE NOTICE 'Dropped redundant constraint fk_users_project_id from users table';
    ELSE
        RAISE NOTICE 'Constraint fk_users_project_id not found, skipping';
    END IF;

    -- Remove uni_projects_tenant_id unique constraint if it exists
    -- Reason: This constraint prevents multiple projects per tenant (incorrect business logic)
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'UNIQUE'
        AND table_name = 'projects'
        AND constraint_name = 'uni_projects_tenant_id'
    ) THEN
        ALTER TABLE projects DROP CONSTRAINT IF EXISTS uni_projects_tenant_id;
        RAISE NOTICE 'Dropped breaking unique constraint uni_projects_tenant_id from projects table';
    ELSE
        RAISE NOTICE 'Unique constraint uni_projects_tenant_id not found, skipping';
    END IF;

    RAISE NOTICE '=== CLEANUP COMPLETED ===';
END $$;