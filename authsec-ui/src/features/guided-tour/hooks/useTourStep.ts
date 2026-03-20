import { useEffect } from 'react';
import { useGuidedTour } from './useGuidedTour';
import type { TourConfig } from '../types';

interface UseTourStepOptions {
  /**
   * Tour configuration (usually from TOUR_REGISTRY)
   */
  tourConfig: TourConfig;

  /**
   * Whether to auto-start the tour on mount (default: true)
   * Set to false if you want manual control
   */
  autoStart?: boolean;

  /**
   * Delay in milliseconds before auto-starting the tour (default: 500ms)
   * Useful to ensure all elements are rendered before starting
   */
  autoStartDelay?: number;
}

/**
 * Hook for pages to register and auto-trigger guided tours
 *
 * @example
 * ```tsx
 * import { useTourStep } from '@/features/guided-tour';
 * import { TOUR_REGISTRY } from '@/features/guided-tour/utils/tourConfig';
 *
 * function ClientsPage() {
 *   useTourStep({
 *     tourConfig: TOUR_REGISTRY['clients-onboarding'],
 *   });
 *
 *   return (
 *     <Button data-tour-id="onboard-button">
 *       Onboard Client
 *     </Button>
 *   );
 * }
 * ```
 */
export function useTourStep({
  tourConfig,
  autoStart = true,
  autoStartDelay = 500,
}: UseTourStepOptions) {
  const { startTour, shouldShowTour, isActive, activeTour } = useGuidedTour();

  useEffect(() => {
    // Don't auto-start if disabled
    if (!autoStart) {
      return;
    }

    // Don't auto-start if tour config doesn't specify autoStart
    if (tourConfig.autoStart === false) {
      return;
    }

    // Don't start if another tour is already active
    if (isActive) {
      return;
    }

    // Check if this tour should be shown (not completed, not dismissed)
    if (!shouldShowTour(tourConfig.tourId)) {
      return;
    }

    // Delay tour start to ensure DOM elements are ready
    const timeoutId = setTimeout(() => {
      startTour(tourConfig.tourId);
    }, autoStartDelay);

    return () => {
      clearTimeout(timeoutId);
    };
  }, [
    tourConfig.tourId,
    tourConfig.autoStart,
    autoStart,
    autoStartDelay,
    startTour,
    shouldShowTour,
    isActive,
  ]);

  return {
    /**
     * Manually trigger the tour (useful if autoStart is disabled)
     */
    triggerTour: () => startTour(tourConfig.tourId),

    /**
     * Whether this specific tour is currently active
     */
    isTourActive: isActive && activeTour === tourConfig.tourId,

    /**
     * Whether any tour is currently active
     */
    isAnyTourActive: isActive,
  };
}
