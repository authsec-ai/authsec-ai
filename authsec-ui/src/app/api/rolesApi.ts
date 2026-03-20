import { baseApi, withSessionData } from './baseApi';
import type {
  Role,
  RoleWithStats,
  RoleFilters,
  RoleAnalytics,
  RolePermission,
  BulkUpdateResult,
  BulkDeleteResult,
  ListParams
} from '@/types/database';

// AuthSec API types
interface AuthSecRole {
  id: string;
  name: string;
  description?: string;
  tenant_id?: string;
  created_at?: string;
  updated_at?: string;
  permissions_count?: number;
  users_assigned?: number;
  user_ids?: string[];
  usernames?: string[];
  group_ids?: string[];
}

interface UserDefinedRoleRequest {
  tenant_id: string;
  name: string;
  description?: string;
  permission_ids?: string[];
  permission_strings?: string[];
  audience?: 'admin' | 'endUser';
}

interface CreateUserDefinedRoleResponse extends Partial<AuthSecRole> {
  roles?: AuthSecRole[];
  message?: string;
  success?: boolean;
}

interface DeleteRolesRequest {
  tenant_id: string;
  role_ids: string[];
  audience?: 'admin' | 'endUser';
}

interface MapRolesRequest {
  tenant_id: string;
  project_id?: string;
  client_id?: string;
  role_ids: string[];
}

