import type {
  EndUserData,
  SessionData,
  ProviderDistribution,
  MFADistribution,
} from "../../../app/api/dashboardApi";

/**
 * Calculate provider distribution from user data
 */
export function calculateProviderDistribution(users: EndUserData[]): ProviderDistribution[] {
  if (!users || users.length === 0) return [];

  const providerCounts = users.reduce((acc, user) => {
    const provider = user.provider || "unknown";
    acc[provider] = (acc[provider] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  const total = users.length;

  return Object.entries(providerCounts)
    .map(([provider, count]) => ({
      provider: formatProviderName(provider),
      count,
      percentage: (count / total) * 100,
    }))
    .sort((a, b) => b.count - a.count);
}

/**
 * Calculate MFA adoption statistics
 */
export function calculateMFADistribution(users: EndUserData[]): MFADistribution[] {
  if (!users || users.length === 0) return [];

  const mfaCounts: Record<string, number> = {
    none: 0,
    totp: 0,
    webauthn: 0,
    both: 0,
  };

  users.forEach((user) => {
    if (!user.mfa_method || user.mfa_method.length === 0) {
      mfaCounts.none++;
    } else if (user.mfa_method.length > 1) {
      mfaCounts.both++;
    } else if (user.mfa_method.includes("totp")) {
      mfaCounts.totp++;
    } else if (user.mfa_method.includes("webauthn")) {
      mfaCounts.webauthn++;
    }
  });

  const total = users.length;

  return Object.entries(mfaCounts)
    .filter(([_, count]) => count > 0)
    .map(([method, count]) => ({
      method: formatMFAMethodName(method),
      count,
      percentage: (count / total) * 100,
    }))
    .sort((a, b) => b.count - a.count);
}

/**
 * Format provider name for display
 */
export function formatProviderName(provider: string): string {
  const providerMap: Record<string, string> = {
    custom: "Custom Auth",
    google: "Google",
    github: "GitHub",
    microsoft: "Microsoft",
    entra_id: "Entra ID",
    ad_sync: "AD Sync",
    unknown: "Unknown",
  };

  return providerMap[provider.toLowerCase()] || provider;
}

/**
 * Format MFA method name for display
 */
export function formatMFAMethodName(method: string): string {
  const methodMap: Record<string, string> = {
    none: "No MFA",
    totp: "TOTP (Authenticator App)",
    webauthn: "WebAuthn (Biometric/Security Key)",
    both: "Multiple Methods",
  };

  return methodMap[method.toLowerCase()] || method;
}

/**
 * Get provider color for charts
 */
export function getProviderColor(provider: string): string {
  const colorMap: Record<string, string> = {
    "Custom Auth": "#8b5cf6", // purple
    Google: "#ea4335", // google red
    GitHub: "#333333", // github black
    Microsoft: "#00a4ef", // microsoft blue
    "Entra ID": "#0078d4", // azure blue
    "AD Sync": "#50e3c2", // teal
    Unknown: "#94a3b8", // gray
  };

  return colorMap[provider] || "#64748b";
}

/**
 * Get MFA method color for charts
 */
export function getMFAColor(method: string): string {
  const colorMap: Record<string, string> = {
    "No MFA": "#ef4444", // red
    "TOTP (Authenticator App)": "#3b82f6", // blue
    "WebAuthn (Biometric/Security Key)": "#10b981", // green
    "Multiple Methods": "#8b5cf6", // purple
  };

  return colorMap[method] || "#64748b";
}

/**
 * Calculate session duration in minutes
 */
export function calculateSessionDuration(session: SessionData): number {
  const created = new Date(session.created_at).getTime();
  const lastActivity = new Date(session.last_activity).getTime();
  return Math.round((lastActivity - created) / 60000); // minutes
}

/**
 * Format duration in human-readable format
 */
export function formatDuration(minutes: number): string {
  if (minutes < 60) {
    return `${minutes}m`;
  }
  const hours = Math.floor(minutes / 60);
  const remainingMinutes = minutes % 60;
  if (hours < 24) {
    return remainingMinutes > 0 ? `${hours}h ${remainingMinutes}m` : `${hours}h`;
  }
  const days = Math.floor(hours / 24);
  const remainingHours = hours % 24;
  return remainingHours > 0 ? `${days}d ${remainingHours}h` : `${days}d`;
}

/**
 * Format relative time (e.g., "2 hours ago")
 */
export function formatRelativeTime(timestamp: string): string {
  const now = Date.now();
  const time = new Date(timestamp).getTime();
  const diffMinutes = Math.floor((now - time) / 60000);

  if (diffMinutes < 1) return "Just now";
  if (diffMinutes < 60) return `${diffMinutes}m ago`;

  const diffHours = Math.floor(diffMinutes / 60);
  if (diffHours < 24) return `${diffHours}h ago`;

  const diffDays = Math.floor(diffHours / 24);
  if (diffDays < 7) return `${diffDays}d ago`;

  const diffWeeks = Math.floor(diffDays / 7);
  if (diffWeeks < 4) return `${diffWeeks}w ago`;

  const diffMonths = Math.floor(diffDays / 30);
  return `${diffMonths}mo ago`;
}

/**
 * Parse accessible tools from session data
 */
export function parseAccessibleTools(session: SessionData): string[] {
  try {
    if (typeof session.accessible_tools === "string") {
      return JSON.parse(session.accessible_tools);
    }
    if (Array.isArray(session.accessible_tools)) {
      return session.accessible_tools;
    }
    return [];
  } catch {
    return [];
  }
}

/**
 * Calculate growth percentage
 */
export function calculateGrowth(current: number, previous: number): {
  percentage: number;
  isPositive: boolean;
} {
  if (previous === 0) {
    return { percentage: current > 0 ? 100 : 0, isPositive: current > 0 };
  }

  const percentage = ((current - previous) / previous) * 100;
  return {
    percentage: Math.abs(Math.round(percentage)),
    isPositive: percentage >= 0,
  };
}

/**
 * Group users by date for time series charts
 */
export function groupUsersByDate(
  users: EndUserData[],
  groupBy: "day" | "week" | "month" = "day"
): { date: string; count: number }[] {
  const grouped = users.reduce((acc, user) => {
    const date = new Date(user.created_at);
    let key: string;

    if (groupBy === "day") {
      key = date.toISOString().split("T")[0];
    } else if (groupBy === "week") {
      const weekStart = new Date(date);
      weekStart.setDate(date.getDate() - date.getDay());
      key = weekStart.toISOString().split("T")[0];
    } else {
      key = `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}`;
    }

    acc[key] = (acc[key] || 0) + 1;
    return acc;
  }, {} as Record<string, number>);

  return Object.entries(grouped)
    .map(([date, count]) => ({ date, count }))
    .sort((a, b) => a.date.localeCompare(b.date));
}

/**
 * Get active users (users who logged in within the last N days)
 */
export function getActiveUsers(users: EndUserData[], daysThreshold: number = 7): EndUserData[] {
  const now = Date.now();
  const threshold = daysThreshold * 24 * 60 * 60 * 1000; // days to ms

  return users.filter((user) => {
    const lastLogin = new Date(user.last_login).getTime();
    return now - lastLogin <= threshold;
  });
}

/**
 * Export data to CSV
 */
export function exportToCSV(data: any[], filename: string) {
  if (!data || data.length === 0) return;

  const headers = Object.keys(data[0]);
  const csvContent = [
    headers.join(","),
    ...data.map((row) =>
      headers.map((header) => JSON.stringify(row[header] ?? "")).join(",")
    ),
  ].join("\n");

  const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
  const link = document.createElement("a");
  const url = URL.createObjectURL(blob);
  link.setAttribute("href", url);
  link.setAttribute("download", `${filename}.csv`);
  link.style.visibility = "hidden";
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
}
