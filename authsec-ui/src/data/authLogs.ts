import type { AuthLog } from "../types/entities";

/**
 * Mock authentication and authorization log data
 * Represents login attempts, auth decisions, and access control events
 */

const users = [
  { id: "user_001", username: "john.doe", email: "john.doe@company.com" },
  { id: "user_002", username: "sarah.johnson", email: "sarah.johnson@company.com" },
  { id: "user_003", username: "alice.smith", email: "alice.smith@company.com" },
  { id: "user_004", username: "bob.wilson", email: "bob.wilson@company.com" },
  { id: "user_005", username: "emma.davis", email: "emma.davis@company.com" },
  { id: "user_006", username: "mike.brown", email: "mike.brown@company.com" },
  { id: "user_007", username: "lisa.garcia", email: "lisa.garcia@company.com" },
  { id: "user_008", username: "david.martinez", email: "david.martinez@company.com" },
  { id: "user_009", username: "emily.rodriguez", email: "emily.rodriguez@company.com" },
  { id: "user_010", username: "james.lee", email: "james.lee@company.com" },
];

const clients = [
  { id: "client_mcp_001", name: "Analytics MCP Server", type: "mcp_server" as const },
  { id: "client_mcp_002", name: "Data Processing MCP", type: "mcp_server" as const },
  { id: "client_mcp_003", name: "Content Management MCP", type: "mcp_server" as const },
  { id: "client_mcp_004", name: "Payment Gateway MCP", type: "mcp_server" as const },
  { id: "client_ai_001", name: "Customer Support Assistant", type: "ai_agent" as const },
  { id: "client_ai_002", name: "Code Review Assistant", type: "ai_agent" as const },
  { id: "client_ai_003", name: "Documentation Generator", type: "ai_agent" as const },
  { id: "client_ai_004", name: "Data Analysis Agent", type: "ai_agent" as const },
];

const locations = [
  "San Francisco, CA",
  "New York, NY",
  "London, UK",
  "Tokyo, Japan",
  "Berlin, Germany",
  "Sydney, Australia",
  "Toronto, Canada",
  "Singapore",
  "Mumbai, India",
  "São Paulo, Brazil",
  "Unknown",
];

const ipAddresses = [
  "192.168.1.100",
  "10.0.1.45",
  "172.16.0.23",
  "203.0.113.42",
  "198.51.100.67",
  "192.0.2.156",
  "198.18.0.99",
  "192.88.99.77",
  "10.0.0.88",
  "172.31.255.99",
];

const userAgents = [
  "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
  "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
  "AuthSec-Agent/2.1.0",
  "MCP-Client/1.5.3",
  "AI-Assistant/3.0.1",
  "curl/7.84.0",
];

const authMethods: Array<AuthLog["authMethod"]> = [
  "password",
  "oauth",
  "saml",
  "webauthn",
  "totp",
  "sms",
];

const statuses: Array<AuthLog["status"]> = ["success", "failure", "denied", "suspicious"];

const resources = [
  "/api/v1/users",
  "/api/v1/admin/config",
  "/api/v1/sensitive-data",
  "/api/v1/analytics/reports",
  "/api/v1/payments/process",
  "/api/v1/clients",
  "/api/v1/roles",
  "/api/v1/policies",
];

const actions = ["read", "write", "delete", "update", "admin"];

function randomItem<T>(arr: T[]): T {
  return arr[Math.floor(Math.random() * arr.length)];
}

function generateAuthLog(index: number, minutesAgo: number): AuthLog {
  const timestamp = new Date(Date.now() - minutesAgo * 60 * 1000).toISOString();
  const user = randomItem(users);
  const client = randomItem(clients);
  const method = randomItem(authMethods);

  // 70% success, 20% failure, 8% denied, 2% suspicious
  let status: AuthLog["status"];
  const rand = Math.random();
  if (rand < 0.70) status = "success";
  else if (rand < 0.90) status = "failure";
  else if (rand < 0.98) status = "denied";
  else status = "suspicious";

  // 80% authn, 20% authz
  const logType: AuthLog["logType"] = Math.random() < 0.8 ? "authn" : "authz";

  const mfaUsed = Math.random() < 0.6; // 60% of attempts use MFA
  const location = status === "suspicious" ? "Unknown" : randomItem(locations);
  const ipAddress = status === "suspicious" ? "203.0.113.42" : randomItem(ipAddresses);

  const log: AuthLog = {
    id: `auth_log_${String(index).padStart(4, "0")}`,
    timestamp,
    logType,
    userId: user.id,
    username: user.username,
    email: user.email,
    clientType: client.type,
    clientId: client.id,
    clientName: client.name,
    authMethod: method,
    status,
    ipAddress,
    location,
    userAgent: randomItem(userAgents),
    mfaUsed,
    sessionId: `sess_${Math.random().toString(36).substr(2, 9)}`,
    metadata: {},
  };

  // Add authz-specific fields
  if (logType === "authz") {
    log.resource = randomItem(resources);
    log.action = randomItem(actions);

    if (status === "denied") {
      log.metadata.requiredRole = "admin";
      log.metadata.userRole = "developer";
    }
  }

  // Add failure reason
  if (status === "failure") {
    const failureReasons = [
      "Invalid credentials",
      "Account locked",
      "Password expired",
      "MFA verification failed",
      "Token expired",
    ];
    log.failureReason = randomItem(failureReasons);
    log.metadata.attemptCount = Math.floor(Math.random() * 5) + 1;
  }

  if (status === "denied") {
    log.failureReason = "Insufficient permissions";
  }

  if (status === "suspicious") {
    log.failureReason = "Unusual activity detected";
    log.metadata.riskScore = Math.random() * 100;
    log.metadata.flags = ["unusual_location", "rapid_requests"];
  }

  return log;
}

// Generate 200+ auth logs
export const mockAuthLogs: AuthLog[] = [];

// Last hour - 50 logs (recent activity)
for (let i = 0; i < 50; i++) {
  mockAuthLogs.push(generateAuthLog(i, Math.random() * 60));
}

// 1-6 hours ago - 60 logs
for (let i = 50; i < 110; i++) {
  mockAuthLogs.push(generateAuthLog(i, 60 + Math.random() * 300));
}

// 6-24 hours ago - 50 logs
for (let i = 110; i < 160; i++) {
  mockAuthLogs.push(generateAuthLog(i, 360 + Math.random() * 1080));
}

// 1-7 days ago - 50 logs
for (let i = 160; i < 210; i++) {
  mockAuthLogs.push(generateAuthLog(i, 1440 + Math.random() * 8640));
}

// Sort by timestamp descending (newest first)
mockAuthLogs.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());

/**
 * Helper functions
 */
export const getAuthLogsByType = (logType: AuthLog["logType"]) => {
  return mockAuthLogs.filter((log) => log.logType === logType);
};

export const getAuthLogsByStatus = (status: AuthLog["status"]) => {
  return mockAuthLogs.filter((log) => log.status === status);
};

export const getAuthLogsByClientType = (clientType: AuthLog["clientType"]) => {
  return mockAuthLogs.filter((log) => log.clientType === clientType);
};

export const getAuthLogsByMethod = (method: AuthLog["authMethod"]) => {
  return mockAuthLogs.filter((log) => log.authMethod === method);
};

export const getAuthLogsByUser = (userId: string) => {
  return mockAuthLogs.filter((log) => log.userId === userId);
};
