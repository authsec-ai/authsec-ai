export interface RoleGrant {
  resource: string;
  scopes: string[];
  client?: string;
  isExternal?: boolean;
}

export interface RoleFormData {
  roleId: string;
  displayName: string;
  description: string;
  grants: RoleGrant[];
  assignedUsers: string[];
  assignedGroups: string[];
}

export interface RoleContext {
  source: "blank" | "resource" | "group";
  prefillResource?: string;
  assignToGroup?: string;
}

export interface ValidationError {
  field: string;
  message: string;
}

export interface ClientOption {
  id: string;
  name: string;
  resources: ResourceOption[];
}

export interface ResourceOption {
  path: string;
  label: string;
  scopes: ScopeOption[];
  isExternal?: boolean;
}

export interface ScopeOption {
  name: string;
  description: string;
  isDeprecated?: boolean;
  isExternal?: boolean;
}

export interface RolePreview {
  roleId: string;
  resourceCount: number;
  scopeCount: number;
  userCount: number;
  groupCount: number;
  warnings: string[];
  isValid: boolean;
}

export interface User {
  id: string;
  name: string;
  email: string;
  avatar?: string;
}

export interface Group {
  id: string;
  name: string;
  memberCount: number;
  description?: string;
}
