import { useState, useEffect } from "react";
import { Check, Sparkles, X, ChevronRight } from "lucide-react";
import { useNavigate, useLocation } from "react-router-dom";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import type { WizardStep } from "../types";
import { SDKIntegrationDialog } from "./SDKIntegrationDialog";
import {
  ClientSelectionStep,
  ChooseAuthMethodStep,
  ConfigureAuthStep,
} from "./UserAuthWizardSteps";
import { CheckSPIREAgentStep } from "./M2MWizardSteps";
import { ContextSelectionStep } from "./RbacWizardSteps";
import { useWizard } from "@/contexts/WizardContext";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";

interface WizardProgressProps {
  totalSteps: number;
  currentStep: number;
  completedSteps: string[];
  steps: WizardStep[];
  title: string;
  onClose: () => void;
  onCompleteStep: (stepId: string) => void;
  getStepStatus: (stepId: string) => "pending" | "in-progress" | "completed";
}

export function WizardProgress({
  totalSteps,
  currentStep,
  completedSteps,
  steps,
  title,
  onClose,
  onCompleteStep,
  getStepStatus,
}: WizardProgressProps) {
  const navigate = useNavigate();
  const contextualNavigate = useContextualNavigate();
  const location = useLocation();
  const { activeWizard, setWizardCompletionData, completeWizard, setIsAwaitingPlatformAction } = useWizard();
  const [sdkDialogOpen, setSdkDialogOpen] = useState(false);
  const [wizardStepData, setWizardStepData] = useState<Record<string, any>>({});
  const [actionTakenForStep, setActionTakenForStep] = useState<string | null>(null);

  // Reset actionTakenForStep when the step changes
  useEffect(() => {
    setActionTakenForStep(null);
  }, [currentStep]);

  // Sync wizardStepData to context whenever it changes
  useEffect(() => {
    if (Object.keys(wizardStepData).length > 0) {
      console.log("[WizardProgress] Syncing wizardStepData to context:", wizardStepData);
      setWizardCompletionData(wizardStepData);
    }
  }, [wizardStepData, setWizardCompletionData]);

  // Helper to update step data and complete step
  const handleStepCompletion = (stepId: string, data: Record<string, any>) => {
    const updatedData = { ...wizardStepData, ...data };
    console.log("[WizardProgress] Updating step data:", updatedData);
    setWizardStepData(updatedData);
    setWizardCompletionData(updatedData);
    onCompleteStep(stepId);
  };

  const handleAction = (step: WizardStep) => {
    switch (step.actionType) {
      case "navigate":
        if (step.actionPayload.route) {
          // Check if external URL
          if (step.actionPayload.route.startsWith("http")) {
            window.open(step.actionPayload.route, "_blank");
          } else {
            // Replace :clientId with actual clientId from wizardStepData
            let route = step.actionPayload.route;
            if (route.includes(":clientId") && wizardStepData.clientId) {
              route = route.replace(":clientId", wizardStepData.clientId);
            }

            // Check if this wizard uses context-aware navigation (RBAC or Scopes wizard)
            // These wizards have "select-context" as their first step
            const usesContextAwareNav = steps[0]?.id === "select-context";

            // Determine wizard ID from the wizard configuration
            // For context-aware wizards, we need to determine which one it is
            let activeWizardId = "user-auth-wizard"; // default
            if (usesContextAwareNav) {
              // Check if this is RBAC wizard (has 3+ steps with permissions/roles/bindings)
              // or Scopes wizard (has 2 steps with create-scope)
              const hasCreateScope = steps.some(s => s.id === "create-scope");
              activeWizardId = hasCreateScope ? "scopes-wizard" : "rbac-wizard";
            }

            const navState = {
              from: location.pathname,
              fromWizard: true,
              wizardId: activeWizardId,
            };
            console.log("[WizardProgress] Navigating with state:", navState);

            if (step.completionTrigger === "navigation-return") {
              setIsAwaitingPlatformAction(true);
            }

            if (usesContextAwareNav) {
              contextualNavigate(route, { state: navState });
            } else {
              navigate(route, { state: navState });
            }

            setActionTakenForStep(step.id);
          }
        }
        break;

      case "dialog":
        setSdkDialogOpen(true);
        break;

      case "custom":
        // Custom handlers are now rendered inline, no action needed
        console.log("[Wizard] Custom step - inline UI rendered");
        break;

      default:
        console.warn(`[Wizard] Unknown action type: ${step.actionType}`);
    }
  };

  const handleDialogClose = (isOpen: boolean) => {
    setSdkDialogOpen(isOpen);

    // Auto-complete step when dialog closes (if trigger is 'auto')
    const currentStepConfig = steps[currentStep];
    if (
      !isOpen &&
      currentStepConfig?.completionTrigger === "auto" &&
      getStepStatus(currentStepConfig.id) !== "completed"
    ) {
      onCompleteStep(currentStepConfig.id);
    }
  };

  return (
    <>
      <div className="w-full bg-background border-b border-border">
        {/* Header Section */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
          <div className="flex items-center gap-3">
            <div className="w-7 h-7 rounded-md bg-[var(--brand-blue-600)]/10 flex items-center justify-center">
              <Sparkles className="h-4 w-4 text-[var(--brand-blue-600)]" />
            </div>
            <div>
              <h3 className="font-medium text-sm">{title}</h3>
              <p className="text-xs text-muted-foreground">
                Step {currentStep + 1} of {totalSteps}
              </p>
            </div>
          </div>
          <Button
            variant="ghost"
            size="sm"
            onClick={onClose}
            className="h-7 w-7 p-0 hover:bg-muted rounded-md"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>

        {/* Steps List - Vertical Timeline with Inline Content */}
        <div className="px-6 py-4">
          {steps.map((step, index) => {
            const isCompleted = completedSteps.includes(step.id);
            const isCurrent = index === currentStep;
            const isPending = !isCompleted && !isCurrent;
            const stepStatus = getStepStatus(step.id);

            return (
              <div key={step.id} className="relative flex gap-4 pb-8 last:pb-4">
                {/* Left: Circle with Vertical Line */}
                <div className="flex flex-col items-center">
                  {/* Circle Indicator */}
                  <div
                    className={cn(
                      "w-12 h-12 rounded-full flex items-center justify-center text-sm font-semibold transition-all duration-200 shrink-0 z-10",
                      isCompleted &&
                        "bg-[var(--brand-blue-600)] text-white shadow-md",
                      isCurrent &&
                        "bg-[var(--brand-blue-600)] text-white shadow-md",
                      isPending &&
                        "bg-muted text-muted-foreground border-2 border-border"
                    )}
                  >
                    {isCompleted ? (
                      <Check className="h-5 w-5" />
                    ) : (
                      <span>{index + 1}</span>
                    )}
                  </div>

                  {/* Vertical Connector Line */}
                  {index < totalSteps - 1 && (
                    <div
                      className={cn(
                        "w-0.5 flex-1 mt-2 transition-all duration-300",
                        "absolute top-12 bottom-0 left-6 transform -translate-x-1/2",
                        isCompleted ? "bg-[var(--brand-blue-600)]" : "bg-border"
                      )}
                    />
                  )}
                </div>

                {/* Right: Step Content */}
                <div className="flex-1 min-w-0">
                  {/* Step Title & Brief Description */}
                  <div className="pt-2">
                    <h4
                      className={cn(
                        "text-base font-semibold mb-1",
                        isCurrent && "text-foreground",
                        isCompleted && "text-foreground",
                        isPending && "text-muted-foreground"
                      )}
                    >
                      {step.title}
                    </h4>

                    {step.briefDescription && (
                      <p
                        className={cn(
                          "text-sm leading-relaxed",
                          isCurrent && "text-muted-foreground",
                          isCompleted && "text-muted-foreground",
                          isPending && "text-muted-foreground/70"
                        )}
                      >
                        {step.briefDescription}
                      </p>
                    )}
                  </div>

                  {/* Inline Step Content - Only for Current Step */}
                  {isCurrent && (
                    <div className="mt-4 space-y-4">
                      {/* Full Description */}
                      {/* <div>
                        <p className="text-sm text-muted-foreground leading-relaxed">
                          {step.description}
                        </p>
                      </div> */}

                      {/* User Auth Wizard Custom Steps */}
                      {step.actionType === "custom" &&
                        step.actionPayload.handler === "client-selection" && (
                          <ClientSelectionStep
                            onComplete={(clientId) => {
                              handleStepCompletion(step.id, { clientId });
                            }}
                          />
                        )}

                      {step.actionType === "custom" &&
                        step.actionPayload.handler === "choose-auth-method" && (
                          <ChooseAuthMethodStep
                            onComplete={(methodType) => {
                              handleStepCompletion(step.id, { authMethodType: methodType });
                            }}
                          />
                        )}

                      {step.actionType === "custom" &&
                        step.actionPayload.handler === "configure-auth" && (
                          <ConfigureAuthStep
                            authMethodType={
                              wizardStepData.authMethodType || "oidc"
                            }
                            onNavigate={(provider) => {
                              const route =
                                wizardStepData.authMethodType === "saml2"
                                  ? "/authentication/saml/create"
                                  : `/authentication/create?provider=${provider}`;
                              setIsAwaitingPlatformAction(true);
                              navigate(route, {
                                state: { from: location.pathname, fromWizard: true },
                              });
                            }}
                            onComplete={() => {
                              handleStepCompletion(step.id, {});
                            }}
                          />
                        )}

                      {/* M2M Wizard Custom Steps */}
                      {step.actionType === "custom" &&
                        step.actionPayload.handler === "check-spire-agent" && (
                          <CheckSPIREAgentStep
                            onComplete={() => {
                              handleStepCompletion(step.id, {});
                            }}
                          />
                        )}

                      {/* RBAC Wizard Custom Steps */}
                      {step.actionType === "custom" &&
                        step.actionPayload.handler === "select-context" && (
                          <ContextSelectionStep
                            onComplete={(context) => {
                              handleStepCompletion(step.id, { selectedContext: context });
                            }}
                          />
                        )}

                      {/* Action Button - hide for custom steps */}
                      {!isCompleted && step.actionType !== "custom" && (
                        <div className="flex flex-col items-center gap-2">
                          <Button
                            onClick={() => handleAction(step)}
                            size="sm"
                            variant="outline"
                            className="bg-[var(--brand-blue-600)] text-white hover:bg-[var(--brand-blue-700)] shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
                          >
                            {step.actionLabel}
                            <ChevronRight className="ml-2 h-3 w-3" />
                          </Button>

                          {actionTakenForStep === step.id && (
                            <Button
                              onClick={() => {
                                onCompleteStep(step.id);
                                navigate("/dashboard");
                              }}
                              size="sm"
                              variant="outline"
                              className="bg-[var(--brand-blue-600)] text-white hover:bg-[var(--brand-blue-700)] shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
                            >
                              <Check className="mr-2 h-3 w-3" />
                              Complete
                            </Button>
                          )}
                        </div>
                      )}

                      {/* Completed State */}
                      {isCompleted && (
                        <div className="p-3 rounded-lg bg-[var(--brand-blue-600)]/5 border border-[var(--brand-blue-600)]/20">
                          <div className="flex items-center gap-2 text-[var(--brand-blue-600)]">
                            <Check className="h-4 w-4" />
                            <span className="text-sm font-medium">
                              Step completed successfully!
                            </span>
                          </div>
                        </div>
                      )}
                    </div>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* SDK Integration Dialog */}
      {sdkDialogOpen && (
        <SDKIntegrationDialog
          open={sdkDialogOpen}
          onOpenChange={handleDialogClose}
        />
      )}
    </>
  );
}
