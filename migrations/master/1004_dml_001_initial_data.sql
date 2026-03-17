-- Initial data inserts for auth-manager
-- Seeds default roles, scopes, and permissions for the system tenant

DO $$
DECLARE
    sys_tenant CONSTANT uuid := '00000000-0000-0000-0000-000000000000';
BEGIN
    -- Ensure system tenant exists
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (sys_tenant, sys_tenant, 'system@authsec.local', 'system.authsec.dev', 'System', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Insert default roles
    INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'admin', 'Administrator role with full access', true, NOW())
    ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO NOTHING;

    INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'user', 'Standard user role', true, NOW())
    ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO NOTHING;

    -- Insert default scopes
    INSERT INTO scopes (id, tenant_id, name, description, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'read', 'Read access scope', NOW())
    ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING;

    INSERT INTO scopes (id, tenant_id, name, description, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'write', 'Write access scope', NOW())
    ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING;

    INSERT INTO scopes (id, tenant_id, name, description, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'delete', 'Delete access scope', NOW())
    ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING;

    INSERT INTO scopes (id, tenant_id, name, description, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'admin', 'Administrative access scope', NOW())
    ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING;

    -- Insert default permissions
    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'users', 'read', 'Read user information', 'users:read', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'users', 'write', 'Create and update users', 'users:write', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'users', 'delete', 'Delete users', 'users:delete', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'roles', 'manage', 'Manage roles and permissions', 'roles:manage', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'tenants', 'manage', 'Manage tenants', 'tenants:manage', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES (gen_random_uuid(), sys_tenant, 'clients', 'manage', 'Manage clients', 'clients:manage', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    -- Assign all permissions to admin role
    INSERT INTO role_permissions (role_id, permission_id)
    SELECT r.id, p.id
    FROM roles r, permissions p
    WHERE r.name = 'admin' AND r.tenant_id = sys_tenant AND p.tenant_id = sys_tenant
    ON CONFLICT DO NOTHING;
END $$;
