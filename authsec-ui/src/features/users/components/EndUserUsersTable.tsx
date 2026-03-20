import * as React from "react";
import type { Row } from "@tanstack/react-table";
import type { EnhancedUser } from "../../../types/entities";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import { Avatar, AvatarFallback, AvatarImage } from "../../../components/ui/avatar";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { CopyButton } from "../../../components/ui/copy-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  Calendar,
  MoreHorizontal,
  Trash2,
  Lock,
  UserCheck,
  UserX,
  Key,
  User,
  Shield,
} from "lucide-react";
import { UserTableUtils, type UserTableActions } from "../utils/user-table-utils";

const formatIdentifier = (value?: string | null) => {
  if (!value) return "";
  if (value.length <= 16) return value;
  return `${value.slice(0, 8)}…${value.slice(-4)}`;
};

interface EndUserUsersTableProps {
  users: EnhancedUser[];
  selectedUserIds: string[];
  onSelectionChange: (selectedIds: string[]) => void;
  onSelectAll: () => void;
  actions: UserTableActions;
}

// User Cell Component
function EndUserUserCell({ user }: { user: EnhancedUser }) {
  const hasDistinctName = Boolean(user.name && user.name !== user.email);
  const primaryLabel = hasDistinctName ? user.name : user.email;

  return (
    <div className="flex items-center gap-3 min-w-0">
      <Avatar className="h-9 w-9 flex-shrink-0">
        <AvatarImage src={user.avatar || (user as any).avatar_url || ""} alt={user.name} />
        <AvatarFallback className="text-xs font-medium">
          {UserTableUtils.getUserInitials(user.name, user.email)}
        </AvatarFallback>
      </Avatar>
      <div className="flex-1 min-w-0 overflow-hidden space-y-0.5">
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-medium truncate text-foreground text-sm" title={primaryLabel || undefined}>
            {primaryLabel || "Unknown user"}
          </span>
          {user.email && (
            <CopyButton
              text={user.email}
              label="Email"
              variant="ghost"
              size="sm"
              className="h-4 w-4 p-0 flex-shrink-0 opacity-0 group-hover:opacity-100 transition-opacity"
            />
          )}
        </div>
        {hasDistinctName && user.email && (
          <div className="text-xs text-muted-foreground truncate" title={user.email}>
            {user.email}
          </div>
        )}
      </div>
    </div>
  );
}

