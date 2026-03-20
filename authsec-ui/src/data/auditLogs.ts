import type { AuditLog } from "../types/entities";

/**
 * Mock audit log data
 * Represents configuration changes, admin actions, and system modifications
 */

const actors = [
  {
    userId: "user_admin_001",
    username: "admin",
    email: "admin@company.com",
    role: "Super Admin",
  },
  {
    userId: "user_admin_002",
    username: "sarah.admin",
    email: "sarah.admin@company.com",
    role: "Admin",
  },
  {
    userId: "user_manager_001",
    username: "john.manager",
    email: "john.manager@company.com",
    role: "Manager",
  },
  {
    userId: "user_manager_002",
    username: "alice.manager",
    email: "alice.manager@company.com",
    role: "Manager",
  },
];

const actions: Array<AuditLog["action"]> = ["created", "updated", "deleted", "enabled", "disabled"];

const resourceTypes: Array<AuditLog["resourceType"]> = [
  "user",
  "group",
  "role",
  "client",
  "resource",
  "auth_method",
  "config",
];

const severities: Array<AuditLog["severity"]> = ["low", "medium", "high", "critical"];

const categories: Array<AuditLog["category"]> = [
  "identity",
  "access",
  "security",
  "configuration",
  "compliance",
];

const ipAddresses = [
  "10.0.1.50",
  "192.168.1.100",
  "172.16.0.45",
  "10.0.2.88",
];

const userAgents = [
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
  "AuthSec-Admin-Console/1.0.0",
];

function randomItem<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function generateAuditLog(index: number, minutesAgo: number): AuditLog {
  const timestamp = new Date(Date.now() - minutesAgo * 60 * 1000).toISOString();
  const actor = randomItem(actors);

  // 40% updated, 30% created, 20% deleted, 10% enabled/disabled
  let action: AuditLog["action"];
  const rand = Math.random();
  if (rand < 0.40) action = "updated";
  else if (rand < 0.70) action = "created";
  else if (rand < 0.90) action = "deleted";
  else action = Math.random() < 0.5 ? "enabled" : "disabled";

  const resourceType = randomItem(resourceTypes);

  // Severity distribution: 50% low, 30% medium, 15% high, 5% critical
  let severity: AuditLog["severity"];
  const sevRand = Math.random();
  if (sevRand < 0.50) severity = "low";
  else if (sevRand < 0.80) severity = "medium";
  else if (sevRand < 0.95) severity = "high";
  else severity = "critical";

  // Critical severity for config and auth_method changes
  if (resourceType === "config" || resourceType === "auth_method") {
    if (Math.random() < 0.4) severity = "critical";
    else if (Math.random() < 0.7) severity = "high";
  }

  const category = randomItem(categories);
  const status: AuditLog["status"] = Math.random() < 0.95 ? "success" : "failed";

  const log: AuditLog = {
    id: `audit_log_${String(index).padStart(4, "0")}`,
    timestamp,
    actor,
    action,
    resourceType,
    resourceId: `${resourceType}_${Math.random().toString(36).substr(2, 9)}`,
    resourceName: generateResourceName(resourceType, action),
    severity,
    category,
    ipAddress: randomItem(ipAddresses),
    userAgent: randomItem(userAgents),
    status,
    rollbackAvailable: action === "updated" && Math.random() < 0.7,
    metadata: {},
  };

  // Add changes for updated actions
  if (action === "updated") {
    log.changes = generateChanges(resourceType);
    if (resourceType === "config") {
      log.reason = randomItem([
        "Security enhancement",
        "Compliance requirement",
        "Performance optimization",
        "Bug fix",
      ]);
    }
  }

  // Add reason for deletions
  if (action === "deleted") {
    log.reason = randomItem([
      "User offboarded",
      "Resource deprecated",
      "Security violation",
      "Duplicate entry",
    ]);
  }

  return log;
}

