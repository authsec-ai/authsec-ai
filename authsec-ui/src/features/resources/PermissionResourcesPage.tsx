import React, { useState, useMemo, useCallback } from "react";
import { CardContent } from "@/components/ui/card";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { RefreshCw, AlertTriangle } from "lucide-react";
import {
  useGetAdminPermissionResourcesQuery,
  useGetEndUserPermissionResourcesQuery,
} from "@/app/api/permissionsResourcesApi";
import { resolveTenantId } from "@/utils/workspace";
import { useResponsiveCards } from "@/hooks/use-mobile";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { Search } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
  type ResponsiveColumnDef,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";

interface SimpleResource {
  id: string;
  name: string;
}

/**
 * Permission Resources page component - View unique resources from permissions
 *
 * Features:
 * - Read-only listing of resources
 * - Uses permission resources API endpoints
 * - Supports both admin and enduser contexts
 * - No create/edit/delete functionality
 */
export function PermissionResourcesPage() {
  const { isAdmin } = useRbacAudience();
  const [searchQuery, setSearchQuery] = useState("");

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Resources",
            subtitle:
              "View unique resources defined in your platform permissions",
          }
        : {
            title: "Resources",
            subtitle: "View resources available across your applications",
          },
    [isAdmin]
  );

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  // API data fetching
  const tenantId = resolveTenantId();

  // Conditionally use admin or end-user APIs based on audience
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

  // Select the correct data based on audience
  const resources: SimpleResource[] = useMemo(() => {
    const rawResources = isAdmin ? adminResources : endUserResources;
    return (rawResources as string[]).map((name) => ({
      id: name,
      name,
    }));
  }, [adminResources, endUserResources, isAdmin]);

  const resourcesLoading = isAdmin
    ? adminResourcesLoading
    : endUserResourcesLoading;
  const resourcesError = isAdmin ? adminResourcesError : endUserResourcesError;

  // Extract error message
  const errorMessage = resourcesError
    ? (resourcesError as any)?.data?.message || "Failed to fetch resources data"
    : null;

  // Apply client-side filtering based on search
  const filteredResources = useMemo(() => {
    let result = [...resources];

    if (searchQuery) {
      const query = searchQuery.toLowerCase();
      result = result.filter((resource) =>
        resource.name.toLowerCase().includes(query)
      );
    }

    return result;
  }, [resources, searchQuery]);

  const showInitialSkeleton =
    resourcesLoading && filteredResources.length === 0 && !errorMessage;

  // Table columns - read-only, just showing resource name
  const columns: ResponsiveColumnDef<SimpleResource, unknown>[] = useMemo(
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
    []
  );

  const tableConfig: ResponsiveTableConfig<SimpleResource> = {
    data: filteredResources,
    columns,
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

  return (
    <div className="min-h-screen" ref={mainAreaRef}>
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header - no actions button since this is read-only */}
        <PageHeader
          title={audienceCopy.title}
          description={audienceCopy.subtitle}
        />

        {/* Filter/Search Card */}
        <FilterShell>
          <CardContent variant="compact">
            <div className="flex flex-col gap-4 lg:flex-row lg:items-center lg:justify-between">
              <div className="flex items-center gap-3">
                <div className="h-8 w-1 rounded-full bg-[color-mix(in_oklab,var(--color-primary)_35%,transparent)]" />
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold text-foreground">
                    Filters
                  </span>
                  {searchQuery.trim() && (
                    <Badge variant="secondary" className="text-xs font-medium">
                      1
                    </Badge>
                  )}
                </div>
              </div>
              <div className="flex w-full flex-1 flex-col gap-3 sm:flex-row sm:items-center">
                <div className="relative w-full">
                  <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground" />
                  <Input
                    placeholder="Search resources..."
                    value={searchQuery}
                    onChange={(e) => setSearchQuery(e.target.value)}
                    className="pl-10 h-[44px]"
                  />
                </div>
                {searchQuery.trim() && (
                  <Button
                    variant="ghost"
                    size="sm"
                    className="sm:w-auto w-full"
                    onClick={() => setSearchQuery("")}
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
              {errorMessage ? (
                <div className="flex flex-col items-center justify-center p-12 space-y-4">
                  <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                    <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                  </div>
                  <div className="text-center space-y-1">
                    <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                      Unable to Load Resources
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
                  <ResponsiveTableProvider tableType="permissionResources">
                    <ResponsiveDataTable {...tableConfig} />
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
      </div>
    </div>
  );
}
