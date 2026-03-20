import { useContext } from 'react';
import { GuidedTourContext } from '../context/GuidedTourProvider';
import type { GuidedTourContextValue } from '../types';

/**
 * Hook to access the guided tour context
 *
 * @throws Error if used outside of GuidedTourProvider
 * @returns The guided tour context value
 *
 * @example
 * ```tsx
 * function MyComponent() {
 *   const { startTour, isActive, nextStep } = useGuidedTour();
 *
 *   return (
 *     <button onClick={() => startTour('my-tour-id')}>
 *       Start Tour
 *     </button>
 *   );
 * }
 * ```
 */
export function useGuidedTour(): GuidedTourContextValue {
  const context = useContext(GuidedTourContext);

  if (!context) {
    throw new Error(
      'useGuidedTour must be used within a GuidedTourProvider. ' +
      'Make sure your component is wrapped with <GuidedTourProvider>.'
    );
  }

  return context;
}
