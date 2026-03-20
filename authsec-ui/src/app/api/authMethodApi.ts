import { baseApi } from "./baseApi";
import type {
  AuthMethod,
  AuthMethodWithStats,
  AuthMethodFilters,
  AuthMethodAnalytics,
  ListParams,
} from "@/types/database";

// OIDC Provider Types
export interface OidcProviderConfig {
  provider_name: string;
  display_name: string;
  client_id: string;
  client_secret: string;
  auth_url: string;
  token_url: string;
  user_info_url: string;
  scopes: string[];
  issuer_url?: string;
  jwks_url?: string;
  additional_params?: Record<string, any>;
  is_active: boolean;
  sort_order?: number;
}

export interface TenantClientConfig {
  client_name: string;
  redirect_uris: string[];
  scopes?: string[];
  grant_types?: string[];
}

export interface OidcProviderRequest {
  tenant_id: string;
  org_id: string;
  provider: OidcProviderConfig;
  react_app_url: string;
  created_by: string;
}

export interface GenerateLoginUrlRequest {
  tenant_id: string;
  org_id: string;
  redirect_uri: string;
  state: string;
}

export interface GenerateLoginUrlResponse {
  success: boolean;
  instructions: string;
  login_endpoint: string;
  oauth_url: string;
  pkce: {
    code_challenge: string;
    code_verifier: string;
    method: string;
  };
  tenant_client_id: string;
}

export interface OidcProviderResponse {
  success: boolean;
  message: string;
  data?: any;
}

export interface OidcConfigRequest {
  tenant_id: string;
  client_id?: string; // Optional - filter by client_id if provided
}

export interface OidcConfigResponse {
  success: boolean;
  message: string;
  data: {
    oidc_providers: Array<{
      provider_name: string;
      display_name: string;
      client_id: string;
      callback_url: string;
      provider_config: {
        client_id: string;
        client_secret: string;
        auth_url: string;
        token_url: string;
        user_info_url: string;
        scopes: string[];
        issuer_url?: string;
        jwks_url?: string;
        additional_params?: any;
      };
      is_active: boolean;
      created_at: string;
      sort_order: number;
    }> | null;
    tenant_id: string;
    org_id?: string;
    provider_count: number;
    tenant_client?: TenantClientConfig & {
      client_id?: string;
      created_at?: string;
    };
  };
  timestamp: string;
}

export interface UpdateCompleteTenantRequest {
  tenant_id: string;
  org_id: string;
  client_id?: string; // Optional - if provided, updates only for this client
  tenant_name?: string;
  tenant_client?: TenantClientConfig;
  oidc_providers?: OidcProviderConfig[];
  updated_by?: string;
}

export interface OocmgrMessageResponse {
  message?: string;
  success?: boolean;
  data?: any;
  timestamp?: string;
  request_id?: string;
}

// Show Auth Providers Types (New API)
export interface ShowAuthProvidersRequest {
  tenant_id: string;
  client_id?: string; // Optional - sent as header if provided
}

export interface ShowAuthProvidersResponse {
  success: boolean;
  message: string;
  data: {
    tenant_id: string;
    client_id?: string;
    count: number;
    providers: Array<{
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
    }>;
  };
  timestamp: string;
}

// Edit Client Auth Provider Types (New API)
export interface EditClientAuthProviderRequest {
  tenant_id: string;
  client_id: string;
  provider_name: string;
  display_name: string;
  is_active: boolean;
  callback_url: string;
  provider_config: {
    auth_url: string;
    token_url: string;
    user_info_url?: string;
  };
  updated_by?: string;
}

export interface EditClientAuthProviderResponse {
  success: boolean;
  message: string;
  data: {
    client_id: string;
    provider_name: string;
    updated_clients: string[];
    updated_fields: {
      provider_name: string;
      display_name: string;
      callback_url: string;
      is_active: boolean;
      provider_config: {
        auth_url: string;
        token_url: string;
        user_info_url?: string;
      };
    };
  };
  timestamp: string;
}

// Update Provider Types (New API)
export interface UpdateProviderRequest {
  tenant_id: string;
  org_id: string;
  provider_name: string;
  display_name: string;
  client_id: string;
  client_secret: string;
  auth_url: string;
  token_url: string;
  user_info_url: string;
  scopes: string[];
  is_active: boolean;
  updated_by: string;
}

