/**
 * Guided Tour System
 *
 * A Google Cloud-style guided tour overlay system that helps onboard users
 * by highlighting key features and actions on feature pages.
 *
 * @example
 * ```tsx
 * // 1. Wrap your app with GuidedTourProvider (in App.tsx)
 * import { GuidedTourProvider, GuidedTourOverlay } from '@/features/guided-tour';
 *
 * function App() {
 *   return (
 *     <GuidedTourProvider>
 *       <YourApp />
 *       <GuidedTourOverlay />
 *     </GuidedTourProvider>
 *   );
 * }
 *
 * // 2. Use useTourStep in feature pages
 * import { useTourStep, TOUR_REGISTRY } from '@/features/guided-tour';
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

// Context & Provider
export { GuidedTourProvider } from './context/GuidedTourProvider';

// Components
export { GuidedTourOverlay } from './components/GuidedTourOverlay';

// Hooks
export { useGuidedTour } from './hooks/useGuidedTour';
export { useTourStep } from './hooks/useTourStep';

// Utils
export { TourStorage } from './utils/tourStorage';
export { TOUR_REGISTRY, getTourConfig, getAllTourIds, getToursForPage } from './utils/tourConfig';

// Types
export type {
  TourStep,
  TourConfig,
  TourContentPosition,
  CompletedTourInfo,
  TourStorageData,
  GuidedTourContextValue,
  SpotlightPosition,
  ContentPosition,
} from './types';
