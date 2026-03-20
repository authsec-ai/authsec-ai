import { baseApi, withSessionData } from "./baseApi";
import type { Client } from "../../types/entities";
import type { AuthMethod } from "../../features/authentication/types";

// AuthSec API specific interfaces
export interface GetClientsRequest {
  tenant_id: string;
  active_only?: boolean;
  filters?: Record<string, any>;
  page?: number;
  limit?: number;
}

export interface ClientData {
  id: string;
  client_id: string;
  tenant_id: string;
  project_id: string;
  owner_id?: string | null;
  org_id?: string | null;
  name: string;
  status?: string;
  email?: string | null;
  tags?: string[] | null;
  active: boolean;
  mfa_enabled?: boolean;
  mfa_method?: string | null;
  mfa_verified?: boolean;
  roles?: string[] | null;
  oidc_enabled?: boolean;
  hydra_client_id?: string | null;
  created_at: string;
  updated_at: string;
  [key: string]: unknown;
}

export interface ClientsFilters {
  email?: string;
  name?: string;
  status?: string;
  tags?: string[];
  [key: string]: unknown;
}

export interface ClientsPagination {
  limit: number;
  page: number;
  total: number;
  total_pages?: number;
  [key: string]: unknown;
}

export interface GetClientsResponse {
  clients: ClientData[];
  filters?: ClientsFilters;
  pagination?: ClientsPagination;
}

// Enhanced client data with authentication methods for /clients/all endpoint
export interface EnhancedClientData extends ClientData {
  authentication_methods?: Array<{
    id: string;
    name: string;
    type: string;
    is_default?: boolean;
    enabled: boolean;
    provider?: string;
    metadata?: Record<string, any>;
  }> | string; // Can be string from API or array of objects
  auth_methods_count?: number;
  client_name?: string; // Alternative name field
  description?: string;
  user_count?: number; // From actual API response
  enabled?: boolean; // From actual API response
}

export interface GetAllClientsResponse {
  clients: EnhancedClientData[];
  total: number;
  filters?: ClientsFilters;
  pagination?: ClientsPagination;
}

export interface RegisterClientRequest {
  name: string;
  email: string;
  tenant_id: string;
  project_id: string;
  react_app_url: string;
}

export interface RegisterClientResponse {
  id: string;
  client_id: string;
  tenant_id: string;
  project_id: string;
  name: string;
  secret_id: string;
  email: string;
  active: boolean;
  created_at: string;
  message: string;
}

export interface DeleteClientRequest {
  tenant_id: string;
  client_id: string;
}

export interface DeleteClientResponse {
  client_id: string;
  message: string;
  tenant_id: string;
}

export interface SetClientStatusRequest {
  tenant_id: string;
  client_id: string;
  active: boolean;
}

export interface SetClientStatusResponse {
  message: string;
  success: boolean;
  data: {
    active: boolean;
    client: ClientData;
    client_id: string;
    tenant_id: string;
  };
  timestamp: string;
}

export interface OIDCProvider {
  provider_name: string;
  display_name: string;
  client_id: string;
  client_secret: string;
  auth_url: string;
  token_url: string;
  user_info_url: string;
  scopes: string[];
  is_active: boolean;
}

export interface AddProviderRequest {
  tenant_id: string;
  client_id: string;
  provider: OIDCProvider;
  created_by: string;
}

export interface AddProviderResponse {
  message: string;
  success: boolean;
  data: {
    callback_url: string;
    client_id: string;
    created_at: string;
    display_name: string;
    is_active: boolean;
    provider_name: string;
    tenant_id: string;
  };
  timestamp: string;
}

export interface GetConfigRequest {
  tenant_id: string;
}

