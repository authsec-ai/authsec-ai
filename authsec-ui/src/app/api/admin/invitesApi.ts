/**
 * ADMIN INVITES API
 *
 * Endpoints for inviting admin users
 * Authentication: Requires AdminAuthMiddleware (admin role in JWT)
 * Base Path: /uflow/admin
 *
 * Available Endpoints:
 * - POST /uflow/admin/invite - Invite an admin user
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface InviteAdminUser {
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

// ============================================================================
// API
// ============================================================================

export const adminInvitesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/invite
    // Invite an admin user
    inviteAdminUser: builder.mutation<InviteResponse, InviteAdminUser>({
      query: (data) => ({
        url: 'authsec/uflow/admin/invite',
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
      invalidatesTags: ['AdminUser'],
    }),

  }),
});

export const {
  useInviteAdminUserMutation,
} = adminInvitesApi;
