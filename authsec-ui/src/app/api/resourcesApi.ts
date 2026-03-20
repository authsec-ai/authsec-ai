import { baseApi, withSessionData } from './baseApi';
import type {
  Resource,
  ResourceWithStats,
  ResourceFilters,
  ResourceAnalytics,
  Scope,
  BulkUpdateResult,
  BulkDeleteResult,
  ListParams
} from '@/types/database';

// AuthSec API types
interface AuthSecResource {
  id: string;
  name: string;
  description?: string;
  tenant_id?: string;
  created_at: string;
  updated_at?: string;
}

interface UserDefinedResourceInput {
  name: string;
  description?: string;
}

interface UserDefinedResourcesRequest {
  tenant_id: string;
  resources: UserDefinedResourceInput[];
}

interface DeleteResourcesRequest {
  tenant_id: string;
  resource_ids: string[];
}

interface MapResourcesRequest {
  tenant_id: string;
  project_id?: string;
  client_id?: string;
  resource_ids: string[];
}

export const resourcesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Resources CRUD
    getResources: builder.query<Resource[], ResourceFilters>({
      query: (params = {}) => {
        const searchParams = new URLSearchParams();
        if (params.search) searchParams.append('search', params.search);
        if (params.status) searchParams.append('status', params.status);
        if (params.resource_type) searchParams.append('resource_type', params.resource_type);
        if (params.limit) searchParams.append('limit', params.limit.toString());
        if (params.offset) searchParams.append('offset', params.offset.toString());
        
        return `resources?${searchParams.toString()}`;
      },
      providesTags: ['Resource'],
    }),

    getResource: builder.query<Resource, string>({
      query: (id) => `resources/${id}`,
      providesTags: (result, error, id) => [{ type: 'Resource', id }],
    }),

    createResource: builder.mutation<Resource, Partial<Resource>>({
      query: (data) => ({
        url: 'resources',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['Resource'],
    }),

    updateResource: builder.mutation<Resource, { id: string; data: Partial<Resource> }>({
      query: ({ id, data }) => ({
        url: `resources/${id}`,
        method: 'PUT',
        body: data,
      }),
      invalidatesTags: (result, error, { id }) => [{ type: 'Resource', id }],
    }),

    deleteResource: builder.mutation<void, string>({
      query: (id) => ({
        url: `resources/${id}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['Resource'],
    }),

    // Resource scopes
    getResourceScopes: builder.query<Scope[], string>({
      query: (resourceId) => `resources/${resourceId}/scopes`,
      providesTags: ['Scope'],
    }),

    addResourceScope: builder.mutation<Scope, { resourceId: string; scope: Partial<Scope> }>({
      query: ({ resourceId, scope }) => ({
        url: `resources/${resourceId}/scopes`,
        method: 'POST',
        body: scope,
      }),
      invalidatesTags: ['Resource', 'Scope'],
    }),

    removeResourceScope: builder.mutation<void, { resourceId: string; scopeId: string }>({
      query: ({ resourceId, scopeId }) => ({
        url: `resources/${resourceId}/scopes/${scopeId}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['Resource', 'Scope'],
    }),

    // Bulk operations
    bulkUpdateResources: builder.mutation<BulkUpdateResult, { ids: string[]; data: Partial<Resource> }>({
      query: ({ ids, data }) => ({
        url: 'resources/bulk-update',
        method: 'POST',
        body: { ids, data },
      }),
      invalidatesTags: ['Resource'],
    }),

    bulkDeleteResources: builder.mutation<BulkDeleteResult, string[]>({
      query: (ids) => ({
        url: 'resources/bulk-delete',
        method: 'POST',
        body: { ids },
      }),
      invalidatesTags: ['Resource'],
    }),

    // Resource Analytics
    getResourceAnalytics: builder.query<ResourceAnalytics, string>({
      query: (projectId) => `resources/analytics?project_id=${projectId}`,
      providesTags: ['Resource'],
    }),
  }),
});

export const {
  useGetResourcesQuery,
  useGetResourceQuery,
  useCreateResourceMutation,
  useUpdateResourceMutation,
  useDeleteResourceMutation,
  useGetResourceScopesQuery,
  useAddResourceScopeMutation,
  useRemoveResourceScopeMutation,
  useBulkUpdateResourcesMutation,
  useBulkDeleteResourcesMutation,
  useGetResourceAnalyticsQuery,
} = resourcesApi;

// NOTE: RBAC resource APIs have been moved to separate files:
// - Admin resources: @/app/api/admin/resourcesApi
// - End-user resources: @/app/api/enduser/resourcesApi
// Use those instead of the old authSecResourcesApi endpoints
