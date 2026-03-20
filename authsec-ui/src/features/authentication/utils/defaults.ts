import type {
  AuthMethodConfigData,
  TokenSettings,
  SecurityPolicies,
  ProvisioningConfig,
  BrandingConfig,
  RegistrationPolicy,
  Environment,
  AuthProviderEnhanced,
} from "../types";

/**
 * Smart defaults for authentication method configuration
 */

// Default token settings (production-ready values)
export const DEFAULT_TOKEN_SETTINGS: TokenSettings = {
  idTokenTTL: 3600, // 1 hour
  accessTokenTTL: 900, // 15 minutes
  refreshTokenTTL: 1209600, // 14 days
  rotateRefreshTokens: true,
  reuseRefreshTokens: false,
  enableTokenRevocation: true,
};

// Default security policies (secure by default)
export const DEFAULT_SECURITY_POLICIES: SecurityPolicies = {
  enforcePKCE: true,
  requireMFA: "never",
  allowedOrigins: [],
  sessionTimeout: 3600,
  maxLoginAttempts: 5,
  lockoutDuration: 900,
};

// Default provisioning config
export const DEFAULT_PROVISIONING: ProvisioningConfig = {
  autoCreateUsers: true,
  updateUserAttributes: true,
  syncGroups: false,
  defaultRole: "user",
  attributeMapping: {
    email: "email",
    name: "name",
    groups: "groups",
  },
};

// Default branding config
export const DEFAULT_BRANDING: BrandingConfig = {
  primaryColor: "#0f172a",
};

// Default registration policy (secure by default)
export const DEFAULT_REGISTRATION_POLICY: RegistrationPolicy = {
  allowSelfRegistration: false,
  requireEmailVerification: true,
  requireAdminApproval: true,
  requireMFAOnFirstLogin: false,
};

// Empty auth method config with smart defaults
export const DEFAULT_AUTH_METHOD_CONFIG: AuthMethodConfigData = {
  displayName: "",
  description: "",
  environment: "development",
  clientType: "mcp_client",
  providerType: "oidc",
  methodKey: "",
  isSlugManuallyEdited: false,
  autoAttachToNewServices: false,
  providerConfig: {},
  tokenSettings: DEFAULT_TOKEN_SETTINGS,
  securityPolicies: DEFAULT_SECURITY_POLICIES,
  provisioning: DEFAULT_PROVISIONING,
  branding: DEFAULT_BRANDING,
  registrationPolicy: DEFAULT_REGISTRATION_POLICY,
  attachedServices: [],
};

// Validation helpers for minimal required fields
export const validateMinimalConfig = (data: AuthMethodConfigData): string[] => {
  const errors: string[] = [];

  if (!data.environment) {
    errors.push("Environment is required");
  }

  if (!data.providerType) {
    errors.push("Provider type is required");
  }

  if (!data.clientId) {
    errors.push("Client selection is required");
  }

  // Provider-specific validation
  if (data.providerType === "oidc") {
    if (!data.providerConfig?.providerName?.trim()) {
      errors.push("Please select an OIDC provider");
    }
    if (!data.providerConfig?.clientId?.trim()) {
      errors.push("Client ID is required for OIDC");
    }
  }

  if (data.providerType === "saml") {
    if (!data.providerConfig?.metadataUrl?.trim() && !data.providerConfig?.metadataXml?.trim()) {
      errors.push("Metadata URL or XML is required for SAML");
    }
  }

  return errors;
};

// Generate display name suggestions
export const generateDisplayNameSuggestions = (
  providerType: AuthProviderEnhanced | null,
  environment: Environment
): string[] => {
  if (!providerType) return [];

  const envPrefix =
    environment === "production" ? "Prod" : environment === "staging" ? "Stage" : "Dev";
  const providerPrefix = providerType.toUpperCase();

  return [
    `${envPrefix} ${providerPrefix}`,
    `${providerPrefix} ${envPrefix}`,
    `${providerPrefix} Authentication`,
    `${envPrefix} ${providerPrefix} Auth`,
  ];
};
