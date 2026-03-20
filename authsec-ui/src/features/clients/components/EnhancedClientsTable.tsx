import React from "react";
import type { Row } from "@tanstack/react-table";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import type { ClientWithAuthMethods } from "@/types/entities";
import {
  createAdaptiveClientsTableColumns,
  ClientExpandedRow,
  type ClientsTableActions,
} from "../utils/clients-table-utils";

interface EnhancedClientsTableProps {
  data: ClientWithAuthMethods[];
  selectedClients?: string[];
  onSelectionChange?: (clientIds: string[]) => void;
  onDeleteClient: (clientId: string) => void;
  onCreateClient: () => void;
  onToggleStatus?: (clientId: string) => void;
  onViewSDK?: (clientId: string) => void;
  onAddAuthMethod?: (clientId: string) => void;
  onShowAuthMethods?: (client: ClientWithAuthMethods) => void;
  onPreviewLogin?: (clientId: string) => void;
  onConfigureVoiceAgent?: (clientId: string) => void;
  newClientId?: string;
  newClientStep?: number;
  onNextNewClientStep?: () => void;
  onDismissNewClient?: () => void;
  pageIndex?: number;
  onPageIndexChange?: (page: number) => void;
  serverTotalItems?: number;
}

export function EnhancedClientsTable({
  data,
  selectedClients = [],
  onSelectionChange,
  onDeleteClient,
  onCreateClient,
  onToggleStatus,
  onViewSDK,
  onAddAuthMethod,
  onShowAuthMethods,
  onPreviewLogin,
  onConfigureVoiceAgent,
  newClientId,
  newClientStep,
  onNextNewClientStep,
  onDismissNewClient,
  pageIndex,
  onPageIndexChange,
  serverTotalItems,
}: EnhancedClientsTableProps) {
  const [expandedRows, setExpandedRows] = React.useState<string[]>([]);

  // Keep expanded rows in sync with current data set
  React.useEffect(() => {
    if (!data.length) {
      setExpandedRows([]);
      return;
    }

    setExpandedRows((prev) => prev.filter((id) => data.some((client) => client.id === id)));
  }, [data]);

  // Scroll newly created client's row into view after server data loads
  React.useEffect(() => {
    if (!newClientId) return;
    // Longer delay to account for server page fetch + table render
    const t = setTimeout(() => {
      const el = document.querySelector("[data-new-client='true']");
      el?.scrollIntoView({ behavior: "smooth", block: "center" });
    }, 800);
    return () => clearTimeout(t);
  }, [newClientId, data]); // re-run when data updates so scroll fires after page load

  // Table actions - memoized to prevent infinite re-renders
  const actions: ClientsTableActions = React.useMemo(
    () => ({
      onDelete: onDeleteClient,
      onToggleStatus: onToggleStatus,
      onViewSDK: onViewSDK,
      onAddAuthMethod,
      onShowAuthMethods,
      onPreviewLogin,
      onConfigureVoiceAgent,
      newClientId,
      newClientStep,
      onNextNewClientStep,
      onDismissNewClient,
    }),
    [
      onDeleteClient,
      onToggleStatus,
      onViewSDK,
      onAddAuthMethod,
      onShowAuthMethods,
      onPreviewLogin,
      onConfigureVoiceAgent,
      newClientId,
      newClientStep,
      onNextNewClientStep,
      onDismissNewClient,
    ],
  );

  // Columns - memoized with actions
  const columns = React.useMemo<AdaptiveColumn<ClientWithAuthMethods>[]>(
    () => createAdaptiveClientsTableColumns(actions),
    [actions],
  );

  // Render expanded row callback
  const renderExpandedRow = React.useCallback(
    (row: Row<ClientWithAuthMethods>) => (
      <ClientExpandedRow client={row.original} actions={actions} />
    ),
    [actions],
  );

  return (
    <AdaptiveTable
      tableId="clients"
      data={data}
      columns={columns}
      enableSelection
      selectedRowIds={selectedClients}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={() => {
        if (onSelectionChange) {
          if (selectedClients.length === data.length) {
            onSelectionChange([]);
          } else {
            onSelectionChange(data.map((client) => client.id));
          }
        }
      }}
      enableExpansion={true}
      renderExpandedRow={renderExpandedRow}
      getRowId={(client) => client.id}
      expandedRowIds={expandedRows}
      onExpandedRowsChange={setExpandedRows}
      enableSorting
      enablePagination
      pagination={{
        pageSize: 10,
        pageSizeOptions: [5, 10, 25, 50],
        alwaysVisible: true,
      }}
      pageIndex={pageIndex}
      onPageIndexChange={onPageIndexChange}
      serverTotalItems={serverTotalItems}
    />
  );
}
