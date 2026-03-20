import { baseApi } from './baseApi';

// SAML Provider Types
export interface SamlProviderConfig {
  tenant_id: string;
  client_id: string;
  provider_name: string;
  display_name: string;
  entity_id: string;
  sso_url: string;
  certificate: string;
  metadata_url?: string;
  name_id_format: string;
  attribute_mapping: {
    email: string;
    first_name: string;
    last_name: string;
  };
  is_active: boolean;
  sort_order: number;
}

export interface SamlMetadataRequest {
  tenant_id: string;
  client_id: string;
}

export interface SamlMetadataResponse {
  xml: string;
  entity_id: string;
  acs_url: string;
}

export interface AddSamlProviderRequest extends SamlProviderConfig {}

export interface AddSamlProviderResponse {
  success: boolean;
  message: string;
  data?: any;
  timestamp: string;
}

// List SAML Providers Types
export interface ListSamlProvidersRequest {
  tenant_id: string;
  client_id?: string; // Optional - filter by client
}

export interface SamlProvider {
  id: string;
  tenant_id: string;
  client_id?: string;
  provider_name: string;
  display_name: string;
  entity_id: string;
  sso_url: string;
  slo_url?: string;
  certificate: string;
  metadata_url?: string;
  name_id_format: string;
  attribute_mapping: {
    email: string;
    first_name: string;
    last_name: string;
    [key: string]: string;
  };
  is_active: boolean;
  sort_order: number;
  created_at: string;
  updated_at: string;
}

export interface ListSamlProvidersResponse {
  success: boolean;
  tenant_id: string;
  client_id?: string;
  providers: SamlProvider[];
}

// Get SAML Provider Types
export interface GetSamlProviderRequest {
  tenant_id: string;
  provider_id: string;
}

export interface GetSamlProviderResponse {
  success: boolean;
  provider: SamlProvider;
}

// Update SAML Provider Types
export interface UpdateSamlProviderRequest {
  tenant_id: string;
  provider_id: string;
  client_id?: string; // Optional - filter by client
  provider_name?: string;
  display_name?: string;
  entity_id?: string;
  sso_url?: string;
  slo_url?: string;
  certificate?: string;
  metadata_url?: string;
  name_id_format?: string;
  attribute_mapping?: {
    email?: string;
    first_name?: string;
    last_name?: string;
    [key: string]: string | undefined;
  };
  is_active?: boolean;
  sort_order?: number;
}

export interface UpdateSamlProviderResponse {
  success: boolean;
  message: string;
  provider: SamlProvider;
}

// Delete SAML Provider Types
export interface DeleteSamlProviderRequest {
  tenant_id: string;
  provider_id: string;
  client_id?: string; // Optional - filter by client
}

export interface DeleteSamlProviderResponse {
  success: boolean;
  message: string;
}

// SAML Templates Types
export interface SamlTemplate {
  provider_name: string;
  display_name: string;
  name_id_format: string;
  attribute_mapping: {
    email: string;
    first_name: string;
    last_name: string;
    [key: string]: string;
  };
  instructions?: string;
  documentation_url?: string;
  config_fields?: string[];
}

export interface GetSamlTemplatesResponse {
  success: boolean;
  templates: {
    [key: string]: SamlTemplate;
  };
}

// Helper function to parse XML metadata
export const parseMetadataXml = (xml: string): { entity_id: string; acs_url: string } => {
  const parser = new DOMParser();
  const xmlDoc = parser.parseFromString(xml, 'text/xml');

  // Extract Entity ID
  const entityDescriptor = xmlDoc.querySelector('EntityDescriptor, md\\:EntityDescriptor');
  const entity_id = entityDescriptor?.getAttribute('entityID') || '';

  // Extract ACS URL (first AssertionConsumerService with isDefault="true" or index="1")
  const acsServices = xmlDoc.querySelectorAll('AssertionConsumerService, md\\:AssertionConsumerService');
  let acs_url = '';

  for (let i = 0; i < acsServices.length; i++) {
    const service = acsServices[i];
    const isDefault = service.getAttribute('isDefault') === 'true';
    const index = service.getAttribute('index') === '1';

    if (isDefault || index) {
      acs_url = service.getAttribute('Location') || '';
      break;
    }
  }

  // Fallback to first ACS if none marked as default
  if (!acs_url && acsServices.length > 0) {
    acs_url = acsServices[0].getAttribute('Location') || '';
  }

  return { entity_id, acs_url };
};

