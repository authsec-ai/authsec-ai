import { baseApi } from "./baseApi";

export interface AdminCibaDevice {
  id: string;
  device_name: string;
  platform: string;
  device_model?: string;
  app_version?: string;
  os_version?: string;
  is_active: boolean;
  last_used?: number | null;
  created_at: number;
}

export interface AdminCibaDevicesResponse {
  success: boolean;
  devices: AdminCibaDevice[];
  message?: string;
  error?: string;
}

export interface AdminTotpDevice {
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

export interface AdminTotpDevicesResponse {
  success: boolean;
  devices: AdminTotpDevice[];
  message?: string;
  error?: string;
}

export const adminVoiceAgentApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getAdminCibaDevices: builder.query<AdminCibaDevicesResponse, void>({
      query: () => ({
        url: "/authsec/uflow/auth/ciba/devices",
        method: "GET",
      }),
      transformResponse: (response: AdminCibaDevicesResponse) => ({
        ...response,
        devices: Array.isArray(response?.devices) ? response.devices : [],
      }),
      providesTags: [{ type: "Agent", id: "ADMIN_CIBA_DEVICES" }],
    }),
    deleteAdminCibaDevice: builder.mutation<{ success?: boolean; message?: string }, { deviceId: string }>({
      query: ({ deviceId }) => ({
        url: `/authsec/uflow/auth/ciba/devices/${deviceId}`,
        method: "DELETE",
      }),
      invalidatesTags: [{ type: "Agent", id: "ADMIN_CIBA_DEVICES" }],
    }),
    getAdminTotpDevices: builder.query<AdminTotpDevicesResponse, void>({
      query: () => ({
        url: "/authsec/uflow/auth/totp/devices",
        method: "GET",
      }),
      transformResponse: (response: AdminTotpDevicesResponse) => ({
        ...response,
        devices: Array.isArray(response?.devices) ? response.devices : [],
      }),
      providesTags: [{ type: "Agent", id: "ADMIN_TOTP_DEVICES" }],
    }),
  }),
  overrideExisting: false,
});

export const {
  useGetAdminCibaDevicesQuery,
  useDeleteAdminCibaDeviceMutation,
  useGetAdminTotpDevicesQuery,
} = adminVoiceAgentApi;
