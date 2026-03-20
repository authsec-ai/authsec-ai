import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { CardContent } from "../../components/ui/card";
import { Shield, RotateCcw, ChevronsRight } from "lucide-react";
import { IconPlus } from "@tabler/icons-react";
import { Button } from "../../components/ui/button";
import { TableCard } from "../../theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { SDKQuickHelp, AUTHENTICATION_SDK_HELP } from "@/features/sdk";

// Import components
import {
  BulkActionsBar,
  AuthenticationFilterCard,
  AuthTableSkeleton,
} from "./components";
import { AuthProvidersTable } from "./components/AuthProvidersTable";
import { AddAuthMethodModal } from "./components/AddAuthMethodModal";
import { DeleteProviderConfirmDialog } from "./components/DeleteProviderConfirmDialog";

// Import types and API
import type { AuthMethodStatus } from "./types";
import { toast } from "@/lib/toast";
import {
  useShowAuthProvidersQuery,
  useEditClientAuthProviderMutation,
  useUpdateProviderMutation,
  useDeleteProviderMutation,
  type EditClientAuthProviderRequest,
  type UpdateProviderRequest,
  type DeleteProviderRequest,
} from "../../app/api/authMethodApi";
import {
  useUpdateSamlProviderMutation,
  useDeleteSamlProviderMutation,
} from "../../app/api/samlApi";
import { useGetClientsQuery } from "../../app/api/clientApi";
import { SessionManager } from "../../utils/sessionManager";
import { useUnifiedProviders } from "./hooks/useUnifiedProviders";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";

/**
 * Authentication Methods page component - Identity-centric management console
 *
 * Focus areas:
 * - Inventory: Which auth methods exist and are enabled
 * - Reuse: Which services are plugged into each method
 * - Reliability: Login success rates and error tracking
 * - Housekeeping: Secret expiry monitoring
 * - Quick actions: One-click edit/clone/disable/attach operations
 */
