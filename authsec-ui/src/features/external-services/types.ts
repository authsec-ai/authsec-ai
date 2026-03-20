import type { ExternalService } from "@/types/entities";

/**
 * Status of an external service connection
 */
export type ExternalServiceStatus =
  | "connected" // Service is fully connected and operational
  | "disconnected" // Service was connected but is now disconnected
  | "needs_consent" // Service needs user consent to complete setup
  | "error" // Service has configuration or connection errors
  | "expired" // Service tokens have expired and need refresh
  | "pending"; // Service is in the process of being configured

/**
 * Provider types supported by the external services feature
 */
export type ExternalServiceProvider =
  | "google"
  | "microsoft"
  | "salesforce"
  | "slack"
  | "github"
  | "dropbox"
  | "box"
  | "aws"
  | "azure"
  | "custom";

/**
 * Category types for external services
 */
export type ExternalServiceCategory =
  | "storage"
  | "communication"
  | "crm"
  | "productivity"
  | "development"
  | "analytics"
  | "security"
  | "other";

/**
 * Configuration for an external service connection
 */
export interface ExternalServiceConfig {
  clientId: string;
  clientSecret?: string;
  redirectUri: string;
  scopes: string[];
  tokenEndpoint?: string;
  authEndpoint?: string;
  apiBaseUrl?: string;
}

/**
 * Extended external service with configuration details
 */
export interface ExternalServiceWithConfig extends ExternalService {
  config?: ExternalServiceConfig;
  lastError?: string;
  lastErrorTime?: string;
  tokenExpiresAt?: string;
  consentUrl?: string;
}

/**
 * Filter options for external services
 */
export interface ExternalServiceFilters {
  searchTerm: string;
  provider: string;
  category: string;
  status: string;
  sortBy: string;
  sortDirection: "asc" | "desc";
}

/**
 * User token for an external service
 */
export interface ExternalServiceToken {
  id: string;
  serviceId: string;
  userId: string;
  accessToken: string;
  refreshToken?: string;
  expiresAt: string;
  scopes: string[];
  createdAt: string;
  updatedAt: string;
}

/**
 * Form data for creating a new external service
 */
export interface AddExternalServiceFormData {
  name: string;
  provider: ExternalServiceProvider;
  category: ExternalServiceCategory;
  description?: string;
  config?: Partial<ExternalServiceConfig>;
}

export interface ExternalServiceFormData {
  provider: string;
  providerName: string;
  serviceId: string;
  serviceName: string;
  clientId: string;
  clientSecret: string;
  redirectUri: string;
  linkedClients: string[];
  defaultClientId: string;
  scopes: string[];
  externalResources: ExternalResource[];
  advancedOptions: {
    syncInterval: string;
    customAuthEndpoints: {
      authorizationUrl: string;
      tokenUrl: string;
      userinfoUrl: string;
    };
    tokenStorageRegion: "us" | "eu";
  };
}

export interface ExternalResource {
  resource: string;
  scopes: string[];
}

export interface ClientOption {
  id: string;
  name: string;
  environment: string;
  type: string;
}

export interface ProviderOption {
  id: string;
  name: string;
  icon: string;
  scopes: ProviderScope[];
  resources: ExternalResource[];
  authEndpoints: {
    authorizationUrl: string;
    tokenUrl: string;
    userinfoUrl: string;
  };
}

export interface ProviderScope {
  id: string;
  name: string;
  description: string;
  isDeprecated?: boolean;
  isSensitive?: boolean;
}

export interface ExternalServiceWarning {
  type: "info" | "warning" | "error";
  message: string;
}
