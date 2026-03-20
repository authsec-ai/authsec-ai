import * as React from "react";
import type { Row } from "@tanstack/react-table";
import type { AdminGroup as Group } from "@/app/api/admin/groupsApi";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Edit,
  Trash2,
  User,
  Search,
  Loader2,
  Users as UsersIcon,
  Hash,
} from "lucide-react";
import { GroupTableUtils, type GroupTableActions } from "../utils/group-table-utils";
import { useGetGroupUsersQuery } from "@/app/api/enduser/groupsApi";
import { SessionManager } from "@/utils/sessionManager";
import { CopyButton } from "@/components/ui/copy-button";

interface GroupsTableProps {
  groups: Group[];
  selectedGroupIds: string[];
  onSelectionChange: (selectedIds: string[]) => void;
  onSelectAll: () => void;
  actions: GroupTableActions;
}

// Group Name Cell Component
function GroupNameCell({ group }: { group: Group }) {
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
        <div className="font-medium truncate text-foreground" title={group.name}>
          {group.name}
        </div>
        {group.description && (
          <div className="text-sm text-foreground truncate" title={group.description}>
            {group.description}
          </div>
        )}
      </div>
    </div>
  );
}

// Group ID Cell Component
function GroupIdCell({ group }: { group: Group }) {
  return (
    <div className="flex items-center gap-2 min-w-0">
      <span className="text-sm font-mono truncate" title={group.id}>
        {group.id.substring(0, 8)}...
      </span>
      <CopyButton
        text={group.id}
        label="Group ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Group Expanded Row Component
function GroupExpandedRow({ group }: { group: Group }) {
  const [searchQuery, setSearchQuery] = React.useState("");
  const [currentPage, setCurrentPage] = React.useState(1);
  const membersPerPage = 10;

  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  const { data: groupUsers = [], isLoading, error } = useGetGroupUsersQuery(
    { tenant_id: tenantId || '', group_id: group.id },
    { skip: !tenantId }
  );

  // Filter members based on search
  const filteredMembers = React.useMemo(() => {
    const users = Array.isArray(groupUsers) ? groupUsers : [];
    if (!searchQuery.trim()) return users;
    const query = searchQuery.toLowerCase();
    return users.filter((user) => {
      const name = user.first_name && user.last_name
        ? `${user.first_name} ${user.last_name}`
        : user.username || user.email || '';
      return (
        name.toLowerCase().includes(query) ||
        user.email?.toLowerCase().includes(query)
      );
    });
  }, [groupUsers, searchQuery]);

  // Paginate filtered members
  const paginatedMembers = React.useMemo(() => {
    if (!Array.isArray(filteredMembers)) return [];
    const startIndex = (currentPage - 1) * membersPerPage;
    return filteredMembers.slice(startIndex, startIndex + membersPerPage);
  }, [filteredMembers, currentPage]);

  const totalPages = Math.ceil((filteredMembers?.length || 0) / membersPerPage);

  // Reset page when search changes
  React.useEffect(() => {
    setCurrentPage(1);
  }, [searchQuery]);

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
    <div className="space-y-6 rounded-lg bg-black/5 dark:bg-white/5 p-6">
      <div className="grid gap-6 md:grid-cols-2">
        {/* Left Column: Group Details */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <UsersIcon className="h-4 w-4" />
            Group Details
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Group Name" value={group.name} />
            <InfoLine label="Group ID" value={group.id} copyable />
            <InfoLine label="Description" value={group.description || "No description"} />
            <div className="flex items-center justify-between gap-3">
              <span className="text-xs font-medium text-foreground">Total Members</span>
              <span className="text-base font-semibold text-foreground">{filteredMembers.length.toLocaleString()}</span>
            </div>
          </div>
        </div>

        {/* Right Column: Members List Section */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <User className="h-4 w-4" />
            Members ({filteredMembers.length})
          </h4>

          {/* Search Members */}
          {Array.isArray(groupUsers) && groupUsers.length > 0 && (
            <div className="relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground" />
              <Input
                placeholder="Search members..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10 h-9"
              />
            </div>
          )}

          {/* Members List */}
          {isLoading ? (
            <div className="flex items-center gap-2 text-foreground py-4">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="text-sm">Loading members...</span>
            </div>
          ) : error ? (
            <p className="text-destructive text-sm py-4">Failed to load group members</p>
          ) : paginatedMembers.length > 0 ? (
            <div className="space-y-3">
              <div className="space-y-2">
                {paginatedMembers.map((user) => (
                  <div
                    key={user.id}
                    className="flex items-center gap-2 py-2 px-3 rounded-md bg-muted/50 hover:bg-muted"
                  >
                    <div className="h-8 w-8 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center flex-shrink-0">
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
                      {(user.first_name || user.last_name) && user.email && (
                        <div className="text-xs text-foreground truncate">{user.email}</div>
                      )}
                    </div>
                    <Badge variant="secondary" className="text-xs flex-shrink-0">Member</Badge>
                  </div>
                ))}
              </div>

              {/* Pagination */}
              {totalPages > 1 && (
                <div className="flex items-center justify-between pt-2 border-t">
                  <p className="text-xs text-foreground">
                    Showing {((currentPage - 1) * membersPerPage) + 1} to{' '}
                    {Math.min(currentPage * membersPerPage, filteredMembers.length)} of{' '}
                    {filteredMembers.length} members
                  </p>
                  <div className="flex items-center gap-1">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(p => Math.max(1, p - 1))}
                      disabled={currentPage === 1}
                      className="h-8"
                    >
                      Previous
                    </Button>
                    <span className="text-sm text-foreground px-2">
                      Page {currentPage} of {totalPages}
                    </span>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() => setCurrentPage(p => Math.min(totalPages, p + 1))}
                      disabled={currentPage === totalPages}
                      className="h-8"
                    >
                      Next
                    </Button>
                  </div>
                </div>
              )}
            </div>
          ) : searchQuery.trim() ? (
            <p className="text-foreground text-sm py-4">No members found matching "{searchQuery}"</p>
          ) : (
            <p className="text-foreground text-sm py-4">No members in this group</p>
          )}
        </div>
      </div>
    </div>
  );
}

// Actions Cell Component
function ActionsCell({ group, actions }: { group: Group; actions: GroupTableActions }) {
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
          <DropdownMenuItem onClick={() => actions.onEdit(group.id)}>
            <Edit className="mr-2 h-4 w-4" />
            Edit Group
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem
            onClick={() => actions.onDelete(group.id)}
            className="text-destructive focus:text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete Group
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export function GroupsTable({
  groups,
  selectedGroupIds,
  onSelectionChange,
  onSelectAll,
  actions,
}: GroupsTableProps) {
  const columns = React.useMemo<AdaptiveColumn<Group>[]>(
    () => [
      {
        id: "name",
        header: "Name",
        accessorKey: "name",
        alwaysVisible: true, // Always shows (mobile + desktop)
        enableSorting: true,
        resizable: true,
        approxWidth: 350,
        cell: ({ row }) => <GroupNameCell group={row.original} />,
      },
      {
        id: "groupId",
        header: "Group ID",
        accessorKey: "id",
        priority: 1, // Hides on small screens
        enableSorting: false,
        resizable: true,
        approxWidth: 200,
        cell: ({ row }) => <GroupIdCell group={row.original} />,
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
        cell: ({ row }) => <ActionsCell group={row.original} actions={actions} />,
      },
    ],
    [actions]
  );

  const renderExpandedRow = React.useCallback(
    (row: Row<Group>) => <GroupExpandedRow group={row.original} />,
    []
  );

  return (
    <AdaptiveTable
      tableId="groups"
      data={groups}
      columns={columns}
      enableSelection
      selectedRowIds={selectedGroupIds}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={onSelectAll}
      enableExpansion
      renderExpandedRow={renderExpandedRow}
      getRowId={(group) => group.id}
      enableSorting
      enablePagination
      pagination={{
        pageSize: 10,
        pageSizeOptions: [5, 10, 25, 50],
        alwaysVisible: true,
      }}
    />
  );
}
