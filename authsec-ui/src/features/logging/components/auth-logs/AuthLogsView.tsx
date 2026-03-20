import { useState, useRef, useEffect } from "react";
import { Button } from "../../../../components/ui/button";
import { Badge } from "../../../../components/ui/badge";
import {
  Play,
  Pause,
  Download,
  Copy,
  Terminal,
  CheckCircle,
  XCircle,
  ShieldAlert,
  AlertTriangle,
  RotateCcw,
  Loader2,
  ChevronRight,
  ChevronDown,
  ChevronLeft,
} from "lucide-react";
import type { AuthLog } from "../../../../types/entities";
import type { PaginationMetadata } from "../../../../app/api/logsApi";

interface AuthLogsViewProps {
  logs: AuthLog[];
  onExport: () => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
  pagination?: PaginationMetadata;
  onPageChange?: (page: number) => void;
}

export function AuthLogsView({
  logs,
  onExport,
  onRefresh,
  isRefreshing,
  pagination,
  onPageChange,
}: AuthLogsViewProps) {
  const [isPaused, setIsPaused] = useState(false);
  const scrollContainerRef = useRef<HTMLDivElement>(null);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  // Scroll to top when logs change (e.g., page change)
  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({ top: 0, behavior: "smooth" });
    }
  }, [pagination?.page]);

  const toggleRowExpansion = (logId: string) => {
    setExpandedRows((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(logId)) {
        newSet.delete(logId);
      } else {
        newSet.add(logId);
      }
      return newSet;
    });
  };

  type LevelStyle = {
    text: string;
    icon: string;
  };

  const statusStyles: Record<AuthLog["status"], LevelStyle> = {
    success: {
      text: "text-emerald-700 dark:text-emerald-300",
      icon: "text-emerald-600 dark:text-emerald-400",
    },
    failure: {
      text: "text-rose-700 dark:text-rose-300",
      icon: "text-rose-600 dark:text-rose-400",
    },
    denied: {
      text: "text-amber-700 dark:text-amber-300",
      icon: "text-amber-600 dark:text-amber-400",
    },
    suspicious: {
      text: "text-blue-700 dark:text-blue-300",
      icon: "text-blue-600 dark:text-blue-400",
    },
  };

  const getStatusStyle = (status: AuthLog["status"]): LevelStyle => {
    return statusStyles[status];
  };

  const getStatusIcon = (status: AuthLog["status"]) => {
    switch (status) {
      case "success":
        return CheckCircle;
      case "failure":
        return XCircle;
      case "denied":
        return ShieldAlert;
      case "suspicious":
        return AlertTriangle;
      default:
        return CheckCircle;
    }
  };

  const formatLogEntry = (log: AuthLog) => {
    const timestamp = new Date(log.timestamp).toISOString();
    const status = log.status.toUpperCase().padEnd(10);
    const logType = log.logType.toUpperCase().padEnd(6);

    // Main log line
    const mainLine = `[${timestamp}] ${status} ${logType} user=${
      log.email || log.username || "undefined"
    } client=${log.clientType}/${log.clientName}`;

    // Details line
    const details: string[] = [];
    if (log.userId) details.push(`user_id=${log.userId}`);
    if (log.sessionId) details.push(`session_id=${log.sessionId}`);
    details.push(`method=${log.authMethod}`);
    details.push(`ip=${log.ipAddress}`);
    if (log.location) details.push(`location="${log.location}"`);
    details.push(`mfa=${log.mfaUsed}`);

    // Auth-specific details
    let authDetails: string[] = [];
    if (log.logType === "authz" && log.resource && log.action) {
      authDetails.push(`resource="${log.resource}"`);
      authDetails.push(`action="${log.action}"`);
    }

    // Message line - extract from metadata
    let messageLine: string | null = null;
    if (log.metadata?.original_message) {
      messageLine = `    message="${log.metadata.original_message}"`;
    }

    // Failure/denial reasons
    let reasonLine: string | null = null;
    if (log.failureReason) {
      reasonLine = `    reason="${log.failureReason}"`;
      if (log.metadata.attemptCount) {
        reasonLine += ` attempts=${log.metadata.attemptCount}`;
      }
      if (log.metadata.requiredRole) {
        reasonLine += ` required_role="${log.metadata.requiredRole}" user_role="${log.metadata.userRole}"`;
      }
      if (log.metadata.riskScore !== undefined) {
        reasonLine += ` risk_score=${log.metadata.riskScore.toFixed(2)}`;
      }
    }

    return {
      main: mainLine,
      details: `    ${details.join(" ")}`,
      authDetails:
        authDetails.length > 0 ? `    ${authDetails.join(" ")}` : null,
      message: messageLine,
      reason: reasonLine,
    };
  };

  const clearConsole = () => {
    console.log("Console cleared");
  };

  return (
    <div className="overflow-hidden border border-slate-200 dark:border-neutral-900 bg-white dark:bg-neutral-950 shadow-lg dark:shadow-[0_24px_40px_rgba(5,5,8,0.45)]">
      {/* Console Controls */}
      <div className="flex items-center justify-between px-6 py-4 bg-slate-100 dark:bg-neutral-950/90 border-b border-slate-200 dark:border-neutral-900">
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 px-3 py-1 rounded-full border border-slate-300 dark:border-neutral-800 bg-white dark:bg-neutral-950/80">
            <Terminal className="h-4 w-4 text-amber-500 dark:text-amber-300" />
            <span className="text-slate-700 dark:text-amber-100 font-mono text-sm tracking-[0.12em] uppercase">
              Auth Logs Console
            </span>
          </div>
          <Badge
            variant="outline"
            className="text-slate-600 dark:text-zinc-300 border-slate-300 dark:border-neutral-700 bg-slate-50 dark:bg-neutral-900/80 font-mono text-[11px] tracking-[0.12em] uppercase"
          >
            {logs.length} entries
          </Badge>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => setIsPaused(!isPaused)}
            className="h-9 px-3 font-mono text-[11px] uppercase tracking-[0.14em] text-slate-600 dark:text-zinc-300 transition-colors duration-150 hover:bg-slate-200 dark:hover:bg-neutral-900/70 hover:text-slate-900 dark:hover:text-amber-100 focus-visible:ring-1 focus-visible:ring-amber-300/40"
          >
            {isPaused ? (
              <Play className="h-3 w-3 mr-1" />
            ) : (
              <Pause className="h-3 w-3 mr-1" />
            )}
            {isPaused ? "Resume" : "Pause"}
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={onRefresh ?? clearConsole}
            disabled={isRefreshing}
            className="h-9 px-3 font-mono text-[11px] uppercase tracking-[0.14em] text-slate-600 dark:text-zinc-300 transition-colors duration-150 hover:bg-slate-200 dark:hover:bg-neutral-900/70 hover:text-slate-900 dark:hover:text-amber-100 focus-visible:ring-1 focus-visible:ring-amber-300/40 disabled:opacity-70"
          >
            {isRefreshing ? (
              <Loader2 className="h-3 w-3 mr-1 animate-spin" />
            ) : (
              <RotateCcw className="h-3 w-3 mr-1" />
            )}
            {onRefresh ? "Refresh" : "Clear"}
          </Button>

          <Button
            variant="ghost"
            size="sm"
            onClick={onExport}
            className="h-9 px-3 font-mono text-[11px] uppercase tracking-[0.14em] text-slate-600 dark:text-zinc-300 transition-colors duration-150 hover:bg-slate-200 dark:hover:bg-neutral-900/70 hover:text-slate-900 dark:hover:text-amber-100 focus-visible:ring-1 focus-visible:ring-amber-300/40"
          >
            <Download className="h-3 w-3 mr-1" />
            Export
          </Button>
        </div>
      </div>

      {/* Console Output */}
      <div className="bg-slate-50 dark:bg-neutral-950/70 border-b border-slate-200 dark:border-neutral-900">
        <div
          className="h-[600px] w-full overflow-y-auto overflow-x-hidden scrollbar-thin scrollbar-track-slate-200 dark:scrollbar-track-neutral-900 scrollbar-thumb-slate-400 dark:scrollbar-thumb-zinc-700"
          ref={scrollContainerRef}
        >
          <div className="p-6 font-mono text-sm text-slate-700 dark:text-zinc-300">
            {logs.length === 0 ? (
              <div className="text-slate-500 dark:text-zinc-500 text-center py-16">
                <Terminal className="h-12 w-12 mx-auto mb-4 text-slate-400 dark:text-zinc-600" />
                <p className="text-lg font-semibold tracking-[0.08em] text-slate-600 dark:text-zinc-300">
                  No auth logs to display
                </p>
                <p className="text-sm text-slate-500 dark:text-zinc-500 mt-2">
                  Auth logs will appear here in real-time
                </p>
              </div>
            ) : (
              <div className="space-y-0 divide-y divide-slate-200 dark:divide-neutral-900/70">
                {logs.map((log) => {
                  const formatted = formatLogEntry(log);
                  const statusStyle = getStatusStyle(log.status);
                  const StatusIcon = getStatusIcon(log.status);
                  const isExpanded = expandedRows.has(log.id);

                  return (
                    <div key={log.id} className="py-3">
                      <div
                        className="flex items-start gap-3 cursor-pointer hover:bg-slate-100 dark:hover:bg-neutral-900/50 transition-colors rounded-lg px-2 -mx-2 py-2"
                        onClick={() => toggleRowExpansion(log.id)}
                      >
                        <div className="flex h-7 w-7 items-center justify-center shrink-0 mt-0.5">
                          {isExpanded ? (
                            <ChevronDown className="h-4 w-4 text-slate-500 dark:text-zinc-400" />
                          ) : (
                            <ChevronRight className="h-4 w-4 text-slate-500 dark:text-zinc-400" />
                          )}
                        </div>

                        <div className="flex h-7 w-7 items-center justify-center shrink-0 mt-0.5">
                          <StatusIcon
                            className={`h-4 w-4 ${statusStyle.icon}`}
                          />
                        </div>

                        <div className="flex-1 min-w-0 space-y-1">
                          <div
                            className={`text-[13px] leading-relaxed ${statusStyle.text}`}
                          >
                            {formatted.main}
                          </div>

                          <div className="pl-2 text-[12px] text-slate-600 dark:text-zinc-500/80 whitespace-pre-wrap">
                            {formatted.details}
                          </div>

                          {formatted.authDetails && (
                            <div className="pl-2 text-[12px] text-slate-600 dark:text-zinc-500/80 whitespace-pre-wrap">
                              {formatted.authDetails}
                            </div>
                          )}

                          {formatted.message && (
                            <div className="pl-2 text-[12px] text-blue-600 dark:text-blue-400/80 whitespace-pre-wrap break-all">
                              {formatted.message}
                            </div>
                          )}

                          {formatted.reason && (
                            <div className="pl-2 text-[12px] text-amber-600 dark:text-amber-400/80 whitespace-pre-wrap">
                              {formatted.reason}
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Expanded content */}
                      {isExpanded && (
                        <div className="ml-16 mt-2 p-4 bg-slate-100/50 dark:bg-neutral-900/30 rounded-lg border border-slate-200 dark:border-neutral-800">
                          <div className="space-y-3 text-[12px] font-mono">
                            {/* User Details */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                User Details:
                              </div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                {log.userId && (
                                  <div>• User ID: {log.userId}</div>
                                )}
                                {log.username && (
                                  <div>• Username: {log.username}</div>
                                )}
                                {log.email && <div>• Email: {log.email}</div>}
                              </div>
                            </div>

                            {/* Session & Auth Details */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                Session & Auth:
                              </div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                {log.sessionId && (
                                  <div>• Session ID: {log.sessionId}</div>
                                )}
                                <div>• Auth Method: {log.authMethod}</div>
                                <div>
                                  • MFA Used: {log.mfaUsed ? "Yes" : "No"}
                                </div>
                                {log.deviceInfo && (
                                  <div>• Device: {log.deviceInfo}</div>
                                )}
                              </div>
                            </div>

                            {/* Client & Location */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                Client & Location:
                              </div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                <div>• Client Type: {log.clientType}</div>
                                <div>• Client Name: {log.clientName}</div>
                                <div>• IP Address: {log.ipAddress}</div>
                                {log.location && (
                                  <div>• Location: {log.location}</div>
                                )}
                                {log.userAgent && (
                                  <div>• User Agent: {log.userAgent}</div>
                                )}
                              </div>
                            </div>

                            {/* Authorization Details (if authz) */}
                            {log.logType === "authz" && (
                              <div>
                                <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                  Authorization:
                                </div>
                                <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                  {log.resource && (
                                    <div>• Resource: {log.resource}</div>
                                  )}
                                  {log.action && (
                                    <div>• Action: {log.action}</div>
                                  )}
                                </div>
                              </div>
                            )}

                            {/* Failure/Risk Details */}
                            {(log.failureReason ||
                              log.metadata?.riskScore !== undefined) && (
                              <div>
                                <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                  Risk & Failure Info:
                                </div>
                                <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                  {log.failureReason && (
                                    <div>
                                      • Failure Reason: {log.failureReason}
                                    </div>
                                  )}
                                  {log.metadata?.attemptCount && (
                                    <div>
                                      • Attempt Count:{" "}
                                      {log.metadata.attemptCount}
                                    </div>
                                  )}
                                  {log.metadata?.riskScore !== undefined && (
                                    <div>
                                      • Risk Score:{" "}
                                      {log.metadata.riskScore.toFixed(2)}
                                    </div>
                                  )}
                                  {log.metadata?.requiredRole && (
                                    <>
                                      <div>
                                        • Required Role:{" "}
                                        {log.metadata.requiredRole}
                                      </div>
                                      <div>
                                        • User Role:{" "}
                                        {log.metadata.userRole || "None"}
                                      </div>
                                    </>
                                  )}
                                </div>
                              </div>
                            )}

                            {/* Full Metadata */}
                            {log.metadata &&
                              Object.keys(log.metadata).length > 0 && (
                                <div>
                                  <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                    Full Metadata (
                                    {Object.keys(log.metadata).length} entries):
                                  </div>
                                  <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                    {Object.entries(log.metadata).map(
                                      ([key, value]) => (
                                        <div key={key} className="break-all">
                                          • {key}: {JSON.stringify(value)}
                                        </div>
                                      )
                                    )}
                                  </div>
                                </div>
                              )}
                          </div>
                        </div>
                      )}
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
      </div>

      {/* Console Footer */}
      <div className="flex items-center justify-between px-6 py-4 bg-slate-100 dark:bg-neutral-950/90 text-[11px] uppercase tracking-[0.18em] text-slate-600 dark:text-zinc-500 border-t border-slate-200 dark:border-neutral-800/70">
        <div className="flex items-center gap-6">
          <div className="flex items-center gap-2">
            <div
              className={`w-2 h-2 rounded-full ${
                isPaused
                  ? "bg-amber-500 dark:bg-amber-300"
                  : "bg-emerald-500 dark:bg-emerald-300"
              }`}
            ></div>
            <span>Status: {isPaused ? "PAUSED" : "LIVE"}</span>
          </div>
        </div>

        <div className="flex items-center gap-6">
          {pagination && (
            <div className="flex items-center gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onPageChange?.(pagination.page - 1)}
                disabled={!pagination.has_prev || isRefreshing}
                className="h-7 w-7 p-0 text-slate-600 dark:text-zinc-300 hover:bg-slate-200 dark:hover:bg-neutral-900/70 disabled:opacity-50"
              >
                <ChevronLeft className="h-4 w-4" />
              </Button>
              <span className="rounded-full border border-slate-300 dark:border-neutral-800/60 bg-white dark:bg-neutral-900/70 px-3 py-1">
                Page {pagination.page} of {pagination.total_pages}
              </span>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => onPageChange?.(pagination.page + 1)}
                disabled={!pagination.has_next || isRefreshing}
                className="h-7 w-7 p-0 text-slate-600 dark:text-zinc-300 hover:bg-slate-200 dark:hover:bg-neutral-900/70 disabled:opacity-50"
              >
                <ChevronRight className="h-4 w-4" />
              </Button>
            </div>
          )}
          <span className="rounded-full border border-slate-300 dark:border-neutral-800/60 bg-white dark:bg-neutral-900/70 px-3 py-1">
            {pagination
              ? `${pagination.total_items} total`
              : `${logs.length} entries`}
          </span>
          <span className="rounded-full border border-slate-300 dark:border-neutral-800/60 bg-white dark:bg-neutral-900/70 px-3 py-1">
            Last updated:{" "}
            {logs.length > 0
              ? new Date(logs[0].timestamp).toLocaleTimeString()
              : "Never"}
          </span>
        </div>
      </div>
    </div>
  );
}