function generateResourceName(type: AuditLog["resourceType"], action: AuditLog["action"]): string {
  const names: Record<AuditLog["resourceType"], string[]> = {
    user: [
      "Alice Johnson <alice.johnson@company.com>",
      "Bob Smith <bob.smith@company.com>",
      "Charlie Davis <charlie.davis@company.com>",
      "Diana Wilson <diana.wilson@company.com>",
    ],
    group: ["Engineering", "Marketing", "Sales", "Finance", "Operations", "Developers"],
    role: ["Admin", "Developer", "Viewer", "Editor", "Auditor", "Security Officer"],
    client: [
      "Analytics MCP Server",
      "Payment Gateway",
      "Customer Support Agent",
      "Data Processing Service",
    ],
    resource: ["/api/v1/users", "/api/v1/admin", "/api/v1/payments", "/api/v1/analytics"],
    auth_method: ["OIDC Provider", "SAML 2.0 Configuration", "LDAP Integration", "WebAuthn Settings"],
    config: [
      "Session Timeout Settings",
      "Password Requirements",
      "Rate Limiting Configuration",
      "Audit Log Retention",
    ],
  };

  return randomItem(names[type]);
}

function generateChanges(
  resourceType: AuditLog["resourceType"]
): Array<{ field: string; oldValue: any; newValue: any }> {
  const changeTemplates: Record<
    AuditLog["resourceType"],
    Array<{ field: string; oldValue: any; newValue: any }>
  > = {
    user: [
      { field: "role", oldValue: "developer", newValue: "admin" },
      { field: "status", oldValue: "active", newValue: "inactive" },
      { field: "email", oldValue: "old.email@company.com", newValue: "new.email@company.com" },
    ],
    group: [
      { field: "members", oldValue: ["user_1", "user_2"], newValue: ["user_1", "user_2", "user_3"] },
      { field: "description", oldValue: "Old description", newValue: "New description" },
    ],
    role: [
      { field: "permissions", oldValue: ["read", "write"], newValue: ["read", "write", "delete"] },
      { field: "users", oldValue: 5, newValue: 8 },
    ],
    client: [
      { field: "status", oldValue: "active", newValue: "disabled" },
      { field: "authentication_type", oldValue: "sso", newValue: "saml2" },
    ],
    resource: [
      { field: "access_level", oldValue: "public", newValue: "restricted" },
      { field: "allowed_methods", oldValue: ["GET"], newValue: ["GET", "POST"] },
    ],
    auth_method: [
      { field: "relying_party_id", oldValue: "old.domain.com", newValue: "new.domain.com" },
      { field: "timeout", oldValue: 60000, newValue: 90000 },
      { field: "enabled", oldValue: false, newValue: true },
    ],
    config: [
      { field: "session_timeout", oldValue: 3600, newValue: 7200 },
      { field: "max_login_attempts", oldValue: 3, newValue: 5 },
      { field: "password_min_length", oldValue: 8, newValue: 12 },
    ],
  };

  const templates = changeTemplates[resourceType];
  const numChanges = Math.floor(Math.random() * 2) + 1; // 1-2 changes
  const selectedChanges: Array<{ field: string; oldValue: any; newValue: any }> = [];

  for (let i = 0; i < numChanges; i++) {
    selectedChanges.push(randomItem(templates));
  }

  return selectedChanges;
}

// Generate 200+ audit logs
export const mockAuditLogs: AuditLog[] = [];

// Last hour - 30 logs
for (let i = 0; i < 30; i++) {
  mockAuditLogs.push(generateAuditLog(i, Math.random() * 60));
}

// 1-6 hours ago - 50 logs
for (let i = 30; i < 80; i++) {
  mockAuditLogs.push(generateAuditLog(i, 60 + Math.random() * 300));
}

// 6-24 hours ago - 60 logs
for (let i = 80; i < 140; i++) {
  mockAuditLogs.push(generateAuditLog(i, 360 + Math.random() * 1080));
}

// 1-7 days ago - 70 logs
for (let i = 140; i < 210; i++) {
  mockAuditLogs.push(generateAuditLog(i, 1440 + Math.random() * 8640));
}

// Sort by timestamp descending (newest first)
mockAuditLogs.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

/**
 * Helper functions
 */
export const getAuditLogsByAction = (action: AuditLog["action"]) => {
  return mockAuditLogs.filter((log) => log.action === action);
};

export const getAuditLogsBySeverity = (severity: AuditLog["severity"]) => {
  return mockAuditLogs.filter((log) => log.severity === severity);
};

export const getAuditLogsByResourceType = (resourceType: AuditLog["resourceType"]) => {
  return mockAuditLogs.filter((log) => log.resourceType === resourceType);
};

export const getAuditLogsByCategory = (category: AuditLog["category"]) => {
  return mockAuditLogs.filter((log) => log.category === category);
};

export const getAuditLogsByActor = (userId: string) => {
  return mockAuditLogs.filter((log) => log.actor.userId === userId);
};