export interface GetConfigResponse {
  message: string;
  success: boolean;
  data: {
    oidc_providers: Array<{
      callback_url: string;
      client_id: string;
      created_at: string;
      display_name: string;
      is_active: boolean;
      provider_config: {
        additional_params: any;
        auth_url: string;
        client_id: string;
        client_secret: string;
        issuer_url: string;
        jwks_url: string;
        scopes: string[];
        token_url: string;
        user_info_url: string;
      };
      provider_name: string;
      sort_order: number;
    }>;
    org_id: string;
    provider_count: number;
    tenant_client: {
      client_id: string;
      client_name: string;
      created_at: string;
      redirect_uris: string[];
      scopes: string[];
    };
    tenant_id: string;
  };
  timestamp: string;
}

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const normalizeClientsResponse = (response: unknown): GetClientsResponse => {
  if (Array.isArray(response)) {
    return { clients: response as ClientData[] };
  }

  if (!isRecord(response)) {
    return { clients: [] };
  }

  // TEMP WORKAROUND: Handle broken API response format
  // The API is returning: { client_name, user_count, authentication_methods: "password", enabled }
  // But we need full ClientData with client_id, tenant_id, etc.
  if (Array.isArray(response.clients)) {
    const hasClientId = response.clients.length > 0 && 'client_id' in response.clients[0];
    
    if (!hasClientId && response.clients.length > 0) {
      // API is returning broken format - transform it
      console.warn('⚠️ CRITICAL: API returning incomplete client data. Generating placeholder IDs.');
      console.warn('⚠️ This will break authentication provider dropdown and other features!');
      console.warn('⚠️ Backend team needs to fix /clients/getClients endpoint ASAP!');
      
      const transformedClients = response.clients.map((client: any, index: number) => {
        // Generate a deterministic ID from client_name to maintain consistency
        const deterministicId = `client-${client.client_name?.toLowerCase().replace(/\s+/g, '-') || index}`;
        
        return {
          id: deterministicId,
          client_id: deterministicId, // This is WRONG but API doesn't provide real UUID
          tenant_id: '', // API doesn't provide this
          project_id: '', // API doesn't provide this
          owner_id: null,
          org_id: null,
          name: client.client_name || 'Unnamed Client',
          status: client.enabled ? 'active' : 'inactive',
          email: null,
          tags: null,
          active: client.enabled ?? true,
          mfa_enabled: true, // Default to ON as per requirement
          mfa_method: 'password',
          mfa_verified: false,
          roles: null,
          oidc_enabled: false,
          hydra_client_id: null,
          created_at: new Date().toISOString(),
          updated_at: new Date().toISOString(),
          // Store original minimal data for reference
          _original_api_data: client,
        } as ClientData;
      });
      
      return {
        clients: transformedClients,
        filters: isRecord(response.filters) ? (response.filters as ClientsFilters) : undefined,
        pagination: isRecord(response.pagination) ? (response.pagination as ClientsPagination) : undefined,
      };
    }
    
    return {
      clients: response.clients as ClientData[],
      filters: isRecord(response.filters) ? (response.filters as ClientsFilters) : undefined,
      pagination: isRecord(response.pagination) ? (response.pagination as ClientsPagination) : undefined,
    };
  }

  const data = response.data;
  if (isRecord(data)) {
    const clients = Array.isArray(data.clients) ? (data.clients as ClientData[]) : [];

    let filters: ClientsFilters | undefined;
    if (isRecord(data.filters)) {
      filters = data.filters as ClientsFilters;
    } else if (isRecord(response.filters)) {
      filters = response.filters as ClientsFilters;
    }

    let pagination: ClientsPagination | undefined;
    if (isRecord(data.pagination)) {
      pagination = data.pagination as ClientsPagination;
    } else if (isRecord(response.pagination)) {
      pagination = response.pagination as ClientsPagination;
    } else {
      const limitValue =
        typeof data.limit === "number"
          ? data.limit
          : typeof data.page_size === "number"
          ? data.page_size
          : undefined;
      const pageValue = typeof data.page === "number" ? data.page : undefined;
      const totalValue = typeof data.count === "number" ? data.count : undefined;
      const totalPagesValue =
        typeof data.total_pages === "number" ? data.total_pages : undefined;

      if (
        limitValue !== undefined ||
        totalValue !== undefined ||
        pageValue !== undefined ||
        totalPagesValue !== undefined
      ) {
        pagination = {
          limit: limitValue ?? clients.length ?? 0,
          page: pageValue ?? 1,
          total: totalValue ?? clients.length ?? 0,
          total_pages: totalPagesValue,
        };
      }
    }

    return {
      clients,
      filters,
      pagination,
    };
  }

  return {
    clients: [],
    filters: isRecord(response.filters) ? (response.filters as ClientsFilters) : undefined,
    pagination: isRecord(response.pagination)
      ? (response.pagination as ClientsPagination)
      : undefined,
  };
};

/**
 * Parses backend responses that may contain multiple concatenated JSON objects
 * Takes the first valid JSON object and ignores subsequent ones
 */
