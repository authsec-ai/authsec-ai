import * as React from "react";
import type { Row } from "@tanstack/react-table";
import type { EnhancedUser } from "../../../types/entities";
import {
  AdminEmailCell,
  AdminActionsCell,
  AdminUserExpandedRow,
  AdminInviteAcceptedCell,
  AdminClientIdCell,
  type AdminUserTableActions,
} from "../utils/admin-user-table-utils";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import { RolesCell } from "../utils/user-table-utils";

interface AdminUsersTableProps {
  users: EnhancedUser[];
  selectedUserIds: string[];
  onSelectionChange: (selectedIds: string[]) => void;
  onSelectAll: () => void;
  actions: AdminUserTableActions;
}

export function AdminUsersTable({
  users,
  selectedUserIds,
  onSelectionChange,
  onSelectAll,
  actions,
}: AdminUsersTableProps) {
  const columns = React.useMemo<AdaptiveColumn<EnhancedUser>[]>(
    () => [
      {
        id: "email",
        header: "Email",
        accessorKey: "email",
        alwaysVisible: true,
        enableSorting: true,
        resizable: true,
        approxWidth: 260,
        cell: ({ row }) => <AdminEmailCell user={row.original} />,
      },
      {
        id: "client_id",
        header: "Client ID",
        accessorKey: "client_id",
        priority: 1,
        enableSorting: false,
        resizable: true,
        approxWidth: 220,
        cell: ({ row }) => <AdminClientIdCell user={row.original} />,
      },
      {
        id: "roles",
        header: "Roles",
        accessorKey: "roles",
        priority: 2,
        enableSorting: false,
        resizable: true,
        approxWidth: 220,
        cell: ({ row }) => <RolesCell user={row.original} />,
      },
      {
        id: "inviteAccepted",
        header: "Invite Accepted",
        accessorKey: "accepted_invite",
        priority: 3,
        enableSorting: false,
        resizable: true,
        approxWidth: 160,
        cell: ({ row }) => <AdminInviteAcceptedCell user={row.original} />,
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
        cell: ({ row }) => <AdminActionsCell user={row.original} actions={actions} />,
      },
    ],
    [actions]
  );

  const renderExpandedRow = React.useCallback(
    (row: Row<EnhancedUser>) => <AdminUserExpandedRow user={row.original} />,
    []
  );

  return (
    <AdaptiveTable
      tableId="admin-users"
      data={users}
      columns={columns}
      rowClassName={() => "[&_td]:py-3.5 [&_td]:px-4 [&_td]:align-middle"}
      enableSelection
      selectedRowIds={selectedUserIds}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={onSelectAll}
      enableExpansion={true}
      renderExpandedRow={renderExpandedRow}
      getRowId={(user) => user.id}
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
