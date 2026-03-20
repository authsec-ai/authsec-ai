import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useGetGroupUsersQuery } from "@/app/api/enduser/groupsApi";
import { SessionManager } from "@/utils/sessionManager";
import { Loader2, User } from "lucide-react";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Users as UsersIcon,
  MoreHorizontal,
  Edit,
  Trash2,
} from "lucide-react";
import type { AdminGroup as Group } from "@/app/api/admin/groupsApi";
import type { ResponsiveColumnDef } from "@/components/ui/responsive-data-table";
import { toast } from "@/lib/toast";
import { Calendar, Hash } from "lucide-react";

export interface GroupTableActions {
  onEdit: (id: string) => void;
  onDelete: (id: string) => void;
}


// Reusable group cell component
export function GroupNameCell({
  group,
  onToggleExpand,
  isExpanded: _isExpanded
}: {
  group: Group;
  onToggleExpand?: () => void;
  isExpanded?: boolean;
}) {
  return (
    <div className="flex items-center gap-3 min-w-0">
      <div className="flex-shrink-0">
        <div className="h-10 w-10 rounded-lg bg-gradient-to-br from-blue-100 to-blue-200 dark:from-blue-900/50 dark:to-blue-800/50 flex items-center justify-center">
          <span className="text-sm font-semibold text-blue-700 dark:text-blue-300">
            {GroupTableUtils.getGroupInitials(group.name)}
          </span>
        </div>
      </div>
      <div className="flex-1 min-w-0 overflow-hidden">
        <div
          className={`font-medium truncate transition-colors ${
            onToggleExpand
              ? "text-blue-600 hover:text-blue-800 cursor-pointer hover:underline"
              : ""
          }`}
          title={group.name}
          onClick={onToggleExpand ? (e) => {
            e.stopPropagation();
            onToggleExpand();
          } : undefined}
        >
          {group.name}
        </div>
        <div className="text-sm text-foreground truncate" title={group.description}>
          {group.description || "No description"}
        </div>
      </div>
    </div>
  );
}

// Reusable group status cell component
export function GroupStatusCell({ group }: { group: Group }) {
  const isMember = (group as any)._isMember;
  const showMembership = typeof isMember === 'boolean';

  return (
    <div className="space-y-1">
      {showMembership ? (
        <Badge
          variant={isMember ? "default" : "outline"}
          className="text-xs"
        >
          {isMember ? "Member" : "Available"}
        </Badge>
      ) : (
        <Badge variant="default" className="text-xs">
          Active
        </Badge>
      )}
      <div className="flex items-center gap-1">
        <Hash className="h-3 w-3 text-foreground" />
        <span className="text-xs text-foreground font-mono">
          {group.id}
        </span>
      </div>
    </div>
  );
}

// Reusable actions cell component
export function GroupActionsCell({
  group,
  actions,
}: {
  group: Group;
  actions: GroupTableActions;
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
          <DropdownMenuItem onClick={() => actions.onEdit(group.id)}>
            <Edit className="mr-2 h-4 w-4" />
            Edit Group
          </DropdownMenuItem>
          <DropdownMenuItem
            onClick={() => actions.onDelete(group.id)}
            className="text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete Group
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// Enhanced expanded row content component
export function GroupExpandedRow({ group }: { group: Group }) {
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  const { data: groupUsers, isLoading, error } = useGetGroupUsersQuery(
    { tenant_id: tenantId || '', group_id: group.id },
    { skip: !tenantId }
  );

  return (
    <div className="p-4 text-sm space-y-4">
      <div>
        <h4 className="font-semibold text-foreground mb-1">Description</h4>
        <p className="text-foreground max-w-prose">{group.description || "No description"}</p>
      </div>

      <div>
        <h4 className="font-semibold text-foreground mb-2 flex items-center gap-2">
          <User className="h-4 w-4" />
          Group Members
        </h4>
        {isLoading ? (
          <div className="flex items-center gap-2 text-foreground">
            <Loader2 className="h-4 w-4 animate-spin" />
            <span>Loading members...</span>
          </div>
        ) : error ? (
          <p className="text-destructive text-sm">Failed to load group members</p>
        ) : groupUsers && groupUsers.length > 0 ? (
          <div className="space-y-2">
            {groupUsers.map((user) => (
              <div key={user.id} className="flex items-center gap-2 py-1">
                <div className="h-6 w-6 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                  <span className="text-xs font-medium text-blue-700 dark:text-blue-300">
                    {user.email?.[0]?.toUpperCase() || 'U'}
                  </span>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="text-sm font-medium truncate">
                    {user.first_name && user.last_name
                      ? `${user.first_name} ${user.last_name}`
                      : user.username || user.email}
                  </div>
                  {(user.first_name || user.last_name) && (
                    <div className="text-xs text-foreground truncate">{user.email}</div>
                  )}
                </div>
                <Badge variant="secondary" className="text-xs">Member</Badge>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-foreground text-sm">No members in this group</p>
        )}
      </div>
    </div>
  );
}

// Column definitions factory
export function createGroupTableColumns(
  actions: GroupTableActions,
  expandedRows?: Set<string>,
  onToggleExpand?: (rowId: string) => void,
  getRowId?: (row: Group) => string
): ResponsiveColumnDef<Group, any>[] {
  return [
    {
      id: "group",
      accessorKey: "name",
      header: "Group",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => {
        const rowId = getRowId ? getRowId(row.original) : row.original.id;
        const isExpanded = expandedRows?.has(rowId) || false;
        return (
          <GroupNameCell
            group={row.original}
            onToggleExpand={onToggleExpand ? () => onToggleExpand(rowId) : undefined}
            isExpanded={isExpanded}
          />
        );
      },
      cellClassName: "max-w-0",
    },
    {
      id: "status",
      accessorKey: "id",
      header: "Status",
      resizable: true,
      responsive: true,
      cell: ({ row }: { row: any }) => <GroupStatusCell group={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      resizable: false,
      responsive: false,
      enableSorting: false,
      cell: ({ row }: { row: any }) => {
        const rowId = getRowId ? getRowId(row.original) : row.original.id;
        return (
          <GroupActionsCell
            group={row.original}
            actions={actions}
          />
        );
      },
      cellClassName: "text-center",
    },
  ];
}

// Legacy column definitions for backward compatibility
export function createSimpleGroupTableColumns(
  actions: GroupTableActions
): ResponsiveColumnDef<Group, any>[] {
  return [
    {
      id: "group",
      accessorKey: "name",
      header: "Group Name",
      cell: ({ row }) => (
        <div className="flex items-center gap-2 min-w-0">
          <UsersIcon className="h-4 w-4 text-blue-600" />
          <span
            className="truncate font-medium"
            title={row.original.name}
          >
            {row.original.name}
          </span>
        </div>
      ),
      minSize: 180,
    },
    {
      id: "description",
      accessorKey: "description",
      header: "Description",
      cell: ({ row }) => (
        <span className="text-sm text-foreground truncate" title={row.original.description}>
          {row.original.description || "No description"}
        </span>
      ),
      minSize: 200,
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
              Delete Group
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ),
      size: 60,
    },
  ];
}

// Utility functions
export const GroupTableUtils = {
  getGroupInitials: (name: string): string => {
    return name
      .split(" ")
      .map(word => word[0])
      .join("")
      .toUpperCase()
      .substring(0, 2);
  }
};
