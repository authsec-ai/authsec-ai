import { baseApi, withSessionData } from './baseApi';
import { SessionManager } from '@/utils/sessionManager';

// AuthSec Scopes API Interfaces (Updated to match new spec)

// Simple scope name string for GET /uflow/admin/scopes
export type ScopeName = string;

// Scope with resources for detailed operations
export interface ScopeWithResources {
  scope_name: string;
  resources: string[];
}

// Scope mapping for GET /uflow/admin/scopes/mappings
export interface ScopeMapping {
  scope_name: string;
  resources: string[];
}

// Create scope request for POST /uflow/admin/scopes
export interface CreateScopeRequest {
  scope_name: string;
  resources: string[];
}

// Update scope request for PUT /uflow/admin/scopes/{scope_name}
export interface UpdateScopeResourcesRequest {
  resources: string[];
}

// Response types
export interface ScopesListResponse {
  resources: string[];
}

export interface ScopeMappingsResponse {
  scope_name: string;
  resources: string[];
}

export interface CreateScopeResponse {
  property1?: string;
  property2?: string;
}

export interface GenericResponse {
  property1?: string;
  property2?: string;
}

// Legacy interfaces (kept for backward compatibility)
export interface Scope {
  id: string;
  name: string;
  description?: string;
  tenant_id: string;
  created_at: string;
  updated_at?: string;
}

export interface UserDefinedScopeInput {
  name: string;
  description?: string;
}

export interface UserDefinedScopesRequest {
  tenant_id: string;
  scopes: UserDefinedScopeInput[];
}

export interface UpdateScopeRequest {
  tenant_id: string;
  name: string;
  description?: string;
}

export interface DeleteScopesRequest {
  tenant_id: string;
  scope_ids: string[];
}

export interface MapScopesRequest {
  tenant_id: string;
  project_id?: string;
  client_id?: string;
  scope_ids: string[];
}

export interface ScopesResponse {
  scopes?: Scope[];
  message?: string;
  success?: boolean;
}

// Helper to consistently resolve tenant_id for scope endpoints
const getTenantId = (): string => {
  if (typeof window === 'undefined') return '';

  // Primary: SessionManager (localStorage)
  const session = SessionManager.getSession();
  if (session?.tenant_id) return session.tenant_id;
  if (session?.jwtPayload?.tenant_id) return session.jwtPayload.tenant_id;

  // Secondary: sessionStorage fallbacks (common ad-hoc storage)
  const fromSessionStorage =
    sessionStorage.getItem('tenant_id') ||
    sessionStorage.getItem('tenantId');
  if (fromSessionStorage) return fromSessionStorage;

  // Tertiary: raw session JSON if SessionManager parsing failed
  try {
    const raw = localStorage.getItem('authsec_session_v2');
    if (raw) {
      const parsed = JSON.parse(raw);
      if (parsed?.tenant_id || parsed?.tenantId) {
        return parsed.tenant_id || parsed.tenantId;
      }
    }
  } catch {
    // ignore
  }

  // Quaternary: URL query fallback (edge cases)
  try {
    const url = new URL(window.location.href);
    const queryTenant = url.searchParams.get('tenant_id') || url.searchParams.get('tenantId');
    if (queryTenant) return queryTenant;
  } catch {
    // ignore
  }

  return '';
};