const parseFirstValidJSON = <T>(response: unknown): T => {
  // If response is already an object, return it
  if (isRecord(response)) {
    return response as T;
  }

  // If response is a string, try to parse multiple JSON objects
  if (typeof response === 'string') {
    // Split on pattern }{ or }\n{ to detect multiple JSON objects
    const jsonPattern = /\}\s*\{/g;
    const matches = response.match(jsonPattern);

    if (matches && matches.length > 0) {
      // Multiple JSON objects detected - extract the first one
      const firstJsonEnd = response.indexOf(matches[0]) + 1;
      const firstJsonStr = response.substring(0, firstJsonEnd);

      try {
        return JSON.parse(firstJsonStr) as T;
      } catch (e) {
        console.error('Failed to parse first JSON object:', e);
        throw new Error('Invalid JSON response from server');
      }
    }

    // No multiple objects, parse normally
    try {
      return JSON.parse(response) as T;
    } catch (e) {
      console.error('Failed to parse JSON response:', e);
      throw new Error('Invalid JSON response from server');
    }
  }

  // Unknown type, return as-is
  return response as T;
};

export const clientApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Delete a client completely
    deleteClientComplete: builder.mutation<DeleteClientResponse, DeleteClientRequest>({
      query: ({ tenant_id, client_id }) => ({
        url: `/authsec/clientms/tenants/${tenant_id}/clients/delete-complete`,
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: withSessionData({
          tenant_id,
          client_id,
        }),
      }),
      invalidatesTags: [{ type: "Client", id: "LIST" }],
    }),

    // Set client status (activate/deactivate)
    setClientStatus: builder.mutation<SetClientStatusResponse, SetClientStatusRequest>({
      query: (data) => ({
        url: `/authsec/clientms/tenants/${data.tenant_id}/clients/set-status`,
        method: "POST",
        body: data,
      }),
      invalidatesTags: [{ type: "Client", id: "LIST" }],
    }),

    // Get all clients for a tenant (query for automatic fetching)
    getClients: builder.query<GetClientsResponse, GetClientsRequest>({
      query: (data) => {
        const params: Record<string, any> = {};

        if (data.active_only !== undefined) {
          params.active_only = data.active_only;
        }

        if (data.filters) {
          for (const [key, value] of Object.entries(data.filters)) {
            if (
              value === undefined ||
              value === null ||
              (typeof value === "string" && value.trim() === "") ||
              (Array.isArray(value) && value.length === 0)
            ) {
              continue;
            }
            params[key] = value;
          }
        }

        return {
          url: `/authsec/clientms/tenants/${data.tenant_id}/clients/getClients`,
          method: "GET",
          params,
          responseHandler: "text", // Get raw text to handle potential multiple JSON objects
        };
      },
      transformResponse: (response: string) => {
        // Parse first valid JSON if backend sends duplicates
        const parsed = parseFirstValidJSON<unknown>(response);
        
        // CRITICAL: Backend API is returning incomplete data structure
        // The /clients/getClients endpoint now returns:
        // { client_name, user_count, authentication_methods: string, enabled }
        // But we need: client_id, tenant_id, id, and full ClientData fields
        //
        // This is a TEMPORARY WORKAROUND - Backend needs to fix this endpoint!
        console.warn('⚠️ WARNING: /clients/getClients endpoint returning incomplete data without client_id');
        console.log('Raw API response simplified:', parsed);
        
        return normalizeClientsResponse(parsed);
      },
      providesTags: (result) =>
        result?.clients && result.clients.length
          ? [
              ...result.clients.map((client) => ({
                type: "Client" as const,
                id: client.client_id ?? client.id ?? "UNKNOWN",
              })),
              { type: "Client" as const, id: "LIST" },
            ]
          : [{ type: "Client" as const, id: "LIST" }],
    }),

    // Get all clients with enhanced data including authentication methods
    // Note: Using /clients/getClients since /clients/all endpoint doesn't exist on backend
    getAllClients: builder.query<GetAllClientsResponse, GetClientsRequest>({
      query: (data) => {
        const params: Record<string, any> = {};

        if (data.active_only !== undefined) {
          params.active_only = data.active_only;
        }

        if (data.page !== undefined) {
          params.page = data.page;
        }

        if (data.limit !== undefined) {
          params.limit = data.limit;
        }

        if (data.filters) {
          for (const [key, value] of Object.entries(data.filters)) {
            if (
              value === undefined ||
              value === null ||
              (typeof value === "string" && value.trim() === "") ||
              (Array.isArray(value) && value.length === 0)
            ) {
              continue;
            }
            params[key] = value;
          }
        }

        return {
          url: `/authsec/clientms/tenants/${data.tenant_id}/clients/getClients`,
          method: "GET",
          params,
          responseHandler: "text", // Handle potential response format issues
        };
      },
      transformResponse: (response: string) => {
        try {
          // Parse the regular clients response and transform it to enhanced format
          const parsed = parseFirstValidJSON<any>(response);
          console.log('Raw API response after parsing:', parsed);
          
          // Handle the actual API response structure which has different fields
          if (parsed.clients && Array.isArray(parsed.clients)) {
            const enhancedClients: EnhancedClientData[] = parsed.clients.map((client: any, index: number) => {
              console.log('Processing client in API transform:', client);
              
              // Handle authentication_methods - can be string, array of strings, or array of objects
              let authMethodsArray: Array<{id: string; name: string; type: string; is_default: boolean; enabled: boolean}> = [];
              
              if (typeof client.authentication_methods === 'string') {
                // String format: "password" or "password,oidc"
                authMethodsArray = client.authentication_methods.split(',').map((method: string, methodIndex: number) => ({
                  id: `auth-${index}-${methodIndex}`,
                  name: method.trim(),
                  type: method.trim(),
                  is_default: methodIndex === 0, // First method is default
                  enabled: client.active ?? true,
                }));
              } else if (Array.isArray(client.authentication_methods)) {
                // Check if array contains strings or objects
                authMethodsArray = client.authentication_methods.map((method: any, methodIndex: number) => {
                  if (typeof method === 'string') {
                    // Array of strings: ["password", "oidc"]
                    return {
                      id: `auth-${index}-${methodIndex}`,
                      name: method,
                      type: method,
                      is_default: methodIndex === 0,
                      enabled: client.active ?? true,
                    };
                  } else if (typeof method === 'object' && method !== null) {
                    // Array of objects: [{id, name, type, ...}]
                    return {
                      id: method.id || `auth-${index}-${methodIndex}`,
                      name: method.name || method.type || 'Unknown',
                      type: method.type || method.name || 'Unknown',
                      is_default: method.is_default ?? (methodIndex === 0),
                      enabled: method.enabled ?? (client.active ?? true),
                    };
                  }
                  return {
                    id: `auth-${index}-${methodIndex}`,
                    name: 'Unknown',
                    type: 'Unknown',
                    is_default: methodIndex === 0,
                    enabled: client.active ?? true,
                  };
                });
              }

              return {
                // Map from actual API response fields
                id: client.id || client.client_id || `client-${index}`,
                client_id: client.client_id || `client-${index}`,
                tenant_id: client.tenant_id || '',
                project_id: client.project_id || '',
                owner_id: client.owner_id || null,
                org_id: client.org_id || null,
                name: client.name || client.client_name || 'Unnamed Client',
                status: client.status || (client.active ? 'Active' : 'Inactive'),
                email: client.email || null,
                tags: client.tags || null,
                active: client.active ?? true,
                mfa_enabled: client.mfa_enabled ?? true, // Default to true as per requirement
                mfa_method: client.mfa_method || null,
                mfa_verified: client.mfa_verified ?? false,
                roles: client.roles || null,
                oidc_enabled: client.oidc_enabled ?? false,
                hydra_client_id: client.hydra_client_id || null,
                created_at: client.created_at || new Date().toISOString(),
                updated_at: client.updated_at || new Date().toISOString(),
                
                // Enhanced fields
                authentication_methods: authMethodsArray,
                auth_methods_count: authMethodsArray.length,
                client_name: client.name || client.client_name || 'Unnamed Client',
                description: `Client: ${client.name || client.client_name || 'Unnamed Client'}`,
                user_count: client.user_count || 0,
                enabled: client.active ?? true,
              } as EnhancedClientData;
            });

            console.log('Enhanced clients after :', enhancedClients);

            return {
              clients: enhancedClients,
              total: parsed.pagination?.total || enhancedClients.length,
              filters: parsed.filters,
              pagination: parsed.pagination,
            };
          }
          
          // Fallback for old response format
          const normalizedResponse = normalizeClientsResponse(parsed);
          console.log('Normalized response (fallback):', normalizedResponse);
          console.log('First client in normalized response:', normalizedResponse.clients?.[0]);
          
          const enhancedClients: EnhancedClientData[] = normalizedResponse.clients.map(client => {
            console.log('Processing client in API transform (fallback):', client);
            return {
              ...client,
              authentication_methods: [],
              auth_methods_count: 0,
              client_name: client.name,
              description: `Client: ${client.name}`,
            };
          });

          return {
            clients: enhancedClients,
            total: enhancedClients.length,
            filters: normalizedResponse.filters,
            pagination: normalizedResponse.pagination,
          };
        } catch (error) {
          console.error("Failed to parse getAllClients response:", error);
          return {
            clients: [],
            total: 0,
            filters: {},
            pagination: { limit: 10, page: 1, total: 0 },
          };
        }
      },
      providesTags: (result) =>
        result?.clients && result.clients.length
          ? [
              ...result.clients.map((client) => ({
                type: "Client" as const,
                id: client.client_id ?? client.id ?? "UNKNOWN",
              })),
              { type: "Client" as const, id: "ALL" },
            ]
          : [{ type: "Client" as const, id: "ALL" }],
    }),

    // Register a new client (Tenant-scoped)
    registerClient: builder.mutation<RegisterClientResponse, RegisterClientRequest>({
      query: (data) => {
        const tenantId = data.tenant_id;
        return {
          url: `/authsec/clientms/tenants/${tenantId}/clients/create`,
          method: "POST",
          body: data,
          responseHandler: "text", // Get raw text to handle multiple JSON objects
        };
      },
      transformResponse: (response: string) => {
        // Backend may send multiple JSON objects concatenated
        // Parse only the first valid one (success response)
        return parseFirstValidJSON<RegisterClientResponse>(response);
      },
      invalidatesTags: [{ type: "Client", id: "LIST" }],
    }),

    // Add OIDC provider to a client
    addOIDCProvider: builder.mutation<AddProviderResponse, AddProviderRequest>({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/add-provider",
        method: "POST",
        body: data,
      }),
      invalidatesTags: ["Client", "OIDCProvider"],
    }),

    // Get OIDC configuration for a tenant
    getOIDCConfig: builder.mutation<GetConfigResponse, GetConfigRequest>({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/get-config", 
        method: "POST",
        body: data,
      }),
      invalidatesTags: ["OIDCProvider"],
    }),

    // Legacy endpoints for backward compatibility
    createClient: builder.mutation<Client, Partial<Client>>({
      queryFn: async () => {
        return { error: { status: 501, data: "Use registerClient instead" } };
      },
      invalidatesTags: ["Client"],
    }),

    updateClient: builder.mutation<Client, { id: string; updates: Partial<Client> }>({
      queryFn: async () => {
        return { error: { status: 501, data: "Not implemented in AuthSec API" } };
      },
      invalidatesTags: ["Client"],
    }),

    deleteClient: builder.mutation<void, { id: string }>({
      queryFn: async () => {
        return { error: { status: 501, data: "Not implemented in AuthSec API" } };
      },
      invalidatesTags: ["Client"],
    }),

    attachAuthMethod: builder.mutation<
      void,
      { clientId: string; authMethodId: string; isDefault?: boolean }
    >({
      queryFn: async () => {
        return { error: { status: 501, data: "Use addOIDCProvider instead" } };
      },
      invalidatesTags: ["Client"],
    }),

    detachAuthMethod: builder.mutation<void, { clientId: string; authMethodId: string }>({
      queryFn: async () => {
        return { error: { status: 501, data: "Not implemented in AuthSec API" } };
      },
      invalidatesTags: ["Client"],
    }),

    setDefaultAuthMethod: builder.mutation<void, { clientId: string; authMethodId: string }>({
      queryFn: async () => {
        return { error: { status: 501, data: "Not implemented in AuthSec API" } };
      },
      invalidatesTags: ["Client"],
    }),

    getClientAuthMethods: builder.query<AuthMethod[], { clientId: string }>({
      queryFn: async () => {
        return { data: [] };
      },
      providesTags: ["Client"],
    }),
  }),
});

export const {
  useGetClientsQuery,
  useGetAllClientsQuery,
  useRegisterClientMutation,
  useDeleteClientCompleteMutation,
  useSetClientStatusMutation,
  useAddOIDCProviderMutation,
  useGetOIDCConfigMutation,
  
  // Legacy hooks for backward compatibility
  useCreateClientMutation,
  useUpdateClientMutation,
  useDeleteClientMutation,
  useAttachAuthMethodMutation,
  useDetachAuthMethodMutation,
  useSetDefaultAuthMethodMutation,
  useGetClientAuthMethodsQuery,
} = clientApi;
// Force rebuild
