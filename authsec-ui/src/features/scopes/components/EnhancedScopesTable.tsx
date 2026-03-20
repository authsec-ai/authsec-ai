import React, { useMemo, useCallback, useState } from "react";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { MoreVertical, Edit, Trash2, Code2, AlertTriangle } from "lucide-react";
import { useDeleteScopeByNameMutation } from "@/app/api/scopesApi";
import { useDeleteEndUserScopeMutation } from "@/app/api/enduser/scopesApi";
import { toast } from "@/lib/toast";
import type { ResponsiveColumnDef } from "@/components/ui/responsive-data-table";
import { DropdownMenuSeparator } from "@/components/ui/dropdown-menu";
import { EditScopeModal } from "./EditScopeModal";
import { ViewSDKModal, generateScopeSDKCode } from "@/features/sdk";

interface ScopeWithResources {
  id: string;
  name: string;
  description?: string;
  resources: string[];
  created_at?: string;
  updated_at?: string;
}

interface EnhancedScopesTableProps {
  scopes: ScopeWithResources[];
  isAdmin: boolean;
  tenantId: string;
}

const ScopeNameCell = ({ scope }: { scope: ScopeWithResources }) => (
  <p
    className="truncate text-sm font-medium text-foreground"
    title={scope.name}
  >
    {scope.name}
  </p>
);

const ResourcesCell = ({ resources }: { resources: string[] }) => {
  if (!resources || resources.length === 0) {
    return <span className="text-foreground text-sm">—</span>;
  }

  const displayResources = resources.slice(0, 3);
  const remaining = resources.length - 3;

  return (
    <div className="flex flex-wrap gap-2 text-sm text-foreground">
      {displayResources.map((resource) => (
        <span key={resource} className="font-mono leading-tight" title={resource}>
          {resource}
        </span>
      ))}
      {remaining > 0 && (
        <span className="text-sm text-foreground">+{remaining} more</span>
      )}
    </div>
  );
};

export function EnhancedScopesTable({
  scopes,
  isAdmin,
  tenantId,
}: EnhancedScopesTableProps) {
  const [editingScope, setEditingScope] = useState<ScopeWithResources | null>(
    null
  );
  const [deleteScopeByName] = useDeleteScopeByNameMutation();
  const [deleteEndUserScope] = useDeleteEndUserScopeMutation();
  const [sdkModalOpen, setSdkModalOpen] = useState(false);
  const [selectedScopeForSDK, setSelectedScopeForSDK] =
    useState<ScopeWithResources | null>(null);
  const [scopeToDelete, setScopeToDelete] = useState<ScopeWithResources | null>(
    null
  );
  const [isDeleting, setIsDeleting] = useState(false);

  const handleViewSDK = useCallback((scope: ScopeWithResources) => {
    setSelectedScopeForSDK(scope);
    setSdkModalOpen(true);
  }, []);

  const handleEdit = useCallback((scope: ScopeWithResources) => {
    setEditingScope(scope);
  }, []);

  const handleDeleteClick = useCallback((scope: ScopeWithResources) => {
    setScopeToDelete(scope);
  }, []);

  const handleConfirmDelete = useCallback(async () => {
    if (!scopeToDelete) return;

    setIsDeleting(true);
    try {
      if (isAdmin) {
        await deleteScopeByName(scopeToDelete.name).unwrap();
      } else {
        await deleteEndUserScope({
          scope_id: scopeToDelete.id,
        }).unwrap();
      }
      toast.success(`Scope "${scopeToDelete.name}" deleted successfully`);
      setScopeToDelete(null);
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete scope");
    } finally {
      setIsDeleting(false);
    }
  }, [scopeToDelete, deleteScopeByName, deleteEndUserScope, isAdmin, tenantId]);

  const handleCancelDelete = useCallback(() => {
    setScopeToDelete(null);
  }, []);

  const columns: ResponsiveColumnDef<ScopeWithResources, unknown>[] = useMemo(
    () => [
      {
        id: "scope",
        header: "Scope Name",
        cell: ({ row }) => <ScopeNameCell scope={row.original} />,
        cellClassName: "max-w-0",
        resizable: true,
        responsive: true,
      },
      {
        id: "resources",
        header: "Resources",
        cell: ({ row }) => <ResourcesCell resources={row.original.resources} />,
        resizable: true,
        responsive: true,
      },
      ...(isAdmin
        ? []
        : [
            {
              id: "description",
              header: "Description",
              cell: ({ row }: { row: any }) => (
                <p
                  className="truncate text-sm text-foreground"
                  title={row.original.description || ""}
                >
                  {row.original.description || "—"}
                </p>
              ),
              resizable: true,
              responsive: true,
            } as ResponsiveColumnDef<ScopeWithResources, unknown>,
          ]),
      {
        id: "actions",
        header: "Actions",
        cell: ({ row }) => (
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
                <MoreVertical className="h-4 w-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
              <DropdownMenuItem onClick={() => handleViewSDK(row.original)}>
                <Code2 className="mr-2 h-4 w-4" />
                View SDK Code
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleEdit(row.original)}>
                <Edit className="mr-2 h-4 w-4" />
                Edit Resources
              </DropdownMenuItem>
              <DropdownMenuSeparator />
              <DropdownMenuItem
                className="text-destructive"
                onClick={() => handleDeleteClick(row.original)}
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
    [handleDeleteClick, handleEdit, handleViewSDK, isAdmin]
  );

  const tableConfig: ResponsiveTableConfig<ScopeWithResources> = {
    data: scopes,
    columns,
    features: {
      selection: false,
      dragDrop: false,
      expandable: false,
      pagination: true,
      sorting: true,
      resizing: true,
    },
    pagination: {
      pageSize: 10,
      pageSizeOptions: [5, 10, 25, 50, 100],
      alwaysVisible: true,
    },
    getRowId: (row) => row.id,
  };

  const sdkCode = selectedScopeForSDK
    ? generateScopeSDKCode({
        id: selectedScopeForSDK.id,
        name: selectedScopeForSDK.name,
        description: selectedScopeForSDK.description,
        resources: selectedScopeForSDK.resources,
      })
    : { python: [], typescript: [] };

  return (
    <>
      {editingScope && (
        <EditScopeModal
          scope={editingScope}
          open={!!editingScope}
          onOpenChange={(open) => !open && setEditingScope(null)}
        />
      )}
      <ResponsiveTableProvider tableType="scopes">
        <ResponsiveDataTable {...tableConfig} />
      </ResponsiveTableProvider>

      {selectedScopeForSDK && (
        <ViewSDKModal
          open={sdkModalOpen}
          onOpenChange={setSdkModalOpen}
          title={`SDK Code for ${selectedScopeForSDK.name}`}
          description="Use this code to check scope access, protect tools, or manage this scope in your application."
          entityType="Scope"
          entityName={selectedScopeForSDK.name}
          pythonCode={sdkCode.python}
          typescriptCode={sdkCode.typescript}
          docsLink="/docs/sdk/scopes"
        />
      )}

      <Dialog
        open={!!scopeToDelete}
        onOpenChange={(open) => !open && handleCancelDelete()}
      >
        <DialogContent>
          <DialogHeader>
            <div className="flex items-center gap-2">
              <div className="flex h-10 w-10 items-center justify-center rounded-full bg-destructive/10">
                <AlertTriangle className="h-5 w-5 text-destructive" />
              </div>
              <DialogTitle>Delete Scope</DialogTitle>
            </div>
            <DialogDescription className="pt-2">
              Are you sure you want to delete the scope{" "}
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
