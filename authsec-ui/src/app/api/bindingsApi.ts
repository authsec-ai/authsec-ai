import { baseApi, withSessionData } from "./baseApi";

// RBAC Audience type
export type RbacAudience = "admin" | "endUser";

// Request/Response types
export interface ScopeBinding {
  id?: string;
  type?: string;
}

export interface CreateBindingRequest {
  user_id: string;
  role_id: string;
  scope?: ScopeBinding;
  conditions?: Record<string, any>;
  audience?: RbacAudience;
}

export interface BindingResponse {
  id: string;
  role_name: string;
  scope_description?: string;
  status: string;
}

// List bindings query parameters
export interface ListBindingsParams {
  user_id?: string;
  role_id?: string;
  scope_type?: string;
  audience: RbacAudience;
}

// Role binding interface for list response
export interface RoleBinding {
  id: string;
  user_id?: string;
  username?: string;
  email?: string;
  role_id: string;
  role_name: string;
  scope_id?: string;
  scope_type?: string;
  service_account_id?: string;
  conditions?: Record<string, any>;
  created_at: string;
  expires_at?: string;
}

/**
 * Bindings API
 * Handles role binding assignments (user + role + scope)
 * Supports both admin and end-user audiences with separate endpoints
 */
export const bindingsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // List all role bindings with optional filters
    listBindings: builder.query<RoleBinding[], ListBindingsParams>({
      query: (params) => {
        const { audience, ...restParams } = params;
        const queryParams = new URLSearchParams();

        if (restParams.user_id) queryParams.append("user_id", restParams.user_id);
        if (restParams.role_id) queryParams.append("role_id", restParams.role_id);
        if (restParams.scope_type) queryParams.append("scope_type", restParams.scope_type);

        const queryString = queryParams.toString();

        // Route based on audience
        const baseUrl = audience === "admin"
          ? "authsec/uflow/admin/bindings"
          : "authsec/uflow/user/rbac/bindings";

        return {
          url: `${baseUrl}${queryString ? `?${queryString}` : ""}`,
          method: "GET",
        };
      },
      providesTags: ["AdminUser", "AdminRBACRole", "AdminRBACScope"],
    }),

    createBinding: builder.mutation<BindingResponse, CreateBindingRequest>({
      query: ({ audience = "admin", ...data }) => {
        // Route based on audience
        const url = audience === "admin"
          ? "authsec/uflow/admin/bindings"
          : "authsec/uflow/user/rbac/bindings";

        return {
          url,
          method: "POST",
          body: withSessionData(data),
        };
      },
      invalidatesTags: ["AdminUser", "AdminRBACRole", "AdminRBACScope"],
    }),
  }),
});

export const {
  useListBindingsQuery,
  useCreateBindingMutation
} = bindingsApi;
