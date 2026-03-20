import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import type { AuditLog } from "../../types/entities";
import { Database, Settings, X, AlertCircle } from "lucide-react";
import {
  M2MLogsView,
  M2MLogsViewSkeleton,
  M2MLogsFilterCard,
  UserAttributeSelectorModal,
  type UserSelection,
} from "./components/m2m-logs";
import { useGetM2MLogsQuery } from "../../app/api/logsApi";
import { SessionManager } from "../../utils/sessionManager";
import { Alert, AlertDescription } from "../../components/ui/alert";

interface M2MLogsFilterParams {
  action?: AuditLog["action"] | "all";
  resourceType?: AuditLog["resourceType"] | "all";
  severity?: AuditLog["severity"] | "all";
  status?: AuditLog["status"] | "all";
  timeRange?: string;
  sort_by?: "ts" | "service" | "event_type" | "operation";
  sort_desc?: boolean;
}

// Helper function to convert timeRange to RFC3339 timestamps
function getTimeRangeTimestamps(timeRange?: string): {
  start_time?: string;
  end_time?: string;
} {
  if (!timeRange || timeRange === "all") return {};

  const now = new Date();
  const end_time = now.toISOString();
  let start_time: string;

  switch (timeRange) {
    case "5m":
      start_time = new Date(now.getTime() - 5 * 60 * 1000).toISOString();
      break;
    case "1h":
      start_time = new Date(now.getTime() - 60 * 60 * 1000).toISOString();
      break;
    case "24h":
      start_time = new Date(now.getTime() - 24 * 60 * 60 * 1000).toISOString();
      break;
    case "7d":
      start_time = new Date(
        now.getTime() - 7 * 24 * 60 * 60 * 1000
      ).toISOString();
      break;
    default:
      return {};
  }

  return { start_time, end_time };
}

export function M2MLogsPage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<M2MLogsFilterParams>({});
  const [page, setPage] = useState(1);
  const pageSize = 50;
  const [isGroupByModalOpen, setIsGroupByModalOpen] = useState(false);
  const [userSelection, setUserSelection] = useState<UserSelection | null>(
    null
  );

  // Get tenant_id from session
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  // Get time range timestamps
  const { start_time, end_time } = getTimeRangeTimestamps(filters.timeRange);

  // Fetch M2M logs from API with server-side filters
  const { data, isLoading, isFetching, isError, error, refetch } =
    useGetM2MLogsQuery(
      {
        tenant_id: tenantId || "",
        page,
        page_size: pageSize,
        sort_by: filters.sort_by,
        sort_desc: filters.sort_desc ?? true,
        operation:
          filters.action && filters.action !== "all"
            ? (filters.action as "create" | "update" | "delete")
            : undefined,
        start_time,
        end_time,
      },
      { skip: !tenantId }
    );

  const m2mLogs = data?.logs || [];
  const pagination = data?.pagination;

  // Filter logs based on filter params and user selection (only client-side filters)
  // Note: action/operation, timeRange (via start_time/end_time), sort_by, sort_desc are handled server-side
  const filteredLogs = useMemo(() => {
    return m2mLogs.filter((log: AuditLog) => {
      // Server-side filters (action via operation, timeRange via start_time/end_time)
      // are already applied by the API, so we don't filter them here

      const matchesResourceType =
        !filters.resourceType ||
        filters.resourceType === "all" ||
        log.resourceType === filters.resourceType;
      const matchesStatus =
        !filters.status ||
        filters.status === "all" ||
        log.status === filters.status;
      const matchesSeverity =
        !filters.severity ||
        filters.severity === "all" ||
        log.severity === filters.severity;

      // User selection filter - using actor fields from AuditLog
      let matchesUserSelection = true;
      if (userSelection && userSelection.values.length > 0) {
        if (userSelection.type === "userId") {
          matchesUserSelection = userSelection.values.includes(
            log.actor.userId || ""
          );
        } else if (userSelection.type === "username") {
          matchesUserSelection = userSelection.values.includes(
            log.actor.username || ""
          );
        }
      }

      return (
        matchesResourceType &&
        matchesStatus &&
        matchesSeverity &&
        matchesUserSelection
      );
    });
  }, [m2mLogs, filters, userSelection]);

  const handleExport = () => {
    const logText = filteredLogs
      .map((log) => JSON.stringify(log, null, 2))
      .join("\n\n");
    const blob = new Blob([logText], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `m2m-logs-${new Date().toISOString()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleFiltersChange = (newFilters: M2MLogsFilterParams) => {
    setFilters(newFilters);
    setPage(1); // Reset to first page when filters change
  };

  const handlePageChange = (newPage: number) => {
    setPage(newPage);
  };

  const handleGroupByApply = (selection: UserSelection) => {
    setUserSelection(selection);
    setPage(1); // Reset to first page when user selection changes
  };

  const handleClearUserSelection = () => {
    setUserSelection(null);
    setPage(1); // Reset to first page when clearing user selection
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      {/* Simple top header */}
      <div className="border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
        <div className="container mx-auto max-w-[1600px] px-6 py-6">
          <div className="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4">
            <div className="flex items-center gap-4">
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-blue-100 dark:bg-blue-900/30">
                <Database className="h-6 w-6 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                  Machine-to-Machine (M2M) Logs
                </h1>
                <p className="text-sm text-muted-foreground mt-0.5">
                  Monitor service-to-service authentication and API access
                </p>
              </div>
            </div>
            <Button
              onClick={() => navigate("/logs/configure")}
              className="gap-2"
            >
              <Settings className="h-4 w-4" />
              Configure Logs
            </Button>
          </div>
        </div>
      </div>

      <div className="container mx-auto max-w-[1600px] px-6 py-6 space-y-6">
        {/* User Selection Badge */}
        {userSelection && userSelection.values.length > 0 && (
          <div className="flex items-center gap-2 rounded-lg border bg-blue-50/50 dark:bg-blue-950/20 px-4 py-3">
            <Badge variant="secondary">
              Filtered by{" "}
              {userSelection.type === "userId" ? "User ID" : "Username"}
            </Badge>
            <span className="text-sm text-muted-foreground">
              {userSelection.values.length}{" "}
              {userSelection.values.length === 1 ? "user" : "users"} selected
            </span>
            <Button
              variant="ghost"
              size="sm"
              onClick={handleClearUserSelection}
              className="ml-auto h-8 px-2"
            >
              <X className="h-4 w-4" />
              Clear
            </Button>
          </div>
        )}

        {/* Error State */}
        {isError && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Failed to load logs. Please try again later.
              {error &&
              "data" in error &&
              typeof error.data === "object" &&
              error.data &&
              "message" in error.data
                ? ` Error: ${error.data.message}`
                : ""}
            </AlertDescription>
          </Alert>
        )}

        {/* Filter Card */}
        <M2MLogsFilterCard
          onFiltersChange={handleFiltersChange}
          initialFilters={filters}
          onGroupByClick={() => setIsGroupByModalOpen(true)}
        />

        {/* Logs View */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.1, ease: "easeOut" }}
        >
          {isLoading ? (
            <M2MLogsViewSkeleton />
          ) : (
            <M2MLogsView
              logs={filteredLogs}
              onExport={handleExport}
              onRefresh={refetch}
              isRefreshing={isFetching}
              pagination={pagination}
              onPageChange={handlePageChange}
            />
          )}
        </motion.div>
      </div>

      {/* Group By Modal */}
      <UserAttributeSelectorModal
        isOpen={isGroupByModalOpen}
        onClose={() => setIsGroupByModalOpen(false)}
        onApply={handleGroupByApply}
      />
    </div>
  );
}
