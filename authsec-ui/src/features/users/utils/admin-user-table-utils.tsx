import * as React from "react";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Trash2,
  RefreshCw,
  Key,
  UserCheck,
  UserX,
  User,
  Shield,
  Calendar,
} from "lucide-react";
import type { EnhancedUser } from "../../../types/entities";
import type { ResponsiveColumnDef } from "../../../components/ui/responsive-data-table";
import { CopyButton } from "../../../components/ui/copy-button";
import { UserTableUtils } from "./user-table-utils";
import { useGetAdminUserQuery } from "@/app/api/admin/usersApi";

// Admin table action handlers interface
export interface AdminUserTableActions {
  onDelete: (userId: string) => void;
  onActivateUser: (userId: string, active: boolean) => void;
  onResetPassword: (userId: string, email: string) => void;
  onChangePassword: (userId: string, email: string) => void;
  onAssignRole?: (userId: string, userName: string, userEmail: string) => void;
}

// Format provider name
const formatProvider = (provider: string) => {
  const providerMap: Record<string, string> = {
    local: "Local",
    entra_id: "Microsoft Entra ID",
    azure_ad: "Azure AD",
    google: "Google",
    github: "GitHub",
    auth0: "Auth0",
    okta: "Okta",
  };
  return providerMap[provider] || provider;
};

// Admin user cell - Name + Email with copy (no avatar, clean design)
export function AdminUserCell({ user }: { user: EnhancedUser }) {
  return (
    <div className="min-w-0">
      <div className="font-medium truncate text-foreground" title={user.name}>
        {user.name}
      </div>
      <div className="flex items-center gap-2 mt-1">
        <span className="text-sm text-foreground truncate" title={user.email}>
          {user.email}
        </span>
        <CopyButton
          text={user.email}
          label="Email"
          variant="ghost"
          size="sm"
          className="h-5 w-5 p-0 flex-shrink-0"
        />
      </div>
    </div>
  );
}

export function AdminEmailCell({ user }: { user: EnhancedUser }) {
  if (!user.email) {
    return <span className="text-xs text-muted-foreground">—</span>;
  }
  return (
    <div className="flex items-center gap-2 group">
      <span className="text-sm truncate" title={user.email}>
        {user.email}
      </span>
      <CopyButton
        text={user.email}
        label="Email"
        variant="ghost"
        size="sm"
        className="h-4 w-4 p-0 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
      />
    </div>
  );
}

export function AdminUsernameCell({ user }: { user: EnhancedUser }) {
  if (!user.username) {
    return <span className="text-sm text-foreground">—</span>;
  }
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-medium truncate" title={user.username}>
        {user.username}
      </span>
      <CopyButton
        text={user.username}
        label="Username"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Admin client ID cell with copy
export function AdminClientIdCell({ user }: { user: EnhancedUser }) {
  const clientId = (user as any).client_id;
  if (!clientId) {
    return <span className="text-xs text-muted-foreground">—</span>;
  }

  return (
    <div className="flex items-center gap-2 group">
      <span className="text-xs font-mono truncate text-muted-foreground" title={clientId}>
        {clientId.length > 16
          ? `${clientId.substring(0, 8)}...${clientId.substring(clientId.length - 4)}`
          : clientId}
      </span>
      <CopyButton
        text={clientId}
        label="Client ID"
        variant="ghost"
        size="sm"
        className="h-4 w-4 p-0 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
      />
    </div>
  );
}

export function AdminIdentifierCell({
  value,
  label,
}: {
  value?: string | null;
  label: string;
}) {
  if (!value) {
    return <span className="text-sm text-foreground">—</span>;
  }
  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-mono truncate" title={value}>
        {value.length > 16 ? `${value.slice(0, 8)}...${value.slice(-4)}` : value}
      </span>
      <CopyButton
        text={value}
        label={label}
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

export function AdminDateCell({ value }: { value?: string | null }) {
  if (!value) {
    return <span className="text-sm text-foreground">—</span>;
  }
  const formatted = (() => {
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) {
      return value;
    }
    return parsed.toLocaleString();
  })();
  return (
    <span className="text-sm text-foreground" title={formatted}>
      {formatted}
    </span>
  );
}

// Admin status cell - Simple Active/Inactive badge
export function AdminStatusCell({ user }: { user: EnhancedUser }) {
  const isActive = user.active !== undefined ? user.active : user.status === "active";
  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
      isActive ? "text-emerald-600 dark:text-emerald-400" : "text-foreground/60"
    }`}>
      <span className={`h-1.5 w-1.5 rounded-full ${isActive ? "bg-emerald-500" : "bg-foreground/40"}`} />
      {isActive ? "Active" : "Inactive"}
    </span>
  );
}

export function AdminInviteAcceptedCell({ user }: { user: EnhancedUser }) {
  const accepted = Boolean(user.accepted_invite);
  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
      accepted ? "text-emerald-600 dark:text-emerald-400" : "text-amber-600 dark:text-amber-400"
    }`}>
      <span className={`h-1.5 w-1.5 rounded-full ${accepted ? "bg-emerald-500" : "bg-amber-500"}`} />
      {accepted ? "Accepted" : "Pending"}
    </span>
  );
}

