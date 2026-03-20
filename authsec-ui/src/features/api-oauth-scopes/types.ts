/**
 * API/OAuth Scope Mapping Types
 *
 * Represents the mapping between OAuth scopes and internal API scopes
 * This is separate from the standard RBAC scopes feature
 */

// Core API/OAuth Scope interface (list response)
export interface ApiOAuthScope {
  id: string;                      // e.g., "scope-proj-r"
  name: string;                    // e.g., "project:read"
  description: string;             // e.g., "Allows reading all metadata and files..."
  permissions_linked: number;      // Number of permissions linked to this scope
  created_at: string;              // Creation timestamp
}

// API/OAuth Scope Details (detail response, extends ApiOAuthScope)
export interface ApiOAuthScopeDetails extends ApiOAuthScope {
  permission_ids: string[];        // Array of permission IDs linked to this scope
  permission_strings: string[];    // Array of full permission strings (e.g., "resource:action")
}

// Create API/OAuth Scope Mapping Request
export interface CreateApiOAuthScopeMappingRequest {
  name: string;
  description: string;
  mapped_permission_ids: string[]; // Array of permission IDs to link to this scope
}

// Update API/OAuth Scope Mapping Request
export interface UpdateApiOAuthScopeMappingRequest {
  scope_id: string;
  name: string;
  description: string;
  mapped_permission_ids: string[];
}

// Delete API/OAuth Scope Mapping Request
export interface DeleteApiOAuthScopeMappingRequest {
  scope_id: string;
}

// Query/Filter params
export interface ApiOAuthScopesQueryParams {
  searchQuery?: string;
}
