-- Migration 115: Drop chk_scope_integrity constraint from role_bindings if it exists
ALTER TABLE IF EXISTS role_bindings
DROP CONSTRAINT IF EXISTS chk_scope_integrity;


alter table if exists role_bindings
drop constraint if exists role_bindings_chk_scope_integrity