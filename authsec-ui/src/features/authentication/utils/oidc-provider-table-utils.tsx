import * as React from "react";
import { Button } from "../../../components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  Shield,
  MoreHorizontal,
  Copy,
  Trash2,
  Eye,
  EyeOff,
  Key,
  Globe,
  Link as LinkIcon,
} from "lucide-react";
import type { ResponsiveColumnDef } from "../../../components/ui/responsive-data-table";
import { CopyButton } from "../../../components/ui/copy-button";
import type { UnifiedAuthProvider } from "../types";

// OIDC Provider interface based on new API response (show-auth-providers)
export interface ApiOidcProvider {
  callback_url: string;
  client_id: string;
  hydra_client_id?: string;
  display_name: string;
  is_active: boolean;
  endpoints: {
    auth_url: string;
    token_url: string;
    user_info_url?: string;
  };
  provider_name: string;
  sort_order: number;
  status: string;
  // Optional backward compatibility fields
  client_ids?: string;
  created_at?: string;
  provider_config?: {
    auth_url?: string;
    token_url?: string;
    user_info_url?: string;
    scopes?: string[];
    redirect_urls?: string[];
    client_id?: string;
    client_secret?: string;
  };
}

// OIDC Provider-specific utility functions
export const OidcProviderTableUtils = {
  // Format provider name for display
  formatProviderName: (providerName: string) => {
    const nameMap: Record<string, string> = {
      google: "Google",
      github: "GitHub",
      microsoft: "Microsoft",
      facebook: "Facebook",
      twitter: "Twitter",
      linkedin: "LinkedIn",
      apple: "Apple",
    };
    return (
      nameMap[providerName.toLowerCase()] ||
      providerName.charAt(0).toUpperCase() + providerName.slice(1)
    );
  },

  // Check if provider has complete configuration
  isConfigurationComplete: (provider: ApiOidcProvider) => {
    const endpoints = provider.endpoints || provider.provider_config || {};
    return Boolean(endpoints.auth_url && endpoints.token_url && endpoints.user_info_url);
  },
};

// OIDC Provider table action handlers interface
export interface OidcProviderTableActions {
  onDuplicate: (providerId: string) => void;
  onDelete: (providerId: string) => void;
  onToggleActive: (providerId: string, isActive: boolean) => void;
  onViewConfiguration: (providerId: string) => void;
  onTestConnection: (providerId: string) => void;
}

// Reusable provider cell component
export function ProviderCell({
  provider,
  onSelect,
}: {
  provider: UnifiedAuthProvider | ApiOidcProvider;
  onSelect?: (provider: UnifiedAuthProvider | ApiOidcProvider) => void;
}) {
  return (
    <div className="min-w-0 space-y-1">
      {onSelect ? (
        <button
          type="button"
          className="w-full truncate text-left text-sm font-medium text-primary hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1"
          title={provider.display_name}
          onClick={(event) => {
            event.stopPropagation();
            onSelect(provider);
          }}
        >
          {provider.display_name}
        </button>
      ) : (
        <p className="truncate text-sm font-medium text-foreground" title={provider.display_name}>
          {provider.display_name}
        </p>
      )}
      <p className="text-xs text-foreground">
        {OidcProviderTableUtils.formatProviderName(provider.provider_name)}
      </p>
    </div>
  );
}

// Reusable status cell component
export function StatusCell({ provider }: { provider: UnifiedAuthProvider | ApiOidcProvider }) {
  // Check if it's a unified provider with provider_type field
  const isUnified = "provider_type" in provider;
  const isSaml = isUnified && provider.provider_type === "saml";

  // For SAML, we consider it complete if it has entity_id and sso_url
  // For OIDC, use the existing check
  const isComplete = isSaml
    ? Boolean(
        (provider as UnifiedAuthProvider).entity_id && (provider as UnifiedAuthProvider).sso_url
      )
    : OidcProviderTableUtils.isConfigurationComplete(provider as ApiOidcProvider);

  return (
    <div className="space-y-1 text-sm">
      <div className="flex items-center gap-2">
        <div
          className={`h-2 w-2 rounded-full ${provider.is_active ? "bg-green-500" : "bg-gray-400"}`}
        />
        <p
          className={`font-medium ${
            provider.is_active
              ? "text-green-700 dark:text-green-400"
              : "text-gray-600 dark:text-gray-400"
          }`}
        >
          {provider.is_active ? "Active" : "Inactive"}
        </p>
      </div>
      <p className="text-xs text-foreground">
        {isComplete ? "Configuration complete" : "Configuration incomplete"}
      </p>
    </div>
  );
}

