// Database entity types matching our optimized schema

export interface Project {
  id: string;
  name: string;
  slug: string;
  owner_id: string;
  project_metadata: Record<string, any>;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface ProjectMember {
  project_id: string;
  user_id: string;
  role: 'owner' | 'admin' | 'member' | 'viewer';
  joined_at: string;
}

export interface AuthMethod {
  id: string;
  project_id: string;
  display_name: string;
  method_key?: string;
  environment: 'development' | 'staging' | 'production';
  provider_type: 'oidc' | 'saml' | 'directory_sync' | 'custom';
  provider_config: Record<string, any>;
  status: 'active' | 'inactive';
  priority: number;
  created_at: string;
  updated_at: string;
}

export interface User {
  id: string;
  client_id: string;
  tenant_id: string;
  project_id: string;
  name: string;
  email: string;
  tenant_domain: string;
  provider: string;
  provider_id: string;
  provider_data: Record<string, any>;
  password_hash?: string;
  active: boolean;
  scopes: any[];
  roles: any[];
  groups: any[];
  resources: any[];
  MFAEnabled: boolean;
  MFAMethod: string | null;
  MFADefaultMethod: string;
  MFAEnrolledAt: string;
  mfa_verified: boolean;
  created_at: string;
  updated_at: string;
  last_login?: string;
}

export interface UserIdentity {
  id: string;
  user_id: string;
  project_id: string;
  provider: string;
  provider_user_id: string;
  provider_type: 'social' | 'saml' | 'oidc' | 'directory_sync';
  is_primary: boolean;
  created_at: string;
}

export interface Group {
  id: string;
  project_id: string;
  name: string;
  display_name?: string;
  type: 'system' | 'custom' | 'directory_sync';
  membership_type: 'static' | 'dynamic' | 'auto';
  sync_source_id?: string;
  external_group_id?: string;
  is_active: boolean;
  group_metadata: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface UserGroupMembership {
  id: string;
  user_id: string;
  group_id: string;
  assignment_source: 'manual' | 'directory_sync' | 'auto_rule';
  is_active: boolean;
  joined_at: string;
}

export interface Role {
  id: string;
  project_id: string;
  name: string;
  display_name?: string;
  type: 'system' | 'custom' | 'template';
  is_active: boolean;
  role_metadata: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface Resource {
  id: string;
  project_id: string;
  client_id?: string;
  resource_id: string;
  display_name: string;
  type: 'api' | 'database' | 'file' | 'service' | 'system';
  status: 'active' | 'inactive' | 'deprecated';
  auto_discovery: boolean;
  discovery_config: Record<string, any>;
  parent_resource_id?: string;
  resource_metadata: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface Scope {
  id: string;
  resource_id: string;
  scope_name: string;
  display_name: string;
  category: 'read' | 'write' | 'delete' | 'admin' | 'custom';
  severity: 'low' | 'medium' | 'high' | 'critical';
  is_discovered: boolean;
  scope_metadata: Record<string, any>;
  created_at: string;
}

export interface RolePermission {
  role_id: string;
  resource_id: string;
  scope_id: string;
  is_active: boolean;
  granted_at: string;
}

export interface UserRoleAssignment {
  id: string;
  user_id: string;
  role_id: string;
  assignment_source: 'manual' | 'automatic' | 'provisioning';
  is_active: boolean;
  assigned_at: string;
}

export interface GroupRoleAssignment {
  group_id: string;
  role_id: string;
  assignment_source: 'manual' | 'directory_sync' | 'provisioning';
  is_active: boolean;
  assigned_at: string;
}

export interface DirectorySyncSource {
  id: string;
  project_id: string;
  name: string;
  provider_type: 'entra_id' | 'google_workspace' | 'okta' | 'custom';
  sync_config: Record<string, any>;
  sync_enabled: boolean;
  sync_frequency: string;
  last_sync_at?: string;
  created_at: string;
}

export interface PendingScopeDiscovery {
  id: string;
  project_id: string;
  resource_id?: string;
  scope_name: string;
  discovery_source: 'openapi' | 'gateway' | 'schema' | 'manual';
  suggested_category?: string;
  confidence_score?: number;
  status: 'pending' | 'approved' | 'rejected';
  discovery_metadata: Record<string, any>;
  discovered_at: string;
}

// Enhanced types with computed fields for UI
export interface AuthMethodWithStats extends AuthMethod {
  user_count?: number;
  success_rate?: number;
  last_used?: string;
}

export interface UserWithDetails extends Omit<User, 'roles' | 'groups'> {
  identities?: UserIdentity[];
  groups?: Group[];
  roles?: Role[];
  last_login?: string;
  identity_count?: number;
  groups_count?: number;
}

export interface GroupWithStats extends Group {
  member_count?: number;
  roles_count?: number;
  members?: User[];
}

export interface RoleWithStats extends Role {
  user_count?: number;
  group_count?: number;
  permission_count?: number;
  permissions?: (RolePermission & { resource?: Resource; scope?: Scope })[];
}

export interface ResourceWithScopes extends Resource {
  scopes?: Scope[];
  scope_count?: number;
  permission_assignments?: number;
}

// Filter and pagination types
export interface ListParams {
  page?: number;
  limit?: number;
  search?: string;
  sort?: string;
  order?: 'asc' | 'desc';
}

export interface AuthMethodFilters extends ListParams {
  environment?: string;
  provider_type?: string;
  status?: string;
  method_type?: string;
  offset?: number;
}

export interface UserFilters extends ListParams {
  status?: string;
  created_via?: string;
  email_verified?: boolean;
  group_id?: string;
  role_id?: string;
}

export interface GroupFilters extends ListParams {
  type?: string;
  membership_type?: string;
  is_active?: boolean;
  sync_source_id?: string;
}

export interface RoleFilters extends ListParams {
  type?: string;
  is_active?: boolean;
}

export interface ResourceFilters extends ListParams {
  type?: string;
  status?: string;
  auto_discovery?: boolean;
  client_id?: string;
}

// Bulk operation types
export interface BulkUpdateResult {
  updated: number;
  failed: number;
  errors?: string[];
}

export interface BulkDeleteResult {
  deleted: number;
  failed: number;
  errors?: string[];
}

// Analytics types
export interface DashboardStats {
  total_users: number;
  active_users: number;
  total_groups: number;
  total_roles: number;
  total_resources: number;
  total_auth_methods: number;
  recent_activity: any[];
}

export interface AuthMethodAnalytics {
  total_methods: number;
  active_methods: number;
  methods_by_provider: Record<string, number>;
  authentication_trends: Array<{
    date: string;
    authentications: number;
    success_rate: number;
  }>;
}

export interface UserAnalytics {
  total_users: number;
  active_users: number;
  users_by_source: Record<string, number>;
  login_trends: Array<{
    date: string;
    logins: number;
    unique_users: number;
  }>;
}

export interface GroupAnalytics {
  total_groups: number;
  groups_by_type: Record<string, number>;
  membership_distribution: Array<{
    range: string;
    count: number;
  }>;
}

export interface RoleAnalytics {
  total_roles: number;
  roles_by_type: Record<string, number>;
  assignment_trends: Array<{
    date: string;
    assignments: number;
    removals: number;
  }>;
}

export interface ResourceAnalytics {
  total_resources: number;
  resources_by_type: Record<string, number>;
  most_accessed: Array<{
    resource_id: string;
    access_count: number;
  }>;
}

// AuthSec API Response Types
export interface AuthSecUsersResponse {
  users: User[];
  total: number;
  page: number;
  limit: number;
  total_pages: number;
}

export interface AuthSecUsersRequest {
  tenant_id: string;
  client_id: string;
  email?: string;
  active?: boolean;
  provider?: string;
  page?: number;
  limit?: number;
}