// Admin provider cell - Simple provider badge
export function AdminProviderCell({ user }: { user: EnhancedUser }) {
  const provider = (user.provider && user.provider.trim().length > 0) ? user.provider : "local";
  return (
    <span className="text-xs font-medium text-foreground">
      {formatProvider(provider)}
    </span>
  );
}

// Admin actions cell - Dropdown menu
export function AdminActionsCell({
  user,
  actions,
}: {
  user: EnhancedUser;
  actions: AdminUserTableActions;
}) {
  return (
    <div className="flex items-center justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" visualVariant="row-actions">
          {/* Activate/Deactivate */}
          {user.active ? (
            <DropdownMenuItem onClick={() => actions.onActivateUser(user.id, false)}>
              <UserX className="mr-2 h-4 w-4" />
              Deactivate User
            </DropdownMenuItem>
          ) : (
            <DropdownMenuItem onClick={() => actions.onActivateUser(user.id, true)}>
              <UserCheck className="mr-2 h-4 w-4" />
              Activate User
            </DropdownMenuItem>
          )}

          {actions.onAssignRole && (
            <DropdownMenuItem onClick={() => actions.onAssignRole!(user.id, user.name || "", user.email)}>
              <Shield className="mr-2 h-4 w-4" />
              Assign Role
            </DropdownMenuItem>
          )}

          <DropdownMenuItem onClick={() => actions.onResetPassword(user.id, user.email)}>
            <RefreshCw className="mr-2 h-4 w-4" />
            Reset Password
          </DropdownMenuItem>

          <DropdownMenuItem onClick={() => actions.onChangePassword(user.id, user.email)}>
            <Key className="mr-2 h-4 w-4" />
            Change Password
          </DropdownMenuItem>

          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => actions.onDelete(user.id)}
            className="text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// Admin User Expanded Row Component - Following ClientsPage pattern
export function AdminUserExpandedRow({ user }: { user: EnhancedUser }) {
  const { data: serverUser, isFetching: isFetchingDetails } = useGetAdminUserQuery(
    user.id ?? "",
    { skip: !user?.id }
  );

  const mergedUser = React.useMemo(() => {
    if (!serverUser) return user;
    return { ...user, ...serverUser };
  }, [user, serverUser]);

  const InfoLine = ({
    label,
    value,
    copyable = false,
  }: {
    label: string;
    value?: string | number | boolean | null;
    copyable?: boolean;
  }) => {
    if (value === undefined || value === null || value === "") return null;
    const stringValue = typeof value === "boolean" ? (value ? "Yes" : "No") : String(value);
    return (
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-foreground">{label}</span>
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono text-xs truncate text-foreground" title={stringValue}>
            {stringValue}
          </span>
          {copyable && <CopyButton text={stringValue} label={label} size="sm" />}
        </div>
      </div>
    );
  };

  const resolvedRoles = Array.isArray(mergedUser.roles) ? mergedUser.roles : [];
  const providerLabel = formatProvider(mergedUser.provider || "local");
  const isSynced = mergedUser.is_synced_user ?? mergedUser.is_synced ?? false;

  const formatDateTime = (value?: string | null) => {
    if (!value) return undefined;
    const parsed = new Date(value);
    if (Number.isNaN(parsed.getTime())) return value;
    return parsed.toLocaleString();
  };

  const lastLoginText = UserTableUtils.formatLastLogin(
    mergedUser.last_login || (serverUser as any)?.last_login
  );
  const lastSyncText = formatDateTime(mergedUser.last_sync_at);
  const updatedAtText = formatDateTime(
    typeof mergedUser.updated_at === "string" ? mergedUser.updated_at : null
  );
  const mfaMethods = mergedUser.MFAMethod || mergedUser.mfa_method;
  const mfaDefaultMethod = mergedUser.MFADefaultMethod || mergedUser.mfa_default_method;
  const mfaEnrolledText = formatDateTime(
    typeof mergedUser.MFAEnrolledAt === "string"
      ? mergedUser.MFAEnrolledAt
      : typeof mergedUser.mfa_enrolled_at === "string"
        ? mergedUser.mfa_enrolled_at
        : null
  );

  return (
    <div className="space-y-5">
      <div className="grid gap-6 md:grid-cols-2">
        <div className="space-y-4">
          <div className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <User className="h-4 w-4" />
            User Details
            {isFetchingDetails && (
              <RefreshCw className="h-3 w-3 animate-spin text-foreground" />
            )}
          </div>
          <div className="space-y-3 text-sm">
            <InfoLine label="Name" value={mergedUser.name || "Unnamed user"} />
            <InfoLine label="Email" value={mergedUser.email} copyable />
            <InfoLine label="Username" value={mergedUser.username} />
            <InfoLine label="Client ID" value={mergedUser.client_id} copyable />
            <InfoLine label="Tenant ID" value={mergedUser.tenant_id} copyable />
            <InfoLine label="Project ID" value={mergedUser.project_id} copyable />
            <InfoLine label="Tenant Domain" value={mergedUser.tenant_domain} />
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-foreground">Status</span>
              <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                mergedUser.active ? "text-emerald-600 dark:text-emerald-400" : "text-foreground/60"
              }`}>
                <span className={`h-1.5 w-1.5 rounded-full ${mergedUser.active ? "bg-emerald-500" : "bg-foreground/40"}`} />
                {mergedUser.active ? "Active" : "Inactive"}
              </span>
            </div>
            <InfoLine label="Invite Accepted" value={mergedUser.accepted_invite ?? false} />
            <InfoLine label="Status (raw)" value={mergedUser.status} />
            <InfoLine label="Provider" value={providerLabel} />
            <InfoLine label="Provider ID" value={mergedUser.provider_id} copyable />
            <InfoLine label="External ID" value={mergedUser.external_id} copyable />
            <InfoLine label="Tenant Domain" value={mergedUser.tenant_domain} />
            <InfoLine label="Updated At" value={updatedAtText} />
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-foreground">Last Login</span>
              <span className="text-xs text-foreground">{lastLoginText}</span>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Shield className="h-4 w-4" />
            Access & Membership
            {isFetchingDetails && resolvedRoles.length === 0 && (
              <RefreshCw className="h-3 w-3 animate-spin text-foreground" />
            )}
          </h4>
          <div className="space-y-4 text-sm">
            <div>
              <span className="text-xs font-medium uppercase tracking-wide text-foreground">
                Roles ({resolvedRoles.length})
              </span>
              <div className="mt-2 flex flex-wrap gap-2">
                {resolvedRoles.length > 0 ? (
                  resolvedRoles.map((role: any, index: number) => (
                    <span
                      key={`${mergedUser.id}-role-${index}`}
                      className="text-xs font-medium text-foreground"
                    >
                      {typeof role === "string"
                        ? role
                        : role?.name || role?.role_name || role?.id || "Role"}
                      {index < resolvedRoles.length - 1 && <span className="text-foreground/40">,</span>}
                    </span>
                  ))
                ) : (
                  <span className="text-xs text-foreground/50">No roles assigned</span>
                )}
              </div>
            </div>

          </div>
        </div>
      </div>

      <div className="grid gap-6 md:grid-cols-2">
        <div className="space-y-4 text-sm">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Shield className="h-4 w-4" />
            Security & MFA
          </h4>
          <div className="space-y-3">
            <InfoLine label="MFA Enabled" value={mergedUser.MFAEnabled ?? mergedUser.mfa_enabled} />
            <InfoLine label="MFA Verified" value={mergedUser.mfa_verified} />
            <InfoLine label="MFA Default Method" value={mfaDefaultMethod} />
            <InfoLine
              label="MFA Methods"
              value={Array.isArray(mfaMethods) ? mfaMethods.join(", ") : mfaMethods}
            />
            <InfoLine label="MFA Enrolled" value={mfaEnrolledText} />
          </div>
        </div>

        <div className="space-y-4 text-sm">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <RefreshCw className="h-4 w-4" />
            Synchronization
          </h4>
          <div className="space-y-3">
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-foreground">Sync Status</span>
              <span className="text-xs font-medium text-foreground">
                {isSynced ? "Externally synchronized" : "Managed locally"}
              </span>
            </div>
            <InfoLine label="Sync Source" value={mergedUser.sync_source || mergedUser.provider_name} />
            <InfoLine label="Sync Provider" value={mergedUser.sync_provider} />
            <InfoLine label="Sync Status (raw)" value={mergedUser.sync_status} />
            <InfoLine label="Last Synced" value={lastSyncText} />
            <InfoLine label="Is Synced User" value={isSynced} />
            <InfoLine label="Provider Data Present" value={mergedUser.provider_data ? "Yes" : "No"} />
          </div>
        </div>
      </div>

      {mergedUser.provider_data && typeof mergedUser.provider_data === "object" && (
        <div className="space-y-3 text-sm">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Shield className="h-4 w-4" />
            Provider Metadata
          </h4>
          <pre className="max-h-64 overflow-auto rounded border border-black/10 dark:border-white/10 bg-black/5 dark:bg-white/5 p-3 text-xs text-left text-foreground">
            {JSON.stringify(mergedUser.provider_data, null, 2)}
          </pre>
        </div>
      )}
    </div>
  );
}

// Admin user table column factory - Simple 5 columns with responsive behavior
export function createAdminUserTableColumns(
  actions: AdminUserTableActions
): ResponsiveColumnDef<EnhancedUser, any>[] {
  return [
    {
      id: "name",
      accessorKey: "name",
      header: "Name",
      resizable: true,
      responsive: false, // Always visible
      cell: ({ row }: { row: any }) => <AdminUserCell user={row.original} />,
      cellClassName: "max-w-0",
    },
    {
      id: "email",
      accessorKey: "email",
      header: "Email",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <AdminEmailCell user={row.original} />,
    },
    {
      id: "username",
      accessorKey: "username",
      header: "Username",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <AdminUsernameCell user={row.original} />,
    },
    {
      id: "status",
      accessorKey: "active",
      header: "Status",
      resizable: true,
      responsive: true, // Hides on small screens
      cell: ({ row }: { row: any }) => <AdminStatusCell user={row.original} />,
    },
    {
      id: "inviteAccepted",
      accessorKey: "accepted_invite",
      header: "Invite Accepted",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <AdminInviteAcceptedCell user={row.original} />,
    },
    {
      id: "provider",
      accessorKey: "provider",
      header: "Provider",
      resizable: true,
      responsive: true, // Hides on small screens
      cell: ({ row }: { row: any }) => <AdminProviderCell user={row.original} />,
    },
    {
      id: "syncSource",
      accessorKey: "sync_source",
      header: "Sync Source",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm capitalize">
          {row.original.sync_source || row.original.provider_name || "—"}
        </span>
      ),
    },
    {
      id: "tenantId",
      accessorKey: "tenant_id",
      header: "Tenant ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <AdminIdentifierCell value={row.original.tenant_id} label="Tenant ID" />
      ),
    },
    {
      id: "clientId",
      accessorKey: "client_id",
      header: "Client ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <AdminClientIdCell user={row.original} />,
    },
    {
      id: "projectId",
      accessorKey: "project_id",
      header: "Project ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <AdminIdentifierCell value={row.original.project_id} label="Project ID" />
      ),
    },
    {
      id: "lastLogin",
      accessorKey: "last_login",
      header: "Last Login",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm">
          {UserTableUtils.formatLastLogin(row.original.last_login)}
        </span>
      ),
    },
    {
      id: "updatedAt",
      accessorKey: "updated_at",
      header: "Updated At",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <AdminDateCell value={row.original.updated_at} />,
    },
    {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false, // Always visible
      enableSorting: false,
      cell: ({ row }: { row: any }) => <AdminActionsCell user={row.original} actions={actions} />,
      cellClassName: "text-center",
    },
  ];
}
