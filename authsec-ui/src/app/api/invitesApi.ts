import { baseApi, withSessionData } from './baseApi';

// Invite API interfaces
export interface InviteUser {
  email: string;
  firstName?: string;
  lastName?: string;
  message?: string;
  roles: string[];
  groups?: string[];
}

export interface DirectorySync {
  provider: string;
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
}

export interface CSVUpload {
  users: Array<{ email: string; name?: string }>;
  roles: string[];
  groups?: string[];
}

export interface SyncResult {
  users_found?: number;
  users_created?: number;
  users_updated?: number;
  success?: boolean;
  message?: string;
}

export const invitesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Send individual user invite
    inviteUser: builder.mutation<any, InviteUser>({
      query: (data) => ({
        url: 'authsec/uflow/admin/invite',
        method: 'POST',
        body: withSessionData({
          email: data.email,
          first_name: data.firstName,
          last_name: data.lastName,
          message: data.message,
          roles: data.roles,
          groups: data.groups || [],
        }),
      }),
      invalidatesTags: ['UnifiedUser'],
    }),

    // Bulk CSV invite
    bulkInviteUsers: builder.mutation<any, CSVUpload>({
      query: (data) => ({
        url: 'authsec/uflow/bulk-invite',
        method: 'POST',
        body: withSessionData({
          users: data.users,
          roles: data.roles,
          groups: data.groups || [],
        }),
      }),
      invalidatesTags: ['UnifiedUser'],
    }),

    // Active Directory sync
    syncActiveDirectory: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => {
        const audiencePath = data.audience === 'admin' ? 'admin/' : '';
        return {
          url: `authsec/uflow/${audiencePath}ad/sync`,
          method: 'POST',
          body: withSessionData({
            dry_run: data.dry_run || false,
            config: data.config,
          }),
        };
      },
      invalidatesTags: ['UnifiedUser'],
    }),

    // Azure Entra ID sync
    syncEntraID: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => {
        const audiencePath = data.audience === 'admin' ? 'admin/' : '';
        return {
          url: `authsec/uflow/${audiencePath}entra/sync`,
          method: 'POST',
          body: withSessionData({
            config: data.config,
            dry_run: data.dry_run || false,
          }),
        };
      },
      invalidatesTags: ['UnifiedUser'],
    }),

    // Generic SCIM sync
    syncSCIM: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => {
        const audiencePath = data.audience === 'admin' ? 'admin/' : '';
        return {
          url: `authsec/uflow/${audiencePath}scim/sync`,
          method: 'POST',
          body: withSessionData({
            provider: data.provider,
            config: data.config,
            dry_run: data.dry_run || false,
          }),
        };
      },
      invalidatesTags: ['UnifiedUser'],
    }),

    // Okta sync
    syncOkta: builder.mutation<SyncResult, DirectorySync>({
      query: (data) => {
        const audiencePath = data.audience === 'admin' ? 'admin/' : '';
        return {
          url: `authsec/uflow/${audiencePath}okta/sync`,
          method: 'POST',
          body: withSessionData({
            config: data.config,
            dry_run: data.dry_run || false,
          }),
        };
      },
      invalidatesTags: ['UnifiedUser'],
    }),
  }),
});

export const {
  useInviteUserMutation,
  useBulkInviteUsersMutation,
  useSyncActiveDirectoryMutation,
  useSyncEntraIDMutation,
  useSyncSCIMMutation,
  useSyncOktaMutation,
} = invitesApi;