export function ConfigurationCell({ provider }: { provider: ApiOidcProvider }) {
  const scopeCount = provider.provider_config?.scopes?.length ?? 0;
  const callbackUrl =
    provider.callback_url ||
    provider.provider_config?.redirect_urls?.[0] ||
    provider.provider_config?.auth_url ||
    "";

  return (
    <div className="space-y-1 text-sm text-foreground">
      <p>
        {scopeCount} {scopeCount === 1 ? "scope" : "scopes"}
      </p>
      <p className="truncate" title={callbackUrl || "No callback URL"}>
        {callbackUrl || "No callback URL"}
      </p>
    </div>
  );
}

// Reusable endpoints cell component
export function EndpointsCell({ provider }: { provider: ApiOidcProvider }) {
  const endpoints = provider.endpoints || provider.provider_config || {};
  const hasAllEndpoints = endpoints.auth_url && endpoints.token_url && endpoints.user_info_url;
  const configured = [
    endpoints.auth_url ? "Auth" : null,
    endpoints.token_url ? "Token" : null,
    endpoints.user_info_url ? "User info" : null,
  ].filter(Boolean);

  return (
    <div className="space-y-1 text-sm">
      <p className="text-foreground">
        {hasAllEndpoints
          ? "Auth • Token • User info"
          : `${configured.length}/3 endpoints configured`}
      </p>
      <p className="text-xs text-foreground">
        {hasAllEndpoints ? "Ready for OAuth exchange" : "Add the missing endpoints to finish setup"}
      </p>
    </div>
  );
}

// Reusable activity cell component
export function ActivityCell({ provider }: { provider: ApiOidcProvider }) {
  return (
    <div className="space-y-1 text-sm">
      <p className="text-foreground">Sort order {provider.sort_order ?? "—"}</p>
      <p className="text-xs text-foreground">{provider.status || "No recent status"}</p>
    </div>
  );
}

