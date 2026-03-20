import * as React from "react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Check, ChevronLeft, ChevronRight } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

interface WizardContextType {
  currentStep: number;
  totalSteps: number;
  goToStep: (step: number) => void;
  nextStep: () => void;
  prevStep: () => void;
  isStepValid: (step: number) => boolean;
  registerStep: (index: number, isValid: boolean) => void;
}

const WizardContext = React.createContext<WizardContextType | undefined>(undefined);

export function useWizard() {
  const context = React.useContext(WizardContext);
  if (!context) {
    throw new Error("useWizard must be used within a WizardProvider");
  }
  return context;
}

interface WizardProps {
  children: React.ReactNode;
  defaultStep?: number;
  className?: string;
  onComplete?: () => void;
}

export function Wizard({ children, defaultStep = 0, className, onComplete }: WizardProps) {
  const [currentStep, setCurrentStep] = React.useState(defaultStep);
  const [stepsValidity, setStepsValidity] = React.useState<Record<number, boolean>>({});

  // Count total steps (children that are WizardStep)
  const steps = React.Children.toArray(children).filter(
    (child) => React.isValidElement(child) && child.type === WizardStep
  );
  const totalSteps = steps.length;

  const registerStep = React.useCallback((index: number, isValid: boolean) => {
    setStepsValidity((prev) => {
      if (prev[index] === isValid) return prev;
      return { ...prev, [index]: isValid };
    });
  }, []);

  const isStepValid = React.useCallback(
    (step: number) => {
      return stepsValidity[step] ?? true;
    },
    [stepsValidity]
  );

  const goToStep = (step: number) => {
    if (step >= 0 && step < totalSteps) {
      setCurrentStep(step);
    }
  };

  const nextStep = () => {
    if (currentStep < totalSteps - 1) {
      setCurrentStep((prev) => prev + 1);
    } else {
      onComplete?.();
    }
  };

  const prevStep = () => {
    if (currentStep > 0) {
      setCurrentStep((prev) => prev - 1);
    }
  };

  return (
    <WizardContext.Provider
      value={{
        currentStep,
        totalSteps,
        goToStep,
        nextStep,
        prevStep,
        isStepValid,
        registerStep,
      }}
    >
      <div className={cn("space-y-6", className)}>{children}</div>
    </WizardContext.Provider>
  );
}

interface WizardHeaderProps {
  children?: React.ReactNode;
  className?: string;
}

export function WizardHeader({ children, className }: WizardHeaderProps) {
  const { currentStep, totalSteps, goToStep, isStepValid } = useWizard();

  // If children are provided, render them (custom header)
  if (children) {
    return <div className={className}>{children}</div>;
  }

  // Default Stepper Header
  return (
    <div className={cn("w-full py-4", className)}>
      <div className="flex items-center justify-between relative">
        {/* Progress Bar Background */}
        <div className="absolute top-1/2 left-0 w-full h-1 bg-muted -z-10 rounded-full" />
        
        {/* Progress Bar Fill */}
        <div 
          className="absolute top-1/2 left-0 h-1 bg-primary -z-10 rounded-full transition-all duration-300 ease-in-out"
          style={{ width: `${(currentStep / (totalSteps - 1)) * 100}%` }}
        />

        {Array.from({ length: totalSteps }).map((_, index) => {
          const isActive = index === currentStep;
          const isCompleted = index < currentStep;
          const isClickable = isCompleted || index === currentStep + 1; // Allow clicking next immediate step if current is valid? Maybe restrict to completed only.

          return (
            <button
              key={index}
              type="button"
              onClick={() => isCompleted && goToStep(index)}
              disabled={!isCompleted && !isActive}
              className={cn(
                "relative flex items-center justify-center w-10 h-10 rounded-full border-2 transition-all duration-200 bg-background",
                isActive
                  ? "border-primary text-primary ring-4 ring-primary/10 scale-110"
                  : isCompleted
                  ? "border-primary bg-primary text-primary-foreground"
                  : "border-muted text-muted-foreground",
                isCompleted && "cursor-pointer hover:opacity-80"
              )}
            >
              {isCompleted ? (
                <Check className="w-5 h-5" />
              ) : (
                <span className="text-sm font-semibold">{index + 1}</span>
              )}
              
              {/* Step Label (Optional - could be passed via props) */}
              {/* <span className="absolute -bottom-6 text-xs font-medium whitespace-nowrap">
                Step {index + 1}
              </span> */}
            </button>
          );
        })}
      </div>
    </div>
  );
}

