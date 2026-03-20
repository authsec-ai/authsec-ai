import { baseApi, withSessionData } from "./baseApi";
import type { AdminUsersResponse } from "./admin/usersApi";

/**
 * Dashboard API Types
 * Based on the AuthSec SDK Manager Dashboard endpoints
 */

// ============================================================================
// Session Types
// ============================================================================

export interface SessionData {
  session_id: string;
  user_email: string;
  token_expires_at: string;
  created_at: string;
  last_activity: string;
  is_active: boolean;
  client_id: string;
  org_id: string | null;
  tenant_id: string;
  provider: string | null;
  accessible_tools: string | null; // JSON string array
}

export interface GetSessionsRequest {
  tenant_id: string;
  is_active: boolean;
}

export interface GetSessionsResponse {
  success: boolean;
  tenant_id: string;
  sessions: SessionData[];
  total_count: number;
  timestamp: string;
}

// ============================================================================
// User Types
// ============================================================================

export interface EndUserData {
  client_id: string;
  tenant_id: string;
  name: string;
  email: string;
  provider: string; // 'custom', 'google', 'github', 'microsoft', 'entra_id', 'ad_sync'
  active: boolean;
  mfa_method: string[] | null; // ['totp', 'webauthn']
  created_at: string;
  updated_at: string;
  last_login: string;
}

export interface GetUsersRequest {
  tenant_id: string;
  provider?: string; // Optional filter by provider
  client_id?: string; // Optional filter by client_id
}

export interface GetUsersResponse {
  success: boolean;
  tenant_id: string;
  filters: {
    provider: string | null;
    client_id: string | null;
  };
  users: EndUserData[];
  total_count?: number;
  timestamp?: string;
}

// ============================================================================
// Dashboard Statistics (Computed from responses)
// ============================================================================

export interface DashboardStats {
  activeSessions: number;
  inactiveSessions: number;
  totalEndUsers: number;
  totalAdminUsers: number;
  // Computed metrics
  totalSessions: number;
  activeUserPercentage: number;
  mfaAdoptionRate: number;
}

export interface ProviderDistribution {
  provider: string;
  count: number;
  percentage: number;
}

export interface MFADistribution {
  method: string;
  count: number;
  percentage: number;
}

// ============================================================================
// Quick Actions Status Types
// ============================================================================

export interface QuickActionsStatus {
  adSync?: {
    configured: boolean;
    type: "AD" | "Entra" | null;
    status: "connected" | "syncing" | "error" | "not_configured";
  };
  authMethods?: {
    count: number;
    providers: string[];
  };
  externalServices?: {
    count: number;
    services: string[];
  };
  logging?: {
    configured: boolean;
    status: "active" | "inactive";
  };
}

// ============================================================================
// Dashboard API Endpoints
// ============================================================================

