import { useState, useMemo, useCallback, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { CardContent } from "../../components/ui/card";
import { Badge } from "../../components/ui/badge";
import { PageHeader } from "@/components/layout/PageHeader";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "../../components/ui/tabs";
import {
  ServerCog,
  Trash2,
  Download,
  Container,
  Box,
  Server,
  Cpu,
  RefreshCw,
  Plus,
} from "lucide-react";
import { WorkloadsFilterCard } from "./components/WorkloadsFilterCard";
import { WorkloadsTable } from "./components/WorkloadsTable";
import { DeleteConfirmDialog } from "./components/DeleteConfirmDialog";
import { TableCard } from "@/theme/components/cards";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { toast } from "react-hot-toast";
import {
  useListWorkloadsQuery,
  useDeleteWorkloadMutation,
  useListAgentsQuery,
  type WorkloadRecord,
  type AgentRecord,
} from "../../app/api/workloadsApi";
import { SessionManager } from "../../utils/sessionManager";
import type {
  DisplayWorkload,
  WorkloadsTableActions,
} from "./utils/workloads-table-utils";
import type {
  DisplayAgent,
  AgentsTableActions,
} from "./utils/workloads-agent-table-utils";
import {
  createAgentsTableColumns,
  transformAgentRecord,
  AgentExpandedRow,
} from "./utils/workloads-agent-table-utils";
import { AdaptiveTable } from "../../components/ui/adaptive-table";

type WorkloadsFilterState = {
  searchQuery?: string;
  type?: string;
  status?: string;
};

const normalizeSelectors = (
  selectors: WorkloadRecord["selectors"]
): string[] => {
  // Handle K8sSelectors object format (new API format)
  if (selectors && typeof selectors === "object" && !Array.isArray(selectors)) {
    return Object.entries(selectors)
      .filter(([, val]) => typeof val === "string" || typeof val === "number")
      .map(([key, val]) => `${key}:${val}`)
      .filter((item) => item.trim().length > 0);
  }

  // Handle array format (legacy format)
  if (!Array.isArray(selectors)) return [];
  return selectors
    .map((selector) => {
      if (typeof selector === "string") {
        return selector;
      }

      if (selector && typeof selector === "object") {
        const typedSelector = selector as Record<string, unknown>;
        const type = typedSelector.type as string | undefined;
        const value = typedSelector.value as string | undefined;
        const match = typedSelector.match as string | undefined;

        if (type && value) {
          return `${type}:${value}`;
        }

        if (type && match) {
          return `${type}:${match}`;
        }

        const parts = Object.entries(typedSelector)
          .filter(
            ([, val]) => typeof val === "string" || typeof val === "number"
          )
          .map(([key, val]) => `${key}:${val}`);

        if (parts.length > 0) {
          return parts.join("|");
        }

        return JSON.stringify(selector);
      }

      return "";
    })
    .filter((item) => typeof item === "string" && item.trim().length > 0);
};

const normalizeStatus = (status: unknown): string => {
  if (typeof status === "string" && status.trim().length > 0) {
    return status.toLowerCase();
  }
  return "unknown";
};

const parseDateValue = (value: unknown): Date | null => {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value === "number") {
    const millis = value > 1e12 ? value : value * 1000;
    const date = new Date(millis);
    return Number.isNaN(date.getTime()) ? null : date;
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return null;

    const numeric = Number(trimmed);
    if (!Number.isNaN(numeric)) {
      return parseDateValue(numeric);
    }

    const date = new Date(trimmed);
    return Number.isNaN(date.getTime()) ? null : date;
  }

  return null;
};

const formatDate = (value: unknown): string => {
  const date = parseDateValue(value);
  if (!date) {
    return typeof value === "string" && value.trim().length > 0 ? value : "—";
  }
  return date.toLocaleString();
};

const coerceErrorValue = (value: unknown): string | null => {
  if (value === undefined || value === null) return null;
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean")
    return String(value);
  if (typeof value === "object") {
    const nestedMessage = (value as { message?: unknown }).message;
    if (typeof nestedMessage === "string") return nestedMessage;
    try {
      return JSON.stringify(value);
    } catch {
      return null;
    }
  }
  return null;
};

