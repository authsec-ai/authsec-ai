-- Test data for RBAC integration tests
-- Creates test tenants, users, roles, permissions, and role bindings

DO $$
DECLARE
    test_tenant_id CONSTANT uuid := '11111111-1111-1111-1111-111111111111';
BEGIN
    -- Test tenant
    INSERT INTO tenants (id, tenant_id, email, tenant_domain, name, created_at)
    VALUES (test_tenant_id, test_tenant_id, 'test@authsec.local', 'test.authsec.dev', 'Test Tenant', NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Test users (client_id, tenant_id, provider, tenant_domain are required in production)
    INSERT INTO users (id, client_id, tenant_id, email, username, active, provider, tenant_domain, created_at, updated_at)
    VALUES
        ('22222222-2222-2222-2222-222222222222', gen_random_uuid(), test_tenant_id, 'admin@test.com', 'admin', true, 'local', 'test.authsec.dev', NOW(), NOW()),
        ('33333333-3333-3333-3333-333333333333', gen_random_uuid(), test_tenant_id, 'user@test.com', 'user', true, 'local', 'test.authsec.dev', NOW(), NOW())
    ON CONFLICT (id) DO NOTHING;

    -- Test roles
    INSERT INTO roles (id, tenant_id, name, description, is_system, created_at)
    VALUES
        ('44444444-4444-4444-4444-444444444444', test_tenant_id, 'project-admin', 'Project Administrator', false, NOW()),
        ('55555555-5555-5555-5555-555555555555', test_tenant_id, 'project-viewer', 'Project Viewer', false, NOW())
    ON CONFLICT ON CONSTRAINT roles_tenant_name_key DO NOTHING;

    -- Test permissions
    INSERT INTO permissions (id, tenant_id, resource, action, description, full_permission_string, created_at)
    VALUES
        ('66666666-6666-6666-6666-666666666666', test_tenant_id, 'project', 'read', 'Read project', 'project:read', NOW()),
        ('77777777-7777-7777-7777-777777777777', test_tenant_id, 'project', 'write', 'Write project', 'project:write', NOW()),
        ('88888888-8888-8888-8888-888888888888', test_tenant_id, 'project', 'delete', 'Delete project', 'project:delete', NOW()),
        ('99999999-9999-9999-9999-999999999999', test_tenant_id, 'document', 'read', 'Read document', 'document:read', NOW()),
        ('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', test_tenant_id, 'document', 'write', 'Write document', 'document:write', NOW())
    ON CONFLICT ON CONSTRAINT permissions_tenant_resource_action_key DO NOTHING;

    -- Role permissions
    INSERT INTO role_permissions (role_id, permission_id)
    VALUES
        ('44444444-4444-4444-4444-444444444444', '66666666-6666-6666-6666-666666666666'),
        ('44444444-4444-4444-4444-444444444444', '77777777-7777-7777-7777-777777777777'),
        ('44444444-4444-4444-4444-444444444444', '88888888-8888-8888-8888-888888888888'),
        ('55555555-5555-5555-5555-555555555555', '66666666-6666-6666-6666-666666666666')
    ON CONFLICT DO NOTHING;

    -- Test OAuth scopes
    INSERT INTO scopes (id, tenant_id, name, description, created_at)
    VALUES
        ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', test_tenant_id, 'projects:read', 'Read projects scope', NOW()),
        ('cccccccc-cccc-cccc-cccc-cccccccccccc', test_tenant_id, 'projects:write', 'Write projects scope', NOW())
    ON CONFLICT ON CONSTRAINT scopes_tenant_name_key DO NOTHING;

    -- Scope permissions
    INSERT INTO scope_permissions (scope_id, permission_id)
    VALUES
        ('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', '66666666-6666-6666-6666-666666666666'),
        ('cccccccc-cccc-cccc-cccc-cccccccccccc', '66666666-6666-6666-6666-666666666666'),
        ('cccccccc-cccc-cccc-cccc-cccccccccccc', '77777777-7777-7777-7777-777777777777')
    ON CONFLICT DO NOTHING;

    -- Role bindings
    INSERT INTO role_bindings (id, tenant_id, user_id, role_id, scope_type, scope_id, created_at, updated_at)
    VALUES
        ('dddddddd-dddd-dddd-dddd-dddddddddddd', test_tenant_id, '22222222-2222-2222-2222-222222222222', '44444444-4444-4444-4444-444444444444', NULL, NULL, NOW(), NOW()),
        ('eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee', test_tenant_id, '33333333-3333-3333-3333-333333333333', '55555555-5555-5555-5555-555555555555', 'project', 'ffffffff-ffff-ffff-ffff-ffffffffffff', NOW(), NOW())
    ON CONFLICT (id) DO NOTHING;
END $$;
