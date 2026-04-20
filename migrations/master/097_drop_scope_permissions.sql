-- Migration: Drop scope_permissions table
-- Description: Removes the scope_permissions table as it is no longer used (replaced by scope_resource_mappings).

DROP TABLE IF EXISTS scope_permissions CASCADE;
