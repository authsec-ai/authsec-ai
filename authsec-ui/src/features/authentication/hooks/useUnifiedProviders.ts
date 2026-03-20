import { useMemo } from 'react';
import { useShowAuthProvidersQuery } from '@/app/api/authMethodApi';
import { useListSamlProvidersQuery } from '@/app/api/samlApi';
import type { UnifiedAuthProvider, ApiOidcProvider, ApiSamlProvider } from '../types';

interface UseUnifiedProvidersParams {
  tenant_id: string;
  client_id?: string;
}

interface UseUnifiedProvidersReturn {
  providers: UnifiedAuthProvider[];
  isLoading: boolean;
  isError: boolean;
  error: any;
  refetch: () => void;
}

/**
 * Hook to fetch and merge both OIDC and SAML providers into a unified list
 */
export const useUnifiedProviders = ({
  tenant_id,
  client_id,
}: UseUnifiedProvidersParams): UseUnifiedProvidersReturn => {
  // Fetch OIDC providers
  const {
    data: oidcData,
    isLoading: isOidcLoading,
    isError: isOidcError,
    error: oidcError,
    refetch: refetchOidc,
  } = useShowAuthProvidersQuery({ tenant_id, client_id });

  // Fetch SAML providers
  const {
    data: samlData,
    isLoading: isSamlLoading,
    isError: isSamlError,
    error: samlError,
    refetch: refetchSaml,
  } = useListSamlProvidersQuery({ tenant_id, client_id });

  // Convert OIDC provider to unified format
  const normalizeOidcProvider = (oidc: ApiOidcProvider): UnifiedAuthProvider => {
    return {
      id: `oidc-${oidc.provider_name}-${oidc.client_id}`,
      provider_type: 'oidc',
      provider_name: oidc.provider_name,
      display_name: oidc.display_name,
      client_id: oidc.client_id,
      is_active: oidc.is_active,
      sort_order: oidc.sort_order,
      status: oidc.status,
      callback_url: oidc.callback_url,
      hydra_client_id: oidc.hydra_client_id,
      endpoints: oidc.endpoints,
      _raw: oidc,
    };
  };

  // Convert SAML provider to unified format
  const normalizeSamlProvider = (saml: ApiSamlProvider): UnifiedAuthProvider => {
    return {
      id: `saml-${saml.id}`,
      provider_type: 'saml',
      provider_name: saml.provider_name,
      display_name: saml.display_name,
      client_id: saml.client_id || '',
      is_active: saml.is_active,
      sort_order: saml.sort_order,
      status: saml.is_active ? 'active' : 'inactive',
      entity_id: saml.entity_id,
      sso_url: saml.sso_url,
      slo_url: saml.slo_url,
      certificate: saml.certificate,
      metadata_url: saml.metadata_url,
      name_id_format: saml.name_id_format,
      attribute_mapping: saml.attribute_mapping,
      created_at: saml.created_at,
      updated_at: saml.updated_at,
      _raw: saml,
    };
  };

  // Merge and normalize providers
  const providers = useMemo(() => {
    const unified: UnifiedAuthProvider[] = [];

    // Add OIDC providers
    if (oidcData?.success && oidcData.data?.providers) {
      const oidcProviders = oidcData.data.providers.map(normalizeOidcProvider);
      unified.push(...oidcProviders);
    }

    // Add SAML providers
    if (samlData?.success && samlData.providers) {
      const samlProviders = samlData.providers.map(normalizeSamlProvider);
      unified.push(...samlProviders);
    }

    // Sort by sort_order
    unified.sort((a, b) => a.sort_order - b.sort_order);

    return unified;
  }, [oidcData, samlData]);

  // Refetch both providers
  const refetch = () => {
    refetchOidc();
    refetchSaml();
  };

  return {
    providers,
    isLoading: isOidcLoading || isSamlLoading,
    isError: isOidcError || isSamlError,
    error: oidcError || samlError,
    refetch,
  };
};