export const scopesApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // NEW: List Scopes (Main DB) - GET /uflow/admin/scopes
    // Returns array of unique scope names
    getScopeNames: builder.query<string[], void>({
      query: () => ({
        url: "authsec/uflow/admin/scopes",
        method: "GET",
        headers: {
          tenant_id: getTenantId(),
        },
      }),
      transformResponse: (response: string[]) => {
        if (!Array.isArray(response)) {
          console.warn('Invalid scope names response format:', response);
          return [];
        }
        return response;
      },
      providesTags: ["UnifiedRBACScope"],
    }),

    // NEW: Get Scope Mappings (Main DB) - GET /uflow/admin/scopes/mappings
    // Returns scopes with their associated resources
    getScopeMappings: builder.query<ScopeMapping[], void>({
      query: () => ({
        url: "authsec/uflow/admin/scopes/mappings",
        method: "GET",
        headers: {
          tenant_id: getTenantId(),
        },
      }),
      transformResponse: (response: ScopeMapping[]) => {
        if (!Array.isArray(response)) {
          console.warn('Invalid scope mappings response format:', response);
          return [];
        }
        return response;
      },
      providesTags: ["UnifiedRBACScope"],
    }),

    // NEW: Add Scope (Main DB) - POST /uflow/admin/scopes
    // Creates a scope with associated resources
    createScope: builder.mutation<CreateScopeResponse, CreateScopeRequest>({
      query: (data) => {
        const tenantId = getTenantId();
        return {
          url: "authsec/uflow/admin/scopes",
          method: "POST",
          headers: {
            tenant_id: tenantId,
          },
          body: {
            ...data,
            tenant_id: tenantId,
          },
        };
      },
      invalidatesTags: ["UnifiedRBACScope"],
    }),

    // NEW: Edit Scope (Main DB) - PUT /uflow/admin/scopes/{scope_name}
    // Updates resources associated with a scope
    updateScopeResources: builder.mutation<GenericResponse, { scope_name: string; resources: string[] }>({
      query: ({ scope_name, resources }) => ({
        url: `authsec/uflow/admin/scopes/${encodeURIComponent(scope_name)}`,
        method: "PUT",
        headers: {
          tenant_id: getTenantId(),
        },
        body: { resources },
      }),
      invalidatesTags: ["UnifiedRBACScope"],
    }),

    // NEW: Delete Scope (Main DB) - DELETE /uflow/admin/scopes/{scope_name}
    // Deletes a scope and all its resource mappings
    deleteScopeByName: builder.mutation<GenericResponse, string>({
      query: (scope_name) => ({
        url: `authsec/uflow/admin/scopes/${encodeURIComponent(scope_name)}`,
        method: "DELETE",
        headers: {
          tenant_id: getTenantId(),
        },
      }),
      invalidatesTags: ["UnifiedRBACScope"],
    }),

    // ===== LEGACY ENDPOINTS (Kept for backward compatibility) =====

    // Legacy: List all scopes for a tenant
    getScopes: builder.query<Scope[], { tenant_id: string; audience: 'admin' | 'endUser' }>({
      query: ({ tenant_id, audience }) => {
        const audiencePath = audience === 'admin' ? 'admin' : 'enduser';
        return `authsec/uflow/${audiencePath}/scopes/${tenant_id}`;
      },
      transformResponse: (response: any) => {
        if (!response || typeof response !== 'object') {
          console.warn('Invalid scopes API response format:', response);
          return [];
        }
        if (!Array.isArray(response.scopes)) {
          console.warn('Scopes response missing scopes array:', response);
          return [];
        }
        return response.scopes;
      },
      providesTags: ["UnifiedRBACScope"],
    }),

    // Legacy: Create scopes
    createScopes: builder.mutation<ScopesResponse, UserDefinedScopesRequest>({
      query: (data) => ({
        url: "authsec/uflow/admin/scopes",
        method: "POST",
        body: withSessionData(data),
      }),
      invalidatesTags: ["UnifiedRBACScope"],
    }),

    // Legacy: Update a scope
    updateScope: builder.mutation<ScopesResponse, { id: string; data: UpdateScopeRequest }>({
      query: ({ id, data }) => ({
        url: `authsec/uflow/admin/scopes/${id}`,
        method: "PUT",
        body: withSessionData(data),
      }),
      invalidatesTags: ["UnifiedRBACScope"],
    }),

    // Legacy: Delete scopes
    deleteScopes: builder.mutation<ScopesResponse, DeleteScopesRequest>({
      query: (data) => ({
        url: "authsec/uflow/admin/scopes",
        method: "DELETE",
        body: withSessionData(data),
      }),
      invalidatesTags: ["UnifiedRBACScope"],
    }),

    // Legacy: Map scopes to client
    mapScopes: builder.mutation<ScopesResponse, MapScopesRequest>({
      query: (data) => ({
        url: "authsec/uflow/admin/scopes/map",
        method: "POST",
        body: withSessionData({
          ...data,
          project_id: data.project_id || data.client_id,
        }),
      }),
      invalidatesTags: ["UnifiedRBACScope", "AuthSecClient"],
    }),

    // Legacy: Get scopes for a specific client
    getClientScopes: builder.query<Scope[], { tenantId: string; clientId: string }>({
      query: ({ tenantId, clientId }) => `authsec/uflow/admin/scopes/${tenantId}/${clientId}`,
      transformResponse: (response: any) => {
        if (!response || typeof response !== 'object') {
          console.warn('Invalid client scopes API response format:', response);
          return [];
        }
        if (!Array.isArray(response.scopes)) {
          console.warn('Client scopes response missing scopes array:', response);
          return [];
        }
        return response.scopes;
      },
      providesTags: ["UnifiedRBACScope"],
    }),
  }),
});

export const {
  // New endpoints
  useGetScopeNamesQuery,
  useGetScopeMappingsQuery,
  useCreateScopeMutation,
  useUpdateScopeResourcesMutation,
  useDeleteScopeByNameMutation,
  // Legacy endpoints
  useGetScopesQuery,
  useCreateScopesMutation,
  useUpdateScopeMutation,
  useDeleteScopesMutation,
  useMapScopesMutation,
  useGetClientScopesQuery,
} = scopesApi;
