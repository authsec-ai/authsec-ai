import React, { useCallback, useMemo, useState } from "react";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
  type ResponsiveColumnDef,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { Button } from "@/components/ui/button";
import { CopyButton } from "@/components/ui/copy-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { format } from "date-fns";
import { Trash2, Shield, Database, Key, Calendar, MoreHorizontal, Code2 } from "lucide-react";
import type { Permission } from "@/app/api/permissionsApi";
import { useDeletePermissionsMutation } from "@/app/api/permissionsApi";
import { toast } from "@/lib/toast";
import { SessionManager } from "@/utils/sessionManager";
import { ViewSDKModal, generatePermissionSDKCode } from "@/features/sdk";
import { DeletePermissionConfirmDialog } from "./DeletePermissionConfirmDialog";

interface EnhancedPermissionsTableProps {
  permissions: Permission[];
  selectedPermissions: string[];
  onSelectAll: () => void;
  onSelectPermission: (permissionId: string) => void;
}

const ActionCell = ({ permission }: { permission: Permission }) => (
  <p className="truncate text-sm font-medium text-foreground" title={permission.action}>
    {permission.action}
  </p>
);

const ResourceCell = ({ permission }: { permission: Permission }) => (
  <p className="text-sm text-foreground" title={permission.resource}>
    {permission.resource}
  </p>
);

const DescriptionCell = ({ permission }: { permission: Permission }) => (
  <p className="text-sm text-foreground truncate" title={permission.description}>
    {permission.description}
  </p>
);

const RolesAssignedCell = ({ permission }: { permission: Permission }) => (
  <p className="text-sm text-foreground text-center">
    {permission.roles_assigned}
  </p>
);

const FullPermissionCell = ({ permission }: { permission: Permission }) => (
  <div className="flex items-center gap-2">
    <code className="text-xs font-mono text-foreground bg-muted px-2 py-1 rounded truncate max-w-[200px]" title={permission.full_permission_string}>
      {permission.full_permission_string}
    </code>
    <CopyButton text={permission.full_permission_string} label="Permission" size="sm" />
  </div>
);

