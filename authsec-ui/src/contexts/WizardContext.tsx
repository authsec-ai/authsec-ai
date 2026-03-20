import {
  createContext,
  useContext,
  useState,
  useEffect,
  useRef,
  type ReactNode,
} from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { WizardStorage } from "@/features/wizards/utils/wizardStorage";
import { getWizardConfig } from "@/features/wizards/configs";
import type { WizardConfig, WizardStep } from "@/features/wizards/types";

interface WizardContextValue {
  // State
  activeWizard: string | null;
  currentStep: number;
  isActive: boolean;
  isCompleted: boolean;
  completedSteps: string[];
  wizardConfig: WizardConfig | null;
  wizardCompletionData: Record<string, any> | null;

  // Actions
  startWizard: (wizardId: string, config: WizardConfig) => void;
  completeStep: (stepId: string) => void;
  nextStep: () => void;
  previousStep: () => void;
  skipWizard: () => void;
  dismissWizard: () => void;
  completeWizard: (data?: Record<string, any>) => void;
  resetCompletion: () => void;
  setWizardCompletionData: (data: Record<string, any>) => void;

  // Platform action state
  isAwaitingPlatformAction: boolean;
  setIsAwaitingPlatformAction: (value: boolean) => void;

  // Utilities
  getCurrentStepConfig: () => WizardStep | null;
  getStepStatus: (stepId: string) => "pending" | "in-progress" | "completed";
}

const WizardContext = createContext<WizardContextValue | null>(null);

interface WizardProviderProps {
  children: ReactNode;
}

