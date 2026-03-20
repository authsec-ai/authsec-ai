import React, { useState, useMemo, useCallback, useRef } from "react";
import { Card, CardContent } from "../../components/ui/card";
import { TableCard } from "../../theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { useAuth } from "../../auth/context/AuthContext";
import { useResponsiveCards } from "../../hooks/use-mobile";
// Import both admin and enduser user APIs
import {
  useGetAdminUsersQuery,
  useDeleteAdminUserMutation,
  useSetAdminUserActiveMutation,
  useResetAdminUserPasswordMutation,
  useChangeAdminUserPasswordMutation,
  type AdminUsersQueryParams
} from "@/app/api/admin/usersApi";
import {
  useGetEndUsersQuery,
  useDeleteUserMutation,
  useSetUserActiveMutation,
  useResetUserPasswordMutation,
  useChangeUserPasswordMutation,
  type UsersQueryParams,
} from "@/app/api/enduser/usersApi";
import type { SyncType } from "@/app/api/syncConfigsApi";
import { useListSyncConfigsQuery } from "@/app/api/syncConfigsApi";
import { useCrossPageNavigation } from "@/lib/cross-page-navigation";
import { toast } from "@/lib/toast.ts";
import { BulkActionsBar, UsersTableSkeleton } from "./components/index.ts";
import { MapRoleToScopeModal } from "@/features/mappings/components/MapRoleToScopeModal";
import { AdminUsersTable } from "./components/AdminUsersTable";
import { EndUserUsersTable } from "./components/EndUserUsersTable";
import UsersFilterCard from "./components/UsersFilterCard.tsx";
import AdminUsersFilterCard, { type AdminUsersFilterState } from "./components/AdminUsersFilterCard";
import { UserSourceTabs, type UserSource } from "./components/UserSourceTabs";
import { AddUsersModal } from "./components/AddUsersModal";
import { InviteUserModal } from "./components/InviteUserModal";
import { ADSyncInlineForm } from "./components/ADSyncInlineForm";
import { EntraSyncInlineForm } from "./components/EntraSyncInlineForm";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { resolveTenantId } from "@/utils/workspace";
import type { EnhancedUser } from "@/types/entities";
import {
  UserX,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";
import { useLocation, useNavigate } from "react-router-dom";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";

/**
 * Users page component - Manage users and team assignments with modern UI
 *
 * Features:
 * - User list with comprehensive filtering and search
 * - Modern design based on Clients page
 * - Real-time data from AuthSec API
 * - Advanced filtering with dynamic options
 * - Beautiful metrics and statistics
 * - Enhanced table with expanded row details
 */

const DEFAULT_ADMIN_PROVIDERS = ["local", "ad_sync", "entra_id", "azure_ad", "google", "github", "auth0", "okta"];

export function UsersPage() {
  console.log("[COMPONENT] 👥 UsersPage RENDERING", {
    timestamp: new Date().toISOString(),
    path: window.location.pathname
  });

  const { currentProject } = useAuth();
  const navigate = useNavigate();
  const { currentContext, navigateWithContext } = useCrossPageNavigation();
  const { isAdmin } = useRbacAudience();
  const location = useLocation();
  const isAdminView = useMemo(
    () => isAdmin || /^\/admin(?:\/|$)/.test(location.pathname),
    [isAdmin, location.pathname]
  );

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY['users-management'],
  });

  // Component lifecycle logging
  React.useEffect(() => {
    console.log("[COMPONENT] 👥 UsersPage MOUNTED", {
      timestamp: new Date().toISOString(),
      path: window.location.pathname,
      isAdmin,
      isAdminView
    });

    return () => {
      console.log("[COMPONENT] 👥 UsersPage UNMOUNTED", {
        timestamp: new Date().toISOString()
      });
    };
  }, [isAdmin, isAdminView]);
  const [selectedUsers, setSelectedUsers] = useState<string[]>([]);
  const [contextSwitchCounter, setContextSwitchCounter] = useState(0);
  const previousIsAdminRef = useRef(isAdmin);
  // Initialize filters from URL context immediately so first query includes them
  const [filters, setFilters] = useState<Partial<UsersQueryParams>>(
    () => (currentContext?.filters as Partial<UsersQueryParams>) || {}
  );
  const [adminFilters, setAdminFilters] = useState<AdminUsersFilterState>({});

  // View mode: 'list' | 'create-ad-sync' | 'create-entra-sync' | 'edit-ad-sync' | 'edit-entra-sync'
  const [viewMode, setViewMode] = useState<'list' | 'create-ad-sync' | 'create-entra-sync' | 'edit-ad-sync' | 'edit-entra-sync'>('list');

  // Source tab and modal states
  const [selectedSource, setSelectedSource] = useState<UserSource>('all');
  const [showAddUsersModal, setShowAddUsersModal] = useState(false);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [showAssignRoleModal, setShowAssignRoleModal] = useState(false);
  const [editingSyncConfig, setEditingSyncConfig] = useState<any>(null);
  const [showNotConfiguredDialog, setShowNotConfiguredDialog] = useState(false);
  const [notConfiguredType, setNotConfiguredType] = useState<'ad' | 'entra' | null>(null);

  // Apply filters coming from cross-page navigation exactly when they change
  const lastAppliedFilters = useRef<string>('{}');
  React.useEffect(() => {
    const contextFilters = currentContext?.filters;
    const contextFiltersString = JSON.stringify(contextFilters ?? {});
    if (contextFilters && Object.keys(contextFilters).length > 0 && lastAppliedFilters.current !== contextFiltersString) {
      lastAppliedFilters.current = contextFiltersString;
      setFilters(contextFilters as Partial<UsersQueryParams>);
    }
  }, [currentContext?.filters]);

  // Check for navigation state to auto-open directory sync form
  React.useEffect(() => {
    const state = location.state as any;
    if (state?.openDirectorySync) {
      const provider = state.provider || 'ad';
      setViewMode(provider === 'entra' ? 'create-entra-sync' : 'create-ad-sync');
      // Clear the state
      window.history.replaceState({}, document.title);
    }
  }, [location.state]);

  React.useEffect(() => {
    if (isAdmin) {
      setFilters({});
      lastAppliedFilters.current = '{}';
    }
  }, [isAdmin]);
  React.useEffect(() => {
    if (!isAdmin) {
      setAdminFilters({});
    }
  }, [isAdmin]);
  const [currentPage, setCurrentPage] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const projectName = currentProject?.name || "your project";
  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Admin Users",
            descriptionPrefix:
              "Manage internal operators",
            descriptionSuffix: "",
            ctaLabel: "Invite Admin",
            totalLabel: "Total Admins",
          }
        : {
            title: "End Users",
            descriptionPrefix:
              "Manage customer identities",
            descriptionSuffix: "",
            ctaLabel: "Invite End User",
            totalLabel: "Total End Users",
          },
    [isAdmin]
  );

  // Clear selected users when page changes
  React.useEffect(() => {
    setSelectedUsers([]);
  }, [currentPage, pageSize]);

  // Memoize onFiltersChange to prevent infinite re-renders
  const handleFiltersChange = useCallback((newFilters: Partial<UsersQueryParams>) => {
    if (isAdminView) return;
    setFilters((prev) => {
      const prevJson = JSON.stringify(prev ?? {});
      const nextJson = JSON.stringify(newFilters ?? {});
      if (prevJson === nextJson) return prev; // No change; avoid update loop
      return newFilters;
    });
    setCurrentPage(1); // Reset to first page when filters change
  }, [isAdminView]); // Empty dependencies since we want this to be stable

  const handleAdminFiltersChange = useCallback((nextFilters: AdminUsersFilterState) => {
    const prevJson = JSON.stringify(adminFilters ?? {});
    const nextJson = JSON.stringify(nextFilters ?? {});
    if (prevJson === nextJson) return;
    setAdminFilters(nextFilters);
    setCurrentPage(1);
  }, [adminFilters]);

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  // Track context switches to force fresh queries on audience toggle
  React.useEffect(() => {
    if (previousIsAdminRef.current !== isAdmin) {
      console.log(
        `[UsersPage] Audience switch: ${previousIsAdminRef.current ? "admin" : "enduser"} → ${
          isAdmin ? "admin" : "enduser"
        }`
      );
      previousIsAdminRef.current = isAdmin;
      setContextSwitchCounter((prev) => prev + 1);
    }
  }, [isAdmin]);

  const contextKey = isAdmin ? "admin" : "enduser";
  const contextSignature = `${contextKey}-${contextSwitchCounter}`;
  const tenantId = resolveTenantId();

  const adminQueryArgs = useMemo(() => {
    const args: AdminUsersQueryParams = {
      page: currentPage,
      limit: pageSize,
    };
    if (tenantId) {
      args.tenant_id = tenantId;
    }
    if (adminFilters.searchQuery && adminFilters.searchQuery.trim()) {
      args.searchQuery = adminFilters.searchQuery.trim();
    }
    if (adminFilters.status && adminFilters.status !== "all") {
      args.status = adminFilters.status;
      if (adminFilters.status === "active") {
        args.active = true;
      } else if (adminFilters.status === "inactive") {
        args.active = false;
      }
    }
    if (adminFilters.provider && adminFilters.provider !== "all") {
      args.provider = adminFilters.provider;
    }
    if (adminFilters.is_synced !== undefined && adminFilters.is_synced !== null) {
      args.is_synced = adminFilters.is_synced;
      args.is_synced_user = adminFilters.is_synced;
    }
    (args as unknown as Record<string, unknown>).__contextSignature = contextSignature;
    return args;
  }, [currentPage, pageSize, contextSignature, tenantId, adminFilters]);

  const endUserQueryArgs = useMemo(() => {
    const args: UsersQueryParams = {
      ...(filters as UsersQueryParams),
      page: currentPage,
      limit: pageSize,
    };
    if (tenantId) {
      args.tenant_id = tenantId;
    }
    (args as unknown as Record<string, unknown>).__contextSignature = contextSignature;
    return args;
  }, [filters, currentPage, pageSize, contextSignature, tenantId]);

  // Configuration status - check if sync configs exist
  const { data: syncConfigsData } = useListSyncConfigsQuery();
  const configs = syncConfigsData?.configs || [];
  const adConfigured = configs.some((c) => c.sync_type === 'active_directory');
  const entraConfigured = configs.some((c) => c.sync_type === 'entra_id');

  // API data fetching with filters - fetch based on audience context
  const {
    data: adminUsersResponse,
    isLoading: adminUsersLoading,
    error: adminUsersError,
    refetch: refetchAdminUsers,
  } = useGetAdminUsersQuery(adminQueryArgs, {
    skip: !isAdmin || !tenantId,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: endUsersResponse,
    isLoading: endUsersLoading,
    error: endUsersError,
    refetch: refetchEndUsers,
  } = useGetEndUsersQuery(endUserQueryArgs, {
    skip: isAdmin || !tenantId,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  // Select appropriate data based on context
  const usersResponse = isAdmin ? adminUsersResponse : endUsersResponse;
  const usersLoading = isAdmin ? adminUsersLoading : endUsersLoading;
  const usersError = isAdmin ? adminUsersError : endUsersError;
  const refetchUsers = isAdmin ? refetchAdminUsers : refetchEndUsers;

  const lastLoggedSwitchRef = useRef(0);

  React.useEffect(() => {
    if (contextSwitchCounter === 0) return;
    if (lastLoggedSwitchRef.current === contextSwitchCounter) return;

    lastLoggedSwitchRef.current = contextSwitchCounter;
    const endpoint = isAdmin ? "/uflow/admin/users/list" : "/uflow/admin/enduser/list";
    console.log(
      `[UsersPage] Context switch #${contextSwitchCounter}: requesting ${endpoint} (page=${currentPage}, limit=${pageSize})`
    );
    if (isAdmin && tenantId) {
      refetchAdminUsers();
    } else if (!isAdmin && tenantId) {
      refetchEndUsers();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [contextSwitchCounter]);

  // Admin action mutations - use appropriate mutations based on audience
  const [deleteAdminUser] = useDeleteAdminUserMutation();
  const [setAdminUserActive] = useSetAdminUserActiveMutation();
  const [resetAdminUserPassword] = useResetAdminUserPasswordMutation();
  const [changeAdminUserPassword] = useChangeAdminUserPasswordMutation();

  const [deleteEndUser] = useDeleteUserMutation();
  const [setEndUserActive] = useSetUserActiveMutation();
  const [resetEndUserPassword] = useResetUserPasswordMutation();
  const [changeEndUserPassword] = useChangeUserPasswordMutation();

  // Select appropriate mutations based on context
  const setUserActive = isAdmin ? setAdminUserActive : setEndUserActive;
  const resetUserPassword = isAdmin ? resetAdminUserPassword : resetEndUserPassword;
  const changeUserPassword = isAdmin ? changeAdminUserPassword : changeEndUserPassword;

  // Process users data from API response
  const users = useMemo(() => {
    if (!usersResponse) return [];
    
    // Handle AuthSec API response structure
    let userData = [];
    if (Array.isArray(usersResponse)) {
      userData = usersResponse;
    } else if (usersResponse.users && Array.isArray(usersResponse.users)) {
      userData = usersResponse.users;
    } else if (usersResponse.data && Array.isArray(usersResponse.data)) {
      userData = usersResponse.data;
    } else {
      // Try to find any array property that might contain users
      const possibleArrays = Object.values(usersResponse).filter(Array.isArray);
      if (possibleArrays.length > 0) {
        userData = possibleArrays[0];
      } else {
        userData = [];
      }
    }
    
    return userData;
  }, [usersResponse]);

  // Transform AuthSec API users to EnhancedUser format for table compatibility
  const enhancedUsers = useMemo(() => {
    return users.map((user: any) => {
      const resolvedRoles = Array.isArray(user.roles) ? user.roles : [];
      const resolvedGroups = Array.isArray(user.groups) ? user.groups : [];

      const roleNames = resolvedRoles.map((role: any) => {
        if (typeof role === "string") return role;
        return role?.name || role?.role_name || role?.role || String(role);
      });
      const roleIds = resolvedRoles
        .map((role: any) => {
          if (typeof role === "string") return role;
          return role?.id || role?.role_id || role?.roleId || role?.name;
        })
        .filter(Boolean);

      const groupNames = resolvedGroups.map((group: any) => {
        if (typeof group === "string") return group;
        return group?.name || group?.group_name || group?.group || String(group);
      });
      const groupIds = resolvedGroups
        .map((group: any) => {
          if (typeof group === "string") return group;
          return group?.id || group?.group_id || group?.groupId || group?.name;
        })
        .filter(Boolean);

      const canonicalId = user.id || user.user_id || user.userId || user.userID;
      const name =
        user.name ||
        [user.first_name, user.last_name].filter(Boolean).join(" ") ||
        user.username ||
        user.email;

      const activeStatus =
        typeof user.active === "boolean" ? user.active : user.status === "active";

      const derivedStatus = (user.status ||
        (activeStatus ? "active" : "inactive")) as EnhancedUser["status"];

      const isSyncedValue =
        typeof user.is_synced === "boolean"
          ? user.is_synced
          : typeof user.is_synced_user === "boolean"
            ? user.is_synced_user
            : null;

      const providerValue =
        typeof user.provider === "string" && user.provider.trim().length > 0
          ? user.provider
          : typeof user.sync_source === "string" && user.sync_source.trim().length > 0
            ? user.sync_source
            : "local";

      return {
        ...user,
        id: canonicalId,
        user_id: canonicalId,
        name,
        email: user.email,
        avatar: user.provider_data?.picture || user.provider_data?.avatar_url || undefined,
        status: derivedStatus,
        active: activeStatus,
        directRoleIds: roleIds,
        groupIds,
        effectiveRoleIds: roleIds,
        lastLogin: user.last_login || user.lastLogin,
        last_login: user.last_login || user.lastLogin,
        lastLoginMethod: user.provider || user.last_login_method,
        loginCount: user.login_count ?? 0,
        groupNames,
        roleNames,
        isOrphan: groupIds.length === 0 && roleIds.length === 0,
        updatedAt: user.updated_at || user.updatedAt,
        updated_at: user.updated_at || user.updatedAt,
        provider: providerValue,
        client_id: user.client_id,
        tenant_id: user.tenant_id,
        project_id: user.project_id,
        tenant_domain: user.tenant_domain,
        provider_id: user.provider_id,
        provider_data: user.provider_data,
        MFAEnabled: user.mfa_enabled,
        MFAMethod: user.mfa_method,
        MFADefaultMethod: user.mfa_default_method,
        MFAEnrolledAt: user.mfa_enrolled_at,
        mfa_verified: user.mfa_verified,
        is_synced_user:
          typeof user.is_synced_user === "boolean" ? user.is_synced_user : isSyncedValue ?? undefined,
        is_synced: isSyncedValue ?? undefined,
        last_sync_at: user.last_sync_at,
        sync_source: user.sync_source,
        sync_status: user.sync_status,
        sync_provider: user.sync_provider,
        external_id: user.external_id,
        username: user.username,
        accepted_invite:
          typeof user.accepted_invite === "boolean"
            ? user.accepted_invite
            : user.accepted_at
              ? true
              : undefined,
        roles: resolvedRoles,
        groups: resolvedGroups,
      } as EnhancedUser;
    });
  }, [users]);

  const enhancedUsersCount = enhancedUsers.length;

  // Calculate counts for each source
  const adCount = useMemo(() => {
    return enhancedUsers.filter((user: any) =>
      user.provider === 'ad_sync' || user.sync_source === 'ad'
    ).length;
  }, [enhancedUsers]);

  const entraCount = useMemo(() => {
    return enhancedUsers.filter((user: any) =>
      user.provider === 'entra_id' || user.provider === 'azure_ad' || user.sync_source === 'entra'
    ).length;
  }, [enhancedUsers]);

  const authsecCount = useMemo(() => {
    return enhancedUsers.filter((user: any) =>
      user.provider !== 'ad_sync' &&
      user.provider !== 'entra_id' &&
      user.provider !== 'azure_ad' &&
      user.sync_source !== 'ad' &&
      user.sync_source !== 'entra'
    ).length;
  }, [enhancedUsers]);

  const adminProviders = useMemo(() => {
    if (!isAdmin) return [];
    const providerSet = new Set<string>();

    DEFAULT_ADMIN_PROVIDERS.forEach((provider) => providerSet.add(provider));

    users.forEach((user: any) => {
      if (user?.provider) providerSet.add(user.provider);
      if (user?.sync_source) providerSet.add(user.sync_source);
      if (user?.sync_provider) providerSet.add(user.sync_provider);
    });

    return Array.from(providerSet).filter(Boolean) as string[];
  }, [users, isAdmin]);

  React.useEffect(() => {
    console.log("[UsersPage] Query status update", {
      audience: contextKey,
      loading: usersLoading,
      users: enhancedUsersCount,
      error: Boolean(usersError),
    });
  }, [contextKey, usersLoading, usersError, enhancedUsersCount]);

  const selectedUserDetails = useMemo(() => {
    if (selectedUsers.length === 0) return [];
    const userMap = new Map(enhancedUsers.map((user: any) => [user.id, user]));
    return selectedUsers
      .map((id) => userMap.get(id))
      .filter(Boolean) as any[];
  }, [selectedUsers, enhancedUsers]);

  const suggestGroupName = useCallback(() => {
    const suffix = "group";
    if (selectedUserDetails.length === 1) {
      const primary = selectedUserDetails[0];
      const baseSource = primary.name || primary.email || primary.id;
      if (baseSource) {
        const slug = baseSource
          .toString()
          .split("@")[0]
          .toLowerCase()
          .replace(/[^a-z0-9]+/g, "-")
          .replace(/-+/g, "-")
          .replace(/^-|-$/g, "");
        if (slug) {
          return `${slug}-${suffix}`;
        }
      }
      return isAdmin ? "new-admin-group" : "new-user-group";
    }

    if (selectedUserDetails.length > 1) {
      return `${isAdmin ? "admin-group" : "user-group"}-${selectedUserDetails.length}`;
    }

    return isAdmin ? "new-admin-group" : "new-user-group";
  }, [selectedUserDetails, isAdmin]);

  // Handle user actions
  const handleDeleteUser = async (userId: string) => {
    try {
      if (isAdmin) {
        await deleteAdminUser({ user_id: userId }).unwrap();
      } else {
        if (!tenantId) {
          toast.error("Tenant context missing; cannot delete end user.");
          return;
        }
        await deleteEndUser({ tenant_id: tenantId, user_id: userId }).unwrap();
      }
      toast.success("User deletion requested; changes may take a moment to reflect.");
      refetchUsers();
    } catch (error) {
      console.error("Failed to delete user:", error);
      toast.error("Failed to delete user");
    }
  };

  const handleActivateUser = async (userId: string, active: boolean) => {
    try {
      await setUserActive({ user_id: userId, active }).unwrap();
      toast.success(`User ${active ? 'activated' : 'deactivated'} successfully`);
      refetchUsers();
    } catch (error) {
      console.error("Failed to update user status:", error);
      toast.error("Failed to update user status");
    }
  };

  const handleResetPassword = async (userId: string, email: string) => {
    try {
      await resetUserPassword({ email }).unwrap();
      toast.success("Password reset email sent");
    } catch (error) {
      console.error("Failed to reset password:", error);
      toast.error("Failed to reset password");
    }
  };

  const handleChangePassword = async (userId: string, email: string) => {
    // For now, we'll use a simple prompt. In a real app, you'd use a proper modal
    const newPassword = prompt("Enter new password for user:");
    if (!newPassword) return;

    try {
      await changeUserPassword({ email, new_password: newPassword }).unwrap();
      toast.success("Password changed successfully");
    } catch (error: any) {
      console.error("Change password error:", error);
      toast.error(`Failed to change password: ${error.data?.message || error.message}`);
    }
  };

  // Handler for source tab changes
  const handleSourceTabChange = useCallback((source: UserSource) => {
    setSelectedSource(source);

    // If unconfigured, show confirmation dialog
    if (source === 'ad' && !adConfigured) {
      setNotConfiguredType('ad');
      setShowNotConfiguredDialog(true);
      return;
    }
    if (source === 'entra' && !entraConfigured) {
      setNotConfiguredType('entra');
      setShowNotConfiguredDialog(true);
      return;
    }

    // Apply filter based on selected source
    if (source === 'all') {
      // Clear provider filter
      if (isAdmin) {
        setAdminFilters({ ...adminFilters, provider: undefined });
      } else {
        setFilters({ ...filters, provider: undefined });
      }
    } else if (source === 'ad') {
      // Filter for AD synced users
      if (isAdmin) {
        setAdminFilters({ ...adminFilters, provider: 'ad_sync' });
      } else {
        setFilters({ ...filters, provider: 'ad_sync' });
      }
    } else if (source === 'entra') {
      // Filter for Entra synced users
      if (isAdmin) {
        setAdminFilters({ ...adminFilters, provider: 'entra_id' });
      } else {
        setFilters({ ...filters, provider: 'entra_id' });
      }
    } else if (source === 'authsec') {
      // Filter for AuthSec users - clear provider to allow user to select
      // User can then select specific providers (Google, GitHub, local, etc.)
      if (isAdmin) {
        setAdminFilters({ ...adminFilters, provider: undefined, is_synced: false });
      } else {
        setFilters({ ...filters, provider: undefined });
      }
    }
    setCurrentPage(1);
  }, [adConfigured, entraConfigured, isAdmin, adminFilters, filters]);

  const handleClearSelection = () => {
    setSelectedUsers([]);
  };

  // Handler to configure AD/Entra from the not configured dialog
  const handleConfigureFromDialog = () => {
    if (notConfiguredType === 'ad') {
      setViewMode('create-ad-sync');
    } else if (notConfiguredType === 'entra') {
      setViewMode('create-entra-sync');
    }
    setShowNotConfiguredDialog(false);
    setNotConfiguredType(null);
  };

  // Handler to cancel the not configured dialog
  const handleCancelNotConfiguredDialog = () => {
    setShowNotConfiguredDialog(false);
    setNotConfiguredType(null);
    setSelectedSource('all'); // Reset to 'all' tab
  };

  // Bulk actions
  const handleBulkAction = async (action: string) => {
    console.log("[UsersPage] Bulk action triggered", {
      action,
      selectedCount: selectedUsers.length,
      context: contextKey,
    });

    if (action === "create-group") {
      const entityLabel = "group";
      const subjectLabel = isAdmin ? "admin" : "end user";

      if (selectedUsers.length === 0) {
        toast.error(`Select at least one ${subjectLabel} before creating a ${entityLabel}.`);
        return;
      }

      const suggestedName = suggestGroupName();
      const selectedUserLabels = selectedUserDetails.map(
        (user: any) => user.email || user.name || user.id
      );

      navigate("/groups/create", {
        state: {
          prefillGroup: {
            suggestedName: suggestedName || (isAdmin ? "new-admin-group" : "new-user-group"),
            selectedUserIds: selectedUsers,
            selectedUserEmails: selectedUserLabels,
            source: `users-bulk-create-${contextKey}`,
          },
        },
      });
      return;
    }

    if (action === "assign-role") {
      if (selectedUsers.length === 0) {
        toast.error("Select at least one user to assign a role.");
        return;
      }
      setShowAssignRoleModal(true);
      return;
    }

    const actionLabels: Record<string, string> = {
      activate: "Activate",
      deactivate: "Deactivate",
      delete: "Delete",
      export: "Export",
    };

    const actionLabel = actionLabels[action] || action;
    toast.success(`${actionLabel} ${selectedUsers.length} users (functionality coming soon)`);

    setSelectedUsers([]);
  };

  // Get selected user details for the assign role modal
  const selectedUsersForModal = useMemo(() => {
    return enhancedUsers
      .filter((user) => selectedUsers.includes(user.id))
      .map((user) => ({
        id: user.id,
        name: user.name || "",
        email: user.email,
      }));
  }, [selectedUsers, enhancedUsers]);

  // State for single user assign role (from row action)
  const [singleUserForRole, setSingleUserForRole] = useState<{ id: string; name: string; email: string } | null>(null);

  // Handler for single user assign role from row action
  const handleAssignRoleForUser = (userId: string, userName: string, userEmail: string) => {
    setSingleUserForRole({ id: userId, name: userName, email: userEmail });
    setShowAssignRoleModal(true);
  };

  // Get users for modal - either selected users (bulk) or single user (row action)
  const usersForAssignRoleModal = useMemo(() => {
    if (singleUserForRole) {
      return [singleUserForRole];
    }
    return selectedUsersForModal;
  }, [singleUserForRole, selectedUsersForModal]);

  // Get error message if exists
  const errorMessage = usersError
    ? (usersError as any)?.data?.message || (usersError as any)?.message || "Failed to fetch user data"
    : null;

  const showInitialSkeleton = usersLoading && !usersError && enhancedUsersCount === 0;
  const skeletonRowCount = Math.max(5, Math.min(pageSize, 10));

  return (
    <div
      className={viewMode !== 'list' ? ' flex flex-col' : 'min-h-screen'}
      ref={mainAreaRef}
    >
      <div className={viewMode !== 'list' ? 'flex-1 flex flex-col overflow-hidden w-full' : 'space-y-4 p-6 max-w-10xl mx-auto'}>
        {/* Show inline forms when in create or edit mode */}
        {viewMode === 'create-ad-sync' || viewMode === 'edit-ad-sync' ? (
          <ADSyncInlineForm
            onClose={() => {
              setViewMode('list');
              setEditingSyncConfig(null);
            }}
            onSuccess={() => {
              setViewMode('list');
              setEditingSyncConfig(null);
              setSelectedSource('ad');
              refetchUsers();
            }}
            editConfig={viewMode === 'edit-ad-sync' ? editingSyncConfig : null}
          />
        ) : viewMode === 'create-entra-sync' || viewMode === 'edit-entra-sync' ? (
          <EntraSyncInlineForm
            onClose={() => {
              setViewMode('list');
              setEditingSyncConfig(null);
            }}
            onSuccess={() => {
              setViewMode('list');
              setEditingSyncConfig(null);
              setSelectedSource('entra');
              refetchUsers();
            }}
            editConfig={viewMode === 'edit-entra-sync' ? editingSyncConfig : null}
          />
        ) : (
          <>
            {/* Header */}
            <PageHeader
              title={audienceCopy.title}
              description={`${audienceCopy.descriptionPrefix}`}
              actionsPosition="below"
              actions={
                <UserSourceTabs
                  selectedSource={selectedSource}
                  onSourceChange={handleSourceTabChange}
                  onAddUsersClick={() => setShowAddUsersModal(true)}
                  adCount={adCount}
                  entraCount={entraCount}
                  authsecCount={authsecCount}
                  adConfigured={adConfigured}
                  entraConfigured={entraConfigured}
                />
              }
            />

            {/* Filter/Search Card */}
            {isAdminView ? (
          <AdminUsersFilterCard
            filters={adminFilters}
            onFiltersChange={handleAdminFiltersChange}
            providers={adminProviders}
            showProviderFilter={selectedSource === 'authsec'}
            authsecMode={selectedSource === 'authsec'}
          />
        ) : (
          <UsersFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
            usersData={users}
          />
        )}

        {/* Table */}
        <TableCard>
            <CardContent variant="flush">
              <div className="relative">
                {errorMessage ? (
                  <div className="p-8 text-center">
                    <div className="flex flex-col items-center space-y-4">
                      <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                        <UserX className="h-8 w-8 text-red-600 dark:text-red-400" />
                      </div>
                      <div>
                        <h3 className="text-lg font-semibold text-red-900 dark:text-red-100">
                          Unable to Load Users
                        </h3>
                        <p className="text-red-700 dark:text-red-300 mt-1">
                          {errorMessage}
                        </p>
                      </div>
                    </div>
                  </div>
                ) : showInitialSkeleton ? (
                  <UsersTableSkeleton rows={skeletonRowCount} />
                ) : isAdmin ? (
                  <AdminUsersTable
                    users={enhancedUsers}
                    selectedUserIds={selectedUsers}
                    onSelectionChange={(selectedIds) => setSelectedUsers(selectedIds)}
                    onSelectAll={() => {
                      if (selectedUsers.length === enhancedUsers.length) {
                        setSelectedUsers([]);
                      } else {
                        setSelectedUsers(enhancedUsers.map((user) => user.id));
                      }
                    }}
                    actions={{
                      onDelete: handleDeleteUser,
                      onActivateUser: handleActivateUser,
                      onResetPassword: handleResetPassword,
                      onChangePassword: handleChangePassword,
                      onAssignRole: handleAssignRoleForUser,
                    }}
                  />
                ) : (
                  <EndUserUsersTable
                    users={enhancedUsers}
                    selectedUserIds={selectedUsers}
                    onSelectionChange={(selectedIds) => setSelectedUsers(selectedIds)}
                    onSelectAll={() => {
                      if (selectedUsers.length === enhancedUsers.length) {
                        setSelectedUsers([]);
                      } else {
                        setSelectedUsers(enhancedUsers.map((user) => user.id));
                      }
                    }}
                    actions={{
                      onDelete: handleDeleteUser,
                      onActivateUser: handleActivateUser,
                      onResetPassword: handleResetPassword,
                      onChangePassword: handleChangePassword,
                      onAssignRole: handleAssignRoleForUser,
                    }}
                  />
                )}
                  {usersLoading && enhancedUsersCount > 0 && (
                    <div className="absolute inset-0 bg-white/50 dark:bg-neutral-900/50 backdrop-blur-sm flex items-center justify-center">
                      <div className="flex items-center space-x-2">
                        <RefreshCw className="h-4 w-4 animate-spin" />
                        <span className="text-sm font-medium">Refreshing...</span>
                      </div>
                    </div>
                )}
              </div>
            </CardContent>
        </TableCard>
          </>
        )}
      </div>

      {/* Bulk Actions Bar - only show in list mode */}
      {viewMode === 'list' && selectedUsers.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedUsers.length}
          onClearSelection={handleClearSelection}
          onBulkAction={handleBulkAction}
        />
      )}

      {/* Modals - only show in list mode */}
      {viewMode === 'list' && (
        <>
          {/* Add Users Modal */}
          {showAddUsersModal && (
        <AddUsersModal
          isOpen={showAddUsersModal}
          onClose={() => setShowAddUsersModal(false)}
          audience={isAdmin ? 'admin' : 'endUser'}
          onOpenInviteModal={() => {
            setShowAddUsersModal(false);
            setShowInviteModal(true);
          }}
          onOpenSyncConfigModal={(config, type) => {
            setShowAddUsersModal(false);
            // Use inline form for both creating and editing
            if (config) {
              // Editing existing config
              setEditingSyncConfig(config);
              if (config.sync_type === 'entra_id') {
                setViewMode('edit-entra-sync');
              } else {
                setViewMode('edit-ad-sync');
              }
            } else {
              // Creating new config
              setEditingSyncConfig(null);
              if (type === 'entra_id') {
                setViewMode('create-entra-sync');
              } else {
                setViewMode('create-ad-sync');
              }
            }
          }}
        />
      )}

      {/* Invite User Modal */}
      {showInviteModal && (
        <InviteUserModal
          isOpen={showInviteModal}
          onClose={() => setShowInviteModal(false)}
          audience={isAdmin ? 'admin' : 'endUser'}
          onSuccess={() => {
            setShowInviteModal(false);
            setSelectedSource('authsec');
            refetchUsers();
          }}
        />
      )}



      {/* Assign Role to Users Modal */}
      <MapRoleToScopeModal
        open={showAssignRoleModal}
        onOpenChange={(open) => {
          setShowAssignRoleModal(open);
          if (!open) {
            setSingleUserForRole(null);
          }
        }}
        preselectedUsers={usersForAssignRoleModal}
        onSuccess={() => {
          setSelectedUsers([]);
          setSingleUserForRole(null);
          refetchUsers();
        }}
      />

      {/* Not Configured Dialog */}
      <Dialog open={showNotConfiguredDialog} onOpenChange={setShowNotConfiguredDialog}>
        <DialogContent>
          <DialogHeader>
            <div className="flex items-center gap-3">
              <div className="p-2 bg-amber-100 dark:bg-amber-900/20 rounded-lg">
                <AlertTriangle className="h-5 w-5 text-amber-600 dark:text-amber-500" />
              </div>
              <DialogTitle>
                {notConfiguredType === 'ad' ? 'Active Directory' : 'Entra ID'} Not Configured
              </DialogTitle>
            </div>
          </DialogHeader>
          <DialogDescription className="pt-4">
            {notConfiguredType === 'ad'
              ? 'Active Directory sync is not configured yet. Would you like to configure it now?'
              : 'Entra ID sync is not configured yet. Would you like to configure it now?'
            }
          </DialogDescription>
          <DialogFooter className="gap-3">
            <Button
              variant="outline"
              onClick={handleCancelNotConfiguredDialog}
            >
              Cancel
            </Button>
            <Button
              onClick={handleConfigureFromDialog}
            >
              Configure {notConfiguredType === 'ad' ? 'Active Directory' : 'Entra ID'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
        </>
      )}
    </div>
  );
}
