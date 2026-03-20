/**
 * END-USER PERMISSIONS API
 *
 * Endpoints for end-users to view their permissions
 * Authentication: Requires AuthMiddleware (user ID and tenant ID extracted from JWT)
 * Base Path: /uflow/user/permissions
 *
 * Documentation Reference: rbac-api-documentation.json -> end_user_flow.permissions
 *
 * Available Endpoints:
 * - GET /uflow/user/permissions           - Get authenticated user's direct permissions
 * - GET /uflow/user/permissions/effective - Get effective permissions (direct + inherited)
 * - GET /uflow/user/permissions/check     - Check if user has specific permission
 */

import { baseApi } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface UserPermission {
  id: string;
  action: string;
  description: string;
  full_permission_string: string;
  resource: string;
  roles_assigned: number;
}

export interface EffectivePermission {
  role_name: string;
  scope_name: string;
  resource_name: string;
  source: 'direct' | 'role' | 'group';
  source_id: string;
}

export interface CheckPermissionParams {
  resource: string;
  scope: string;
}

export interface CheckPermissionResponse {
  has_permission: boolean;
  source?: 'direct' | 'role' | 'group' | null;
  source_id?: string | null;
}

export interface GetMyPermissionsResponse {
  permissions: UserPermission[];
}

export interface GetMyEffectivePermissionsResponse {
  permissions: EffectivePermission[];
}

// ============================================================================
// API
// ============================================================================

export const endUserPermissionsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // GET /uflow/uflow/user/rbac/permissions
    // Get all direct permissions assigned to the authenticated user
    getMyPermissions: builder.query<UserPermission[], void>({
      query: () => 'authsec/uflow/uflow/user/rbac/permissions',
      transformResponse: (response: GetMyPermissionsResponse | UserPermission[]) => {
        if (Array.isArray(response)) return response;
        return response.permissions || [];
      },
      providesTags: ['EndUserRBACPermission', 'EndUser'],
    }),

    // GET /uflow/user/permissions/effective
    // Get all effective permissions (direct + inherited from roles and groups)
    getMyEffectivePermissions: builder.query<EffectivePermission[], void>({
      query: () => 'authsec/uflow/user/permissions/effective',
      transformResponse: (response: GetMyEffectivePermissionsResponse) => response.permissions,
      providesTags: ['EndUserRBACPermission', 'EndUser'],
    }),

    // GET /uflow/user/permissions/check?resource=X&scope=Y
    // Check if the authenticated user has a specific permission
    checkMyPermission: builder.query<CheckPermissionResponse, CheckPermissionParams>({
      query: ({ resource, scope }) => ({
        url: 'authsec/uflow/user/permissions/check',
        params: { resource, scope },
      }),
      providesTags: ['EndUserRBACPermission', 'EndUser'],
    }),

  }),
});

export const {
  useGetMyPermissionsQuery,
  useGetMyEffectivePermissionsQuery,
  useCheckMyPermissionQuery,
} = endUserPermissionsApi;
