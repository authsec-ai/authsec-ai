import { RefObject } from 'react';

/**
 * Position where the content card should appear relative to the target element
 */
export type TourContentPosition = 'top' | 'bottom' | 'left' | 'right' | 'center';

/**
 * Individual step in a guided tour
 */
export interface TourStep {
  /** Unique identifier for this step */
  id: string;

  /** Target element to highlight - either a React ref or a CSS selector (data-tour-id) */
  target: RefObject<HTMLElement> | string;

  /** Heading text shown in the content card */
  heading: string;

  /** Description text explaining what the user should do */
  description: string;

  /** Where to position the content card relative to the target */
  position: TourContentPosition;

  /** Additional padding around the spotlight (default: 8px) */
  spotlightPadding?: number;

  /** Custom offset for content positioning */
  offset?: { x: number; y: number };

  /** Callback executed before showing this step */
  onBeforeShow?: () => void | Promise<void>;

  /** Callback executed after dismissing this step */
  onAfterDismiss?: () => void;
}

/**
 * Configuration for a complete tour (one or more steps)
 */
export interface TourConfig {
  /** Unique identifier for this tour */
  tourId: string;

  /** Page/feature identifier where this tour appears */
  pageId: string;

  /** Array of steps in this tour */
  steps: TourStep[];

  /** Whether to auto-start this tour on first visit (default: true) */
  autoStart?: boolean;

  /** Priority when multiple tours exist on same page (higher = shown first) */
  priority?: number;
}

/**
 * Information about a completed tour stored in localStorage
 */
export interface CompletedTourInfo {
  /** Tour identifier */
  tourId: string;

  /** ISO timestamp when completed */
  completedAt: string;

  /** Array of step IDs that were completed */
  completedSteps: string[];

  /** Tour configuration version (for versioning) */
  version: string;
}

/**
 * Structure of data stored in localStorage
 */
export interface TourStorageData {
  /** Map of completed tours by tourId */
  completedTours: Record<string, CompletedTourInfo>;

  /** List of permanently dismissed tour IDs */
  dismissedTours: string[];

  /** User preferences for tour system */
  userPreferences: {
    /** Disable all tours globally */
    disableAllTours?: boolean;

    /** Last time preferences were updated */
    lastUpdated: string;
  };
}

/**
 * Calculated position and dimensions for spotlight effect
 */
export interface SpotlightPosition {
  /** X coordinate (left edge) */
  x: number;

  /** Y coordinate (top edge) */
  y: number;

  /** Width of spotlight */
  width: number;

  /** Height of spotlight */
  height: number;

  /** Border radius for spotlight */
  borderRadius: number;
}

/**
 * Calculated position for content card
 */
export interface ContentPosition {
  /** X coordinate (left edge) or 'center' */
  x: number | 'center';

  /** Y coordinate (top edge) or 'center' */
  y: number | 'center';

  /** Maximum width for content */
  maxWidth: number;
}

/**
 * Context value provided by GuidedTourProvider
 */
export interface GuidedTourContextValue {
  /** Currently active tour ID (null if none active) */
  activeTour: string | null;

  /** Current step index (0-based) */
  currentStep: number;

  /** Whether a tour is currently active */
  isActive: boolean;

  /** Start a tour by ID */
  startTour: (tourId: string) => void;

  /** Advance to next step */
  nextStep: () => void;

  /** Go back to previous step */
  previousStep: () => void;

  /** Mark current tour as completed and dismiss */
  completeTour: () => void;

  /** Dismiss tour (permanent if parameter is true) */
  dismissTour: (permanent?: boolean) => void;

  /** Skip entire tour without marking individual steps */
  skipTour: () => void;

  /** Check if a specific tour has been completed */
  isTourCompleted: (tourId: string) => boolean;

  /** Check if a tour should be shown (not completed, not dismissed) */
  shouldShowTour: (tourId: string) => boolean;

  /** Reset a specific tour (for testing) */
  resetTour: (tourId: string) => void;

  /** Reset all tours (for testing) */
  resetAllTours: () => void;

  /** Get current tour configuration */
  getCurrentTourConfig: () => TourConfig | null;

  /** Get current step configuration */
  getCurrentStepConfig: () => TourStep | null;
}

