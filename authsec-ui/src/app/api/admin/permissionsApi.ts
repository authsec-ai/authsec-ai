/**
 * ADMIN PERMISSIONS API
 *
 * Endpoints for admin-only permission management operations
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin/permissions
 *
 * Documentation Reference: rbac-api-documentation.json -> admin_flow.permissions
 *
 * Available Endpoints:
 * - POST   /uflow/admin/permissions             - Create a permission
 * - GET    /uflow/admin/permissions             - Get all permissions (tenant inferred from session)
 * - DELETE /uflow/admin/permissions             - Delete permissions
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface AdminPermission {
  id: string;
  action: string;
  description: string;
  full_permission_string: string;
  resource: string;
  roles_assigned: number;
}

export interface CreatePermissionRequest {
  tenant_id: string;
  role_id: string;
  scope_id: string;
  resource_id: string;
}

export interface CreatePermissionResponse {
  message: string;
  permission: {
    id: string;
    role_id: string;
    scope_id: string;
    resource_id: string;
    created_at: string;
  };
}

export interface DeletePermissionsRequest {
  tenant_id: string;
  permission_ids: string[];
}

export interface ApiResponse {
  message: string;
}

// ============================================================================
// API
// ============================================================================

export const adminPermissionsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/permissions
    createPermission: builder.mutation<CreatePermissionResponse, CreatePermissionRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/permissions',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACPermission'],
    }),

    // GET /uflow/uflow/admin/permissions
    getPermissionsByTenant: builder.query<AdminPermission[], void>({
      query: () => 'authsec/uflow/admin/permissions',
      transformResponse: (response: { permissions: AdminPermission[] } | AdminPermission[]) => {
        if (Array.isArray(response)) return response;
        return response.permissions || [];
      },
      providesTags: ['AdminRBACPermission'],
    }),

    // DELETE /uflow/admin/permissions
    deletePermissions: builder.mutation<ApiResponse, DeletePermissionsRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/permissions',
        method: 'DELETE',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACPermission'],
    }),

  }),
});

export const {
  useCreatePermissionMutation,
  useGetPermissionsByTenantQuery,
  useDeletePermissionsMutation,
} = adminPermissionsApi;