// Client ID Cell Component
function ClientIdCell({ user }: { user: EnhancedUser }) {
  const clientId = user.client_id;

  if (!clientId) {
    return <span className="text-xs text-muted-foreground">—</span>;
  }

  const formatClientId = (id: string) => {
    if (id.length <= 16) return id;
    return `${id.slice(0, 8)}…${id.slice(-4)}`;
  };

  return (
    <div className="flex items-center gap-2 min-w-0 group">
      <span className="text-xs font-mono truncate text-muted-foreground" title={clientId}>
        {formatClientId(clientId)}
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

// Roles & Teams Cell Component
function RolesTeamsCell({ user }: { user: EnhancedUser }) {
  const roleNames = (user.roles || []).map((role: any) =>
    typeof role === "string" ? role : role?.name || "Role"
  );
  const roleSummary = roleNames.length > 0
    ? `${roleNames.slice(0, 2).join(", ")}${roleNames.length > 2 ? ` +${roleNames.length - 2}` : ""}`
    : "No roles";

  return (
    <div className="text-xs text-muted-foreground truncate min-w-0">
      <span>{roleSummary}</span>
    </div>
  );
}

// Status Cell Component
function StatusCell({ user }: { user: EnhancedUser }) {
  const isActive = Boolean(user.active);

  return (
    <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
      isActive ? "text-emerald-600 dark:text-emerald-400" : "text-muted-foreground"
    }`}>
      <span className={`h-1.5 w-1.5 rounded-full ${isActive ? "bg-emerald-500" : "bg-muted-foreground/40"}`} />
      {isActive ? "Active" : "Inactive"}
    </span>
  );
}

// Activity Cell Component (Last Login + Provider)
function ActivityCell({ user }: { user: EnhancedUser }) {
  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground truncate min-w-0">
      <Calendar className="h-3 w-3 flex-shrink-0" />
      <span className="truncate">
        {UserTableUtils.formatLastLogin(user.last_login)}
        <span className="mx-1.5">•</span>
        {UserTableUtils.formatProvider(user.provider)}
      </span>
    </div>
  );
}

// Actions Cell Component
function ActionsCell({ user, actions }: { user: EnhancedUser; actions: UserTableActions }) {
  return (
    <div className="flex justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
            <MoreHorizontal className="h-4 w-4" />
            <span className="sr-only">Open menu</span>
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
          <DropdownMenuItem onClick={() => actions.onActivateUser(user.id, !user.active)}>
            {user.active ? (
              <>
                <UserX className="mr-2 h-4 w-4" />
                Deactivate
              </>
            ) : (
              <>
                <UserCheck className="mr-2 h-4 w-4" />
                Activate
              </>
            )}
          </DropdownMenuItem>
          {actions.onAssignRole && (
            <DropdownMenuItem onClick={() => actions.onAssignRole!(user.id, user.name || "", user.email)}>
              <Shield className="mr-2 h-4 w-4" />
              Assign Role
            </DropdownMenuItem>
          )}
          <DropdownMenuItem onClick={() => actions.onResetPassword(user.id, user.email)}>
            <Key className="mr-2 h-4 w-4" />
            Reset Password
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => actions.onChangePassword(user.id, user.email)}>
            <Lock className="mr-2 h-4 w-4" />
            Change Password
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => actions.onDelete(user.id)}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// User Expanded Row Component - Redesigned following ClientsPage pattern
function UserExpandedRow({ user }: { user: EnhancedUser }) {
  // InfoLine component for horizontal label-value pairs (like ClientsPage)
  const InfoLine = ({ label, value, copyable = false }: { label: string; value?: string | number | null; copyable?: boolean }) => {
    if (value === undefined || value === null || value === "") return null;
    return (
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-foreground">{label}</span>
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono text-xs truncate text-foreground" title={String(value)}>{String(value)}</span>
          {copyable && <CopyButton text={String(value)} label={label} size="sm" />}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-5">
      <div className="grid gap-6 md:grid-cols-2">
        {/* Left Column: User Details */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <User className="h-4 w-4" />
            User Details
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Name" value={user.name || "Unnamed user"} />
            <InfoLine label="Email" value={user.email} copyable />
            {user.id && <InfoLine label="User ID" value={user.id} copyable />}
            {user.client_id && <InfoLine label="Client ID" value={user.client_id} copyable />}
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-foreground">Status</span>
              <Badge variant={UserTableUtils.getStatusVariant(user.active)}>
                {user.active ? "Active" : "Inactive"}
              </Badge>
            </div>
            <InfoLine label="Provider" value={UserTableUtils.formatProvider(user.provider)} />
            {(user as any).MFAMethod && <InfoLine label="MFA Method" value={String((user as any).MFAMethod)} />}
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-foreground">Last Login</span>
              <span className="text-xs text-foreground">{UserTableUtils.formatLastLogin(user.last_login)}</span>
            </div>
          </div>
        </div>

        {/* Right Column: Access & Membership */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Shield className="h-4 w-4" />
            Access
          </h4>
          <div className="space-y-4 text-sm">
            {/* Roles Section */}
            <div>
              <span className="text-xs font-medium uppercase tracking-wide text-foreground">Roles ({user.roles?.length || 0})</span>
              <div className="mt-2 flex flex-wrap gap-2">
                {user.roles && user.roles.length > 0 ? (
                  user.roles.map((role: any, index) => (
                    <Badge key={index} variant="secondary" className="text-xs">
                      {typeof role === "string" ? role : role.name || "Role"}
                    </Badge>
                  ))
                ) : (
                  <span className="text-xs text-foreground">No roles assigned</span>
                )}
              </div>
            </div>

          </div>
        </div>
      </div>
    </div>
  );
}

export function EndUserUsersTable({
  users,
  selectedUserIds,
  onSelectionChange,
  onSelectAll,
  actions,
}: EndUserUsersTableProps) {
  const columns = React.useMemo<AdaptiveColumn<EnhancedUser>[]>(
    () => [
      {
        id: "user",
        header: "User",
        accessorKey: "name",
        alwaysVisible: true, // Always shows (mobile + desktop)
        enableSorting: true,
        resizable: true,
        approxWidth: 280,
        cell: ({ row }) => <EndUserUserCell user={row.original} />,
      },
      {
        id: "clientId",
        header: "Client ID",
        accessorKey: "client_id",
        priority: 1, // Hides first on small screens
        enableSorting: true,
        resizable: true,
        approxWidth: 220,
        cell: ({ row }) => <ClientIdCell user={row.original} />,
      },
      {
        id: "roles",
        header: "Roles & Teams",
        accessorKey: "roles",
        priority: 2, // Hides second on small screens
        enableSorting: false,
        resizable: true,
        approxWidth: 200,
        cell: ({ row }) => <RolesTeamsCell user={row.original} />,
      },
      {
        id: "status",
        header: "Status",
        accessorFn: (user) => (user.active ? 1 : 0),
        priority: 3, // Hides third on small screens
        enableSorting: true,
        resizable: true,
        approxWidth: 140,
        cell: ({ row }) => <StatusCell user={row.original} />,
      },
      {
        id: "activity",
        header: "Activity",
        accessorKey: "last_login",
        priority: 4, // Hides fourth on small screens
        enableSorting: true,
        resizable: true,
        approxWidth: 180,
        cell: ({ row }) => <ActivityCell user={row.original} />,
      },
      {
        id: "actions",
        header: "Actions",
        alwaysVisible: true, // Always shows (mobile + desktop)
        enableSorting: false,
        resizable: false,
        size: 80,
        className: "w-[80px] text-right",
        cellClassName: "text-right",
        approxWidth: 100,
        cell: ({ row }) => <ActionsCell user={row.original} actions={actions} />,
      },
    ],
    [actions]
  );

  const renderExpandedRow = React.useCallback(
    (row: Row<EnhancedUser>) => <UserExpandedRow user={row.original} />,
    []
  );

  return (
    <AdaptiveTable
      tableId="enduser-users"
      data={users}
      columns={columns}
      rowClassName={() => "[&_td]:py-3.5 [&_td]:px-4 [&_td]:align-middle"}
      enableSelection={true}
      selectedRowIds={selectedUserIds}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={onSelectAll}
      enableExpansion={true}
      renderExpandedRow={renderExpandedRow}
      getRowId={(user) => user.id}
      enableSorting={true}
      enablePagination={true}
      pagination={{
        pageSize: 10,
        pageSizeOptions: [5, 10, 25, 50],
        alwaysVisible: true,
      }}
    />
  );
}
