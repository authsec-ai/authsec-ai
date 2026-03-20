import { useState, useEffect } from "react";
import { Button } from "@/components/ui/button";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Server, AlertCircle, CheckCircle, ChevronRight, RefreshCw } from "lucide-react";
import { useListAgentsQuery } from "@/app/api/workloadsApi";
import { SPIREAgentFAQDialog } from "./SPIREAgentFAQDialog";

interface CheckSPIREAgentStepProps {
  onComplete: () => void;
}

export function CheckSPIREAgentStep({ onComplete }: CheckSPIREAgentStepProps) {
  // State management
  const [showFAQDialog, setShowFAQDialog] = useState(false);
  const [hasCheckedOnce, setHasCheckedOnce] = useState(false);
  const [showError, setShowError] = useState(false);

  // API query
  const {
    data: agents,
    isLoading,
    refetch,
    isFetching,
  } = useListAgentsQuery(undefined, {
    refetchOnMountOrArgChange: true,
  });

  const agentsAvailable = agents && agents.length > 0;

  // Auto-skip logic: If agents exist on mount, automatically complete
  useEffect(() => {
    if (!isLoading && agentsAvailable && !hasCheckedOnce) {
      console.log("[M2M Wizard] Agents detected, auto-skipping to next step");
      setHasCheckedOnce(true); // Prevent re-triggering
      onComplete();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isLoading, agentsAvailable, hasCheckedOnce]);

  // Handle "Done" button click from FAQ dialog
  const handleDoneClick = async () => {
    setShowFAQDialog(false);
    setHasCheckedOnce(true);
    setShowError(false);

    // Re-check agent availability
    const result = await refetch();

    if (result.data && result.data.length > 0) {
      // Success: agents detected, proceed
      console.log("[M2M Wizard] Agents detected after setup, proceeding");
      onComplete();
    } else {
      // Still no agents: show error
      setShowError(true);
    }
  };

  // Handle "View Setup Guide" button
  const handleViewGuide = () => {
    setShowFAQDialog(true);
    setShowError(false);
  };

  // Handle "Recheck" button on the error alert
  const handleRecheck = async () => {
    setShowError(false);
    const result = await refetch();
    if (result.data && result.data.length > 0) {
      onComplete();
    } else {
      setShowError(true);
    }
  };

  // If loading on initial mount, show loading state
  if (isLoading && !hasCheckedOnce) {
    return (
      <div className="flex items-center justify-center p-6">
        <div className="flex items-center gap-2 text-muted-foreground">
          <Server className="h-4 w-4 animate-pulse" />
          <span className="text-sm">Checking for SPIRE agents...</span>
        </div>
      </div>
    );
  }

  // If no agents found (or after failed re-check)
  if (!agentsAvailable) {
    return (
      <div className="space-y-4">
        {/* Info box */}
        <Alert>
          <Server className="h-4 w-4" />
          <AlertDescription>
            Before creating an M2M workload, you need at least one SPIRE agent
            deployed in your infrastructure to attest workload identities.
          </AlertDescription>
        </Alert>

        {/* Error message if user clicked "Done" but still no agents */}
        {showError && (
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription className="flex items-center justify-between gap-3">
              <span>
                No SPIRE agents detected. Please ensure your agent is properly
                deployed and running, then try again.
              </span>
              <Button
                size="sm"
                variant="outline"
                className="shrink-0 h-7 px-2 text-xs border-destructive text-destructive hover:bg-destructive/10"
                onClick={handleRecheck}
                disabled={isFetching}
              >
                <RefreshCw className={`h-3 w-3 mr-1 ${isFetching ? "animate-spin" : ""}`} />
                Recheck
              </Button>
            </AlertDescription>
          </Alert>
        )}

        {/* Status display */}
        <div className="p-4 rounded-lg border border-border bg-muted/30">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-full bg-yellow-100 dark:bg-yellow-900/30">
              <Server className="h-5 w-5 text-yellow-600 dark:text-yellow-400" />
            </div>
            <div className="flex-1">
              <h4 className="font-semibold text-sm">SPIRE Agent Required</h4>
              <p className="text-xs text-muted-foreground mt-1">
                {hasCheckedOnce
                  ? "Still no agents detected. Please check your deployment."
                  : "No SPIRE agents currently detected"}
              </p>
            </div>
          </div>
        </div>

        {/* Action button */}
        <div className="flex justify-center">
          <Button
            onClick={handleViewGuide}
            variant="outline"
            className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
          >
            View SPIRE Agent Setup Guide
            <ChevronRight className="ml-2 h-3 w-3" />
          </Button>
        </div>

        {/* FAQ Dialog */}
        <SPIREAgentFAQDialog
          open={showFAQDialog}
          onDone={handleDoneClick}
          isFetching={isFetching}
        />
      </div>
    );
  }

  // This component should auto-complete when agents exist,
  // so this return should rarely be reached
  return (
    <div className="space-y-4">
      <Alert>
        <CheckCircle className="h-4 w-4 text-green-600" />
        <AlertDescription>
          SPIRE agents detected! Proceeding to workload creation...
        </AlertDescription>
      </Alert>
    </div>
  );
}
