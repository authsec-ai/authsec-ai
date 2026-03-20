import { baseApi } from "./baseApi";

export interface VoiceDeviceInfo {
  source?: string;
  voice_platform?: string;
  voice_user_id?: string;
  [key: string]: string | undefined;
}

export interface VoiceDeviceRequest {
  id: string;
  client_id: string;
  created_at: number;
  expires_at: number;
  device_code: string;
  device_info?: VoiceDeviceInfo;
  scopes?: string[];
  status: "pending" | "approved" | "denied" | string;
  user_code: string;
  verification_uri: string;
}

export interface VoicePendingResponse {
  count: number;
  requests: VoiceDeviceRequest[];
}

export interface VoiceApproveRequest {
  user_code: string;
  approve: boolean;
}

export const voiceAgentApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    getPendingVoiceRequests: builder.query<VoicePendingResponse, { clientId: string }>({
      /* Original implementation commented out at request.
      query: ({ clientId }) => ({
        url: "/authsec/uflow/auth/voice/device-pending",
        method: "GET",
        params: { client_id: clientId },
      }),
      transformResponse: (response: { count?: number; requests?: VoiceDeviceRequest[] | null }) => ({
        count: response?.count ?? 0,
        requests: Array.isArray(response?.requests) ? response.requests : [],
      }),
      providesTags: (result) =>
        result?.requests?.length
          ? [
              ...result.requests.map((request) => ({ type: "Agent" as const, id: request.id })),
              { type: "Agent" as const, id: "VOICE_PENDING" },
            ]
          : [{ type: "Agent" as const, id: "VOICE_PENDING" }],
      */
      queryFn: async () => ({ data: { count: 0, requests: [] } }),
      providesTags: [{ type: "Agent" as const, id: "VOICE_PENDING" }],
    }),

    approveVoiceRequest: builder.mutation<{ message?: string }, VoiceApproveRequest>({
      /* Original implementation commented out at request.
      query: (payload) => ({
        url: "/authsec/uflow/auth/voice/device-approve",
        method: "POST",
        body: payload,
      }),
      invalidatesTags: [{ type: "Agent", id: "VOICE_PENDING" }],
      */
      queryFn: async () => ({ data: { message: "voice approval disabled" } }),
      invalidatesTags: [{ type: "Agent", id: "VOICE_PENDING" }],
    }),
  }),
});

export const { useGetPendingVoiceRequestsQuery, useApproveVoiceRequestMutation } = voiceAgentApi;
