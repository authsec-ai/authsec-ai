import { baseApi } from "./baseApi";
import type { AuthLog, AuditLog } from "@/types/entities";

// Types for logs API (paginated endpoint)
export interface FetchLogsParams {
  tenant_id: string;
  page?: number;
  page_size?: number;
  sort_by?: "ts" | "log_level" | "event_type" | "status";
  sort_desc?: boolean;
  username?: string; // Filter by actor email
  client_id?: string; // Filter by actor ID (renamed from user_id)
  event_type?: string;
  log_level?: "INFO" | "ERROR" | "WARN" | "DEBUG";
  status?: "SUCCESS" | "FAILURE";
  start_time?: string; // RFC3339 format
  end_time?: string; // RFC3339 format
  // Legacy parameters for backward compatibility
  user_id?: string; // Deprecated: use client_id
  limit?: number; // Deprecated: use page_size
  offset?: number; // Deprecated
  logType?: "authn" | "authz" | "all"; // Deprecated
  clientType?: "mcp_server" | "ai_agent" | "all"; // Deprecated
  authMethod?: string; // Deprecated
  startDate?: string; // Deprecated: use start_time
  endDate?: string; // Deprecated: use end_time
}

// Raw API response format from backend
export interface RawLogEntry {
  timestamp: string;
  log_level?: string;
  tenant_id: string;
  event_type?: string;
  source_ip?: string;
  message?: string;
  action_type?: string;
  metadata?: Record<string, any>;
  security_context?: {
    risk_score?: number;
    trusted_device?: boolean;
    trusted_location?: boolean;
    trusted_network?: boolean;
    anomalous_activity?: boolean;
  };
  processed_at?: string;
  [key: string]: any;
}

export interface FetchLogsResponse {
  data?: AuthLog[];
  logs?: AuthLog[];
  pagination?: PaginationMetadata;
  // Legacy fields for backward compatibility
  total?: number;
  count?: number;
  page?: number;
  limit?: number;
}

export interface LogAnalytics {
  total: number;
  errors: number;
  warnings: number;
  success?: number;
  [key: string]: any;
}

// Unified configuration request for all log services
export interface ConfigureLogServiceRequest {
  host: string; // Domain or IP:port
  tenant_id: string; // From JWT token
  name: "splunk" | "fluentbit" | "elasticsearch" | "syslog";
}

export interface ConfigureLogServiceResponse {
  success: boolean;
  message?: string;
}

// Log configuration status types
export interface LogConfigurationItem {
  id: string;
  tenant_id: string;
  name: string; // e.g. "splunk", "es", "fluentbit", "syslog"
  host: string;
  alias: string;
  created_at: string;
  updated_at: string;
}

export interface GetLogConfigurationStatusResponse {
  configurations: LogConfigurationItem[];
  tenant_id: string;
}

export interface GetLogConfigurationStatusParams {
  tenant_id: string;
}

export interface LogConfigurationStatus {
  splunk: boolean;
  fluentbit: boolean;
  syslog: boolean;
  elasticsearch: boolean;
}

// Deprecated - use ConfigureLogServiceRequest instead
export interface ConfigureFluentbitRequest {
  host: string;
}

// Deprecated - use ConfigureLogServiceResponse instead
export interface ConfigureFluentbitResponse {
  success: boolean;
  message?: string;
}

// Parameters for paginated audit logs API
export interface FetchAuditLogsParams {
  tenant_id: string;
  page?: number;
  page_size?: number;
  sort_by?: "ts" | "service" | "event_type" | "operation";
  sort_desc?: boolean;
  service?: string;
  event_type?: string;
  actor_id?: string;
  object_type?: string;
  operation?: "create" | "update" | "delete";
  start_time?: string; // RFC3339 format
  end_time?: string; // RFC3339 format
}

// Pagination metadata from API response
export interface PaginationMetadata {
  page: number;
  page_size: number;
  total_items: number;
  total_pages: number;
  has_next: boolean;
  has_prev: boolean;
}

// Response format for audit logs API
export interface FetchAuditLogsResponse {
  data?: AuditLog[];
  logs?: AuditLog[];
  pagination?: PaginationMetadata;
  // Legacy fields for backward compatibility
  total?: number;
  count?: number;
  page?: number;
  limit?: number;
  page_size?: number;
}

