import { useState, useEffect, useCallback, useRef } from "react";
import type {
  ClientWithAuthMethods,
  ClientsFilters,
  ClientsPagination,
  PaginatedClientsResponse,
} from "@/types/entities";


import { ClientsService } from "../services/clients";

export interface UseClientsState {
  clients: ClientWithAuthMethods[];
  loading: boolean;
  error: string | null;
  pagination: {
    currentPage: number;
    totalPages: number;
    pageSize: number;
    total: number;
    hasMore: boolean;
  };
  filters: ClientsFilters;
  stats: {
    total: number;
    byStatus: Record<string, number>;
    byType: Record<string, number>;
    byAuthType: Record<string, number>;
    security: {
      mfaEnabled: number;
      highAuthSuccess: number;
    };
  } | null;
}

export interface UseClientsActions {
  loadClients: () => Promise<void>;
  loadMore: () => Promise<void>;
  setPage: (page: number) => void;
  setPageSize: (pageSize: number) => void;
  setFilters: (filters: Partial<ClientsFilters>) => void;
  clearFilters: () => void;
  setSorting: (sortBy: string, sortOrder: "asc" | "desc") => void;
  searchClients: (searchTerm: string) => void;
  refreshClients: () => Promise<void>;
  toggleClientStatus: (clientId: string) => Promise<void>;
  loadStats: () => Promise<void>;
  // New bulk operations
  bulkUpdateStatus: (clientIds: string[], status: "active" | "restricted" | "disabled") => Promise<void>;
  bulkAssignRoles: (clientIds: string[], roles: string[], mode?: "add" | "replace") => Promise<void>;
  bulkConfigureMFA: (clientIds: string[], mfaConfig: any) => Promise<void>;
  bulkApplyPolicies: (clientIds: string[], policyIds: string[], mode?: "add" | "replace") => Promise<void>;
}

