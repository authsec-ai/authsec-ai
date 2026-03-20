import { ReactNode } from "react";

/**
 * Wizard Step Status
 */
export type WizardStepStatus = "pending" | "in-progress" | "completed";

/**
 * Wizard Step Action Types
 */
export type WizardStepActionType = "navigate" | "dialog" | "custom";

/**
 * Wizard Step Completion Triggers
 */
export type WizardCompletionTrigger =
  | "manual" // User must manually complete
  | "auto" // Auto-complete when action finishes
  | "navigation-return"; // Auto-complete on navigation return

/**
 * Wizard Step Action Payload
 */
export interface WizardStepActionPayload {
  route?: string; // For navigate type
  contentId?: string; // For dialog type
  handler?: string; // For custom type
  [key: string]: any; // Allow additional custom data
}

/**
 * Wizard Step Configuration
 */
export interface WizardStep {
  id: string;
  title: string;
  description: string;
  briefDescription?: string; // Short description for vertical stepper sidebar
  icon?: ReactNode;
  actionLabel: string;
  actionType: WizardStepActionType;
  actionPayload: WizardStepActionPayload;
  completionTrigger: WizardCompletionTrigger;
  status?: WizardStepStatus;
}

/**
 * Wizard Configuration
 */
export interface WizardConfig {
  wizardId: string;
  title: string;
  description?: string;
  steps: WizardStep[];
  onComplete?: () => void;
  onSkip?: () => void;
  onDismiss?: () => void;
}

/**
 * Wizard State (for storage/context)
 */
export interface WizardState {
  activeWizard: string | null;
  currentStep: number;
  completedSteps: string[];
  stepData: Record<string, any>;
}

/**
 * Wizard Storage Data (localStorage)
 */
export interface WizardStorageData {
  completedWizards: string[];
  currentWizard: {
    wizardId: string;
    currentStep: number;
    completedSteps: string[];
    startedAt: string;
  } | null;
  dismissedWizards: string[];
}

/**
 * Wizard Registry - Maps wizard IDs to configurations
 */
export type WizardRegistry = Record<string, WizardConfig>;
