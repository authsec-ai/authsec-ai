/**
 * END-USER INVITES API
 *
 * Endpoints for inviting end-users and directory sync
 * Authentication: Requires AuthMiddleware
 * Base Path: /uflow
 *
 * Available Endpoints:
 * - POST /uflow/invite - Invite an end-user
 * - POST /uflow/admin/ad/sync - Sync Active Directory for end-users
 * - POST /uflow/admin/entra/sync - Sync Azure Entra ID for end-users
 * - POST /uflow/admin/admin-users/ad/sync - Sync Active Directory for admin users
 * - POST /uflow/admin/admin-users/entra/sync - Sync Azure Entra ID for admin users
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface InviteEndUser {
  email: string;
  first_name?: string;
  last_name?: string;
  username?: string;
  roles: string[];
  groups?: string[];
  tenant_domain?: string;
  tenant_id?: string;
  client_id?: string;
  project_id?: string;
}

export interface InviteResponse {
  success: boolean;
  message: string;
  user_id?: string;
}

export interface DirectorySync {
  provider: string;
  config_id?: string; // Use stored configuration
  config?: {
    server?: string;
    username?: string;
    password?: string;
    base_dn?: string;
    tenant_id?: string;
    client_id?: string;
    client_secret?: string;
    use_ssl?: boolean;
    skip_verify?: boolean;
  };
  dry_run?: boolean;
  audience?: 'admin' | 'endUser';
  sync_type?: string;
  tenant_id?: string;
  client_id?: string;
  project_id?: string;
}

export interface SyncResult {
  users_found?: number;
  users_created?: number;
  users_updated?: number;
  success?: boolean;
  message?: string;
}

// ============================================================================
// API
// ============================================================================

export const endUserInvitesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/invite
    // Invite an end-user
    inviteEndUser: builder.mutation<InviteResponse, InviteEndUser>({
      query: (data) => ({
        url: 'authsec/uflow/invite',
        method: 'POST',
        body: withSessionData({
          email: data.email,
          first_name: data.first_name,
          last_name: data.last_name,
          username: data.username,
          roles: data.roles,
          groups: data.groups || [],
          tenant_domain: data.tenant_domain,
          tenant_id: data.tenant_id,
          client_id: data.client_id,
          project_id: data.project_id,
        }),
      }),
      invalidatesTags: ['EndUser'],
    }),

    // POST /uflow/admin/ad/sync
    // Active Directory sync for end-users
    syncActiveDirectory: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => ({
        url: 'authsec/uflow/admin/ad/sync',
        method: 'POST',
        body: withSessionData({
          config_id: data.config_id,
          config: data.config,
          dry_run: data.dry_run || false,
          sync_type: data.sync_type || 'ad',
          tenant_id: data.tenant_id,
          client_id: data.client_id,
          project_id: data.project_id,
        }),
      }),
      invalidatesTags: ['EndUser', 'AdminUser', 'SyncConfig'],
    }),

    // POST /uflow/admin/entra/sync
    // Azure Entra ID sync for end-users
    syncEntraID: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => ({
        url: 'authsec/uflow/admin/entra/sync',
        method: 'POST',
        body: withSessionData({
          config_id: data.config_id,
          config: data.config,
          dry_run: data.dry_run || false,
          sync_type: data.sync_type || 'entra_id',
          tenant_id: data.tenant_id,
          client_id: data.client_id,
          project_id: data.project_id,
        }),
      }),
      invalidatesTags: ['EndUser', 'AdminUser', 'SyncConfig'],
    }),

    // POST /uflow/admin/admin-users/ad/sync
    // Active Directory sync to Admin Users list
    syncAdminUsersActiveDirectory: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => ({
        url: 'authsec/uflow/admin/admin-users/ad/sync',
        method: 'POST',
        body: withSessionData({
          config_id: data.config_id,
          config: data.config,
          dry_run: data.dry_run || false,
          sync_type: data.sync_type || 'ad',
          tenant_id: data.tenant_id,
          client_id: data.client_id,
          project_id: data.project_id,
        }),
      }),
      invalidatesTags: ['AdminUser', 'SyncConfig'],
    }),

    // POST /uflow/admin/admin-users/entra/sync
    // Azure Entra ID sync to Admin Users list
    syncAdminUsersEntraID: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => ({
        url: 'authsec/uflow/admin/admin-users/entra/sync',
        method: 'POST',
        body: withSessionData({
          config_id: data.config_id,
          config: data.config,
          dry_run: data.dry_run || false,
          sync_type: data.sync_type || 'entra_id',
          tenant_id: data.tenant_id,
          client_id: data.client_id,
          project_id: data.project_id,
        }),
      }),
      invalidatesTags: ['AdminUser', 'SyncConfig'],
    }),

  }),
});

export const {
  useInviteEndUserMutation,
  useSyncActiveDirectoryMutation,
  useSyncEntraIDMutation,
  useSyncAdminUsersActiveDirectoryMutation,
  useSyncAdminUsersEntraIDMutation,
} = endUserInvitesApi;
