import { baseApi } from "./baseApi";
import type { ExternalService } from "@/types/entities";
import { endUserResourcesApi } from "./enduser/resourcesApi";

// Raw backend service shape (from /exsvc/services endpoints)
export interface RawExternalService {
  id: string;
  name: string;
  type: string; // e.g. "API"
  url: string;
  description?: string;
  tags?: string[];
  resource_id: number;
  auth_type: "oauth2" | "api_key" | "basic_auth" | "bearer_token" | "none" | string;
  auth_config?: string;
  vault_path?: string;
  created_by?: string;
  agent_accessible: boolean;
  secret_data?: Record<string, any>;
  created_at: string;
  updated_at: string;
}

export interface ExternalServiceRequest {
  name: string;
  type?: string; // API default
  url: string;
  description?: string;
  tags?: string[];
  resource_id: number;
  auth_type: "oauth2" | "api_key" | "basic_auth" | "bearer_token" | "none";
  agent_accessible?: boolean;
  secret_data?: Record<string, any>;
}

export interface ExternalServiceUpdateRequest {
  name?: string;
  type?: string;
  url?: string;
  description?: string;
  tags?: string[];
  auth_type?: "oauth2" | "api_key" | "basic_auth" | "bearer_token" | "none";
  agent_accessible?: boolean;
  secret_data?: Record<string, any>;
}

// Map raw backend service to UI ExternalService type
const deriveProviderFromUrl = (url?: string): string => {
  if (!url) return "custom";
  try {
    const host = new URL(url).hostname.toLowerCase();
    const parts = host.split(".").filter((p) => !["api", "www", "dev", "staging", "v1", "v2"].includes(p));
    // Prefer well-known providers if present
    const known = ["google", "microsoft", "salesforce", "slack", "github", "stripe", "dropbox", "box", "aws", "azure"];
    const hit = parts.find((p) => known.includes(p));
    return hit || parts[0] || "custom";
  } catch {
    return "custom";
  }
};

const mapRawToExternalService = (raw: RawExternalService): ExternalService => {
  const provider = deriveProviderFromUrl(raw.url);
  const status: ExternalService["status"] =
    raw.auth_type === "none"
      ? "needs_consent"
      : raw.auth_type === "oauth2"
      ? "needs_consent"
      : "connected";

  return {
    id: raw.id,
    name: raw.name,
    provider,
    // Use backend type if reasonable; fallback to 'other'
    category: (raw.type || "other").toString().toLowerCase(),
    clientCount: 0,
    userTokenCount: 0,
    status,
    lastSync: raw.updated_at || raw.created_at,
    lastError: undefined,
    createdAt: raw.created_at,
  };
};

