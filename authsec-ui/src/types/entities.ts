// Agent/Service permission (legacy)
export interface Permission {
  id: string;
  resource: string;
  actions: string[];
  conditions?: Record<string, any>;
}

export interface Agent {
  id: string;
  name: string;
  type: "API" | "Service" | "Voice" | "MCP";
  clientId: string;
  status: "active" | "inactive" | "suspended";
  tags: string[];
  createdAt: string;
  updatedAt: string;
  lastActivity?: string;
  permissions: Permission[];
  metadata: Record<string, any>;
  hasSecret: boolean;
  secretLastRotated?: string;
  secretExpiresAt?: string;
}

export interface ServiceConfig {
  authentication: "none" | "basic" | "bearer" | "oauth";
  timeout?: number;
  retryPolicy?: {
    maxRetries: number;
    backoffMs: number;
  };
  healthCheck?: {
    endpoint: string;
    intervalMs: number;
  };
}

export interface Service {
  id: string;
  name: string;
  type: "API" | "Database" | "Storage" | "MCP-Server";
  endpoint: string;
  accessStatus: "active" | "restricted" | "disabled";
  authorizedAgents: string[];
  permissions: string[];
  accessLevel: "public" | "internal" | "restricted" | "confidential";
  lastAccessed?: string;
  totalRequests: number;
  createdAt: string;
  updatedAt: string;
}

export interface Client {
  id: string;
  workspace_id: string;
  secret_id: string | null;
  name: string;
  description: string | null;
  type: "mcp_server" | "app" | "api" | "other";
  tags: string;
  authentication_type: "sso" | "custom" | "saml2";
  metadata: Record<string, any>;
  roles: string[];
  mfa_config: {
    enabled: boolean;
    methods: string[];
    backup_codes?: boolean;
    grace_period?: number;
  } | null;
  successful_authentications: number | null;
  denied_authentications: number | null;
  endpoint: string;
  access_status: "active" | "restricted" | "disabled";
  access_level: "public" | "internal" | "restricted" | "confidential";
  total_requests: number | null;
  last_accessed: string | null;
  created_at: string;
  updated_at: string;
  created_by: string | null;
}

export interface ClientWithAuthMethods extends Client {
  attachedMethods: Array<{
    id: string;
    name: string;
    isDefault: boolean;
  }>;
}

export interface ClientInsert {
  id?: string;
  workspace_id: string;
  secret_id?: string | null;
  name: string;
  description?: string | null;
  type?: "mcp_server" | "app" | "api" | "other";
  tags?: string;
  authentication_type?: "sso" | "custom" | "saml2";
  metadata?: Record<string, any>;
  roles?: string[];
  mfa_config?: {
    enabled: boolean;
    methods: string[];
    backup_codes?: boolean;
    grace_period?: number;
  } | null;
  successful_authentications?: number;
  denied_authentications?: number;
  endpoint?: string;
  access_status?: "active" | "restricted" | "disabled";
  access_level?: "public" | "internal" | "restricted" | "confidential";
  total_requests?: number;
  last_accessed?: string | null;
  created_at?: string;
  updated_at?: string;
  created_by?: string | null;
}

export interface ClientUpdate {
  id?: string;
  workspace_id?: string;
  secret_id?: string | null;
  name?: string;
  description?: string | null;
  type?: "mcp_server" | "app" | "api" | "other";
  tags?: string;
  authentication_type?: "sso" | "custom" | "saml2";
  metadata?: Record<string, any>;
  roles?: string[];
  mfa_config?: {
    enabled: boolean;
    methods: string[];
    backup_codes?: boolean;
    grace_period?: number;
  } | null;
  successful_authentications?: number;
  denied_authentications?: number;
  endpoint?: string;
  access_status?: "active" | "restricted" | "disabled";
  access_level?: "public" | "internal" | "restricted" | "confidential";
  total_requests?: number;
  last_accessed?: string | null;
  created_at?: string;
  updated_at?: string;
  created_by?: string | null;
}

export interface ClientsFilters {
  workspace_id?: string;
  name?: string;
  email?: string;
  status?: string;
  type?: "mcp_server" | "app" | "api" | "other";
  authentication_type?: "sso" | "custom" | "saml2";
  access_status?: "active" | "restricted" | "disabled";
  access_level?: "public" | "internal" | "restricted" | "confidential";
  tags?: string[];
  mfa_enabled?: boolean;
  roles?: string[];
  has_security_issues?: boolean;
  auth_success_rate_min?: number;
  auth_success_rate_max?: number;
  search?: string;
}

