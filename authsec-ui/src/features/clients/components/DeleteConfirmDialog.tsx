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
  clientName?: string;
  clientId: string;
  isLoading?: boolean;
}

export function DeleteConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  clientName,
  clientId,
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
            Delete Client
          </DialogTitle>
          <DialogDescription className="text-left">
            Are you sure you want to delete this client? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>
        
        <div className="space-y-3">
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3">
            <div className="text-sm">
              <div className="font-medium text-foreground">
                {clientName || "Unnamed Client"}
              </div>
              <div className="font-mono text-xs text-foreground mt-1">
                ID: {clientId}
              </div>
            </div>
          </div>
          
          <div className="text-sm text-foreground">
            This will permanently remove:
            <ul className="list-disc list-inside mt-1 space-y-1">
              <li>Client configuration and settings</li>
              <li>Authentication methods and credentials</li>
              <li>Access permissions and policies</li>
              <li>All associated logs and history</li>
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
            {isLoading ? "Deleting..." : "Delete Client"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}