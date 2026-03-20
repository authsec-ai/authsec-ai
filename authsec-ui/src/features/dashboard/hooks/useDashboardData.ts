import {
  // useGetDashboardStatsQuery, // COMMENTED OUT
  useGetQuickActionsStatusQuery,
  type QuickActionsStatus,
} from "../../../app/api/dashboardApi";

export interface UseDashboardDataProps {
  tenantId: string;
}

export interface DashboardData {
  // Loading states
  isLoading: boolean;
  isError: boolean;
  error: any;

  // Stats
  stats: {
    activeSessions: number;
    inactiveSessions: number;
    totalSessions: number;
    totalEndUsers: number;
    totalAdminUsers: number;
    activeUserPercentage: number;
    mfaAdoptionRate: number;
  } | null;

  // Quick Actions Status
  quickActionsStatus: QuickActionsStatus | null;

  // Refetch functions
  refetch: () => void;
}

/**
 * Custom hook to fetch and process all dashboard data
 */
export function useDashboardData({ tenantId }: UseDashboardDataProps): DashboardData {
  // COMMENTED OUT: Dashboard stats API
  // const {
  //   data: statsData,
  //   isLoading: statsLoading,
  //   error: statsError,
  //   refetch: refetchStats,
  // } = useGetDashboardStatsQuery({ tenant_id: tenantId }, { skip: !tenantId });

  const {
    data: quickActionsData,
    isLoading: quickActionsLoading,
    error: quickActionsError,
    refetch: refetchQuickActions,
  } = useGetQuickActionsStatusQuery({ tenant_id: tenantId }, { skip: !tenantId });

  const refetchAll = () => {
    // refetchStats(); // COMMENTED OUT
    refetchQuickActions();
  };

  // COMMENTED OUT: Return mock/empty stats data
  const statsData = null;
  const statsLoading = false;
  const statsError = null;

  return {
    isLoading: statsLoading || quickActionsLoading,
    isError: !!statsError || !!quickActionsError,
    error: statsError || quickActionsError,
    stats: statsData || null,
    quickActionsStatus: quickActionsData || null,
    refetch: refetchAll,
  };
}
