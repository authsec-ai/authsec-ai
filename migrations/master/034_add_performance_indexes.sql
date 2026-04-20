-- Migration: Add Performance Indexes
-- Description: Add critical indexes for frequently queried columns to improve performance
-- NOTE: Using regular CREATE INDEX (not CONCURRENTLY) to allow running inside transactions

-- =====================================================
-- TENANT-ID INDEXES (CRITICAL for multi-tenant performance)
-- =====================================================

-- Users table - tenant filtering
CREATE INDEX IF NOT EXISTS idx_users_tenant_id ON users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_users_tenant_email ON users(tenant_id, email);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(active) WHERE active = true;

-- Roles table - tenant filtering
CREATE INDEX IF NOT EXISTS idx_roles_tenant_id ON roles(tenant_id);
CREATE INDEX IF NOT EXISTS idx_roles_tenant_name ON roles(tenant_id, name);

-- Scopes table - tenant filtering
CREATE INDEX IF NOT EXISTS idx_scopes_tenant_id ON scopes(tenant_id);
CREATE INDEX IF NOT EXISTS idx_scopes_tenant_name ON scopes(tenant_id, name);

-- Groups table - tenant filtering
CREATE INDEX IF NOT EXISTS idx_groups_tenant_id ON groups(tenant_id);
CREATE INDEX IF NOT EXISTS idx_groups_tenant_name ON groups(tenant_id, name);

-- Projects table - tenant filtering
CREATE INDEX IF NOT EXISTS idx_projects_tenant_id ON projects(tenant_id);
CREATE INDEX IF NOT EXISTS idx_projects_active ON projects(active) WHERE active = true;

-- =====================================================
-- TIMESTAMP INDEXES (for audit trails and time-based queries)
-- =====================================================

-- Users table timestamps
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);
CREATE INDEX IF NOT EXISTS idx_users_updated_at ON users(updated_at);

-- Roles table timestamps
CREATE INDEX IF NOT EXISTS idx_roles_created_at ON roles(created_at);
CREATE INDEX IF NOT EXISTS idx_roles_updated_at ON roles(updated_at);

-- Scopes table timestamps
CREATE INDEX IF NOT EXISTS idx_scopes_created_at ON scopes(created_at);
CREATE INDEX IF NOT EXISTS idx_scopes_updated_at ON scopes(updated_at);

-- Groups table timestamps
CREATE INDEX IF NOT EXISTS idx_groups_created_at ON groups(created_at);
CREATE INDEX IF NOT EXISTS idx_groups_updated_at ON groups(updated_at);

-- Projects table timestamps
CREATE INDEX IF NOT EXISTS idx_projects_created_at ON projects(created_at);
CREATE INDEX IF NOT EXISTS idx_projects_updated_at ON projects(updated_at);

-- Permissions table timestamps
CREATE INDEX IF NOT EXISTS idx_permissions_created_at ON permissions(created_at);

-- Credentials table timestamps
CREATE INDEX IF NOT EXISTS idx_credentials_created_at ON credentials(created_at);
CREATE INDEX IF NOT EXISTS idx_credentials_updated_at ON credentials(updated_at);

-- =====================================================
-- SPECIALIZED INDEXES FOR COMMON QUERIES
-- =====================================================

-- Tenants by status
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);

-- =====================================================
-- INDEX MAINTENANCE AND MONITORING
-- =====================================================

DO $$
BEGIN
    RAISE NOTICE 'Migration 048: Performance indexes created successfully';
    
    -- Log index statistics
    RAISE NOTICE 'Index Summary:';
    RAISE NOTICE '- Tenant-scoped indexes: Critical for multi-tenant queries';  
    RAISE NOTICE '- Timestamp indexes: For audit and time-based filtering';
    RAISE NOTICE '- Composite indexes: For common multi-column queries';
    RAISE NOTICE '- Partial indexes: For active records (space efficient)';
    
    -- Recommend running ANALYZE after index creation
    RAISE NOTICE 'Recommendation: Run ANALYZE on affected tables to update query planner statistics';
END $$;
