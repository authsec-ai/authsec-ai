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

interface DeletePermissionConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  permissionId: string;
  fullPermissionString: string;
  action?: string;
  resource?: string;
  description?: string;
  rolesAssigned?: number;
  isLoading?: boolean;
}

export function DeletePermissionConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  permissionId,
  fullPermissionString,
  action,
  resource,
  description,
  rolesAssigned = 0,
  isLoading = false,
}: DeletePermissionConfirmDialogProps) {
  const handleConfirm = () => {
    onConfirm();
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
            Delete Permission
          </DialogTitle>
          <DialogDescription className="text-left">
            Are you sure you want to delete this permission? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3">
            <div className="text-sm space-y-1">
              <div className="font-medium text-foreground">
                {fullPermissionString}
              </div>
              <div className="font-mono text-xs text-foreground">
                ID: {permissionId}
              </div>
              {action && (
                <div className="text-xs text-foreground">
                  Action: {action}
                </div>
              )}
              {resource && (
                <div className="text-xs text-foreground">
                  Resource: {resource}
                </div>
              )}
              {description && (
                <div className="text-xs text-foreground mt-1">
                  {description}
                </div>
              )}
            </div>
          </div>

          {rolesAssigned > 0 && (
            <div className="rounded-md bg-yellow-500/10 border border-yellow-500/20 p-3">
              <p className="text-sm font-medium text-yellow-700 dark:text-yellow-400">
                Warning: This permission is currently assigned to {rolesAssigned} role{rolesAssigned !== 1 ? 's' : ''}
              </p>
            </div>
          )}

          <div className="text-sm text-foreground">
            This will permanently remove:
            <ul className="list-disc list-inside mt-1 space-y-1">
              {rolesAssigned > 0 && <li>Permission from all assigned roles</li>}
              <li>Permission definition and metadata</li>
              <li>Access control rules using this permission</li>
              <li>All associated audit logs</li>
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
            {isLoading ? "Deleting..." : "Delete Permission"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
