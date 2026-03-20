/**
 * END-USER RESOURCES API
 *
 * Endpoints for end-user resource management operations (TENANT-SPECIFIC resources)
 * Authentication: Requires AuthMiddleware (user ID and tenant ID extracted from JWT)
 * Base Path: /admin/endusers/:tenant_id/resources
 *
 * Documentation Reference: New RBAC System - End-User Resource Management
 *
 * Available Endpoints:
 * - GET    /admin/endusers/:tenant_id/resources                    - List tenant resources
 * - GET    /admin/endusers/:tenant_id/resources/:resource_id       - Get specific resource
 * - POST   /admin/endusers/:tenant_id/resources                    - Create tenant resource
 * - PUT    /admin/endusers/:tenant_id/resources/:resource_id       - Update resource
 * - DELETE /admin/endusers/:tenant_id/resources/:resource_id       - Delete resource
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface EndUserResource {
  id: string;
  tenant_id: string;
  name: string;
  description?: string;
  created_at: string;
  updated_at?: string;
}

export interface CreateResourceInput {
  name: string;
  description?: string;
}

export interface CreateResourcesResponse {
  resources: EndUserResource[];
}

export interface UpdateResourceRequest {
  name: string;
  description?: string;
}

export interface ApiResponse {
  message?: string;
}

// ============================================================================
// API
// ============================================================================

export const endUserResourcesApi = baseApi.injectEndpoints({
  endpoints: (builder) => {
    const buildPath = (tenant_id: string, suffix = "") =>
      `authsec/uflow/admin/endusers/${tenant_id}/resources${suffix}`;

    return {
      // GET /admin/endusers/:tenant_id/resources - List all tenant resources
      getEndUserResources: builder.query<EndUserResource[], string>({
        query: (tenant_id) => buildPath(tenant_id),
        transformResponse: (response: { resources: EndUserResource[] }) => response.resources,
        providesTags: ["EndUserRBACResource"],
      }),

      // GET /admin/endusers/:tenant_id/resources/:resource_id - Get specific resource
      getEndUserResource: builder.query<
        EndUserResource,
        { tenant_id: string; resource_id: string }
      >({
        query: ({ tenant_id, resource_id }) => buildPath(tenant_id, `/${resource_id}`),
        providesTags: (result, error, { resource_id }) => [
          { type: "EndUserRBACResource", id: resource_id },
        ],
      }),

      // POST /admin/endusers/:tenant_id/resources - Create tenant resource
      createEndUserResource: builder.mutation<
        CreateResourcesResponse,
        { tenant_id: string; data: CreateResourceInput }
      >({
        query: ({ tenant_id, data }) => ({
          url: buildPath(tenant_id),
          method: "POST",
          body: withSessionData(data),
        }),
        invalidatesTags: ["EndUserRBACResource"],
      }),

      // PUT /admin/endusers/:tenant_id/resources/:resource_id - Update resource
      updateEndUserResource: builder.mutation<
        ApiResponse,
        { tenant_id: string; id: string; data: UpdateResourceRequest }
      >({
        query: ({ tenant_id, id, data }) => ({
          url: buildPath(tenant_id, `/${id}`),
          method: "PUT",
          body: withSessionData(data),
        }),
        invalidatesTags: (result, error, { id }) => [
          { type: "EndUserRBACResource", id },
          "EndUserRBACResource",
        ],
      }),

      // DELETE /admin/endusers/:tenant_id/resources/:resource_id - Delete resource
      deleteEndUserResource: builder.mutation<
        ApiResponse,
        { tenant_id: string; resource_id: string }
      >({
        query: ({ tenant_id, resource_id }) => ({
          url: buildPath(tenant_id, `/${resource_id}`),
          method: "DELETE",
        }),
        invalidatesTags: ["EndUserRBACResource"],
      }),
    };
  },
});

export const {
  useGetEndUserResourcesQuery,
  useLazyGetEndUserResourcesQuery,
  useGetEndUserResourceQuery,
  useCreateEndUserResourceMutation,
  useUpdateEndUserResourceMutation,
  useDeleteEndUserResourceMutation,
} = endUserResourcesApi;
