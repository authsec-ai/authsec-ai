import * as React from "react";
import type { Row } from "@tanstack/react-table";
import type { ApiOidcProvider, OidcProviderTableActions } from "../utils/oidc-provider-table-utils";
import {
  ProviderCell,
  StatusCell,
  ActionsCell,
  ProviderExpandedRow,
  OidcProviderTableUtils,
} from "../utils/oidc-provider-table-utils";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import { CopyButton } from "@/components/ui/copy-button";

interface OidcProvidersTableProps {
  providers: ApiOidcProvider[];
  selectedProviderIds: string[];
  onSelectionChange: (selectedIds: string[]) => void;
  onSelectAll: () => void;
  actions: OidcProviderTableActions;
}

export function OidcProvidersTable({
  providers,
  selectedProviderIds,
  onSelectionChange,
  onSelectAll,
  actions,
}: OidcProvidersTableProps) {
  const columns = React.useMemo<AdaptiveColumn<ApiOidcProvider>[]>(() => {
    return [
      {
        id: "provider",
        header: "Provider",
        accessorKey: "display_name",
        alwaysVisible: true,
        enableSorting: true,
        resizable: true,
        approxWidth: 260,
        cell: ({ row }) => <ProviderCell provider={row.original} />,
      },
      {
        id: "status",
        header: "Status",
        accessorFn: (provider) => (provider.is_active ? 1 : 0),
        priority: 1,
        enableSorting: true,
        resizable: true,
        approxWidth: 160,
        cell: ({ row }) => (
          <StatusCell provider={row.original} />
        ),
      },
      {
        id: "providerName",
        header: "Type",
        accessorKey: "provider_name",
        priority: 2,
        enableSorting: true,
        resizable: true,
        approxWidth: 160,
        cell: ({ row }) => (
          <span className="text-sm text-foreground">
            {OidcProviderTableUtils.formatProviderName(row.original.provider_name)}
          </span>
        ),
      },
      {
        id: "clientId",
        header: "Client ID",
        accessorKey: "client_id",
        priority: 3,
        enableSorting: true,
        resizable: true,
        approxWidth: 220,
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <span className="text-sm font-mono truncate" title={row.original.client_id}>
              {row.original.client_id}
            </span>
            <CopyButton text={row.original.client_id} label="Client ID" size="sm" variant="ghost" />
          </div>
        ),
      },
      {
        id: "callbackUrl",
        header: "Callback URL",
        accessorKey: "callback_url",
        priority: 4,
        enableSorting: true,
        resizable: true,
        approxWidth: 240,
        cell: ({ row }) => (
          <div className="flex items-center gap-2 min-w-0">
            <span className="truncate text-sm" title={row.original.callback_url}>
              {row.original.callback_url}
            </span>
            <CopyButton text={row.original.callback_url} label="Callback URL" size="sm" variant="ghost" />
          </div>
        ),
      },
      {
        id: "providerStatus",
        header: "Provider Status",
        accessorKey: "status",
        priority: 5,
        enableSorting: true,
        resizable: true,
        approxWidth: 160,
        cell: ({ row }) => (
          <span className="text-sm text-foreground capitalize">{row.original.status}</span>
        ),
      },
      {
        id: "actions",
        header: "Actions",
        alwaysVisible: true,
        enableSorting: false,
        resizable: false,
        size: 80,
        className: "w-[80px] text-right",
        cellClassName: "text-right",
        approxWidth: 100,
        cell: ({ row }) => <ActionsCell provider={row.original} actions={actions} />,
      },
    ];
  }, [actions]);

  const renderExpandedRow = React.useCallback(
    (row: Row<ApiOidcProvider>) => <ProviderExpandedRow provider={row.original} />,
    []
  );

  return (
    <AdaptiveTable
      tableId="oidc-providers"
      data={providers}
      columns={columns}
      enableSelection
      selectedRowIds={selectedProviderIds}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={onSelectAll}
      enableExpansion
      renderExpandedRow={renderExpandedRow}
      getRowId={(provider) => provider.client_id}
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
