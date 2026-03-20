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

interface DeleteProviderConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
  providerName?: string;
  providerId: string;
  providerType?: 'oidc' | 'saml';
  isLoading?: boolean;
}

export function DeleteProviderConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  providerName,
  providerId,
  providerType = 'oidc',
  isLoading = false,
}: DeleteProviderConfirmDialogProps) {
  const handleConfirm = () => {
    onConfirm();
    onOpenChange(false);
  };

  const handleCancel = () => {
    onOpenChange(false);
  };

  const providerTypeLabel = providerType === 'saml' ? 'SAML' : 'OIDC';
  const configLabel = providerType === 'saml' ? 'SAML configuration' : 'OAuth/OIDC configuration and endpoints';

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <AlertTriangle className="h-5 w-5" />
            Delete Provider
          </DialogTitle>
          <DialogDescription className="text-left">
            Are you sure you want to delete this provider? This action cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-3">
            <div className="text-sm">
              <div className="font-medium text-foreground">
                {providerName || "Unnamed Provider"}
              </div>
              <div className="font-mono text-xs text-foreground mt-1">
                ID: {providerId}
              </div>
              <div className="text-xs text-foreground mt-1">
                Type: {providerTypeLabel}
              </div>
            </div>
          </div>

          <div className="text-sm text-foreground">
            This will permanently remove:
            <ul className="list-disc list-inside mt-1 space-y-1">
              <li>All {configLabel}</li>
              <li>Registered applications and redirect URLs</li>
              <li>Token signing keys and client secrets</li>
              <li>User authentication flows using this provider</li>
              <li>Associated logs and audit trails</li>
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
            {isLoading ? "Deleting..." : "Delete Provider"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
