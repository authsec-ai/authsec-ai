import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { motion } from "framer-motion";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import type { AuthLog } from "../../types/entities";
import { Activity, Settings, X, AlertCircle } from "lucide-react";
import {
  AuthLogsView,
  AuthLogsViewSkeleton,
  AuthLogsFilterCard,
  UserAttributeSelectorModal,
  type UserSelection,
} from "./components/auth-logs";
import { useGetLogsQuery } from "../../app/api/logsApi";
import { SessionManager } from "../../utils/sessionManager";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";

interface AuthLogsFilterParams {
  logType?: AuthLog["logType"] | "all";
  clientType?: AuthLog["clientType"] | "all";
  status?: AuthLog["status"] | "all";
  authMethod?: AuthLog["authMethod"] | "all";
  timeRange?: string;
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

export function AuthLogsPage() {
  const navigate = useNavigate();
  const [filters, setFilters] = useState<AuthLogsFilterParams>({});
  const [page, setPage] = useState(1);
  const pageSize = 50;
  const [isGroupByModalOpen, setIsGroupByModalOpen] = useState(false);
  const [userSelection, setUserSelection] = useState<UserSelection | null>(
    null
  );

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["auth-logs-intro"],
  });

  // Get tenant_id from session
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  // Get time range timestamps
  const { start_time, end_time } = getTimeRangeTimestamps(filters.timeRange);

  // Fetch logs from API with server-side filters
  const { data, isLoading, isFetching, isError, error, refetch } =
    useGetLogsQuery(
      {
        tenant_id: tenantId || "",
        page,
        page_size: pageSize,
        username:
          userSelection?.type === "username" &&
          userSelection.values.length === 1
            ? userSelection.values[0]
            : undefined,
        user_id:
          userSelection?.type === "userId" && userSelection.values.length === 1
            ? userSelection.values[0]
            : undefined,
        status:
          filters.status && filters.status !== "all"
            ? (filters.status.toUpperCase() as "SUCCESS" | "FAILURE")
            : undefined,
        event_type:
          filters.logType && filters.logType !== "all"
            ? filters.logType
            : undefined,
        start_time,
        end_time,
      },
      {
        skip: !tenantId,
        refetchOnMountOrArgChange: true, // Force refetch when args change
      }
    );

  const apiLogs = data?.logs || [];
  const pagination = data?.pagination;

  // Filter logs based on filter params and user selection (only client-side filters)
  // Note: status, logType, and timeRange are handled server-side
  const filteredLogs = useMemo(() => {
    return apiLogs.filter((log: AuthLog) => {
      // Server-side filters (status, logType/event_type, timeRange via start_time/end_time)
      // are already applied by the API, so we don't filter them here

      const matchesClientType =
        !filters.clientType ||
        filters.clientType === "all" ||
        log.clientType === filters.clientType;
      const matchesAuthMethod =
        !filters.authMethod ||
        filters.authMethod === "all" ||
        log.authMethod === filters.authMethod;

      // User selection filter (multiple users)
      let matchesUserSelection = true;
      if (userSelection && userSelection.values.length > 0) {
        // Single user is handled server-side via username/user_id params
        // Multiple users need client-side filtering
        if (userSelection.values.length > 1) {
          if (userSelection.type === "userId") {
            matchesUserSelection = userSelection.values.includes(
              log.userId || ""
            );
          } else if (userSelection.type === "username") {
            // Match against username OR email (fallback)
            const userIdentifier = log.username || log.email || "";
            matchesUserSelection =
              userSelection.values.includes(userIdentifier);
          }
        }
      }

      return matchesClientType && matchesAuthMethod && matchesUserSelection;
    });
  }, [apiLogs, filters, userSelection]);

  const handleExport = () => {
    const logText = filteredLogs
      .map((log) => JSON.stringify(log, null, 2))
      .join("\n\n");
    const blob = new Blob([logText], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `auth-logs-${new Date().toISOString()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  };

  const handleFiltersChange = (newFilters: AuthLogsFilterParams) => {
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
              <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-emerald-100 dark:bg-emerald-900/30">
                <Activity className="h-6 w-6 text-emerald-600 dark:text-emerald-400" />
              </div>
              <div>
                <h1 className="text-2xl font-semibold tracking-tight">
                  Authentication & Authorization Logs
                </h1>
                <p className="text-sm text-muted-foreground mt-0.5">
                  Monitor authentication attempts and authorization decisions
                </p>
              </div>
            </div>
            <Button
              onClick={() => navigate("/logs/configure")}
              className="gap-2"
              data-tour-id="logs-configure"
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
          <div className="flex items-center gap-2 rounded-lg border bg-emerald-50/50 dark:bg-emerald-950/20 px-4 py-3">
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
        <div data-tour-id="auth-logs-filters">
          <AuthLogsFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
            onGroupByClick={() => setIsGroupByModalOpen(true)}
          />
        </div>

        {/* Logs View */}
        <motion.div
          data-tour-id="auth-logs-table"
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.1, ease: "easeOut" }}
        >
          {isLoading ? (
            <AuthLogsViewSkeleton />
          ) : (
            <AuthLogsView
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
