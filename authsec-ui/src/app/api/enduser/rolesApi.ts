/**
 * END-USER ROLES API
 *
 * Endpoints for end-users to view and request roles
 * Authentication: Requires AuthMiddleware (user ID and tenant ID extracted from JWT)
 * Base Path: /uflow/user/roles
 *
 * Documentation Reference: rbac-api-documentation.json -> end_user_flow.roles
 *
 * Available Endpoints:
 * - GET  /uflow/user/roles            - Get authenticated user's assigned roles
 * - GET  /uflow/user/roles/available  - Get roles user can request
 * - POST /uflow/user/roles/request    - Request role assignment (pending approval)
 * - GET  /uflow/user/roles/requests   - Get user's role requests history
 */

import { baseApi, withSessionData } from '../baseApi';

// ============================================================================
// TYPES
// ============================================================================

export interface UserRole {
  id: string;
  name: string;
  description: string;
  tenant_id: string;
  assigned_at: string;
}

export interface AvailableRole {
  id: string;
  name: string;
  description: string;
  tenant_id: string;
  is_assigned: boolean;
}

export interface RoleRequest {
  id: string;
  role_id: string;
  role_name: string;
  user_id: string;
  status: 'pending' | 'approved' | 'rejected';
  justification: string;
  requested_at: string;
  reviewed_at?: string | null;
  reviewed_by?: string | null;
}

export interface RequestRoleAssignmentRequest {
  role_id: string;
  justification?: string;
}

export interface RequestRoleAssignmentResponse {
  message: string;
  request_id: string;
  status: 'pending';
}

export interface GetMyRolesResponse {
  roles: UserRole[];
}

export interface GetAvailableRolesResponse {
  roles: AvailableRole[];
}

export interface GetMyRoleRequestsResponse {
  requests: RoleRequest[];
}

// ============================================================================
// API
// ============================================================================

export const endUserRolesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // GET /uflow/user/roles
    // Get all roles assigned to the authenticated user (extracted from JWT)
    getMyRoles: builder.query<UserRole[], void>({
      query: () => 'authsec/uflow/user/roles',
      transformResponse: (response: GetMyRolesResponse) => response.roles,
      providesTags: ['EndUserRBACRole', 'EndUser'],
    }),

    // GET /uflow/user/roles/available
    // Get all roles available in the tenant that can be requested
    getAvailableRoles: builder.query<AvailableRole[], void>({
      query: () => 'authsec/uflow/user/roles/available',
      transformResponse: (response: GetAvailableRolesResponse) => response.roles,
      providesTags: ['EndUserRBACRole'],
    }),

    // POST /uflow/user/roles/request
    // Request assignment of a role (creates a pending role request for admin approval)
    requestRoleAssignment: builder.mutation<RequestRoleAssignmentResponse, RequestRoleAssignmentRequest>({
      query: (data) => ({
        url: 'authsec/uflow/user/roles/request',
        method: 'POST',
        body: withSessionData(data),
      }),
      invalidatesTags: ['EndUserRBACRole'],
    }),

    // GET /uflow/user/roles/requests
    // Get all role requests made by the authenticated user
    getMyRoleRequests: builder.query<RoleRequest[], void>({
      query: () => 'authsec/uflow/user/roles/requests',
      transformResponse: (response: GetMyRoleRequestsResponse) => response.requests,
      providesTags: ['EndUserRBACRole'],
    }),

  }),
});

export const {
  useGetMyRolesQuery,
  useGetAvailableRolesQuery,
  useRequestRoleAssignmentMutation,
  useGetMyRoleRequestsQuery,
} = endUserRolesApi;
