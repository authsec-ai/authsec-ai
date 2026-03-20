/**
 * OIDC/OAuth API - Pure OIDC operations and token exchange
 * Updated to support universal callback URL approach
 * Handles OAuth flows and final token exchange operations
 */

import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import config from '../../config';

export interface OIDCTokenExchangeRequest {
  login_challenge: string;
  code: string;
  state: string;
  provider: string;
  redirect_uri: string;
}

export interface OIDCTokenExchangeResponse {
  success: boolean;
  tokens?: {
    access_token: string;
    token_type: string;
    expires_in: number;
    refresh_token?: string;
  };
  error?: string;
}

export interface OIDCLoginRequest {
  access_token: string;
  expires_in: number;
}

export interface OIDCLoginResponse {
  success: boolean;
  data?: {
    tenant_id: string;
    email: string;
    first_login: boolean;
  };
  error?: string;
}

// Updated: Removed provider from CallbackRequest since it comes from state
export interface CallbackRequest {
  code: string;
  state: string;
  error?: string;
}

export interface CallbackResponse {
  success: boolean;
  redirect_to?: string;
  user_info?: any;
  error?: string;
}

export interface OIDCProvider {
  provider_name: string;
  display_name: string;
  is_active: boolean;
  sort_order: number;
  callback_url: string;
  config: Record<string, any>;
}

export interface LoginPageData {
  success: boolean;
  login_challenge: string;
  tenant_name: string;
  client_name: string;
  client_id: string;
  providers: OIDCProvider[];
  base_url: string;
  error?: string;
}

export interface AuthInitiateRequest {
  login_challenge: string;
}

// add optional SAML fields
export interface AuthInitiateResponse {
  success: boolean;
  auth_url: string;
  state: string;
  provider: string;
  error?: string;

  // SAML-specific (optional)
  sso_url?: string; // e.g., Okta SSO URL
  method?: "GET" | "POST"; // binding hint from backend (if present)
  form_params?: Record<string, string>; // e.g., { SAMLRequest, RelayState, SigAlg, Signature }
}

export interface CustomLoginStatusRequest {
  client_id: string;
  email: string;
  tenant_domain?: string;
}

export interface CustomLoginStatusResponse {
  success: boolean;
  response: boolean | string;
  error?: string;
}

export interface CustomLoginRegisterRequest {
  client_id: string;
  email: string;
  password: string;
  name: string;
  tenant_domain?: string;
}

export interface CustomLoginRegisterResponse {
  success: boolean;
  message?: string;
  email?: string;
  error?: string;
}

export interface CustomLoginRegisterCompleteRequest {
  client_id: string;
  email: string;
  otp: string;
}

export interface CustomLoginRegisterCompleteResponse {
  success: boolean;
  message?: string;
  email?: string;
  error?: string;
}

export interface SamlLoginRequest {
  client_id: string;
  email: string;
}

export interface SamlLoginResponse {
  tenant_id: string;
  email: string;
  first_login: boolean;
  otp_required: boolean;
  mfa_required: boolean;
}

// UFlow OAuth Provider Interfaces
export interface UFlowOIDCProvider {
  provider_name: string;
  display_name: string;
  icon_url: string;
}

export interface UFlowOIDCProvidersResponse {
  providers: UFlowOIDCProvider[];
}

export interface UFlowOIDCInitiateRequest {
  provider: string;
}

export interface UFlowOIDCInitiateResponse {
  action: string;
  redirect_url: string;
  state: string;
}

export interface UFlowOIDCCallbackData {
  email: string;
  message?: string;
  name: string;
  needs_domain: boolean;
  picture: string;
  provider: string;
  provider_user_id: string;
  success: boolean;
  // For existing users
  client_id?: string;
  tenant_domain?: string;
  tenant_id?: string;
}

export interface TenantDomainCheckResponse {
  action: string;
  domain: string;
  exists: boolean;
}

export interface CompleteUFlowOIDCRegistrationRequest {
  tenant_domain: string;
  provider: string;
  email: string;
  name: string;
  picture: string;
  provider_user_id: string;
}

