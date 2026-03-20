/**
 * ADMIN SCOPES API
 *
 * Endpoints for admin-only scope management operations
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin/scopes
 *
 * Documentation Reference: rbac-api-documentation.json -> admin_flow.scopes
 *
 * Available Endpoints:
 * - POST   /uflow/admin/scopes                      - Create scopes
 * - GET    /uflow/admin/scopes/:tenant_id           - Get all scopes for tenant
 * - GET    /uflow/admin/scopes/:tenant_id/:project_id - Get scopes for specific client
 * - PUT    /uflow/admin/scopes/:id                  - Update a scope
 * - DELETE /uflow/admin/scopes                      - Delete scopes
 * - POST   /uflow/admin/scopes/map                  - Map scopes to client/project
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface AdminScope {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface CreateScopeInput {
  name: string;
  description?: string;
}

export interface CreateScopesRequest {
  tenant_id: string;
  scopes: CreateScopeInput[];
}

export interface CreateScopesResponse {
  message: string;
  scopes: AdminScope[];
}

export interface UpdateScopeRequest {
  tenant_id: string;
  name?: string;
  description?: string;
}

export interface DeleteScopesRequest {
  tenant_id: string;
  scope_ids: string[];
}

export interface MapScopesToClientRequest {
  tenant_id: string;
  project_id: string;
  scope_ids: string[];
}

export interface ApiResponse {
  message: string;
}

// ============================================================================
// API
// ============================================================================

export const adminScopesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/scopes
    createScopes: builder.mutation<CreateScopesResponse, CreateScopesRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/scopes',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACScope'],
    }),

    // GET /uflow/admin/scopes/:tenant_id
    getScopesByTenant: builder.query<AdminScope[], string>({
      query: (tenant_id) => `authsec/uflow/admin/scopes/${tenant_id}`,
      transformResponse: (response: { scopes: AdminScope[] }) => response.scopes,
      providesTags: ['AdminRBACScope'],
    }),

    // GET /uflow/admin/scopes/:tenant_id/:project_id
    getScopesByClient: builder.query<AdminScope[], { tenant_id: string; project_id: string }>({
      query: ({ tenant_id, project_id }) => `authsec/uflow/admin/scopes/${tenant_id}/${project_id}`,
      transformResponse: (response: { scopes: AdminScope[] }) => response.scopes,
      providesTags: ['AdminRBACScope'],
    }),

    // PUT /uflow/admin/scopes/:id
    updateScope: builder.mutation<ApiResponse, { id: string; data: UpdateScopeRequest }>({
      query: ({ id, data }) => ({
        url: `authsec/uflow/admin/scopes/${id}`,
        method: 'PUT',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACScope'],
    }),

    // DELETE /uflow/admin/scopes
    deleteScopes: builder.mutation<ApiResponse, DeleteScopesRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/scopes',
        method: 'DELETE',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACScope'],
    }),

    // POST /uflow/admin/scopes/map
    mapScopesToClient: builder.mutation<ApiResponse, MapScopesToClientRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/scopes/map',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['AdminRBACScope', 'AuthSecClient'],
    }),

  }),
});

export const {
  useCreateScopesMutation,
  useGetScopesByTenantQuery,
  useGetScopesByClientQuery,
  useUpdateScopeMutation,
  useDeleteScopesMutation,
  useMapScopesToClientMutation,
} = adminScopesApi;