export const externalServiceApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // GET /exsvc/services - list services (response shape: { services: RawExternalService[] })
    getExternalServices: builder.query<RawExternalService[], void>({
      query: () => ({ url: "/authsec/exsvc/services", method: "GET" }),
      transformResponse: (response: { services: RawExternalService[] }) => {
        const items = Array.isArray((response as any)?.services) ? (response as any).services : [];
        return items;
      },
      providesTags: (result) =>
        result
          ? [
              ...result.map(({ id }) => ({ type: "ExternalService" as const, id })),
              { type: "ExternalService", id: "LIST" },
            ]
          : [{ type: "ExternalService", id: "LIST" }],
    }),

    // GET /exsvc/services/{id}
    getExternalService: builder.query<ExternalService, string>({
      query: (id) => ({ url: `/authsec/exsvc/services/${id}`, method: "GET" }),
      transformResponse: (raw: RawExternalService) => mapRawToExternalService(raw),
      providesTags: (_res, _err, id) => [{ type: "ExternalService", id }],
    }),

    // GET /exsvc/services/{id}/credentials - fetch service credentials (requires MFA)
    getExternalServiceCredentials: builder.query<Record<string, any>, string>({
      query: (id) => ({ url: `/authsec/exsvc/services/${id}/credentials`, method: "GET" }),
      providesTags: (_res, _err, id) => [{ type: "ExternalService", id: `${id}-credentials` }],
    }),

    // POST /exsvc/services
    createExternalService: builder.mutation<
      ExternalService,
      ExternalServiceRequest & { tenant_id?: string; auto_create_resource?: boolean }
    >({
      queryFn: async (arg, api, _extraOptions, baseQuery) => {
        const { tenant_id, auto_create_resource = true, ...body } = arg;

        // If auto_create_resource is true and tenant_id is provided, create the resource first
        if (auto_create_resource && tenant_id) {
          try {
            const resourceResult = await api.dispatch(
              endUserResourcesApi.endpoints.createEndUserResource.initiate({
                tenant_id,
                data: {
                  name: `${body.name} Resource`,
                  description: `Auto-generated resource for external service: ${body.name}`,
                },
              })
            );

            if ("error" in resourceResult) {
              return { error: resourceResult.error as any };
            }

            // Get the created resource ID
            const createdResource = resourceResult.data?.resources?.[0];
            if (createdResource) {
              body.resource_id = parseInt(createdResource.id);
            }
          } catch (error) {
            console.error("Failed to auto-create resource:", error);
            // Continue with service creation even if resource creation fails
          }
        }

        // Create the external service
        const result = await baseQuery({
          url: "/authsec/exsvc/services",
          method: "POST",
          body,
        });

        if (result.error) {
          return { error: result.error as any };
        }

        return {
          data: mapRawToExternalService(result.data as RawExternalService),
        };
      },
      invalidatesTags: [{ type: "ExternalService", id: "LIST" }, "EndUserRBACResource"],
    }),

    // PATCH /exsvc/services/{id}
    updateExternalService: builder.mutation<
      ExternalService,
      { id: string; body: ExternalServiceUpdateRequest }
    >({
      query: ({ id, body }) => ({ url: `/authsec/exsvc/services/${id}`, method: "PATCH", body }),
      transformResponse: (raw: RawExternalService) => mapRawToExternalService(raw),
      invalidatesTags: (_r, _e, arg) => [
        { type: "ExternalService", id: arg.id },
        { type: "ExternalService", id: "LIST" },
      ],
    }),

    // DELETE /exsvc/services/{id}
    deleteExternalService: builder.mutation<
      { success: boolean; id: string },
      string | { id: string; tenant_id?: string; resource_id?: number; auto_delete_resource?: boolean }
    >({
      queryFn: async (arg, api, _extraOptions, baseQuery) => {
        // Handle both string ID and object with options
        const id = typeof arg === "string" ? arg : arg.id;
        const tenant_id = typeof arg === "object" ? arg.tenant_id : undefined;
        const resource_id = typeof arg === "object" ? arg.resource_id : undefined;
        const auto_delete_resource = typeof arg === "object" ? arg.auto_delete_resource !== false : true;

        // Delete the external service first
        const result = await baseQuery({
          url: `/authsec/exsvc/services/${id}`,
          method: "DELETE",
        });

        if (result.error) {
          return { error: result.error as any };
        }

        // If auto_delete_resource is true and we have tenant_id and resource_id, delete the resource
        if (auto_delete_resource && tenant_id && resource_id) {
          try {
            await api.dispatch(
              endUserResourcesApi.endpoints.deleteEndUserResource.initiate({
                tenant_id,
                resource_id: String(resource_id),
              })
            );
          } catch (error) {
            console.error("Failed to auto-delete resource:", error);
            // Service is already deleted, so we consider this a success
          }
        }

        return {
          data: { success: true, id },
        };
      },
      invalidatesTags: (_r, _e, arg) => {
        const id = typeof arg === "string" ? arg : arg.id;
        return [
          { type: "ExternalService", id },
          { type: "ExternalService", id: "LIST" },
          "EndUserRBACResource",
        ];
      },
    }),
  }),
  overrideExisting: true,
});

export const {
  useGetExternalServicesQuery,
  useGetExternalServiceQuery,
  useGetExternalServiceCredentialsQuery,
  useLazyGetExternalServiceCredentialsQuery,
  useCreateExternalServiceMutation,
  useUpdateExternalServiceMutation,
  useDeleteExternalServiceMutation,
} = externalServiceApi;