// Response format for M2M logs API (same as audit logs)
export interface FetchM2MLogsResponse {
  data?: AuditLog[];
  logs?: AuditLog[];
  pagination?: PaginationMetadata;
  // Legacy fields for backward compatibility
  total?: number;
  count?: number;
  page?: number;
  limit?: number;
}

// Raw audit log entry from backend
export interface RawAuditLogEntry {
  ts: string;
  log_type: string;
  tenant: {
    id: string;
  };
  event: {
    id?: string;
    type: string;
    category: string;
  };
  actor: {
    type: string;
    id: string;
    email?: string;
    username?: string;
  };
  object: {
    type: string;
    id: string;
    name?: string;
  };
  action: {
    operation: string;
  };
  changes?: {
    before?: any;
    after?: any;
  };
  result: {
    status: string;
  };
  context: {
    ip: string;
    user_agent: string;
  };
  correlation?: {
    request_id?: string;
  };
  metadata?: Record<string, any>;
}

// Helper function to transform raw audit log to AuditLog format
function transformRawAuditLogToAuditLog(
  rawLog: RawAuditLogEntry,
  index: number
): AuditLog {
  // Map operation to action
  let action: AuditLog["action"] = "updated";
  const operation = rawLog.action?.operation?.toLowerCase() || "";
  if (operation === "create") action = "created";
  else if (operation === "update" || operation === "modify") action = "updated";
  else if (operation === "delete" || operation === "remove") action = "deleted";
  else if (operation === "enable" || operation === "activate")
    action = "enabled";
  else if (operation === "disable" || operation === "deactivate")
    action = "disabled";

  // Map object type to resourceType
  let resourceType: AuditLog["resourceType"] = "config";
  const objectType = rawLog.object?.type?.toLowerCase() || "";
  if (objectType.includes("user")) resourceType = "user";
  else if (objectType.includes("group")) resourceType = "group";
  else if (objectType.includes("role")) resourceType = "role";
  else if (objectType.includes("client") || objectType.includes("workload"))
    resourceType = "client";
  else if (objectType.includes("resource") || objectType.includes("entry"))
    resourceType = "resource";
  else if (objectType.includes("auth")) resourceType = "auth_method";

  // Map status
  let status: AuditLog["status"] = "success";
  const resultStatus = rawLog.result?.status?.toLowerCase() || "";
  if (resultStatus === "success" || resultStatus === "ok") status = "success";
  else if (
    resultStatus === "failed" ||
    resultStatus === "failure" ||
    resultStatus === "error"
  )
    status = "failed";
  else if (resultStatus === "pending") status = "pending";

  // Map category
  let category: AuditLog["category"] = "configuration";
  const eventCategory = rawLog.event?.category?.toLowerCase() || "";
  if (
    eventCategory.includes("identity") ||
    eventCategory.includes("authentication")
  )
    category = "identity";
  else if (
    eventCategory.includes("access") ||
    eventCategory.includes("authorization")
  )
    category = "access";
  else if (eventCategory.includes("security")) category = "security";
  else if (eventCategory.includes("compliance")) category = "compliance";

  // Determine severity based on action and status
  let severity: AuditLog["severity"] = "low";
  if (action === "deleted") severity = "high";
  else if (action === "disabled") severity = "medium";
  else if (action === "created" || action === "enabled") severity = "low";
  else if (action === "updated") severity = "low";
  if (status === "failed") severity = "high";

  // Build changes array
  let changes: AuditLog["changes"] | undefined;
  if (rawLog.changes) {
    changes = [];
    if (rawLog.changes.after) {
      Object.entries(rawLog.changes.after).forEach(([field, newValue]) => {
        const oldValue = rawLog.changes?.before?.[field];
        changes!.push({ field, oldValue, newValue });
      });
    }
  }

  return {
    id:
      rawLog.event?.id ||
      rawLog.correlation?.request_id ||
      `audit-${rawLog.tenant.id}-${index}-${Date.now()}`,
    timestamp: rawLog.ts,
    actor: {
      userId: rawLog.actor?.id || "",
      username: rawLog.actor?.username || rawLog.actor?.id || "",
      email: rawLog.actor?.email || "",
      role: rawLog.actor?.type || "",
    },
    action,
    resourceType,
    resourceId: rawLog.object?.id || "",
    resourceName: rawLog.object?.name || rawLog.object?.id || "",
    changes,
    severity,
    category,
    ipAddress: rawLog.context?.ip || "0.0.0.0",
    userAgent: rawLog.context?.user_agent || "",
    status,
    rollbackAvailable: false,
    metadata: {
      ...rawLog.metadata,
      event_type: rawLog.event?.type,
      log_type: rawLog.log_type,
      correlation_id: rawLog.correlation?.request_id,
    },
  };
}

