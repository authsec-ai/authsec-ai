// Simplified Resource interface matching actual API response
export interface Resource {
  id: string;
  name: string;
  description?: string;
  created_at: string;
  updated_at?: string;
}

export interface Scope {
  id: string;
  name: string;
  description: string;
  resourceId: string;
  type: ScopeType;
  isDefault: boolean;
  usageCount: number;
  createdAt: string;
  updatedAt: string;
}

export interface ExternalIntegration {
  id: string;
  name: string;
  provider: ExternalProvider;
  description: string;
  logoUrl: string;
  status: IntegrationStatus;
  configUrl: string;
  sdkSnippet: string;
  defaultScopes: string[];
  configuredScopes: string[];
  connectedAt?: string;
  lastSyncAt?: string;
}

export interface ResourceMetadata {
  apiEndpoint?: string;
  serviceUrl?: string;
  uiRoute?: string;
  tags: string[];
  version?: string;
  deprecated?: boolean;
  sdkSnippet?: string;
}

export type ResourceType = 
  | "api"
  | "service" 
  | "ui"
  | "external"
  | "database"
  | "file"
  | "microservice";

export type ResourceStatus = 
  | "active"
  | "inactive" 
  | "deprecated"
  | "pending";

export type ScopeType = 
  | "read"
  | "write"
  | "delete"
  | "admin"
  | "custom";

export type ExternalProvider = 
  | "google_workspace"
  | "microsoft_365"
  | "salesforce"
  | "github"
  | "slack"
  | "azure_ad"
  | "okta"
  | "aws"
  | "datadog"
  | "stripe";

export type IntegrationStatus = 
  | "connected"
  | "disconnected"
  | "error"
  | "configuring";

// Simplified form data matching API requirements
export interface ResourceFormData {
  name: string;
  description?: string;
}

export interface ResourceFilters {
  search: string;
}

export interface ScopeMatrix {
  roleId: string;
  roleName: string;
  scopes: {
    scopeId: string;
    scopeName: string;
    hasAccess: boolean;
  }[];
}