-- Migration: Align SQL Migration Constraints with GORM Expectations
-- Description: This migration resolves constraint naming conflicts between SQL migrations and GORM auto-migration
-- Phase 1: Fix critical foreign key issues and rename constraints to GORM conventions

-- ============================================================================
-- PHASE 1: Fix Critical Foreign Key Issues
-- ============================================================================

-- REMOVED: Breaking unique constraint that prevents multiple projects per tenant
-- Fix projects.tenant_id constraint issue
-- Problem: GORM expects unique constraint on projects.tenant_id for foreign key reference
-- Current: Only has index idx_projects_tenant_id
-- Solution: Add unique constraint (will also handle the relationship issue)

-- DO $$
-- BEGIN
--     -- First, check if projects.tenant_id has data that would violate uniqueness
--     DECLARE
--         duplicate_count INTEGER;
--     BEGIN
--         SELECT COUNT(*) INTO duplicate_count
--         FROM (
--             SELECT tenant_id, COUNT(*)
--             FROM projects
--             WHERE tenant_id IS NOT NULL
--             GROUP BY tenant_id
--             HAVING COUNT(*) > 1
--         ) duplicates;

--         IF duplicate_count > 0 THEN
--             RAISE WARNING 'Found % duplicate tenant_id values in projects table. Foreign key constraint may fail.', duplicate_count;
--             -- Log the duplicate values for investigation
--             RAISE NOTICE 'Duplicate tenant_id values: %', (
--                 SELECT string_agg(tenant_id::text, ', ')
--                 FROM (
--                     SELECT tenant_id
--                     FROM projects
--                     WHERE tenant_id IS NOT NULL
--                     GROUP BY tenant_id
--                     HAVING COUNT(*) > 1
--                 ) dups
--             );
--         END IF;
--     END;

--     -- Add unique constraint to projects.tenant_id to support foreign key reference
--     -- Note: This changes the business logic - each tenant can only have one project
--     -- If this is not desired, the Tenant model's foreign key relationship needs to be redesigned
--     BEGIN
--         ALTER TABLE projects ADD CONSTRAINT uni_projects_tenant_id UNIQUE (tenant_id);
--         RAISE NOTICE 'Successfully added unique constraint uni_projects_tenant_id to projects.tenant_id';
--     EXCEPTION
--         WHEN duplicate_object THEN
--             RAISE NOTICE 'Unique constraint uni_projects_tenant_id already exists, skipping';
--         WHEN unique_violation THEN
--             RAISE WARNING 'Cannot add unique constraint: duplicate tenant_id values exist in projects table';
--             RAISE WARNING 'Manual data cleanup required before this constraint can be applied';
--     END;
-- END $$;

-- ============================================================================
-- PHASE 2: Rename Existing Constraints to GORM Naming Convention
-- ============================================================================

-- Rename constraint: tenants_tenant_id_key → uni_tenants_tenant_id
DO $$
BEGIN
    BEGIN
        ALTER TABLE tenants RENAME CONSTRAINT tenants_tenant_id_key TO uni_tenants_tenant_id;
        RAISE NOTICE 'Successfully renamed tenants_tenant_id_key to uni_tenants_tenant_id';
    EXCEPTION
        WHEN undefined_object THEN
            RAISE NOTICE 'Constraint tenants_tenant_id_key does not exist, skipping rename';
        WHEN duplicate_object THEN
            RAISE NOTICE 'Constraint uni_tenants_tenant_id already exists, skipping rename';
    END;
END $$;

-- Rename constraint: users_email_key → uni_users_email
DO $$
BEGIN
    BEGIN
        ALTER TABLE users RENAME CONSTRAINT users_email_key TO uni_users_email;
        RAISE NOTICE 'Successfully renamed users_email_key to uni_users_email';
    EXCEPTION
        WHEN undefined_object THEN
            RAISE NOTICE 'Constraint users_email_key does not exist, skipping rename';
        WHEN duplicate_object THEN
            RAISE NOTICE 'Constraint uni_users_email already exists, skipping rename';
    END;
END $$;

-- Rename constraint: groups_name_key → uni_groups_name
DO $$
BEGIN
    BEGIN
        ALTER TABLE groups RENAME CONSTRAINT groups_name_key TO uni_groups_name;
        RAISE NOTICE 'Successfully renamed groups_name_key to uni_groups_name';
    EXCEPTION
        WHEN undefined_object THEN
            RAISE NOTICE 'Constraint groups_name_key does not exist, skipping rename';
        WHEN duplicate_object THEN
            RAISE NOTICE 'Constraint uni_groups_name already exists, skipping rename';
    END;
