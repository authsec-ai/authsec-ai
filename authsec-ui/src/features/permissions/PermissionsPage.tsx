import React, { useState, useMemo, useEffect } from "react";
import { CardContent } from "@/components/ui/card";
import { toast } from "@/lib/toast";
import { Plus, RefreshCw, AlertTriangle, Search } from "lucide-react";
import {
  useGetPermissionsQuery,
  useDeletePermissionsMutation,
} from "@/app/api/permissionsApi";
import {
  useGetAdminPermissionResourcesQuery,
  useGetEndUserPermissionResourcesQuery,
} from "@/app/api/permissionsResourcesApi";
import { baseApi } from "@/app/api/baseApi";
import { resolveTenantId } from "@/utils/workspace";
import { useResponsiveCards } from "@/hooks/use-mobile";
import { EnhancedPermissionsTable } from "./components/EnhancedPermissionsTable";
import { PermissionsFilterCard } from "./components/PermissionsFilterCard";
import { BulkActionsBar } from "./components/BulkActionsBar";
import { CreatePermissionModal } from "./components/CreatePermissionModal";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { useDispatch } from "react-redux";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
  type ResponsiveColumnDef,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { SDKQuickHelp, PERMISSIONS_SDK_HELP } from "@/features/sdk";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { Shield, Lock, Users } from "lucide-react";

export interface PermissionsQueryParams {
  searchQuery?: string;
  actionFilter?: string;
  resourceFilter?: string;
}

interface SimpleResource {
  id: string;
  name: string;
}

/**
 * Permissions page component - Manage RBAC permissions (Role + Scope + Resource)
 *
 * Features:
 * - Define permissions linking roles, scopes, and resources
 * - Visual permission builder
 * - Bulk operations
 * - Real-time data from AuthSec API
 */
