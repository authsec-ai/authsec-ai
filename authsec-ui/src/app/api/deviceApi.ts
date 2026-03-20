/**
 * Device Management API - TOTP and CIBA device operations
 * Handles device registration, listing, and deletion for authenticated users
 */

import { createApi, fetchBaseQuery } from "@reduxjs/toolkit/query/react";
import config from '../../config';

// ============ TOTP Device Types ============
export interface TOTPDevice {
  id: string;
  user_id: string;
  tenant_id: string;
  device_name: string;
  device_type: string;
  last_used: number | null;
  is_active: boolean;
  is_primary: boolean;
  created_at: number;
  updated_at: number;
}

export interface TOTPDevicesResponse {
  success: boolean;
  devices: TOTPDevice[];
  message?: string;
  error?: string;
}

export interface TOTPRegisterRequest {
  device_name: string;
  device_type: string;
}

export interface TOTPRegisterResponse {
  success: boolean;
  secret: string;
  qr_code_url: string;
  device_id: string;
  backup_codes: string[];
  message?: string;
  error?: string;
}

export interface TOTPConfirmRequest {
  device_id: string;
  totp_code: string;
}

export interface TOTPConfirmResponse {
  success: boolean;
  message?: string;
  device_id: string;
  device_name: string;
  error?: string;
}

export interface TOTPDeleteResponse {
  success: boolean;
  message?: string;
  error?: string;
}

// ============ CIBA Device Types ============
export interface CIBADevice {
  id: string;
  device_name: string;
  platform: string;
  device_model: string;
  app_version: string;
  os_version: string;
  is_active: boolean;
  created_at: number;
}

export interface CIBADevicesResponse {
  success: boolean;
  devices: CIBADevice[];
  message?: string;
  error?: string;
}

export interface CIBADeleteResponse {
  success: boolean;
  message?: string;
  error?: string;
}

// ============ API Definition ============
export const deviceApi = createApi({
  reducerPath: "deviceApi",
  baseQuery: fetchBaseQuery({
    baseUrl: config.VITE_API_URL || "https://test.api.authsec.dev",
    timeout: 30000,
    credentials: "include",
    prepareHeaders: (headers, { getState }) => {
      // Token will be passed dynamically via endpoint args
      headers.set("Content-Type", "application/json");
      return headers;
    },
  }),
  tagTypes: ["TOTPDevices", "CIBADevices"],
  endpoints: (builder) => ({
    // ============ TOTP Endpoints ============

    // Get TOTP devices
    getTOTPDevices: builder.query<TOTPDevicesResponse, { token: string }>({
      query: ({ token }) => ({
        url: "/authsec/uflow/auth/tenant/totp/devices",
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }),
      providesTags: ["TOTPDevices"],
    }),

    // Register new TOTP device
    registerTOTPDevice: builder.mutation<
      TOTPRegisterResponse,
      { token: string; data: TOTPRegisterRequest }
    >({
      query: ({ token, data }) => ({
        url: "/authsec/uflow/auth/tenant/totp/register",
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: data,
      }),
      invalidatesTags: ["TOTPDevices"],
    }),

    // Confirm TOTP device with code
    confirmTOTPDevice: builder.mutation<
      TOTPConfirmResponse,
      { token: string; data: TOTPConfirmRequest }
    >({
      query: ({ token, data }) => ({
        url: "/authsec/uflow/auth/tenant/totp/confirm",
        method: "POST",
        headers: {
          Authorization: `Bearer ${token}`,
        },
        body: data,
      }),
      invalidatesTags: ["TOTPDevices"],
    }),

    // Delete TOTP device
    deleteTOTPDevice: builder.mutation<TOTPDeleteResponse, { token: string; deviceId: string }>({
      query: ({ token, deviceId }) => ({
        url: `/authsec/uflow/auth/tenant/totp/devices/${deviceId}`,
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }),
      invalidatesTags: ["TOTPDevices"],
    }),

    // ============ CIBA Endpoints ============

    // Get CIBA devices
    getCIBADevices: builder.query<CIBADevicesResponse, { token: string }>({
      query: ({ token }) => ({
        url: "/authsec/uflow/auth/tenant/ciba/devices",
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }),
      providesTags: ["CIBADevices"],
    }),

    // Delete CIBA device
    deleteCIBADevice: builder.mutation<CIBADeleteResponse, { token: string; deviceId: string }>({
      query: ({ token, deviceId }) => ({
        url: `/authsec/uflow/auth/tenant/ciba/devices/${deviceId}`,
        method: "DELETE",
        headers: {
          Authorization: `Bearer ${token}`,
        },
      }),
      invalidatesTags: ["CIBADevices"],
    }),
  }),
});

export const {
  useGetTOTPDevicesQuery,
  useLazyGetTOTPDevicesQuery,
  useRegisterTOTPDeviceMutation,
  useConfirmTOTPDeviceMutation,
  useDeleteTOTPDeviceMutation,
  useGetCIBADevicesQuery,
  useLazyGetCIBADevicesQuery,
  useDeleteCIBADeviceMutation,
} = deviceApi;
