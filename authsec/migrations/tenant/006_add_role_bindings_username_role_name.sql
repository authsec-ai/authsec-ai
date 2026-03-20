-- Align tenant role_bindings schema with primary schema by adding optional denormalized columns.
-- Adds username and role_name columns (if missing) and aligns scope_type default for wildcard bindings.

ALTER TABLE role_bindings
ADD COLUMN IF NOT EXISTS username TEXT;

ALTER TABLE role_bindings
ADD COLUMN IF NOT EXISTS role_name TEXT;

-- Default scope_type to '*' for tenant-wide/wildcard bindings (matches primary schema behavior)
ALTER TABLE role_bindings
ALTER COLUMN scope_type SET DEFAULT '*';

-- Add simple FKs for clarity (in addition to tenant-scoped constraints)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'role_bindings_user_fk_simple') THEN
        ALTER TABLE role_bindings
        ADD CONSTRAINT role_bindings_user_fk_simple FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'role_bindings_role_fk_simple') THEN
        ALTER TABLE role_bindings
        ADD CONSTRAINT role_bindings_role_fk_simple FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
    END IF;
END$$;
