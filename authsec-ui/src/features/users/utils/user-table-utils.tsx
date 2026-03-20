import * as React from "react";
import { Avatar, AvatarFallback, AvatarImage } from "../../../components/ui/avatar";
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
  Calendar,
  Users,
  Mail,
  Shield,
  MoreHorizontal,
  Edit,
  Copy,
  Settings,
  Trash2,
  Lock,
  UserCheck,
  UserX,
  Key,
  RefreshCw,
  AlertTriangle,
} from "lucide-react";
import type { User } from "../../../types/database";
import type { EnhancedUser } from "../../../types/entities";
import { CopyButton } from "../../../components/ui/copy-button";

// Extended User type that includes fields from API response not in the base User type
export interface ApiUser extends User {
  external_id?: string;
  sync_source?: string;
  last_sync_at?: string;
  is_synced_user?: boolean;
  avatar?: string;
  avatar_url?: string;
}
import type { ResponsiveColumnDef } from "../../../components/ui/responsive-data-table";

const formatIdentifier = (value?: string | null) => {
  if (!value) return "";
  if (value.length <= 16) return value;
  return `${value.slice(0, 8)}…${value.slice(-4)}`;
};

// User-specific utility functions
export const UserTableUtils = {
  // Status badge variant mapping
  getStatusVariant: (active: boolean) => {
    return active ? "default" : "secondary";
  },

  // Role badge color mapping
  getRoleColor: (roles: any[]) => {
    if (!roles || roles.length === 0) {
      return "bg-gray-100 text-gray-800 border-gray-200";
    }
    return "bg-blue-100 text-blue-800 border-blue-200";
  },

  formatProvider: (provider: string) => {
    const providerMap: Record<string, string> = {
      entra_id: "Microsoft Entra ID",
      azure_ad: "Azure AD",
      google: "Google",
      github: "GitHub",
      auth0: "Auth0",
      okta: "Okta",
    };
    return providerMap[provider] || provider;
  },

  getMfaStatusVariant: (mfaEnabled: boolean) => {
    return mfaEnabled ? "default" : "outline";
  },

  // Time formatting for last login
  formatLastLogin: (timestamp?: string) => {
    if (!timestamp) return "Never";

    const now = new Date();
    const time = new Date(timestamp);
    const diffInMinutes = Math.floor((now.getTime() - time.getTime()) / (1000 * 60));

    if (diffInMinutes < 60) return `${diffInMinutes}m ago`;
    if (diffInMinutes < 1440) return `${Math.floor(diffInMinutes / 60)}h ago`;
    return `${Math.floor(diffInMinutes / 1440)}d ago`;
  },

  // User initials generator
  getUserInitials: (name: string, email: string) => {
    return name 
      ? name.split(" ").map((n) => n[0]).join("").toUpperCase()
      : email.substring(0, 2).toUpperCase();
  },
};

// User table action handlers interface
export interface UserTableActions {
  onDelete: (userId: string) => void;
  // Admin actions
  onActivateUser: (userId: string, active: boolean) => void;
  onResetPassword: (userId: string, email: string) => void;
  onChangePassword: (userId: string, email: string) => void;
  onAssignRole?: (userId: string, userName: string, userEmail: string) => void;
}

// Reusable user cell component
export function UserCell({
  user,
  onToggleExpand,
  isExpanded
}: {
  user: EnhancedUser;
  onToggleExpand?: () => void;
  isExpanded?: boolean;
}) {
  return (
    <div className="flex items-center gap-2 sm:gap-3 min-w-0">
      <Avatar className="h-8 w-8 flex-shrink-0">
        <AvatarImage src={user.avatar || (user as any).avatar_url || ""} alt={user.name} />
        <AvatarFallback>
          {UserTableUtils.getUserInitials(user.name, user.email)}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 min-w-0 overflow-hidden">
        <div
          className={`font-medium truncate transition-colors ${
            onToggleExpand
              ? "text-blue-600 hover:text-blue-800 cursor-pointer hover:underline"
              : ""
          }`}
          title={user.name}
          onClick={onToggleExpand ? (e) => {
            e.stopPropagation();
            onToggleExpand();
          } : undefined}
        >
          {user.name}
        </div>
        <div className="text-sm text-foreground truncate" title={user.email}>
          {user.email}
        </div>
      </div>
    </div>
  );
}

