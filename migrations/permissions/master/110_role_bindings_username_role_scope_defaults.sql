-- Add username column to role_bindings and ensure role_name exists
ALTER TABLE role_bindings
ADD COLUMN IF NOT EXISTS username TEXT;

ALTER TABLE role_bindings
ADD COLUMN IF NOT EXISTS role_name TEXT;

-- Add simple FK to users(id) and roles(id) for clarity (alongside existing tenant-scoped constraints)
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

-- Default scope_type to '*' for wildcard scopes
ALTER TABLE role_bindings
ALTER COLUMN scope_type SET DEFAULT '*';
