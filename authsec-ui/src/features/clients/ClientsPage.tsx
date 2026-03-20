import React, { useCallback, useMemo, useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import { useAuth } from "../../auth/context/AuthContext";
import { Button } from "../../components/ui/button";
import { CardContent } from "../../components/ui/card";
import { useResponsiveCards } from "../../hooks/use-mobile";
import {

  useGetAllClientsQuery,
  useDeleteClientCompleteMutation,
  useSetClientStatusMutation,

  type EnhancedClientData,
  type GetClientsRequest,
} from "../../app/api/clientApi";
import type {
  ClientWithAuthMethods,
  ClientsFilters,
} from "../../types/entities";
import { SessionManager } from "../../utils/sessionManager";
import { toast } from "react-hot-toast";
import { generateOAuth2AuthorizationUrl } from "../../utils/oauthUtils";

import { FullPageErrorDisplay } from "../../components/ui/error-display";
import { DeleteConfirmDialog } from "./components/DeleteConfirmDialog";
import { ClientAuthMethodsModal } from "./components/ClientAuthMethodsModal";
import { OnboardClientModal } from "./components/OnboardClientModal";

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { Plus, ServerCog, ChevronsRight, Mic } from "lucide-react";

// Import components
import { EnhancedClientsTable } from "./components/EnhancedClientsTable";
import { NewClientSpotlightOverlay } from "./components/NewClientSpotlightOverlay";
import { BulkActionsBar } from "./components/BulkActionsBar";
import { RefreshingSkeleton } from "./components/ClientsPageSkeleton";
import { FilterCard } from "./components/FilterCard";
import { FloatingFAQ } from "./components/FloatingFAQ";
import {
  Tabs,
  TabsList,
  TabsTrigger,
  TabsContent,
} from "../../components/ui/tabs";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../components/ui/table";
import { TableCard, PaginationCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";

/**
 * Utility function to map API EnhancedClientData to ClientWithAuthMethods
 */
function mapClientDataToTableFormat(
  client: EnhancedClientData,
): ClientWithAuthMethods {
  console.log("Mapping enhanced client data:", {
    rawClient: client,
    name: client.name,
    client_name: client.client_name,
    authentication_methods: client.authentication_methods,
    auth_methods_count: client.auth_methods_count,
  });

  // Determine client type based on name patterns
  const inferClientType = (name: string): ClientWithAuthMethods["type"] => {
    const lowerName = name.toLowerCase();
    if (lowerName.includes("mcp") || lowerName.includes("server"))
      return "mcp_server";
    if (lowerName.includes("app") || lowerName.includes("application"))
      return "app";
    if (lowerName.includes("api") || lowerName.includes("service"))
      return "api";
    return "other";
  };

  const clientType = inferClientType(client.name || "");
  const email = client.email ?? "";
  const emailPrefix = email
    ? email.includes("@")
      ? email.split("@")[0]
      : email
    : "unknown";
  const tenantSuffix = client.tenant_id
    ? client.tenant_id.slice(-8)
    : "unknown";
  const projectSuffix = client.project_id
    ? client.project_id.slice(-8)
    : "unknown";
  const isActive = client.active ?? client.status?.toLowerCase() === "active";
  const apiTags = Array.isArray(client.tags)
    ? client.tags
    : typeof client.tags === "string"
      ? (client.tags as string)
          .split(",")
          .map((tag: string) => tag.trim())
          .filter(Boolean)
      : [];

  // Handle authentication_methods from EnhancedClientData
  // It can be either an array of objects or a string
  const authMethods: Array<{ id: string; name: string; isDefault: boolean }> =
    [];

  if (Array.isArray(client.authentication_methods)) {
    // Array of auth method objects
    client.authentication_methods.forEach((method, index) => {
      authMethods.push({
        id: method.id || `auth-${index}`,
        name: method.name || method.type || "Unknown",
        isDefault: method.is_default ?? index === 0, // First one is default if not specified
      });
    });
  } else if (typeof client.authentication_methods === "string") {
    // String representation (e.g., "password,oidc")
    const methodNames = client.authentication_methods
      .split(",")
      .map((m) => m.trim());
    methodNames.forEach((methodName, index) => {
      authMethods.push({
        id: `auth-${index}`,
        name: methodName,
        isDefault: index === 0,
      });
    });
  }

  console.log("Mapped authentication methods:", authMethods);

  return {
    id: client.client_id,
    workspace_id: client.tenant_id,
    secret_id: client.id,
    name: client.name || client.client_name || "Unnamed Client",
    description:
      client.description ||
      `Client: ${client.name || client.client_name || "Unnamed Client"}`,
    type: clientType,
    tags:
      apiTags.length > 0
        ? apiTags.join(",")
        : `email:${emailPrefix},tenant:${tenantSuffix},project:${projectSuffix}`,
    authentication_type: "custom" as const,
    metadata: {
      project_id: client.project_id,
      tenant_id: client.tenant_id,
      original_id: client.id,
      email,
      org_id: client.org_id,
      hydra_client_id: client.hydra_client_id,
      raw_client: {
        ...client,
        mfa_enabled: true, // Override: Always show MFA as ON
      },
      authentication_methods: authMethods,
      auth_methods_count: client.auth_methods_count,
      user_count: client.user_count,
      last_modified_at: (client as any).last_modified_at,
    },
    roles: [],
    mfa_config: null,
    successful_authentications: null, // No data available from API
    denied_authentications: null, // No data available from API
    view_policies_applicable: [],
    endpoint: `/clientms/clients/${client.client_id}`,
    access_status: isActive ? ("active" as const) : ("disabled" as const),
    access_level: "internal" as const,
    total_requests: null, // No data available from API
    last_accessed: (client as any).last_modified_at || client.updated_at,
    updated_at: client.updated_at,
    created_by: email,
    attachedMethods: authMethods,
  };
}

/**
 * Clients page component - Manage MCP Servers and AI Agents
 *
 * Features:
 * - Client onboarding and management
 * - API usage analytics
 * - Client configuration management
 * - Pro tips and recommendations
 * - Enhanced metrics and analytics
 */
export function ClientsPage() {
  const navigate = useNavigate();
  const { user, currentProject } = useAuth();
  const [selectedClients, setSelectedClients] = useState<string[]>([]);
  const [drawerClient, setDrawerClient] =
    useState<ClientWithAuthMethods | null>(null);
  const [clients, setClients] = useState<EnhancedClientData[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filtersState, setFiltersState] = useState<Partial<ClientsFilters>>({});
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    clientId: string;
    clientName?: string;
  }>({
    open: false,
    clientId: "",
    clientName: undefined,
  });
  const [isDeleting, setIsDeleting] = useState(false);
  const [authDialogClient, setAuthDialogClient] =
    useState<ClientWithAuthMethods | null>(null);
  const [showOnboardModal, setShowOnboardModal] = useState(false);
  const [newlyCreatedClientId, setNewlyCreatedClientId] = useState<string | null>(null);
  const [newClientStep, setNewClientStep] = useState(0);
  const [tablePageIndex, setTablePageIndex] = useState<number | undefined>(undefined);
  const [queryPage, setQueryPage] = useState(1);

  const handleNextNewClientStep = useCallback(() => {
    setNewClientStep((s) => {
      if (s >= 2) {
        setNewlyCreatedClientId(null);
        setNewClientStep(0);
        setTablePageIndex(undefined);
        return 0;
      }
      return s + 1;
    });
  }, []);

  const handleDismissNewClient = useCallback(() => {
    setNewlyCreatedClientId(null);
    setNewClientStep(0);
    setTablePageIndex(undefined);
    // Do NOT reset queryPage — keep the user on the current page
  }, []);

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["clients-onboarding"],
  });

  // Animation refs

  // AuthSec API integration
  const [deleteClientComplete] = useDeleteClientCompleteMutation();
  const [setClientStatus] = useSetClientStatusMutation();

  // Get session data for tenant ID
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  const queryArgs = useMemo(() => {
    if (!tenantId) return undefined;
    const cleanedFilters: Record<string, any> = {};
    Object.entries(filtersState || {}).forEach(([key, value]) => {
      if (
        value === undefined ||
        value === null ||
        (typeof value === "string" && value.trim() === "") ||
        (Array.isArray(value) && value.length === 0)
      ) {
        return;
      }
      cleanedFilters[key] = value;
    });

    return {
      tenant_id: tenantId,
      active_only: false,
      filters: cleanedFilters,
      page: queryPage,
      limit: 10,
    } as GetClientsRequest;
  }, [tenantId, filtersState, queryPage]);

  // Use getAllClients to get enhanced data with authentication methods
  const {
    data: clientsResponse,
    isLoading: clientsLoading,
    error: clientsError,
    refetch: refetchClients,
  } = useGetAllClientsQuery(queryArgs as GetClientsRequest, {
    skip: !queryArgs,
    refetchOnMountOrArgChange: true,
    // Disable aggressive refetching to prevent infinite loops
    refetchOnFocus: false,
    refetchOnReconnect: false,
  });

  // Load clients when component mounts or when data changes
  React.useEffect(() => {
    if (!tenantId) {
      // Allow the UI to render with empty data when no tenant/session is present
      setClients([]);
      setLoading(false);
      setError(null);
      return;
    }

    setLoading(clientsLoading);

    if (clientsResponse) {
      console.log("Raw clients API response:", clientsResponse);
      console.log("First client from API:", clientsResponse.clients?.[0]);

      setClients(
        Array.isArray(clientsResponse.clients) ? clientsResponse.clients : [],
      );
      setError(null);
      return;
    }

    if (!clientsLoading && !clientsResponse) {
      // No data returned yet; ensure we present an empty table without errors.
      setClients([]);
      if (!clientsError) {
        setError(null);
      }
    }

    if (clientsError && !clientsLoading) {
      console.error("Failed to load clients:", clientsError);

      const errorWithData = clientsError as {
        status?: number;
        data?: { message?: string; error?: string };
      };
      if (
        errorWithData?.status === 500 &&
        errorWithData?.data?.message?.includes("user not found")
      ) {
        setError(
          "User not found in AuthSec system. Please complete OIDC login flow first.",
        );
      } else {
        setError(
          errorWithData?.data?.message ||
            errorWithData?.data?.error ||
            `API Error: ${errorWithData?.status || "Unknown"}`,
        );
      }
    }
  }, [tenantId, clientsResponse, clientsError, clientsLoading]); // Removed sessionData from dependencies

  // Create pagination data
  const pagination = React.useMemo(() => {
    if (clientsResponse?.pagination) {
      const { page, total_pages, limit, total } = clientsResponse.pagination;
      const currentPage = page ?? 1;
      const pageSize = limit ?? clients.length;
      const totalItems = total ?? clients.length;

      return {
        currentPage,
        totalPages: total_pages ?? 1,
        pageSize,
        total: totalItems,
        hasMore: Boolean(
          totalItems && pageSize && currentPage * pageSize < totalItems,
        ),
      };
    }

    return {
      currentPage: 1,
      totalPages: 1,
      pageSize: clients.length,
      total: clients.length,
      hasMore: false,
    };
  }, [clients.length, clientsResponse?.pagination]);

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  const getProjectName = () => {
    return currentProject?.name || "your project";
  };

  // Create display clients from AuthSec API data with client-side filtering
  const displayClients = React.useMemo(() => {
    if (clients.length === 0) {
      return []; // Return empty array if no clients
    }

    // Convert AuthSec client objects to displayable format
    let filteredClients = clients.map(mapClientDataToTableFormat);

    // Apply client-side filters as fallback (in case API doesn't support all filters)
    if (filtersState) {
      filteredClients = filteredClients.filter((client) => {
        // Search filter (name, email, endpoint)
        if (filtersState.search || filtersState.name || filtersState.email) {
          const searchTerm = (
            filtersState.search ||
            filtersState.name ||
            filtersState.email ||
            ""
          ).toLowerCase();
          const matchesName = client.name?.toLowerCase().includes(searchTerm);
          const matchesEmail = client.metadata?.email
            ?.toLowerCase()
            .includes(searchTerm);
          const matchesEndpoint = client.endpoint
            ?.toLowerCase()
            .includes(searchTerm);
          const matchesClientId = client.id?.toLowerCase().includes(searchTerm);

          if (
            !matchesName &&
            !matchesEmail &&
            !matchesEndpoint &&
            !matchesClientId
          ) {
            return false;
          }
        }

        // Status filter
        if (filtersState.status) {
          const statusMatch =
            client.access_status === filtersState.access_status ||
            client.metadata?.raw_client?.status?.toLowerCase() ===
              filtersState.status.toLowerCase();
          if (!statusMatch) {
            return false;
          }
        }

        return true;
      });
    }

    return filteredClients;
  }, [clients, filtersState]);

  // Client-side fallback: compute page from displayClients index
  // (skipped when server-side navigation already set tablePageIndex in onSuccess)
  useEffect(() => {
    if (!newlyCreatedClientId || !displayClients.length) return;
    if (tablePageIndex !== undefined) return; // server-side nav already handled in onSuccess
    const PAGE_SIZE = 10;
    const idx = displayClients.findIndex((c) => {
      const raw = (c.metadata?.raw_client as any) ?? c;
      return (raw?.client_id || c.id) === newlyCreatedClientId;
    });
    if (idx === -1) return;
    setTablePageIndex(Math.floor(idx / PAGE_SIZE) + 1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [newlyCreatedClientId, displayClients]); // tablePageIndex intentionally excluded to avoid loop

  const handleClearSelection = () => {
    setSelectedClients([]);
  };

  // Delete client - show dialog
  const handleDeleteClient = (clientId: string) => {
    // Find the client to get its name
    const client = clients.find((c) => c.client_id === clientId);
    setDeleteDialog({
      open: true,
      clientId,
      clientName: client?.name || client?.client_name,
    });
  };

  // Confirm delete client
  const handleConfirmDelete = async () => {
    const sessionData = SessionManager.getSession();
    const tenantId = sessionData?.tenant_id;
    if (!tenantId) {
      toast.error("Missing tenant context");
      return;
    }

    setIsDeleting(true);
    try {
      await deleteClientComplete({
        tenant_id: tenantId,
        client_id: deleteDialog.clientId,
      }).unwrap();
      toast.success("Client deleted successfully");
      // Optimistically update list
      setClients((prev) =>
        prev.filter((client) => client.client_id !== deleteDialog.clientId),
      );
      // Explicitly refetch to ensure fresh data
      refetchClients();
    } catch (err: unknown) {
      console.error("Delete failed", err);
      const errorWithData = err as { data?: { message?: string } };
      toast.error(errorWithData?.data?.message || "Failed to delete client");
    } finally {
      setIsDeleting(false);
    }
  };

  const handleToggleStatus = async (clientId: string) => {
    const sessionData = SessionManager.getSession();
    const tenantId = sessionData?.tenant_id;

    if (!tenantId) {
      toast.error("Missing tenant context");
      return;
    }

    // Find current client to get its current status
    const currentClient = clients.find((c) => c.client_id === clientId);
    if (!currentClient) {
      toast.error("Client not found");
      return;
    }

    // Validate client ID format (should be UUID)
    const uuidRegex =
      /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;
    if (!uuidRegex.test(clientId)) {
      toast.error("Invalid client ID format");
      return;
    }

    try {
      const newStatus = !currentClient.active;
      await setClientStatus({
        tenant_id: tenantId,
        client_id: clientId,
        active: newStatus,
      }).unwrap();

      // Optimistically update the client in the list
      setClients((prev) =>
        prev.map((client) =>
          client.client_id === clientId
            ? {
                ...client,
                active: newStatus,
                updated_at: new Date().toISOString(),
              }
            : client,
        ),
      );

      toast.success(
        `Client ${newStatus ? "activated" : "deactivated"} successfully`,
      );

      // Explicitly refetch to ensure fresh data
      refetchClients();
    } catch (error: unknown) {
      console.error("Failed to toggle client status:", error);
      const errorWithData = error as { data?: { message?: string } };
      toast.error(
        errorWithData?.data?.message || "Failed to toggle client status",
      );
    }
  };

  const handlePageChange = useCallback((page: number) => {
    setQueryPage(page);
    setTablePageIndex(page);
  }, []);

  const setFilters = useCallback((next: Partial<ClientsFilters>) => {
    setFiltersState(next);
    setQueryPage(1);
    setTablePageIndex(1);
  }, []);

  const handleCreateClient = () => {
    setShowOnboardModal(true);
  };

  const handleAddAuthMethod = useCallback(
    (clientId: string) => {
      navigate("/authentication/create", {
        state: { prefillClientId: clientId },
      });
    },
    [navigate],
  );

  const handleShowAuthMethods = useCallback(
    (clientRecord: ClientWithAuthMethods) => {
      console.info("[ClientsPage] Edit Auth Methods clicked", {
        clientId: clientRecord.id,
        name: clientRecord.name,
      });
      setAuthDialogClient(clientRecord);
    },
    [],
  );

  const handlePreviewLogin = useCallback(async (clientId: string) => {
    const currentSession = SessionManager.getSession();

    // Priority: 1) session tenant_domain, 2) extract from current hostname
    let tenantDomainForOAuth = currentSession?.tenant_domain;
    let tenantDomainFromHostname: string | undefined;

    // Only extract from hostname if not found in session
    if (!tenantDomainForOAuth) {
      // Extract from hostname: dec10.app.authsec.dev -> dec10
      const hostname = window.location.hostname;
      const hostParts = hostname.split(".");
      if (
        hostParts.length >= 4 &&
        hostParts[0] !== "app" &&
        hostParts[0] !== "www"
      ) {
        tenantDomainFromHostname = hostParts[0];
        tenantDomainForOAuth = tenantDomainFromHostname;
      }
    }

    // eslint-disable-next-line no-console
    console.log("[PreviewLogin] 🔐 Generating OAuth URL with:", {
      clientId,
      tenantDomainFromSession: currentSession?.tenant_domain,
      tenantDomainFromHostname,
      finalTenantDomain: tenantDomainForOAuth,
    });

    try {
      // Generate the OAuth2 authorization URL with PKCE
      const { authorizationUrl } = await generateOAuth2AuthorizationUrl({
        clientId,
        tenantDomain: tenantDomainForOAuth,
        scopes: ["openid", "profile", "email"],
      });

      window.open(authorizationUrl, "_blank", "noopener,noreferrer");
      toast.success("Opening end-user login preview in a new tab");
    } catch (error) {
      console.error("Failed to generate OAuth2 URL:", error);
      toast.error("Failed to generate login preview URL");
    }
  }, []);

  const handleConfigureVoiceAgent = useCallback(
    (clientId: string) => {
      navigate(`/clients/voice-agent?clientId=${clientId}`);
    },
    [navigate],
  );

  const handleOpenVoiceAgentFromHeader = useCallback(() => {
    navigate("/clients/voice-agent");
  }, [navigate]);

  // Bulk actions
  const handleBulkAction = async (action: string) => {
    // TODO: Replace with actual service calls when connected to backend
    switch (action) {
      case "enable":
        // await ClientsService.bulkUpdateStatus(selectedClients, "active");
        break;
      case "disable":
        // await ClientsService.bulkUpdateStatus(selectedClients, "disabled");
        break;
      case "bulk-assign-roles":
        // TODO: Show role selection modal
        // await ClientsService.bulkAssignRoles(selectedClients, selectedRoles);
        break;
      case "bulk-configure-mfa":
        // TODO: Show MFA configuration modal
        // await ClientsService.bulkConfigureMFA(selectedClients, mfaConfig);
        break;
      case "bulk-update-auth-type":
        // TODO: Show auth type selection modal
        // await ClientsService.bulkUpdateAuthType(selectedClients, selectedAuthType);
        break;
      case "bulk-reset-auth-stats":
        // await ClientsService.bulkResetAuthStats(selectedClients);
        break;
      case "bulk-duplicate":
        // TODO: Implement bulk duplication logic
        break;
      case "bulk-export-config":
        // TODO: Generate and download configuration export
        break;
      case "bulk-security-report":
        // TODO: Generate and download security report
        break;
      case "bulk-delete":
        // TODO: Show confirmation dialog
        // await ClientsService.bulkDeleteClients(selectedClients);
        break;
      default:
        // Unknown action
        break;
    }

    setSelectedClients([]);
  };

  return (
    <div className="min-h-screen" ref={mainAreaRef} data-page="clients-mcp">
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title="MCP Servers / AI Agents"
          description={`Manage MCP Servers and AI Agents for ${getProjectName()}`}
          actions={
            <div className="flex flex-wrap items-center justify-end gap-2">
              <Button
                variant="outline"
                onClick={handleOpenVoiceAgentFromHeader}
                className="admin-tonal-cta gap-2 shadow-none"
                data-tone="voice"
                data-tour-id="add-voice-agent-button"
              >
                <Mic className="h-4 w-4" />
                Add Voice Agent
              </Button>
              <Button
                onClick={() => setShowOnboardModal(true)}
                className="shadow-none"
                data-tour-id="onboard-button"
              >
                <Plus className="mr-2 h-4 w-4" />
                Onboard Client
              </Button>
            </div>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="Secure your MCP Servers and AI Agents"
          description="Enterprise-grade authentication for AI agents with OAuth 2.0, SPIFFE/SPIRE identity, and RBAC."
          featuresTitle="Key features:"
          features={[
            { text: "OAuth 2.1 with PKCE support" },
            { text: "SPIFFE/SPIRE workload identity" },
            { text: "Role-based access control" },
          ]}
          primaryAction={{
            label: "Read docs",
            onClick: () =>
              window.open(
                "https://docs.authsec.dev/administration/clients/",
                "_blank",
              ),
            variant: "outline",
            className:
              "border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] text-[var(--editorial-text-1)] hover:bg-[var(--editorial-panel-soft)] shadow-none h-8 px-3 text-xs",
            icon: ChevronsRight,
          }}
          storageKey="clients-page-info"
          dismissible={true}
        />

        {/* Filter/Search Card */}
        <div data-tour-id="filter-card">
          <FilterCard setFilters={setFilters} />
        </div>

        {/* Table */}
        <div className="clients-table-container" data-tour-id="clients-table">
          <TableCard className="transition-all duration-500">
            <CardContent variant="flush">
              {error ? (
                <div className="p-6 text-center">
                  <div className="flex flex-col items-center space-y-4">
                    <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                      <ServerCog className="h-8 w-8 text-red-600 dark:text-red-400" />
                    </div>
                    <div>
                      <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                        Unable to Load Clients
                      </h3>
                      <p className="text-red-700 dark:text-red-300 mt-1">
                        {error}
                      </p>
                    </div>
                  </div>
                </div>
              ) : (
                <div className="relative">
                  <EnhancedClientsTable
                    data={displayClients}
                    selectedClients={selectedClients}
                    onSelectionChange={setSelectedClients}
                    onDeleteClient={handleDeleteClient}
                    onCreateClient={handleCreateClient}
                    onToggleStatus={handleToggleStatus}
                    onViewSDK={(clientId) =>
                      navigate(`/sdk/clients/${clientId}`)
                    }
                    onAddAuthMethod={handleAddAuthMethod}
                    onShowAuthMethods={handleShowAuthMethods}
                    onPreviewLogin={handlePreviewLogin}
                    onConfigureVoiceAgent={handleConfigureVoiceAgent}
                    newClientId={newlyCreatedClientId ?? undefined}
                    newClientStep={newClientStep}
                    onNextNewClientStep={handleNextNewClientStep}
                    onDismissNewClient={handleDismissNewClient}
                    pageIndex={tablePageIndex}
                    onPageIndexChange={handlePageChange}
                    serverTotalItems={pagination.total}
                  />
                  {loading && displayClients.length > 0 && (
                    <RefreshingSkeleton />
                  )}
                </div>
              )}
            </CardContent>
          </TableCard>
        </div>
      </div>

      {/* Bulk Actions Bar */}
      {selectedClients.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedClients.length}
          onClearSelection={handleClearSelection}
          onBulkAction={handleBulkAction}
        />
      )}

      <ClientAuthMethodsModal
        open={Boolean(authDialogClient)}
        client={authDialogClient}
        onClose={() => setAuthDialogClient(null)}
      />

      {/* Spotlight overlay for new client button guidance */}
      <NewClientSpotlightOverlay
        isActive={!!newlyCreatedClientId}
        onDismiss={handleDismissNewClient}
      />

      {/* Onboard Client Modal */}
      <OnboardClientModal
        isOpen={showOnboardModal}
        onClose={() => setShowOnboardModal(false)}
        preventNavigation={true}
        onSuccess={(clientId) => {
          // Compute the page where the new client appears (oldest-first sort, new item at end)
          const PAGE_SIZE = 10;
          const targetPage = Math.max(1, Math.ceil((pagination.total + 1) / PAGE_SIZE));
          setQueryPage(targetPage);       // RTK Query auto-refetches with new page
          setTablePageIndex(targetPage);  // Jump table UI to that page
          setNewClientStep(0);
          setNewlyCreatedClientId(clientId);
        }}
      />

      {/* Drawer (expanded details) */}
      {drawerClient && (
        <div className="fixed inset-0 z-50 bg-black/40 flex items-center justify-center">
          <div className="bg-background rounded-lg shadow-xl p-6 max-w-2xl w-full">
            <div className="flex justify-between items-center mb-4">
              <div className="font-bold text-base">{drawerClient.name}</div>
              <Button variant="ghost" onClick={() => setDrawerClient(null)}>
                Close
              </Button>
            </div>
            <Tabs defaultValue="overview">
              <TabsList className="mb-4">
                <TabsTrigger value="overview">Overview</TabsTrigger>
                <TabsTrigger value="config">Config</TabsTrigger>
                <TabsTrigger value="methods">Attached Methods</TabsTrigger>
                <TabsTrigger value="audit">Audit</TabsTrigger>
              </TabsList>
              <TabsContent value="overview">
                <div className="space-y-1">
                  <div>
                    <b>Client ID:</b> {drawerClient.id}
                  </div>
                  <div>
                    <b>Access Status:</b> {drawerClient.access_status}
                  </div>
                  <div>
                    <b>Default Method:</b>{" "}
                    {drawerClient.attachedMethods.find((m) => m.isDefault)
                      ?.name || "—"}
                  </div>
                  <div className="flex items-center gap-2">
                    <b>Endpoint:</b>{" "}
                    <span className="font-mono text-xs">
                      {drawerClient.endpoint}
                    </span>
                  </div>
                  <div>
                    <b>Last Accessed:</b>{" "}
                    {drawerClient.last_accessed || "Never"}
                  </div>
                  <div>
                    <b>Created by:</b> {drawerClient.created_by || "System"}
                  </div>
                </div>
              </TabsContent>
              <TabsContent value="config">
                <div>Config tab (coming soon)</div>
              </TabsContent>
              <TabsContent value="methods">
                <div className="mb-2 flex justify-between items-center">
                  <div className="font-semibold">Attached Methods</div>
                  <Button
                    size="sm"
                    variant="outline"
                    onClick={() => alert("Attach more modal (mock)")}
                  >
                    Attach more
                  </Button>
                </div>
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Attached</TableHead>
                      <TableHead>Name</TableHead>
                      <TableHead>Env</TableHead>
                      <TableHead>Default?</TableHead>
                      <TableHead>Toggle</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {(drawerClient.attachedMethods || []).map((m) => (
                      <TableRow key={m.id}>
                        <TableCell>✅</TableCell>
                        <TableCell>{m.name}</TableCell>
                        <TableCell>{drawerClient.access_level}</TableCell>
                        <TableCell>{m.isDefault ? "⭐" : ""}</TableCell>
                        <TableCell>
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={m.isDefault}
                            onClick={() => alert("Detach (mock)")}
                          >
                            Detach
                          </Button>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </TabsContent>
              <TabsContent value="audit">
                <div>Audit tab (coming soon)</div>
              </TabsContent>
            </Tabs>
          </div>
        </div>
      )}

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={deleteDialog.open}
        onOpenChange={(open) => setDeleteDialog((prev) => ({ ...prev, open }))}
        onConfirm={handleConfirmDelete}
        clientId={deleteDialog.clientId}
        clientName={deleteDialog.clientName}
        isLoading={isDeleting}
      />

      {/* Floating FAQ */}
      <FloatingFAQ />
    </div>
  );
}
