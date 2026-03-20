/**
 * ADMIN GROUPS API
 *
 * Endpoints for admin-only group management operations
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin/groups
 *
 * Documentation Reference: rbac-api-documentation.json -> admin_flow.groups
 *
 * Available Endpoints:
 * - POST   /uflow/admin/groups             - Create groups
 * - GET    /uflow/admin/groups/:tenant_id  - Get all groups for tenant
 * - PUT    /uflow/admin/groups/:id         - Update a group
 * - DELETE /uflow/admin/groups             - Delete groups
 * - POST   /uflow/admin/groups/map         - Map groups to client/project
 * - DELETE /uflow/admin/groups/map         - Unmap groups from client/project
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface AdminGroup {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateGroupInput {
  name: string;
}

export interface CreateGroupsRequest {
  tenant_id: string;
  groups: string[] | CreateGroupInput[];
}

export interface CreateGroupsResponse {
  message: string;
  groups: AdminGroup[];
}

export interface UpdateGroupRequest {
  tenant_id: string;
  name?: string;
  description?: string;
}

export interface DeleteGroupsRequest {
  tenant_id: string;
  groups: string[];
}

export interface MapGroupsToClientRequest {
  tenant_id: string;
  client_id: string;
  groups: string[];
}

export interface UnmapGroupsFromClientRequest {
  tenant_id: string;
  client_id: string;
  groups: string[];
}

export interface ApiResponse {
  message: string;
}

// ============================================================================
// API
// ============================================================================

export const adminGroupsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/groups
    createGroups: builder.mutation<CreateGroupsResponse, CreateGroupsRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/groups',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACGroup'],
    }),

    // GET /uflow/admin/groups/:tenant_id
    getGroupsByTenant: builder.query<AdminGroup[], string>({
      query: (tenant_id) => `authsec/uflow/admin/groups/${tenant_id}`,
      transformResponse: (response: { groups: AdminGroup[] }) => response.groups,
      providesTags: ['AdminRBACGroup'],
    }),

    // PUT /uflow/admin/groups/:id
    updateGroup: builder.mutation<ApiResponse, { id: string; data: UpdateGroupRequest }>({
      query: ({ id, data }) => ({
        url: `authsec/uflow/admin/groups/${id}`,
        method: 'PUT',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACGroup'],
    }),

    // DELETE /uflow/admin/groups
    deleteGroups: builder.mutation<ApiResponse, DeleteGroupsRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/groups',
        method: 'DELETE',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACGroup'],
    }),

    // POST /uflow/admin/groups/map
    mapGroupsToClient: builder.mutation<ApiResponse, MapGroupsToClientRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/groups/map',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACGroup', 'AuthSecClient'],
    }),

    // DELETE /uflow/admin/groups/map
    unmapGroupsFromClient: builder.mutation<ApiResponse, UnmapGroupsFromClientRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/groups/map',
        method: 'DELETE',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACGroup', 'AuthSecClient'],
    }),

  }),
});

export const {
  useCreateGroupsMutation,
  useGetGroupsByTenantQuery,
  useUpdateGroupMutation,
  useDeleteGroupsMutation,
  useMapGroupsToClientMutation,
  useUnmapGroupsFromClientMutation,
} = adminGroupsApi;
