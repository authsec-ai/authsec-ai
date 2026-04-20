-- Migration: Add Missing GORM-Expected Indexes
-- Description: This migration adds indexes that GORM auto-migration expects but SQL migrations don't create
-- UPDATED: Removed redundant unique indexes - unique constraints in 012_align_all_constraints.sql already create them

-- ============================================================================
-- REMOVED: REDUNDANT UNIQUE INDEXES
-- ============================================================================
-- These unique indexes are redundant because unique constraints automatically create unique indexes.
-- The unique constraints in migration 012_align_all_constraints.sql handle this requirement.

-- REMOVED: CREATE UNIQUE INDEX IF NOT EXISTS idx_resources_name ON resources (name);
-- REMOVED: CREATE UNIQUE INDEX IF NOT EXISTS idx_roles_name ON roles (name);
-- REMOVED: CREATE UNIQUE INDEX IF NOT EXISTS idx_scopes_name ON scopes (name);
-- REMOVED: CREATE UNIQUE INDEX IF NOT EXISTS idx_groups_name ON groups (name);
-- REMOVED: CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
-- REMOVED: CREATE UNIQUE INDEX IF NOT EXISTS idx_clients_client_id ON clients (client_id);

-- ============================================================================
-- REGULAR INDEXES THAT GORM CREATES
-- ============================================================================

-- Add regular indexes that GORM creates for foreign key relationships
-- Most of these likely already exist from SQL migrations, but ensure they're present

-- Users table indexes
CREATE INDEX IF NOT EXISTS idx_users_active ON users (active);
CREATE INDEX IF NOT EXISTS idx_users_provider ON users (provider, provider_id);
CREATE INDEX IF NOT EXISTS idx_users_project_id ON users (project_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users (tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_client_id ON users (client_id);

-- Projects table indexes
-- CREATE INDEX IF NOT EXISTS idx_projects_deleted_at ON projects (deleted_at);  -- Column doesn't exist
-- idx_projects_tenant_id already exists from SQL migrations
-- CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects (tenant_id);

-- ============================================================================
-- CONSTRAINTS THAT GORM MIGHT EXPECT BUT ARE BUSINESS LOGIC DEPENDENT
-- ============================================================================

-- Note: The following constraints exist in SQL migrations but differ from GORM expectations:
--
-- 1. resources_tenant_id_name_key (SQL) vs idx_resources_name (GORM)
--    SQL: UNIQUE(tenant_id, name) - names unique per tenant
--    GORM: UNIQUE(name) - names globally unique
--    Decision: Keep both for now, but this may cause conflicts
--
-- 2. roles_tenant_id_name_key (SQL) vs idx_roles_name (GORM)
--    SQL: UNIQUE(tenant_id, name) - names unique per tenant
--    GORM: UNIQUE(name) - names globally unique
--    Decision: Keep both for now, but this may cause conflicts
--
-- 3. scopes_tenant_id_name_key (SQL) vs idx_scopes_name (GORM)
--    SQL: UNIQUE(tenant_id, name) - names unique per tenant
--    GORM: UNIQUE(name) - names globally unique
--    Decision: Keep both for now, but this may cause conflicts

-- ============================================================================
-- VERIFICATION: Report on Index Creation
-- ============================================================================

DO $$
DECLARE
    index_report TEXT := '';
    missing_indexes TEXT := '';
    rec RECORD;
    index_exists BOOLEAN;
BEGIN
    RAISE NOTICE '=== GORM INDEX ALIGNMENT VERIFICATION ===';

    -- Check critical unique indexes that GORM expects
    FOR rec IN
        SELECT 'idx_resources_name' as expected_index, 'resources' as table_name, 'name' as column_name
        UNION ALL SELECT 'idx_roles_name', 'roles', 'name'
        UNION ALL SELECT 'idx_scopes_name', 'scopes', 'name'
        UNION ALL SELECT 'idx_groups_name', 'groups', 'name'
        UNION ALL SELECT 'idx_users_email', 'users', 'email'
        UNION ALL SELECT 'idx_clients_client_id', 'clients', 'client_id'
        UNION ALL SELECT 'idx_users_active', 'users', 'active'
        UNION ALL SELECT 'idx_users_provider', 'users', 'provider'
        -- Removed client indexes for non-existent columns:
        -- UNION ALL SELECT 'idx_clients_deleted_at', 'clients', 'deleted_at'  -- Column doesn't exist
        -- UNION ALL SELECT 'idx_clients_status', 'clients', 'status'          -- Column doesn't exist
    LOOP
        SELECT EXISTS(
            SELECT 1 FROM pg_indexes
            WHERE tablename = rec.table_name
            AND indexname = rec.expected_index
            AND schemaname = 'public'
        ) INTO index_exists;

        IF index_exists THEN
            index_report := index_report || format('✓ %s.%s has index %s\n',
                rec.table_name, rec.column_name, rec.expected_index);
        ELSE
            missing_indexes := missing_indexes || format('✗ Missing: %s.%s index %s\n',
                rec.table_name, rec.column_name, rec.expected_index);
        END IF;
    END LOOP;

    RAISE NOTICE 'Index Status Report:';
    RAISE NOTICE '%', index_report;

    IF missing_indexes != '' THEN
        RAISE WARNING 'Missing Indexes:';
        RAISE WARNING '%', missing_indexes;
    ELSE
        RAISE NOTICE 'All expected GORM indexes are present!';
    END IF;

    -- Report on potential conflicts
    RAISE NOTICE '=== POTENTIAL BUSINESS LOGIC CONFLICTS ===';
    RAISE NOTICE 'The following tables have both per-tenant and global uniqueness constraints:';
    RAISE NOTICE '• Resources: resources_tenant_id_name_key (per-tenant) + idx_resources_name (global)';
    RAISE NOTICE '• Roles: roles_tenant_id_name_key (per-tenant) + idx_roles_name (global)';
    RAISE NOTICE '• Scopes: scopes_tenant_id_name_key (per-tenant) + idx_scopes_name (global)';
    RAISE NOTICE 'Review business requirements to determine correct uniqueness scope.';

    RAISE NOTICE '=== END VERIFICATION ===';
END $$;