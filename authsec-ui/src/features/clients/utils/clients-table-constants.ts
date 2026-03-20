// Utility constants for clients table
export const ClientsTableUtils = {
  getStatusVariant: (status: "active" | "restricted" | "disabled") => {
    const statusMap = {
      active: "default",
      restricted: "secondary",
      disabled: "destructive",
    } as const;
    return statusMap[status] || "secondary";
  },
  
  getTypeBadge: (type: string) => {
    const typeMap = {
      mcp_server: "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/30 dark:text-blue-300",
      app: "bg-green-100 text-green-800 border-green-200 dark:bg-green-900/30 dark:text-green-300",
      api: "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/30 dark:text-blue-300",
      other: "bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-900/30 dark:text-gray-300",
      // Legacy support for old schema
      "MCP-Server": "bg-blue-100 text-blue-800 border-blue-200 dark:bg-blue-900/30 dark:text-blue-300",
      "AI-Agent": "bg-green-100 text-green-800 border-green-200 dark:bg-green-900/30 dark:text-green-300",
    };
    return typeMap[type as keyof typeof typeMap] || "bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-900/30 dark:text-gray-300";
  },
  
  getAuthTypeBadge: (authType: string) => {
    const authTypeMap = {
      sso: "bg-emerald-100 text-emerald-800 border-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-300",
      custom: "bg-orange-100 text-orange-800 border-orange-200 dark:bg-orange-900/30 dark:text-orange-300",
      saml2: "bg-sky-100 text-sky-800 border-sky-200 dark:bg-sky-900/30 dark:text-sky-300",
    };
    return authTypeMap[authType as keyof typeof authTypeMap] || "bg-gray-100 text-gray-800 border-gray-200 dark:bg-gray-900/30 dark:text-gray-300";
  },
  
  calculateAuthSuccessRate: (successful: number, denied: number) => {
    const total = successful + denied;
    if (total === 0) return null;
    return Math.round((successful / total) * 100);
  },
  
  getSuccessRateColor: (rate: number | null) => {
    if (rate === null) return "text-gray-500";
    if (rate >= 95) return "text-green-600 dark:text-green-400";
    if (rate >= 80) return "text-yellow-600 dark:text-yellow-400";
    if (rate >= 60) return "text-orange-600 dark:text-orange-400";
    return "text-red-600 dark:text-red-400";
  },
  
  formatLastAccessed: (timestamp?: string | null) => {
    if (!timestamp) return "Never";
    const now = new Date();
    const time = new Date(timestamp);
    const diffInMinutes = Math.floor((now.getTime() - time.getTime()) / (1000 * 60));
    if (diffInMinutes < 60) return `${diffInMinutes}m ago`;
    if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)}h ago`;
    return `${Math.floor(diffInMinutes / 1440)}d ago`;
  },
  
  parseTags: (tagsString: string): string[] => {
    if (!tagsString || tagsString.trim() === "") return [];
    return tagsString.split(",").map(tag => tag.trim()).filter(tag => tag !== "");
  },
  
  getAuthTypeIcon: (authType: string) => "Key",
  getTypeIcon: (type: string) => "Settings",
} as const;