export const samlApi = baseApi.injectEndpoints({
  endpoints: (builder) => ({
    // Get SAML Metadata
    getSamlMetadata: builder.query<SamlMetadataResponse, SamlMetadataRequest>({
      query: ({ tenant_id, client_id }) => ({
        url: `/authsec/hmgr/saml/metadata/${tenant_id}/${client_id}`,
        method: 'GET',
        responseHandler: async (response) => {
          const xml = await response.text();
          const parsed = parseMetadataXml(xml);
          return {
            xml,
            entity_id: parsed.entity_id,
            acs_url: parsed.acs_url,
          };
        },
      }),
      providesTags: ['SamlMetadata'],
    }),

    // Add SAML Provider
    addSamlProvider: builder.mutation<AddSamlProviderResponse, AddSamlProviderRequest>({
      query: (data) => ({
        url: '/authsec/oocmgr/saml/add-provider',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: data,
      }),
      invalidatesTags: ['AuthMethodOIDCProvider', 'SamlProvider'],
    }),

    // List SAML Providers
    listSamlProviders: builder.query<ListSamlProvidersResponse, ListSamlProvidersRequest>({
      query: (data) => ({
        url: '/authsec/oocmgr/saml/list-providers',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: data,
      }),
      providesTags: (result, error, arg) =>
        arg.client_id
          ? [{ type: 'SamlProvider', id: arg.client_id }, { type: 'SamlProvider', id: 'LIST' }]
          : [{ type: 'SamlProvider', id: 'LIST' }],
    }),

    // Get Specific SAML Provider
    getSamlProvider: builder.query<GetSamlProviderResponse, GetSamlProviderRequest>({
      query: (data) => ({
        url: '/authsec/oocmgr/saml/get-provider',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: data,
      }),
      providesTags: (result, error, arg) => [{ type: 'SamlProvider', id: arg.provider_id }],
    }),

    // Update SAML Provider
    updateSamlProvider: builder.mutation<UpdateSamlProviderResponse, UpdateSamlProviderRequest>({
      query: (data) => ({
        url: '/authsec/oocmgr/saml/update-provider',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: data,
      }),
      invalidatesTags: (result, error, arg) => [
        { type: 'SamlProvider', id: arg.provider_id },
        { type: 'SamlProvider', id: 'LIST' },
        'AuthMethodOIDCProvider', // Also invalidate OIDC to refresh unified list
      ],
    }),

    // Delete SAML Provider
    deleteSamlProvider: builder.mutation<DeleteSamlProviderResponse, DeleteSamlProviderRequest>({
      query: (data) => ({
        url: '/authsec/oocmgr/saml/delete-provider',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: data,
      }),
      invalidatesTags: ['SamlProvider', 'AuthMethodOIDCProvider'],
    }),

    // Get SAML Templates
    getSamlTemplates: builder.query<GetSamlTemplatesResponse, {}>({
      query: () => ({
        url: '/authsec/oocmgr/saml/templates',
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: {},
      }),
      providesTags: ['SamlTemplate'],
    }),
  }),
});

export const {
  useGetSamlMetadataQuery,
  useLazyGetSamlMetadataQuery,
  useAddSamlProviderMutation,
  useListSamlProvidersQuery,
  useLazyListSamlProvidersQuery,
  useGetSamlProviderQuery,
  useLazyGetSamlProviderQuery,
  useUpdateSamlProviderMutation,
  useDeleteSamlProviderMutation,
  useGetSamlTemplatesQuery,
  useLazyGetSamlTemplatesQuery,
} = samlApi;
