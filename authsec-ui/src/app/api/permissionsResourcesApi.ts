/**
 * PERMISSION RESOURCES API
 *
 * Endpoints for fetching unique resources from permissions
 * - Admin: GET /uflow/admin/permissions/resources
 * - EndUser: GET /uflow/user/rbac/permissions/resources
 */

import { baseApi } from './baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface PermissionResource {
    name: string;
}

interface PermissionResourcesResponse {
    resources: string[];
}

// ============================================================================
// API
// ============================================================================

export const permissionsResourcesApi = baseApi.injectEndpoints({
    endpoints: (builder) => ({
        // GET /uflow/admin/permissions/resources - List unique resources for admin
        getAdminPermissionResources: builder.query<string[], void>({
            query: () => 'authsec/uflow/admin/permissions/resources',
            transformResponse: (response: PermissionResourcesResponse) => response.resources || [],
            providesTags: ['AdminRBACResource'],
        }),

        // GET /uflow/user/rbac/permissions/resources - List unique resources for enduser
        getEndUserPermissionResources: builder.query<string[], string>({
            query: (tenant_id) => `authsec/uflow/user/rbac/permissions/resources?tenant_id=${tenant_id}`,
            transformResponse: (response: PermissionResourcesResponse) => response.resources || [],
            providesTags: ['EndUserRBACResource'],
        }),
    }),
});

export const {
    useGetAdminPermissionResourcesQuery,
    useGetEndUserPermissionResourcesQuery,
} = permissionsResourcesApi;
