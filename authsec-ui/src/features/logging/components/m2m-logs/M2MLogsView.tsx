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
  AlertTriangle,
  RotateCcw,
  Loader2,
  ChevronRight,
  ChevronDown,
  ChevronLeft,
} from "lucide-react";
import type { AuditLog } from "../../../../types/entities";
import type { PaginationMetadata } from "../../../../app/api/logsApi";

interface M2MLogsViewProps {
  logs: AuditLog[];
  onExport: () => void;
  onRefresh?: () => void;
  isRefreshing?: boolean;
  pagination?: PaginationMetadata;
  onPageChange?: (page: number) => void;
}

export function M2MLogsView({ logs, onExport, onRefresh, isRefreshing, pagination, onPageChange }: M2MLogsViewProps) {
  const [isPaused, setIsPaused] = useState(false);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const scrollContainerRef = useRef<HTMLDivElement>(null);

  // Scroll to top when logs change (e.g., page change)
  useEffect(() => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({ top: 0, behavior: 'smooth' });
    }
  }, [pagination?.page]);

  const toggleRowExpansion = (logId: string) => {
    setExpandedRows(prev => {
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

  const statusStyles: Record<AuditLog["status"], LevelStyle> = {
    success: {
      text: "text-emerald-700 dark:text-emerald-300",
      icon: "text-emerald-600 dark:text-emerald-400",
    },
    failed: {
      text: "text-rose-700 dark:text-rose-300",
      icon: "text-rose-600 dark:text-rose-400",
    },
    pending: {
      text: "text-amber-700 dark:text-amber-300",
      icon: "text-amber-600 dark:text-amber-400",
    },
  };

  const getStatusStyle = (status: AuditLog["status"]): LevelStyle => {
    return statusStyles[status];
  };

  const getStatusIcon = (status: AuditLog["status"]) => {
    switch (status) {
      case "success":
        return CheckCircle;
      case "failed":
        return XCircle;
      case "pending":
        return AlertTriangle;
      default:
        return CheckCircle;
    }
  };

  const formatLogEntry = (log: AuditLog) => {
    const timestamp = new Date(log.timestamp).toISOString();
    const status = log.status.toUpperCase().padEnd(10);
    const action = log.action.toUpperCase().padEnd(10);

    // Main log line - using AuditLog fields
    const mainLine = `[${timestamp}] ${status} ${action} actor=${log.actor.email || log.actor.username || 'undefined'} resource=${log.resourceType}/${log.resourceName}`;

    // Details line
    const details: string[] = [];
    if (log.actor.userId) details.push(`actor_id=${log.actor.userId}`);
    if (log.resourceId) details.push(`resource_id=${log.resourceId}`);
    details.push(`severity=${log.severity}`);
    details.push(`category=${log.category}`);
    details.push(`ip=${log.ipAddress}`);

    // Resource-specific details
    let resourceDetails: string[] = [];
    if (log.changes && log.changes.length > 0) {
      resourceDetails.push(`changes=${log.changes.length}`);
    }
    if (log.rollbackAvailable) {
      resourceDetails.push(`rollback_available=true`);
    }

    // Reason line
    let reasonLine: string | null = null;
    if (log.reason) {
      reasonLine = `    reason="${log.reason}"`;
    }

    // Metadata line
    let metadataLine: string | null = null;
    if (log.metadata && Object.keys(log.metadata).length > 0) {
      const metadataStr = Object.entries(log.metadata)
        .slice(0, 3)
        .map(([key, value]) => `${key}=${JSON.stringify(value)}`)
        .join(" ");
      if (metadataStr) {
        metadataLine = `    metadata: ${metadataStr}`;
      }
    }

    return {
      main: mainLine,
      details: `    ${details.join(" ")}`,
      resourceDetails: resourceDetails.length > 0 ? `    ${resourceDetails.join(" ")}` : null,
      reason: reasonLine,
      metadata: metadataLine,
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
              M2M Logs Console
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
            {isPaused ? <Play className="h-3 w-3 mr-1" /> : <Pause className="h-3 w-3 mr-1" />}
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
                  No M2M logs to display
                </p>
                <p className="text-sm text-slate-500 dark:text-zinc-500 mt-2">M2M logs will appear here in real-time</p>
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
                          <StatusIcon className={`h-4 w-4 ${statusStyle.icon}`} />
                        </div>

                        <div className="flex-1 min-w-0 space-y-1">
                          <div className={`text-[13px] leading-relaxed ${statusStyle.text}`}>
                            {formatted.main}
                          </div>

                          <div className="pl-2 text-[12px] text-slate-600 dark:text-zinc-500/80 whitespace-pre-wrap">
                            {formatted.details}
                          </div>

                          {formatted.resourceDetails && (
                            <div className="pl-2 text-[12px] text-slate-600 dark:text-zinc-500/80 whitespace-pre-wrap">
                              {formatted.resourceDetails}
                            </div>
                          )}

                          {formatted.reason && (
                            <div className="pl-2 text-[12px] text-amber-600 dark:text-amber-400/80 whitespace-pre-wrap">
                              {formatted.reason}
                            </div>
                          )}

                          {formatted.metadata && (
                            <div className="pl-2 text-[12px] text-blue-600 dark:text-blue-400/80 whitespace-pre-wrap break-all">
                              {formatted.metadata}
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
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">Actor Details:</div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                {log.actor.userId && <div>• Actor ID: {log.actor.userId}</div>}
                                {log.actor.username && <div>• Username: {log.actor.username}</div>}
                                {log.actor.email && <div>• Email: {log.actor.email}</div>}
                                {log.actor.role && <div>• Role: {log.actor.role}</div>}
                              </div>
                            </div>

                            {/* Resource Details */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">Resource Details:</div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                <div>• Resource Type: {log.resourceType}</div>
                                <div>• Resource Name: {log.resourceName}</div>
                                {log.resourceId && <div>• Resource ID: {log.resourceId}</div>}
                              </div>
                            </div>

                            {/* System Context */}
                            <div>
                              <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">System Context:</div>
                              <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                {log.userAgent && <div>• User Agent: {log.userAgent}</div>}
                                {log.correlationId && <div>• Request ID: {log.correlationId}</div>}
                                <div>• IP Address: {log.ipAddress}</div>
                              </div>
                            </div>

                            {/* Full Changes (show all details, not just count) */}
                            {log.changes && log.changes.length > 0 && (
                              <div>
                                <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">Complete Changes ({log.changes.length}):</div>
                                <div className="pl-3 space-y-2">
                                  {log.changes.map((change, idx) => (
                                    <div key={idx} className="text-slate-600 dark:text-zinc-400">
                                      <div className="font-medium text-slate-700 dark:text-zinc-300">• {change.field}:</div>
                                      <div className="pl-4 space-y-0.5">
                                        {change.oldValue !== undefined && change.oldValue !== null && change.oldValue !== '' && (
                                          <div className="text-rose-600 dark:text-rose-400">
                                            - Old: {JSON.stringify(change.oldValue, null, 2)}
                                          </div>
                                        )}
                                        <div className="text-emerald-600 dark:text-emerald-400">
                                          + New: {JSON.stringify(change.newValue, null, 2)}
                                        </div>
                                      </div>
                                    </div>
                                  ))}
                                </div>
                              </div>
                            )}

                            {/* Full Metadata (show all, not just first 3) */}
                            {log.metadata && Object.keys(log.metadata).length > 0 && (
                              <div>
                                <div className="font-semibold text-slate-700 dark:text-zinc-300 mb-1.5">Full Metadata ({Object.keys(log.metadata).length} entries):</div>
                                <div className="pl-3 space-y-1 text-slate-600 dark:text-zinc-400">
                                  {Object.entries(log.metadata).map(([key, value]) => (
                                    <div key={key} className="break-all">• {key}: {JSON.stringify(value)}</div>
                                  ))}
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
              className={`w-2 h-2 rounded-full ${isPaused ? "bg-amber-500 dark:bg-amber-300" : "bg-emerald-500 dark:bg-emerald-300"}`}
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