// Expanded row component following ClientsPage pattern
const PermissionExpandedRow = ({ permission }: { permission: Permission }) => {
  // InfoLine component for horizontal label-value pairs
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
        {/* Left Column: Permission Details */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Shield className="h-4 w-4" />
            Permission Details
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Permission ID" value={permission.id} copyable />
            <InfoLine label="Full Permission" value={permission.full_permission_string} copyable />
            <InfoLine label="Action" value={permission.action} />
            <InfoLine label="Resource" value={permission.resource} />
          </div>
        </div>

        {/* Right Column: Additional Information */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Key className="h-4 w-4" />
            Additional Information
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Description" value={permission.description} />
            <InfoLine label="Roles Assigned" value={permission.roles_assigned} />
          </div>
        </div>
      </div>
    </div>
  );
};

export function EnhancedPermissionsTable({
  permissions,
  selectedPermissions,
  onSelectAll,
  onSelectPermission,
}: EnhancedPermissionsTableProps) {
  const [deletePermissions, { isLoading: isDeleting }] = useDeletePermissionsMutation();
  const tenantId = SessionManager.getSession()?.tenant_id || "";
  const [sdkModalOpen, setSdkModalOpen] = useState(false);
  const [selectedPermissionForSDK, setSelectedPermissionForSDK] = useState<Permission | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [permissionToDelete, setPermissionToDelete] = useState<Permission | null>(null);

  const handleViewSDK = useCallback((permission: Permission) => {
    setSelectedPermissionForSDK(permission);
    setSdkModalOpen(true);
  }, []);

  const handleDeleteClick = useCallback((permission: Permission) => {
    setPermissionToDelete(permission);
    setDeleteDialogOpen(true);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!permissionToDelete) return;

    try {
      await deletePermissions({
        tenant_id: tenantId,
        permission_ids: [permissionToDelete.id],
      }).unwrap();
      toast.success(`Permission deleted successfully`);
      setDeleteDialogOpen(false);
      setPermissionToDelete(null);
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete permission");
      // Keep modal open on error for retry
    }
  }, [permissionToDelete, deletePermissions, tenantId]);

  const columns: ResponsiveColumnDef<Permission, unknown>[] = useMemo(
    () => [
      {
        id: "action",
        header: "Action",
        cell: ({ row }) => <ActionCell permission={row.original} />,
        resizable: true,
        responsive: true,
        cellClassName: "max-w-0",
      },
      {
        id: "resource",
        header: "Resource",
        cell: ({ row }) => <ResourceCell permission={row.original} />,
        resizable: true,
        responsive: true,
      },
      {
        id: "full_permission_string",
        header: "Permission String",
        cell: ({ row }) => <FullPermissionCell permission={row.original} />,
        resizable: true,
        responsive: true,
        cellClassName: "min-w-[250px]",
      },
      {
        id: "description",
        header: "Description",
        cell: ({ row }) => <DescriptionCell permission={row.original} />,
        resizable: true,
        responsive: true,
      },
      {
        id: "roles_assigned",
        header: "Roles",
        cell: ({ row }) => <RolesAssignedCell permission={row.original} />,
        resizable: true,
        responsive: true,
        cellClassName: "text-center",
      },
      {
        id: "actions",
        header: "Actions",
        cell: ({ row }) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0" onClick={(e) => e.stopPropagation()}>
                <MoreHorizontal className="h-4 w-4" />
                <span className="sr-only">Open menu</span>
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
              <DropdownMenuItem onClick={() => handleViewSDK(row.original)}>
                <Code2 className="mr-2 h-4 w-4" />
                View SDK Code
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                onClick={() => handleDeleteClick(row.original)}
                className="text-destructive focus:text-destructive"
              >
                <Trash2 className="mr-2 h-4 w-4" />
                Delete
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        ),
        enableSorting: false,
        resizable: false,
        responsive: false,
        cellClassName: "text-center",
      },
    ],
    [handleDeleteClick, handleViewSDK]
  );

  const tableConfig: ResponsiveTableConfig<Permission> = {
    data: permissions,
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
    selectedRowIds: selectedPermissions,
    onRowSelectionChange: (ids) => {
      const prev = new Set(selectedPermissions);
      ids.forEach((id) => {
        if (!prev.has(id)) {
          onSelectPermission(id);
        }
      });
      selectedPermissions.forEach((id) => {
        if (!ids.includes(id)) {
          onSelectPermission(id);
        }
      });
    },
    onSelectAll,
    renderExpandedRow: (row) => <PermissionExpandedRow permission={row.original} />,
    getRowId: (row) => row.id,
  };

  const sdkCode = selectedPermissionForSDK
    ? generatePermissionSDKCode({
        action: selectedPermissionForSDK.action,
        resource: selectedPermissionForSDK.resource,
        full_permission_string: selectedPermissionForSDK.full_permission_string,
        description: selectedPermissionForSDK.description,
      })
    : { python: [], typescript: [] };

  return (
    <>
      <ResponsiveTableProvider tableType="permissions">
        <ResponsiveDataTable {...tableConfig} />
      </ResponsiveTableProvider>

      {selectedPermissionForSDK && (
        <ViewSDKModal
          open={sdkModalOpen}
          onOpenChange={setSdkModalOpen}
          title={`SDK Code for ${selectedPermissionForSDK.full_permission_string}`}
          description="Use this code to check, enforce, or manage this permission in your application."
          entityType="Permission"
          entityName={selectedPermissionForSDK.full_permission_string}
          pythonCode={sdkCode.python}
          typescriptCode={sdkCode.typescript}
          docsLink="/docs/sdk/permissions"
        />
      )}

      {permissionToDelete && (
        <DeletePermissionConfirmDialog
          open={deleteDialogOpen}
          onOpenChange={setDeleteDialogOpen}
          onConfirm={handleConfirmDelete}
          permissionId={permissionToDelete.id}
          fullPermissionString={permissionToDelete.full_permission_string}
          action={permissionToDelete.action}
          resource={permissionToDelete.resource}
          description={permissionToDelete.description}
          rolesAssigned={permissionToDelete.roles_assigned}
          isLoading={isDeleting}
        />
      )}
    </>
  );
}
