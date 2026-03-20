import * as React from "react";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Edit,
  Copy,
  Trash2,
  Key,
  UserPlus,
  History,
  Lock,
  Code2,
} from "lucide-react";
import type { EnhancedRole } from "@/types/entities";
import type { ResponsiveColumnDef } from "@/components/ui/responsive-data-table";
import { toast } from "@/lib/toast";
import { Badge } from "@/components/ui/badge";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { ScrollArea } from "@/components/ui/scroll-area";

export interface RoleTableActions {
  onEdit: (id: string) => void;
  onDuplicate: (id: string) => void;
  onDelete: (id: string) => void;
  onAssignUsers: (id: string) => void;
  onEditPermissions: (id: string) => void;
  onViewVersionHistory: (id: string) => void;
  onViewSDK?: (role: EnhancedRole) => void;
}

// Utility functions
export class RoleTableUtils {
  static getRoleInitials(name: string): string {
    return name
      .split(" ")
      .map((word) => word.charAt(0).toUpperCase())
      .join("")
      .slice(0, 2);
  }

  static getPermissionCount(role: EnhancedRole): number {
    const rawCount =
      role.permissionCount ??
      (role as any)?.permissions_count ??
      (role as any)?.permissionsCount;

    if (rawCount !== undefined) {
      const parsed = Number(rawCount);
      if (Number.isFinite(parsed)) {
        return parsed;
      }
    }

    return Array.isArray(role.permissions) ? role.permissions.length : 0;
  }

  static getRoleLevel(roleName: string): string {
    const name = roleName.toLowerCase();
    if (name.includes('admin') || name.includes('super')) {
      return 'Admin';
    }
    if (name.includes('manager') || name.includes('lead')) {
      return 'Manager';
    }
    if (name.includes('user') || name.includes('member')) {
      return 'User';
    }
    return 'Custom';
  }
}

// Reusable role cell component
export function RoleNameCell({
  role,
  onToggleExpand,
  isExpanded: _isExpanded
}: {
  role: EnhancedRole;
  onToggleExpand?: () => void;
  isExpanded?: boolean;
}) {
  const level = RoleTableUtils.getRoleLevel(role.name);
  const permissionCount = RoleTableUtils.getPermissionCount(role);

  const handleToggle = onToggleExpand
    ? (event: React.MouseEvent) => {
        event.stopPropagation();
        onToggleExpand();
      }
    : undefined;

  return (
    <div className="min-w-0">
      <p
        className={`truncate text-sm font-medium ${
          onToggleExpand ? "cursor-pointer text-foreground hover:underline" : "text-foreground"
        }`}
        title={role.name}
        onClick={handleToggle}
      >
        {role.name}
      </p>
    </div>
  );
}

// Reusable permissions cell component
export function RolePermissionsCell({ role }: { role: EnhancedRole }) {
  const permissionCount = RoleTableUtils.getPermissionCount(role);

  if (!permissionCount) {
    return <p className="text-sm text-foreground">—</p>;
  }

  return (
    <p className="text-sm text-foreground">
      {permissionCount} {permissionCount === 1 ? "permission" : "permissions"}
    </p>
  );
}

// Reusable actions cell component
export function RoleActionsCell({
  role,
  actions,
}: {
  role: EnhancedRole;
  actions: RoleTableActions;
}) {
  return (
    <div className="flex items-center justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
          {actions.onViewSDK && (
            <DropdownMenuItem
              onClick={() => actions.onViewSDK?.(role)}
              className="admin-menu-item-sdk"
            >
              <Code2 className="mr-2 h-4 w-4" />
              View SDK Code
            </DropdownMenuItem>
          )}
          <DropdownMenuItem onClick={() => actions.onEdit(role.id)}>
            <Edit className="mr-2 h-4 w-4" />
            Edit Role
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => actions.onAssignUsers(role.id)}>
            <UserPlus className="mr-2 h-4 w-4" />
            Assign Users
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => actions.onDelete(role.id)}
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