export function useClients(
  workspaceId: string,
  initialFilters: ClientsFilters = {},
  initialPagination: ClientsPagination = {}
): UseClientsState & UseClientsActions {
  const [state, setState] = useState<UseClientsState>({
    clients: [],
    loading: false,
    error: null,
    pagination: {
      currentPage: initialPagination.page || 1,
      totalPages: 0,
      pageSize: initialPagination.pageSize || 10,
      total: 0,
      hasMore: false,
    },
    filters: {
      workspace_id: workspaceId,
      ...initialFilters,
    },
    stats: null,
  });

  // Use refs to avoid stale closures
  const stateRef = useRef(state);
  stateRef.current = state;

  const [sortConfig, setSortConfig] = useState<{
    sortBy: string;
    sortOrder: "asc" | "desc";
  }>({
    sortBy: initialPagination.sortBy || "created_at",
    sortOrder: initialPagination.sortOrder || "desc",
  });

  const loadClients = useCallback(
    async (replace = true) => {
      setState((prev) => ({ ...prev, loading: true, error: null }));

      try {
        // Use current state from ref to avoid stale closures
        const currentState = stateRef.current;
        const paginationParams: ClientsPagination = {
          page: currentState.pagination.currentPage,
          pageSize: currentState.pagination.pageSize,
          sortBy: sortConfig.sortBy as any,
          sortOrder: sortConfig.sortOrder,
        };

        const response: PaginatedClientsResponse = await ClientsService.getClients(
          currentState.filters,
          paginationParams
        );

        console.log("useClients response:", response.data.length, "clients loaded");
        console.log("First client auth methods:", response.data[0]?.attachedMethods);

        const clients = response.data || [];
        const stats = calculateStats(clients);

        setState((prev) => ({
          ...prev,
          clients,
          stats,
          pagination: {
            currentPage: response.currentPage,
            totalPages: response.totalPages,
            pageSize: prev.pagination.pageSize,
            total: response.count || 0,
            hasMore: response.hasMore,
          },
          loading: false,
          error: null,
        }));
      } catch (error) {
        setState((prev) => ({
          ...prev,
          loading: false,
          error: error instanceof Error ? error.message : "Failed to load clients",
        }));
      }
    },
    [sortConfig]
  );

  const loadMore = useCallback(async () => {
    if (!state.pagination.hasMore || state.loading) return;

    setState((prev) => ({
      ...prev,
      pagination: {
        ...prev.pagination,
        currentPage: prev.pagination.currentPage + 1,
      },
    }));

    // loadClients will be called by the effect when pagination changes
  }, [state.pagination.hasMore, state.loading]);

  const setPage = useCallback((page: number) => {
    setState((prev) => ({
      ...prev,
      pagination: {
        ...prev.pagination,
        currentPage: page,
      },
    }));
  }, []);

  const setPageSize = useCallback((pageSize: number) => {
    setState((prev) => ({
      ...prev,
      pagination: {
        ...prev.pagination,
        pageSize,
        currentPage: 1, // Reset to first page when changing page size
      },
    }));
  }, []);

  const setFilters = useCallback((newFilters: Partial<ClientsFilters>) => {
    setState((prev) => ({
      ...prev,
      filters: {
        ...prev.filters,
        ...newFilters,
      },
      pagination: {
        ...prev.pagination,
        currentPage: 1, // Reset to first page when changing filters
      },
    }));
  }, []);

  const clearFilters = useCallback(() => {
    setState((prev) => ({
      ...prev,
      filters: {
        workspace_id: workspaceId,
      },
      pagination: {
        ...prev.pagination,
        currentPage: 1,
      },
    }));
  }, [workspaceId]);

  const setSorting = useCallback((sortBy: string, sortOrder: "asc" | "desc") => {
    setSortConfig({ sortBy, sortOrder });
    setState((prev) => ({
      ...prev,
      pagination: {
        ...prev.pagination,
        currentPage: 1, // Reset to first page when changing sorting
      },
    }));
  }, []);

  const searchClients = useCallback(
    (searchTerm: string) => {
      setFilters({ search: searchTerm });
    },
    [setFilters]
  );

  const refreshClients = useCallback(async () => {
    setState((prev) => ({
      ...prev,
      pagination: {
        ...prev.pagination,
        currentPage: 1,
      },
    }));
    await loadClients(true);
  }, [loadClients]);

  const toggleClientStatus = useCallback(
    async (clientId: string) => {
      const client = state.clients.find((c) => c.id === clientId);
      if (!client) return;

      try {
        setState((prev) => ({
          ...prev,
          clients: prev.clients.map((c) =>
            c.id === clientId ? { ...c, access_status: c.access_status === "active" ? "disabled" : "active" } : c
          ),
        }));

        await ClientsService.toggleClientStatus(clientId);
      } catch (error) {
        console.error("Failed to toggle client status:", error);
        loadClients();
      }
    },
    [state.clients, loadClients]
  );

  const loadStats = useCallback(async () => {
    try {
      // Use local calculation to ensure byAccessLevel is included
      const stats = calculateStats(state.clients);
      setState((prev) => ({
        ...prev,
        stats,
      }));
    } catch (error) {
      console.error("Failed to load client stats:", error);
    }
  }, [state.clients]);

  // Load clients when key dependencies change
  useEffect(() => {
    loadClients(true);
  }, [state.filters, state.pagination.currentPage, state.pagination.pageSize, sortConfig.sortBy, sortConfig.sortOrder, loadClients]);

  // Load stats only when clients change, not on every render
  useEffect(() => {
    if (state.clients.length > 0) {
      const stats = calculateStats(state.clients);
      setState((prev) => ({
        ...prev,
        stats,
      }));
    }
  }, [state.clients]);

  // New bulk operations
  const bulkUpdateStatus = useCallback(
    async (clientIds: string[], status: "active" | "restricted" | "disabled") => {
      try {
        await ClientsService.bulkUpdateStatus(clientIds, status);
        await refreshClients();
      } catch (error) {
        console.error("Failed to bulk update status:", error);
        throw error;
      }
    },
    [refreshClients]
  );

  const bulkAssignRoles = useCallback(
    async (clientIds: string[], roles: string[], mode: "add" | "replace" = "add") => {
      try {
        await ClientsService.bulkAssignRoles(clientIds, roles, mode);
        await refreshClients();
      } catch (error) {
        console.error("Failed to bulk assign roles:", error);
        throw error;
      }
    },
    [refreshClients]
  );

  const bulkConfigureMFA = useCallback(
    async (clientIds: string[], mfaConfig: any) => {
      try {
        await ClientsService.bulkConfigureMFA(clientIds, mfaConfig);
        await refreshClients();
      } catch (error) {
        console.error("Failed to bulk configure MFA:", error);
        throw error;
      }
    },
    [refreshClients]
  );

  const bulkApplyPolicies = useCallback(
    async (clientIds: string[], policyIds: string[], mode: "add" | "replace" = "add") => {
      try {
        await ClientsService.bulkApplyPolicies(clientIds, policyIds, mode);
        await refreshClients();
      } catch (error) {
        console.error("Failed to bulk apply policies:", error);
        throw error;
      }
    },
    [refreshClients]
  );

  return {
    ...state,
    loadClients: () => loadClients(true),
    loadMore,
    setPage,
    setPageSize,
    setFilters,
    clearFilters,
    setSorting,
    searchClients,
    refreshClients,
    toggleClientStatus,
    loadStats,
    bulkUpdateStatus,
    bulkAssignRoles,
    bulkConfigureMFA,
    bulkApplyPolicies,
  };
}