// Helper function to parse JSON from message field
function parseMessageJson(message?: string): Record<string, any> {
  if (!message) return {};

  try {
    // Message might have prefix like "started\t{...}" - extract JSON part
    const jsonMatch = message.match(/\{[\s\S]*\}/);
    if (jsonMatch) {
      return JSON.parse(jsonMatch[0]);
    }
  } catch (e) {
    // If parsing fails, return empty object
  }

  return {};
}

// Helper function to transform raw log entry to AuthLog format
function transformRawLogToAuthLog(rawLog: RawLogEntry, index: number): AuthLog {
  // Parse the message JSON to extract embedded data
  const messageData = parseMessageJson(rawLog.message);

  // Extract username - priority: actor > messageData > rawLog fields > undefined
  const username =
    rawLog.actor?.email ||
    rawLog.actor?.username ||
    rawLog.actor?.id ||
    messageData.user ||
    messageData.username ||
    messageData.common_name?.trim() ||
    messageData.node_id ||
    (rawLog.actor?.type === "anonymous" ? "anonymous" : undefined);

  // Determine status based on log_level and event_type
  let status: AuthLog["status"] = "success";

  // Priority 1: Check result.status field (from new API format)
  if (rawLog.result?.status) {
    const resultStatus = rawLog.result.status.toUpperCase();
    if (resultStatus === "SUCCESS") {
      status = "success";
    } else if (resultStatus === "FAILURE") {
      status = "failure";
    } else {
      status = "denied"; // Default for other statuses
    }
  }
  // Priority 2: Check log_level and message (fallback for legacy format)
  else if (
    rawLog.log_level?.toLowerCase().includes("error") ||
    rawLog.message?.toLowerCase().includes("failed")
  ) {
    status = "failure";
  } else if (rawLog.log_level?.toLowerCase().includes("warn")) {
    status = "denied";
  } else if (rawLog.security_context?.anomalous_activity) {
    status = "suspicious";
  }

  // Determine log type based on event_type or action_type
  let logType: AuthLog["logType"] = "authn";
  if (
    rawLog.event_type?.toLowerCase().includes("authz") ||
    rawLog.action_type?.toLowerCase().includes("authorization")
  ) {
    logType = "authz";
  }

  // Extract client name - use available data
  const clientName =
    messageData.node_id ||
    messageData.domain ||
    messageData.common_name?.trim() ||
    rawLog.metadata?.client_name ||
    "System";

  // Extract authentication method
  const authMethod =
    messageData.attestation_type ||
    messageData.auth_method ||
    rawLog.metadata?.auth_method ||
    (messageData.common_name ? "certificate" : "password");

  // Determine client type
  let clientType: AuthLog["clientType"] = "mcp_server";
  if (rawLog.metadata?.client_type === "api") {
    clientType = "ai_agent";
  } else if (messageData.attestation_type === "kubernetes") {
    clientType = "mcp_server";
  }

  // Parse timestamp - extract ISO date from string that may contain ANSI codes
  // Format: "2025-12-01T08:18:31.068Z\t[34mINFO[0m\t..."
  let parsedTimestamp = new Date().toISOString();
  try {
    const timestampStr = rawLog.timestamp || rawLog.processed_at || "";
    // Extract ISO timestamp (YYYY-MM-DDTHH:mm:ss.sssZ or YYYY-MM-DDTHH:mm:ssZ)
    const isoMatch = timestampStr.match(
      /(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:\.\d{3})?Z?)/
    );
    if (isoMatch) {
      const isoStr = isoMatch[1];
      // Ensure it ends with Z
      const fullIso = isoStr.endsWith("Z") ? isoStr : `${isoStr}Z`;
      // Validate it's a valid date
      const testDate = new Date(fullIso);
      if (!isNaN(testDate.getTime())) {
        parsedTimestamp = fullIso;
      }
    } else if (rawLog.processed_at) {
      const testDate = new Date(rawLog.processed_at);
      if (!isNaN(testDate.getTime())) {
        parsedTimestamp = rawLog.processed_at;
      }
    }
  } catch (e) {
    // Use current time as fallback
    parsedTimestamp = new Date().toISOString();
  }

  return {
    id: `log-${rawLog.tenant_id}-${index}-${Date.now()}`,
    timestamp: parsedTimestamp,
    logType,
    userId: rawLog.actor?.id || rawLog.metadata?.user_id || messageData.user_id,
    username,
    email: rawLog.actor?.email || rawLog.metadata?.email || messageData.email,
    clientType,
    clientId: rawLog.tenant_id,
    clientName,
    authMethod,
    status,
    ipAddress: rawLog.client?.ip_address || rawLog.source_ip || "0.0.0.0",
    location: rawLog.metadata?.location,
    userAgent:
      rawLog.client?.user_agent ||
      rawLog.metadata?.user_agent ||
      messageData.domain ||
      "System",
    resource: rawLog.metadata?.resource || messageData.domain,
    action: rawLog.action_type,
    mfaUsed: rawLog.metadata?.mfa_used || false,
    sessionId: rawLog.metadata?.session_id,
    failureReason: status === "failure" ? rawLog.message : undefined,
    metadata: {
      ...rawLog.metadata,
      ...messageData, // Include all parsed message data in metadata
      log_level: rawLog.log_level,
      event_type: rawLog.event_type,
      security_context: rawLog.security_context,
      original_message: rawLog.message,
    },
    rawPayload: rawLog as any, // Preserve complete raw API payload for detailed view
  };
}

