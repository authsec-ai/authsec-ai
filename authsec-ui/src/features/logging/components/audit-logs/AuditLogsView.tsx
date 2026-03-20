import { useState, useRef, useEffect } from "react";
import { Button } from "../../../../components/ui/button";
import { Badge } from "../../../../components/ui/badge";
import {
  Play,
  Pause,
  RotateCcw,
  Download,
  Copy,
  Terminal,
  Info,
  AlertTriangle,
  AlertCircle,
  AlertOctagon,
  Loader2,
  ChevronRight,
  ChevronDown,
  ChevronLeft,
} from "lucide-react";
import type { AuditLog } from "../../../../types/entities";
import type { PaginationMetadata } from "../../../../app/api/logsApi";

interface AuditLogsViewProps {
  logs: AuditLog[];
  onExport: () => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
  pagination?: PaginationMetadata;
  onPageChange?: (page: number) => void;
}

export function AuditLogsView({
  logs,
  onExport,
  onRefresh,
  isRefreshing,
  pagination,
  onPageChange,
}: AuditLogsViewProps) {
  const [isPaused, setIsPaused] = useState(false);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const scrollContainerRef = useRef<HTMLDivElement>(null);

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

  type SeverityStyle = {
    text: string;
    icon: string;
    label: string;
  };

  const severityStyles: Record<AuditLog["severity"], SeverityStyle> = {
    low: {
      text: "text-blue-700 dark:text-blue-300",
      icon: "text-blue-600 dark:text-blue-400",
      label: "LOW",
    },
    medium: {
      text: "text-yellow-700 dark:text-yellow-300",
      icon: "text-yellow-600 dark:text-yellow-400",
      label: "MEDIUM",
    },
    high: {
      text: "text-orange-700 dark:text-orange-300",
      icon: "text-orange-600 dark:text-orange-400",
      label: "HIGH",
    },
    critical: {
      text: "text-red-700 dark:text-red-300",
      icon: "text-red-600 dark:text-red-400",
      label: "CRITICAL",
    },
  };

  const getSeverityStyle = (severity: AuditLog["severity"]): SeverityStyle => {
    return severityStyles[severity];
  };

  const getSeverityIcon = (severity: AuditLog["severity"]) => {
    switch (severity) {
      case "low":
        return Info;
      case "medium":
        return AlertTriangle;
      case "high":
        return AlertCircle;
      case "critical":
        return AlertOctagon;
      default:
        return Info;
    }
  };

  const formatLogEntry = (log: AuditLog) => {
    const timestamp = new Date(log.timestamp).toISOString();
    const severity = getSeverityStyle(log.severity).label.padEnd(8);
    const category = log.category.toUpperCase().padEnd(12);
    const action = log.action.toUpperCase().padEnd(8);

    // Main log line
    const mainLine = `[${timestamp}] ${severity} ${category} actor=${log.actor.email} action=${action} resource=${log.resourceType}/${log.resourceName}`;

    // Details line
    const details: string[] = [];
    details.push(`resource_id=${log.resourceId}`);
    details.push(`ip=${log.ipAddress}`);
    if (log.status) details.push(`status=${log.status}`);
    if (log.rollbackAvailable) details.push(`rollback_available=true`);

    // Reason
    let reasonLine: string | null = null;
    if (log.reason) {
      reasonLine = `    reason="${log.reason}"`;
    }

    return {
      main: mainLine,
      details: `    ${details.join(" ")}`,
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
            <Terminal className="h-4 w-4 text-blue-500 dark:text-blue-300" />
            <span className="text-slate-700 dark:text-blue-100 font-mono text-sm tracking-[0.12em] uppercase">
              Audit Logs Console
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
            className="h-9 px-3 font-mono text-[11px] uppercase tracking-[0.14em] text-slate-600 dark:text-zinc-300 transition-colors duration-150 hover:bg-slate-200 dark:hover:bg-neutral-900/70 hover:text-slate-900 dark:hover:text-blue-100 focus-visible:ring-1 focus-visible:ring-blue-300/40"
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
            className="h-9 px-3 font-mono text-[11px] uppercase tracking-[0.14em] text-slate-600 dark:text-zinc-300 transition-colors duration-150 hover:bg-slate-200 dark:hover:bg-neutral-900/70 hover:text-slate-900 dark:hover:text-blue-100 focus-visible:ring-1 focus-visible:ring-blue-300/40 disabled:opacity-70"
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
            className="h-9 px-3 font-mono text-[11px] uppercase tracking-[0.14em] text-slate-600 dark:text-zinc-300 transition-colors duration-150 hover:bg-slate-200 dark:hover:bg-neutral-900/70 hover:text-slate-900 dark:hover:text-blue-100 focus-visible:ring-1 focus-visible:ring-blue-300/40"
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
                  No audit logs to display
                </p>
                <p className="text-sm text-slate-500 dark:text-zinc-500 mt-2">
                  Audit logs will appear here in real-time
                </p>
              </div>
            ) : (
              <div className="space-y-0 divide-y divide-slate-200 dark:divide-neutral-900/70">
                {logs.map((log) => {
                  const formatted = formatLogEntry(log);
                  const severityStyle = getSeverityStyle(log.severity);
                  const SeverityIcon = getSeverityIcon(log.severity);
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
                          <SeverityIcon
                            className={`h-4 w-4 ${severityStyle.icon}`}
                          />
                        </div>

                        <div className="flex-1 min-w-0 space-y-1">
                          <div
                            className={`text-[13px] leading-relaxed ${severityStyle.text}`}
                          >
                            {formatted.main}
                          </div>

                          <div className="pl-2 text-[12px] text-slate-600 dark:text-zinc-500/80 whitespace-pre-wrap">
                            {formatted.details}
                          </div>

                          {formatted.reason && (
                            <div className="pl-2 text-[12px] text-blue-600 dark:text-blue-400/80 whitespace-pre-wrap">
                              {formatted.reason}
                            </div>
                          )}
                        </div>
                      </div>

                      {/* Expanded content */}
                      {isExpanded && (
                        <div className="ml-16 mt-2 p-4 bg-slate-100/50 dark:bg-neutral-900/30 rounded-lg border border-slate-200 dark:border-neutral-800">
                          <div className="space-y-3 text-[12px] font-mono">
                            {/* Actor Details */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                Actor Details:
                              </div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                {log.actor.userId && (
                                  <div>• User ID: {log.actor.userId}</div>
                                )}
                                {log.actor.username && (
                                  <div>• Username: {log.actor.username}</div>
                                )}
                                {log.actor.email && (
                                  <div>• Email: {log.actor.email}</div>
                                )}
                                {log.actor.role && (
                                  <div>• Role: {log.actor.role}</div>
                                )}
                              </div>
                            </div>

                            {/* System Context */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                System Context:
                              </div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                {log.userAgent && (
                                  <div>• User Agent: {log.userAgent}</div>
                                )}
                                {log.correlationId && (
                                  <div>• Request ID: {log.correlationId}</div>
                                )}
                              </div>
                            </div>

                            {/* Full Changes (if available) */}
                            {log.changes && log.changes.length > 0 && (
                              <div>
                                <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                  Complete Changes:
                                </div>
                                <div className="pl-3 space-y-1">
                                  {log.changes.map((change, idx) => (
                                    <div
                                      key={idx}
                                      className="text-slate-600 dark:text-zinc-400"
                                    >
                                      <div className="font-medium text-slate-700 dark:text-zinc-300">
                                        • {change.field}:
                                      </div>
                                      <div className="pl-4 space-y-0.5">
                                        {change.oldValue !== undefined &&
                                          change.oldValue !== null &&
                                          change.oldValue !== "" && (
                                            <div className="text-rose-600 dark:text-rose-400">
                                              - Old:{" "}
                                              {JSON.stringify(
                                                change.oldValue,
                                                null,
                                                2
                                              )}
                                            </div>
                                          )}
                                        <div className="text-emerald-600 dark:text-emerald-400">
                                          + New:{" "}
                                          {JSON.stringify(
                                            change.newValue,
                                            null,
                                            2
                                          )}
                                        </div>
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            )}

                            {/* Full Metadata */}
                            {log.metadata &&
                              Object.keys(log.metadata).length > 0 && (
                                <div>
                                  <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">
                                    Full Metadata:
                                  </div>
                                  <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                    {Object.entries(log.metadata).map(
                                      ([key, value]) => (
                                        <div key={key}>
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
