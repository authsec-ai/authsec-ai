/**
 * UNIFIED GROUPS API - Simple Wrapper
 *
 * This file provides a clean, unified interface that switches between
 * admin and enduser group APIs based on audience context.
 *
 * Uses the separate admin and enduser API files under the hood.
 */

import { baseApi, withSessionData } from './baseApi';

// Re-export types
export type { AdminGroup as Group } from './admin/groupsApi';

export interface UnifiedGetGroupsParams {
  tenant_id: string;
  audience: 'admin' | 'endUser';
  user_id?: string;
}

/**
 * Unified Groups API
 *
 * Provides a single getGroups query that routes to the correct endpoint
 * based on the audience parameter.
 */
export const groupsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    /**
     * Get Groups - Unified Query
     *
     * Routes to:
     * - Admin: GET /uflow/admin/groups/:tenant_id
     * - Enduser (impersonation via admin): POST /uflow/admin/groups/list with user_id filter
     * - Enduser (self-service): GET /uflow/user/groups/users
     */
    getGroups: builder.query<any[], UnifiedGetGroupsParams>({
      query: ({ tenant_id, audience, user_id }) => {
        if (audience === 'admin') {
          return `authsec/uflow/admin/groups/${tenant_id}`;
        }

        if (user_id) {
          const requestBody = {
            tenant_id,
            user_id,
          };

          return {
            url: 'authsec/uflow/admin/groups/list',
            method: 'POST',
            body: withSessionData(requestBody),
          };
        }

        // End-user self-service call
        return {
          url: 'authsec/uflow/user/groups/users',
          method: 'GET',
        };
      },
      transformResponse: (response: any) => {
        if (!response || !response.groups || !Array.isArray(response.groups)) {
          return [];
        }
        return response.groups;
      },
      providesTags: ['UnifiedRBACGroup'],
    }),
  }),
});

export const {
  useGetGroupsQuery,
} = groupsApi;
