import React, { useState, useCallback } from "react";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
} from "../../../components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "../../../components/ui/responsive-table";
import type { ApiOidcProvider } from "../utils/oidc-provider-table-utils";
import {
  createDynamicOidcProviderTableColumns,
  ProviderExpandedRow,
  type OidcProviderTableActions,
} from "../utils/oidc-provider-table-utils";
import { ColumnSelector, type ColumnConfig } from "./ColumnSelector";
import { DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS } from "../utils/oidc-provider-dynamic-columns";

interface EnhancedOidcProvidersTableProps {
  data: ApiOidcProvider[];
  selectedProviders: string[];
  onSelectAll: () => void;
  onSelectProvider: (providerId: string) => void;
  onDuplicateProvider: (providerId: string) => void;
  onDeleteProvider: (providerId: string) => void;
  onToggleActive: (providerId: string, isActive: boolean) => void;
  onViewConfiguration: (providerId: string) => void;
  onTestConnection: (providerId: string) => void;
  onCreateProvider: () => void;
  enableDynamicColumns?: boolean;
  initialColumnConfig?: ColumnConfig[];
}

export function EnhancedOidcProvidersTable({
  data,
  selectedProviders,
  onSelectAll,
  onSelectProvider,
  onDuplicateProvider,
  onDeleteProvider,
  onToggleActive,
  onViewConfiguration,
  onTestConnection,
  onCreateProvider,
  enableDynamicColumns = true,
  initialColumnConfig,
}: EnhancedOidcProvidersTableProps) {
  return (
    <ResponsiveTableProvider tableType="oidc-providers">
      <EnhancedOidcProvidersTableContent
        data={data}
        selectedProviders={selectedProviders}
        onSelectAll={onSelectAll}
        onSelectProvider={onSelectProvider}
        onDuplicateProvider={onDuplicateProvider}
        onDeleteProvider={onDeleteProvider}
        onToggleActive={onToggleActive}
        onViewConfiguration={onViewConfiguration}
        onTestConnection={onTestConnection}
        onCreateProvider={onCreateProvider}
        enableDynamicColumns={enableDynamicColumns}
        initialColumnConfig={initialColumnConfig}
      />
    </ResponsiveTableProvider>
  );
}

function EnhancedOidcProvidersTableContent({
  data,
  selectedProviders,
  onSelectAll,
  onSelectProvider,
  onDuplicateProvider,
  onDeleteProvider,
  onToggleActive,
  onViewConfiguration,
  onTestConnection,
  onCreateProvider,
  enableDynamicColumns = true,
  initialColumnConfig,
}: EnhancedOidcProvidersTableProps) {
  // Column management state with localStorage persistence
  const [columnConfigs, setColumnConfigs] = useState<ColumnConfig[]>(() => {
    if (initialColumnConfig) return initialColumnConfig;

    // Try to load from localStorage
    try {
      const saved = localStorage.getItem("oidcProviderTableColumns");
      if (saved) {
        const parsed = JSON.parse(saved);
        // Validate that the saved config has all required columns
        const savedIds = parsed.map((col: ColumnConfig) => col.id);
        const defaultIds = DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS.map((col) => col.id);
        const hasAllColumns = defaultIds.every((id) => savedIds.includes(id));

        if (hasAllColumns) {
          return parsed;
        }
      }
    } catch (error) {
      console.warn("Failed to load OIDC provider column preferences:", error);
    }

    return DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS;
  });

  // Expansion state for provider click functionality
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());

  // Create table actions object
  const actions: OidcProviderTableActions = {
    onDuplicate: onDuplicateProvider,
    onDelete: onDeleteProvider,
    onToggleActive,
    onViewConfiguration,
    onTestConnection,
  };

  // Handle column configuration changes
  const handleColumnsChange = useCallback((newConfigs: ColumnConfig[]) => {
    setColumnConfigs(newConfigs);
    // Save to localStorage
    localStorage.setItem("oidcProviderTableColumns", JSON.stringify(newConfigs));
  }, []);

  // Reset to default columns
  const handleResetColumns = useCallback(() => {
    setColumnConfigs(DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS);
    localStorage.removeItem("oidcProviderTableColumns");
  }, []);

  // Get visible column IDs
  const visibleColumnIds = React.useMemo(
    () => columnConfigs.filter((col) => col.isVisible).map((col) => col.id),
    [columnConfigs]
  );

  // Handle expansion toggle
  const handleToggleExpand = useCallback((rowId: string) => {
    setExpandedRows((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(rowId)) {
        newSet.delete(rowId);
      } else {
        newSet.add(rowId);
      }
      return newSet;
    });
  }, []);

  // Get row ID for expansion tracking
  const getRowId = useCallback((provider: ApiOidcProvider) => provider.client_id, []);

  // Create dynamic columns based on visible configuration
  const columns = React.useMemo(() => {
    return createDynamicOidcProviderTableColumns(visibleColumnIds, actions);
  }, [visibleColumnIds, actions]);

  // Table configuration with expandable feature enabled
  const tableConfig: ResponsiveTableConfig = {
    enableRowSelection: true,
    enableColumnResizing: true,
    enableColumnOrdering: true,
    enablePagination: true,
    enableSearch: true,
    searchPlaceholder: "Search OIDC providers...",
    features: {
      expandable: true,
    },
    emptyStateConfig: {
      title: "No OIDC Providers",
      description: "No OAuth/OIDC providers have been configured yet.",
      actionLabel: "Add Provider",
      onAction: onCreateProvider,
    },
  };

  // Handle row selection
  const handleRowSelection = useCallback(
    (selectedRowIds: string[]) => {
      // Update selected providers based on row selection
      selectedRowIds.forEach((rowId) => {
        const provider = data.find((p) => getRowId(p) === rowId);
        if (provider) {
          onSelectProvider(provider.client_id);
        }
      });
    },
    [data, onSelectProvider, getRowId]
  );

  // Render expanded row content
  const renderExpandedRow = useCallback((row: any) => {
    return <ProviderExpandedRow provider={row.original} />;
  }, []);

  return (
    <div className="space-y-4">
      {/* Responsive Data Table */}
      <ResponsiveDataTable
        data={data}
        columns={columns}
        config={tableConfig}
        selectedRowIds={selectedProviders}
        onRowSelection={handleRowSelection}
        onSelectAll={onSelectAll}
        getRowId={getRowId}
        expandedRowIds={Array.from(expandedRows)}
        onExpandedRowsChange={(expandedIds: string[]) => setExpandedRows(new Set(expandedIds))}
        renderExpandedRow={renderExpandedRow}
      />
    </div>
  );
}

// Export default configuration for easy import
export const DefaultOidcProvidersTableConfig = {
  columns: DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS,
  defaultVisibleColumns: DEFAULT_OIDC_PROVIDER_COLUMN_CONFIGS.filter((col) => col.isVisible).map(
    (col) => col.id
  ),
};
