import React from "react";
import { AlertTriangle } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";

interface DeleteConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  workloadId: string;
  workloadName?: string;
  isLoading?: boolean;
}

export function DeleteConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  workloadId,
  workloadName,
  isLoading = false,
}: DeleteConfirmDialogProps) {
  const handleConfirm = () => {
    onConfirm();
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <AlertTriangle className="h-5 w-5" />
            Delete Workload
          </DialogTitle>
          <DialogDescription className="text-left">
            Are you sure you want to delete this workload? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3">
            <div className="text-sm">
              <div className="font-medium text-foreground">
                {workloadName || "Unnamed Workload"}
              </div>
              <div className="font-mono text-xs text-foreground mt-1">
                ID: {workloadId}
              </div>
            </div>
          </div>

          <div className="text-sm text-foreground">
            This will permanently remove:
            <ul className="list-disc list-inside mt-1 space-y-1">
              <li>Workload registration and identity</li>
              <li>SPIFFE ID and selectors</li>
              <li>Attestation configuration</li>
              <li>All associated metadata</li>
            </ul>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleCancel}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleConfirm}
            disabled={isLoading}
          >
            {isLoading ? "Deleting..." : "Delete Workload"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