const getErrorMessage = (error: unknown): string => {
  if (error === null || error === undefined) return "Failed to load workloads.";

  if (typeof error === "string") {
    return error;
  }

  if (typeof error === "number" || typeof error === "boolean") {
    return String(error);
  }

  if (typeof error === "object") {
    const err = error as {
      status?: number;
      data?: unknown;
      error?: unknown;
      message?: unknown;
    };

    const directMessage = coerceErrorValue(err.message);
    if (directMessage) return directMessage;

    if (err.data) {
      const dataMessage =
        coerceErrorValue((err.data as { message?: unknown }).message) ||
        coerceErrorValue((err.data as { error?: unknown }).error) ||
        coerceErrorValue((err.data as { detail?: unknown }).detail) ||
        coerceErrorValue(err.data);
      if (dataMessage) return dataMessage;
    }

    const nestedError = coerceErrorValue(err.error);
    if (nestedError) return nestedError;

    if (typeof err.status === "number") {
      if (err.status === 401) {
        return "Unauthorized. Please sign in again to load workloads.";
      }
      return `Request failed (${err.status})`;
    }

    const fallback = coerceErrorValue(err);
    if (fallback) return fallback;
  }

  return "Failed to load workloads.";
};

const mapWorkloadRecord = (
  workload: WorkloadRecord,
  index: number
): DisplayWorkload => {
  const record = workload as Record<string, unknown>;
  const idCandidate =
    (record.id as string | undefined) ??
    (record.entry_id as string | undefined) ??
    (record.registration_id as string | undefined) ??
    (record.registration_entry_id as string | undefined) ??
    (record.spiffe_id as string | undefined) ??
    (record.spiffeId as string | undefined) ??
    (record.spiffe as string | undefined) ??
    `workload-${index + 1}`;

  const spiffeId =
    (record.spiffe_id as string | undefined) ??
    (record.spiffeId as string | undefined) ??
    (record.spiffe as string | undefined) ??
    String(idCandidate);

  const type =
    (record.type as string | undefined) ??
    (record.attestation_type as string | undefined) ??
    (record.platform as string | undefined) ??
    (record.workload_type as string | undefined) ??
    "unknown";

  const selectors = normalizeSelectors(
    record.selectors as WorkloadRecord["selectors"]
  );

  const status = normalizeStatus(
    record.status ??
      record.attestation_status ??
      record.state ??
      record.registration_status ??
      record.result ??
      record.phase
  );

  const createdAt = formatDate(
    record.created_at ??
      record.createdAt ??
      record.issued_at ??
      record.issuedAt ??
      record.updated_at ??
      record.timestamp ??
      record.generated_at ??
      record.not_before
  );

  return {
    id: String(idCandidate),
    spiffeId,
    type: type.toString(),
    selectors,
    status,
    createdAt,
    raw: workload,
  };
};

// Mock data for workloads

