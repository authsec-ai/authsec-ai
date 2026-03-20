import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";

interface RevokeDelegationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  delegationLabel?: string;
  onConfirm: (reason?: string) => Promise<void> | void;
  isLoading?: boolean;
}

export function RevokeDelegationDialog({
  open,
  onOpenChange,
  delegationLabel,
  onConfirm,
  isLoading = false,
}: RevokeDelegationDialogProps) {
  const [reason, setReason] = useState("");

  const handleConfirm = async () => {
    await onConfirm(reason.trim() || undefined);
    setReason("");
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Revoke delegation</DialogTitle>
          <DialogDescription>
            Revoking {delegationLabel || "this delegation"} will stop future use
            under the delegated trust. Existing sessions may end immediately or
            on the next token refresh, depending on backend enforcement.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2">
          <Label htmlFor="revoke-reason">Reason (optional)</Label>
          <Input
            id="revoke-reason"
            placeholder="Example: policy no longer valid"
            value={reason}
            onChange={(event) => setReason(event.target.value)}
          />
        </div>

        <DialogFooter>
          <Button
            type="button"
            variant="outline"
            onClick={() => onOpenChange(false)}
          >
            Cancel
          </Button>
          <Button type="button" variant="destructive" onClick={handleConfirm} disabled={isLoading}>
            {isLoading ? "Revoking..." : "Revoke delegation"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
