import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Check, ChevronRight } from "lucide-react";
import { cn } from "@/lib/utils";
import type { WizardStep } from "../types";
import { SDKIntegrationDialog } from "./SDKIntegrationDialog";
import { useNavigate, useLocation } from "react-router-dom";

interface WizardStepContentProps {
  step: WizardStep;
  stepStatus: "pending" | "in-progress" | "completed";
  onCompleteStep: (stepId: string) => void;
}

export function WizardStepContent({
  step,
  stepStatus,
  onCompleteStep,
}: WizardStepContentProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const [sdkDialogOpen, setSdkDialogOpen] = useState(false);

  const handleAction = () => {
    switch (step.actionType) {
      case "navigate":
        // Navigate to the route with current location in state
        if (step.actionPayload.route) {
          navigate(step.actionPayload.route, {
            state: { from: location.pathname }
          });
        }
        break;

      case "dialog":
        // Open dialog (for SDK integration)
        setSdkDialogOpen(true);
        break;

      case "custom":
        // Handle custom actions
        console.log("[Wizard] Custom action:", step.actionPayload.handler);
        break;

      default:
        console.warn(`[Wizard] Unknown action type: ${step.actionType}`);
    }
  };

  const handleDialogClose = (isOpen: boolean) => {
    setSdkDialogOpen(isOpen);

    // Auto-complete step when dialog closes (if trigger is 'auto')
    if (!isOpen && step.completionTrigger === "auto" && stepStatus !== "completed") {
      onCompleteStep(step.id);
    }
  };

  const isCompleted = stepStatus === "completed";

  return (
    <>
      <div className="p-6">
        <div className="max-w-2xl mx-auto space-y-6">
          {/* Step Icon & Status */}
          <div className="flex items-center gap-4">
            <div
              className={cn(
                "flex-shrink-0 w-12 h-12 rounded-lg flex items-center justify-center transition-all duration-200",
                isCompleted
                  ? "bg-primary/10 text-primary"
                  : "bg-muted/50 text-muted-foreground"
              )}
            >
              {isCompleted ? (
                <Check className="h-6 w-6" />
              ) : (
                step.icon || <ChevronRight className="h-6 w-6" />
              )}
            </div>
            <div className="flex-1">
              <h3
                className={cn(
                  "text-lg font-semibold",
                  isCompleted && "text-primary"
                )}
              >
                {step.title}
              </h3>
              {isCompleted && (
                <p className="text-sm text-primary flex items-center gap-1 mt-0.5">
                  <Check className="h-3 w-3" />
                  Completed
                </p>
              )}
            </div>
          </div>

          {/* Step Description */}
          <div className="space-y-3">
            <p className="text-sm text-muted-foreground leading-relaxed">
              {step.description}
            </p>
          </div>

          {/* Action Button */}
          {!isCompleted && (
            <div className="pt-2">
              <Button
                onClick={handleAction}
                size="default"
              >
                {step.actionLabel}
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            </div>
          )}

          {/* Completed State */}
          {isCompleted && (
            <div className="pt-2">
              <div className="w-full p-4 rounded-lg bg-primary/5 border border-primary/20">
                <div className="flex items-center gap-2 text-primary">
                  <Check className="h-5 w-5" />
                  <span className="text-sm font-medium">
                    Step completed successfully!
                  </span>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* SDK Integration Dialog */}
      {step.actionType === "dialog" && step.actionPayload.contentId === "sdk-attestation" && (
        <SDKIntegrationDialog
          open={sdkDialogOpen}
          onOpenChange={handleDialogClose}
        />
      )}
    </>
  );
}
