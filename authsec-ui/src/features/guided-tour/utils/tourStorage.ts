import type { TourStorageData, CompletedTourInfo } from '../types';

/**
 * LocalStorage key for storing guided tour data
 */
const STORAGE_KEY = 'authsec_guided_tours';

/**
 * Current version of tour data structure (for migrations)
 */
const STORAGE_VERSION = '1.0.0';

/**
 * Get default/empty tour storage data
 */
function getDefaultStorageData(): TourStorageData {
  return {
    completedTours: {},
    dismissedTours: [],
    userPreferences: {
      disableAllTours: false,
      lastUpdated: new Date().toISOString(),
    },
  };
}

/**
 * Utility class for managing guided tour persistence in localStorage
 */
export class TourStorage {
  /**
   * Load tour data from localStorage
   */
  static load(): TourStorageData {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      if (!stored) {
        return getDefaultStorageData();
      }

      const parsed = JSON.parse(stored) as TourStorageData;

      // Validate structure and provide defaults for missing fields
      return {
        completedTours: parsed.completedTours || {},
        dismissedTours: parsed.dismissedTours || [],
        userPreferences: {
          disableAllTours: parsed.userPreferences?.disableAllTours || false,
          lastUpdated: parsed.userPreferences?.lastUpdated || new Date().toISOString(),
        },
      };
    } catch (error) {
      console.warn('[GuidedTour] Failed to load tour data from localStorage:', error);
      return getDefaultStorageData();
    }
  }

  /**
   * Save tour data to localStorage
   */
  static save(data: TourStorageData): void {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(data));
    } catch (error) {
      console.error('[GuidedTour] Failed to save tour data to localStorage:', error);
    }
  }

  /**
   * Mark a tour as completed
   */
  static markTourCompleted(tourId: string, stepIds: string[]): void {
    const data = this.load();

    data.completedTours[tourId] = {
      tourId,
      completedAt: new Date().toISOString(),
      completedSteps: stepIds,
      version: STORAGE_VERSION,
    };

    // Remove from dismissed list if present
    data.dismissedTours = data.dismissedTours.filter(id => id !== tourId);

    this.save(data);
  }

  /**
   * Check if a tour has been completed
   */
  static isTourCompleted(tourId: string): boolean {
    const data = this.load();
    return tourId in data.completedTours;
  }

  /**
   * Get completion info for a specific tour
   */
  static getTourCompletionInfo(tourId: string): CompletedTourInfo | null {
    const data = this.load();
    return data.completedTours[tourId] || null;
  }

  /**
   * Dismiss a tour permanently (without marking as completed)
   */
  static dismissTour(tourId: string): void {
    const data = this.load();

    if (!data.dismissedTours.includes(tourId)) {
      data.dismissedTours.push(tourId);
      this.save(data);
    }
  }

  /**
   * Check if a tour has been dismissed
   */
  static isDismissed(tourId: string): boolean {
    const data = this.load();
    return data.dismissedTours.includes(tourId);
  }

  /**
   * Reset a specific tour (remove from completed and dismissed)
   */
  static resetTour(tourId: string): void {
    const data = this.load();

    delete data.completedTours[tourId];
    data.dismissedTours = data.dismissedTours.filter(id => id !== tourId);

    this.save(data);
  }

  /**
   * Reset all tours (clear all completion and dismissal data)
   */
  static resetAllTours(): void {
    const data = getDefaultStorageData();
    this.save(data);
  }

  /**
   * Check if a tour should be shown (not completed, not dismissed, tours not globally disabled)
   */
  static shouldShowTour(tourId: string): boolean {
    const data = this.load();

    // Check global disable flag
    if (data.userPreferences.disableAllTours) {
      return false;
    }

    // Check if completed
    if (tourId in data.completedTours) {
      return false;
    }

    // Check if dismissed
    if (data.dismissedTours.includes(tourId)) {
      return false;
    }

    return true;
  }

  /**
   * Set global tour disable preference
   */
  static setToursDisabled(disabled: boolean): void {
    const data = this.load();
    data.userPreferences.disableAllTours = disabled;
    data.userPreferences.lastUpdated = new Date().toISOString();
    this.save(data);
  }

  /**
   * Check if tours are globally disabled
   */
  static areToursDisabled(): boolean {
    const data = this.load();
    return data.userPreferences.disableAllTours || false;
  }

  /**
   * Get all completed tour IDs
   */
  static getCompletedTourIds(): string[] {
    const data = this.load();
    return Object.keys(data.completedTours);
  }

  /**
   * Get all dismissed tour IDs
   */
  static getDismissedTourIds(): string[] {
    const data = this.load();
    return [...data.dismissedTours];
  }

  /**
   * Clear all tour data (useful for logout or testing)
   */
  static clear(): void {
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch (error) {
      console.error('[GuidedTour] Failed to clear tour data:', error);
    }
  }

  /**
   * Export tour data (for debugging or analytics)
   */
  static export(): TourStorageData {
    return this.load();
  }

  /**
   * Import tour data (for restoring backup or testing)
   */
  static import(data: TourStorageData): void {
    this.save(data);
  }
}