export const dashboardApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // COMMENTED OUT: Dashboard API endpoints
    // Get active sessions
    // getActiveSessions: builder.query<GetSessionsResponse, GetSessionsRequest>({
    //   query: (data) => {
    //     console.log("[DASHBOARD API] 🔵 getActiveSessions called", {
    //       tenant_id: data.tenant_id,
    //       timestamp: new Date().toISOString(),
    //       stack: new Error().stack
    //     });
    //     return {
    //       url: "/sdkmgr/dashboard/sessions",
    //       method: "POST",
    //       body: {
    //         tenant_id: data.tenant_id,
    //         is_active: true,
    //       },
    //     };
    //   },
    //   providesTags: ["DashboardUserStats"],
    // }),

    // Get inactive sessions
    // getInactiveSessions: builder.query<GetSessionsResponse, GetSessionsRequest>({
    //   query: (data) => {
    //     console.log("[DASHBOARD API] 🔵 getInactiveSessions called", {
    //       tenant_id: data.tenant_id,
    //       timestamp: new Date().toISOString(),
    //       stack: new Error().stack
    //     });
    //     return {
    //       url: "/sdkmgr/dashboard/sessions",
    //       method: "POST",
    //       body: {
    //         tenant_id: data.tenant_id,
    //         is_active: false,
    //       },
    //     };
    //   },
    //   providesTags: ["DashboardUserStats"],
    // }),

    // Get all end users (with optional filters) - Dashboard specific
    // getDashboardEndUsers: builder.query<GetUsersResponse, GetUsersRequest>({
    //   query: (data) => {
    //     console.log("[DASHBOARD API] 🔴 getDashboardEndUsers called - THIS SHOULD ONLY BE CALLED FROM DASHBOARD PAGE!", {
    //       tenant_id: data.tenant_id,
    //       provider: data.provider,
    //       client_id: data.client_id,
    //       timestamp: new Date().toISOString(),
    //       currentPath: window.location.pathname,
    //       stack: new Error().stack
    //     });

    //     const body: any = {
    //       tenant_id: data.tenant_id,
    //     };

    //     // Add optional filters
    //     if (data.provider) {
    //       body.provider = data.provider;
    //     }
    //     if (data.client_id) {
    //       body.client_id = data.client_id;
    //     }

    //     return {
    //       url: "/sdkmgr/dashboard/users",
    //       method: "POST",
    //       body,
    //     };
    //   },
    //   providesTags: ["DashboardEndUser", "DashboardUserStats"],
    // }),

    // Combined dashboard data query (fetches all stats at once)
    // getDashboardStats: builder.query<DashboardStats, { tenant_id: string }>({
    //   async queryFn(arg, _queryApi, _extraOptions, fetchWithBQ) {
    //     console.log("[DASHBOARD API] 🔵 getDashboardStats called", {
    //       tenant_id: arg.tenant_id,
    //       timestamp: new Date().toISOString(),
    //       currentPath: window.location.pathname,
    //       stack: new Error().stack
    //     });

    //     try {
    //       // Fetch all data in parallel
    //       const [activeSessions, inactiveSessions, endUsers, adminUsers] = await Promise.all([
    //         fetchWithBQ({
    //           url: "/sdkmgr/dashboard/sessions",
    //           method: "POST",
    //           body: { tenant_id: arg.tenant_id, is_active: true },
    //         }),
    //         fetchWithBQ({
    //           url: "/sdkmgr/dashboard/sessions",
    //           method: "POST",
    //           body: { tenant_id: arg.tenant_id, is_active: false },
    //         }),
    //         fetchWithBQ({
    //           url: "/sdkmgr/dashboard/users",
    //           method: "POST",
    //           body: { tenant_id: arg.tenant_id },
    //         }),
    //         fetchWithBQ({
    //           url: "uflow/admin/users/list",
    //           method: "POST",
    //           body: withSessionData({
    //             tenant_id: arg.tenant_id,
    //             page: 1,
    //             limit: 1,
    //           }),
    //         }),
    //       ]);

    //       // Check for errors
    //       if (activeSessions.error) return { error: activeSessions.error as any };
    //       if (inactiveSessions.error) return { error: inactiveSessions.error as any };
    //       if (endUsers.error) return { error: endUsers.error as any };
    //       if (adminUsers.error) return { error: adminUsers.error as any };

    //       const activeSessionsData = activeSessions.data as GetSessionsResponse;
    //       const inactiveSessionsData = inactiveSessions.data as GetSessionsResponse;
    //       const endUsersData = endUsers.data as GetUsersResponse;
    //       const adminUsersData = adminUsers.data as AdminUsersResponse;
    //       const totalAdminUsers =
    //         typeof adminUsersData?.total === "number"
    //           ? adminUsersData.total
    //           : Array.isArray(adminUsersData?.users)
    //             ? adminUsersData.users.length
    //             : 0;

    //       // Calculate stats
    //       const totalUsers = endUsersData.users.length;
    //       const usersWithMFA = endUsersData.users.filter(
    //         (user) => user.mfa_method && user.mfa_method.length > 0
    //       ).length;

    //       const stats: DashboardStats = {
    //         activeSessions: activeSessionsData.total_count,
    //         inactiveSessions: inactiveSessionsData.total_count,
    //         totalEndUsers: totalUsers,
    //         totalAdminUsers,
    //         totalSessions: activeSessionsData.total_count + inactiveSessionsData.total_count,
    //         activeUserPercentage:
    //           totalUsers > 0 ? (activeSessionsData.total_count / totalUsers) * 100 : 0,
    //         mfaAdoptionRate: totalUsers > 0 ? (usersWithMFA / totalUsers) * 100 : 0,
    //       };

    //       return { data: stats };
    //     } catch (error: any) {
    //       return { error: { status: "CUSTOM_ERROR", error: error.message } };
    //     }
    //   },
    //   providesTags: ["DashboardStats", "DashboardUserStats", "DashboardUserAnalytics"],
    // }),

    // Get Quick Actions Status (aggregated from various sources)
    getQuickActionsStatus: builder.query<QuickActionsStatus, { tenant_id: string }>({
      async queryFn(arg, _queryApi, _extraOptions, fetchWithBQ) {
        try {
          // For now, return placeholder data
          // TODO: Replace with real API calls when backend endpoints are available
          const status: QuickActionsStatus = {
            adSync: {
              configured: false,
              type: null,
              status: "not_configured",
            },
            authMethods: {
              count: 0,
              providers: [],
            },
            externalServices: {
              count: 0,
              services: [],
            },
            logging: {
              configured: true,
              status: "active",
            },
          };

          // In the future, fetch real data:
          // - AD sync status from users API
          // - Auth methods from authentication API
          // - External services from external services API
          // - Logging status from logs API

          return { data: status };
        } catch (error: any) {
          return { error: { status: "CUSTOM_ERROR", error: error.message } };
        }
      },
      providesTags: ["Dashboard"],
    }),
  }),
});

// COMMENTED OUT: Dashboard API hooks
export const {
  // useGetActiveSessionsQuery,
  // useGetInactiveSessionsQuery,
  // useGetDashboardEndUsersQuery,
  // useGetDashboardStatsQuery,
  useGetQuickActionsStatusQuery,
} = dashboardApi;