// Simplified user cell with email copy for admin view
export function SimpleUserCell({ user }: { user: EnhancedUser }) {
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

// Simple client ID cell
export function ClientIdCell({ user }: { user: EnhancedUser }) {
  const clientId = (user as any).client_id;
  if (!clientId) return <span className="text-sm text-foreground">—</span>;

  return (
    <div className="flex items-center gap-2">
      <span className="text-sm font-mono truncate" title={clientId}>
        {clientId.length > 16 ? `${clientId.substring(0,8)}...${clientId.substring(clientId.length - 4)}` : clientId}
      </span>
      <CopyButton
        text={clientId}
        label="Client ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Simple status cell
export function SimpleStatusCell({ user }: { user: EnhancedUser }) {
  const isActive = user.active !== undefined ? user.active : (user.status === 'active');
  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
      isActive ? "text-emerald-600 dark:text-emerald-400" : "text-foreground/60"
    }`}>
      <span className={`h-1.5 w-1.5 rounded-full ${isActive ? "bg-emerald-500" : "bg-foreground/40"}`} />
      {isActive ? "Active" : "Inactive"}
    </span>
  );
}

// Simple provider cell
export function SimpleProviderCell({ user }: { user: EnhancedUser }) {
  const provider = user.provider || 'local';
  return (
    <span className="text-xs font-medium text-foreground">
      {UserTableUtils.formatProvider(provider)}
    </span>
  );
}

// Reusable roles cell component
export function RolesCell({ user }: { user: EnhancedUser }) {
  return (
    <div className="flex flex-wrap gap-1">
      {user.roles && user.roles.length > 0 ? (
        <span className="text-xs font-medium text-foreground">
          {user.roles.slice(0, 2).map((role: any, index) => (
            <span key={index}>
              {typeof role === 'string' ? role : role.name || 'Role'}
              {index < Math.min(user.roles.length, 2) - 1 && <span className="text-foreground/40">, </span>}
            </span>
          ))}
          {user.roles.length > 2 && (
            <span className="text-foreground/50"> +{user.roles.length - 2}</span>
          )}
        </span>
      ) : (
        <span className="text-xs text-foreground/50">No roles</span>
      )}
    </div>
  );
}

// Reusable status cell component
export function StatusCell({ user }: { user: EnhancedUser }) {
  const isActive = user.active !== undefined ? user.active : (user.status === 'active');
  return (
    <div className="space-y-1">
      <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
        isActive ? "text-emerald-600 dark:text-emerald-400" : "text-foreground/60"
      }`}>
        <span className={`h-1.5 w-1.5 rounded-full ${isActive ? "bg-emerald-500" : "bg-foreground/40"}`} />
        {isActive ? "Active" : "Inactive"}
      </span>
      {user.MFAEnabled && (
        <div className="flex items-center gap-1">
          <Shield className="h-3 w-3 text-emerald-600" />
          <span className="text-xs text-emerald-600 dark:text-emerald-400 font-medium">MFA</span>
        </div>
      )}
    </div>
  );
}

// Reusable activity cell component
export function ActivityCell({ user }: { user: EnhancedUser }) {
  return (
    <div className="space-y-1">
      <div className="flex items-center gap-1 text-sm">
        <Calendar className="h-3 w-3 text-foreground" />
        <span className="text-foreground">
          {UserTableUtils.formatLastLogin(user.last_login || user.lastLogin)}
        </span>
      </div>
      <div className="text-sm text-foreground">
        Provider: {UserTableUtils.formatProvider(user.provider || 'local')}
      </div>
    </div>
  );
}