// Hook for managing a single client
export function useClient(clientId: string) {
  const [client, setClient] = useState<ClientWithAuthMethods | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const loadClient = useCallback(async () => {
    if (!clientId) return;

    setLoading(true);
    setError(null);

    try {
      const clientData = await ClientsService.getClient(clientId);
      setClient(clientData);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load client");
    } finally {
      setLoading(false);
    }
  }, [clientId]);

  useEffect(() => {
    loadClient();
  }, [loadClient]);

  return {
    client,
    loading,
    error,
    refetch: loadClient,
  };
}

const calculateStats = (clients: ClientWithAuthMethods[]) => {
  const stats = {
    total: clients.length,
    byStatus: {
      active: 0,
      disabled: 0,
      restricted: 0,
    },
    byType: {
      mcp_server: 0,
      app: 0,
      api: 0,
      other: 0,
    },
    byAuthType: {
      sso: 0,
      custom: 0,
      saml2: 0,
    },
    security: {
      mfaEnabled: 0,
      highAuthSuccess: 0,
    },
  };

  clients.forEach((client) => {
    // Count by status
    if (client.access_status === "active") stats.byStatus.active++;
    else if (client.access_status === "disabled") stats.byStatus.disabled++;
    else if (client.access_status === "restricted") stats.byStatus.restricted++;

    // Count by type
    if (client.type === "mcp_server") stats.byType.mcp_server++;
    else if (client.type === "app") stats.byType.app++;
    else if (client.type === "api") stats.byType.api++;
    else if (client.type === "other") stats.byType.other++;

    // Count by auth type
    if (client.authentication_type === "sso") stats.byAuthType.sso++;
    else if (client.authentication_type === "custom") stats.byAuthType.custom++;
    else if (client.authentication_type === "saml2") stats.byAuthType.saml2++;

    // Security stats
    if (client.mfa_config?.enabled) stats.security.mfaEnabled++;
    
    const total = (client.successful_authentications || 0) + (client.denied_authentications || 0);
    if (total > 0 && (client.successful_authentications || 0) / total > 0.95) {
      stats.security.highAuthSuccess++;
    }
  });

  return stats;
};