END $$;

-- ============================================================================
-- PHASE 3: Add Missing Constraint Names for Existing Multi-Column Constraints
-- ============================================================================

-- Handle existing multi-column unique constraints that need GORM-compatible names
-- Note: PostgreSQL automatically names these, but GORM expects specific patterns

-- Resources table: resources_tenant_id_name_key → uni_resources_name
DO $$
BEGIN
    -- GORM creates unique constraint only on 'name' field, not tenant_id + name
    -- Need to check if current constraint matches GORM expectation
    DECLARE
        current_constraint_columns TEXT;
    BEGIN
        SELECT string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) INTO current_constraint_columns
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'resources'
        AND tc.constraint_type = 'UNIQUE'
        AND tc.constraint_name = 'resources_tenant_id_name_key';

        IF current_constraint_columns = 'tenant_id,name' THEN
            RAISE NOTICE 'Resources constraint includes tenant_id + name, but GORM expects only name uniqueness';
            RAISE NOTICE 'Consider reviewing business logic: should resource names be unique globally or per-tenant?';
        END IF;
    END;
END $$;

-- Similar check for roles table: roles_tenant_id_name_key
DO $$
BEGIN
    DECLARE
        current_constraint_columns TEXT;
    BEGIN
        SELECT string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) INTO current_constraint_columns
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'roles'
        AND tc.constraint_type = 'UNIQUE'
        AND tc.constraint_name = 'roles_tenant_id_name_key';

        IF current_constraint_columns = 'tenant_id,name' THEN
            RAISE NOTICE 'Roles constraint includes tenant_id + name, but GORM expects only name uniqueness';
            RAISE NOTICE 'Consider reviewing business logic: should role names be unique globally or per-tenant?';
        END IF;
    END;
END $$;

-- Similar check for scopes table: scopes_tenant_id_name_key
DO $$
BEGIN
    DECLARE
        current_constraint_columns TEXT;
    BEGIN
        SELECT string_agg(kcu.column_name, ',' ORDER BY kcu.ordinal_position) INTO current_constraint_columns
        FROM information_schema.table_constraints tc
        JOIN information_schema.key_column_usage kcu ON tc.constraint_name = kcu.constraint_name
        WHERE tc.table_name = 'scopes'
        AND tc.constraint_type = 'UNIQUE'
        AND tc.constraint_name = 'scopes_tenant_id_name_key';

        IF current_constraint_columns = 'name,tenant_id' THEN
            RAISE NOTICE 'Scopes constraint includes tenant_id + name, but GORM expects only name uniqueness';
            RAISE NOTICE 'Consider reviewing business logic: should scope names be unique globally or per-tenant?';
        END IF;
    END;
END $$;

-- ============================================================================
-- VERIFICATION: Check Constraint Alignment Status
-- ============================================================================

DO $$
DECLARE
    constraint_report TEXT := '';
    missing_constraints TEXT := '';
    rec RECORD;
BEGIN
    RAISE NOTICE '=== CONSTRAINT ALIGNMENT VERIFICATION ===';

    -- Check critical constraints that GORM expects
    FOR rec IN
        SELECT
            'uni_tenants_tenant_id' as expected_name,
            'tenants' as table_name,
            'tenant_id' as column_name
        UNION ALL SELECT 'uni_users_email', 'users', 'email'
        UNION ALL SELECT 'uni_groups_name', 'groups', 'name'
        UNION ALL SELECT 'uni_projects_tenant_id', 'projects', 'tenant_id'
    LOOP
        PERFORM 1 FROM information_schema.table_constraints
        WHERE table_name = rec.table_name
        AND constraint_name = rec.expected_name
        AND constraint_type = 'UNIQUE';

        IF FOUND THEN
            constraint_report := constraint_report || format('✓ %s.%s has constraint %s\n',
                rec.table_name, rec.column_name, rec.expected_name);
        ELSE
            missing_constraints := missing_constraints || format('✗ Missing: %s.%s constraint %s\n',
                rec.table_name, rec.column_name, rec.expected_name);
        END IF;
    END LOOP;

    RAISE NOTICE 'Constraint Status Report:';
    RAISE NOTICE '%', constraint_report;

    IF missing_constraints != '' THEN
        RAISE WARNING 'Missing Constraints:';
        RAISE WARNING '%', missing_constraints;
    END IF;

    RAISE NOTICE '=== END VERIFICATION ===';
END $$;