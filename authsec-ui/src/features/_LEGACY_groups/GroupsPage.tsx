import React, { useState, useMemo, useRef, useCallback } from "react";
import { CardContent } from "@/components/ui/card";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import { toast } from "@/lib/toast";
import { Users2, Plus, RefreshCw, AlertTriangle } from "lucide-react";
import { GroupsTable } from "./components/GroupsTable";
import { BulkActionsBar } from "./components/BulkActionsBar";
import GroupsFilterCard, {
  type GroupsQueryParams,
} from "./components/GroupsFilterCard";
// Unified groups query
// Admin APIs (for mutations)
import {
  useDeleteGroupsMutation as useAdminDeleteGroupsMutation,
  useGetGroupsByTenantQuery as useAdminGetAllGroupsQuery,
} from "@/app/api/admin/groupsApi";
// End-user APIs (for mutations)
import { useRemoveUserFromGroupsMutation as useEndUserRemoveFromGroupsMutation } from "@/app/api/enduser/groupsApi";
import { useGetMyGroupsQuery } from "@/app/api/enduser/groupsApi";

import { useCrossPageNavigation } from "@/lib/cross-page-navigation";
import { SessionManager } from "../../utils/sessionManager";
import { useResponsiveCards } from "../../hooks/use-mobile";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import { resolveTenantId } from "@/utils/workspace";

/**
 * Groups page component - Manage user groups and role assignments with modern UI
 *
 * Features:
 * - Modern design based on Users page
 * - Real-time data from AuthSec API
 * - Advanced filtering with dynamic options
 * - Beautiful metrics and statistics
 * - Enhanced table with expanded row details
 */
