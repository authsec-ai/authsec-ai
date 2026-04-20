-- Migration: Schema Validation and Data Cleanup
-- Description: Validate schema integrity and clean up any inconsistent data

-- =====================================================
-- DATA VALIDATION AND CLEANUP
-- =====================================================

-- Function to check for constraint violations before they become errors
DO $$
DECLARE
    violation_count INTEGER;
    cleanup_count INTEGER;
BEGIN
    RAISE NOTICE 'Starting schema validation and data cleanup...';
    
    -- 1. Check for duplicate emails within tenants
    SELECT COUNT(*) INTO violation_count
    FROM (
        SELECT email, tenant_id, COUNT(*) as cnt
        FROM users 
        WHERE email IS NOT NULL
        GROUP BY email, tenant_id
        HAVING COUNT(*) > 1
    ) duplicates;
    
    IF violation_count > 0 THEN
        RAISE WARNING 'Found % sets of duplicate emails within tenants', violation_count;
        
        -- Mark duplicate emails with a suffix to make them unique
        WITH ranked_users AS (
            SELECT id, email, tenant_id,
                   ROW_NUMBER() OVER (PARTITION BY email, tenant_id ORDER BY created_at) as rn
            FROM users
            WHERE email IS NOT NULL
        )
        UPDATE users 
        SET email = CONCAT(users.email, '_duplicate_', ru.rn)
        FROM ranked_users ru
        WHERE users.id = ru.id 
        AND ru.rn > 1;
        
        GET DIAGNOSTICS cleanup_count = ROW_COUNT;
        RAISE NOTICE 'Cleaned up % duplicate email records', cleanup_count;
    END IF;
    
    -- 2. Check for duplicate role names within tenants  
    SELECT COUNT(*) INTO violation_count
    FROM (
        SELECT name, tenant_id, COUNT(*) as cnt
        FROM roles 
        WHERE name IS NOT NULL
        GROUP BY name, tenant_id
        HAVING COUNT(*) > 1
    ) duplicates;
    
    IF violation_count > 0 THEN
        RAISE WARNING 'Found % sets of duplicate role names within tenant scopes', violation_count;
        
        -- Mark duplicate role names with a suffix
        WITH ranked_roles AS (
            SELECT id, name, tenant_id,
                   ROW_NUMBER() OVER (PARTITION BY name, tenant_id ORDER BY created_at) as rn
            FROM roles
            WHERE name IS NOT NULL
        )
        UPDATE roles 
        SET name = CONCAT(roles.name, '_dup_', rr.rn)
        FROM ranked_roles rr
        WHERE roles.id = rr.id 
        AND rr.rn > 1;
        
        GET DIAGNOSTICS cleanup_count = ROW_COUNT;
        RAISE NOTICE 'Cleaned up % duplicate role name records', cleanup_count;
    END IF;
    

    
    -- 4. Check for duplicate group names within tenants
    SELECT COUNT(*) INTO violation_count
    FROM (
        SELECT name, tenant_id, COUNT(*) as cnt
        FROM groups 
        WHERE name IS NOT NULL
        GROUP BY name, tenant_id
        HAVING COUNT(*) > 1
    ) duplicates;
    
    IF violation_count > 0 THEN
        RAISE WARNING 'Found % sets of duplicate group names within tenant scopes', violation_count;
        
        WITH ranked_groups AS (
            SELECT id, name, tenant_id,
                   ROW_NUMBER() OVER (PARTITION BY name, tenant_id ORDER BY created_at) as rn
            FROM groups
            WHERE name IS NOT NULL
        )
        UPDATE groups 
        SET name = CONCAT(groups.name, '_dup_', rg.rn)
        FROM ranked_groups rg
        WHERE groups.id = rg.id 
        AND rg.rn > 1;
        
        GET DIAGNOSTICS cleanup_count = ROW_COUNT;
        RAISE NOTICE 'Cleaned up % duplicate group name records', cleanup_count;
    END IF;
    

    
    -- 6. Check for duplicate credential IDs
    SELECT COUNT(*) INTO violation_count
    FROM (
        SELECT credential_id, COUNT(*) as cnt
        FROM credentials 
        WHERE credential_id IS NOT NULL
        GROUP BY credential_id
        HAVING COUNT(*) > 1
    ) duplicates;
    
    IF violation_count > 0 THEN
        RAISE WARNING 'Found % sets of duplicate credential IDs', violation_count;
        
        -- For credentials, we need to be more careful - mark older duplicates as inactive
        WITH ranked_credentials AS (
            SELECT id, credential_id,
                   ROW_NUMBER() OVER (PARTITION BY credential_id ORDER BY created_at DESC) as rn
            FROM credentials
            WHERE credential_id IS NOT NULL
        )
        UPDATE credentials 
        SET active = false,
            updated_at = CURRENT_TIMESTAMP
        FROM ranked_credentials rc
        WHERE credentials.id = rc.id 
        AND rc.rn > 1;
        
        GET DIAGNOSTICS cleanup_count = ROW_COUNT;
        RAISE NOTICE 'Deactivated % duplicate credential records (kept most recent)', cleanup_count;
    END IF;
    
END $$;

-- =====================================================
-- CONSTRAINT VERIFICATION
-- =====================================================

-- Verify all expected constraints exist
DO $$
DECLARE
    constraint_exists BOOLEAN;
    constraint_name_var TEXT;
    table_name_var TEXT;
    expected_constraints TEXT[] := ARRAY[
        'users.users_email_tenant_unique',
        'roles.roles_name_tenant_unique', 

        'credentials.credentials_credential_id_unique',
        'permissions.fk_permissions_role_id',
        'permissions.fk_permissions_scope_id',
        'permissions.fk_permissions_resource_id',
        'resource_methods.fk_resource_methods_resource_id',
        'projects.fk_projects_tenant_id'
    ];
    constraint_record TEXT;
BEGIN
    RAISE NOTICE 'Verifying constraint installation...';
    
    FOREACH constraint_record IN ARRAY expected_constraints
    LOOP
        table_name_var := split_part(constraint_record, '.', 1);
        constraint_name_var := split_part(constraint_record, '.', 2);
        
        SELECT EXISTS (
            SELECT 1 FROM information_schema.table_constraints tc
            WHERE tc.table_name = table_name_var 
            AND tc.constraint_name = constraint_name_var
        ) INTO constraint_exists;
        
        IF constraint_exists THEN
            RAISE NOTICE '✓ Constraint % exists on table %', constraint_name_var, table_name_var;
        ELSE
            RAISE WARNING '✗ Missing constraint % on table %', constraint_name_var, table_name_var;
        END IF;
    END LOOP;
    
    RAISE NOTICE 'Schema constraint validation completed';
END $$;

-- =====================================================
-- PERFORMANCE ANALYSIS RECOMMENDATION
-- =====================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 049: Schema validation and cleanup completed';
    RAISE NOTICE 'Recommendations:';
    RAISE NOTICE '1. Run ANALYZE on all tables to update statistics';
    RAISE NOTICE '2. Monitor query performance improvements';
    RAISE NOTICE '3. Verify application functionality with new constraints';
    RAISE NOTICE '4. Consider pg_stat_user_indexes to monitor index usage';
END $$;