// Reusable permissions cell component
export function PermissionsCell({ user }: { user: EnhancedUser }) {
  // Convert ApiUser to EnhancedUser format for the modal
  const enhancedUser = {
    ...user,
    status: user.active ? 'active' as const : 'inactive' as const,
    directRoleIds: [],
    groupIds: [],
    effectiveRoleIds: [],
    lastLogin: user.last_login,
    loginCount: 0,
    groupNames: [],
    roleNames: user.roles?.map((r: any) => typeof r === 'string' ? r : r.name) || [],
    isOrphan: false,
    createdAt: user.created_at,
    updatedAt: user.updated_at
  };
  
  return (
    <Button
      variant="ghost"
      size="sm"
      className="h-8 text-sm text-foreground hover:text-foreground"
      onClick={() => {
        // Permissions matrix functionality removed
        console.log('Permissions matrix for user:', enhancedUser);
      }}
    >
      <Shield className="h-3 w-3 mr-1" />
      View Matrix
    </Button>
  );
}

// Reusable actions cell component
export function ActionsCell({
  user,
  actions,
  onToggleExpand,
  isExpanded,
}: {
  user: EnhancedUser;
  actions: UserTableActions;
  onToggleExpand?: () => void;
  isExpanded?: boolean;
}) {
  return (
    <div className="flex items-center justify-end gap-1">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" visualVariant="row-actions">
          {/* Admin Actions */}
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

      {onToggleExpand && (
        <Button
          variant="ghost"
          size="sm"
          onClick={(e) => {
            e.stopPropagation();
            onToggleExpand();
          }}
          className="h-8 w-8 p-0 opacity-70 hover:opacity-100 transition-opacity duration-200"
        >
          {isExpanded ? (
            <Calendar className="h-4 w-4" />
          ) : (
            <Users className="h-4 w-4" />
          )}
        </Button>
      )}
    </div>
  );
}

