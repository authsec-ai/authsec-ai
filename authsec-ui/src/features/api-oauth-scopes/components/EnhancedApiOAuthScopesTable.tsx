import React, { useCallback, useMemo, useState } from "react";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
  type ResponsiveColumnDef,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Trash2, Edit, AlertTriangle } from "lucide-react";
import type { ApiOAuthScope } from "../types";
import { useDeleteAdminApiOAuthScopeMutation } from "@/app/api/admin/apiOAuthScopesApi";
import { useDeleteEndUserApiOAuthScopeMutation } from "@/app/api/enduser/apiOAuthScopesApi";
import { toast } from "@/lib/toast";
import {
  IdCell,
  NameCell,
  DescriptionCell,
  PermissionsLinkedCell,
  CreatedAtCell,
  ApiOAuthScopeExpandedRow,
} from "../utils/api-oauth-scopes-table-utils";

interface EnhancedApiOAuthScopesTableProps {
  scopes: ApiOAuthScope[];
  isAdmin: boolean;
  onEdit?: (scope: ApiOAuthScope) => void;
}

export function EnhancedApiOAuthScopesTable({
  scopes,
  isAdmin,
  onEdit,
}: EnhancedApiOAuthScopesTableProps) {
  const [deleteAdminScope] = useDeleteAdminApiOAuthScopeMutation();
  const [deleteEndUserScope] = useDeleteEndUserApiOAuthScopeMutation();
  const [scopeToDelete, setScopeToDelete] = useState<ApiOAuthScope | null>(null);
  const [isDeleting, setIsDeleting] = useState(false);

  const handleDeleteClick = useCallback((scope: ApiOAuthScope) => {
    setScopeToDelete(scope);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!scopeToDelete) return;

    setIsDeleting(true);
    try {
      if (isAdmin) {
        await deleteAdminScope({
          scope_id: scopeToDelete.id,
        }).unwrap();
      } else {
        await deleteEndUserScope({
          scope_id: scopeToDelete.id,
        }).unwrap();
      }
      toast.success(`API/OAuth scope "${scopeToDelete.name}" deleted successfully`);
      setScopeToDelete(null);
    } catch (error: any) {
      toast.error(
        error?.data?.message || "Failed to delete API/OAuth scope"
      );
    } finally {
      setIsDeleting(false);
    }
  }, [scopeToDelete, deleteAdminScope, deleteEndUserScope, isAdmin]);

  const handleCancelDelete = useCallback(() => {
    setScopeToDelete(null);
  }, []);

  const columns: ResponsiveColumnDef<ApiOAuthScope, unknown>[] = useMemo(
    () => [
      {
        id: "id",
        header: "ID",
        cell: ({ row }) => <IdCell scope={row.original} />,
        resizable: true,
        responsive: true,
        cellClassName: "min-w-[180px]",
      },
      {
        id: "name",
        header: "Name",
        cell: ({ row }) => <NameCell scope={row.original} />,
        resizable: true,
        responsive: true,
        cellClassName: "max-w-0",
      },
      {
        id: "description",
        header: "Description",
        cell: ({ row }) => <DescriptionCell scope={row.original} />,
        resizable: true,
        responsive: true,
      },
      {
        id: "permissions_linked",
        header: "Permissions",
        cell: ({ row }) => <PermissionsLinkedCell scope={row.original} />,
        resizable: true,
        responsive: true,
        cellClassName: "text-center",
      },
      {
        id: "created_at",
        header: "Created",
        cell: ({ row }) => <CreatedAtCell scope={row.original} />,
        resizable: true,
        responsive: true,
      },
      {
        id: "actions",
        header: "Actions",
        cell: ({ row }) => (
          <div className="flex items-center gap-2">
            {onEdit && (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 hover:bg-accent"
                onClick={(event) => {
                  event.stopPropagation();
                  onEdit(row.original);
                }}
              >
                <Edit className="h-4 w-4" />
                <span className="sr-only">Edit scope</span>
              </Button>
            )}
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-destructive hover:bg-destructive/10"
              onClick={(event) => {
                event.stopPropagation();
                handleDeleteClick(row.original);
              }}
            >
              <Trash2 className="h-4 w-4" />
              <span className="sr-only">Delete scope</span>
            </Button>
          </div>
        ),
        enableSorting: false,
        resizable: false,
        responsive: false,
        cellClassName: "text-center",
      },
    ],
    [handleDeleteClick, onEdit]
  );

  const tableConfig: ResponsiveTableConfig<ApiOAuthScope> = {
    data: scopes,
    columns,
    features: {
      selection: false,
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
    renderExpandedRow: (row) => <ApiOAuthScopeExpandedRow scope={row.original} />,
    getRowId: (row) => row.id,
  };

  return (
    <>
      <ResponsiveTableProvider tableType="api-oauth-scopes">
        <ResponsiveDataTable {...tableConfig} />
      </ResponsiveTableProvider>

      <Dialog open={!!scopeToDelete} onOpenChange={(open) => !open && handleCancelDelete()}>
        <DialogContent>
          <DialogHeader>
            <div className="flex items-center gap-2">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10">
                <AlertTriangle className="h-5 w-5 text-destructive" />
              </div>
              <DialogTitle>Delete API/OAuth Scope</DialogTitle>
            </div>
            <DialogDescription className="pt-2">
              Are you sure you want to delete the API/OAuth scope{" "}
              <span className="font-semibold text-foreground">
                "{scopeToDelete?.name}"
              </span>
              ? This action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={handleCancelDelete}
              disabled={isDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={isDeleting}
            >
              {isDeleting ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
