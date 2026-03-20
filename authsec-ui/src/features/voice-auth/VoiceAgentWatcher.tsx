import { useEffect, useMemo, useState } from "react";
import { PhoneCall, ShieldCheck, Clock3, AlertCircle } from "lucide-react";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { toast } from "react-hot-toast";
import { useAuth } from "@/auth/context/AuthContext";
import { SessionManager } from "@/utils/sessionManager";
import {
  useApproveVoiceRequestMutation,
  useGetPendingVoiceRequestsQuery,
  type VoiceDeviceRequest,
} from "@/app/api/voiceAgentApi";

const formatTimestamp = (value?: number) => {
  if (!value) return "—";
  const milliseconds = value > 1e12 ? value : value * 1000;
  return new Date(milliseconds).toLocaleString();
};

export function VoiceAgentWatcher() {
  const { isAuthenticated } = useAuth();
  const [clientId, setClientId] = useState<string>("");
  const [activeRequest, setActiveRequest] = useState<VoiceDeviceRequest | null>(null);

  useEffect(() => {
    const syncClientId = () => {
      const session = SessionManager.getSession();
      setClientId(session?.client_id || "");
    };

    syncClientId();

    const handleStorage = (event: StorageEvent) => {
      if (event.key === "authsec_session_v2") {
        syncClientId();
      }
    };

    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, [isAuthenticated]);

  const shouldPoll = useMemo(() => isAuthenticated && Boolean(clientId), [clientId, isAuthenticated]);

  const { data, isFetching, refetch } = useGetPendingVoiceRequestsQuery(
    { clientId },
    {
      skip: !shouldPoll,
      pollingInterval: 3000,
      refetchOnFocus: true,
      refetchOnReconnect: true,
    }
  );

  useEffect(() => {
    if (data?.requests?.length) {
      const pending = data.requests.find((request) => request.status === "pending") ?? data.requests[0];
      if (!activeRequest || activeRequest.id !== pending.id) {
        setActiveRequest(pending);
      }
    } else if (!isFetching) {
      setActiveRequest(null);
    }
  }, [data, activeRequest, isFetching]);

  const [approveRequest, { isLoading: isUpdating }] = useApproveVoiceRequestMutation();

  const handleDecision = async (approve: boolean) => {
    if (!activeRequest) return;
    try {
      await approveRequest({ user_code: activeRequest.user_code, approve }).unwrap();
      toast.success(approve ? "Voice agent request approved" : "Voice agent request denied");
      setActiveRequest(null);
      refetch();
    } catch (error: any) {
      const message = error?.data?.message || "Failed to update voice agent request";
      toast.error(message);
    }
  };

  if (!shouldPoll) return null;

  return (
    <Dialog open={Boolean(activeRequest)} onOpenChange={(open) => !open && setActiveRequest(null)}>
      <DialogContent className="max-w-xl">
        <DialogHeader className="space-y-2">
          <div className="flex items-center gap-2">
            <PhoneCall className="h-5 w-5 text-primary" />
            <DialogTitle>Approve voice agent sign-in</DialogTitle>
          </div>
          <DialogDescription>
            A voice assistant is requesting access. Confirm the user code or deny the request.
          </DialogDescription>
        </DialogHeader>

        {activeRequest && (
          <div className="space-y-4">
           

            <div className="rounded-lg border bg-background px-4 py-3 text-xs text-foreground">
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Clock3 className="h-4 w-4" />
                  <span>Requested {formatTimestamp(activeRequest.created_at)}</span>
                </div>
                <span>Expires {formatTimestamp(activeRequest.expires_at)}</span>
              </div>
            </div>

            <div className="flex items-center justify-end gap-3">
              <Button
                variant="outline"
                onClick={() => handleDecision(false)}
                disabled={isUpdating}
              >
                Deny
              </Button>
              <Button onClick={() => handleDecision(true)} disabled={isUpdating}>
                Approve
              </Button>
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  );
}