export interface ClientsPagination {
  page?: number;
  pageSize?: number;
  sortBy?: "name" | "created_at" | "last_accessed" | "total_requests";
  sortOrder?: "asc" | "desc";
}

export interface PaginatedClientsResponse {
  data: ClientWithAuthMethods[];
  count: number | null;
  totalPages: number;
  currentPage: number;
  hasMore: boolean;
}



export interface User {
  id: string;
  name: string;
  email: string;
  avatar?: string;
  status: "active" | "inactive" | "pending";
  role: string;
  team?: string;
  permissions: Permission[];
  lastLogin?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RotationPolicy {
  enabled: boolean;
  intervalDays: number;
  autoRotate: boolean;
  notifyBeforeDays: number;
}

export interface VaultSecret {
  id: string;
  name: string;
  type: "api_key" | "database" | "oauth" | "certificate";
  serviceId: string;
  serviceName: string;
  hasValue: boolean;
  valueRedacted?: string;
  status: "active" | "inactive" | "expired";
  expiresAt?: string;
  lastRotated?: string;
  rotationPolicy?: RotationPolicy;
  createdAt: string;
  updatedAt: string;
}

export interface EventLog {
  id: string;
  timestamp: string;
  level: "error" | "warn" | "info" | "debug";
  service: string;
  event: string;
  userId?: string;
  agentId?: string;
  details: Record<string, any>;
  source: string;
}

export interface AuthLog {
  id: string;
  timestamp: string;
  logType: 'authn' | 'authz'; // authentication vs authorization
  userId?: string;
  username?: string;
  email?: string;
  agentId?: string;
  clientType: 'mcp_server' | 'ai_agent';
  clientId: string;
  clientName: string;
  authMethod: 'password' | 'oauth' | 'saml' | 'webauthn' | 'totp' | 'sms';
  status: 'success' | 'failure' | 'denied' | 'suspicious';
  ipAddress: string;
  location?: string;
  userAgent: string;
  resource?: string; // for authz logs
  action?: string; // for authz logs
  mfaUsed: boolean;
  sessionId?: string;
  failureReason?: string;
  metadata: Record<string, any>;
  rawPayload?: RawAuthLogPayload; // Complete raw API payload for detailed view
}

export interface RawAuthLogPayload {
  timestamp: string;
  log_level?: string;
  event?: {
    type?: string;
    category?: string;
    display_message?: string;
    severity?: string;
    version?: number;
  };
  tenant_id?: string;
  actor?: {
    type?: string;
    id?: string;
    client_id?: string;
    user_id?: string;
    username?: string;
    email?: string;
  };
  client?: Record<string, any>;
  device?: Record<string, any>;
  authentication_context?: Record<string, any>;
  protocol?: Record<string, any>;
  result?: {
    status?: string;
    reason?: string;
    [key: string]: any;
  };
  message?: string;
  action_type?: string;
  metadata?: Record<string, any>;
  security_context?: Record<string, any>;
  policy?: Record<string, any>;
  transaction?: Record<string, any>;
  debug?: {
    internal?: {
      correlation_id?: string;
      [key: string]: any;
    };
    [key: string]: any;
  };
  request?: Record<string, any>;
  processed_at?: string;
  [key: string]: any; // Allow for future fields
}

export interface AuditLog {
  id: string;
  timestamp: string;
  actor: {
    userId: string;
    username: string;
    email: string;
    role: string;
  };
  action: 'created' | 'updated' | 'deleted' | 'enabled' | 'disabled';
  resourceType: 'user' | 'group' | 'role' | 'client' | 'resource' | 'auth_method' | 'config';
  resourceId: string;
  resourceName: string;
  changes?: {
    field: string;
    oldValue: any;
    newValue: any;
  }[];
  severity: 'low' | 'medium' | 'high' | 'critical';
  category: 'identity' | 'access' | 'security' | 'configuration' | 'compliance';
  reason?: string;
  ipAddress: string;
  userAgent: string;
  status: 'success' | 'failed' | 'pending';
  rollbackAvailable?: boolean;
  metadata: Record<string, any>;
}

export interface Role {
  id: string;
  name: string;
  description?: string;
  permissions: Permission[];
  users: string[];
  isSystem: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface AuthMethod {
  id: string;
  name: string;
  type: "oidc" | "saml" | "ldap" | "certificate";
  provider: string;
  configuration: Record<string, any>;
  status: "active" | "inactive";
  userContext: "mcp" | "agent" | "both";
  createdAt: string;
  updatedAt: string;
}

export interface DashboardMetrics {
  totalAgents: number;
  activeAgents: number;
  totalServices: number;
  healthyServices: number;
  totalUsers: number;
  activeUsers: number;
  totalSecrets: number;
  expiringSecrets: number;
  recentEvents: number;
  mcpClients: number;
}

export interface ActivityFeedItem {
  id: string;
  type: "agent_created" | "service_added" | "secret_rotated" | "user_login";
  title: string;
  description: string;
  timestamp: string;
  userId?: string;
  metadata?: Record<string, any>;
}

// ===== RBAC System Types =====

// RBAC Permission (new spec) - Atomic permission (resource + action)
export interface RbacPermission {
  id: string;
  resource: string;
  action: string; // singular action
  description: string;
  full_permission_string: string; // "resource:action" format
  roles_assigned?: number; // Admin context only
  role_names?: string[]; // Tenant context only
  created_at?: string;
  updated_at?: string;
  tenant_id?: string;
}

// Scope with resources (new spec)
export interface RbacScope {
  scope_name: string; // Primary identifier (not ID-based)
  resources: string[];
}

// Scope mapping (for GET /uflow/admin/scopes/mappings)
export interface ScopeMapping {
  scope_name: string;
  resources: string[];
}

// Legacy Scope interface (kept for backward compatibility)
export interface Scope {
  id: string;
  name: string;
  description?: string;
  resourceId: string;
  isDeprecated?: boolean;
  createdAt: string;
  updatedAt: string;
}

export interface Resource {
  id: string;
  name: string;
  description?: string;
  clientId: string;
  clientName: string;
  type?: "database" | "system" | "api" | "file" | "service" | "external";
  scopes: Scope[];
  linkedRoles: string[];
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  isExternal?: boolean;
  externalServiceId?: string;
  externalServiceName?: string;
}

export interface Group {
  id: string;
  name: string;
  description?: string;
  memberCount: number;
  roleIds: string[];
  userIds: string[];
  createdAt: string;
  updatedAt: string;
  createdBy: string;
  isSystem?: boolean;
}

export interface RolePermission {
  resourceId: string;
  resourceName: string;
  scopes: string[];
  scopeNames?: string[];
  isExternal?: boolean;
}

export interface EnhancedRole {
  id: string;
  name: string;
  description?: string;
  type?: "system" | "custom";
  permissions?: RolePermission[];
  permissionCount?: number;
  permissions_count?: number;
  users_assigned?: number;
  usernames?: string[];
  tenant_id?: string;
  client_id?: string;
  project_id?: string;
  userIds?: string[];
  groupIds?: string[];
  userCount?: number;
  groupCount?: number;
  isBuiltIn?: boolean;
  version?: number;
  versions?: RoleVersion[];
  createdAt?: string;
  created_by?: string;
  updatedAt?: string;
  createdBy?: string;
}

export interface RoleVersion {
  version: number;
  changes: string[];
  permissions: RolePermission[];
  changedBy: string;
  changedAt: string;
  timestamp: string;
  author: string;
  changeType: "create" | "update" | "permissions" | "rollback";
}

export interface EnhancedUser {
  id: string;
  name: string;
  email: string;
  avatar?: string;
  status: "active" | "inactive" | "pending" | "locked";
  directRoleIds: string[];
  groupIds: string[];
  effectiveRoleIds: string[];
  lastLogin?: string;
  lastLoginMethod?: string;
  loginCount: number;
  groupNames: string[];
  roleNames: string[];
  isOrphan: boolean;
  createdAt: string;
  updatedAt: string;
  invitedBy?: string;
  invitedAt?: string;
  // Additional fields for enhanced data and dynamic columns
  first_name?: string | null;
  last_name?: string | null;
  avatar_url?: string | null;
  provider?: string;
  failed_login_count?: number;
  password_reset_required?: boolean;
  accepted_at?: string | null;
  metadata?: {
    department?: string;
    job_title?: string;
    manager_id?: string;
    employee_id?: string;
    location?: string;
    phone?: string;
    timezone?: string;
    start_date?: string;
    cost_center?: string;
    security_clearance?: string;
  };
  // Computed fields for UI
  displayName?: string;
  initials?: string;
  lastActiveText?: string;
  roleText?: string;
  teamText?: string;
  permissionsCount?: number;
  // API fields (snake_case from backend)
  active?: boolean;
  roles?: any[];
  groups?: any[];
  last_login?: string;
  created_at?: string;
  updated_at?: string;
  MFAEnabled?: boolean;
  MFAMethod?: string[] | null;
  MFADefaultMethod?: string;
  mfa_verified?: boolean;
  is_synced_user?: boolean;
  last_sync_at?: string;
  username?: string;
  client_id?: string;
  tenant_id?: string;
  project_id?: string;
  tenant_domain?: string;
  provider_id?: string;
  provider_data?: any;
  external_id?: string;
  sync_source?: string;
  is_synced?: boolean;
  accepted_invite?: boolean;
  MFAEnrolledAt?: string;
  scopes?: any[];
  resources?: any[];
}

export interface GroupMembership {
  userId: string;
  groupId: string;
  addedAt: string;
  addedBy: string;
}

export interface RoleAssignment {
  id: string;
  roleId: string;
  assigneeType: "user" | "group";
  assigneeId: string;
  assignedAt: string;
  assignedBy: string;
}

export interface PermissionMatrix {
  userId: string;
  resourceId: string;
  resourceName: string;
  scopes: string[];
  scopeNames: string[];
  grantedVia: Array<{
    type: "direct_role" | "group_role";
    roleName: string;
    groupName?: string;
  }>;
}

export interface RbacKpis {
  totalUsers: number;
  activeUsers: number;
  pendingInvites: number;
  usersWithoutGroup: number;
  totalGroups: number;
  groupsWithoutRoles: number;
  totalRoles: number;
  rolesWithoutResources: number;
  totalResources: number;
  totalScopes: number;
  activeLastWeek: number;
}

export interface BulkUserImport {
  email: string;
  name: string;
  groupSlugs: string[];
  roleSlugs: string[];
  sendInvite?: boolean;
}

export interface InviteRequest {
  emails: string[];
  groupIds?: string[];
  roleIds?: string[];
  message?: string;
  expiresAt?: string;
}

export interface ClientRbacConfig {
  clientId: string;
  defaultGroupId?: string;
  defaultRoleId?: string;
  autoCreateResourcePrefix: boolean;
  resourcePrefix?: string;
  allowSelfRegistration: boolean;
  selfRegistrationGroupId?: string;
  selfRegistrationRoleId?: string;
}

export interface CrossPageContext {
  sourceScreen: string;
  filters: Record<string, any>;
  selectedItems: string[];
  returnUrl?: string;
}

export interface PendingScopeDiscovery {
  id: string;
  name: string;
  description: string;
  resourceId: string;
  clientId: string;
  clientName: string;
  discoveredAt: string;
  status: "pending_review" | "approved" | "rejected";
}

export interface ExternalService {
  id: string;
  name: string;
  provider: string; // e.g., google, microsoft
  category: string; // Storage, CRM, etc.
  clientCount: number;
  userTokenCount: number;
  status: "connected" | "needs_consent" | "error";
  lastSync: string; // ISO date
  lastError?: string | null;
  createdAt?: string;
}

export interface ClientBranding {
  logo?: string;
  logoUrl?: string;
  companyName: string;
  primaryColor?: string;
  secondaryColor?: string;
  backgroundColor?: string;
  textColor?: string;
  customCss?: string;
}

export interface SSOProvider {
  id: string;
  name: string;
  type: "saml" | "oidc" | "oauth2";
  displayName: string;
  iconUrl?: string;
  buttonColor?: string;
  textColor?: string;
  config: {
    // SAML specific
    entityId?: string;
    ssoUrl?: string;
    x509Certificate?: string;
    // OIDC specific
    issuer?: string;
    clientId?: string;
    clientSecret?: string;
    authorizationUrl?: string;
    tokenUrl?: string;
    userInfoUrl?: string;
    // OAuth2 specific
    authUrl?: string;
    scopes?: string[];
  };
  attributeMapping?: {
    email: string;
    firstName?: string;
    lastName?: string;
    groups?: string;
  };
  isActive: boolean;
}

export interface ClientAuthSettings {
  enabledMethods: Array<"email" | "google" | "github" | "microsoft" | "sso">;
  allowSignup: boolean;
  requireEmailVerification: boolean;
  passwordRequirements?: {
    minLength: number;
    requireUppercase: boolean;
    requireLowercase: boolean;
    requireNumbers: boolean;
    requireSpecialChars: boolean;
  };
  ssoProviders?: SSOProvider[];
  defaultSSOProvider?: string; // SSO provider ID to use as default
}

export interface ClientConfiguration {
  id: string;
  clientId: string;
  clientName: string;
  clientDomain?: string;
  branding: ClientBranding;
  authSettings: ClientAuthSettings;
  customFields?: Array<{
    name: string;
    type: "text" | "email" | "select";
    required: boolean;
    options?: string[];
  }>;
  redirectUrls: {
    success: string;
    failure: string;
    logout?: string;
  };
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  createdBy: string;
}
