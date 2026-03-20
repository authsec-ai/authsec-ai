import { Button } from "@/components/ui/button";
import { X, Sparkles } from "lucide-react";

interface WizardHeaderProps {
  title: string;
  currentStep: number;
  totalSteps: number;
  onClose: () => void;
}

export function WizardHeader({
  title,
  currentStep,
  totalSteps,
  onClose,
}: WizardHeaderProps) {
  return (
    <div className="flex items-center justify-between px-4 py-3 border-b border-border bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="flex items-center gap-3">
        <div className="flex items-center justify-center w-7 h-7 rounded-md bg-primary/10">
          <Sparkles className="h-4 w-4 text-primary" />
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
  );
}
