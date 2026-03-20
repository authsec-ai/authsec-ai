import { baseApi } from "./baseApi";

export interface LoginRequest {
  email: string;
  password: string;
  tenant_domain?: string;
}

export interface RegisterInitiateRequest {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  tenant_domain?: string;
}

export interface RegisterVerifyRequest {
  email: string;
  otp: string;
}

export interface ResendOtpRequest {
  email: string;
}

export interface ForgotPasswordRequest {
  email: string;
  client_id: string;
}

export interface ForgotPasswordVerifyOtpRequest {
  email: string;
  otp: string;
}

export interface ForgotPasswordResetRequest {
  email: string;
  new_password: string;
  client_id: string;
}

// Admin forgot password interfaces
export interface AdminForgotPasswordRequest {
  email: string;
}

export interface AdminForgotPasswordVerifyOtpRequest {
  email: string;
  otp: string;
}

export interface AdminForgotPasswordResetRequest {
  email: string;
  new_password: string;
}

// New admin login flow objects
export interface AdminLoginPrecheckRequest {
  email: string;
}

export interface AdminLoginPrecheckResponse {
  email: string;
  exists: boolean;
  display_name?: string;
  tenant_domain?: string;
  next_step: "login" | "register";
  requires_password?: boolean;
  available_providers?: Array<"github" | "google" | "microsoft" | "email">;
}

export interface AdminBootstrapAccountRequest {
  email: string;
  password: string;
  confirm_password?: string;
  tenant_domain: string;
}

export interface AdminBootstrapAccountResponse {
  message: string;
  status: "pending_verification" | "registered";
  tenant_id?: string;
  tenant_domain?: string;
}

export interface ForgotPasswordResponse {
  message: string;
  email: string;
}

export interface NotifyNewUserRequest {
  token?: string;
}

export interface NotifyNewUserResponse {
  message?: string;
  success?: boolean;
  // Add other fields based on actual API response
}

export interface AuthUser {
  id: string;
  email: string;
  first_name?: string;
  last_name?: string;
  avatar_url?: string;
}

export interface LoginResponse {
  tenant_id: string;
  email: string;
  first_login: boolean;
}

export interface RegisterInitiateResponse {
  message?: string;
  email?: string;
  // Add any other fields the API returns
}

export interface RegisterVerifyResponse {
  tenant_id: string;
  project_id: string;
  client_id: string;
  email_id: string;
}

export interface ResendOtpResponse {
  message?: string;
}

export const authApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    login: builder.mutation<LoginResponse, LoginRequest>({
      query: (credentials) => {
        const currentDomain = window.location.hostname;
        const bodyWithDomain = {
          ...credentials,
          tenant_domain: credentials.tenant_domain ?? currentDomain,
        };
        return {
          url: "authsec/uflow/auth/admin/login",
          method: "POST",
          body: bodyWithDomain,
        };
      },
      invalidatesTags: ["AdminAuth"],
    }),
    registerInitiate: builder.mutation<RegisterInitiateResponse, RegisterInitiateRequest>({
      query: (userData) => {
        return {
          url: "/authsec/uflow/register/initiate",
          method: "POST",
          body: userData,
        };
      },
      invalidatesTags: ["AdminAuth"],
    }),
    registerVerify: builder.mutation<RegisterVerifyResponse, RegisterVerifyRequest>({
      query: (verifyData) => ({
        url: "/authsec/uflow/register/verify",
        method: "POST",
        body: verifyData,
      }),
      invalidatesTags: ["AdminAuth"],
    }),
    resendOtp: builder.mutation<ResendOtpResponse, ResendOtpRequest>({
      query: (resendData) => ({
        url: "/authsec/uflow/register/resendOtp",
        method: "POST",
        body: resendData,
      }),
      invalidatesTags: ["AdminAuth"],
    }),
    // Forgot Password endpoints - End User
    forgotPassword: builder.mutation<ForgotPasswordResponse, ForgotPasswordRequest>({
      query: (forgotPasswordData) => ({
        url: "/authsec/uflow/user/forgot-password",
        method: "POST",
        body: forgotPasswordData,
      }),
    }),

    forgotPasswordVerifyOtp: builder.mutation<
      ForgotPasswordResponse,
      ForgotPasswordVerifyOtpRequest
    >({
      query: (verifyOtpData) => ({
        url: "/authsec/uflow/user/forgot-password/verify-otp",
        method: "POST",
        body: verifyOtpData,
      }),
    }),

    forgotPasswordReset: builder.mutation<ForgotPasswordResponse, ForgotPasswordResetRequest>({
      query: (resetPasswordData) => ({
        url: "/authsec/uflow/user/forgot-password/reset",
        method: "POST",
        body: resetPasswordData,
      }),
      invalidatesTags: ["AdminAuth"],
    }),

    // Admin forgot password endpoints
    adminForgotPassword: builder.mutation<ForgotPasswordResponse, AdminForgotPasswordRequest>({
      query: (adminForgotPasswordData) => ({
        url: "/authsec/uflow/auth/admin/forgot-password",
        method: "POST",
        body: adminForgotPasswordData,
      }),
    }),

    adminForgotPasswordVerifyOtp: builder.mutation<
      ForgotPasswordResponse,
      AdminForgotPasswordVerifyOtpRequest
    >({
      query: (adminVerifyOtpData) => ({
        url: "/authsec/uflow/auth/admin/forgot-password/verify-otp",
        method: "POST",
        body: adminVerifyOtpData,
      }),
    }),

    adminForgotPasswordReset: builder.mutation<
      ForgotPasswordResponse,
      AdminForgotPasswordResetRequest
    >({
      query: (adminResetPasswordData) => ({
        url: "/authsec/uflow/auth/admin/forgot-password/reset",
        method: "POST",
        body: adminResetPasswordData,
      }),
      invalidatesTags: ["AdminAuth"],
    }),
    // Admin login precheck/ bootstrap for modern flow
    adminLoginPrecheck: builder.mutation<AdminLoginPrecheckResponse, AdminLoginPrecheckRequest>({
      query: (body) => ({
        url: "/authsec/uflow/auth/admin/login/precheck",
        method: "POST",
        body,
      }),
    }),
    adminBootstrapAccount: builder.mutation<AdminBootstrapAccountResponse, AdminBootstrapAccountRequest>({
      query: (body) => ({
        url: "/authsec/uflow/auth/admin/login/bootstrap",
        method: "POST",
        body,
      }),
      invalidatesTags: ["AdminAuth"],
    }),
    // New user registration notification
    notifyNewUserRegistration: builder.mutation<NotifyNewUserResponse, NotifyNewUserRequest>({
      query: (data) => ({
        url: '/authsec/uflow/auth/notify/new-user-registration',
        method: 'POST',
        body: {},
        headers: data?.token ? { Authorization: `Bearer ${data.token}` } : undefined,
      }),
    }),
  }),
  overrideExisting: false,
});

export const {
  useLoginMutation,
  useRegisterInitiateMutation,
  useRegisterVerifyMutation,
  useResendOtpMutation,
  useForgotPasswordMutation,
  useForgotPasswordVerifyOtpMutation,
  useForgotPasswordResetMutation,
  useAdminForgotPasswordMutation,
  useAdminForgotPasswordVerifyOtpMutation,
  useAdminForgotPasswordResetMutation,
  useAdminLoginPrecheckMutation,
  useAdminBootstrapAccountMutation,
  useNotifyNewUserRegistrationMutation,
} = authApi;
