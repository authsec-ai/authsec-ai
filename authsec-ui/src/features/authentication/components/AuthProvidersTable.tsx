import * as React from "react";
import type { Row } from "@tanstack/react-table";
import type { UnifiedAuthProvider } from "../types";
import type { OidcProviderTableActions } from "../utils/oidc-provider-table-utils";
import {
  ProviderCell,
  StatusCell,
  ActionsCell,
  ProviderExpandedRow,
  OidcProviderTableUtils,
} from "../utils/oidc-provider-table-utils";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import { CopyButton } from "@/components/ui/copy-button";
import { Badge } from "@/components/ui/badge";

interface AuthProvidersTableProps {
  providers: UnifiedAuthProvider[];
  selectedProviderIds: string[];
  onSelectionChange: (selectedIds: string[]) => void;
  onSelectAll: () => void;
  actions: OidcProviderTableActions;
}

export function AuthProvidersTable({
  providers,
  selectedProviderIds,
  onSelectionChange,
  onSelectAll,
  actions,
}: AuthProvidersTableProps) {
  const columns = React.useMemo<AdaptiveColumn<UnifiedAuthProvider>[]>(() => {
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
        approxWidth: 180,
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            <Badge variant={row.original.provider_type === 'oidc' ? 'default' : 'secondary'} className="text-xs">
              {row.original.provider_type.toUpperCase()}
            </Badge>
            <span className="text-sm text-foreground truncate">
              {OidcProviderTableUtils.formatProviderName(row.original.provider_name)}
            </span>
          </div>
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
        header: "Configuration",
        accessorFn: (provider) => provider.callback_url || provider.entity_id || '',
        priority: 4,
        enableSorting: true,
        resizable: true,
        approxWidth: 240,
        cell: ({ row }) => {
          const provider = row.original;
          const isOidc = provider.provider_type === 'oidc';
          const value = isOidc ? provider.callback_url : provider.entity_id;
          const label = isOidc ? 'Callback URL' : 'Entity ID';

          return (
            <div className="flex items-center gap-2 min-w-0">
              <div className="flex flex-col min-w-0">
                <span className="text-xs text-foreground">{label}</span>
                <span className="truncate text-sm" title={value}>
                  {value}
                </span>
              </div>
              {value && <CopyButton text={value} label={label} size="sm" variant="ghost" />}
            </div>
          );
        },
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
    (row: Row<UnifiedAuthProvider>) => <ProviderExpandedRow provider={row.original} />,
    []
  );

  return (
    <AdaptiveTable
      tableId="auth-providers"
      data={providers}
      columns={columns}
      enableSelection
      selectedRowIds={selectedProviderIds}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={onSelectAll}
      enableExpansion
      renderExpandedRow={renderExpandedRow}
      getRowId={(provider) => provider.id}
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
