-- Migration: Add performance indexes for admin user queries
-- Issue: ListAdminUsers queries taking 1-2 seconds
-- Fix: Add indexes on frequently queried columns

-- Index for tenant_id on users table (most common filter)
CREATE INDEX IF NOT EXISTS idx_users_tenant_id_active 
ON users(tenant_id, active) 
WHERE active = true;

-- Index for created_at for ordering (DESC for most recent first)
CREATE INDEX IF NOT EXISTS idx_users_created_at_desc 
ON users(created_at DESC);

-- Composite index for role_bindings join optimization
CREATE INDEX IF NOT EXISTS idx_role_bindings_user_tenant 
ON role_bindings(user_id, tenant_id);

-- Index for roles table to speed up role name lookups
CREATE INDEX IF NOT EXISTS idx_roles_tenant_name 
ON roles(tenant_id, LOWER(name));

-- Index for last_login to support sorting by activity
CREATE INDEX IF NOT EXISTS idx_users_last_login 
ON users(last_login DESC) 
WHERE last_login IS NOT NULL;

-- Analyze tables to update statistics for query planner
ANALYZE users;
ANALYZE role_bindings;
ANALYZE roles;
