import React, { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { CardContent } from "@/components/ui/card";
import { Boxes, Plus, AlertTriangle, RefreshCw } from "lucide-react";
import EnhancedResourcesTable from "./components/EnhancedResourcesTable";
import ResourcesFilterCard, { type ResourcesQueryParams } from "./components/ResourcesFilterCard";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
// Admin APIs
import {
  useGetAdminResourcesQuery,
  useDeleteAdminResourceMutation,
} from "@/app/api/admin/resourcesApi";
// End-user APIs
import {
  useGetEndUserResourcesQuery,
  useDeleteEndUserResourceMutation,
} from "@/app/api/enduser/resourcesApi";
import { toast } from "@/lib/toast";
import { BulkActionsBar } from "./components";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import type { Resource } from "./types";
import { resolveTenantId } from "@/utils/workspace";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import { performDiscreteDeletes } from "@/utils/bulk-actions";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { SDKQuickHelp, RESOURCES_SDK_HELP } from "@/features/sdk";

export default function ResourcesPage() {
  const navigate = useContextualNavigate();
  const { isAdmin } = useRbacAudience();

  // Get tenant ID from context/session
  const tenantId = resolveTenantId();

  // Conditionally use Admin or End-User APIs based on audience toggle
  const [adminDeleteResource] = useDeleteAdminResourceMutation();
  const [endUserDeleteResource] = useDeleteEndUserResourceMutation();

  // API data fetching - audience context triggers refetch on toggle
  const {
    data: adminResources = [],
    isLoading: adminResourcesLoading,
    error: adminResourcesError,
    refetch: refetchAdminResources,
  } = useGetAdminResourcesQuery(undefined, {
    skip: !isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: endUserResources = [],
    isLoading: endUserResourcesLoading,
    error: endUserResourcesError,
    refetch: refetchEndUserResources,
  } = useGetEndUserResourcesQuery(tenantId || '', {
    skip: isAdmin || !tenantId,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  // Select correct data based on audience
  const rawResources = isAdmin ? adminResources : endUserResources;
  const resourcesLoading = isAdmin ? adminResourcesLoading : endUserResourcesLoading;
  const resourcesError = isAdmin ? adminResourcesError : endUserResourcesError;

  const previousContextRef = useRef<{ isAdmin: boolean; tenantId?: string | null }>({
    isAdmin,
    tenantId,
  });

  useEffect(() => {
    const previous = previousContextRef.current;
    const audienceChanged = previous.isAdmin !== isAdmin;
    const tenantChanged = previous.tenantId !== tenantId;

    if (audienceChanged || tenantChanged) {
      if (isAdmin) {
        refetchAdminResources();
      } else if (tenantId) {
        refetchEndUserResources();
      }
    }

    previousContextRef.current = { isAdmin, tenantId };
  }, [isAdmin, tenantId, refetchAdminResources, refetchEndUserResources]);

  // State
  const [selectedResources, setSelectedResources] = useState<string[]>([]);
  const [filters, setFilters] = useState<Partial<ResourcesQueryParams>>({});
  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Admin Resources",
            subtitle:
              "Catalog the internal services and tools protected for staff",
            ctaLabel: "Create Resource",
          }
        : {
            title: "End-user Resources",
            subtitle:
              "Track customer-facing APIs and applications that need scoped access",
            ctaLabel: "Create Resource",
          },
    [isAdmin]
  );

  // Extract error message
  const errorMessage = resourcesError
    ? (resourcesError as any)?.data?.message || "Failed to fetch resource data"
    : null;

  // Use resources directly from API (already transformed by transformResponse)
  const processedResources = rawResources || [];

  // Apply client-side filtering based on filter state
  const filteredResources = useMemo<Resource[]>(() => {
    let result = processedResources;

    // Apply search filter
    if (filters.searchQuery) {
      const query = filters.searchQuery.toLowerCase();
      result = result.filter((resource) =>
        resource.name.toLowerCase().includes(query)
      );
    }

    return result;
  }, [processedResources, filters]);

  const showInitialSkeleton = resourcesLoading && filteredResources.length === 0 && !errorMessage;

  // Calculate statistics

  const handleSelectionChange = useCallback((resourceIds: string[]) => {
    setSelectedResources(resourceIds);
  }, []);

  const handleClearSelection = () => setSelectedResources([]);

  const handleDeleteResource = async (resourceId: string) => {
    if (!isAdmin && !tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      return;
    }

    try {
      if (isAdmin) {
        await adminDeleteResource(resourceId).unwrap();
      } else {
        await endUserDeleteResource({ tenant_id: tenantId!, resource_id: resourceId }).unwrap();
      }
      toast.success("Resource deleted successfully");
      setSelectedResources((prev) => prev.filter((id) => id !== resourceId));
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete resource");
    }
  };

  const handleCreateResource = () => navigate("/resources/create");

  const handleEditResource = (resourceId: string) => {
    navigate(`/resources/edit/${resourceId}`);
  };

  const handleBulkAction = async (action: string) => {
    switch (action) {
      case "delete":
        if (!selectedResources.length) {
          toast.info("Select at least one resource to delete.");
          return;
        }
        if (!isAdmin && !tenantId) {
          toast.error("Tenant context missing; please sign in again.");
          return;
        }
        try {
          const deleteResult = await performDiscreteDeletes(selectedResources, (resourceId) =>
            isAdmin
              ? adminDeleteResource(resourceId).unwrap()
              : endUserDeleteResource({ tenant_id: tenantId!, resource_id: resourceId }).unwrap()
          );

          if (deleteResult.successCount) {
            toast.success(`Deleted ${deleteResult.successCount} resources`);
          }
          if (deleteResult.failureCount) {
            toast.error(`Failed to delete ${deleteResult.failureCount} resources`);
            const failedIds = new Set(deleteResult.failures.map((failure) => failure.id));
            setSelectedResources(selectedResources.filter((id) => failedIds.has(id)));
          } else {
            setSelectedResources([]);
          }
        } catch (error: any) {
          toast.error(error?.data?.message || "Failed to delete resources");
        }
        break;
      default:
        toast.info(`${action} action for ${selectedResources.length} resources`);
    }
  };

  const handleFiltersChange = (newFilters: Partial<ResourcesQueryParams>) => {
    setFilters(newFilters);
  };

  return (
    <div className="min-h-screen">
      <div className="mx-auto max-w-10xl p-6">
        <div className="space-y-4">
          <PageHeader
            title={audienceCopy.title}
            description={audienceCopy.subtitle}
            actions={
              <Button onClick={handleCreateResource}>
                <Plus className="mr-2 h-4 w-4" />
                {audienceCopy.ctaLabel}
              </Button>
            }
          />

          <ResourcesFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
            resourcesData={processedResources}
          />

          <div className="resources-table-container">
            <TableCard className="transition-all duration-500">
              <CardContent variant="flush" className="space-y-4">
                {errorMessage ? (
                  <div className="flex flex-col items-center justify-center p-12 space-y-4">
                    <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                      <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                    </div>
                    <div className="text-center space-y-1">
                      <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                        Unable to Load Resources
                      </h3>
                      <p className="text-red-700 dark:text-red-300">{errorMessage}</p>
                    </div>
                  </div>
                ) : showInitialSkeleton ? (
                  <DataTableSkeleton rows={8} />
                ) : (
                  <div className="relative">
                    <EnhancedResourcesTable
                      data={filteredResources}
                      selectedResources={selectedResources}
                      onSelectionChange={handleSelectionChange}
                      onEditResource={handleEditResource}
                      onDeleteResource={handleDeleteResource}
                    />
                    {resourcesLoading && filteredResources.length > 0 && (
                      <div className="absolute inset-0 flex items-center justify-center bg-white/50 backdrop-blur-sm dark:bg-neutral-900/50">
                        <div className="flex items-center space-x-2">
                          <RefreshCw className="h-4 w-4 animate-spin" />
                          <span className="text-sm font-medium">Refreshing...</span>
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </CardContent>
            </TableCard>
          </div>
        </div>
      </div>

      {selectedResources.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedResources.length}
          onBulkAction={handleBulkAction}
          onClearSelection={handleClearSelection}
        />
      )}

      {/* SDK Quick Help */}
      <SDKQuickHelp
        helpItems={RESOURCES_SDK_HELP}
        title="Resources SDK"
      />
    </div>
  );
}