export interface CompleteUFlowOIDCRegistrationResponse {
  client_id: string;
  message: string;
  success: boolean;
  tenant_domain: string;
  tenant_id: string;
}

export interface AdminOIDCExchangeRequest {
  code: string;
  state: string;
}

export interface AdminOIDCExchangeSuccessResponse {
  tenant_id: string;
  email: string;
  first_login: boolean;
  otp_required: boolean;
  mfa_required: boolean;
  tenant_domain?: string;
  client_id?: string;
}

export interface AdminOIDCExchangeErrorResponse {
  error?: string;
  message?: string;
  needs_domain?: boolean;
  provider_data?: Omit<CompleteUFlowOIDCRegistrationRequest, "tenant_domain">;
}

// Helper function to get session data
const getSessionData = () => {
  const sessionData = localStorage.getItem("authsec_session_v2");
  if (sessionData) {
    try {
      return JSON.parse(sessionData);
    } catch {
      console.error("Session data parsing failed - invalid JSON format");
    }
  }
  return null;
};

// RTK Query API for OIDC/OAuth operations
export const oidcApi = createApi({
  reducerPath: "oidcApi",
  baseQuery: fetchBaseQuery({
    baseUrl: config.VITE_API_URL || "https://test.api.authsec.dev",
    timeout: 30000,
    credentials: "include",
    prepareHeaders: (headers) => {
      const session = getSessionData();
      if (session?.token) {
        headers.set("Authorization", `Bearer ${session.token}`);
      }
      if (!headers.has("Content-Type")) {
        headers.set("Content-Type", "application/json");
      }
      return headers;
    },
  }),
  tagTypes: ["OIDC", "Token"],
  endpoints: (builder) => ({
    // Exchange OAuth code for tokens (Hydra flow)
    exchangeCodeForTokens: builder.mutation<OIDCTokenExchangeResponse, OIDCTokenExchangeRequest>({
      query: (data) => ({
        url: "/authsec/hmgr/auth/exchange-token",
        method: "POST",
        body: data,
      }),
      transformResponse: (res: any): OIDCTokenExchangeResponse => {
        // Normalize backend variations into a consistent tokens shape
        if (res?.tokens?.access_token) {
          return res as OIDCTokenExchangeResponse;
        }
        const access_token = res?.access_token || res?.id_token || res?.jwt || res?.jwt_token;
        const expires_in = res?.expires_in ?? res?.token?.expires_in ?? 3600;
        const token_type = res?.token_type || "Bearer";
        const refresh_token = res?.refresh_token || "";
        return {
          success: typeof res?.success === "boolean" ? res.success : true,
          tokens: access_token
            ? { access_token, token_type, expires_in, refresh_token }
            : undefined,
          error: res?.error,
        };
      },
    }),

    // UPDATED: Universal callback handler - no provider in URL path
    // Provider information is extracted from the state parameter on the backend
    handleCallback: builder.mutation<CallbackResponse, CallbackRequest>({
      query: (data) => ({
        url: "/authsec/hmgr/auth/callback", // Universal callback URL - no provider parameter
        method: "POST",
        body: data,
        credentials: "include",
      }),
    }),

    // Get login page data
    getLoginPageData: builder.query<
      LoginPageData,
      { login_challenge: string; extraQuery?: string }
    >({
      query: ({ login_challenge, extraQuery }) => {
        const qs = new URLSearchParams((extraQuery || "").replace(/^\?/, ""));
        // Ensure login_challenge is present/overrides any duplicate
        if (login_challenge) qs.set("login_challenge", login_challenge);
        const queryString = qs.toString();

        return {
          url: `/authsec/hmgr/login/page-data${queryString ? `?${queryString}` : ""}`,
          method: "GET",
          credentials: "include",
        };
      },
      transformResponse: (res: any): LoginPageData => {
        const providers = Array.isArray(res?.providers) ? res.providers : [];
        const success = typeof res?.success === "boolean" ? res.success : true;
        const error = res?.error || undefined;

        let clientId = res?.client_id || "";
        if (clientId && clientId.endsWith("-main-client")) {
          clientId = clientId.replace("-main-client", "");
        }

        return {
          success,
          login_challenge: res?.login_challenge || res?.challenge || "",
          tenant_name: res?.tenant_name || res?.tenant || "Tenant",
          client_name: res?.client_name || res?.client || "Client",
          client_id: clientId,
          providers,
          base_url: res?.base_url || "",
          error,
        };
      },
      providesTags: ["OIDC"],
    }),
    // Initiate OAuth authentication
    // UPDATED: Now uses universal callback URL in the auth initiation
    initiateAuth: builder.mutation<
      AuthInitiateResponse,
      { provider: string; login_challenge: string; isSaml?: boolean; extraQuery?: string }
    >({
      query: ({ provider, login_challenge, isSaml, extraQuery }) => {
        const qs = new URLSearchParams((extraQuery || "").replace(/^\?/, ""));
        if (login_challenge) qs.set("login_challenge", login_challenge);
        const suffix = qs.toString() ? `?${qs.toString()}` : "";

        return {
          url:
            (isSaml ? `/authsec/hmgr/saml/initiate/${provider}` : `/authsec/hmgr/auth/initiate/${provider}`) +
            suffix,
          method: "POST",
          body: { login_challenge }, // keep for compatibility
          credentials: "include",
        };
      },
      transformResponse: (res: any): AuthInitiateResponse => {
        const sso_url = res?.sso_url || res?.ssoUrl || res?.ssourl;
        const auth_url = sso_url || res?.auth_url || res?.url || res?.redirect || "";
        const state = res?.state || res?.nonce || "";
        const providerName = res?.provider || res?.idp || "";

        // detect POST-binding payload if provided
        const methodRaw = (res?.method || res?.http_method || "").toString().toUpperCase();
        const method: "GET" | "POST" = methodRaw === "POST" ? "POST" : "GET";

        const form_params =
          res?.form_params ||
          (res?.SAMLRequest
            ? {
                SAMLRequest: res.SAMLRequest,
                ...(res?.RelayState ? { RelayState: res.RelayState } : {}),
                ...(res?.SigAlg ? { SigAlg: res.SigAlg } : {}),
                ...(res?.Signature ? { Signature: res.Signature } : {}),
              }
            : undefined);

        const success = typeof res?.success === "boolean" ? res.success : Boolean(auth_url);

        return {
          success,
          auth_url,
          sso_url,
          state,
          provider: providerName,
          method,
          form_params,
          error: res?.error,
        };
      },
    }),

    // Check custom login user status
    checkCustomLoginStatus: builder.mutation<CustomLoginStatusResponse, CustomLoginStatusRequest>({
      query: (data) => ({
        url: "/authsec/uflow/user/login/status",
        method: "POST",
        body: data,
      }),
      transformResponse: (res: any): CustomLoginStatusResponse => {
        const responseVal = res?.response;
        const normalizedResponse =
          typeof responseVal === "boolean"
            ? responseVal
            : typeof responseVal === "string"
            ? responseVal.toLowerCase() === "true"
            : Boolean(responseVal);

        const success = typeof res?.success === "boolean" ? res.success : true;
        const error = res?.error;

        return {
          success,
          response: normalizedResponse,
          error,
        };
      },
    }),

    // Register custom login user
    registerCustomUser: builder.mutation<CustomLoginRegisterResponse, CustomLoginRegisterRequest>({
      query: (data) => ({
        url: "/authsec/uflow/user/register/initiate",
        method: "POST",
        body: data,
      }),
      transformResponse: (res: any): CustomLoginRegisterResponse => {
        const message: string | undefined = res?.message;
        const hasPositiveMessage =
          typeof message === "string" && /(initiated|otp|verification|success)/i.test(message);
        const hasEmail = typeof res?.email === "string" && res.email.length > 3;
        const success =
          typeof res?.success === "boolean" ? res.success : hasPositiveMessage || hasEmail;
        const error =
          res?.error || (!success ? message || "Failed to initiate registration" : undefined);
        return {
          success,
          message,
          email: res?.email,
          error,
        };
      },
    }),
    completeCustomUserRegistration: builder.mutation<
      CustomLoginRegisterCompleteResponse,
      CustomLoginRegisterCompleteRequest
    >({
      query: (data) => ({
        url: "/authsec/uflow/user/register/complete",
        method: "POST",
        body: data,
      }),
      transformResponse: (res: any): CustomLoginRegisterCompleteResponse => {
        const message: string | undefined = res?.message;
        const hasPositiveMessage =
          typeof message === "string" && /(success|completed|verified)/i.test(message);
        const hasEmail = typeof res?.email === "string" && res.email.length > 3;
        const success =
          typeof res?.success === "boolean" ? res.success : hasPositiveMessage || hasEmail;
        const error =
          res?.error || (!success ? message || "Failed to complete registration" : undefined);
        return {
          success,
          message,
          email: res?.email,
          error,
        };
      },
    }),

    // SAML login check (similar to custom login but for SAML flows)
    samlLogin: builder.mutation<SamlLoginResponse, SamlLoginRequest>({
      query: (data) => ({
        url: "/authsec/uflow/user/saml/login",
        method: "POST",
        body: data,
      }),
    }),

    // Send token to OIDC login endpoint (enhanced)
    sendTokenToOIDCLogin: builder.mutation<OIDCLoginResponse, OIDCLoginRequest>({
      query: (data) => ({
        url: "/authsec/uflow/user/oidc/login",
        method: "POST",
        body: data,
        credentials: "include",
      }),
    }),

    // UFlow OAuth Provider Endpoints
    // Get list of available OAuth providers
    getUFlowOIDCProviders: builder.mutation<UFlowOIDCProvidersResponse, { email: string }>({
      query: (_data) => ({
        url: "/authsec/uflow/oidc/providers",
        method: "GET",
      }),
    }),

    // Initiate UFlow OAuth authentication
    initiateUFlowOIDC: builder.mutation<UFlowOIDCInitiateResponse, UFlowOIDCInitiateRequest>({
      query: (data) => ({
        url: "/authsec/uflow/oidc/initiate",
        method: "POST",
        body: data,
      }),
    }),

    // Exchange admin OIDC authorization code
    exchangeAdminOIDCCode: builder.mutation<
      AdminOIDCExchangeSuccessResponse,
      AdminOIDCExchangeRequest
    >({
      query: (data) => ({
        url: "/authsec/uflow/oidc/exchange-code",
        method: "POST",
        body: data,
      }),
    }),

    // Check tenant domain availability
    checkTenantDomain: builder.query<TenantDomainCheckResponse, string>({
      query: (domain) => ({
        url: `/authsec/uflow/oidc/check-tenant?domain=${encodeURIComponent(domain)}`,
        method: "GET",
      }),
    }),

    // Complete UFlow OAuth registration
    completeUFlowOIDCRegistration: builder.mutation<
      CompleteUFlowOIDCRegistrationResponse,
      CompleteUFlowOIDCRegistrationRequest
    >({
      query: (data) => ({
        url: "/authsec/uflow/oidc/complete-registration",
        method: "POST",
        body: data,
      }),
    }),
  }),
});

export const {
  useExchangeCodeForTokensMutation,
  useHandleCallbackMutation,
  useLazyGetLoginPageDataQuery,
  useInitiateAuthMutation,
  useCheckCustomLoginStatusMutation,
  useRegisterCustomUserMutation,
  useCompleteCustomUserRegistrationMutation,
  useSamlLoginMutation,
  useSendTokenToOIDCLoginMutation,
  useGetUFlowOIDCProvidersMutation,
  useInitiateUFlowOIDCMutation,
  useExchangeAdminOIDCCodeMutation,
  useLazyCheckTenantDomainQuery,
  useCompleteUFlowOIDCRegistrationMutation,
} = oidcApi;
