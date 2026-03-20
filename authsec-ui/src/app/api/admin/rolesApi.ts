/**
 * ADMIN ROLES API
 *
 * Endpoints for admin-only role management operations
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin/roles
 *
 * Documentation Reference: rbac-api-documentation.json -> admin_flow.roles
 *
 * Available Endpoints:
 * - POST   /uflow/admin/roles          - Create roles
 * - GET    /uflow/admin/roles/:tenant_id - Get all roles for tenant
 * - PUT    /uflow/admin/roles/:id      - Update a role
 * - DELETE /uflow/admin/roles          - Delete roles
 * - POST   /uflow/admin/roles/map      - Map roles to client/project
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface AdminRole {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateRoleInput {
  name: string;
  description?: string;
}

export interface CreateRolesRequest {
  tenant_id: string;
  roles: CreateRoleInput[];
}

export interface CreateRolesResponse {
  message: string;
  roles: AdminRole[];
}

export interface UpdateRoleRequest {
  tenant_id: string;
  name?: string;
  description?: string;
}

export interface DeleteRolesRequest {
  tenant_id: string;
  role_ids: string[];
}

export interface MapRolesToClientRequest {
  tenant_id: string;
  project_id: string;
  role_ids: string[];
}

export interface ApiResponse {
  message: string;
}

// ============================================================================
// API
// ============================================================================

export const adminRolesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/roles
    createRoles: builder.mutation<CreateRolesResponse, CreateRolesRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/roles',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACRole'],
    }),

    // GET /uflow/admin/roles/:tenant_id
    getRolesByTenant: builder.query<AdminRole[], string>({
      query: (tenant_id) => `authsec/uflow/admin/roles/${tenant_id}`,
      transformResponse: (response: { roles: AdminRole[] }) => response.roles,
      providesTags: ['AdminRBACRole'],
    }),

    // PUT /uflow/admin/roles/:id
    updateRole: builder.mutation<ApiResponse, { id: string; data: UpdateRoleRequest }>({
      query: ({ id, data }) => ({
        url: `authsec/uflow/admin/roles/${id}`,
        method: 'PUT',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACRole'],
    }),

    // DELETE /uflow/admin/roles
    deleteRoles: builder.mutation<ApiResponse, DeleteRolesRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/roles',
        method: 'DELETE',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACRole'],
    }),

    // POST /uflow/admin/roles/map
    mapRolesToClient: builder.mutation<ApiResponse, MapRolesToClientRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/roles/map',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACRole', 'AuthSecClient'],
    }),

  }),
});

export const {
  useCreateRolesMutation,
  useGetRolesByTenantQuery,
  useUpdateRoleMutation,
  useDeleteRolesMutation,
  useMapRolesToClientMutation,
} = adminRolesApi;