// API for logging feature
export const logsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Fetch logs by tenant_id and optional filters (paginated)
    getLogs: builder.query<
      { logs: AuthLog[]; pagination?: PaginationMetadata },
      FetchLogsParams
    >({
      query: ({
        tenant_id,
        page,
        page_size,
        sort_by,
        sort_desc,
        username,
        client_id,
        user_id,
        event_type,
        log_level,
        status,
        start_time,
        end_time,
      }) => {
        const params = new URLSearchParams();
        params.append("tenant_id", tenant_id);

        if (page !== undefined) {
          params.append("page", page.toString());
        }
        if (page_size !== undefined) {
          params.append("page_size", page_size.toString());
        }
        if (sort_by) {
          params.append("sort_by", sort_by);
        }
        if (sort_desc !== undefined) {
          params.append("sort_desc", sort_desc.toString());
        }
        if (username) {
          params.append("username", username);
        }
        if (client_id || user_id) {
          params.append("client_id", client_id || user_id || "");
        }
        if (event_type) {
          params.append("event_type", event_type);
        }
        if (log_level) {
          params.append("log_level", log_level);
        }
        if (status) {
          params.append("status", status);
        }
        if (start_time) {
          params.append("start_time", start_time);
        }
        if (end_time) {
          params.append("end_time", end_time);
        }

        return {
          url: `/logs/auth/paginated?${params.toString()}`,
          method: "GET",
        };
      },
      transformResponse: (
        response: FetchLogsResponse | AuthLog[] | RawLogEntry[]
      ) => {
        // Handle different response formats from the backend
        let rawLogs: any[] = [];
        let pagination: PaginationMetadata | undefined;

        if (Array.isArray(response)) {
          rawLogs = response;
        } else {
          rawLogs = response.data || response.logs || [];
          pagination = response.pagination;
        }

        // Check if the logs are already in AuthLog format or need transformation
        if (rawLogs.length === 0) {
          return { logs: [], pagination };
        }

        // If the first log has the expected AuthLog properties, return as-is
        const firstLog = rawLogs[0];
        let transformedLogs: AuthLog[];

        if (
          firstLog &&
          "logType" in firstLog &&
          "clientType" in firstLog &&
          "authMethod" in firstLog
        ) {
          transformedLogs = rawLogs as AuthLog[];
        } else {
          // Otherwise, transform from raw format to AuthLog format
          transformedLogs = rawLogs.map((log, index) =>
            transformRawLogToAuthLog(log as RawLogEntry, index)
          );
        }

        return { logs: transformedLogs, pagination };
      },
      providesTags: ["Log"],
    }),

    // Placeholder - to be implemented when backend endpoint is available
    getLog: builder.query<AuthLog, string>({
      queryFn: () => ({ data: null as any }),
      providesTags: (result, error, id) => [{ type: "Log", id }],
    }),

    // Placeholder - to be implemented when backend endpoint is available
    createLog: builder.mutation<AuthLog, Partial<AuthLog>>({
      queryFn: () => ({ data: null as any }),
      invalidatesTags: ["Log"],
    }),

    // Placeholder - to be implemented when backend endpoint is available
    exportLogs: builder.mutation<
      Blob,
      FetchLogsParams & { format?: "csv" | "json" }
    >({
      queryFn: () => ({ data: new Blob() }),
    }),

    // Placeholder - to be implemented when backend endpoint is available
    getLogAnalytics: builder.query<LogAnalytics, FetchLogsParams>({
      queryFn: () => ({ data: { total: 0, errors: 0, warnings: 0 } }),
      providesTags: ["Log"],
    }),

    // Fetch audit logs
    getAuditLogs: builder.query<
      { logs: AuditLog[]; pagination?: PaginationMetadata },
      FetchAuditLogsParams
    >({
      query: ({
        tenant_id,
        page,
        page_size,
        sort_by,
        sort_desc,
        service,
        event_type,
        actor_id,
        object_type,
        operation,
        start_time,
        end_time,
      }) => {
        const params = new URLSearchParams();
        params.append("tenant_id", tenant_id);

        if (page !== undefined) {
          params.append("page", page.toString());
        }
        if (page_size !== undefined) {
          params.append("page_size", page_size.toString());
        }
        if (sort_by) {
          params.append("sort_by", sort_by);
        }
        if (sort_desc !== undefined) {
          params.append("sort_desc", sort_desc.toString());
        }
        if (service) {
          params.append("service", service);
        }
        if (event_type) {
          params.append("event_type", event_type);
        }
        if (actor_id) {
          params.append("actor_id", actor_id);
        }
        if (object_type) {
          params.append("object_type", object_type);
        }
        if (operation) {
          params.append("operation", operation);
        }
        if (start_time) {
          params.append("start_time", start_time);
        }
        if (end_time) {
          params.append("end_time", end_time);
        }

        return {
          url: `/logs/audit/paginated?${params.toString()}`,
          method: "GET",
        };
      },
      transformResponse: (
        response: FetchAuditLogsResponse | AuditLog[] | RawAuditLogEntry[]
      ) => {
        // Handle different response formats
        let rawLogs: any[] = [];
        let pagination: PaginationMetadata | undefined;

        if (Array.isArray(response)) {
          rawLogs = response;
        } else {
          rawLogs = response.data || response.logs || [];
          pagination = response.pagination;
        }

        if (rawLogs.length === 0) {
          return { logs: [], pagination };
        }

        // Check if the logs are already in AuditLog format or need transformation
        const firstLog = rawLogs[0];
        let transformedLogs: AuditLog[];

        // If it has 'ts' field, it's a raw audit log that needs transformation
        if ("ts" in firstLog && "event" in firstLog && "actor" in firstLog) {
          transformedLogs = rawLogs.map((log, index) =>
            transformRawAuditLogToAuditLog(log as RawAuditLogEntry, index)
          );
        }
        // If it already has 'timestamp' and 'action' fields, it's already in AuditLog format
        else if ("timestamp" in firstLog && "action" in firstLog) {
          transformedLogs = rawLogs as AuditLog[];
        }
        // Fallback: try to transform anyway
        else {
          transformedLogs = rawLogs.map((log, index) =>
            transformRawAuditLogToAuditLog(log as RawAuditLogEntry, index)
          );
        }

        return { logs: transformedLogs, pagination };
      },
      providesTags: ["Log"],
    }),

    // Fetch M2M logs (uses audit endpoint with service=m2m parameter)
    getM2MLogs: builder.query<
      { logs: AuditLog[]; pagination?: PaginationMetadata },
      FetchAuditLogsParams
    >({
      query: ({
        tenant_id,
        page,
        page_size,
        sort_by,
        sort_desc,
        service = "m2m", // Default to m2m service
        event_type,
        actor_id,
        object_type,
        operation,
        start_time,
        end_time,
      }) => {
        const params = new URLSearchParams();
        params.append("tenant_id", tenant_id);
        params.append("service", service); // Always filter by service for M2M

        if (page !== undefined) {
          params.append("page", page.toString());
        }
        if (page_size !== undefined) {
          params.append("page_size", page_size.toString());
        }
        if (sort_by) {
          params.append("sort_by", sort_by);
        }
        if (sort_desc !== undefined) {
          params.append("sort_desc", sort_desc.toString());
        }
        if (event_type) {
          params.append("event_type", event_type);
        }
        if (actor_id) {
          params.append("actor_id", actor_id);
        }
        if (object_type) {
          params.append("object_type", object_type);
        }
        if (operation) {
          params.append("operation", operation);
        }
        if (start_time) {
          params.append("start_time", start_time);
        }
        if (end_time) {
          params.append("end_time", end_time);
        }

        return {
          url: `/logs/audit/paginated?${params.toString()}`,
          method: "GET",
        };
      },
      transformResponse: (
        response: FetchM2MLogsResponse | AuditLog[] | RawAuditLogEntry[]
      ) => {
        // Handle different response formats
        let rawLogs: any[] = [];
        let pagination: PaginationMetadata | undefined;

        if (Array.isArray(response)) {
          rawLogs = response;
        } else {
          rawLogs = response.data || response.logs || [];
          pagination = response.pagination;
        }

        if (rawLogs.length === 0) {
          return { logs: [], pagination };
        }

        // Check if the logs are already in AuditLog format or need transformation
        const firstLog = rawLogs[0];
        let transformedLogs: AuditLog[];

        // If it has 'ts' field, it's a raw audit log that needs transformation
        if ("ts" in firstLog && "event" in firstLog && "actor" in firstLog) {
          transformedLogs = rawLogs.map((log, index) =>
            transformRawAuditLogToAuditLog(log as RawAuditLogEntry, index)
          );
        }
        // If it already has 'timestamp' and 'action' fields, it's already in AuditLog format
        else if ("timestamp" in firstLog && "action" in firstLog) {
          transformedLogs = rawLogs as AuditLog[];
        }
        // Fallback: try to transform anyway
        else {
          transformedLogs = rawLogs.map((log, index) =>
            transformRawAuditLogToAuditLog(log as RawAuditLogEntry, index)
          );
        }

        return { logs: transformedLogs, pagination };
      },
      providesTags: ["Log"],
    }),

    // Fetch the current configuration status for all log forwarding services
    getLogConfigurationStatus: builder.query<
      LogConfigurationStatus,
      GetLogConfigurationStatusParams
    >({
      query: ({ tenant_id }) => ({
        url: `/logs/status?tenant_id=${tenant_id}`,
        method: "GET",
      }),
      transformResponse: (
        response: GetLogConfigurationStatusResponse
      ): LogConfigurationStatus => {
        const names = new Set(
          response.configurations.map((c) => c.name.toLowerCase())
        );
        return {
          splunk: names.has("splunk"),
          fluentbit: names.has("fluentbit"),
          syslog: names.has("syslog"),
          // API returns "es" for Elasticsearch
          elasticsearch: names.has("elasticsearch") || names.has("es"),
        };
      },
      providesTags: ["LogConfigurationStatus"],
    }),

    // Configure any log forwarding service
    configureLogService: builder.mutation<
      ConfigureLogServiceResponse,
      ConfigureLogServiceRequest
    >({
      query: (config) => ({
        url: "/logs/admin/fluent-bit", // Single endpoint for all services
        method: "POST",
        body: config,
      }),
      invalidatesTags: ["LogConfigurationStatus"],
    }),

    // Deprecated - use configureLogService instead
    configureFluentbit: builder.mutation<
      ConfigureFluentbitResponse,
      ConfigureFluentbitRequest
    >({
      query: (config) => ({
        url: "/logs/admin/fluent-bit",
        method: "POST",
        body: config,
      }),
    }),
  }),
});

export const {
  useGetLogsQuery,
  useGetLogQuery,
  useCreateLogMutation,
  useExportLogsMutation,
  useGetLogAnalyticsQuery,
  useGetAuditLogsQuery,
  useGetM2MLogsQuery,
  useGetLogConfigurationStatusQuery,
  useConfigureLogServiceMutation,
  useConfigureFluentbitMutation, // Deprecated - use useConfigureLogServiceMutation
} = logsApi;
