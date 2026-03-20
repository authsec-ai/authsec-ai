/**
 * END-USER GROUPS API
 *
 * Endpoints for end-users to manage their group memberships
 * Authentication: Requires AuthMiddleware (user ID and tenant ID extracted from JWT)
 * Base Path: /uflow/user/groups
 *
 * Documentation Reference: rbac-api-documentation.json -> end_user_flow.groups
 *
 * Available Endpoints:
 * - POST /uflow/user/groups/users/add         - Add user to groups
 * - POST /uflow/user/groups/users/remove      - Remove user from groups
 * - GET  /uflow/user/groups/users             - Get authenticated user's groups (JWT-based)
 * - GET  /uflow/user/groups/:tenant_id/:group_id/users - Get all users in a group
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface UserGroup {
  id: string;
  tenant_id: string;
  name: string;
  description: string;
  created_at: string;
  updated_at: string;
}

export interface GroupUser {
  id: string;
  email: string;
  username: string;
  first_name: string;
  last_name: string;
  created_at: string;
}

export interface AddUserToGroupsRequest {
  tenant_id: string;
  user_id: string;
  groups: string[];
}

export interface RemoveUserFromGroupsRequest {
  tenant_id: string;
  user_id: string;
  groups: string[];
}

export interface ApiResponse {
  message: string;
}

export interface GetUserGroupsResponse {
  groups: UserGroup[];
}

export interface GetGroupUsersResponse {
  users: GroupUser[];
}

// ============================================================================
// API
// ============================================================================

export const endUserGroupsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/user/groups/users/add
    // Add a user to multiple groups
    addUserToGroups: builder.mutation<ApiResponse, AddUserToGroupsRequest>({
      query: (data) => ({
        url: 'authsec/uflow/user/groups/users/add',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['EndUserRBACGroup', 'EndUser'],
    }),

    // POST /uflow/user/groups/users/remove
    // Remove a user from multiple groups
    removeUserFromGroups: builder.mutation<ApiResponse, RemoveUserFromGroupsRequest>({
      query: (data) => ({
        url: 'authsec/uflow/user/groups/users/remove',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['EndUserRBACGroup', 'EndUser'],
    }),

    // GET /uflow/user/groups/users
    // Get authenticated user's groups (JWT-based, no parameters needed)
    getMyGroups: builder.query<UserGroup[], void>({
      query: () => 'authsec/uflow/user/groups/users',
      transformResponse: (response: GetUserGroupsResponse) => response.groups,
      providesTags: ['EndUserRBACGroup', 'EndUser'],
    }),

    // GET /uflow/user/groups/:tenant_id/:group_id/users
    // Get all users in a specific group
    getGroupUsers: builder.query<GroupUser[], { tenant_id: string; group_id: string }>({
      query: ({ tenant_id, group_id }) => `authsec/uflow/user/groups/${tenant_id}/${group_id}/users`,
      transformResponse: (response: GetGroupUsersResponse) => response.users,
      providesTags: ['EndUserRBACGroup', 'EndUser'],
    }),

  }),
});

export const {
  useAddUserToGroupsMutation,
  useRemoveUserFromGroupsMutation,
  useGetMyGroupsQuery,
  useGetGroupUsersQuery,
} = endUserGroupsApi;
