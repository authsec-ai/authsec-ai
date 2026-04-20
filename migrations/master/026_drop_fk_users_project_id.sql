-- Migration: Drop foreign key constraint fk_users_project_id
-- Purpose: Remove FK constraint that causes issues during user creation
-- Description: The FK constraint requires project to exist before user, but in some flows
--              the user and project need to be created in the same transaction without strict ordering.
--              The project_id relationship will be maintained by application logic instead.

DO $$
BEGIN
    RAISE NOTICE '=== DROPPING FK CONSTRAINT fk_users_project_id ===';

    -- Drop fk_users_project_id if it exists
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_type = 'FOREIGN KEY'
        AND table_name = 'users'
        AND constraint_name = 'fk_users_project_id'
    ) THEN
        ALTER TABLE users DROP CONSTRAINT fk_users_project_id;
        RAISE NOTICE 'Successfully dropped constraint fk_users_project_id from users table';
    ELSE
        RAISE NOTICE 'Constraint fk_users_project_id not found, skipping';
    END IF;

    RAISE NOTICE '=== MIGRATION 035 COMPLETED ===';
END $$;