export function WizardProvider({ children }: WizardProviderProps) {
  const location = useLocation();
  const navigate = useNavigate();
  const previousLocation = useRef(location);

  const [activeWizard, setActiveWizard] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState(0);
  const [completedSteps, setCompletedSteps] = useState<string[]>([]);
  const [wizardConfig, setWizardConfig] = useState<WizardConfig | null>(null);
  const [isCompleted, setIsCompleted] = useState(false);
  const [wizardCompletionData, setWizardCompletionData] = useState<Record<
    string,
    any
  > | null>(null);
  const [isAwaitingPlatformAction, setIsAwaitingPlatformAction] = useState(false);

  // Initialize from localStorage on mount
  useEffect(() => {
    const stored = WizardStorage.getCurrentWizard();
    if (!stored) return;

    const restoredConfig = getWizardConfig(stored.wizardId);
    if (!restoredConfig) {
      // Stale storage entry (unknown wizard id) should not lock UI in "active" state.
      WizardStorage.clearCurrent();
      return;
    }

    const maxStepIndex = Math.max(restoredConfig.steps.length - 1, 0);
    const safeStep = Math.min(Math.max(stored.currentStep, 0), maxStepIndex);

    setActiveWizard(stored.wizardId);
    setWizardConfig(restoredConfig);
    setCurrentStep(safeStep);
    setCompletedSteps(Array.isArray(stored.completedSteps) ? stored.completedSteps : []);
  }, []);

  // Track location changes for navigation-return completion triggers
  useEffect(() => {
    if (!activeWizard || !wizardConfig) {
      previousLocation.current = location;
      return;
    }

    const currentStepConfig = wizardConfig.steps[currentStep];
    if (!currentStepConfig) {
      previousLocation.current = location;
      return;
    }

    if (
      currentStepConfig.actionType === "navigate" &&
      currentStepConfig.completionTrigger === "navigation-return"
    ) {
      const targetRoute = currentStepConfig.actionPayload.route;
      const wasOnTargetRoute =
        previousLocation.current.pathname === targetRoute;
      const isBackFromTarget =
        location.pathname !== targetRoute && wasOnTargetRoute;

      if (isBackFromTarget && !completedSteps.includes(currentStepConfig.id)) {
        const locationState = location.state as any;
        if (locationState?.workloadCreated === true) {
          completeStep(currentStepConfig.id);
        }
      }
    }

    if (
      currentStepConfig.actionType === "custom" &&
      currentStepConfig.completionTrigger === "navigation-return"
    ) {
      const locationState = location.state as any;

      if (
        activeWizard === "user-auth-wizard" &&
        currentStepConfig.id === "configure-auth"
      ) {
        const wasOnAuthPage =
          previousLocation.current.pathname === "/authentication/create" ||
          previousLocation.current.pathname === "/authentication/saml/create";
        const isBackOnDashboard = location.pathname === "/" || location.pathname === "/dashboard";

        if (
          wasOnAuthPage &&
          isBackOnDashboard &&
          locationState?.authProviderCreated === true &&
          !completedSteps.includes(currentStepConfig.id)
        ) {

          if (locationState?.clientId) {
            setWizardCompletionData((prev) => ({
              ...prev,
              clientId: locationState.clientId,
            }));
          }

          completeStep(currentStepConfig.id);
        }
      }
    }

    // Handle navigation-return for integrate-sdk step (Step 4)
    if (
      currentStepConfig.actionType === "navigate" &&
      currentStepConfig.completionTrigger === "navigation-return"
    ) {
      const locationState = location.state as any;

      if (
        activeWizard === "user-auth-wizard" &&
        currentStepConfig.id === "integrate-sdk"
      ) {
        const wasOnOnboardPage =
          previousLocation.current.pathname.startsWith("/clients/onboard/") ||
          previousLocation.current.pathname.startsWith("/sdk/clients/");
        const isBackOnDashboard =
          location.pathname === "/" || location.pathname === "/dashboard";

        if (
          wasOnOnboardPage &&
          isBackOnDashboard &&
          locationState?.sdkIntegrationComplete === true &&
          !completedSteps.includes(currentStepConfig.id)
        ) {
          completeStep(currentStepConfig.id);
        }
      }
    }

    if (
      activeWizard === "rbac-wizard" &&
      currentStepConfig.actionType === "navigate" &&
      currentStepConfig.completionTrigger === "navigation-return"
    ) {
      const locationState = location.state as any;
      const isBackOnRoot = location.pathname === "/" || location.pathname === "/dashboard";

      // Check for RBAC creation flags
      if (
        currentStepConfig.id === "create-permissions" &&
        locationState?.permissionCreated === true &&
        isBackOnRoot &&
        !completedSteps.includes(currentStepConfig.id)
      ) {
        completeStep(currentStepConfig.id);
      } else if (
        currentStepConfig.id === "create-roles" &&
        locationState?.roleCreated === true &&
        isBackOnRoot &&
        !completedSteps.includes(currentStepConfig.id)
      ) {
        completeStep(currentStepConfig.id);
      } else if (
        currentStepConfig.id === "create-bindings" &&
        locationState?.bindingCreated === true &&
        isBackOnRoot &&
        !completedSteps.includes(currentStepConfig.id)
      ) {
        completeStep(currentStepConfig.id);
      }
    }

    // Handle Scopes wizard navigation-return triggers
    if (
      activeWizard === "scopes-wizard" &&
      currentStepConfig.actionType === "navigate" &&
      currentStepConfig.completionTrigger === "navigation-return"
    ) {
      const locationState = location.state as any;
      const isBackOnRoot = location.pathname === "/" || location.pathname === "/dashboard";

      // Check for scope creation flag
      if (
        currentStepConfig.id === "create-scope" &&
        locationState?.scopeCreated === true &&
        isBackOnRoot &&
        !completedSteps.includes(currentStepConfig.id)
      ) {
        console.log(
          `[Wizard] Auto-completing step "${currentStepConfig.id}" - scope created`
        );
        completeStep(currentStepConfig.id);
      }
    }

    previousLocation.current = location;
  }, [location, activeWizard, currentStep, wizardConfig, completedSteps]);

  useEffect(() => {
    if (!activeWizard || !wizardConfig) return;

    const currentStepConfig = wizardConfig.steps[currentStep];
    if (!currentStepConfig) return;

    if (
      activeWizard === "user-auth-wizard" &&
      currentStepConfig.id === "client-selection" &&
      wizardCompletionData?.clientId &&
      !completedSteps.includes(currentStepConfig.id)
    ) {
      setTimeout(() => {
        completeStep(currentStepConfig.id);
      }, 100);
    }
  }, [
    activeWizard,
    currentStep,
    wizardConfig,
    wizardCompletionData,
    completedSteps,
  ]);

  // Start a new wizard
  const startWizard = (wizardId: string, config: WizardConfig) => {
    if (
      WizardStorage.isCompleted(wizardId) ||
      WizardStorage.isDismissed(wizardId)
    ) {
      console.log(
        `[Wizard] Wizard "${wizardId}" already completed or dismissed`
      );
      return;
    }

    setActiveWizard(wizardId);
    setWizardConfig(config);
    setCurrentStep(0);
    setCompletedSteps([]);
    setIsCompleted(false);
    setIsAwaitingPlatformAction(false);

    WizardStorage.startWizard(wizardId);
    console.log(`[Wizard] Started wizard: ${wizardId}`);
  };

  const completeStep = (stepId: string) => {
    if (completedSteps.includes(stepId)) return;
    setIsAwaitingPlatformAction(false);

    const newCompletedSteps = [...completedSteps, stepId];
    setCompletedSteps(newCompletedSteps);

    if (activeWizard) {
      WizardStorage.updateWizard(activeWizard, {
        completedSteps: newCompletedSteps,
      });
    }

    console.log(`[Wizard] Completed step: ${stepId}`);
    if (wizardConfig && wizardConfig.steps[currentStep]?.id === stepId) {
      const nextStepIndex = currentStep + 1;
      if (nextStepIndex < wizardConfig.steps.length) {
        setTimeout(() => nextStep(), 300);
      } else {
        setTimeout(() => completeWizard(), 500);
      }
    }
  };

  const nextStep = () => {
    if (!wizardConfig) return;

    const nextStepIndex = currentStep + 1;
    if (nextStepIndex < wizardConfig.steps.length) {
      setCurrentStep(nextStepIndex);

      if (activeWizard) {
        WizardStorage.updateWizard(activeWizard, {
          currentStep: nextStepIndex,
        });
      }

      console.log(`[Wizard] Advanced to step ${nextStepIndex + 1}`);
    }
  };

  const previousStep = () => {
    if (currentStep > 0) {
      const prevStepIndex = currentStep - 1;
      setCurrentStep(prevStepIndex);

      if (activeWizard) {
        WizardStorage.updateWizard(activeWizard, {
          currentStep: prevStepIndex,
        });
      }

      console.log(`[Wizard] Moved back to step ${prevStepIndex + 1}`);
    }
  };

  const skipWizard = () => {
    if (!activeWizard) return;

    WizardStorage.dismissWizard(activeWizard);

    if (wizardConfig?.onSkip) {
      wizardConfig.onSkip();
    }

    setActiveWizard(null);
    setWizardConfig(null);
    setCurrentStep(0);
    setCompletedSteps([]);

    console.log("[Wizard] Wizard skipped");
  };

  const dismissWizard = () => {
    if (!activeWizard) return;

    WizardStorage.updateWizard(activeWizard, {
      currentStep,
      completedSteps,
    });

    if (wizardConfig?.onDismiss) {
      wizardConfig.onDismiss();
    }

    setActiveWizard(null);
    setWizardConfig(null);
    setIsCompleted(false);
    setIsAwaitingPlatformAction(false);

    console.log("[Wizard] Wizard dismissed (progress saved)");
  };

  const completeWizard = (data?: Record<string, any>) => {
    if (!activeWizard) return;
    const completionDataWithWizardId = {
      ...wizardCompletionData,
      ...data,
      wizardId: activeWizard,
    };

    setWizardCompletionData(completionDataWithWizardId);
    console.log("[Wizard] Stored completion data:", completionDataWithWizardId);

    WizardStorage.completeWizard(activeWizard);

    if (wizardConfig?.onComplete) {
      wizardConfig.onComplete();
    }

    setIsCompleted(true);

    setActiveWizard(null);
    setWizardConfig(null);
    setCurrentStep(0);
    setCompletedSteps([]);

    console.log("[Wizard] Wizard completed!");
  };

  const resetCompletion = () => {
    setIsCompleted(false);
    setWizardCompletionData(null);
    console.log("[Wizard] Completion state reset");
  };

  const getCurrentStepConfig = (): WizardStep | null => {
    if (!wizardConfig) return null;
    return wizardConfig.steps[currentStep] || null;
  };

  const getStepStatus = (
    stepId: string
  ): "pending" | "in-progress" | "completed" => {
    if (completedSteps.includes(stepId)) return "completed";

    const currentStepConfig = getCurrentStepConfig();
    if (currentStepConfig?.id === stepId) return "in-progress";

    return "pending";
  };

  const value: WizardContextValue = {
    // State
    activeWizard,
    currentStep,
    isActive: activeWizard !== null,
    isCompleted,
    completedSteps,
    wizardConfig,
    wizardCompletionData,

    // Actions
    startWizard,
    completeStep,
    nextStep,
    previousStep,
    skipWizard,
    dismissWizard,
    completeWizard,
    resetCompletion,
    setWizardCompletionData,

    // Platform action state
    isAwaitingPlatformAction,
    setIsAwaitingPlatformAction,

    // Utilities
    getCurrentStepConfig,
    getStepStatus,
  };

  return (
    <WizardContext.Provider value={value}>{children}</WizardContext.Provider>
  );
}

/**
 * Hook to use wizard context
 */
export function useWizard() {
  const context = useContext(WizardContext);
  if (!context) {
    throw new Error("useWizard must be used within WizardProvider");
  }
  return context;
}
