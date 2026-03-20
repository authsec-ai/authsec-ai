/**
 * SYNC CONFIGURATIONS API
 *
 * Endpoints for managing directory sync configurations
 * Authentication: Requires AuthMiddleware
 * Base Path: /uflow/admin/sync-configs
 *
 * Available Endpoints:
 * - POST /uflow/admin/sync-configs/create - Create sync configuration
 * - POST /uflow/admin/sync-configs/list - List sync configurations
 * - POST /uflow/admin/sync-configs/update - Update sync configuration
 * - POST /uflow/admin/sync-configs/delete - Delete sync configuration
 */

import { baseApi, withSessionData } from './baseApi';

// ============================================================================
// TYPES
// ============================================================================

export type SyncType = 'active_directory' | 'entra_id';

export interface ADConfig {
  server: string;
  username: string;
  password: string;
  base_dn: string;
  filter?: string;
  use_ssl?: boolean;
  skip_verify?: boolean;
}

export interface EntraConfig {
  tenant_id: string;
  client_id: string;
  client_secret: string;
  skip_verify?: boolean;
}

export interface SyncConfig {
  id?: string;
  tenant_id?: string;
  client_id?: string;
  project_id?: string;
  sync_type: SyncType;
  config_name: string;
  description?: string;
  is_active?: boolean;
  ad_config?: ADConfig;
  entra_config?: EntraConfig;
  created_at?: string;
  updated_at?: string;
  last_sync_at?: string;
  last_sync_status?: string;
  last_sync_users_count?: number;
}

export interface CreateSyncConfigRequest {
  tenant_id?: string;
  client_id?: string;
  project_id?: string;
  sync_type: SyncType;
  config_name: string;
  description?: string;
  ad_config?: ADConfig;
  entra_config?: EntraConfig;
}

export interface CreateSyncConfigResponse {
  success: boolean;
  message: string;
  data?: SyncConfig;
}

export interface ListSyncConfigsRequest {
  tenant_id?: string;
  client_id?: string;
  sync_type?: SyncType;
}

export interface ListSyncConfigsResponse {
  success: boolean;
  message: string;
  configs: SyncConfig[];
}

export interface UpdateSyncConfigRequest {
  id: string;
  tenant_id?: string;
  client_id?: string;
  config_name?: string;
  description?: string;
  is_active?: boolean;
  ad_config?: Partial<ADConfig>;
  entra_config?: Partial<EntraConfig>;
}

export interface UpdateSyncConfigResponse {
  success: boolean;
  message: string;
  data?: SyncConfig;
}

export interface DeleteSyncConfigRequest {
  id: string;
  tenant_id?: string;
  client_id?: string;
}

export interface DeleteSyncConfigResponse {
  success: boolean;
  message: string;
}

// ============================================================================
// API
// ============================================================================

export const syncConfigsApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({

    // POST /uflow/admin/sync-configs/create
    // Create a new sync configuration
    createSyncConfig: builder.mutation<CreateSyncConfigResponse, CreateSyncConfigRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/sync-configs/create',
        method: 'POST',
        body: withSessionData({
          sync_type: data.sync_type,
          config_name: data.config_name,
          description: data.description,
          ad_config: data.ad_config,
          entra_config: data.entra_config,
        }),
      }),
      invalidatesTags: ['SyncConfig', 'AdminUser', 'EndUser'],
    }),

    // POST /uflow/admin/sync-configs/list
    // List all sync configurations
    listSyncConfigs: builder.query<ListSyncConfigsResponse, ListSyncConfigsRequest | void>({
      query: (data = {}) => ({
        url: 'authsec/uflow/admin/sync-configs/list',
        method: 'POST',
        body: withSessionData({
          sync_type: data.sync_type,
        }),
      }),
      providesTags: ['SyncConfig'],
    }),

    // POST /uflow/admin/sync-configs/update
    // Update an existing sync configuration
    updateSyncConfig: builder.mutation<UpdateSyncConfigResponse, UpdateSyncConfigRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/sync-configs/update',
        method: 'POST',
        body: withSessionData({
          id: data.id,
          config_name: data.config_name,
          description: data.description,
          is_active: data.is_active,
          ad_config: data.ad_config,
          entra_config: data.entra_config,
        }),
      }),
      invalidatesTags: ['SyncConfig', 'AdminUser', 'EndUser'],
    }),

    // POST /uflow/admin/sync-configs/delete
    // Delete a sync configuration
    deleteSyncConfig: builder.mutation<DeleteSyncConfigResponse, DeleteSyncConfigRequest>({
      query: (data) => ({
        url: 'authsec/uflow/admin/sync-configs/delete',
        method: 'POST',
        body: withSessionData({
          id: data.id,
        }),
      }),
      invalidatesTags: ['SyncConfig', 'AdminUser', 'EndUser'],
    }),

  }),
});

export const {
  useCreateSyncConfigMutation,
  useListSyncConfigsQuery,
  useUpdateSyncConfigMutation,
  useDeleteSyncConfigMutation,
} = syncConfigsApi;

// Re-export types for better module resolution
export type {
  SyncType,
  ADConfig,
  EntraConfig,
  SyncConfig,
  CreateSyncConfigRequest,
  CreateSyncConfigResponse,
  ListSyncConfigsRequest,
  ListSyncConfigsResponse,
  UpdateSyncConfigRequest,
  UpdateSyncConfigResponse,
  DeleteSyncConfigRequest,
  DeleteSyncConfigResponse,
};
