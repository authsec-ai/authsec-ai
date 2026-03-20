import type { WizardStorageData } from "../types";

const STORAGE_KEY = "authsec_wizards";

/**
 * Default wizard storage state
 */
const defaultStorage: WizardStorageData = {
  completedWizards: [],
  currentWizard: null,
  dismissedWizards: [],
};

/**
 * Wizard Storage Utility
 * Manages wizard state persistence in localStorage
 */
export const WizardStorage = {
  /**
   * Get all wizard data from localStorage
   */
  get(): WizardStorageData {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) return defaultStorage;

      const parsed = JSON.parse(stored);
      return { ...defaultStorage, ...parsed };
    } catch (error) {
      console.error("[WizardStorage] Error reading storage:", error);
      return defaultStorage;
    }
  },

  /**
   * Save wizard data to localStorage
   */
  set(data: Partial<WizardStorageData>): void {
    try {
      const current = this.get();
      const updated = { ...current, ...data };
      localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
    } catch (error) {
      console.error("[WizardStorage] Error saving storage:", error);
    }
  },

  /**
   * Start a wizard - set as current wizard
   */
  startWizard(wizardId: string): void {
    this.set({
      currentWizard: {
        wizardId,
        currentStep: 0,
        completedSteps: [],
        startedAt: new Date().toISOString(),
      },
    });
  },

  /**
   * Update current wizard state
   */
  updateWizard(
    wizardId: string,
    updates: {
      currentStep?: number;
      completedSteps?: string[];
    }
  ): void {
    const data = this.get();
    if (data.currentWizard?.wizardId === wizardId) {
      this.set({
        currentWizard: {
          ...data.currentWizard,
          ...updates,
        },
      });
    }
  },

  /**
   * Complete a wizard - add to completed list and clear current
   */
  completeWizard(wizardId: string): void {
    const data = this.get();
    const completedWizards = Array.from(
      new Set([...data.completedWizards, wizardId])
    );

    this.set({
      completedWizards,
      currentWizard: null,
    });
  },

  /**
   * Dismiss a wizard - add to dismissed list and clear current
   */
  dismissWizard(wizardId: string): void {
    const data = this.get();
    const dismissedWizards = Array.from(
      new Set([...data.dismissedWizards, wizardId])
    );

    this.set({
      dismissedWizards,
      currentWizard: null,
    });
  },

  /**
   * Check if wizard is completed
   */
  isCompleted(wizardId: string): boolean {
    const data = this.get();
    return data.completedWizards.includes(wizardId);
  },

  /**
   * Check if wizard is dismissed
   */
  isDismissed(wizardId: string): boolean {
    const data = this.get();
    return data.dismissedWizards.includes(wizardId);
  },

  /**
   * Get current wizard state
   */
  getCurrentWizard(): WizardStorageData["currentWizard"] {
    return this.get().currentWizard;
  },

  /**
   * Clear current wizard (without marking completed/dismissed)
   */
  clearCurrent(): void {
    this.set({ currentWizard: null });
  },

  /**
   * Reset a specific wizard - remove from completed and dismissed
   */
  resetWizard(wizardId: string): void {
    const data = this.get();
    this.set({
      completedWizards: data.completedWizards.filter((id) => id !== wizardId),
      dismissedWizards: data.dismissedWizards.filter((id) => id !== wizardId),
      currentWizard:
        data.currentWizard?.wizardId === wizardId ? null : data.currentWizard,
    });
  },

  /**
   * Clear all wizard data
   */
  clear(): void {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.error("[WizardStorage] Error clearing storage:", error);
    }
  },
};