export const rolesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Roles CRUD
    getRoles: builder.query<Role[], RoleFilters>({
      query: (params = {}) => {
        const searchParams = new URLSearchParams();
        if (params.search) searchParams.append('search', params.search);
        if (params.status) searchParams.append('status', params.status);
        if (params.role_type) searchParams.append('role_type', params.role_type);
        if (params.limit) searchParams.append('limit', params.limit.toString());
        if (params.offset) searchParams.append('offset', params.offset.toString());
        
        return `roles?${searchParams.toString()}`;
      },
      providesTags: ['Role'],
    }),

    getRole: builder.query<Role, string>({
      query: (id) => `roles/${id}`,
      providesTags: (result, error, id) => [{ type: 'Role', id }],
    }),

    createRole: builder.mutation<Role, Partial<Role>>({
      query: (data) => ({
        url: 'roles',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['Role'],
    }),

    updateRole: builder.mutation<Role, { id: string; data: Partial<Role> }>({
      query: ({ id, data }) => ({
        url: `roles/${id}`,
        method: 'PUT',
        body: data,
      }),
      invalidatesTags: (result, error, { id }) => [{ type: 'Role', id }],
    }),

    deleteRole: builder.mutation<void, string>({
      query: (id) => ({
        url: `roles/${id}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['Role'],
    }),

    // Role permissions
    getRolePermissions: builder.query<RolePermission[], string>({
      query: (roleId) => `roles/${roleId}/permissions`,
      providesTags: ['RolePermission'],
    }),

    addRolePermission: builder.mutation<RolePermission, { roleId: string; permission: Partial<RolePermission> }>({
      query: ({ roleId, permission }) => ({
        url: `roles/${roleId}/permissions`,
        method: 'POST',
        body: permission,
      }),
      invalidatesTags: ['Role', 'RolePermission'],
    }),

    removeRolePermission: builder.mutation<void, { roleId: string; permissionId: string }>({
      query: ({ roleId, permissionId }) => ({
        url: `roles/${roleId}/permissions/${permissionId}`,
        method: 'DELETE',
      }),
      invalidatesTags: ['Role', 'RolePermission'],
    }),

    // Bulk operations
    bulkUpdateRoles: builder.mutation<BulkUpdateResult, { ids: string[]; data: Partial<Role> }>({
      query: ({ ids, data }) => ({
        url: 'roles/bulk-update',
        method: 'POST',
        body: { ids, data },
      }),
      invalidatesTags: ['Role'],
    }),

    bulkDeleteRoles: builder.mutation<BulkDeleteResult, string[]>({
      query: (ids) => ({
        url: 'roles/bulk-delete',
        method: 'POST',
        body: { ids },
      }),
      invalidatesTags: ['Role'],
    }),

    // Role Analytics
    getRoleAnalytics: builder.query<RoleAnalytics, string>({
      query: (projectId) => `roles/analytics?project_id=${projectId}`,
      providesTags: ['Role'],
    }),
  }),
});

export const {
  useGetRolesQuery,
  useGetRoleQuery,
  useCreateRoleMutation,
  useUpdateRoleMutation,
  useDeleteRoleMutation,
  useGetRolePermissionsQuery,
  useAddRolePermissionMutation,
  useRemoveRolePermissionMutation,
  useBulkUpdateRolesMutation,
  useBulkDeleteRolesMutation,
  useGetRoleAnalyticsQuery,
} = rolesApi;

// AuthSec API endpoints
export const authSecRolesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Get user-defined roles for a tenant
    getAuthSecRoles: builder.query<AuthSecRole[], { tenant_id: string; audience?: 'admin' | 'endUser' }>({
      async queryFn(args, _api, _extraOptions, baseQuery) {
        const tenantId = (args?.tenant_id ?? '').trim();
        const audience = args?.audience ?? 'admin';

        const basePath =
          audience === 'admin' ? 'authsec/uflow/admin/roles' : 'authsec/uflow/user/rbac/roles';

        const candidateEndpoints = [
          tenantId ? `${basePath}/${encodeURIComponent(tenantId)}` : null,
          basePath,
        ].filter((endpoint): endpoint is string => Boolean(endpoint));

        let lastError: unknown = null;

        for (const endpoint of candidateEndpoints) {
          const result = await baseQuery(endpoint);

          if (result.error) {
            lastError = result.error;
            continue;
          }

          const response = result.data as any;

          if (!response) {
            return { data: [] };
          }

          // Common shapes: raw array, { roles: [...] }, { data: [...] }
          if (Array.isArray(response)) {
            return { data: response };
          }

          if (Array.isArray(response.roles)) {
            return { data: response.roles };
          }

          if (Array.isArray(response.data)) {
            return { data: response.data };
          }

          return { data: [] };
        }

        return {
          error:
            (lastError as any) ?? {
              status: 'CUSTOM_ERROR',
              error: 'Unable to fetch roles',
            },
        };
      },
      providesTags: ['UnifiedRBACRole'],
    }),

    // Add user-defined roles
    addUserDefinedRoles: builder.mutation<CreateUserDefinedRoleResponse, UserDefinedRoleRequest>({
      query: ({ audience = 'admin', ...data }) => ({
        url: audience === 'admin' ? 'authsec/uflow/admin/roles' : 'authsec/uflow/user/rbac/roles',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['UnifiedRBACRole'],
    }),

    // Update a role
    updateUserDefinedRole: builder.mutation<{ message?: string; success?: boolean }, { id: string; data: { tenant_id: string; name: string; description?: string } }>({
      query: ({ id, data }) => ({
        url: `authsec/uflow/admin/roles/${id}`,
        method: 'PUT',
        body: withSessionData(data),
      }),
      invalidatesTags: ['UnifiedRBACRole'],
    }),

    // Delete user-defined roles
    deleteUserDefinedRoles: builder.mutation<{ message?: string; success?: boolean }, DeleteRolesRequest>({
      query: ({ audience = 'admin', ...data }) => ({
        url: audience === 'admin' ? 'authsec/uflow/admin/roles' : 'authsec/uflow/user/rbac/roles',
        method: 'DELETE',
        body: withSessionData(data),
      }),
      invalidatesTags: ['UnifiedRBACRole'],
    }),

    // Map roles to client
    mapRolesToClient: builder.mutation<{ message?: string; success?: boolean }, MapRolesRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/roles/map',
        method: 'POST',
        body: withSessionData({
          ...data,
          project_id: data.project_id || data.client_id,
        }),
      }),
      invalidatesTags: ['AuthSecClient', 'UnifiedRBACRole'],
    }),
  }),
});

export const {
  useGetAuthSecRolesQuery,
  useAddUserDefinedRolesMutation,
  useUpdateUserDefinedRoleMutation,
  useDeleteUserDefinedRolesMutation,
  useMapRolesToClientMutation,
} = authSecRolesApi;
