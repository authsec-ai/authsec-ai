/**
 * ADMIN API/OAUTH SCOPES API
 *
 * Endpoints for admin-only API/OAuth scope mapping operations
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin/api_scopes
 *
 * Available Endpoints:
 * - GET    /uflow/admin/api_scopes           - List all API/OAuth scopes
 * - GET    /uflow/admin/api_scopes/:scope_id - Get scope by ID
 * - POST   /uflow/admin/api_scopes           - Create API/OAuth scope
 * - PUT    /uflow/admin/api_scopes/:scope_id - Update API/OAuth scope
 * - DELETE /uflow/admin/api_scopes/:scope_id - Delete API/OAuth scope
 */

import { baseApi, withSessionData } from "../baseApi";
import type {
  ApiOAuthScope,
  ApiOAuthScopeDetails,
  CreateApiOAuthScopeMappingRequest,
  UpdateApiOAuthScopeMappingRequest,
  DeleteApiOAuthScopeMappingRequest,
} from "@/features/api-oauth-scopes/types";

export const adminApiOAuthScopesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // GET /uflow/admin/api_scopes
    getAdminApiOAuthScopes: builder.query<ApiOAuthScope[], void>({
      query: () => "authsec/uflow/admin/api_scopes",
      providesTags: ["AdminApiOAuthScope"],
    }),

    // GET /uflow/admin/api_scopes/:scope_id
    getAdminApiOAuthScope: builder.query<ApiOAuthScopeDetails, string>({
      query: (scope_id) => `authsec/uflow/admin/api_scopes/${scope_id}`,
      providesTags: ["AdminApiOAuthScope"],
    }),

    // POST /uflow/admin/api_scopes
    createAdminApiOAuthScope: builder.mutation<
      any,
      CreateApiOAuthScopeMappingRequest
    >({
      query: (body) => ({
        url: "authsec/uflow/admin/api_scopes",
        method: "POST",
        body: withSessionData(body),
      }),
      invalidatesTags: ["AdminApiOAuthScope"],
    }),

    // PUT /uflow/admin/api_scopes/:scope_id
    updateAdminApiOAuthScope: builder.mutation<
      any,
      UpdateApiOAuthScopeMappingRequest
    >({
      query: ({ scope_id, ...body }) => ({
        url: `authsec/uflow/admin/api_scopes/${scope_id}`,
        method: "PUT",
        body: withSessionData(body),
      }),
      invalidatesTags: ["AdminApiOAuthScope"],
    }),

    // DELETE /uflow/admin/api_scopes/:scope_id
    deleteAdminApiOAuthScope: builder.mutation<
      any,
      DeleteApiOAuthScopeMappingRequest
    >({
      query: ({ scope_id }) => ({
        url: `authsec/uflow/admin/api_scopes/${scope_id}`,
        method: "DELETE",
        body: withSessionData({}),
      }),
      invalidatesTags: ["AdminApiOAuthScope"],
    }),
  }),
});

export const {
  useGetAdminApiOAuthScopesQuery,
  useGetAdminApiOAuthScopeQuery,
  useCreateAdminApiOAuthScopeMutation,
  useUpdateAdminApiOAuthScopeMutation,
  useDeleteAdminApiOAuthScopeMutation,
} = adminApiOAuthScopesApi;
