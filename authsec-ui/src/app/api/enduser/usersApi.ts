/**
 * END-USER USERS API
 *
 * Endpoints for listing and managing end-users (customers/external users)
 * Authentication: Requires AuthMiddleware
 * Base Path: /uflow/enduser
 *
 * Available Endpoints:
 * - POST /uflow/enduser/list - List all end-users for a tenant
 * - GET  /uflow/enduser/:id - Get specific end-user
 * - POST /uflow/enduser/delete - Delete end-user
 * - POST /uflow/enduser/active - Activate/deactivate end-user
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface UsersQueryParams {
  page?: number;
  limit?: number;
  searchQuery?: string;
  active?: boolean;
  provider?: string;
  mfaEnabled?: boolean;
  mfaMethod?: string;
  email?: string;
  name?: string;
  roles?: string[];
  groups?: string[];
  createdAfter?: string;
  createdBefore?: string;
  lastLoginAfter?: string;
  lastLoginBefore?: string;
  client_id?: string;
  tenant_id?: string;
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

export const endUserUsersApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/enduser/list
    // Get end-users from AuthSec API
    getEndUsers: builder.query<any, UsersQueryParams>({
      query: (params = {}) => {
        console.log("[ENDUSER API] 🟢 getEndUsers called - THIS SHOULD BE CALLED FROM USERS PAGE", {
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

        // Provider filters
        if (params.provider) body.provider = params.provider;

        // MFA filters
        if (params.mfaEnabled !== undefined) body.MFAEnabled = params.mfaEnabled;
        if (params.mfaMethod) body.MFAMethod = params.mfaMethod;

        // Role and group filters
        if (params.roles && params.roles.length > 0) body.roles = params.roles;
        if (params.groups && params.groups.length > 0) body.groups = params.groups;

        // Date filters
        if (params.createdAfter) body.created_after = params.createdAfter;
        if (params.createdBefore) body.created_before = params.createdBefore;
        if (params.lastLoginAfter) body.last_login_after = params.lastLoginAfter;
        if (params.lastLoginBefore) body.last_login_before = params.lastLoginBefore;

        // Client filter
        if (params.client_id !== undefined) body.client_id = params.client_id;
        if (params.tenant_id) body.tenant_id = params.tenant_id;

        const payload = withSessionData(body);

        // Preserve explicit "all clients" intent by forcing blank client_id.
        if (params.client_id === "" || params.client_id === undefined || params.client_id === null) {
          payload.client_id = "";
        }

        return {
          url: 'authsec/uflow/admin/enduser/list',
          method: 'POST',
          body: payload,
        };
      },
      providesTags: ['EndUser'],
    }),

    // GET /uflow/enduser/:id
    // Get single user by ID
    getUser: builder.query<any, string>({
      query: (id) => `authsec/uflow/enduser/${id}`,
      providesTags: (result, error, id) => [{ type: 'EndUser', id }],
    }),

    // POST /uflow/admin/enduser/ad/status
    // Check Active Directory configuration status for end users
    checkADConfigStatus: builder.query<ConfigStatus, void>({
      query: () => ({
        url: 'authsec/uflow/admin/enduser/ad/status',
        method: 'POST',
        body: withSessionData({}),
      }),
      providesTags: ['EndUser'],
    }),

    // POST /uflow/admin/enduser/entra/status
    // Check Azure Entra ID configuration status for end users
    checkEntraConfigStatus: builder.query<ConfigStatus, void>({
      query: () => ({
        url: 'authsec/uflow/admin/enduser/entra/status',
        method: 'POST',
        body: withSessionData({}),
      }),
      providesTags: ['EndUser'],
    }),

    // Admin Actions

    // DELETE /uflow/user/enduser/:tenant_id/:user_id
    // Soft delete end user
    deleteUser: builder.mutation<any, { tenant_id: string; user_id: string }>({
      query: ({ tenant_id, user_id }) => ({
        url: `authsec/uflow/user/enduser/${tenant_id}/${user_id}`,
        method: 'DELETE',
        headers: {
          'Content-Type': 'application/json',
        },
      }),
      invalidatesTags: ['EndUser'],
    }),

    // POST /uflow/enduser/active
    // Activate/Deactivate user
    setUserActive: builder.mutation<any, { user_id: string; active: boolean }>({
      query: ({ user_id, active }) => ({
        url: 'authsec/uflow/enduser/active',
        method: 'POST',
        body: withSessionData({
          user_id,
          active: active.toString()
        }),
      }),
      invalidatesTags: ['EndUser'],
    }),

    // POST /uflow/admin/reset-password
    // Reset user password (admin)
    resetUserPassword: builder.mutation<any, { email: string; send_email?: boolean }>({
      query: ({ email, send_email = true }) => ({
        url: 'authsec/uflow/admin/reset-password',
        method: 'POST',
        body: withSessionData({
          email,
          send_email
        }),
      }),
      invalidatesTags: (result, error, { email }) => [
        { type: 'EndUser', id: email }
      ],
    }),

    // POST /uflow/admin/change-password
    // Change user password (admin)
    changeUserPassword: builder.mutation<any, { email: string; new_password: string }>({
      query: ({ email, new_password }) => ({
        url: 'authsec/uflow/admin/change-password',
        method: 'POST',
        body: withSessionData({
          email,
          new_password
        }),
      }),
      invalidatesTags: (result, error, { email }) => [
        { type: 'EndUser', id: email }
      ],
    }),

  }),
});

export const {
  useGetEndUsersQuery,
  useGetUserQuery,
  useCheckADConfigStatusQuery,
  useCheckEntraConfigStatusQuery,
  useDeleteUserMutation,
  useSetUserActiveMutation,
  useResetUserPasswordMutation,
  useChangeUserPasswordMutation,
} = endUserUsersApi;
