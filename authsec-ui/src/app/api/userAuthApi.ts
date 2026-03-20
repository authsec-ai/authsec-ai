/**
 * User Authentication API - Direct email/password login only
 * Clean separation from OAuth and WebAuthn flows
 */

import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import config from '../../config';

export interface CustomLoginRequest {
  client_id: string;
  email: string;
  password: string;
  tenant_domain?: string;
}

export interface CustomLoginResponse {
  tenant_id: string;
  email: string;
  first_login: boolean;
  otp_required: boolean;
  mfa_required: boolean;
}

// RTK Query API for direct user authentication
export const userAuthApi = createApi({
  reducerPath: 'userAuthApi',
  baseQuery: fetchBaseQuery({
    baseUrl: `${config.VITE_API_URL}`,
  }),
  tagTypes: ['UserAuth'],
  endpoints: (builder) => ({
    // Custom login for OIDC flow
    customLogin: builder.mutation<CustomLoginResponse, CustomLoginRequest>({
      query: (loginData) => ({
        url: '/authsec/uflow/user/login',
        method: 'POST',
        body: loginData,
      }),
    }),

  }),
});

export const {
  useCustomLoginMutation,
} = userAuthApi;
