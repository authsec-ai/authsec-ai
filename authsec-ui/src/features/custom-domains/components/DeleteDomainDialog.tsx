import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { AlertTriangle, Globe, Loader2 } from "lucide-react";
import { useDeleteDomainMutation } from "@/app/api/domainApi";
import { SessionManager } from "@/utils/sessionManager";
import { toast } from "@/lib/toast";
import type { CustomDomain } from "@/app/api/domainApi";

interface DeleteDomainDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  domain: CustomDomain | null;
  onSuccess?: () => void;
}

export function DeleteDomainDialog({
  open,
  onOpenChange,
  domain,
  onSuccess,
}: DeleteDomainDialogProps) {
  const [deleteDomain, { isLoading }] = useDeleteDomainMutation();

  const handleDelete = async () => {
    if (!domain) return;

    const session = SessionManager.getSession();
    if (!session?.tenant_id) {
      toast.error("Session expired. Please log in again.");
      return;
    }

    try {
      await deleteDomain({
        tenant_id: session.tenant_id,
        domain_id: domain.id,
      }).unwrap();

      toast.success(`Domain "${domain.domain}" deleted successfully`);
      onOpenChange(false);
      onSuccess?.();
    } catch (error: any) {
      const message =
        error?.data?.error ||
        error?.data?.message ||
        "Failed to delete domain. Please try again.";
      toast.error(message);
    }
  };

  if (!domain) return null;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2 text-destructive">
            <AlertTriangle className="h-5 w-5" />
            Delete Domain
          </DialogTitle>
          <DialogDescription>
            Are you sure you want to delete this domain? This action cannot be
            undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Domain Info */}
          <div className="rounded-md bg-destructive/10 border border-destructive/20 p-4">
            <div className="flex items-center gap-2">
              <Globe className="h-4 w-4 text-muted-foreground" />
              <span className="font-medium">{domain.domain}</span>
            </div>
            <div className="mt-2 text-xs text-muted-foreground font-mono">
              ID: {domain.id}
            </div>
          </div>

          {/* Warning List */}
          <div className="text-sm">
            <p className="font-medium mb-2">This will permanently remove:</p>
            <ul className="list-disc list-inside space-y-1 text-muted-foreground">
              <li>All DNS verification records</li>
              <li>Domain configuration and settings</li>
              {domain.is_primary && (
                <li className="text-destructive font-medium">
                  Primary domain status (authentication may be affected)
                </li>
              )}
            </ul>
          </div>

          {/* Primary Domain Warning */}
          {domain.is_primary && (
            <div className="rounded-md bg-yellow-500/10 border border-yellow-500/30 p-3">
              <p className="text-sm text-yellow-600 dark:text-yellow-400">
                <AlertTriangle className="h-4 w-4 inline mr-1" />
                <strong>Warning:</strong> This is your primary domain. Deleting
                it may affect authentication flows until you set another domain
                as primary.
              </p>
            </div>
          )}
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={isLoading}
          >
            Cancel
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={handleDelete}
            disabled={isLoading}
          >
            {isLoading ? (
              <>
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                Deleting...
              </>
            ) : (
              "Delete Domain"
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