// Enhanced expanded row content component
export function RoleExpandedRow({ role }: { role: EnhancedRole }) {
  const userNames = (role.usernames ?? []).map(String);
  const userIds = (role.userIds ?? []).map(String);
  const usersList = (userNames.length ? userNames : userIds).filter(Boolean);
  const usersAssignedRaw =
    role.users_assigned ?? role.userCount ?? (userNames.length || userIds.length);
  const usersAssigned = Number.isFinite(Number(usersAssignedRaw))
    ? Number(usersAssignedRaw)
    : 0;

  const [usersModalOpen, setUsersModalOpen] = React.useState(false);
  const [userSearch, setUserSearch] = React.useState("");
  const filteredUsers = React.useMemo(() => {
    if (!userSearch) return usersList;
    return usersList.filter((u) => u.toLowerCase().includes(userSearch.toLowerCase()));
  }, [usersList, userSearch]);

  const previewUsers = usersList.slice(0, 5);
  const extraCount = Math.max(usersList.length - previewUsers.length, 0);

  return (
    <div className="text-[12px]">
      <div className="grid gap-4 md:grid-cols-12">
        <div className="md:col-span-7 space-y-1">
          <p className="text-[11px] font-semibold text-foreground uppercase tracking-wide">
            Description
          </p>
          {role.description ? (
            <p className="text-[13px] leading-relaxed text-foreground">
              {role.description}
            </p>
          ) : (
            <p className="text-[12px] italic text-foreground">
              No description provided for this role.
            </p>
          )}
        </div>

        <div className="md:col-span-5 space-y-2">
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-[11px] font-semibold uppercase tracking-wide text-foreground">
                Assigned Users
              </p>
              <p className="text-foreground">
                {usersAssigned || usersList.length || 0} user
                {(usersAssigned || usersList.length || 0) === 1 ? "" : "s"}
              </p>
            </div>
            {usersList.length > 5 && (
              <Button variant="ghost" size="sm" className="h-7 px-2" onClick={() => setUsersModalOpen(true)}>
                View all
              </Button>
            )}
          </div>

          {usersList.length ? (
            <div className="space-y-1">
              <ul className="space-y-1 text-foreground">
                {previewUsers.map((user, idx) => (
                  <li key={`${user}-${idx}`} className="flex items-center gap-2">
                    <span className="h-1.5 w-1.5 rounded-full bg-foreground/40" />
                    <span className="truncate">{user}</span>
                  </li>
                ))}
              </ul>
              {extraCount > 0 && (
                <Button
                  variant="outline"
                  size="sm"
                  className="mt-2 w-full justify-center h-8 text-[12px]"
                  onClick={() => setUsersModalOpen(true)}
                >
                  View {extraCount} more
                </Button>
              )}
            </div>
          ) : (
            <p className="text-[12px] text-foreground">No users assigned</p>
          )}
        </div>
      </div>

      <Dialog open={usersModalOpen} onOpenChange={setUsersModalOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="text-xl">Users with this role</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <Input
              placeholder="Search users by name or ID"
              value={userSearch}
              onChange={(e) => setUserSearch(e.target.value)}
            />
            <ScrollArea className="h-64 rounded-md border border-border p-3">
              {filteredUsers.length ? (
                <div className="space-y-2">
                  {filteredUsers.map((user, idx) => (
                    <div
                      key={`${user}-${idx}`}
                      className="rounded-md bg-black/5 dark:bg-white/5 px-3 py-2 text-sm text-foreground"
                    >
                      {user}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-sm text-foreground">
                  No users match your search.
                </div>
              )}
            </ScrollArea>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  );
}

export function RoleTypeStatusCell({ role }: { role: EnhancedRole }) {
  const typeLabel =
    role.type === "system" ? "System" : role.type === "custom" ? "Custom" : "—";
  const editableLabel =
    role.isBuiltIn !== undefined ? (role.isBuiltIn ? "Built-in" : "Editable") : "—";
  const versionLabel = role.version !== undefined ? `v${role.version}` : "—";

  return (
    <div className="space-y-1 text-xs text-foreground">
      <p>{typeLabel}</p>
      <p>
        {editableLabel} · {versionLabel}
      </p>
    </div>
  );
}

export function RoleUserCountCell({ role }: { role: EnhancedRole }) {
  const hasUserData =
    role.userCount !== undefined ||
    role.users_assigned !== undefined ||
    role.userIds !== undefined ||
    role.usernames !== undefined;

  const usersRaw =
    role.userCount ??
    role.users_assigned ??
    role.usernames?.length ??
    role.userIds?.length ??
    0;

  const users = Number.isFinite(Number(usersRaw)) ? Number(usersRaw) : 0;

  if (!hasUserData) {
    return <div className="text-sm text-foreground">—</div>;
  }

  return (
    <div className="text-sm text-[color:var(--color-text-primary)]">
      <span className="font-semibold">{users}</span> users
    </div>
  );
}

export function RoleGroupCountCell({ role }: { role: EnhancedRole }) {
  const hasGroupData =
    role.groupCount !== undefined || role.groupIds !== undefined;

  const groups =
    role.groupCount ?? (Array.isArray(role.groupIds) ? role.groupIds.length : undefined);

  if (!hasGroupData || groups === undefined) {
    return <div className="text-sm text-foreground">—</div>;
  }

  return (
    <div className="text-sm text-[color:var(--color-text-primary)]">
      <span className="font-semibold">{groups}</span> groups
    </div>
  );
}

// Column definitions factory
export function createRoleTableColumns(
  actions: RoleTableActions,
  expandedRows?: Set<string>,
  onToggleExpand?: (rowId: string) => void,
  getRowId?: (row: EnhancedRole) => string
): ResponsiveColumnDef<EnhancedRole, any>[] {
  return [
    {
      id: "role",
      accessorKey: "name",
      header: "Role",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => {
        const rowId = getRowId ? getRowId(row.original) : row.original.id.toString();
        const isExpanded = expandedRows?.has(rowId) || false;
        return (
          <RoleNameCell
            role={row.original}
            onToggleExpand={onToggleExpand ? () => onToggleExpand(rowId) : undefined}
            isExpanded={isExpanded}
          />
        );
      },
      cellClassName: "max-w-0",
    },
    {
      id: "permissions",
      accessorKey: "permissions",
      header: "Permissions",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <RolePermissionsCell role={row.original} />,
    },
    {
      id: "usersCount",
      accessorKey: "userCount",
      header: "Users",
      resizable: true,
      responsive: true,
      cell: ({ row }) => <RoleUserCountCell role={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => (
        <RoleActionsCell role={row.original} actions={actions} />
      ),
      cellClassName: "text-center",
    },
  ];
}

// Legacy column definitions for backward compatibility
export function createSimpleRoleTableColumns(
  actions: RoleTableActions
): ResponsiveColumnDef<EnhancedRole, any>[] {
  return [
    {
      id: "role",
      accessorKey: "name",
      header: "Role Name",
      cell: ({ row }) => (
        <span className="block truncate font-medium" title={row.original.name}>
          {row.original.name}
        </span>
      ),
      minSize: 180,
    },
    {
      id: "permissions",
      accessorKey: "permissions",
      header: "Permissions",
      cell: ({ row }) => {
        const count = RoleTableUtils.getPermissionCount(row.original);
        return (
          <span className="text-sm text-foreground">
            {count} {count === 1 ? 'permission' : 'permissions'}
          </span>
        );
      },
      minSize: 120,
    },
    {
      id: "actions",
      header: "",
      cell: ({ row }) => (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
              <MoreHorizontal className="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" visualVariant="row-actions">
            <DropdownMenuItem onClick={() => actions.onDelete(row.original.id)}>
              <Trash2 className="mr-2 h-4 w-4" />
              Delete Role
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
      size: 60,
    },
  ];
}
