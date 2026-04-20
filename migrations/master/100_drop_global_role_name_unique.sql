-- Migration: Drop global UNIQUE(name) constraint on roles
-- roles_name_key and scopes_name_key are global unique constraints that prevent
-- multiple tenants from having roles/scopes with the same name (e.g. "admin").
-- Per-tenant uniqueness is already enforced by (tenant_id, name) constraints.

ALTER TABLE roles DROP CONSTRAINT IF EXISTS roles_name_key;
ALTER TABLE scopes DROP CONSTRAINT IF EXISTS scopes_name_key;
