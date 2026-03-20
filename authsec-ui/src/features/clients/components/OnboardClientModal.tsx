import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Plus, Loader2 } from "lucide-react";
import { toast } from "react-hot-toast";
import { useRegisterClientMutation } from "@/app/api/clientApi";
import { SessionManager } from "@/utils/sessionManager";
import { useNavigate } from "react-router-dom";

interface OnboardClientModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess?: (clientId: string) => void;
  preventNavigation?: boolean;
}

export function OnboardClientModal({
  isOpen,
  onClose,
  onSuccess,
  preventNavigation = false,
}: OnboardClientModalProps) {
  const navigate = useNavigate();
  const [clientName, setClientName] = useState("");
  const [registerClient, { isLoading }] = useRegisterClientMutation();

  // Reset form when modal opens
  useEffect(() => {
    if (isOpen) {
      setClientName("");
    }
  }, [isOpen]);

  const handleCreateClient = async () => {
    if (!clientName.trim()) {
      toast.error("Please provide a client name");
      return;
    }

    try {
      // Get session data for tenant, project, and email
      const session = SessionManager.getSession();

      if (!session?.tenant_id || !session?.project_id) {
        toast.error("Missing tenant or project. Please sign in.");
        return;
      }

      const email = session.user?.email || session.user?.email_id || "";

      if (!email) {
        toast.error("Missing email from session");
        return;
      }

      // Create the client
      const response = await registerClient({
        name: clientName,
        email,
        tenant_id: session.tenant_id,
        project_id: session.project_id,
        react_app_url: window.location.origin,
      }).unwrap();

      // Show success toast with client ID
      toast.success(`Client created with ID: ${response.client_id}`);

      // Close modal
      onClose();

      // Call success callback with client ID
      onSuccess?.(response.client_id);

      // Navigate to SDK integration page (unless prevented)
      if (!preventNavigation) {
        navigate(`/sdk/clients/${response.client_id}`);
      }
    } catch (err: any) {
      console.error(err);
      const msg = err?.data?.message || err?.error || "Failed to create client";
      toast.error(msg);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !isLoading && clientName.trim()) {
      handleCreateClient();
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle className="text-2xl font-bold">
            Onboard New Client
          </DialogTitle>
          <DialogDescription>
            Create a new MCP Server or AI Agent client
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="space-y-2">
            <Label htmlFor="clientName" className="text-sm font-semibold">
              Client Name
            </Label>
            <Input
              id="clientName"
              placeholder="e.g., My MCP Server"
              value={clientName}
              onChange={(e) => setClientName(e.target.value)}
              onKeyPress={handleKeyPress}
              className="h-12 text-base"
              autoFocus
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isLoading}>
            Cancel
          </Button>
          <Button
            onClick={handleCreateClient}
            disabled={!clientName.trim() || isLoading}
            className="min-w-[140px]"
          >
            {isLoading ? (
              <>
                <Loader2 className="h-4 w-4 animate-spin" />
                Onboarding...
              </>
            ) : (
              <>
                <Plus className="h-4 w-4" />
                Onboard
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
