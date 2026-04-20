-- Migration 113: Ensure admin users have wildcard (*) role_bindings in primary DB
-- This assigns a tenant-wide/admin-wide binding to any user with the admin role.
-- Updated to use role_bindings instead of deprecated user_roles table
DO $$
DECLARE
    nil_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Insert wildcard role_bindings for admin users who don't have one yet
    -- This uses existing role_bindings to find admin users (not user_roles which is deprecated)
    INSERT INTO role_bindings (
        id, tenant_id, user_id, role_id, role_name, username,
        scope_type, scope_id, created_at, updated_at
    )
    SELECT
        gen_random_uuid(),
        COALESCE(rb.tenant_id, nil_tenant),
        rb.user_id,
        rb.role_id,
        r.name,
        COALESCE(u.username, ''),
        '*',
        NULL,
        NOW(),
        NOW()
    FROM role_bindings rb
    JOIN roles r ON rb.role_id = r.id
    JOIN users u ON rb.user_id = u.id
    WHERE LOWER(r.name) = 'admin'
      AND rb.scope_type IS NOT NULL 
      AND rb.scope_type != '*'
      AND NOT EXISTS (
          SELECT 1
          FROM role_bindings rb2
          WHERE rb2.user_id = rb.user_id
            AND rb2.role_id = rb.role_id
            AND (rb2.scope_type = '*' OR rb2.scope_type IS NULL)
      )
    ON CONFLICT DO NOTHING;
END $$;
