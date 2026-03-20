/**
 * END-USER API/OAUTH SCOPES API
 *
 * Endpoints for end-user API/OAuth scope mapping operations
 * Authentication: Requires AuthMiddleware (user ID and tenant ID from JWT)
 * Base Path: /uflow/user/api_scopes
 *
 * Available Endpoints:
 * - GET    /uflow/user/api_scopes           - List all API/OAuth scopes
 * - GET    /uflow/user/api_scopes/:scope_id - Get scope by ID
 * - POST   /uflow/user/api_scopes           - Create API/OAuth scope
 * - PUT    /uflow/user/api_scopes/:scope_id - Update API/OAuth scope
 * - DELETE /uflow/user/api_scopes/:scope_id - Delete API/OAuth scope
 */

import { baseApi, withSessionData } from "../baseApi";
import type {
  ApiOAuthScope,
  ApiOAuthScopeDetails,
  CreateApiOAuthScopeMappingRequest,
  UpdateApiOAuthScopeMappingRequest,
  DeleteApiOAuthScopeMappingRequest,
} from "@/features/api-oauth-scopes/types";

export const endUserApiOAuthScopesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // GET /uflow/user/api_scopes
    getEndUserApiOAuthScopes: builder.query<ApiOAuthScope[], void>({
      query: () => "authsec/uflow/user/api_scopes",
      providesTags: ["EndUserApiOAuthScope"],
    }),

    // GET /uflow/user/api_scopes/:scope_id
    getEndUserApiOAuthScope: builder.query<ApiOAuthScopeDetails, string>({
      query: (scope_id) => `authsec/uflow/user/api_scopes/${scope_id}`,
      providesTags: ["EndUserApiOAuthScope"],
    }),

    // POST /uflow/user/api_scopes
    createEndUserApiOAuthScope: builder.mutation<
      any,
      CreateApiOAuthScopeMappingRequest
    >({
      query: (body) => ({
        url: "authsec/uflow/user/api_scopes",
        method: "POST",
        body: withSessionData(body),
      }),
      invalidatesTags: ["EndUserApiOAuthScope"],
    }),

    // PUT /uflow/user/api_scopes/:scope_id
    updateEndUserApiOAuthScope: builder.mutation<
      any,
      UpdateApiOAuthScopeMappingRequest
    >({
      query: ({ scope_id, ...body }) => ({
        url: `authsec/uflow/user/api_scopes/${scope_id}`,
        method: "PUT",
        body: withSessionData(body),
      }),
      invalidatesTags: ["EndUserApiOAuthScope"],
    }),

    // DELETE /uflow/user/api_scopes/:scope_id
    deleteEndUserApiOAuthScope: builder.mutation<
      any,
      DeleteApiOAuthScopeMappingRequest
    >({
      query: ({ scope_id }) => ({
        url: `authsec/uflow/user/api_scopes/${scope_id}`,
        method: "DELETE",
        body: withSessionData({}),
      }),
      invalidatesTags: ["EndUserApiOAuthScope"],
    }),
  }),
});

export const {
  useGetEndUserApiOAuthScopesQuery,
  useGetEndUserApiOAuthScopeQuery,
  useCreateEndUserApiOAuthScopeMutation,
  useUpdateEndUserApiOAuthScopeMutation,
  useDeleteEndUserApiOAuthScopeMutation,
} = endUserApiOAuthScopesApi;
