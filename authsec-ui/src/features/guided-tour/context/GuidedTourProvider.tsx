import React, { createContext, useState, useCallback, ReactNode } from 'react';
import type { GuidedTourContextValue, TourConfig, TourStep } from '../types';
import { TourStorage } from '../utils/tourStorage';
import { TOUR_REGISTRY } from '../utils/tourConfig';

/**
 * Context for guided tour system
 */
export const GuidedTourContext = createContext<GuidedTourContextValue | null>(null);

interface GuidedTourProviderProps {
  children: ReactNode;
}

/**
 * Provider component for guided tour system
 * Manages tour state, lifecycle, and persistence
 */
export function GuidedTourProvider({ children }: GuidedTourProviderProps) {
  const [activeTour, setActiveTour] = useState<string | null>(null);
  const [currentStep, setCurrentStep] = useState<number>(0);

  /**
   * Get the current tour configuration
   */
  const getCurrentTourConfig = useCallback((): TourConfig | null => {
    if (!activeTour) return null;
    return TOUR_REGISTRY[activeTour] || null;
  }, [activeTour]);

  /**
   * Get the current step configuration
   */
  const getCurrentStepConfig = useCallback((): TourStep | null => {
    const tourConfig = getCurrentTourConfig();
    if (!tourConfig) return null;
    return tourConfig.steps[currentStep] || null;
  }, [getCurrentTourConfig, currentStep]);

  /**
   * Start a tour by ID
   */
  const startTour = useCallback((tourId: string) => {
    // Check if tour exists in registry
    const tourConfig = TOUR_REGISTRY[tourId];
    if (!tourConfig) {
      console.warn(`[GuidedTour] Tour "${tourId}" not found in registry`);
      return;
    }

    // Check if tour should be shown
    if (!TourStorage.shouldShowTour(tourId)) {
      console.log(`[GuidedTour] Tour "${tourId}" already completed or dismissed`);
      return;
    }

    // Check if steps exist
    if (!tourConfig.steps || tourConfig.steps.length === 0) {
      console.warn(`[GuidedTour] Tour "${tourId}" has no steps`);
      return;
    }

    // Execute onBeforeShow callback for first step
    const firstStep = tourConfig.steps[0];
    if (firstStep.onBeforeShow) {
      Promise.resolve(firstStep.onBeforeShow()).then(() => {
        setActiveTour(tourId);
        setCurrentStep(0);
      }).catch((error) => {
        console.error('[GuidedTour] Error in onBeforeShow callback:', error);
      });
    } else {
      setActiveTour(tourId);
      setCurrentStep(0);
    }
  }, []);

  /**
   * Advance to next step
   */
  const nextStep = useCallback(() => {
    const tourConfig = getCurrentTourConfig();
    if (!tourConfig) return;

    const currentStepConfig = getCurrentStepConfig();

    // Execute onAfterDismiss callback for current step
    if (currentStepConfig?.onAfterDismiss) {
      currentStepConfig.onAfterDismiss();
    }

    const nextStepIndex = currentStep + 1;

    // Check if there are more steps
    if (nextStepIndex < tourConfig.steps.length) {
      const nextStepConfig = tourConfig.steps[nextStepIndex];

      // Execute onBeforeShow callback for next step
      if (nextStepConfig.onBeforeShow) {
        Promise.resolve(nextStepConfig.onBeforeShow()).then(() => {
          setCurrentStep(nextStepIndex);
        }).catch((error) => {
          console.error('[GuidedTour] Error in onBeforeShow callback:', error);
          setCurrentStep(nextStepIndex);
        });
      } else {
        setCurrentStep(nextStepIndex);
      }
    } else {
      // No more steps, complete the tour
      completeTour();
    }
  }, [currentStep, getCurrentTourConfig, getCurrentStepConfig]);

  /**
   * Go back to previous step
   */
  const previousStep = useCallback(() => {
    if (currentStep > 0) {
      const tourConfig = getCurrentTourConfig();
      if (!tourConfig) return;

      const currentStepConfig = getCurrentStepConfig();

      // Execute onAfterDismiss callback for current step
      if (currentStepConfig?.onAfterDismiss) {
        currentStepConfig.onAfterDismiss();
      }

      const prevStepIndex = currentStep - 1;
      const prevStepConfig = tourConfig.steps[prevStepIndex];

      // Execute onBeforeShow callback for previous step
      if (prevStepConfig.onBeforeShow) {
        Promise.resolve(prevStepConfig.onBeforeShow()).then(() => {
          setCurrentStep(prevStepIndex);
        }).catch((error) => {
          console.error('[GuidedTour] Error in onBeforeShow callback:', error);
          setCurrentStep(prevStepIndex);
        });
      } else {
        setCurrentStep(prevStepIndex);
      }
    }
  }, [currentStep, getCurrentTourConfig, getCurrentStepConfig]);

  /**
   * Mark tour as completed and dismiss
   */
  const completeTour = useCallback(() => {
    if (!activeTour) return;

    const tourConfig = getCurrentTourConfig();
    if (!tourConfig) return;

    // Get all step IDs
    const stepIds = tourConfig.steps.map(step => step.id);

    // Execute onAfterDismiss callback for current step
    const currentStepConfig = getCurrentStepConfig();
    if (currentStepConfig?.onAfterDismiss) {
      currentStepConfig.onAfterDismiss();
    }

    // Mark as completed in storage
    TourStorage.markTourCompleted(activeTour, stepIds);

    // Reset state
    setActiveTour(null);
    setCurrentStep(0);
  }, [activeTour, getCurrentTourConfig, getCurrentStepConfig]);

  /**
   * Dismiss tour (permanently if parameter is true)
   */
  const dismissTour = useCallback((permanent: boolean = false) => {
    if (!activeTour) return;

    // Execute onAfterDismiss callback for current step
    const currentStepConfig = getCurrentStepConfig();
    if (currentStepConfig?.onAfterDismiss) {
      currentStepConfig.onAfterDismiss();
    }

    // If permanent, mark as dismissed in storage
    if (permanent) {
      TourStorage.dismissTour(activeTour);
    }

    // Reset state
    setActiveTour(null);
    setCurrentStep(0);
  }, [activeTour, getCurrentStepConfig]);

  /**
   * Skip entire tour (mark as dismissed, don't complete)
   */
  const skipTour = useCallback(() => {
    dismissTour(true);
  }, [dismissTour]);

  /**
   * Check if a specific tour has been completed
   */
  const isTourCompleted = useCallback((tourId: string): boolean => {
    return TourStorage.isTourCompleted(tourId);
  }, []);

  /**
   * Check if a tour should be shown
   */
  const shouldShowTour = useCallback((tourId: string): boolean => {
    return TourStorage.shouldShowTour(tourId);
  }, []);

  /**
   * Reset a specific tour (for testing)
   */
  const resetTour = useCallback((tourId: string) => {
    TourStorage.resetTour(tourId);

    // If this is the active tour, dismiss it
    if (activeTour === tourId) {
      setActiveTour(null);
      setCurrentStep(0);
    }
  }, [activeTour]);

  /**
   * Reset all tours (for testing)
   */
  const resetAllTours = useCallback(() => {
    TourStorage.resetAllTours();

    // Dismiss any active tour
    setActiveTour(null);
    setCurrentStep(0);
  }, []);

  const value: GuidedTourContextValue = {
    activeTour,
    currentStep,
    isActive: activeTour !== null,
    startTour,
    nextStep,
    previousStep,
    completeTour,
    dismissTour,
    skipTour,
    isTourCompleted,
    shouldShowTour,
    resetTour,
    resetAllTours,
    getCurrentTourConfig,
    getCurrentStepConfig,
  };

  return (
    <GuidedTourContext.Provider value={value}>
      {children}
    </GuidedTourContext.Provider>
  );
}
