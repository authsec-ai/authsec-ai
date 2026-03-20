import React, { useState, useMemo, useCallback } from "react";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { Button } from "@/components/ui/button";
import { toast } from "@/lib/toast";
import { Plus } from "lucide-react";
import type { EnhancedRole } from "@/types/entities";
import {
  createRoleTableColumns,
  RoleExpandedRow,
  type RoleTableActions,
} from "../utils/role-table-utils";
import { ViewSDKModal, generateRoleSDKCode } from "@/features/sdk";

interface EnhancedRolesTableProps {
  data: EnhancedRole[];
  selectedRoles?: string[];
  onSelectRole?: (roleId: string) => void;
  onSelectAll?: () => void;
  onCreateRole?: () => void;
  // Additional callbacks for RolesPage convenience
  onOpenDrawer?: (roleId: string) => void;
  onEditRole?: (roleId: string) => void;
  onDuplicateRole?: (roleId: string) => void;
  onDeleteRole?: (roleId: string) => void;
  onAssignUsers?: (roleId: string) => void;
  onViewVersionHistory?: (roleId: string) => void;
  onEditPermissions?: (roleId: string) => void;
  highlightRoleId?: string | null;
}

export function EnhancedRolesTable({
  data,
  selectedRoles: externalSelected = [],
  onSelectRole,
  onSelectAll,
  onCreateRole,
  onOpenDrawer,
  onEditRole,
  onDuplicateRole,
  onDeleteRole,
  onAssignUsers,
  onViewVersionHistory,
  onEditPermissions,
  highlightRoleId,
}: EnhancedRolesTableProps) {
  /*
    We reuse the Responsive table infrastructure that powers the Users page.
    If parent passes controlled selection arrays we use them, otherwise we
    manage selection internally.
  */
  const [internalSelected, setInternalSelected] = useState<string[]>([]);
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [sdkModalOpen, setSdkModalOpen] = useState(false);
  const [selectedRoleForSDK, setSelectedRoleForSDK] = useState<EnhancedRole | null>(null);

  const selectedRowIds = externalSelected.length > 0 ? externalSelected : internalSelected;

  const handleViewSDK = useCallback((role: EnhancedRole) => {
    setSelectedRoleForSDK(role);
    setSdkModalOpen(true);
  }, []);

  // Handle row expansion
  const handleToggleExpand = (rowId: string) => {
    setExpandedRows(prev => {
      const newSet = new Set(prev);
      if (newSet.has(rowId)) {
        newSet.delete(rowId);
      } else {
        newSet.add(rowId);
      }
      return newSet;
    });
  };

  // Get row ID for expansion tracking
  const getRowId = (row: EnhancedRole) => row.id;

  // Memoize actions to avoid lint warning
  const actions = React.useMemo<RoleTableActions>(
    () => ({
      onEdit: onEditRole || ((id) => toast.info(`Edit role ${id}`)),
      onDuplicate: onDuplicateRole || ((id) => toast.info(`Duplicate role ${id}`)),
      onDelete: onDeleteRole || ((id) => toast.error(`Delete role ${id}`)),
      onAssignUsers: onAssignUsers || ((id) => toast.info(`Assign users to role ${id}`)),
      onEditPermissions: onEditPermissions || ((id) => toast.info(`Edit permissions for role ${id}`)),
      onViewVersionHistory: onViewVersionHistory || ((id) => toast.info(`View version history for role ${id}`)),
      onViewSDK: handleViewSDK,
    }),
    [onEditRole, onDuplicateRole, onDeleteRole, onAssignUsers, onEditPermissions, onViewVersionHistory, handleViewSDK]
  );

  const columns = useMemo(() => createRoleTableColumns(
    actions,
    expandedRows,
    handleToggleExpand,
    getRowId
  ), [actions, expandedRows]);

  const rowClassName = React.useCallback(
    (row: EnhancedRole) =>
      row.id === highlightRoleId
        ? "bg-amber-50 dark:bg-amber-900/20 ring-1 ring-amber-200 dark:ring-amber-800/40"
        : "",
    [highlightRoleId]
  );

  const tableConfig: ResponsiveTableConfig<EnhancedRole> = {
    data: Array.isArray(data) ? data : [],
    columns,
    features: {
      selection: true,
      dragDrop: false,
      expandable: true,
      pagination: true,
      sorting: true,
      resizing: true,
    },
    pagination: {
      pageSize: 10,
      pageSizeOptions: [5, 10, 25, 50, 100],
      alwaysVisible: true,
    },
    selectedRowIds,
    onRowSelectionChange: (ids) => {
      if (onSelectRole) {
        // Controlled by parent – emit individual changes
        const prev = new Set(selectedRowIds);
        ids.forEach((id) => {
          if (!prev.has(id)) onSelectRole(id);
        });
        selectedRowIds.forEach((id) => {
          if (!ids.includes(id)) onSelectRole(id);
        });
      } else {
        // Internal state
        setInternalSelected(ids);
      }
    },
    onSelectAll:
      onSelectAll ??
      (() => {
        const safeData = Array.isArray(data) ? data : [];
        setInternalSelected(
          selectedRowIds.length === safeData.length ? [] : safeData.map((r) => r.id)
        );
      }),
    renderExpandedRow: (row) => <RoleExpandedRow role={row.original} />,
    getRowId: (row) => row.id,
    expandedRowIds: Array.from(expandedRows),
    onRowExpansionChange: (ids) => {
      setExpandedRows(new Set(ids));
    },
    onRowClick: (row) => onOpenDrawer?.(row.id),
    rowClassName,
  };

  const sdkCode = selectedRoleForSDK
    ? generateRoleSDKCode({
        id: selectedRoleForSDK.id,
        name: selectedRoleForSDK.name,
        description: selectedRoleForSDK.description,
        permissions: selectedRoleForSDK.permissions,
        grants: (selectedRoleForSDK as any).grants,
      })
    : { python: [], typescript: [] };

  return (
    <>
      <ResponsiveTableProvider tableType="roles">
        <ResponsiveDataTable {...tableConfig} />
      </ResponsiveTableProvider>

      {selectedRoleForSDK && (
        <ViewSDKModal
          open={sdkModalOpen}
          onOpenChange={setSdkModalOpen}
          title={`SDK Code for ${selectedRoleForSDK.name}`}
          description="Use this code to check role membership, protect tools, or manage this role in your application."
          entityType="Role"
          entityName={selectedRoleForSDK.name}
          pythonCode={sdkCode.python}
          typescriptCode={sdkCode.typescript}
          docsLink="/docs/sdk/roles"
        />
      )}
    </>
  );
}