// Expanded row content component with dynamic fields based on user source
export function UserExpandedRow({ user }: { user: any }) {
  const hasProviderData = user.provider_data && typeof user.provider_data === 'object';
  const hasMfaInfo = user.MFAEnabled || user.MFAMethod?.length > 0;
  const isDirectoryUser = user.is_synced_user;
  const hasExtendedInfo = user.username || user.tenant_domain;

  // Helper function to format MFA methods
  const formatMfaMethods = (methods: string[] | null) => {
    if (!methods || !Array.isArray(methods)) return "None";
    return methods.map(method => method.toUpperCase()).join(", ");
  };

  // Helper function to format provider data
  const renderProviderData = () => {
    if (!hasProviderData) return null;
    
    const data = user.provider_data;
    const commonFields = [];
    
    // Handle different provider types
    if (user.provider === 'google' || user.provider === 'github') {
      if (data.name && data.name !== user.name) commonFields.push({ label: "Provider Name", value: data.name });
      if (data.picture || data.avatar_url) commonFields.push({ label: "Avatar", value: "Available" });
      if (data.verified_email !== undefined) commonFields.push({ label: "Email Verified", value: data.verified_email ? "Yes" : "No" });
    }
    
    if (user.provider === 'ad_sync' || user.provider === 'entra_id') {
      if (data.attributes?.cn) commonFields.push({ label: "CN", value: data.attributes.cn });
      if (data.attributes?.sAMAccountName) commonFields.push({ label: "SAM Account", value: data.attributes.sAMAccountName });
      if (data.attributes?.userPrincipalName) commonFields.push({ label: "UPN", value: data.attributes.userPrincipalName });
    }
    
    return commonFields;
  };

  return (
    <div className="space-y-5">
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
        {/* Basic User Details */}
        <div className="space-y-3">
          <h4 className="font-semibold flex items-center gap-2 text-foreground text-sm uppercase tracking-wide">
            <Users className="h-4 w-4 text-blue-600" />
            User Details
          </h4>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between items-center py-1 gap-2">
              <span className="text-foreground text-xs font-medium">User ID:</span>
              <div className="flex items-center gap-2">
                <span
                  className="text-xs font-mono bg-black/5 dark:bg-white/10 px-2 py-1 rounded max-w-[180px] truncate"
                  title={user.id}
                >
                  {formatIdentifier(user.id)}
                </span>
                {user.id && (
                  <CopyButton
                    text={user.id}
                    label="User ID"
                    variant="ghost"
                    size="sm"
                    className="h-6 px-2"
                  />
                )}
              </div>
            </div>
            {hasExtendedInfo && user.username && (
              <div className="flex justify-between items-center py-1">
                <span className="text-foreground text-xs font-medium">Username:</span>
                <span className="text-xs font-medium">{user.username}</span>
              </div>
            )}
            <div className="flex justify-between items-center py-1">
              <span className="text-foreground text-xs font-medium">Status:</span>
              <Badge variant={user.active ? "default" : "secondary"} className="text-xs h-6">
                {user.active ? "Active" : "Inactive"}
              </Badge>
            </div>
            {user.tenant_domain && (
              <div className="flex justify-between items-center py-1">
                <span className="text-foreground text-xs font-medium">Domain:</span>
                <span className="text-xs">{user.tenant_domain}</span>
              </div>
            )}
            <div className="flex justify-between items-center py-1">
              <span className="text-foreground text-xs font-medium">Source:</span>
              <Badge variant={isDirectoryUser ? "outline" : "secondary"} className="text-xs h-6">
                {isDirectoryUser ? "Directory Sync" : "Manual"}
              </Badge>
            </div>
          </div>
        </div>

        {/* Authentication & Security */}
        <div className="space-y-3">
          <h4 className="font-semibold flex items-center gap-2 text-foreground text-sm uppercase tracking-wide">
            <Shield className="h-4 w-4 text-green-600" />
            Security
          </h4>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between items-center py-1">
              <span className="text-foreground text-xs font-medium">Provider:</span>
              <Badge variant="outline" className="text-xs h-6 capitalize">
                {user.provider || "Unknown"}
              </Badge>
            </div>
            <div className="flex justify-between items-center py-1">
              <span className="text-foreground text-xs font-medium">MFA Enabled:</span>
              <Badge variant={user.MFAEnabled ? "default" : "destructive"} className="text-xs h-6">
                {user.MFAEnabled ? "Yes" : "No"}
              </Badge>
            </div>
            {hasMfaInfo && (
              <>
                <div className="flex justify-between items-center py-1">
                  <span className="text-foreground text-xs font-medium">MFA Methods:</span>
                  <span className="text-xs font-medium">{formatMfaMethods(user.MFAMethod)}</span>
                </div>
                {user.MFADefaultMethod && (
                  <div className="flex justify-between items-center py-1">
                    <span className="text-foreground text-xs font-medium">Default MFA:</span>
                    <Badge variant="secondary" className="text-xs h-6 uppercase">
                      {user.MFADefaultMethod}
                    </Badge>
                  </div>
                )}
                <div className="flex justify-between items-center py-1">
                  <span className="text-foreground text-xs font-medium">MFA Verified:</span>
                  <Badge variant={user.mfa_verified ? "default" : "secondary"} className="text-xs h-6">
                    {user.mfa_verified ? "Yes" : "No"}
                  </Badge>
                </div>
              </>
            )}
          </div>
        </div>

        {/* Activity & Timestamps */}
        <div className="space-y-3">
          <h4 className="font-semibold flex items-center gap-2 text-foreground text-sm uppercase tracking-wide">
            <Calendar className="h-4 w-4 text-orange-600" />
            Activity
          </h4>
          <div className="space-y-2 text-sm">
            <div className="flex justify-between items-center py-1">
              <span className="text-foreground text-xs font-medium">Updated:</span>
              <span className="text-xs">
                {new Date(user.updated_at).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric'
                })}
              </span>
            </div>
            <div className="flex justify-between items-center py-1">
              <span className="text-foreground text-xs font-medium">Last Login:</span>
              <span className="text-xs">
                {user.last_login ? new Date(user.last_login).toLocaleDateString('en-US', {
                  month: 'short',
                  day: 'numeric',
                  year: 'numeric',
                  hour: '2-digit',
                  minute: '2-digit'
                }) : "Never"}
              </span>
            </div>
            {user.MFAEnrolledAt && user.MFAEnrolledAt !== "0001-01-01T00:00:00Z" && (
              <div className="flex justify-between items-center py-1">
                <span className="text-foreground text-xs font-medium">MFA Enrolled:</span>
                <span className="text-xs">
                  {new Date(user.MFAEnrolledAt).toLocaleDateString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    year: 'numeric'
                  })}
                </span>
              </div>
            )}
            {isDirectoryUser && user.last_sync_at && (
              <div className="flex justify-between items-center py-1">
                <span className="text-foreground text-xs font-medium">Last Sync:</span>
                <span className="text-xs">
                  {new Date(user.last_sync_at).toLocaleDateString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    year: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit'
                  })}
                </span>
              </div>
            )}
          </div>
        </div>

        {/* Provider-Specific Data */}
        {hasProviderData && (
          <div className="space-y-3">
            <h4 className="font-semibold flex items-center gap-2 text-foreground text-sm uppercase tracking-wide">
              <Settings className="h-4 w-4 text-blue-600" />
              Provider Data
            </h4>
            <div className="space-y-2 text-sm">
              {renderProviderData()?.map((field, index) => (
                <div key={index} className="flex justify-between items-center py-1">
                  <span className="text-foreground text-xs font-medium">{field.label}:</span>
                  <span className="text-xs max-w-32 truncate" title={field.value}>{field.value}</span>
                </div>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Roles Section */}
      {((user.roles && user.roles.length > 0) || (user.scopes && user.scopes.length > 0)) && (
        <div className="border-t border-border pt-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Roles */}
            {user.roles && user.roles.length > 0 && (
              <div className="space-y-2">
                <h5 className="font-medium text-foreground text-xs uppercase tracking-wide flex items-center gap-1">
                  <Key className="h-3 w-3" />
                  Roles ({user.roles.length})
                </h5>
                <div className="flex flex-wrap gap-1">
                  {user.roles.slice(0, 4).map((role: any, index: number) => (
                    <Badge key={index} variant="outline" className="text-xs h-5">
                      {role.name}
                    </Badge>
                  ))}
                  {user.roles.length > 4 && (
                    <Badge variant="outline" className="text-xs h-5">+{user.roles.length - 4}</Badge>
                  )}
                </div>
              </div>
            )}

            {/* Scopes */}
            {user.scopes && user.scopes.length > 0 && (
              <div className="space-y-2">
                <h5 className="font-medium text-foreground text-xs uppercase tracking-wide flex items-center gap-1">
                  <Shield className="h-3 w-3" />
                  Scopes ({user.scopes.length})
                </h5>
                <div className="flex flex-wrap gap-1">
                  {user.scopes.slice(0, 3).map((scope: any, index: number) => (
                    <Badge key={index} variant="default" className="text-xs h-5">
                      {scope.name}
                    </Badge>
                  ))}
                  {user.scopes.length > 3 && (
                    <Badge variant="default" className="text-xs h-5">+{user.scopes.length - 3}</Badge>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>
      )}


    </div>
  );
}

// Column definitions factory (legacy for backwards compatibility)
export function createUserTableColumns(
  actions: UserTableActions,
  expandedRows?: Set<string>,
  onToggleExpand?: (rowId: string) => void,
  getRowId?: (row: EnhancedUser) => string
): ResponsiveColumnDef<EnhancedUser, any>[] {
  return [
    {
      id: "user",
      accessorKey: "name",
      header: "User",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => {
        const rowId = getRowId ? getRowId(row.original) : row.original.id;
        const isExpanded = expandedRows?.has(rowId) || false;
        return (
          <UserCell 
            user={row.original} 
            onToggleExpand={onToggleExpand ? () => onToggleExpand(rowId) : undefined}
            isExpanded={isExpanded}
          />
        );
      },
      cellClassName: "max-w-0",
    },
    {
      id: "roleTeam",
      accessorKey: "roles",
      header: "Roles",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <RolesCell user={row.original} />,
    },
    {
      id: "status",
      accessorKey: "active",
      header: "Status",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <StatusCell user={row.original} />,
    },
    {
      id: "activity",
      accessorKey: "last_login",
      header: "Activity",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <ActivityCell user={row.original} />,
    },
    {
      id: "permissions",
      header: "Permissions",
      resizable: true,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => <PermissionsCell user={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => <ActionsCell user={row.original} actions={actions} />,
      cellClassName: "text-center",
    },
  ];
}

// Dynamic column definitions factory
export function createDynamicUserTableColumns(
  visibleColumns: string[],
  actions: UserTableActions,
  expandedRows?: Set<string>,
  onToggleExpand?: (rowId: string) => void,
  getRowId?: (row: EnhancedUser) => string
): ResponsiveColumnDef<EnhancedUser, any>[] {
  const availableColumns: Record<string, ResponsiveColumnDef<EnhancedUser, any>> = {
    user: {
      id: "user",
      accessorKey: "name",
      header: "User",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => {
        const rowId = getRowId ? getRowId(row.original) : row.original.id;
        const isExpanded = expandedRows?.has(rowId) || false;
        return (
          <UserCell 
            user={row.original} 
            onToggleExpand={onToggleExpand ? () => onToggleExpand(rowId) : undefined}
            isExpanded={isExpanded}
          />
        );
      },
    },
    status: {
      id: "status",
      accessorKey: "active",
      header: "Status",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <StatusCell user={row.original} />,
    },
    roleTeam: {
      id: "roleTeam",
      accessorKey: "roles",
      header: "Roles",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <RolesCell user={row.original} />,
    },
    lastLogin: {
      id: "lastLogin",
      accessorKey: "last_login",
      header: "Last Login",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <ActivityCell user={row.original} />,
    },
    loginCount: {
      id: "loginCount",
      accessorKey: "roles",
      header: "Login Count",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <div className="flex items-center gap-1 text-sm">
          <span>{row.original.loginCount || 0}</span>
        </div>
      ),
    },
    syncSource: {
      id: "syncSource",
      accessorKey: "sync_source",
      header: "Sync Source",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <Badge variant="outline" className="text-xs">
          {row.original.sync_source || "manual"}
        </Badge>
      ),
    },
    provider: {
      id: "provider",
      accessorKey: "provider",
      header: "Auth Provider",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <Badge variant="outline" className="text-xs">
          {row.original.provider || "email"}
        </Badge>
      ),
    },
    mfaEnabled: {
      id: "mfaEnabled",
      accessorKey: "MFAEnabled",
      header: "MFA Status",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <Badge variant={row.original.MFAEnabled ? "default" : "outline"} className="text-xs">
          {row.original.MFAEnabled ? "Enabled" : "Disabled"}
        </Badge>
      ),
    },
    tenantDomain: {
      id: "tenantDomain",
      accessorKey: "tenant_domain",
      header: "Tenant Domain",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm font-mono">
          {row.original.tenant_domain || "—"}
        </span>
      ),
    },
    providerId: {
      id: "providerId",
      accessorKey: "provider_id",
      header: "Provider ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm font-mono">
          {row.original.provider_id ? 
            (row.original.provider_id.length > 20 ? 
              `${row.original.provider_id.substring(0, 20)}...` : 
              row.original.provider_id) : 
            "—"
          }
        </span>
      ),
    },
    updatedAt: {
      id: "updatedAt",
      accessorKey: "updated_at",
      header: "Last Updated",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <div className="text-sm text-foreground">
          {new Date(row.original.updated_at).toLocaleDateString()}
        </div>
      ),
    },
    mfaVerified: {
      id: "mfaVerified",
      accessorKey: "mfa_verified",
      header: "MFA Verified",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <Badge variant={row.original.mfa_verified ? "default" : "outline"} className="text-xs">
          {row.original.mfa_verified ? "Verified" : "Not Verified"}
        </Badge>
      ),
    },
    clientId: {
      id: "clientId",
      accessorKey: "client_id",
      header: "Client ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm font-mono">
          {row.original.client_id ? 
            `${row.original.client_id.substring(0, 8)}...` : 
            "—"
          }
        </span>
      ),
    },
    projectId: {
      id: "projectId",
      accessorKey: "project_id",
      header: "Project ID",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <span className="text-sm font-mono">
          {row.original.project_id ? 
            `${row.original.project_id.substring(0, 8)}...` : 
            "—"
          }
        </span>
      ),
    },
    scopesCount: {
      id: "scopesCount",
      accessorKey: "scopes",
      header: "Scopes",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <div className="flex items-center gap-1">
          <Shield className="h-3 w-3 text-foreground" />
          <span className="text-sm">{row.original.scopes?.length || 0}</span>
        </div>
      ),
    },
    resourcesCount: {
      id: "resourcesCount",
      accessorKey: "resources",
      header: "Resources",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => (
        <div className="flex items-center gap-1">
          <Key className="h-3 w-3 text-foreground" />
          <span className="text-sm">{row.original.resources?.length || 0}</span>
        </div>
      ),
    },
    permissions: {
      id: "permissions",
      header: "Permissions",
      resizable: true,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => <PermissionsCell user={row.original} />,
    },
    actions: {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => <ActionsCell user={row.original} actions={actions} />,
      cellClassName: "text-center",
    },
  };

  return visibleColumns
    .map(columnId => availableColumns[columnId])
    .filter(Boolean);
}