export function AuthenticationPage() {
  const navigate = useNavigate();
  const [selectedMethods, setSelectedMethods] = useState<string[]>([]);
  const [isAddMethodModalOpen, setIsAddMethodModalOpen] = useState(false);
  const [filters, setFilters] = useState({
    searchQuery: "",
    providerType: undefined as string | undefined,
    status: undefined as string | undefined,
  });

  // Get session data for OIDC config
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  // Client filtering state - no default selection
  const [selectedClientId, setSelectedClientId] = useState<string>("");

  // Delete dialog state
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    providerId: string;
    providerName?: string;
    providerType?: "oidc" | "saml";
  }>({
    open: false,
    providerId: "",
    providerName: undefined,
    providerType: undefined,
  });

  // Fetch clients using query hook
  const {
    data: clientsResponse,
    isLoading: loadingClients,
    error: clientsError,
  } = useGetClientsQuery(
    tenantId
      ? { tenant_id: tenantId, active_only: false }
      : { tenant_id: "", active_only: false },
    { skip: !tenantId },
  );

  // Mutations
  const [editClientAuthProvider] = useEditClientAuthProviderMutation();
  const [updateOidcProvider] = useUpdateProviderMutation();
  const [deleteOidcProvider, { isLoading: isDeletingOidc }] =
    useDeleteProviderMutation();
  const [updateSamlProvider] = useUpdateSamlProviderMutation();
  const [deleteSamlProvider, { isLoading: isDeletingSaml }] =
    useDeleteSamlProviderMutation();

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["authentication-setup"],
  });

  // Extract clients from response
  const clients = useMemo(() => {
    if (!clientsResponse?.clients) return [];
    return Array.isArray(clientsResponse.clients)
      ? clientsResponse.clients
      : [];
  }, [clientsResponse]);

  // Handle filter changes from the filter card
  const handleFiltersChange = (newFilters: any) => {
    setFilters((prev) => ({
      searchQuery: newFilters.searchQuery ?? prev.searchQuery ?? "",
      providerType: newFilters.providerType,
      status: newFilters.status,
    }));
  };

  // Fetch unified providers (both OIDC and SAML)
  const {
    providers: unifiedProviders,
    isLoading: isProvidersLoading,
    isError: hasProviderError,
    error: providerError,
    refetch: refetchProviders,
  } = useUnifiedProviders({
    tenant_id: tenantId || "",
    client_id: selectedClientId || undefined,
  });

  // Filter unified providers based on search, status, and provider type
  const filteredProviders = useMemo(() => {
    if (!unifiedProviders) return [];
    return unifiedProviders.filter((provider) => {
      // Search across common fields and provider-specific fields
      const matchesSearch =
        !filters.searchQuery ||
        provider.display_name
          ?.toLowerCase()
          .includes(filters.searchQuery.toLowerCase()) ||
        provider.provider_name
          ?.toLowerCase()
          .includes(filters.searchQuery.toLowerCase()) ||
        provider.client_id
          ?.toLowerCase()
          .includes(filters.searchQuery.toLowerCase()) ||
        (provider.provider_type === "oidc" &&
          provider.callback_url
            ?.toLowerCase()
            .includes(filters.searchQuery.toLowerCase())) ||
        (provider.provider_type === "saml" &&
          provider.entity_id
            ?.toLowerCase()
            .includes(filters.searchQuery.toLowerCase())) ||
        (provider.provider_type === "saml" &&
          provider.sso_url
            ?.toLowerCase()
            .includes(filters.searchQuery.toLowerCase()));

      const matchesStatus =
        !filters.status ||
        filters.status === (provider.is_active ? "active" : "inactive");

      const matchesProviderType =
        !filters.providerType ||
        filters.providerType === provider.provider_type;

      return matchesSearch && matchesStatus && matchesProviderType;
    });
  }, [unifiedProviders, filters]);

  // Get selected providers data
  const selectedProviders = useMemo(() => {
    return filteredProviders.filter((p) => selectedMethods.includes(p.id));
  }, [filteredProviders, selectedMethods]);

  // Selection handlers
  const handleSelectAll = () => {
    if (selectedMethods.length === filteredProviders.length) {
      setSelectedMethods([]);
    } else {
      setSelectedMethods(filteredProviders.map((p) => p.id));
    }
  };

  const handleRowSelectionChange = (selectedIds: string[]) => {
    setSelectedMethods(selectedIds);
  };

  const handleClearSelection = () => {
    setSelectedMethods([]);
  };

  // Bulk actions
  const handleBulkAction = (action: string) => {
    // TODO: Implement bulk actions
    console.log(`Bulk action: ${action} on providers:`, selectedProviders);
    setSelectedMethods([]);
  };

  // Provider management functions
  const handleDuplicateProvider = (providerId: string) => {
    // TODO: Implement provider duplication
  };

  const handleDeleteProvider = (providerId: string) => {
    const provider = unifiedProviders.find((item) => item.id === providerId);
    if (!provider) {
      toast.error("Provider not found.");
      return;
    }

    // Open delete confirmation modal
    setDeleteDialog({
      open: true,
      providerId,
      providerName: provider.display_name,
      providerType: provider.provider_type === "saml" ? "saml" : "oidc",
    });
  };

  const handleConfirmDelete = async () => {
    if (!tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      setDeleteDialog({
        open: false,
        providerId: "",
        providerName: undefined,
        providerType: undefined,
      });
      return;
    }

    const provider = unifiedProviders.find(
      (item) => item.id === deleteDialog.providerId,
    );
    if (!provider) {
      toast.error("Provider not found.");
      setDeleteDialog({
        open: false,
        providerId: "",
        providerName: undefined,
        providerType: undefined,
      });
      return;
    }

    try {
      if (provider.provider_type === "saml") {
        // Delete SAML provider
        const samlId = deleteDialog.providerId.replace("saml-", ""); // Remove prefix to get actual ID
        await deleteSamlProvider({
          tenant_id: tenantId,
          provider_id: samlId,
        }).unwrap();
      } else {
        // Delete OIDC provider
        const payload: DeleteProviderRequest = {
          tenant_id: tenantId,
          client_id: provider.client_id,
          provider_name: provider.provider_name,
        };
        await deleteOidcProvider(payload).unwrap();
      }

      toast.success(`${provider.display_name} deleted successfully`);
      setDeleteDialog({
        open: false,
        providerId: "",
        providerName: undefined,
        providerType: undefined,
      });
      refetchProviders();
    } catch (error: any) {
      const message =
        error?.data?.message || `Failed to delete ${provider.display_name}`;
      toast.error(message);
      console.error("Delete provider error:", error);
      // Keep modal open on error so user can retry or cancel
    }
  };

  const handleToggleActive = async (providerId: string, isActive: boolean) => {
    if (!tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      return;
    }

    const provider = unifiedProviders.find((item) => item.id === providerId);
    if (!provider) {
      toast.error("Provider not found.");
      return;
    }

    try {
      if (provider.provider_type === "saml") {
        // Update SAML provider
        const samlId = providerId.replace("saml-", ""); // Remove prefix to get actual ID
        await updateSamlProvider({
          tenant_id: tenantId,
          provider_id: samlId,
          is_active: isActive,
        }).unwrap();
      } else {
        // Update OIDC provider
        const orgId = sessionData?.org_id || "";
        const clientId =
          provider.client_id || provider.hydra_client_id || tenantId;

        const payload: UpdateProviderRequest = {
          tenant_id: tenantId,
          org_id: orgId,
          provider_name: provider.provider_name,
          display_name: provider.display_name,
          client_id: clientId,
          client_secret: "", // Client secret must be provided separately or kept as empty for updates
          auth_url: provider.endpoints?.auth_url || "",
          token_url: provider.endpoints?.token_url || "",
          user_info_url: provider.endpoints?.user_info_url || "",
          scopes: ["openid", "profile", "email"], // Default scopes, adjust as needed
          is_active: isActive,
          updated_by: sessionData?.user?.email || "system",
        };

        await updateOidcProvider(payload).unwrap();
      }

      toast.success(
        `${provider.display_name} ${isActive ? "activated" : "deactivated"}`,
      );
      refetchProviders();
    } catch (error: any) {
      const message =
        error?.data?.message ||
        `Failed to ${isActive ? "activate" : "deactivate"} ${provider.display_name}`;
      toast.error(message);
      console.error("Update provider error:", error);
    }
  };

  const handleViewConfiguration = (providerId: string) => {
    // TODO: Show configuration modal
  };

  const handleTestConnection = (providerId: string) => {
    // TODO: Implement connection test
  };

  const isInitialLoading = isProvidersLoading;

  return (
    <div className="min-h-screen">
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title="Authentication Methods"
          description="Manage identity providers, monitor reliability, and track secret expiry"
          actions={
            <Button
              onClick={() => setIsAddMethodModalOpen(true)}
              data-tour-id="create-auth-method-button"
            >
              <IconPlus className="h-4 w-4 mr-2" />
              Add Method
            </Button>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="Identity Provider Management"
          description="Manage OIDC and SAML authentication methods for secure user identity verification across your applications."
          featuresTitle="Key capabilities:"
          features={[
            { text: "Multiple protocol support (OIDC/SAML)" },
            { text: "Real-time provider status monitoring" },
            { text: "Enterprise SSO integration" },
          ]}
          primaryAction={{
            label: "Read docs",
            onClick: () =>
              window.open(
                "https://docs.authsec.dev/administration/category/authentication-5",
                "_blank",
              ),
            variant: "outline",
            className:
              "bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs",
            icon: ChevronsRight,
          }}
          faqsTitle="Common questions:"
          faqs={[
            {
              id: "oidc-vs-saml",
              question: "OIDC vs SAML - which should I use?",
              answer:
                "OIDC is modern, lightweight, and ideal for mobile/web apps. SAML is enterprise-focused with better support for legacy systems. Most new applications should use OIDC.",
            },
            {
              id: "add-provider",
              question: "How do I add an identity provider?",
              answer:
                "Click 'Add Method', choose OIDC or SAML, then configure the issuer URL, client credentials, and callback URLs. Test the connection before saving.",
            },
            {
              id: "troubleshoot",
              question: "Provider shows as inactive?",
              answer:
                "Check that your issuer URL is accessible, client credentials are correct, and redirect URIs match your application's callback URLs. Review logs for specific error details.",
            },
          ]}
          storageKey="authentication-page"
          dismissible={true}
        />

        {/* Filter Card */}
        <AuthenticationFilterCard
          onFiltersChange={handleFiltersChange}
          initialFilters={filters}
          clients={clients}
          loadingClients={loadingClients}
          selectedClientId={selectedClientId}
          onClientChange={setSelectedClientId}
        />

        {/* Bulk Actions Bar */}
        {selectedMethods.length > 0 && (
          <BulkActionsBar
            selectedProviders={selectedProviders}
            onClearSelection={handleClearSelection}
            onBulkAction={handleBulkAction}
          />
        )}

        {/* Enhanced Authentication Table */}
        <div className="auth-table-container">
          <style>{`
            .auth-table-container [data-slot="table-container"] {
              border: none !important;
              background: transparent !important;
            }
            .auth-table-container [data-slot="table-header"] {
              background: transparent !important;
            }
            .auth-table-container .bg-muted\\/50,
            .auth-table-container .bg-muted\\/30,
            .auth-table-container [class*="bg-muted"] {
              background: transparent !important;
            }
            .auth-table-container .hover\\:bg-muted\\/50:hover,
            .auth-table-container .hover\\:bg-muted\\/30:hover {
              background: rgba(148, 163, 184, 0.1) !important;
            }
            .auth-table-container .shadow-xl {
              box-shadow: none !important;
            }
          `}</style>
          <TableCard className="transition-all duration-500">
            <CardContent variant="flush">
              <div className="relative">
                {isInitialLoading ? (
                  <AuthTableSkeleton rows={6} />
                ) : (
                  <AuthProvidersTable
                    providers={filteredProviders}
                    selectedProviderIds={selectedMethods}
                    onSelectionChange={handleRowSelectionChange}
                    onSelectAll={handleSelectAll}
                    actions={{
                      onDuplicate: handleDuplicateProvider,
                      onDelete: handleDeleteProvider,
                      onToggleActive: handleToggleActive,
                      onViewConfiguration: handleViewConfiguration,
                      onTestConnection: handleTestConnection,
                    }}
                  />
                )}
              </div>
            </CardContent>
          </TableCard>
        </div>
      </div>

      {/* Add Method Modal */}
      <AddAuthMethodModal
        open={isAddMethodModalOpen}
        onOpenChange={setIsAddMethodModalOpen}
      />

      {/* Delete Provider Confirmation Dialog */}
      <DeleteProviderConfirmDialog
        open={deleteDialog.open}
        onOpenChange={(open) => {
          if (!open) {
            setDeleteDialog({
              open: false,
              providerId: "",
              providerName: undefined,
              providerType: undefined,
            });
          }
        }}
        onConfirm={handleConfirmDelete}
        providerId={deleteDialog.providerId}
        providerName={deleteDialog.providerName}
        providerType={deleteDialog.providerType}
        isLoading={isDeletingOidc || isDeletingSaml}
      />

      {/* SDK Quick Help */}
      <SDKQuickHelp
        helpItems={AUTHENTICATION_SDK_HELP}
        title="Authentication SDK"
      />
    </div>
  );
}
