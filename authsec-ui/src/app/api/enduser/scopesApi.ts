/**
 * END-USER SCOPES API
 *
 * Endpoints for end-user scope management operations
 * Authentication: Requires AuthMiddleware (user ID and tenant ID extracted from JWT)
 * Base Path: /uflow/user/scopes
 *
 * Available Endpoints:
 * - GET    /uflow/user/scopes                - List all end-user scopes
 * - GET    /uflow/user/scopes/:scope_id      - Get scope by ID
 * - POST   /uflow/user/scopes                - Create end-user scope
 * - PUT    /uflow/user/scopes/:scope_id      - Update end-user scope
 * - DELETE /uflow/user/scopes/:scope_id      - Delete end-user scope
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface EndUserScope {
  id: string;
  scope_name: string;
  resources: string[];
  description?: string;
  created_at?: string;
  updated_at?: string;

  // Computed field for component compatibility
  name: string;
}

export interface CreateEndUserScopeRequest {
  scope_name: string;
  resources: string[];
  description?: string;
}

export interface UpdateEndUserScopeRequest {
  scope_id: string;
  scope_name?: string;
  resources?: string[];
  description?: string;
}

export interface DeleteEndUserScopeRequest {
  scope_id: string;
}

export interface EndUserScopesResponse {
  scopes: EndUserScope[];
}

export interface EndUserScopeMapping {
  scope_name: string;
  resources: string[];
}

export interface ApiResponse {
  message: string;
}

// ============================================================================
// API
// ============================================================================

export const endUserScopesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // GET /uflow/user/scopes
    getEndUserScopes: builder.query<EndUserScope[], void>({
      query: () => "authsec/uflow/user/scopes",
      transformResponse: (response: EndUserScopesResponse | EndUserScope[]) => {
        const scopes = Array.isArray(response) ? response : response.scopes;
        return scopes.map(scope => ({
          ...scope,
          id: scope.id || scope.scope_name,
          name: scope.scope_name,
          resources: scope.resources || [],
        }));
      },
      providesTags: ['EndUserRBACScope'],
    }),

    // GET /uflow/user/scopes/mappings
    getEndUserScopeMappings: builder.query<EndUserScopeMapping[], void>({
      query: () => "authsec/uflow/user/scopes/mappings",
      transformResponse: (response: EndUserScopeMapping[]) => {
        if (!Array.isArray(response)) {
          console.warn('Invalid end-user scope mappings response format:', response);
          return [];
        }
        return response.map(mapping => ({
          ...mapping,
          // Add computed name field for component compatibility
          name: mapping.scope_name,
          id: mapping.scope_name,
        })) as unknown as EndUserScopeMapping[];
      },
      providesTags: ['EndUserRBACScope'],
    }),

    // GET /uflow/user/scopes/:scope_id
    getEndUserScope: builder.query<EndUserScope, string>({
      query: (scope_id) => `authsec/uflow/user/scopes/${scope_id}`,
      transformResponse: (response: EndUserScope) => ({
        ...response,
        id: response.id || response.scope_name,
        name: response.scope_name,
        resources: response.resources || [],
      }),
      providesTags: ['EndUserRBACScope'],
    }),

    // POST /uflow/user/scopes
    createEndUserScope: builder.mutation<ApiResponse, CreateEndUserScopeRequest>({
      query: (body) => ({
        url: "authsec/uflow/user/scopes",
        method: 'POST',
        body: withSessionData(body),
      }),
      invalidatesTags: ['EndUserRBACScope'],
    }),

    // PUT /uflow/user/scopes/:scope_id
    updateEndUserScope: builder.mutation<ApiResponse, UpdateEndUserScopeRequest>({
      query: ({ scope_id, ...body }) => ({
        url: `authsec/uflow/user/scopes/${scope_id}`,
        method: 'PUT',
        body: withSessionData(body),
      }),
      invalidatesTags: ['EndUserRBACScope'],
    }),

    // DELETE /uflow/user/scopes/:scope_id
    deleteEndUserScope: builder.mutation<ApiResponse, DeleteEndUserScopeRequest>({
      query: ({ scope_id }) => ({
        url: `authsec/uflow/user/scopes/${scope_id}`,
        method: 'DELETE',
        body: withSessionData({}),
      }),
      invalidatesTags: ['EndUserRBACScope'],
    }),

  }),
});

export const {
  useGetEndUserScopesQuery,
  useGetEndUserScopeMappingsQuery,
  useGetEndUserScopeQuery,
  useCreateEndUserScopeMutation,
  useUpdateEndUserScopeMutation,
  useDeleteEndUserScopeMutation,
} = endUserScopesApi;