// Reusable actions cell component
export function ActionsCell({
  provider,
  actions,
}: {
  provider: UnifiedAuthProvider | ApiOidcProvider;
  actions: OidcProviderTableActions;
}) {
  const [isOpen, setIsOpen] = React.useState(false);

  // Determine provider ID based on type
  const providerId = "provider_type" in provider ? provider.id : provider.client_id;

  // Log when dropdown state changes
  React.useEffect(() => {
    // eslint-disable-next-line no-console
    console.log(
      `[ActionsCell] Dropdown for provider "${provider.display_name}" (${providerId}) is now:`,
      isOpen ? "OPEN" : "CLOSED"
    );
  }, [isOpen, provider.display_name, providerId]);

  const handleOpenChange = (open: boolean) => {
    // eslint-disable-next-line no-console
    console.log(`[ActionsCell] handleOpenChange called for "${provider.display_name}":`, open);
    setIsOpen(open);
  };

  const handleToggleActive = (e: React.MouseEvent, isActive: boolean) => {
    // eslint-disable-next-line no-console
    console.log(`[ActionsCell] handleToggleActive for "${provider.display_name}":`, isActive);
    e.stopPropagation();
    e.preventDefault();
    actions.onToggleActive(providerId, isActive);
    setIsOpen(false);
  };

  const handleDelete = (e: React.MouseEvent) => {
    // eslint-disable-next-line no-console
    console.log(`[ActionsCell] handleDelete for "${provider.display_name}"`);
    e.stopPropagation();
    e.preventDefault();
    actions.onDelete(providerId);
    setIsOpen(false);
  };

  const handleTriggerClick = (e: React.MouseEvent) => {
    // eslint-disable-next-line no-console
    console.log(`[ActionsCell] Trigger button clicked for "${provider.display_name}"`, {
      clientId: provider.client_id,
      currentOpenState: isOpen,
      eventTarget: (e.target as HTMLElement).tagName,
    });
    e.stopPropagation();
    e.preventDefault();
  };

  const handleContainerClick = (e: React.MouseEvent) => {
    // eslint-disable-next-line no-console
    console.log(`[ActionsCell] Container div clicked for "${provider.display_name}"`);
    e.stopPropagation();
  };

  const handleContentClick = (e: React.MouseEvent) => {
    // eslint-disable-next-line no-console
    console.log(`[ActionsCell] Dropdown content clicked for "${provider.display_name}"`);
    e.stopPropagation();
    e.preventDefault();
  };

  // Check if this is the authsec provider (system default, cannot be deleted)
  const isAuthSec = provider.provider_name?.toLowerCase() === "authsec";

  return (
    <div className="flex items-center justify-end gap-1" onClick={handleContainerClick}>
      <DropdownMenu open={isOpen} onOpenChange={handleOpenChange}>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0" onClick={handleTriggerClick}>
            <MoreHorizontal className="h-4 w-4" />
            <span className="sr-only">Open menu for {provider.display_name}</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" visualVariant="row-actions" onClick={handleContentClick}>
          {/* Status Toggle */}
          {provider.is_active ? (
            <DropdownMenuItem onClick={(e) => handleToggleActive(e, false)}>
              <EyeOff className="mr-2 h-4 w-4" />
              Deactivate Provider
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={(e) => handleToggleActive(e, true)}>
              <Eye className="mr-2 h-4 w-4" />
              Activate Provider
            </DropdownMenuItem>
          )}

          {/* Delete - hidden for authsec provider */}
          {!isAuthSec && (
            <>
              <DropdownMenuSeparator />
              <DropdownMenuItem onClick={handleDelete} className="text-destructive">
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </>
          )}
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// Expanded row content component with OIDC provider details
export function ProviderExpandedRow({
  provider,
}: {
  provider: UnifiedAuthProvider | ApiOidcProvider;
}) {
  const isUnified = "provider_type" in provider;
  const isSaml = isUnified && provider.provider_type === "saml";

  // OIDC configuration
  const config = !isSaml
    ? (provider as ApiOidcProvider).provider_config ?? (provider as ApiOidcProvider).endpoints ?? {}
    : {};
  const isComplete = isSaml
    ? Boolean(
        (provider as UnifiedAuthProvider).entity_id && (provider as UnifiedAuthProvider).sso_url
      )
    : OidcProviderTableUtils.isConfigurationComplete(provider as ApiOidcProvider);

  const endpointEntries = isSaml
    ? [
        { label: "Entity ID", value: (provider as UnifiedAuthProvider).entity_id },
        { label: "SSO URL", value: (provider as UnifiedAuthProvider).sso_url },
        { label: "SLO URL", value: (provider as UnifiedAuthProvider).slo_url },
        { label: "Metadata URL", value: (provider as UnifiedAuthProvider).metadata_url },
      ]
    : [
        {
          label: "Auth URL",
          value: config.auth_url ?? (provider as ApiOidcProvider).endpoints?.auth_url,
        },
        {
          label: "Token URL",
          value: config.token_url ?? (provider as ApiOidcProvider).endpoints?.token_url,
        },
        {
          label: "UserInfo URL",
          value: config.user_info_url ?? (provider as ApiOidcProvider).endpoints?.user_info_url,
        },
      ];

  return (
    <div className="px-0 py-0">
      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 lg:grid-cols-3">
        {/* PROVIDER - Details Section */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 border-b border-border pb-2">
            <Globe className="h-4 w-4 text-primary" />
            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-foreground">
                PROVIDER
              </div>
              <div className="text-sm font-medium text-foreground">Details</div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">Provider</div>
              <div className="text-sm text-foreground">
                {OidcProviderTableUtils.formatProviderName(provider.provider_name)}
              </div>
            </div>

            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">Display Name</div>
              <div className="text-sm font-medium text-foreground">{provider.display_name}</div>
            </div>

            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">Status</div>
              <div className="text-sm text-foreground">
                {provider.is_active ? "Active" : "Inactive"}
              </div>
            </div>

            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">Sort Order</div>
              <div className="text-sm font-medium text-foreground">{provider.sort_order}</div>
            </div>
          </div>
        </div>

        {/* CONFIGURATION - Status Section */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 border-b border-border pb-2">
            <Shield className="h-4 w-4 text-green-600" />
            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-foreground">
                CONFIGURATION
              </div>
              <div className="text-sm font-medium text-foreground">Status</div>
            </div>
          </div>

          <div className="space-y-3">
            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">Status</div>
              <div
                className={`text-sm font-medium ${
                  isComplete ? "text-foreground" : "text-destructive"
                }`}
              >
                {isComplete ? "Complete" : "Incomplete"}
              </div>
            </div>

            <div className="space-y-1">
              <div className="text-xs font-medium text-foreground">
                {isSaml ? "Name ID Format" : "Client ID"}
              </div>
              <div className="flex items-center gap-2 min-w-0">
                <code
                  className="flex-1 truncate text-xs font-mono text-foreground"
                  title={
                    isSaml ? (provider as UnifiedAuthProvider).name_id_format : provider.client_id
                  }
                >
                  {isSaml ? (provider as UnifiedAuthProvider).name_id_format : provider.client_id}
                </code>
                <CopyButton
                  text={
                    isSaml
                      ? (provider as UnifiedAuthProvider).name_id_format || ""
                      : provider.client_id
                  }
                  label={isSaml ? "Name ID Format" : "Client ID"}
                  size="sm"
                  variant="ghost"
                  className="flex-shrink-0"
                />
              </div>
            </div>

            {isSaml ? (
              <div className="space-y-1">
                <div className="text-xs font-medium text-foreground">Attribute Mapping</div>
                <div className="space-y-0.5 text-xs text-foreground">
                  {(provider as UnifiedAuthProvider).attribute_mapping ? (
                    Object.entries((provider as UnifiedAuthProvider).attribute_mapping || {}).map(
                      ([key, value]) => (
                        <p key={key}>
                          {key}: {value}
                        </p>
                      )
                    )
                  ) : (
                    <p>No attribute mapping defined</p>
                  )}
                </div>
              </div>
            ) : (
              <div className="space-y-1">
                <div className="text-xs font-medium text-foreground">Scopes</div>
                <div className="text-sm font-medium text-foreground">
                  {config.scopes?.length || 0} configured
                </div>
                <div className="space-y-0.5 text-xs text-foreground">
                  {config.scopes?.length
                    ? config.scopes.map((scope: string) => <p key={scope}>{scope}</p>)
                    : "No scopes defined"}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* ENDPOINTS - Connectivity Section */}
        <div className="space-y-4">
          <div className="flex items-center gap-2 border-b border-border pb-2">
            <LinkIcon className="h-4 w-4 text-blue-600" />
            <div>
              <div className="text-[10px] font-semibold uppercase tracking-wider text-foreground">
                ENDPOINTS
              </div>
              <div className="text-sm font-medium text-foreground">Connectivity</div>
            </div>
          </div>

          <div className="space-y-3">
            {endpointEntries.map((endpoint) => (
              <div key={endpoint.label} className="space-y-1">
                <div className="text-xs font-medium text-foreground">{endpoint.label}</div>
                {endpoint.value ? (
                  <div className="flex items-center gap-2 min-w-0">
                    <code
                      className="flex-1 truncate text-xs text-foreground"
                      title={endpoint.value}
                    >
                      {endpoint.value}
                    </code>
                    <CopyButton
                      text={endpoint.value}
                      label={endpoint.label}
                      size="sm"
                      variant="ghost"
                      className="flex-shrink-0"
                    />
                  </div>
                ) : (
                  <div className="text-xs text-foreground italic">Not configured</div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}

// Column definitions factory
export function createOidcProviderTableColumns(
  actions: OidcProviderTableActions
): ResponsiveColumnDef<ApiOidcProvider, any>[] {
  return [
    {
      id: "provider",
      accessorKey: "display_name",
      header: "Provider",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => {
        return <ProviderCell provider={row.original} />;
      },
      cellClassName: "max-w-0",
    },
    {
      id: "status",
      accessorKey: "is_active",
      header: "Status",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <StatusCell provider={row.original} />,
    },
    {
      id: "configuration",
      accessorKey: "provider_config",
      header: "Configuration",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <ConfigurationCell provider={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => <ActionsCell provider={row.original} actions={actions} />,
      cellClassName: "text-center",
    },
  ];
}

// Dynamic column definitions factory
export function createDynamicOidcProviderTableColumns(
  visibleColumns: string[],
  actions: OidcProviderTableActions
): ResponsiveColumnDef<ApiOidcProvider, any>[] {
  const availableColumns: Record<string, ResponsiveColumnDef<ApiOidcProvider, any>> = {
    provider: {
      id: "provider",
      accessorKey: "display_name",
      header: "Provider",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => {
        return <ProviderCell provider={row.original} />;
      },
    },
    status: {
      id: "status",
      accessorKey: "is_active",
      header: "Status",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <StatusCell provider={row.original} />,
    },
    providerName: {
      id: "providerName",
      accessorKey: "provider_name",
      header: "Provider Type",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm text-foreground">
          {OidcProviderTableUtils.formatProviderName(row.original.provider_name)}
        </span>
      ),
    },
    configuration: {
      id: "configuration",
      accessorKey: "provider_config",
      header: "Configuration",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <ConfigurationCell provider={row.original} />,
    },
    clientId: {
      id: "clientId",
      accessorKey: "client_id",
      header: "Client ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm font-mono">{row.original.client_id.substring(0, 12)}...</span>
      ),
    },
    callbackUrl: {
      id: "callbackUrl",
      accessorKey: "callback_url",
      header: "Callback URL",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm truncate max-w-48" title={row.original.callback_url}>
          {row.original.callback_url}
        </span>
      ),
    },
    scopes: {
      id: "scopes",
      accessorKey: "provider_config.scopes",
      header: "Scopes",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm text-foreground">
          {row.original.provider_config.scopes?.length || 0}
        </span>
      ),
    },
    actions: {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => <ActionsCell provider={row.original} actions={actions} />,
      cellClassName: "text-center",
    },
  };

  return visibleColumns.map((columnId) => availableColumns[columnId]).filter(Boolean);
}

// Available columns for dynamic table configuration
export const AVAILABLE_OIDC_PROVIDER_COLUMNS = {
  // Core provider info
  provider: { label: "Provider", description: "Display name and provider type" },
  status: { label: "Status", description: "Active/Inactive status and configuration completeness" },
  providerName: { label: "Provider Type", description: "OAuth provider type" },

  // Configuration
  configuration: { label: "Configuration", description: "OAuth scopes and setup status" },
  scopes: { label: "Scopes", description: "Number of OAuth scopes configured" },

  // Technical details
  clientId: { label: "Client ID", description: "OAuth client identifier" },
  callbackUrl: { label: "Callback URL", description: "OAuth redirect URI" },

  // Actions
  actions: { label: "Actions", description: "Provider management actions" },
} as const;

// Default visible columns for the OIDC provider table
export const DEFAULT_OIDC_PROVIDER_COLUMNS = [
  "provider",
  "status",
  "configuration",
  "actions",
] as const;

// All available column keys
export const ALL_OIDC_PROVIDER_COLUMN_KEYS = Object.keys(AVAILABLE_OIDC_PROVIDER_COLUMNS) as Array<
  keyof typeof AVAILABLE_OIDC_PROVIDER_COLUMNS
>;

// Helper function to get column metadata
export function getOidcProviderColumnMetadata(columnId: string) {
  return AVAILABLE_OIDC_PROVIDER_COLUMNS[columnId as keyof typeof AVAILABLE_OIDC_PROVIDER_COLUMNS];
}
