import { baseApi, withSessionData } from './baseApi';

// AuthSec Permissions API Interfaces
export interface Permission {
  id: string;
  action: string;
  description: string;
  full_permission_string: string;
  resource: string;
  roles_assigned: number;
}

// Atomic permission creation (resource + action)
export interface CreatePermissionRequest {
  resource: string;
  action: string;
  description?: string;
}

// Legacy composite permission (kept for backward compatibility if needed)
export interface LegacyCreatePermissionRequest {
  role_id: string;
  scope_id: string;
  resource_id: string;
}

export interface DeletePermissionsRequest {
  tenant_id: string;
  permission_ids: string[];
}

// Delete permission by body (resource + action + description)
export interface DeletePermissionByBodyRequest {
  resource: string;
  action: string;
  description?: string;
}

export interface PermissionsResponse {
  permissions?: Permission[];
  message?: string;
  success?: boolean;
}

export interface GenericDeleteResponse {
  property1?: string;
  property2?: string;
}

export interface ResourcesResponse {
  resources: string[];
}

export interface CreatePermissionResponse {
  id: string;
  full_string: string;
}

// For effective permissions calculation
export interface EffectivePermission {
  resource_id: string;
  resource_name: string;
  scopes: {
    scope_id: string;
    scope_name: string;
    source: 'direct' | 'inherited';
    source_name?: string; // Group name or "Direct Assignment"
  }[];
}

export const permissionsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Get available permission resources
    getPermissionResources: builder.query<string[], { audience: 'admin' | 'endUser' }>({
      query: ({ audience }) => {
        if (audience === 'admin') {
          return 'authsec/uflow/admin/permissions/resources';
        }
        return 'authsec/uflow/user/rbac/permissions/resources';
      },
      transformResponse: (response: ResourcesResponse) => {
        if (!response || !Array.isArray(response.resources)) {
          console.warn('Invalid resources API response:', response);
          return [];
        }
        return response.resources;
      },
      providesTags: ["UnifiedRBACPermission"],
    }),

    // List all permissions for a tenant
    getPermissions: builder.query<Permission[], { tenant_id: string; audience: 'admin' | 'endUser' }>({
      query: ({ audience }) => {
        if (audience === 'admin') {
          return 'authsec/uflow/admin/permissions';
        }
        return 'authsec/uflow/user/rbac/permissions';
      },
      transformResponse: (response: any) => {
        if (!response) {
          console.warn('Invalid permissions API response:', response);
          return [];
        }
        // Response is an array directly
        if (Array.isArray(response)) {
          return response;
        }
        if (Array.isArray(response.permissions)) {
          return response.permissions;
        }
        console.warn('Permissions response missing permissions array:', response);
        return [];
      },
      providesTags: ["UnifiedRBACPermission"],
    }),

    // Create an atomic permission (resource + action)
    createPermission: builder.mutation<CreatePermissionResponse, CreatePermissionRequest & { audience: 'admin' | 'endUser' }>({
      query: ({ audience, ...data }) => ({
        url: audience === 'admin' ? "authsec/uflow/admin/permissions" : "authsec/uflow/user/rbac/permissions",
        method: "POST",
        body: withSessionData(data),
      }),
      invalidatesTags: ["UnifiedRBACPermission"],
    }),

    // Bulk create permissions
    createPermissions: builder.mutation<PermissionsResponse, { permissions: CreatePermissionRequest[] }>({
      query: (data) => ({
        url: "authsec/uflow/admin/permissions/bulk",
        method: "POST",
        body: withSessionData(data),
      }),
      invalidatesTags: ["UnifiedRBACPermission"],
    }),

    // Delete permissions (legacy bulk delete)
    deletePermissions: builder.mutation<PermissionsResponse, DeletePermissionsRequest>({
      query: (data) => ({
        url: "authsec/uflow/admin/permissions",
        method: "DELETE",
        body: withSessionData(data),
      }),
      invalidatesTags: ["UnifiedRBACPermission"],
    }),

    // NEW: Delete Permission by Body (Main DB) - DELETE /uflow/admin/permissions
    // Deletes a permission using resource and action in body
    deletePermissionByBody: builder.mutation<GenericDeleteResponse, DeletePermissionByBodyRequest & { audience: 'admin' | 'endUser' }>({
      query: ({ audience, ...data }) => ({
        url: audience === 'admin' ? "authsec/uflow/admin/permissions" : "authsec/uflow/user/rbac/permissions",
        method: "DELETE",
        body: data,
      }),
      invalidatesTags: ["UnifiedRBACPermission"],
    }),

    // NEW: Delete Permission by ID (Main DB) - DELETE /uflow/admin/permissions/{id}
    // Deletes a permission by its ID
    deletePermissionById: builder.mutation<GenericDeleteResponse, { id: string; audience: 'admin' | 'endUser' }>({
      query: ({ id, audience }) => ({
        url: audience === 'admin' ? `authsec/uflow/admin/permissions/${id}` : `authsec/uflow/user/rbac/permissions/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: ["UnifiedRBACPermission"],
    }),

    // Get effective permissions for a user (including inherited from groups)
    getUserEffectivePermissions: builder.query<EffectivePermission[], { tenantId: string; userId: string }>({
      query: ({ tenantId, userId }) => `authsec/uflow/user/permissions/${tenantId}/${userId}`,
      transformResponse: (response: any) => {
        if (!response || !Array.isArray(response.effective_permissions)) {
          console.warn('Invalid effective permissions response:', response);
          return [];
        }
        return response.effective_permissions;
      },
      providesTags: ["UnifiedRBACPermission", "UnifiedRBACGroup"],
    }),

    // Get permissions by role
    getRolePermissions: builder.query<Permission[], { tenantId: string; roleId: string }>({
      query: ({ tenantId, roleId }) => `authsec/uflow/admin/permissions/${tenantId}/role/${roleId}`,
      transformResponse: (response: any) => {
        if (!response || !Array.isArray(response)) {
          console.warn('Invalid role permissions response:', response);
          return [];
        }
        return response;
      },
      providesTags: ["UnifiedRBACPermission"],
    }),
  }),
});

export const {
  useGetPermissionResourcesQuery,
  useGetPermissionsQuery,
  useCreatePermissionMutation,
  useCreatePermissionsMutation,
  useDeletePermissionsMutation,
  // New delete methods
  useDeletePermissionByBodyMutation,
  useDeletePermissionByIdMutation,
  // Other queries
  useGetUserEffectivePermissionsQuery,
  useGetRolePermissionsQuery,
} = permissionsApi;