// Available columns for dynamic table configuration
export const AVAILABLE_USER_COLUMNS = {
  // Core user info
  user: { label: "User", description: "Name and email" },
  status: { label: "Status", description: "Active/Inactive status" },
  provider: { label: "Auth Provider", description: "Authentication provider" },
  
  // Roles and permissions
  roleTeam: { label: "Roles", description: "User roles" },
  scopesCount: { label: "Scopes", description: "Number of scopes assigned" },
  resourcesCount: { label: "Resources", description: "Number of resources accessible" },
  
  // Security and MFA
  mfaEnabled: { label: "MFA Status", description: "Multi-factor authentication status" },
  mfaVerified: { label: "MFA Verified", description: "Whether MFA is verified" },
  
  // Activity and timestamps
  lastLogin: { label: "Last Login", description: "Last login time and activity" },
  updatedAt: { label: "Last Updated", description: "Last modification date" },
  
  // Technical identifiers
  clientId: { label: "Client ID", description: "Associated client identifier" },
  projectId: { label: "Project ID", description: "Associated project identifier" },
  providerId: { label: "Provider ID", description: "External provider identifier" },
  tenantDomain: { label: "Tenant Domain", description: "Tenant domain information" },
  syncSource: { label: "Sync Source", description: "Directory sync source" },
  
  // Actions
  permissions: { label: "Permissions", description: "View permission matrix" },
  actions: { label: "Actions", description: "User management actions" },
} as const;

// Default visible columns for the user table
export const DEFAULT_USER_COLUMNS = [
  "user",
  "status", 
  "roleTeam",
  "provider",
  "mfaEnabled",
  "lastLogin",
  "actions"
] as const;

// All available column keys
export const ALL_USER_COLUMN_KEYS = Object.keys(AVAILABLE_USER_COLUMNS) as Array<keyof typeof AVAILABLE_USER_COLUMNS>;

// Helper function to get column metadata
export function getUserColumnMetadata(columnId: string) {
  return AVAILABLE_USER_COLUMNS[columnId as keyof typeof AVAILABLE_USER_COLUMNS];
}
