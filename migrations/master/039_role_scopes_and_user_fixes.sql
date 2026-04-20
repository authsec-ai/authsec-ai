-- Migration 060: Ensure role_scopes table exists with required indexes and relax users.project_id constraint
CREATE TABLE IF NOT EXISTS role_scopes(
    id SERIAL NOT NULL,
    role_id uuid,
    scope_id uuid,
    created_at timestamp with time zone DEFAULT now(),
    PRIMARY KEY(id)
);
-- Create indexes only when columns exist (role_scopes schema may differ if table pre-existed)
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'role_scopes' AND column_name = 'scope_id'
    ) THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_role_scopes_scope_id ON public.role_scopes USING btree (scope_id)';
        EXECUTE 'CREATE UNIQUE INDEX IF NOT EXISTS role_scopes_role_id_scope_id_key ON public.role_scopes USING btree (role_id, scope_id)';
    END IF;
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'role_scopes' AND column_name = 'role_id'
    ) THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_role_scopes_role_id ON public.role_scopes USING btree (role_id)';
    END IF;
END $$;




ALTER TABLE users
ALTER COLUMN project_id DROP NOT NULL;