interface WizardStepProps {
  children: React.ReactNode;
  title?: string;
  description?: string;
  isValid?: boolean;
}

export function WizardStep({ children, title, description, isValid = true }: WizardStepProps) {
  const { currentStep, registerStep } = useWizard();
  
  // We need to know which index this step is. 
  // Since we can't easily get index from context without registering, 
  // we rely on the parent rendering only the active step or all steps hidden.
  // A better approach for animations is to render all but hide/show.
  
  // However, for simplicity in usage, let's assume Wizard renders all children 
  // and we control visibility here based on order.
  // BUT, React.Children.map in Wizard is better for injecting index.
  
  // Let's change approach: Wizard renders the ACTIVE step content.
  // But to support animations, we might want AnimatePresence.
  
  return (
    <div className="space-y-6">
      {(title || description) && (
        <div className="space-y-1 mb-6">
          {title && <h2 className="text-2xl font-semibold tracking-tight">{title}</h2>}
          {description && <p className="text-muted-foreground">{description}</p>}
        </div>
      )}
      {children}
    </div>
  );
}

// Wrapper to handle step logic and animations
export function WizardContent({ children }: { children: React.ReactNode }) {
  const { currentStep, registerStep } = useWizard();
  const steps = React.Children.toArray(children);
  
  // Register validity for the current step
  const activeChild = steps[currentStep];
  if (React.isValidElement(activeChild) && activeChild.type === WizardStep) {
    // We can't easily extract props here to register validity without effects in the child.
    // So we'll let the user pass `isValid` to WizardStep, but we need to capture it.
    // Actually, `registerStep` should be called by `WizardStep` effect.
  }

  return (
    <div className="min-h-[400px] relative">
      <AnimatePresence mode="wait">
        <motion.div
          key={currentStep}
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: -20 }}
          transition={{ duration: 0.2 }}
          className="w-full"
        >
          {steps[currentStep]}
        </motion.div>
      </AnimatePresence>
    </div>
  );
}

interface WizardFooterProps {
  children?: React.ReactNode;
  nextLabel?: string;
  backLabel?: string;
  finishLabel?: string;
  onNext?: () => void;
  onBack?: () => void;
  isNextDisabled?: boolean;
  isLoading?: boolean;
}

export function WizardFooter({
  children,
  nextLabel = "Next",
  backLabel = "Back",
  finishLabel = "Complete",
  onNext,
  onBack,
  isNextDisabled,
  isLoading,
}: WizardFooterProps) {
  const { currentStep, totalSteps, nextStep, prevStep } = useWizard();
  const isLastStep = currentStep === totalSteps - 1;
  const isFirstStep = currentStep === 0;

  const handleNext = () => {
    if (onNext) {
      onNext();
    } else {
      nextStep();
    }
  };

  const handleBack = () => {
    if (onBack) {
      onBack();
    } else {
      prevStep();
    }
  };

  return (
    <div className="flex items-center justify-between pt-8 border-t mt-8">
      <div>
        {!isFirstStep && (
          <Button variant="outline" onClick={handleBack} disabled={isLoading}>
            <ChevronLeft className="mr-2 h-4 w-4" />
            {backLabel}
          </Button>
        )}
      </div>
      <div className="flex items-center gap-2">
        {children}
        <Button 
          onClick={handleNext} 
          disabled={isNextDisabled || isLoading}
          className="min-w-[100px]"
        >
          {isLoading ? (
            "Loading..."
          ) : isLastStep ? (
            <>
              {finishLabel}
              <Check className="ml-2 h-4 w-4" />
            </>
          ) : (
            <>
              {nextLabel}
              <ChevronRight className="ml-2 h-4 w-4" />
            </>
          )}
        </Button>
      </div>
    </div>
  );
}
