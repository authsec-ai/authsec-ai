-- Migration: Drop foreign key constraint from user_roles to users
-- This allows assigning roles to users before they are fully created in the users table

DO $$
BEGIN
    -- Drop the user_id foreign key constraint from user_roles table
    IF EXISTS (
        SELECT 1 FROM information_schema.table_constraints
        WHERE constraint_name = 'user_roles_user_id_fkey'
        AND table_name = 'user_roles'
    ) THEN
        ALTER TABLE user_roles
        DROP CONSTRAINT user_roles_user_id_fkey;
        RAISE NOTICE 'Dropped foreign key user_roles_user_id_fkey';
    END IF;

    -- Keep the role_id foreign key as roles should exist
    -- The role_id constraint ensures data integrity for roles
END$$;