export function GroupsPage() {
  const navigate = useContextualNavigate();
  const { navigateWithContext } = useCrossPageNavigation();
  const { isAdmin, audience } = useRbacAudience();
  const [selectedGroups, setSelectedGroups] = useState<string[]>([]);
  const [filters, setFilters] = useState<Partial<GroupsQueryParams>>({});

  // Get session data
  const sessionData = SessionManager.getSession();
  const tenantId = resolveTenantId();
  const userId = sessionData?.user_id;

  // Conditionally use Admin or End-User APIs based on isAdmin toggle
  const [adminDeleteGroups] = useAdminDeleteGroupsMutation();
  const [endUserRemoveFromGroups] = useEndUserRemoveFromGroupsMutation();

  const {
    data: adminGroups = [],
    isLoading: adminGroupsLoading,
    error: adminGroupsError,
    refetch: refetchAdminGroups,
  } = useAdminGetAllGroupsQuery(tenantId || "", {
    skip: !tenantId || !isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: endUserGroups = [],
    isLoading: endUserGroupsLoading,
    error: endUserGroupsError,
    refetch: refetchEndUserGroups,
  } = useGetMyGroupsQuery(undefined, {
    skip: isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Admin Groups",
            subtitle:
              "Organize internal teams into reusable assignment bundles",
            ctaLabel: "Create Group",
          }
        : {
            title: "End-user Groups",
            subtitle:
              "Segment customers into audiences for tailored access policies",
            ctaLabel: "Create Group",
          },
    [isAdmin]
  );

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  const previousAudienceRef = useRef(audience);

  React.useEffect(() => {
    if (previousAudienceRef.current !== audience) {
      if (isAdmin) {
        if (tenantId) {
          refetchAdminGroups();
        }
      } else {
        refetchEndUserGroups();
      }
    }
    previousAudienceRef.current = audience;
  }, [audience, isAdmin, refetchAdminGroups, refetchEndUserGroups, tenantId]);

  const groupsData = useMemo(() => {
    return isAdmin ? adminGroups : endUserGroups;
  }, [isAdmin, adminGroups, endUserGroups]);

  const groupsLoading = isAdmin ? adminGroupsLoading : endUserGroupsLoading;
  const groupsError = isAdmin ? adminGroupsError : endUserGroupsError;
  const refetchGroups = isAdmin ? refetchAdminGroups : refetchEndUserGroups;

  const errorMessage = groupsError
    ? (groupsError as any)?.data?.message || "Failed to fetch group data"
    : null;

  // Process groups data from API response
  const processedGroups = useMemo(() => {
    if (!groupsData) return [];
    const list = Array.isArray(groupsData) ? groupsData : [];
    return list;
  }, [groupsData]);

  // Apply client-side filtering based on filter state
  const filteredGroups = useMemo(() => {
    let result = processedGroups;

    // Apply search filter
    if (filters.searchQuery) {
      const query = filters.searchQuery.toLowerCase();
      result = result.filter(
        (group) =>
          group.name.toLowerCase().includes(query) ||
          (group.description && group.description.toLowerCase().includes(query))
      );
    }

    return result;
  }, [processedGroups, filters]);

  const showInitialSkeleton =
    groupsLoading && filteredGroups.length === 0 && !errorMessage;

  // Calculate statistics

  // Memoize onFiltersChange to prevent infinite re-renders
  const handleFiltersChange = useCallback(
    (newFilters: Partial<GroupsQueryParams>) => {
      setFilters((prev) => {
        const prevJson = JSON.stringify(prev ?? {});
        const nextJson = JSON.stringify(newFilters ?? {});
        if (prevJson === nextJson) return prev; // No change; avoid update loop
        return newFilters;
      });
    },
    []
  );

  // Selection handlers
  const handleSelectionChange = (selectedIds: string[]) => {
    setSelectedGroups(selectedIds);
  };

  const handleSelectAll = () => {
    setSelectedGroups((prev) =>
      prev.length === filteredGroups.length
        ? []
        : filteredGroups.map((group) => group.id)
    );
  };

  const handleClearSelection = () => setSelectedGroups([]);

  const handleCreateGroup = () => navigate("/groups/create");

  const handleEditGroup = (groupId: string) =>
    navigate(`/groups/edit/${groupId}`);

  const handleDeleteGroup = async (groupId: string) => {
    if (!tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      return;
    }

    try {
      if (isAdmin) {
        // Admin: Delete group
        await adminDeleteGroups({
          tenant_id: tenantId,
          group_ids: [groupId],
        }).unwrap();
        toast.success("Group deleted successfully");
      } else {
        // End-user: Remove self from group
        if (!userId) {
          toast.error("User context missing");
          return;
        }
        await endUserRemoveFromGroups({
          tenant_id: tenantId,
          user_id: userId,
          groups: [groupId],
        }).unwrap();
        toast.success("Left group successfully");
      }
      setSelectedGroups((prev) => prev.filter((id) => id !== groupId));
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete group");
    }
  };

  const handleBulkAction = async (action: string) => {
    switch (action) {
      case "delete":
        if (!selectedGroups.length) {
          toast.info("Select at least one group to delete.");
          return;
        }
        if (!tenantId) {
          toast.error("Tenant context missing; please sign in again.");
          return;
        }
        try {
          if (isAdmin) {
            // Admin: Delete groups
            await adminDeleteGroups({
              tenant_id: tenantId,
              group_ids: selectedGroups,
            }).unwrap();
            toast.success(`Deleted ${selectedGroups.length} groups`);
          } else {
            // End-user: Remove self from groups
            if (!userId) {
              toast.error("User context missing");
              return;
            }
            await endUserRemoveFromGroups({
              tenant_id: tenantId,
              user_id: userId,
              group_ids: selectedGroups,
            }).unwrap();
            toast.success(`Removed from ${selectedGroups.length} groups`);
          }
          setSelectedGroups([]);
        } catch (error: any) {
          toast.error(error?.data?.message || "Failed to delete groups");
        }
        break;
      default:
        toast.info(`${action} action for ${selectedGroups.length} groups`);
    }
  };

  return (
    <div
      className="min-h-screen "
      ref={mainAreaRef}
    >
      <div className="p-6 max-w-10xl mx-auto">
        <div className="space-y-4">
          <PageHeader
            title={audienceCopy.title}
            description={audienceCopy.subtitle}
            actions={
              <Button onClick={handleCreateGroup}>
                <Plus className="h-4 w-4 mr-2" />
                {audienceCopy.ctaLabel}
              </Button>
            }
          />

          <GroupsFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
            groupsData={processedGroups}
          />

          <TableCard>
            <CardContent variant="flush">
              {errorMessage ? (
                <div className="flex flex-col items-center justify-center p-12 space-y-4">
                  <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                    <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                  </div>
                  <div className="text-center space-y-1">
                    <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                      Unable to Load Groups
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
                  <style>{`
                        .adaptive-table-container {
                          border-radius: 0 !important;
                        }
                        .adaptive-table-container > div:first-child {
                          border-radius: 0 !important;
                        }
                      `}</style>
                  <GroupsTable
                    groups={filteredGroups}
                    selectedGroupIds={selectedGroups}
                    onSelectionChange={handleSelectionChange}
                    onSelectAll={handleSelectAll}
                    actions={{
                      onEdit: handleEditGroup,
                      onDelete: handleDeleteGroup,
                    }}
                  />
                  {groupsLoading && filteredGroups.length > 0 && (
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

      {selectedGroups.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedGroups.length}
          onBulkAction={handleBulkAction}
          onClearSelection={handleClearSelection}
        />
      )}
    </div>
  );
}
