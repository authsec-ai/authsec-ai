/**
 * WebAuthn API - Central API layer for WebAuthn operations
 * Clean, explicit endpoint selection - no automatic flow detection
 */

import { baseApi } from "./baseApi";

export interface WebAuthnRequest {
  email: string;
  tenant_id: string;
  client_id?: string;
}

export interface AdminWebAuthnRequest {
  email: string;
}

export interface AdminMFAStatusRequest {
  email: string;
}

export interface MFAStatusForLoginRequest {
  email: string;
  tenant_id: string;
  client_id: string;
}

export interface MFAStatusMethod {
  id?: string;
  client_id?: string;
  user_id?: string;
  method_type: string;
  display_name?: string;
  description?: string;
  recommended?: boolean;
  method_data?: Record<string, unknown>;
  enabled?: boolean;
  is_primary?: boolean;
  verified?: boolean;
  enrolled_at?: string;
  last_used_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface MFAStatusResponse {
  custom_domain?: string;
  message?: string;
  mfa_enabled?: boolean;
  total_methods?: number;
  configured_methods?: Array<{
    type: string;
    enabled?: boolean;
  }>;
  mfa_default_method?: string | null;
  methods?: MFAStatusMethod[];
  mfa_required?: boolean;
  requires_registration?: boolean;
}

export interface WebAuthnCredential {
  id: string;
  rawId: string;
  type: "public-key";
  response: {
    clientDataJSON: string;
    authenticatorData: string;
    signature: string;
    userHandle: string | null;
  };
}

export interface WebAuthnRegistrationCredential {
  id: string;
  rawId: string;
  type: "public-key";
  response: {
    attestationObject: string;
    clientDataJSON: string;
  };
}

export interface FinishAuthRequest {
  email: string;
  tenant_id: string;
  client_id?: string;
  credential: WebAuthnCredential;
}

export interface AdminFinishAuthRequest {
  email: string;
  credential: WebAuthnCredential;
}

export interface FinishRegistrationRequest {
  email: string;
  tenant_id: string;
  client_id?: string;
  credential: WebAuthnRegistrationCredential;
}

export interface AdminFinishRegistrationRequest {
  email: string;
  credential: WebAuthnRegistrationCredential;
}

export interface TOTPSetupRequest {
  email: string;
  tenant_id: string;
  client_id?: string;
}

export interface TOTPSetupResponse {
  account: string;
  issuer: string;
  qr_code: string;
  secret: string;
}

export interface TOTPConfirmRequest {
  email: string;
  tenant_id: string;
  client_id?: string;
  secret: string;
  code: string;
}

export interface TOTPVerifyRequest {
  email: string;
  tenant_id: string;
  client_id?: string;
  code: string;
}

export interface MFAMethod {
  type: "webauthn" | "totp";
  display_name: string;
  description: string;
  enabled: boolean;
  recommended: boolean;
}

export interface WebAuthnCallbackRequest {
  email: string;
  mfa_verified: boolean;
  tenant_id?: string;
  client_id?: string;
  flow_context?: 'admin' | 'oidc' | 'enduser';
}

export interface WebAuthnCallbackResponse {
  success: boolean;
  token?: string;
  // Some backends may return alternative token keys
  access_token?: string;
  jwt?: string;
  jwt_token?: string;
  id_token?: string;
  error?: string;
  flow_context?: 'admin' | 'oidc' | 'enduser';
}

// RTK Query API for WebAuthn operations
export const webauthnApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    getMFAStatusForLogin: builder.mutation<MFAStatusResponse, MFAStatusForLoginRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/enduser/mfa/loginStatus',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['MFA'],
    }),

    // Admin MFA Status
    getAdminMFAStatus: builder.mutation<MFAStatusResponse, AdminMFAStatusRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/admin/mfa/loginStatus',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['MFA'],
    }),

    // Admin WebAuthn Registration Flow
    beginAdminRegistration: builder.mutation<any, AdminWebAuthnRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/admin/beginRegistration',
        method: 'POST',
        body: data,
      }),
    }),

    finishAdminRegistration: builder.mutation<any, AdminFinishRegistrationRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/admin/finishRegistration',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['WebAuthn', 'MFA'],
    }),

    // Admin WebAuthn Authentication Flow
    beginAdminAuthentication: builder.mutation<any, AdminWebAuthnRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/admin/beginAuthentication',
        method: 'POST',
        body: data,
      }),
    }),

    finishAdminAuthentication: builder.mutation<any, AdminFinishAuthRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/admin/finishAuthentication',
        method: 'POST',
        body: data,
      }),
    }),

    // WebAuthn Authentication Flow
    beginWebAuthnAuth: builder.mutation<any, WebAuthnRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/enduser/beginAuthentication',
        method: 'POST',
        body: data,
      }),
    }),

    finishWebAuthnAuth: builder.mutation<any, FinishAuthRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/enduser/finishAuthentication',
        method: 'POST',
        body: data,
      }),
    }),

    // WebAuthn Registration Flow - MFA method discovery
    beginWebAuthnRegistration: builder.mutation<any, WebAuthnRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/enduser/beginRegistration',
        method: 'POST',
        body: data,
      }),
    }),

    finishWebAuthnRegistration: builder.mutation<any, FinishRegistrationRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/enduser/finishRegistration',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['WebAuthn', 'MFA'],
    }),

    // Admin TOTP Setup Flow (endpoints WITH "Login")
    beginTOTPLoginSetup: builder.mutation<TOTPSetupResponse, TOTPSetupRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/totp/beginLoginSetup',
        method: 'POST',
        body: data,
      }),
    }),

    confirmTOTPLoginSetup: builder.mutation<any, TOTPConfirmRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/totp/confirmLoginSetup',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['TOTP', 'MFA'],
    }),

    verifyTOTPLogin: builder.mutation<any, TOTPVerifyRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/totp/verifyLogin',
        method: 'POST',
        body: data,
      }),
    }),

    // End-User/OIDC TOTP Flow (simple endpoints WITHOUT "Login")
    beginTOTPSetup: builder.mutation<TOTPSetupResponse, TOTPSetupRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/totp/beginSetup',
        method: 'POST',
        body: data,
      }),
    }),

    confirmTOTPSetup: builder.mutation<any, TOTPConfirmRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/totp/confirmSetup',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['TOTP', 'MFA'],
    }),

    verifyTOTP: builder.mutation<any, TOTPVerifyRequest>({
      query: (data) => ({
        url: '/authsec/webauthn/totp/verify',
        method: 'POST',
        body: data,
      }),
    }),

    // WebAuthn Callback - Simple, explicit endpoint (Admin flow)
    webauthnCallback: builder.mutation<WebAuthnCallbackResponse, WebAuthnCallbackRequest>({
      query: (data) => ({
        url: '/authsec/uflow/login/webauthn-callback',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['MFAAuth', 'MFA'],
    }),

    // WebAuthn Callback - Enduser flow specific endpoint
    webauthnEnduserCallback: builder.mutation<WebAuthnCallbackResponse, WebAuthnCallbackRequest>({
      query: (data) => ({
        url: '/authsec/uflow/auth/enduser/webauthn-callback',
        method: 'POST',
        body: data,
      }),
      invalidatesTags: ['MFAAuth', 'MFA'],
    }),

  }),
  overrideExisting: false,
});

export const {
  useGetMFAStatusForLoginMutation,
  // Admin MFA and WebAuthn
  useGetAdminMFAStatusMutation,
  useBeginAdminRegistrationMutation,
  useFinishAdminRegistrationMutation,
  useBeginAdminAuthenticationMutation,
  useFinishAdminAuthenticationMutation,
  // Enduser WebAuthn
  useBeginWebAuthnAuthMutation,
  useFinishWebAuthnAuthMutation,
  useBeginWebAuthnRegistrationMutation,
  useFinishWebAuthnRegistrationMutation,
  // Admin TOTP endpoints (WITH "Login")
  useBeginTOTPLoginSetupMutation,
  useConfirmTOTPLoginSetupMutation,
  useVerifyTOTPLoginMutation,
  // End-User/OIDC TOTP endpoints (WITHOUT "Login")
  useBeginTOTPSetupMutation,
  useConfirmTOTPSetupMutation,
  useVerifyTOTPMutation,
  // WebAuthn callbacks
  useWebauthnCallbackMutation,
  useWebauthnEnduserCallbackMutation,
} = webauthnApi;