export function PermissionsPage() {
  const contextualNavigate = useContextualNavigate();
  const standardNavigate = useNavigate();
  const { isAdmin, audience } = useRbacAudience();
  const location = useLocation();
  const dispatch = useDispatch();
  const [searchParams, setSearchParams] = useSearchParams();
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const [filters, setFilters] = useState<Partial<PermissionsQueryParams>>({});
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [activeTab, setActiveTab] = useState<"permissions" | "resources">(
    "permissions",
  );
  const [resourceSearchQuery, setResourceSearchQuery] = useState("");

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["permissions-management"],
  });

  // API data fetching
  const tenantId = resolveTenantId();

  // audience in query params triggers refetch on context change
  const {
    data: permissions = [],
    isLoading: permissionsLoading,
    error: permissionsError,
  } = useGetPermissionsQuery(
    { tenant_id: tenantId || "", audience },
    {
      skip: !tenantId,
    },
  );

  // Invalidate cache when audience changes to force fresh API call
  useEffect(() => {
    if (tenantId) {
      dispatch(baseApi.util.invalidateTags(["UnifiedRBACPermission"]));
    }
  }, [audience, dispatch, tenantId]);

  // Auto-open modal if query param present (from wizard)
  useEffect(() => {
    console.log(
      "[PermissionsPage] Query param effect - openModal:",
      searchParams.get("openModal"),
    );
    console.log("[PermissionsPage] Current location.state:", location.state);
    if (searchParams.get("openModal") === "create") {
      setCreateModalOpen(true);
      // Clean up query param while preserving location state
      searchParams.delete("openModal");
      setSearchParams(searchParams, { replace: true, state: location.state });
    }
  }, [searchParams, setSearchParams, location.state]);

  // Handle modal close with wizard awareness
  const handlePermissionModalSuccess = () => {
    console.log("[PermissionsPage] Permission created successfully");
    console.log("[PermissionsPage] location.state:", location.state);
    // Don't close modal here - it will be closed by onOpenChange
    // If coming from wizard, navigate back to root with success flag
    if (location.state?.fromWizard) {
      console.log(
        "[PermissionsPage] Navigating to / with permissionCreated flag",
      );
      standardNavigate("/", { state: { permissionCreated: true } });
    } else {
      console.log("[PermissionsPage] Not from wizard, skipping navigation");
    }
  };

  const [deletePermissions] = useDeletePermissionsMutation();
  const {
    data: adminResources = [],
    isLoading: adminResourcesLoading,
    error: adminResourcesError,
  } = useGetAdminPermissionResourcesQuery(undefined, {
    skip: !isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: endUserResources = [],
    isLoading: endUserResourcesLoading,
    error: endUserResourcesError,
  } = useGetEndUserPermissionResourcesQuery(tenantId || "", {
    skip: isAdmin || !tenantId,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Permissions and Resources",
            subtitle: "Create permissions and resources",
            ctaLabel: "Create Permission and resources",
          }
        : {
            title: "Permissions and Resources",
            subtitle: "Create permissions and resources",
            ctaLabel: "Create Permission and resources",
          },
    [isAdmin],
  );

  // Apply client-side filtering
  const filteredPermissions = useMemo(() => {
    let result = [...permissions];

    if (filters.searchQuery) {
      const query = filters.searchQuery.toLowerCase();
      result = result.filter(
        (p) =>
          p.action?.toLowerCase().includes(query) ||
          p.resource?.toLowerCase().includes(query) ||
          p.description?.toLowerCase().includes(query) ||
          p.full_permission_string?.toLowerCase().includes(query),
      );
    }

    if (filters.actionFilter) {
      result = result.filter((p) => p.action === filters.actionFilter);
    }

    if (filters.resourceFilter) {
      result = result.filter((p) => p.resource === filters.resourceFilter);
    }

    return result;
  }, [permissions, filters]);

  // Extract error message
  const errorMessage = permissionsError
    ? (permissionsError as any)?.data?.message ||
      "Failed to fetch permissions data"
    : null;

  const showInitialSkeleton =
    permissionsLoading && filteredPermissions.length === 0 && !errorMessage;

  const resources = useMemo<SimpleResource[]>(() => {
    const rawResources = isAdmin ? adminResources : endUserResources;
    if (!Array.isArray(rawResources)) return [];

    return (rawResources as string[]).map((name, index) => ({
      id: name ?? `resource-${index}`,
      name: name ?? `Resource ${index + 1}`,
    }));
  }, [adminResources, endUserResources, isAdmin]);

  const filteredResources = useMemo(() => {
    let result = [...resources];

    if (resourceSearchQuery) {
      const query = resourceSearchQuery.toLowerCase();
      result = result.filter((resource) =>
        resource.name.toLowerCase().includes(query),
      );
    }

    return result;
  }, [resourceSearchQuery, resources]);

  const resourcesLoading = isAdmin
    ? adminResourcesLoading
    : endUserResourcesLoading;
  const resourcesError = isAdmin ? adminResourcesError : endUserResourcesError;

  const resourcesErrorMessage = resourcesError
    ? (resourcesError as any)?.data?.message || "Failed to fetch resources data"
    : null;

  const showResourcesSkeleton =
    resourcesLoading &&
    filteredResources.length === 0 &&
    !resourcesErrorMessage;

  const resourceColumns: ResponsiveColumnDef<SimpleResource, unknown>[] =
    useMemo(
      () => [
        {
          id: "resource",
          header: "Resource",
          cell: ({ row }) => (
            <p
              className="truncate text-sm font-medium text-foreground"
              title={row.original.name}
            >
              {row.original.name}
            </p>
          ),
          cellClassName: "max-w-0",
          resizable: true,
          responsive: true,
        },
      ],
      [],
    );

  const resourcesTableConfig: ResponsiveTableConfig<SimpleResource> = {
    data: filteredResources,
    columns: resourceColumns,
    features: {
      selection: false,
      dragDrop: false,
      expandable: false,
      pagination: true,
      sorting: true,
      resizing: true,
    },
    pagination: {
      pageSize: 10,
      pageSizeOptions: [5, 10, 25, 50, 100],
    },
    getRowId: (row) => row.id,
  };

  // Selection handlers
  const handleSelectPermission = (permissionId: string) => {
    setSelectedPermissions((prev) =>
      prev.includes(permissionId)
        ? prev.filter((id) => id !== permissionId)
        : [...prev, permissionId],
    );
  };

  const handleSelectAll = () => {
    setSelectedPermissions((prev) =>
      prev.length === filteredPermissions.length
        ? []
        : filteredPermissions.map((p) => p.id),
    );
  };

  const handleClearSelection = () => setSelectedPermissions([]);

  const handleCreatePermission = () => setCreateModalOpen(true);

  const handleTabChange = (value: string) =>
    setActiveTab(value as "permissions" | "resources");

  const handleBulkDelete = async () => {
    if (!selectedPermissions.length) {
      return;
    }

    try {
      await deletePermissions({
        tenant_id: tenantId || "",
        permission_ids: selectedPermissions,
      }).unwrap();

      toast.success(`Deleted ${selectedPermissions.length} permissions`);
      setSelectedPermissions([]);
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete permissions");
    }
  };

  return (
    <div className="min-h-screen" ref={mainAreaRef}>
      <div className="p-6 max-w-10xl mx-auto">
        <Tabs
          value={activeTab}
          onValueChange={handleTabChange}
          className="space-y-4"
        >
          <div className="space-y-4">
            {/* Header */}
            <PageHeader
              title={audienceCopy.title}
              description={audienceCopy.subtitle}
              actionsPosition="below"
              actions={
                <div className="flex flex-col sm:flex-row items-stretch sm:items-center gap-3 w-full">
                  <TabsList className="w-full sm:w-auto grid grid-cols-2 sm:inline-flex h-auto flex-1 min-w-0">
                    <TabsTrigger value="permissions">Permissions</TabsTrigger>
                    <TabsTrigger value="resources">Resources</TabsTrigger>
                  </TabsList>
                  <Button
                    onClick={handleCreatePermission}
                    className="shrink-0"
                    data-tour-id="create-permission-button"
                  >
                    <Plus className="h-4 w-4 sm:mr-1.5" />
                    <span className="hidden sm:inline">
                      {audienceCopy.ctaLabel}
                    </span>
                    <span className="sm:hidden">Create</span>
                  </Button>
                </div>
              }
            />
          </div>

          <TabsContent value="permissions" className="space-y-4">
            {/* Info Banner */}
            <PageInfoBanner
              title="Understanding Permissions"
              description="Permissions define what actions users can perform on specific resources. They combine roles, scopes, and resources to create fine-grained access control policies."
              features={[
                {
                  text: "Link roles with specific scopes and resources",
                  icon: Shield,
                },
                { text: "Create granular access control policies", icon: Lock },
                {
                  text: "Manage user capabilities across your system",
                  icon: Users,
                },
              ]}
              featuresTitle="Key Capabilities"
              faqs={[
                {
                  id: "1",
                  question: "What are permissions?",
                  answer:
                    "Permissions are the combination of a role, scope, and resource that define what actions a user can perform. For example, 'Admin role with write scope on documents resource' grants document editing capabilities.",
                },
                {
                  id: "2",
                  question: "How do permissions work with roles?",
                  answer:
                    "Roles define a set of capabilities (like 'Admin' or 'Editor'), while permissions specify exactly what those roles can access. A user assigned a role inherits all permissions associated with that role.",
                },
                {
                  id: "3",
                  question: "When should I create a new permission?",
                  answer:
                    "Create a new permission when you need to grant specific access to a resource for a particular role. For example, if you want 'Editors' to have 'read' access to 'reports', you'd create that permission mapping.",
                },
              ]}
              faqsTitle="Common Questions"
              storageKey="permissions-page-banner"
              dismissible={true}
            />

            {/* Filter/Search Card */}
            <div data-tour-id="permissions-filters">
              <PermissionsFilterCard
                onFiltersChange={setFilters}
                initialFilters={filters}
                permissionsData={permissions}
              />
            </div>

            {/* Table */}
            <div
              className="permissions-table-container"
              data-tour-id="permissions-table"
            >
              <style>{`
                .permissions-table-container [data-slot="table-container"] {
                  border: none !important;
                  background: transparent !important;
                }
                .permissions-table-container [data-slot="table-header"] {
                  background: transparent !important;
                }
                .permissions-table-container .bg-muted\\/50,
                .permissions-table-container .bg-muted\\/30,
                .permissions-table-container [class*="bg-muted"] {
                  background: transparent !important;
                }
                .permissions-table-container .shadow-xl {
                  box-shadow: none !important;
                }
              `}</style>
              <TableCard className="transition-all duration-500">
                <CardContent variant="flush">
                  {errorMessage ? (
                    <div className="flex flex-col items-center justify-center p-12 space-y-4">
                      <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                        <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                      </div>
                      <div className="text-center space-y-1">
                        <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                          Unable to Load Permissions
                        </h3>
                        <p className="text-red-700 dark:text-red-300">
                          {errorMessage}
                        </p>
                      </div>
                    </div>
                  ) : showInitialSkeleton ? (
                    <DataTableSkeleton rows={8} />
                  ) : (
                    <div className="relative">
                      <EnhancedPermissionsTable
                        permissions={filteredPermissions}
                        selectedPermissions={selectedPermissions}
                        onSelectAll={handleSelectAll}
                        onSelectPermission={handleSelectPermission}
                      />
                      {permissionsLoading && filteredPermissions.length > 0 && (
                        <div className="absolute inset-0 bg-white/50 dark:bg-neutral-900/50 backdrop-blur-sm flex items-center justify-center">
                          <div className="flex items-center space-x-2">
                            <RefreshCw className="h-4 w-4 animate-spin" />
                            <span className="text-sm font-medium">
                              Refreshing...
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </CardContent>
              </TableCard>
            </div>
          </TabsContent>

          <TabsContent value="resources" className="space-y-4">
            {/* Info Banner */}
            <PageInfoBanner
              title="Understanding Resources"
              description="Resources represent the entities or objects in your system that permissions can be granted on. They define what can be accessed or modified."
              features={[
                {
                  text: "Define entities that users can access",
                  icon: Shield,
                },
                { text: "Create hierarchical access structures", icon: Lock },
                {
                  text: "Map resources to real system objects",
                  icon: Users,
                },
              ]}
              featuresTitle="Key Capabilities"
              faqs={[
                {
                  id: "1",
                  question: "What are resources?",
                  answer:
                    "Resources are the objects or entities in your system that can be protected by permissions. Examples include 'documents', 'users', 'reports', or any other entity you want to control access to.",
                },
                {
                  id: "2",
                  question: "How do resources relate to permissions?",
                  answer:
                    "Resources are one part of a permission. A permission combines a role, a scope (action), and a resource to define access. For example, 'Editor role can write to documents resource'.",
                },
                {
                  id: "3",
                  question: "When should I create a new resource?",
                  answer:
                    "Create a new resource when you have a new type of entity in your system that needs access control. For instance, if you're adding a 'projects' feature, you'd create a 'projects' resource to manage who can access it.",
                },
              ]}
              faqsTitle="Common Questions"
              storageKey="resources-page-banner"
              dismissible={true}
            />

            {/* Filter/Search Card */}
            <FilterShell>
              <CardContent variant="compact">
                <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
                  <div className="flex items-center gap-3">
                    <div className="h-8 w-1 rounded-full bg-[color-mix(in_oklab,var(--color-primary)_35%,transparent)]" />
                    <h3 className="whitespace-nowrap text-[length:var(--font-size-body-md)] font-[var(--font-weight-semibold)] text-[color:var(--color-text-primary)]">
                      Filters
                    </h3>
                  </div>
                  <div className="flex w-full flex-1 flex-col gap-3 sm:flex-row sm:items-center">
                    <div className="relative w-full">
                      <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground" />
                      <Input
                        placeholder="Search resources..."
                        value={resourceSearchQuery}
                        onChange={(e) => setResourceSearchQuery(e.target.value)}
                        className="pl-10 h-[44px]"
                      />
                    </div>
                    {resourceSearchQuery.trim() && (
                      <Button
                        variant="ghost"
                        size="sm"
                        className="sm:w-auto w-full"
                        onClick={() => setResourceSearchQuery("")}
                      >
                        Clear
                      </Button>
                    )}
                  </div>
                </div>
              </CardContent>
            </FilterShell>

            {/* Table */}
            <div className="resources-table-container">
              <style>{`
                .resources-table-container [data-slot="table-container"] {
                  border: none !important;
                  background: transparent !important;
                }
                .resources-table-container [data-slot="table-header"] {
                  background: transparent !important;
                }
                .resources-table-container .bg-muted\\/50,
                .resources-table-container .bg-muted\\/30,
                .resources-table-container [class*="bg-muted"] {
                  background: transparent !important;
                }
                .resources-table-container .shadow-xl {
                  box-shadow: none !important;
                }
              `}</style>
              <TableCard className="transition-all duration-500">
                <CardContent variant="flush">
                  {resourcesErrorMessage ? (
                    <div className="flex flex-col items-center justify-center p-12 space-y-4">
                      <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                        <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                      </div>
                      <div className="text-center space-y-1">
                        <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                          Unable to Load Resources
                        </h3>
                        <p className="text-red-700 dark:text-red-300">
                          {resourcesErrorMessage}
                        </p>
                      </div>
                    </div>
                  ) : showResourcesSkeleton ? (
                    <DataTableSkeleton rows={8} />
                  ) : (
                    <div className="relative">
                      <ResponsiveTableProvider tableType="permissionResources">
                        <ResponsiveDataTable {...resourcesTableConfig} />
                      </ResponsiveTableProvider>
                      {resourcesLoading && filteredResources.length > 0 && (
                        <div className="absolute inset-0 bg-white/50 dark:bg-neutral-900/50 backdrop-blur-sm flex items-center justify-center">
                          <div className="flex items-center space-x-2">
                            <RefreshCw className="h-4 w-4 animate-spin" />
                            <span className="text-sm font-medium">
                              Refreshing...
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </CardContent>
              </TableCard>
            </div>
          </TabsContent>
        </Tabs>
      </div>

      {/* Bulk Actions Bar */}
      {activeTab === "permissions" && selectedPermissions.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedPermissions.length}
          onDeleteSelected={handleBulkDelete}
          onClearSelection={handleClearSelection}
        />
      )}

      {/* Create Permission Modal */}
      <CreatePermissionModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onSuccess={handlePermissionModalSuccess}
      />

      {/* SDK Quick Help */}
      <SDKQuickHelp helpItems={PERMISSIONS_SDK_HELP} title="Permissions SDK" />
    </div>
  );
}