export interface UpdateProviderResponse {
  success: boolean;
  message: string;
  data?: any;
  timestamp: string;
}

// Delete Provider Types (New API)
export interface DeleteProviderRequest {
  tenant_id: string;
  client_id: string;
  provider_name: string;
}

export interface DeleteProviderResponse {
  success: boolean;
  message: string;
  timestamp: string;
}

export interface RawHydraDumpRequest {
  tenant_id: string;
  client_type: string;
  provider_name: string;
}

export interface RawHydraDumpResponse {
  success: boolean;
  message?: string;
  data?: Record<string, any>;
  timestamp?: string;
}

export const authMethodApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Authentication Methods CRUD
    getAuthMethods: builder.query<AuthMethod[], AuthMethodFilters>({
      query: (params = {}) => {
        const searchParams = new URLSearchParams();
        if (params.search) searchParams.append("search", params.search);
        if (params.status) searchParams.append("status", params.status);
        if (params.method_type)
          searchParams.append("method_type", params.method_type);
        if (params.limit) searchParams.append("limit", params.limit.toString());
        if (params.offset)
          searchParams.append("offset", params.offset.toString());

        return `auth-methods?${searchParams.toString()}`;
      },
      providesTags: ["AuthMethod"],
    }),

    getAuthMethod: builder.query<AuthMethod, string>({
      query: (id) => `auth-methods/${id}`,
      providesTags: (result, error, id) => [{ type: "AuthMethod", id }],
    }),

    createAuthMethod: builder.mutation<AuthMethod, Partial<AuthMethod>>({
      query: (data) => ({
        url: "auth-methods",
        method: "POST",
        body: data,
      }),
      invalidatesTags: ["AuthMethod"],
    }),

    updateAuthMethod: builder.mutation<
      AuthMethod,
      { id: string; data: Partial<AuthMethod> }
    >({
      query: ({ id, data }) => ({
        url: `auth-methods/${id}`,
        method: "PUT",
        body: data,
      }),
      invalidatesTags: (result, error, { id }) => [{ type: "AuthMethod", id }],
    }),

    deleteAuthMethod: builder.mutation<void, string>({
      query: (id) => ({
        url: `auth-methods/${id}`,
        method: "DELETE",
      }),
      invalidatesTags: ["AuthMethod"],
    }),

    // Auth Method Stats
    getAuthMethodStats: builder.query<any, string>({
      query: (projectId) => `auth-methods/stats?project_id=${projectId}`,
      providesTags: ["AuthMethod"],
    }),

    // Auth Method Analytics
    getAuthMethodAnalytics: builder.query<AuthMethodAnalytics, string>({
      query: (projectId) => `auth-methods/analytics?project_id=${projectId}`,
      providesTags: ["AuthMethod"],
    }),

    // Toggle auth method status
    toggleAuthMethodStatus: builder.mutation<AuthMethod, string>({
      query: (id) => ({
        url: `auth-methods/${id}/toggle-status`,
        method: "POST",
      }),
      invalidatesTags: (result, error, id) => [
        { type: "AuthMethod", id },
        "AuthMethod",
      ],
    }),

    // Get auth methods by project
    getAuthMethodsByProject: builder.query<AuthMethod[], string>({
      query: (projectId) => `projects/${projectId}/auth-methods`,
      providesTags: ["AuthMethod"],
    }),

    // OIDC Provider Endpoints
    addOidcProvider: builder.mutation<
      OidcProviderResponse,
      OidcProviderRequest
    >({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/add-provider",
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: data,
      }),
      invalidatesTags: ["AuthMethodOIDCProvider"],
    }),

    getOidcConfig: builder.query<OidcConfigResponse, OidcConfigRequest>({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/get-config",
        method: "POST",
        body: data,
      }),
      providesTags: (result, error, arg) =>
        // Provide specific tags based on whether client_id is in the request
        arg.client_id
          ? [
              { type: "AuthMethodOIDCProvider", id: arg.client_id },
              { type: "AuthMethodOIDCProvider", id: "LIST" },
            ]
          : [{ type: "AuthMethodOIDCProvider", id: "LIST" }],
    }),

    updateCompleteTenantConfig: builder.mutation<
      OocmgrMessageResponse,
      UpdateCompleteTenantRequest
    >({
      query: (data) => ({
        url: "/authsec/oocmgr/tenant/update-complete",
        method: "POST",
        body: data,
      }),
      invalidatesTags: (result, error, arg) => {
        // Only invalidate the specific client's cache if client_id is provided
        // Otherwise invalidate all OIDC provider caches
        if (arg.client_id) {
          return [
            { type: "AuthMethodOIDCProvider", id: arg.client_id },
            { type: "Client", id: arg.client_id },
            "Client", // Also invalidate client list to refresh the table
          ];
        }
        return ["AuthMethodOIDCProvider", "Client"];
      },
    }),

    // New OIDC Provider Management Endpoints
    showAuthProviders: builder.query<
      ShowAuthProvidersResponse,
      ShowAuthProvidersRequest
    >({
      query: (data) => {
        console.log("[showAuthProviders] Request data:", data);

        const headers: Record<string, string> = {
          "Content-Type": "application/json",
        };

        // ALWAYS add Client-Id header (required by backend)
        // Use provided client_id or empty string if not available
        const clientIdValue = data.client_id || "";
        headers["Client-Id"] = clientIdValue;

        // Always include client_id in body as well (fallback for CORS issues)
        const requestBody = {
          tenant_id: data.tenant_id,
          client_id: clientIdValue,
        };

        console.log("[showAuthProviders] Sending request:", {
          url: "/authsec/oocmgr/oidc/show-auth-providers",
          headers,
          body: requestBody,
        });

        return {
          url: "/authsec/oocmgr/oidc/show-auth-providers",
          method: "POST",
          headers,
          body: requestBody,
        };
      },
      providesTags: (result, error, arg) =>
        // Provide specific tags based on whether client_id is in the request
        arg.client_id
          ? [
              { type: "AuthMethodOIDCProvider", id: arg.client_id },
              { type: "AuthMethodOIDCProvider", id: "LIST" },
            ]
          : [{ type: "AuthMethodOIDCProvider", id: "LIST" }],
    }),

    editClientAuthProvider: builder.mutation<
      EditClientAuthProviderResponse,
      EditClientAuthProviderRequest
    >({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/edit-client-auth-provider",
        method: "POST",
        body: data,
      }),
      invalidatesTags: (result, error, arg) => {
        // Only invalidate the specific client's cache if client_id is provided
        // Otherwise invalidate all OIDC provider caches
        if (arg.client_id) {
          return [
            { type: "AuthMethodOIDCProvider", id: arg.client_id },
            { type: "Client", id: arg.client_id },
            "Client", // Also invalidate client list to refresh the table
          ];
        }
        return ["AuthMethodOIDCProvider", "Client"];
      },
    }),

    updateProvider: builder.mutation<
      UpdateProviderResponse,
      UpdateProviderRequest
    >({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/update-provider",
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: data,
      }),
      invalidatesTags: ["OIDCProvider", "Client"],
    }),

    deleteProvider: builder.mutation<
      DeleteProviderResponse,
      DeleteProviderRequest
    >({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/delete-provider",
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: data,
      }),
      invalidatesTags: ["OIDCProvider", "Client"],
    }),

    rawHydraDump: builder.query<RawHydraDumpResponse, RawHydraDumpRequest>({
      query: (data) => ({
        url: "/authsec/oocmgr/oidc/raw-hydra-dump",
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        body: data,
      }),
      providesTags: ["AuthMethodOIDCProvider"],
    }),
  }),
});

export const {
  useGetAuthMethodsQuery,
  useGetAuthMethodQuery,
  useCreateAuthMethodMutation,
  useUpdateAuthMethodMutation,
  useDeleteAuthMethodMutation,
  useGetAuthMethodStatsQuery,
  useGetAuthMethodAnalyticsQuery,
  useToggleAuthMethodStatusMutation,
  useGetAuthMethodsByProjectQuery,
  useAddOidcProviderMutation,
  useGetOidcConfigQuery,
  useUpdateCompleteTenantConfigMutation,
  useShowAuthProvidersQuery,
  useEditClientAuthProviderMutation,
  useUpdateProviderMutation,
  useDeleteProviderMutation,
  useRawHydraDumpQuery,
  useLazyRawHydraDumpQuery,
} = authMethodApi;
