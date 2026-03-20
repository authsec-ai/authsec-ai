/**
 * Admin RBAC APIs - Index
 *
 * All admin-only endpoints for managing RBAC entities.
 * These require AdminAuthMiddleware (admin role in JWT).
 */

export * from './rolesApi';
export * from './scopesApi';
export * from './resourcesApi';
export * from './permissionsApi';
export * from './usersApi';