export function WorkloadsPage() {
  const navigate = useNavigate();
  const session = SessionManager.getSession();

  const [filters, setFilters] = useState<WorkloadsFilterState>({});
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [workloadToDelete, setWorkloadToDelete] =
    useState<DisplayWorkload | null>(null);
  const [selectedWorkloadIds, setSelectedWorkloadIds] = useState<string[]>([]);
  const [bulkDeleteDialogOpen, setBulkDeleteDialogOpen] = useState(false);

  const [deleteWorkload, { isLoading: isDeleting }] =
    useDeleteWorkloadMutation();

  const {
    data: workloadsData,
    isLoading: workloadsLoading,
    isFetching: workloadsFetching,
    error: workloadsError,
    refetch,
  } = useListWorkloadsQuery(undefined, {
    skip: !session?.token,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: agentsData = [],
    isLoading: agentsLoading,
    refetch: refetchAgents,
  } = useListAgentsQuery();

  const workloads = useMemo(
    () => (workloadsData ? workloadsData.workloads.map(mapWorkloadRecord) : []),
    [workloadsData]
  );

  const agents = useMemo(
    () => agentsData.map(transformAgentRecord),
    [agentsData]
  );

  const filteredWorkloads = useMemo(() => {
    if (!workloads.length) return [];

    const searchQuery = filters.searchQuery?.trim().toLowerCase();
    const typeFilter = filters.type?.toLowerCase();
    const statusFilter = filters.status?.toLowerCase();

    return workloads.filter((workload) => {
      const matchesSearch = searchQuery
        ? workload.spiffeId.toLowerCase().includes(searchQuery) ||
          workload.id.toLowerCase().includes(searchQuery) ||
          workload.selectors.some((selector) =>
            selector.toLowerCase().includes(searchQuery)
          )
        : true;

      const matchesType =
        typeFilter && typeFilter !== "all"
          ? workload.type.toLowerCase() === typeFilter
          : true;

      const matchesStatus =
        statusFilter && statusFilter !== "all"
          ? workload.status === statusFilter
          : true;

      return matchesSearch && matchesType && matchesStatus;
    });
  }, [filters, workloads]);

  const isTableLoading = workloadsLoading || workloadsFetching;
  const showInitialSkeleton =
    isTableLoading && filteredWorkloads.length === 0 && !errorMessage;
  const totalWorkloads = workloads.length;

  useEffect(() => {
    if (!workloadsError) {
      setErrorMessage(null);
      return;
    }

    const message = getErrorMessage(workloadsError);
    setErrorMessage(message);
    toast.error(message);
  }, [workloadsError]);

  const handleFiltersChange = useCallback(
    (newFilters: WorkloadsFilterState) => {
      setFilters(newFilters);
    },
    []
  );

  const handleEditClick = (workload: DisplayWorkload) => {
    navigate(`/clients/workloads/edit/${workload.id}`);
  };

  const handleDeleteClick = (workload: DisplayWorkload) => {
    setWorkloadToDelete(workload);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!workloadToDelete) return;

    try {
      const workloadId =
        workloadToDelete.raw.id ||
        workloadToDelete.raw.workload_id ||
        workloadToDelete.id;
      await deleteWorkload({ workload_id: workloadId }).unwrap();
      toast.success("Workload deleted successfully");
      setDeleteDialogOpen(false);
      setWorkloadToDelete(null);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleBulkDelete = async () => {
    if (selectedWorkloadIds.length === 0) return;

    try {
      const deletePromises = selectedWorkloadIds.map((id) => {
        const workload = workloads.find((w) => w.id === id);
        if (!workload) return Promise.resolve();
        const workloadId =
          workload.raw.id || workload.raw.workload_id || workload.id;
        return deleteWorkload({ workload_id: workloadId }).unwrap();
      });

      await Promise.all(deletePromises);
      toast.success(
        `${selectedWorkloadIds.length} workload(s) deleted successfully`
      );
      setSelectedWorkloadIds([]);
      setBulkDeleteDialogOpen(false);
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleBulkExport = () => {
    if (selectedWorkloadIds.length === 0) return;

    const selectedWorkloads = workloads.filter((w) =>
      selectedWorkloadIds.includes(w.id)
    );
    const exportData = selectedWorkloads.map((w) => ({
      id: w.raw.id || w.raw.workload_id,
      spiffe_id: w.raw.spiffe_id || w.raw.spiffeId,
      type: w.raw.type,
      selectors: w.raw.selectors,
      vault_role: w.raw.vault_role,
      status: w.raw.status,
      attestation_type: w.raw.attestation_type,
      created_at: w.raw.created_at,
      updated_at: w.raw.updated_at,
    }));

    const blob = new Blob([JSON.stringify(exportData, null, 2)], {
      type: "application/json",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `workloads-export-${
      new Date().toISOString().split("T")[0]
    }.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success(`${selectedWorkloadIds.length} workload(s) exported`);
  };

  const getStatusBadge = (status: string) => {
    const normalized = status ? status.toLowerCase() : "unknown";
    const label = normalized.charAt(0).toUpperCase() + normalized.slice(1);

    switch (normalized) {
      case "attested":
      case "issued":
      case "active":
        return (
          <Badge className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200">
            {label}
          </Badge>
        );
      case "pending":
      case "registering":
        return (
          <Badge className="bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200">
            {label}
          </Badge>
        );
      case "expired":
      case "revoked":
        return (
          <Badge className="bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200">
            {label}
          </Badge>
        );
      default:
        return <Badge variant="outline">{label}</Badge>;
    }
  };

  const getTypeBadge = (type: string) => {
    const normalized = type ? type.toLowerCase() : "unknown";
    const icons: Record<string, any> = {
      kubernetes: Container,
      k8s: Container,
      docker: Box,
      unix: Server,
      vm: Cpu,
      application: Box,
      service: Server,
    };
    const Icon = icons[normalized] || Server;
    const labelSource =
      normalized === "unknown" ? "Unknown" : normalized.replace(/[-_]/g, " ");
    const label = labelSource.charAt(0).toUpperCase() + labelSource.slice(1);
    return (
      <Badge variant="outline" className="flex items-center gap-1 w-fit">
        <Icon className="h-3 w-3" />
        {label}
      </Badge>
    );
  };

  // Table actions
  const actions: WorkloadsTableActions = {
    onEdit: handleEditClick,
    onDelete: handleDeleteClick,
  };

  const agentActions: AgentsTableActions = {
    onRefresh: (agent) => {
      refetchAgents();
      toast.success(`Refreshed status for agent ${agent.nodeId}`);
    },
    onViewDetails: (agent) => {
      // For now, just show a toast. Could expand to show detailed modal
      toast.success(`Viewing details for agent ${agent.nodeId}`);
    },
  };

  return (
    <div className="min-h-screen">
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title="Agent Workloads"
          description={`Manage SPIFFE workloads, identities, and SPIRE agents${
            workloadsData
              ? ` (${workloadsData.count} workloads, ${agents.length} agents)`
              : ""
          }`}
          // actions={
          //   <Button onClick={() => navigate("/clients/workloads/create")}>
          //     <Plus className="mr-2 h-4 w-4" />
          //     Create Workload
          //   </Button>
          // }
        />

        {/* Bulk Actions Bar */}
        {selectedWorkloadIds.length > 0 && (
          <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Badge variant="default" className="bg-blue-600">
                  {selectedWorkloadIds.length} selected
                </Badge>
                <span className="text-sm text-foreground">
                  Bulk actions available
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={handleBulkExport}
                  className="gap-2"
                >
                  <Download className="h-4 w-4" />
                  Export Selected
                </Button>
                <Button
                  variant="destructive"
                  size="sm"
                  onClick={() => setBulkDeleteDialogOpen(true)}
                  className="gap-2"
                >
                  <Trash2 className="h-4 w-4" />
                  Delete Selected
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSelectedWorkloadIds([])}
                >
                  Clear Selection
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Tabs */}
        <Tabs defaultValue="workloads" className="space-y-4">
          <TabsList className="grid w-full grid-cols-2">
            <TabsTrigger value="workloads">Workloads</TabsTrigger>
            <TabsTrigger value="agents">Agents</TabsTrigger>
          </TabsList>

          <TabsContent value="workloads" className="space-y-4">
            {/* Filter Card */}
            <WorkloadsFilterCard
              onFiltersChange={handleFiltersChange}
              initialFilters={filters}
            />

            {/* Workloads Table */}
            <div className="workloads-table-container">
              <style>{`
            .workloads-table-container [data-slot="table-container"] {
              border: none !important;
              background: transparent !important;
            }
            .workloads-table-container [data-slot="table-header"] {
              background: transparent !important;
            }
            .workloads-table-container .bg-muted\\/50,
            .workloads-table-container .bg-muted\\/30,
            .workloads-table-container [class*="bg-muted"] {
              background: transparent !important;
            }
            .workloads-table-container .hover\\:bg-muted\\/50:hover,
            .workloads-table-container .hover\\:bg-muted\\/30:hover {
              background: rgba(148, 163, 184, 0.1) !important;
            }
            .workloads-table-container .border,
            .workloads-table-container .border-b {
              border-color: rgba(148, 163, 184, 0.2) !important;
            }
            .workloads-table-container .shadow-xl {
              box-shadow: none !important;
            }
          `}</style>
              <TableCard className="transition-all duration-500">
                <CardContent variant="flush">
                  {errorMessage ? (
                    <div className="p-8 text-center">
                      <div className="flex flex-col items-center space-y-4">
                        <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                          <ServerCog className="h-8 w-8 text-red-600 dark:text-red-400" />
                        </div>
                        <div>
                          <h3 className="text-lg font-semibold text-red-900 dark:text-red-100">
                            Unable to Load Workloads
                          </h3>
                          <p className="text-red-700 dark:text-red-300 mt-1">
                            {errorMessage}
                          </p>
                        </div>
                      </div>
                    </div>
                  ) : showInitialSkeleton ? (
                    <DataTableSkeleton rows={10} />
                  ) : (
                    <div className="relative">
                      <WorkloadsTable
                        workloads={filteredWorkloads}
                        selectedWorkloadIds={selectedWorkloadIds}
                        onSelectionChange={setSelectedWorkloadIds}
                        onSelectAll={() => {
                          if (
                            selectedWorkloadIds.length ===
                            filteredWorkloads.length
                          ) {
                            setSelectedWorkloadIds([]);
                          } else {
                            setSelectedWorkloadIds(
                              filteredWorkloads.map((w) => w.id)
                            );
                          }
                        }}
                        actions={actions}
                      />
                      {isTableLoading && filteredWorkloads.length > 0 && (
                        <div className="absolute inset-0 bg-white/50 dark:bg-neutral-900/50 backdrop-blur-sm flex items-center justify-center">
                          <div className="flex items-center space-x-2">
                            <RefreshCw className="h-4 w-4 animate-spin" />
                            <span className="text-sm font-medium">
                              Refreshing...
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </CardContent>
              </TableCard>
            </div>
          </TabsContent>

          <TabsContent value="agents" className="space-y-4">
            {/* Agents Table */}
            <div className="agents-table-container">
              <TableCard>
                <CardContent className="p-6">
                  <div className="space-y-4">
                    <div className="flex items-center justify-between">
                      <h2 className="text-lg font-semibold">SPIRE Agents</h2>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => refetchAgents()}
                        disabled={agentsLoading}
                      >
                        <RefreshCw
                          className={`h-4 w-4 mr-2 ${
                            agentsLoading ? "animate-spin" : ""
                          }`}
                        />
                        Refresh
                      </Button>
                    </div>

                    {/* Agents Table */}
                    <div className="border rounded-lg overflow-hidden">
                      <AdaptiveTable
                        data={agents}
                        columns={createAgentsTableColumns(agentActions)}
                        loading={agentsLoading}
                        emptyMessage="No agents found"
                        expandable={{
                          renderExpandedRow: (row) => (
                            <AgentExpandedRow
                              agent={row.original}
                              actions={agentActions}
                            />
                          ),
                        }}
                      />
                    </div>

                    {/* Agent count */}
                    {!agentsLoading && agents.length > 0 && (
                      <p className="text-sm text-foreground text-center">
                        Showing {agents.length} agent
                        {agents.length !== 1 ? "s" : ""}
                      </p>
                    )}
                  </div>
                </CardContent>
              </TableCard>
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onConfirm={handleDeleteConfirm}
        workloadId={workloadToDelete?.id || ""}
        workloadName={workloadToDelete?.spiffeId}
        isLoading={isDeleting}
      />

      {/* Bulk Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={bulkDeleteDialogOpen}
        onOpenChange={setBulkDeleteDialogOpen}
        onConfirm={handleBulkDelete}
        workloadId={`${selectedWorkloadIds.length} workloads`}
        workloadName="selected workloads"
        isLoading={isDeleting}
      />
    </div>
  );
}
