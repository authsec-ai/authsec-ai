-- Migration: Ensure fk_users_project_id constraint is removed
-- Purpose: Admin self-service registration creates users before projects exist,
--          so we must keep the users.project_id column decoupled from projects.

DO $$
BEGIN
    RAISE NOTICE '=== MIGRATION 074: Dropping fk_users_project_id if present ===';

    IF EXISTS (
        SELECT 1
        FROM information_schema.table_constraints
        WHERE constraint_type = 'FOREIGN KEY'
          AND table_name = 'users'
          AND constraint_name = 'fk_users_project_id'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT fk_users_project_id;
        RAISE NOTICE 'Dropped fk_users_project_id constraint from users table';
    ELSE
        RAISE NOTICE 'Constraint fk_users_project_id was not found, skipping';
    END IF;

    RAISE NOTICE '=== MIGRATION 074 COMPLETED ===';
END $$;
