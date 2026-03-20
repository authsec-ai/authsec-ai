/**
 * ADMIN USERS API
 *
 * Endpoints for listing and managing admin users (staff/internal users)
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin/users
 *
 * Available Endpoints:
 * - POST /uflow/admin/users/list - List all admin users for a tenant
 * - GET  /uflow/admin/users/:user_id - Get specific admin user
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface AdminUser {
  id: string;
  email: string;
  username: string;
  name: string;
  first_name?: string;
  last_name?: string;
  active: boolean;
  roles?: string[];
  groups?: string[];
  created_at: string;
  updated_at: string;
  last_login?: string;
  mfa_enabled?: boolean;
  mfa_method?: string[] | null;
  mfa_default_method?: string;
  mfa_verified?: boolean;
  client_id?: string;
  tenant_id?: string;
  project_id?: string;
  tenant_domain?: string;
  provider?: string;
  provider_name?: string;
  provider_id?: string;
  provider_data?: any;
  is_synced_user?: boolean;
  is_synced?: boolean;
  external_id?: string;
  sync_source?: string;
  sync_status?: string;
  sync_provider?: string;
  last_sync_at?: string;
  mfa_enrolled_at?: string;
  status?: string;
  accepted_invite?: boolean;
}

export interface AdminUsersQueryParams {
  page?: number;
  limit?: number;
  searchQuery?: string;
  email?: string;
  name?: string;
  active?: boolean;
  roles?: string[];
  created_after?: string;
  created_before?: string;
  tenant_id?: string;
  status?: string;
  provider?: string;
  is_synced?: boolean;
  is_synced_user?: boolean;
  sync_source?: string;
}

export interface AdminUsersResponse {
  users: AdminUser[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface ConfigStatus {
  configured: boolean;
  last_sync?: string;
  user_count?: number;
  status?: 'active' | 'error' | 'pending';
  error?: string;
}

// ============================================================================
// API
// ============================================================================

export const adminUsersApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/users/list
    // List all admin users (staff) for a tenant
    getAdminUsers: builder.query<AdminUsersResponse, AdminUsersQueryParams>({
      query: (params = {}) => {
        console.log("[ADMIN API] 🟡 getAdminUsers called - THIS SHOULD BE CALLED FROM ADMIN USERS PAGE", {
          page: params.page,
          limit: params.limit,
          searchQuery: params.searchQuery,
          provider: params.provider,
          timestamp: new Date().toISOString(),
          currentPath: window.location.pathname,
          stack: new Error().stack
        });

        const body: any = {
          page: params.page || 1,
          limit: params.limit || 50,
        };

        // Search filters
        if (params.searchQuery) {
          body.email = params.searchQuery;
          body.name = params.searchQuery;
        }
        if (params.email) body.email = params.email;
        if (params.name) body.name = params.name;

        // Status filters
        if (params.active !== undefined) body.active = params.active;

        // Role filters
        if (params.roles && params.roles.length > 0) body.roles = params.roles;

        // Date filters
        if (params.created_after) body.created_after = params.created_after;
        if (params.created_before) body.created_before = params.created_before;
        if (params.tenant_id) body.tenant_id = params.tenant_id;
        if (params.status) body.status = params.status;
        if (params.provider) body.provider = params.provider;
        if (params.sync_source) body.sync_source = params.sync_source;
        if (params.is_synced !== undefined) body.is_synced = params.is_synced;
        if (params.is_synced_user !== undefined) body.is_synced_user = params.is_synced_user;

        return {
          url: 'authsec/uflow/admin/users/list',
          method: 'POST',
          body: withSessionData(body),
        };
      },
      providesTags: ['AdminUser'],
    }),

    // GET /uflow/admin/users/:user_id
    // Get a specific admin user by ID
    getAdminUser: builder.query<AdminUser, string>({
      query: (user_id) => `authsec/uflow/admin/users/${user_id}`,
      providesTags: (result, error, id) => [{ type: 'AdminUser', id }],
    }),

    // POST /uflow/admin/users/ad/status
    // Check Active Directory configuration status for admin users
    checkADConfigStatus: builder.query<ConfigStatus, void>({
      query: () => ({
        url: 'authsec/uflow/admin/users/ad/status',
        method: 'POST',
        body: withSessionData({}),
      }),
      providesTags: ['AdminUser'],
    }),

    // POST /uflow/admin/users/entra/status
    // Check Azure Entra ID configuration status for admin users
    checkEntraConfigStatus: builder.query<ConfigStatus, void>({
      query: () => ({
        url: 'authsec/uflow/admin/users/entra/status',
        method: 'POST',
        body: withSessionData({}),
      }),
      providesTags: ['AdminUser'],
    }),

    // Admin Actions (mutations)

    // DELETE /uflow/admin/users/:user_id
    // Soft delete admin user
    deleteAdminUser: builder.mutation<any, { user_id: string }>({
      query: ({ user_id }) => ({
        url: `authsec/uflow/admin/users/${user_id}`,
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
      }),
      invalidatesTags: ['AdminUser'],
    }),

    // POST /uflow/admin/users/active
    // Activate/Deactivate admin user
    setAdminUserActive: builder.mutation<any, { user_id: string; active: boolean }>({
      query: ({ user_id, active }) => ({
        url: 'authsec/uflow/admin/users/active',
        method: 'POST',
        body: withSessionData({
          user_id,
          active: active.toString()
        }),
      }),
      invalidatesTags: ['AdminUser'],
    }),

    // POST /uflow/admin/reset-password
    // Reset admin user password
    resetAdminUserPassword: builder.mutation<any, { email: string; send_email?: boolean }>({
      query: ({ email, send_email = true }) => ({
        url: 'authsec/uflow/admin/reset-password',
        method: 'POST',
        body: withSessionData({
          email,
          send_email
        }),
      }),
      invalidatesTags: (result, error, { email }) => [
        { type: 'AdminUser', id: email }
      ],
    }),

    // POST /uflow/admin/change-password
    // Change admin user password
    changeAdminUserPassword: builder.mutation<any, { email: string; new_password: string }>({
      query: ({ email, new_password }) => ({
        url: 'authsec/uflow/admin/change-password',
        method: 'POST',
        body: withSessionData({
          email,
          new_password
        }),
      }),
      invalidatesTags: (result, error, { email }) => [
        { type: 'AdminUser', id: email }
      ],
    }),

  }),
});

export const {
  useGetAdminUsersQuery,
  useGetAdminUserQuery,
  useCheckADConfigStatusQuery,
  useCheckEntraConfigStatusQuery,
  useDeleteAdminUserMutation,
  useSetAdminUserActiveMutation,
  useResetAdminUserPasswordMutation,
  useChangeAdminUserPasswordMutation,
} = adminUsersApi;
