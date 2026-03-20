import React from "react";
import type { ApiOidcProvider } from "./oidc-provider-table-utils";
import type { ColumnConfig } from "../components/ColumnSelector";
import {
  OidcProviderTableUtils,
  ProviderCell,
  StatusCell,
  ConfigurationCell,
  EndpointsCell,
} from "./oidc-provider-table-utils";

// Define all possible column configurations based on OIDC provider API response structure
export const DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS: ColumnConfig[] = [
  // Basic Information
  {
    id: "provider",
    label: "Provider",
    description: "Display name and provider type with icon",
    isVisible: true,
    isRequired: true,
    category: "basic"
  },
  {
    id: "status",
    label: "Status",
    description: "Active/Inactive status and configuration completeness",
    isVisible: true,
    isRequired: false,
    category: "basic"
  },
  {
    id: "providerName",
    label: "Provider Type",
    description: "OAuth provider type (Google, GitHub, etc.)",
    isVisible: false,
    isRequired: false,
    category: "basic"
  },
  
  // Configuration & Setup
  {
    id: "configuration",
    label: "Configuration",
    description: "OAuth scopes and setup details",
    isVisible: true,
    isRequired: false,
    category: "configuration"
  },
  {
    id: "endpoints",
    label: "Endpoints",
    description: "OAuth endpoint configuration status",
    isVisible: true,
    isRequired: false,
    category: "configuration"
  },
  {
    id: "scopes",
    label: "Scopes",
    description: "Number of OAuth scopes configured",
    isVisible: false,
    isRequired: false,
    category: "configuration"
  },

  // Technical Details
  {
    id: "clientId",
    label: "Client ID",
    description: "OAuth client identifier",
    isVisible: false,
    isRequired: false,
    category: "technical"
  },
  {
    id: "callbackUrl",
    label: "Callback URL",
    description: "OAuth redirect URI",
    isVisible: false,
    isRequired: false,
    category: "technical"
  },
  {
    id: "sortOrder",
    label: "Display Order",
    description: "Display sort order in authentication flow",
    isVisible: true,
    isRequired: false,
    category: "settings"
  },

  // Actions (always visible)
  {
    id: "actions",
    label: "Actions",
    description: "Provider management actions",
    isVisible: true,
    isRequired: true,
    category: "actions"
  }
];

// Cell components for dynamic columns
export const DynamicOidcProviderCellComponents = {
  provider: ({
    provider,
    onProviderClick,
  }: {
    provider: ApiOidcProvider;
    onProviderClick?: (provider: ApiOidcProvider) => void;
  }) => <ProviderCell provider={provider} onSelect={onProviderClick} />,

  status: ({ provider }: { provider: ApiOidcProvider }) => <StatusCell provider={provider} />,

  providerName: ({ provider }: { provider: ApiOidcProvider }) => (
    <span className="text-sm text-foreground">
      {OidcProviderTableUtils.formatProviderName(provider.provider_name)}
    </span>
  ),

  configuration: ({ provider }: { provider: ApiOidcProvider }) => (
    <ConfigurationCell provider={provider} />
  ),

  endpoints: ({ provider }: { provider: ApiOidcProvider }) => <EndpointsCell provider={provider} />,

  scopes: ({ provider }: { provider: ApiOidcProvider }) => (
    <span className="text-sm text-foreground">
      {provider.provider_config.scopes?.length || 0}
    </span>
  ),

  clientId: ({ provider }: { provider: ApiOidcProvider }) => (
    <span className="text-sm font-mono text-foreground">{provider.client_id.substring(0, 12)}...</span>
  ),

  callbackUrl: ({ provider }: { provider: ApiOidcProvider }) => (
    <span className="block max-w-48 truncate text-sm text-foreground" title={provider.callback_url}>
      {provider.callback_url}
    </span>
  ),

  sortOrder: ({ provider }: { provider: ApiOidcProvider }) => (
    <span className="text-sm text-foreground">{provider.sort_order}</span>
  ),
};

// Get column header display name
export function getOidcProviderColumnHeader(columnId: string): string {
  const config = DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS.find(col => col.id === columnId);
  return config?.label || columnId;
}

// Get column accessor key for sorting
export function getOidcProviderColumnAccessorKey(columnId: string): string {
  const accessorMap: Record<string, string> = {
    provider: "display_name",
    status: "is_active",
    providerName: "provider_name",
    configuration: "provider_config.scopes",
    endpoints: "provider_config.auth_url",
    scopes: "provider_config.scopes",
    clientId: "client_id",
    callbackUrl: "callback_url",
    sortOrder: "sort_order",
  };
  return accessorMap[columnId] || columnId;
}

// Category-based column grouping for better organization
export const OIDC_PROVIDER_COLUMN_CATEGORIES = {
  basic: {
    label: "Basic Information",
    description: "Core provider details and status",
    color: "blue"
  },
  configuration: {
    label: "OAuth Configuration",
    description: "OAuth scopes, endpoints, and setup",
    color: "green"
  },
  technical: {
    label: "Technical Details",
    description: "Client IDs, URLs, and identifiers",
    color: "purple"
  },
  settings: {
    label: "Settings",
    description: "Display order and preferences",
    color: "orange"
  },
  activity: {
    label: "Activity",
    description: "Creation dates and usage",
    color: "gray"
  },
  actions: {
    label: "Actions",
    description: "Management and configuration actions",
    color: "red"
  }
} as const;

// Helper function to get columns by category
export function getOidcProviderColumnsByCategory(category: keyof typeof OIDC_PROVIDER_COLUMN_CATEGORIES) {
  return DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS.filter(col => col.category === category);
}

// Validation helper for OIDC provider configuration
export function validateOidcProviderConfiguration(provider: ApiOidcProvider) {
  const issues: string[] = [];
  const config = provider.provider_config;

  if (!config.auth_url) issues.push("Missing authorization URL");
  if (!config.token_url) issues.push("Missing token URL");
  if (!config.client_id) issues.push("Missing client ID");
  if (!config.client_secret) issues.push("Missing client secret");
  if (!config.scopes || config.scopes.length === 0) issues.push("No OAuth scopes configured");
  if (!provider.callback_url) issues.push("Missing callback URL");

  return {
    isValid: issues.length === 0,
    issues,
    completionPercentage: Math.round(((6 - issues.length) / 6) * 100)
  };
}
