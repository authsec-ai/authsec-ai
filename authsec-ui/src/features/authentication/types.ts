// Core authentication types - streamlined and focused

// Status and environment types
export type AuthMethodStatus =
  | "active"
  | "inactive"
  | "beta"
  | "deprecated"
  | "enabled"
  | "disabled";
export type Environment = "development" | "staging" | "production";

// Client and provider types
export type ClientType = "mcp-client" | "ai-agent"; // Legacy values
export type AuthProvider = "oauth2" | "saml" | "ldap" | "api-key" | "oidc"; // Legacy values
export type AuthMethodType = "sso" | "mfa" | "oauth" | "biometric" | "password"; // Legacy values

// Enhanced types for new implementation
export type ClientTypeEnhanced = "mcp_client" | "ai_agent";
export type AuthProviderEnhanced = "oidc" | "saml";

// Configuration data interface for auth method creation/editing
export interface AuthMethodConfigData {
  // Basic info
  displayName: string;
  description?: string;
  environment: Environment;
  clientType?: ClientTypeEnhanced;
  providerType?: AuthProviderEnhanced;
  methodKey?: string;
  clientId?: string; // Added for AuthSec API

  // UI state flags
  isSlugManuallyEdited?: boolean;
  autoAttachToNewServices?: boolean;

  // Provider configuration
  providerConfig?: ProviderConfig;

  // Settings
  tokenSettings?: TokenSettings;
  securityPolicies?: SecurityPolicies;
  provisioning?: ProvisioningConfig;
  branding?: BrandingConfig;
  registrationPolicy?: RegistrationPolicy;

  // Management
  status?: AuthMethodStatus;
  attachedServices?: string[];
}

// Simplified provider configuration
export interface ProviderConfig {
  // Common fields
  providerName?: string;

  // OIDC Config
  issuerUrl?: string;
  clientId?: string;
  clientSecret?: string;
  redirectUris?: string[] | string;
  postLogoutUris?: string[];
  scopes?: string[] | string;
  responseType?: string;

  // SAML Config
  entityId?: string;
  ssoUrl?: string;
  acsUrl?: string;
  logoutUrl?: string;
  certificate?: string;
  metadataUrl?: string;
  metadataXml?: string;
  signResponse?: boolean;
  attributeMapping?: {
    userId?: string;
    email?: string;
    name?: string;
    groups?: string;
  };

  // Generic config for other types
  [key: string]: any;
}

// Token configuration
export interface TokenSettings {
  idTokenTTL: number;
  accessTokenTTL: number;
  refreshTokenTTL: number;
  rotateRefreshTokens: boolean;
  reuseRefreshTokens: boolean;
  enableTokenRevocation: boolean;
}

// Security policies
export interface SecurityPolicies {
  enforcePKCE: boolean;
  requireMFA: "never" | "always" | "high_risk";
  allowedOrigins: string[];
  sessionTimeout: number;
  maxLoginAttempts: number;
  lockoutDuration: number;
}

// User provisioning settings
export interface ProvisioningConfig {
  autoCreateUsers: boolean;
  updateUserAttributes: boolean;
  syncGroups: boolean;
  defaultRole: string;
  attributeMapping: Record<string, string>;
}

// Branding configuration
export interface BrandingConfig {
  loginPageUrl?: string;
  logoUrl?: string;
  primaryColor?: string;
  customCSS?: string;
}

// Registration policy
export interface RegistrationPolicy {
  allowSelfRegistration: boolean;
  requireEmailVerification: boolean;
  requireAdminApproval: boolean;
  requireMFAOnFirstLogin: boolean;
}

// Complete auth method interface (what the API returns)
export interface AuthMethod {
  // Core fields
  id: string;
  workspaceId: string;
  name: string; // Legacy field
  displayName: string;
  description?: string;

  // Enhanced fields
  environment?: Environment;
  methodKey?: string;
  version?: number;
  priority?: number;
  attachedServices?: string[];
  autoAttachToNewServices?: boolean;

  // Settings interfaces
  tokenSettings?: TokenSettings;
  securityPolicies?: SecurityPolicies;
  provisioning?: ProvisioningConfig;
  branding?: BrandingConfig;
  registrationPolicy?: RegistrationPolicy;
  audit?: any[];

  // Legacy compatibility fields
  type: AuthMethodType;
  provider: AuthProvider;
  clientType: ClientType;
  providerType: AuthProvider;
  configuration: Record<string, any>;
  providerConfig: ProviderConfig;
  status: AuthMethodStatus;
  usersCount: number;
  lastUsed?: string;
  lastUsedAt?: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
  secretExpiresAt?: string;
  lastError?: string;
}

// Updated AuthStats interface to match API usage
export interface AuthStats {
  totalMethods: number;
  activeMethods: number;
  inactiveMethods: number;
  totalUsers: number;
  averageSuccessRate: number;
  recentlyUsedMethods: number;
  methodsByEnvironment: {
    development: number;
    staging: number;
    production: number;
  };
}

// ===== UNIFIED PROVIDER TYPES =====
// These types unify OIDC and SAML providers for the auth methods table

export type ProviderType = 'oidc' | 'saml';

// API OIDC Provider (from ShowAuthProviders endpoint)
export interface ApiOidcProvider {
  provider_name: string;
  display_name: string;
  client_id: string;
  hydra_client_id?: string;
  callback_url: string;
  endpoints: {
    auth_url: string;
    token_url: string;
    user_info_url?: string;
  };
  is_active: boolean;
  sort_order: number;
  status: string;
}

// API SAML Provider (from ListSamlProviders endpoint)
export interface ApiSamlProvider {
  id: string;
  tenant_id: string;
  client_id?: string;
  provider_name: string;
  display_name: string;
  entity_id: string;
  sso_url: string;
  slo_url?: string;
  certificate: string;
  metadata_url?: string;
  name_id_format: string;
  attribute_mapping: {
    email: string;
    first_name: string;
    last_name: string;
    [key: string]: string;
  };
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

// Unified Auth Provider - represents both OIDC and SAML in a single structure
export interface UnifiedAuthProvider {
  // Common fields
  id: string; // For OIDC: provider_name, For SAML: id
  provider_type: ProviderType;
  provider_name: string;
  display_name: string;
  client_id: string;
  is_active: boolean;
  sort_order: number;
  status: string;

  // OIDC-specific fields (optional)
  callback_url?: string;
  hydra_client_id?: string;
  endpoints?: {
    auth_url: string;
    token_url: string;
    user_info_url?: string;
  };

  // SAML-specific fields (optional)
  entity_id?: string;
  sso_url?: string;
  slo_url?: string;
  certificate?: string;
  metadata_url?: string;
  name_id_format?: string;
  attribute_mapping?: {
    email: string;
    first_name: string;
    last_name: string;
    [key: string]: string;
  };

  // Metadata
  created_at?: string;
  updated_at?: string;

  // Original data for type-specific operations
  _raw?: ApiOidcProvider | ApiSamlProvider;
}
