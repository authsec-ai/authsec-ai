import React from "react";
import { useNavigate } from "react-router-dom";
import {
  ResponsiveDataTable,
  type ResponsiveTableConfig,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { Card, CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { toast } from "@/lib/toast";
import type { RawExternalService } from "@/app/api/externalServiceApi";
import {
  createExternalServiceTableColumns,
  ExternalServiceExpandedRow,
  type ExternalServiceTableActions,
} from "../utils/external-services-table-utils";
import {
  useDeleteExternalServiceMutation,
} from "@/app/api/externalServiceApi";

interface EnhancedExternalServicesTableProps {
  data: RawExternalService[];
  selectedServices?: string[];
  onSelectService?: (serviceId: string) => void;
  onSelectAll?: () => void;
  onCreateService?: () => void;
}

export function EnhancedExternalServicesTable({
  data,
  selectedServices: externalSelected = [],
  onSelectService,
  onSelectAll,
  onCreateService,
}: EnhancedExternalServicesTableProps) {
  const navigate = useNavigate();

  // If parent controls selection use that, otherwise maintain internal state
  const [internalSelected, setInternalSelected] = React.useState<string[]>([]);
  const selectedRowIds = externalSelected.length > 0 ? externalSelected : internalSelected;

  const [deleteService] = useDeleteExternalServiceMutation();
  const [serviceToDelete, setServiceToDelete] = React.useState<RawExternalService | null>(null);
  const [isDeleteLoading, setIsDeleteLoading] = React.useState(false);
  const [isBulkDeleteOpen, setIsBulkDeleteOpen] = React.useState(false);
  const [isBulkDeleting, setIsBulkDeleting] = React.useState(false);

  // Action handlers
  const handleEdit = (service: RawExternalService) => {
    toast.info(`Edit service: ${service.name}`);
    // TODO: Navigate to edit page or open edit modal
  };

  const handleDelete = (service: RawExternalService) => {
    setServiceToDelete(service);
  };

  const handleViewSDK = (service: RawExternalService) => {
    // Navigate to SDK page for this service
    navigate(`/sdk/external-services/${service.id}`);
  };

  const handleViewSecret = (service: RawExternalService) => {
    // Find the row and expand it to show secrets
    toast.info(`Expand the row to view secrets for ${service.name}`);
  };

  const handleBulkDelete = () => {
    if (selectedRowIds.length === 0) return;
    setIsBulkDeleteOpen(true);
  };

  const confirmDelete = async () => {
    if (!serviceToDelete) return;
    setIsDeleteLoading(true);
    try {
      await deleteService(serviceToDelete.id).unwrap();
      toast.success(`Service "${serviceToDelete.name}" deleted successfully`);
      setServiceToDelete(null);
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to delete service");
    } finally {
      setIsDeleteLoading(false);
    }
  };

  const confirmBulkDelete = async () => {
    if (selectedRowIds.length === 0) {
      setIsBulkDeleteOpen(false);
      return;
    }

    setIsBulkDeleting(true);
    try {
      await Promise.all(selectedRowIds.map((id) => deleteService(id).unwrap()));
      toast.success(`${selectedRowIds.length} service(s) deleted successfully`);
      setInternalSelected((prev) => prev.filter((id) => !selectedRowIds.includes(id)));
    } catch (error) {
      toast.error("Failed to delete some services");
    } finally {
      setIsBulkDeleting(false);
      setIsBulkDeleteOpen(false);
    }
  };

  const actions = React.useMemo<ExternalServiceTableActions>(() => {
    return {
      onEdit: handleEdit,
      onDelete: handleDelete,
      onViewSDK: handleViewSDK,
      onViewSecret: handleViewSecret,
    };
  }, [handleEdit, handleDelete, handleViewSDK, handleViewSecret]);

  const columns = React.useMemo(() => createExternalServiceTableColumns(actions), [actions]);

  const tableConfig: ResponsiveTableConfig<RawExternalService> = {
    data,
    columns,
    features: {
      selection: true,
      dragDrop: false,
      expandable: true,
      pagination: true,
      sorting: true,
      resizing: true,
    },
    selectedRowIds,
    onRowSelectionChange: (ids) => {
      if (onSelectService) {
        // controlled by parent – emit individual changes
        const prev = new Set(selectedRowIds);
        ids.forEach((id) => {
          if (!prev.has(id)) onSelectService(id);
        });
        selectedRowIds.forEach((id) => {
          if (!ids.includes(id)) onSelectService(id);
        });
      } else {
        setInternalSelected(ids);
      }
    },
    onSelectAll:
      onSelectAll ??
      (() =>
        setInternalSelected(selectedRowIds.length === data.length ? [] : data.map((s) => s.id))),
    renderExpandedRow: (row) => <ExternalServiceExpandedRow service={row.original} />,
    getRowId: (row) => row.id,
  };

  return (
    <>
      <ResponsiveTableProvider tableType="externalServices">
        <Card className="border-0 bg-card">
          <CardContent className="p-0">
            {/* Quick Actions Bar */}
            {selectedRowIds.length > 0 && (
              <div className="flex items-center justify-between gap-4 border-b border-border px-6 py-4 bg-muted/50">
                <div className="flex items-center gap-4">
                  <span className="text-sm font-medium">
                    {selectedRowIds.length} service(s) selected
                  </span>
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="destructive"
                    size="sm"
                    onClick={handleBulkDelete}
                  >
                    Delete Selected
                  </Button>
                </div>
              </div>
            )}

            {/* Table */}
            <ResponsiveDataTable {...tableConfig} />

            {/* {data.length === 0 && onCreateService && (
              <div className="flex flex-col items-center gap-2 py-8">
                <span className="text-foreground">No external services found.</span>
                <Button variant="outline" onClick={onCreateService}>
                  Add External Service
                </Button>
              </div>
            )} */}
          </CardContent>
        </Card>
      </ResponsiveTableProvider>

      <Dialog
        open={!!serviceToDelete}
        onOpenChange={(open) => {
          if (!open && !isDeleteLoading) {
            setServiceToDelete(null);
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete external service</DialogTitle>
            <DialogDescription>
              This removes the integration and its credentials. You cannot undo this action.
            </DialogDescription>
          </DialogHeader>
          <div className="rounded-md border border-destructive/20 bg-destructive/5 p-4 text-sm">
            <p className="font-semibold text-destructive">
              {serviceToDelete?.name}
            </p>
            <p className="mt-1 text-foreground">
              {serviceToDelete?.description || "Please confirm to proceed."}
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setServiceToDelete(null)}
              disabled={isDeleteLoading}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmDelete}
              disabled={isDeleteLoading}
            >
              {isDeleteLoading ? "Deleting..." : "Delete"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog
        open={isBulkDeleteOpen}
        onOpenChange={(open) => {
          if (isBulkDeleting) return;
          setIsBulkDeleteOpen(open);
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete selected services</DialogTitle>
            <DialogDescription>
              You are about to delete {selectedRowIds.length} service(s). This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setIsBulkDeleteOpen(false)}
              disabled={isBulkDeleting}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={confirmBulkDelete}
              disabled={isBulkDeleting || selectedRowIds.length === 0}
            >
              {isBulkDeleting ? "Deleting..." : "Delete Selected"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}
