import type { ReactNode } from "react";
import React from "react";
import type { LucideIcon } from "lucide-react";
import { ArrowLeft, Check, ChevronRight, Loader2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

export interface TrustDelegationWizardStep {
  id: string;
  label: string;
  icon: LucideIcon;
}

interface TrustDelegationWizardShellProps {
  title: string;
  description: string;
  steps: TrustDelegationWizardStep[];
  currentStepIndex: number;
  onClose: () => void;
  onBack: () => void;
  onPrimaryAction: () => void;
  primaryActionLabel: string;
  primaryActionIcon?: LucideIcon;
  primaryActionIconPosition?: "left" | "right";
  primaryActionDisabled?: boolean;
  primaryActionLoading?: boolean;
  primaryActionLoadingLabel?: string;
  children: ReactNode;
}

export function TrustDelegationWizardShell({
  title,
  description,
  steps,
  currentStepIndex,
  onClose,
  onBack,
  onPrimaryAction,
  primaryActionLabel,
  primaryActionIcon: PrimaryActionIcon,
  primaryActionIconPosition = "left",
  primaryActionDisabled = false,
  primaryActionLoading = false,
  primaryActionLoadingLabel,
  children,
}: TrustDelegationWizardShellProps) {
  return (
    <div className="flex h-[90vh] w-full flex-col">
      <div className="flex-shrink-0 border-b px-8 py-4">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <h2 className="text-lg font-semibold">{title}</h2>
            <p className="mt-0.5 text-xs text-muted-foreground">{description}</p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            className="h-8 w-8 rounded-full bg-red-50 text-red-600 hover:bg-red-100 hover:text-red-700 dark:bg-red-950 dark:text-red-400 dark:hover:bg-red-900 dark:hover:text-red-300"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-8 py-4">
        <div className="w-full max-w-6xl">{children}</div>
      </div>

      <div className="mt-auto flex-shrink-0 border-t bg-background px-8 pb-4 pt-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex min-w-[120px] items-center gap-2">
            <Button variant="outline" onClick={onBack} size="default">
              {currentStepIndex > 0 ? (
                <>
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back
                </>
              ) : (
                "Cancel"
              )}
            </Button>
          </div>

          <div className="flex flex-1 items-center justify-center gap-2">
            {steps.map((step, index) => {
              const StepIcon = step.icon;
              const isActive = index === currentStepIndex;
              const isCompleted = index < currentStepIndex;

              return (
                <React.Fragment key={step.id}>
                  <div
                    className={cn(
                      "flex items-center gap-2 rounded-lg px-3 py-2",
                      isActive && "bg-primary/10",
                      isCompleted && "opacity-60",
                    )}
                  >
                    <div
                      className={cn(
                        "flex h-6 w-6 items-center justify-center rounded-full text-xs",
                        isCompleted && "bg-primary text-primary-foreground",
                        isActive && "bg-primary/20 text-primary",
                        !isActive &&
                          !isCompleted &&
                          "bg-muted text-muted-foreground",
                      )}
                    >
                      {isCompleted ? (
                        <Check className="h-3 w-3" />
                      ) : (
                        <StepIcon className="h-3 w-3" />
                      )}
                    </div>
                    <span
                      className={cn(
                        "text-sm font-medium",
                        isActive && "text-foreground",
                        !isActive && "text-muted-foreground",
                      )}
                    >
                      {step.label}
                    </span>
                  </div>
                  {index < steps.length - 1 && (
                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  )}
                </React.Fragment>
              );
            })}
          </div>

          <div className="flex min-w-[160px] items-center justify-end gap-2">
            <Button
              onClick={onPrimaryAction}
              disabled={primaryActionDisabled || primaryActionLoading}
              size="default"
            >
              {primaryActionLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {primaryActionLoadingLabel || primaryActionLabel}
                </>
              ) : (
                <>
                  {PrimaryActionIcon && primaryActionIconPosition === "left" ? (
                    <PrimaryActionIcon className="mr-2 h-4 w-4" />
                  ) : null}
                  {primaryActionLabel}
                  {PrimaryActionIcon && primaryActionIconPosition === "right" ? (
                    <PrimaryActionIcon className="ml-2 h-4 w-4" />
                  ) : null}
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
