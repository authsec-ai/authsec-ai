import React, { useState, useMemo, useRef, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { CardContent } from "@/components/ui/card";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { ShieldHalf, Plus, AlertTriangle, RefreshCw } from "lucide-react";
import { EnhancedRolesTable, BulkActionsBar } from "./components";
import { MapRoleToScopeModal } from "@/features/mappings/components/MapRoleToScopeModal";
import RolesFilterCard, {
  type RolesQueryParams,
} from "./components/RolesFilterCard";
import { CreateRoleModal } from "./components/CreateRoleModal";
import {
  useGetAuthSecRolesQuery,
  useDeleteUserDefinedRolesMutation,
} from "@/app/api/rolesApi";
import type { EnhancedRole } from "@/types/entities";
import { SessionManager } from "../../utils/sessionManager";
import { toast } from "@/lib/toast";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { SDKQuickHelp, ROLES_SDK_HELP } from "@/features/sdk";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { Shield, Users, Layers } from "lucide-react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

const RECENT_ROLE_STORAGE_KEY = "authsec_recent_role_id";
export function RolesPage() {
  const contextualNavigate = useContextualNavigate();
  const standardNavigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();
  const mainAreaRef = useRef<HTMLDivElement>(null);
  const { isAdmin, audience } = useRbacAudience();

  // Get tenant ID from session
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  // API data fetching - audience in query params triggers refetch on context change
  const {
    data: roles = [],
    isLoading: rolesLoading,
    error: rolesError,
  } = useGetAuthSecRolesQuery(
    { tenant_id: tenantId || "", audience },
    {
      skip: !tenantId,
    }
  );

  // Extract error message
  const errorMessage = rolesError
    ? (rolesError as any)?.data?.message || "Failed to fetch roles data"
    : null;

  // State
  const [selectedRoles, setSelectedRoles] = useState<string[]>([]);
  const [deleteRoles, { isLoading: deletingRoles }] =
    useDeleteUserDefinedRolesMutation();
  const [filters, setFilters] = useState<Partial<RolesQueryParams>>({});

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["roles-management"],
  });
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [assignUsersModalOpen, setAssignUsersModalOpen] = useState(false);
  const [recentRoleId, setRecentRoleId] = useState<string | null>(() =>
    typeof window !== "undefined"
      ? sessionStorage.getItem(RECENT_ROLE_STORAGE_KEY)
      : null
  );
  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Roles and mapped permissions",
            subtitle:
              "Define reusable permission sets for privileged operators",
            ctaLabel: "Create Roles and map permissions",
          }
        : {
            title: "Roles and mapped permissions",
            subtitle:
              "Shape customer capabilities with tailored permission bundles",
            ctaLabel: "Create Roles and map permissions",
          },
    [isAdmin]
  );

  // Auto-open modal if query param present (from wizard)
  useEffect(() => {
    if (searchParams.get("openModal") === "create") {
      setCreateModalOpen(true);
      // Clean up query param while preserving location state
      searchParams.delete("openModal");
      setSearchParams(searchParams, { replace: true, state: location.state });
    }
  }, [searchParams, setSearchParams, location.state]);

  // Handle modal close with wizard awareness
  const handleRoleModalSuccess = () => {
    // Don't close modal here - it will be closed by onOpenChange
    // If coming from wizard, navigate back to root with success flag
    if (location.state?.fromWizard) {
      standardNavigate("/", { state: { roleCreated: true } });
    }
  };

  const normalizedRoles = useMemo<EnhancedRole[]>(() => {
    if (!Array.isArray(roles)) return [];

    return roles
      .map((role: any, index: number) => {
        const rawId =
          role?.id ??
          role?.role_id ??
          role?.roleId ??
          role?.uuid ??
          role?.uid ??
          role?.name ??
          `role-${index}`;

        if (!rawId) return null;

        const permissionCountRaw =
          role?.permissionCount ??
          role?.permissions_count ??
          (Array.isArray(role?.permissions)
            ? role.permissions.length
            : undefined);
        const permissionCount = Number.isFinite(Number(permissionCountRaw))
          ? Number(permissionCountRaw)
          : undefined;

        const userIds = Array.isArray(role?.user_ids)
          ? role.user_ids.map(String)
          : Array.isArray(role?.userIds)
          ? role.userIds.map(String)
          : undefined;

        const usernames = Array.isArray(role?.usernames)
          ? role.usernames.map(String)
          : undefined;

        const userCountRaw =
          role?.users_assigned ??
          role?.userCount ??
          (usernames ? usernames.length : undefined) ??
          (userIds ? userIds.length : undefined);
        const userCount = Number.isFinite(Number(userCountRaw))
          ? Number(userCountRaw)
          : undefined;

        const groupIds = Array.isArray(role?.group_ids)
          ? role.group_ids.map(String)
          : Array.isArray(role?.groupIds)
          ? role.groupIds.map(String)
          : undefined;

        const groupCountRaw =
          role?.group_count ??
          role?.groupCount ??
          (groupIds ? groupIds.length : undefined);
        const groupCount = Number.isFinite(Number(groupCountRaw))
          ? Number(groupCountRaw)
          : undefined;

        const typeNormalized =
          role?.type === "system"
            ? "system"
            : role?.type === "custom"
            ? "custom"
            : undefined;

        return {
          id: String(rawId),
          name: String(role?.name ?? role?.role_name ?? `Role ${index + 1}`),
          description: role?.description ?? "",
          type: typeNormalized,
          permissions: Array.isArray(role?.permissions) ? role.permissions : [],
          permissionCount,
          users_assigned: role?.users_assigned,
          usernames,
          tenant_id: role?.tenant_id,
          client_id: role?.client_id ?? role?.project_id,
          project_id: role?.project_id,
          userIds,
          groupIds,
          userCount,
          groupCount,
          isBuiltIn:
            role?.isBuiltIn ??
            role?.is_built_in ??
            (role?.type === "system" ? true : undefined),
          version: role?.version ?? 1,
          createdAt: role?.created_at ?? role?.createdAt ?? "",
          updatedAt: role?.updated_at ?? role?.updatedAt ?? "",
          createdBy: role?.created_by ?? role?.createdBy ?? "",
        };
      })
      .filter((role): role is EnhancedRole => Boolean(role));
  }, [roles]);

  // Filtering logic with simplified filters
  const filteredRoles = useMemo(() => {
    let filtered = [...normalizedRoles];

    // Apply search filter
    if (filters.searchQuery) {
      const query = filters.searchQuery.toLowerCase();
      filtered = filtered.filter(
        (role) =>
          role.name.toLowerCase().includes(query) ||
          (role.description && role.description.toLowerCase().includes(query))
      );
    }

    return filtered;
  }, [normalizedRoles, filters]);

  useEffect(() => {
    if (!recentRoleId) return;

    const exists = normalizedRoles.some((role) => role.id === recentRoleId);
    if (!exists) return;

    sessionStorage.setItem(RECENT_ROLE_STORAGE_KEY, recentRoleId);
    const timer = window.setTimeout(() => {
      setRecentRoleId(null);
      sessionStorage.removeItem(RECENT_ROLE_STORAGE_KEY);
    }, 5000);

    return () => window.clearTimeout(timer);
  }, [recentRoleId, normalizedRoles]);

  useEffect(() => {
    if (!recentRoleId) return;
    const exists = normalizedRoles.some((role) => role.id === recentRoleId);
    if (!exists && normalizedRoles.length > 0) {
      setRecentRoleId(null);
      sessionStorage.removeItem(RECENT_ROLE_STORAGE_KEY);
    }
  }, [recentRoleId, normalizedRoles]);

  const showInitialSkeleton =
    rolesLoading && filteredRoles.length === 0 && !errorMessage;

  // Selection handlers
  const handleSelectRole = (roleId: string) => {
    setSelectedRoles((prev) =>
      prev.includes(roleId)
        ? prev.filter((id) => id !== roleId)
        : [...prev, roleId]
    );
  };

  const handleSelectAll = () => {
    setSelectedRoles((prev) =>
      prev.length === filteredRoles.length
        ? []
        : filteredRoles.map((role) => role.id)
    );
  };

  const handleClearSelection = () => setSelectedRoles([]);

  const handleCreateRole = () => setCreateModalOpen(true);

  const handleDeleteRole = async (roleId: string) => {
    if (!tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      return;
    }

    try {
      await deleteRoles({
        tenant_id: tenantId,
        role_ids: [roleId],
      }).unwrap();
      toast.success("Role deleted successfully");
      setSelectedRoles((prev) => prev.filter((id) => id !== roleId));
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete role");
    }
  };

  const handleBulkAction = async (action: string) => {
    switch (action) {
      case "delete":
        if (!selectedRoles.length) {
          toast.info("Select at least one role to delete.");
          return;
        }
        if (!tenantId) {
          toast.error("Tenant context missing; please sign in again.");
          return;
        }
        try {
          await deleteRoles({
            tenant_id: tenantId,
            audience,
            role_ids: selectedRoles,
          }).unwrap();
          toast.success(`Deleted ${selectedRoles.length} roles`);
          setSelectedRoles([]);
        } catch (error: any) {
          toast.error(error?.data?.message || "Failed to delete roles");
        }
        break;
      case "assign-users":
        if (!selectedRoles.length) {
          toast.info("Select at least one role to assign users.");
          return;
        }
        setAssignUsersModalOpen(true);
        break;
      default:
        toast.info(`${action} action for ${selectedRoles.length} roles`);
    }
  };

  // Get selected role details for the modal
  const selectedRolesForModal = useMemo(() => {
    return filteredRoles
      .filter((role) => selectedRoles.includes(role.id))
      .map((role) => ({ id: role.id, name: role.name }));
  }, [selectedRoles, filteredRoles]);

  // State for single role assign users (from row action)
  const [singleRoleForUsers, setSingleRoleForUsers] = useState<{
    id: string;
    name: string;
  } | null>(null);

  // Handler for single role assign users from row action
  const handleAssignUsersForRole = (roleId: string) => {
    const role = filteredRoles.find((r) => r.id === roleId);
    if (role) {
      setSingleRoleForUsers({ id: role.id, name: role.name });
      setAssignUsersModalOpen(true);
    }
  };

  // Get roles for modal - either selected roles (bulk) or single role (row action)
  const rolesForAssignUsersModal = useMemo(() => {
    if (singleRoleForUsers) {
      return [singleRoleForUsers];
    }
    return selectedRolesForModal;
  }, [singleRoleForUsers, selectedRolesForModal]);

  // Calculate statistics

  return (
    <div className="min-h-screen">
      <div className="p-6 max-w-10xl mx-auto" ref={mainAreaRef}>
        <div className="space-y-4">
          <PageHeader
            title={audienceCopy.title}
            description={audienceCopy.subtitle}
            actions={
              <Button
                onClick={handleCreateRole}
                data-tour-id="create-role-button"
              >
                <Plus className="mr-2 h-4 w-4" />
                {audienceCopy.ctaLabel}
              </Button>
            }
          />

          {/* Info Banner */}
          <PageInfoBanner
            title="Understanding Roles"
            description="Roles are reusable permission sets that define what users can do in your system. Instead of assigning permissions individually, assign users to roles for easier management."
            features={[
              {
                text: "Group related permissions into logical sets",
                icon: Layers,
              },
              { text: "Assign roles to users and groups", icon: Users },
              { text: "Manage access control at scale", icon: Shield },
            ]}
            featuresTitle="Key Benefits"
            faqs={[
              {
                id: "1",
                question:
                  "What's the difference between roles and permissions?",
                answer:
                  "Roles are containers for permissions. A role like 'Editor' might contain multiple permissions such as 'read:documents', 'write:documents', and 'delete:own-documents'. Users are assigned roles, not individual permissions.",
              },
              {
                id: "2",
                question: "Should I use system roles or create custom roles?",
                answer:
                  "System roles (like Admin, User) cover common scenarios and are maintained by the platform. Create custom roles when you need specific permission combinations for your use case, such as 'Content Moderator' or 'Billing Manager'.",
              },
              {
                id: "3",
                question: "How do I assign roles to users?",
                answer:
                  "Use Role Bindings to connect users with roles. You can assign roles individually or in bulk. Role bindings can also be scoped to specific resources or contexts.",
              },
            ]}
            faqsTitle="Common Questions"
            storageKey="roles-page-banner"
            dismissible={true}
          />

          <div data-tour-id="roles-filters">
            <RolesFilterCard
              onFiltersChange={setFilters}
              initialFilters={filters}
              rolesData={roles}
            />
          </div>

          <div className="roles-table-container" data-tour-id="roles-table">
            <style>{`
                  .roles-table-container [data-slot="table-container"] {
                    border: none !important;
                    background: transparent !important;
                  }
                  .roles-table-container [data-slot="table-header"] {
                    background: transparent !important;
                  }
                  .roles-table-container .bg-muted\\/50,
                  .roles-table-container .bg-muted\\/30,
                  .roles-table-container [class*="bg-muted"] {
                    background: transparent !important;
                  }
                  .roles-table-container .shadow-xl {
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
                        Unable to Load Roles
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
                    <EnhancedRolesTable
                      data={filteredRoles}
                      selectedRoles={selectedRoles}
                      onSelectRole={handleSelectRole}
                      onSelectAll={handleSelectAll}
                      highlightRoleId={recentRoleId}
                      onDeleteRole={handleDeleteRole}
                      onAssignUsers={handleAssignUsersForRole}
                    />
                    {rolesLoading && filteredRoles.length > 0 && (
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

      {selectedRoles.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedRoles.length}
          onBulkAction={handleBulkAction}
          onClearSelection={handleClearSelection}
        />
      )}

      {/* Create Role Modal */}
      <CreateRoleModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
        onRoleCreated={(id) => setRecentRoleId(id)}
        onSuccess={handleRoleModalSuccess}
      />

      {/* Assign Users to Role Modal */}
      <MapRoleToScopeModal
        open={assignUsersModalOpen}
        onOpenChange={(open) => {
          setAssignUsersModalOpen(open);
          if (!open) {
            setSingleRoleForUsers(null);
          }
        }}
        preselectedRoles={rolesForAssignUsersModal}
        onSuccess={() => {
          setSelectedRoles([]);
          setSingleRoleForUsers(null);
        }}
      />

      {/* SDK Quick Help */}
      <SDKQuickHelp helpItems={ROLES_SDK_HELP} title="Roles SDK" />
    </div>
  );
}
