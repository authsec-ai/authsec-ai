# Permission Migrations

This directory contains all RBAC (Role-Based Access Control) and permission-related database migrations.

## Directory Structure

```
migrations/permissions/
├── master/          # Permission migrations for master database (36 files)
└── tenant/          # Permission migrations for tenant databases (6 files)
```

## Master Database Permission Migrations (36 files)

### RBAC Core Schema
- `001_create_rbac_tables.sql` - Initial RBAC tables (roles, permissions, role_permissions)
- `029_create_rbac_tables_fixed.sql` - RBAC schema fixes
- `102_scoped_rbac_schema.sql` - Complete scoped RBAC implementation
- `104_enforce_scoped_rbac.sql` - Enforce RBAC constraints
- `105_allow_global_rbac.sql` - Support for global roles
- `107_align_rbac_schema.sql` - Schema alignment

### Roles
- `015_add_group_roles_table.sql` - Group-based role assignments
- `033_drop_user_roles_fk.sql` - Remove user_roles foreign key
- `054_role_scopes_and_user_fixes.sql` - Role scoping
- `062_ensure_global_role_uniqueness.sql` - Unique global roles
- `114_fix_roles_timestamps.sql` - Timestamp fields for roles

### Permissions
- `066_admin_permissions_updates.sql` - Admin permission updates
- `067_admin_route_permission_alignment.sql` - Route permission alignment
- `071_add_permissions_unique_constraint.sql` - Unique constraints
- `079_add_admin_users_active_permission.sql` - User active permission
- `082_add_admin_users_delete_permission.sql` - User delete permission
- `103_add_user_flow_permissions.sql` - User-flow service permissions
- `117_external_service_permissions.sql` - External service permissions
- `119_fix_permissions_unique_constraint.sql` - Fix unique constraints
- `125_reseed_tenant_admin_permissions.sql` - Reseed admin permissions
- `200_add_migration_service_permissions.sql` - Migration service permissions

### Role Bindings
- `025_add_role_assignment_requests.sql` - Role assignment workflow
- `108_add_role_name_to_role_bindings.sql` - Add role name field
- `109_add_full_permission_string.sql` - Full permission strings
- `110_role_bindings_username_role_scope_defaults.sql` - Defaults
- `111_add_username_rolename_to_tenant_role_bindings.sql` - Tenant bindings
- `113_add_admin_wildcard_role_bindings.sql` - Wildcard admin bindings
- `115_drop_role_bindings_scope_integrity.sql` - Drop scope integrity

### Scopes & API Scopes
- `048_replace_scopes_with_mappings.sql` - Scope mappings
- `049_drop_client_scopes.sql` - Remove client scopes
- `050_drop_scope_permissions.sql` - Remove scope permissions
- `065_ensure_global_scope_uniqueness.sql` - Unique global scopes
- `068_allow_client_scoped_user_emails.sql` - Client-scoped emails
- `118_create_api_scopes.sql` - API scope tables
- `124_fix_api_scopes_tenant_fk.sql` - API scope foreign keys

### Admin RBAC
- `036_add_admin_rbac_endpoints.sql` - Admin RBAC endpoints

## Tenant Database Permission Migrations (6 files)

### RBAC Schema
- `003_enforce_scoped_rbac_tenant.sql` - **CORE** tenant RBAC schema
  - Creates roles, permissions, role_permissions, role_bindings tables
  - Sets up service_accounts, scopes, scope_permissions
  - Enforces multi-tenant isolation

### API Scopes
- `004_create_api_scopes_tenant.sql` - API scope tables for tenant
- `005_fix_api_scopes_tenant_fk.sql` - API scope foreign key fixes

### Role Bindings
- `006_add_role_bindings_username_role_name.sql` - Username/role indexes

### Permission Seeding (DML)
- `009_dml_100_seed_external_service_rbac.sql` - External service permissions
  - Permissions: create, read, update, delete, credentials
  - Grants to admin role
  
- `010_dml_003_admin_permissions.sql` - Admin panel permissions
  - Permissions: access, read, write, delete
  - Grants to admin and admin-like roles
  - ⚠️ Modified to handle missing tenants table

## Migration Execution Order

### Master Database
Migrations run in numerical order:
1. RBAC core (001, 029)
2. Groups and roles (015, 033, 054, 062, 114)
3. Permissions (066, 067, 071, 079, 082, 103, 117, 119, 125, 200)
4. Scopes (048, 049, 050, 065, 068, 118, 124)
5. Advanced RBAC (102, 104, 105, 107)
6. Role bindings (025, 108, 109, 110, 111, 113, 115)

### Tenant Database
Migrations run in numerical order:
1. Core RBAC schema (003)
2. API scopes (004, 005)
3. Role bindings optimization (006)
4. Permission seeding (009, 010)

## Architecture Notes

### Master vs Tenant RBAC

**Master Database:**
- Global system roles and permissions
- Cross-tenant RBAC configuration
- Service-level permissions (migration service, external services, etc.)
- Admin panel permissions

**Tenant Database:**
- Isolated tenant-specific RBAC
- Tenant admin and user roles
- Tenant-scoped permissions
- API access control per tenant

### Common Tables (Both Master & Tenant)

These tables exist in both databases for proper isolation:
- `roles` - Role definitions
- `permissions` - Permission definitions
- `role_permissions` - Role-permission mappings
- `role_bindings` - User-role assignments
- `scopes` - API/resource scopes
- `scope_permissions` - Scope-permission mappings
- `api_scopes` - API scope definitions
- `api_scope_permissions` - API scope permissions
- `service_accounts` - Service account management

## Key Features

### Multi-Tenancy
- Tenant ID scoping on all RBAC tables
- Isolation between tenants
- Global roles for system-wide access

### Scoped RBAC
- Resource-based permissions (e.g., `users:read`, `admin:access`)
- Action-based access control
- Scope-based API access

### Flexibility
- Parameterized tenant_id in tenant migrations
- Idempotent migrations (safe to re-run)
- Conditional logic for backward compatibility

## Troubleshooting

### Missing Tenants Table in Tenant DB
Migration 010 handles this gracefully:
- Checks for tenants table existence
- Uses NULL tenant_id if table missing
- Relies on database-level isolation

### Duplicate Permissions
Most migrations use `ON CONFLICT DO NOTHING` to prevent duplicates.

### Permission Verification
```sql
-- List all permissions for a tenant
SELECT resource, action, description 
FROM permissions 
WHERE tenant_id = '<tenant-uuid>';

-- List all role bindings
SELECT u.username, r.name as role_name, rb.created_at
FROM role_bindings rb
JOIN users u ON rb.user_id = u.id
JOIN roles r ON rb.role_id = r.id;
```

## Testing

### Master RBAC Test
```bash
psql -h localhost -U kloudone -d authsec \
  -c "SELECT COUNT(*) FROM roles WHERE is_system = true;"
```

### Tenant RBAC Test
```bash
psql -h localhost -U kloudone -d tenant_<uuid> \
  -c "SELECT resource, action FROM permissions;"
```

---

**Note:** These migrations are executed by the authsec-migration service with automatic retry logic (3 attempts per migration).
