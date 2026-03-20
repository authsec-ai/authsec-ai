import type { RefObject } from 'react';
import type { SpotlightPosition, ContentPosition, TourContentPosition, TourStep } from '../types';

/**
 * Get the target element from a ref or selector
 */
export function getTargetElement(target: RefObject<HTMLElement> | string): HTMLElement | null {
  if (typeof target === 'string') {
    // CSS selector (e.g., '[data-tour-id="onboard-button"]')
    return document.querySelector<HTMLElement>(target);
  } else {
    // React ref
    return target.current;
  }
}

/**
 * Calculate spotlight position and dimensions for a target element
 */
export function calculateSpotlightPosition(
  target: RefObject<HTMLElement> | string,
  padding: number = 8
): SpotlightPosition | null {
  const element = getTargetElement(target);

  if (!element) {
    console.warn('[GuidedTour] Target element not found for spotlight');
    return null;
  }

  const rect = element.getBoundingClientRect();

  // Calculate border radius based on element's computed style
  const computedStyle = window.getComputedStyle(element);
  const borderRadius = parseInt(computedStyle.borderRadius) || 8;

  return {
    x: rect.left - padding,
    y: rect.top - padding,
    width: rect.width + padding * 2,
    height: rect.height + padding * 2,
    borderRadius: borderRadius + padding,
  };
}

/**
 * Calculate content card position relative to target element and spotlight
 */
export function calculateContentPosition(
  step: TourStep,
  spotlightPos: SpotlightPosition | null,
  viewportWidth: number,
  viewportHeight: number
): ContentPosition {
  const contentMaxWidth = Math.min(400, viewportWidth - 32); // 16px padding on each side
  const contentGap = 16; // Gap between spotlight and content

  // Fallback to center if no spotlight position
  if (!spotlightPos) {
    return {
      x: 'center',
      y: 'center',
      maxWidth: contentMaxWidth,
    };
  }

  // Apply custom offset if provided
  const offsetX = step.offset?.x || 0;
  const offsetY = step.offset?.y || 0;

  let x: number | 'center' = 'center';
  let y: number | 'center' = 'center';

  switch (step.position) {
    case 'bottom':
      // Position below the spotlight
      x = spotlightPos.x + spotlightPos.width / 2 - contentMaxWidth / 2 + offsetX;
      y = spotlightPos.y + spotlightPos.height + contentGap + offsetY;

      // Ensure content stays within viewport horizontally
      if (x < 16) x = 16;
      if (x + contentMaxWidth > viewportWidth - 16) {
        x = viewportWidth - contentMaxWidth - 16;
      }

      // If content would overflow bottom, position above instead
      if (y + 200 > viewportHeight - 16) {
        y = spotlightPos.y - 200 - contentGap + offsetY;
      }
      break;

    case 'top':
      // Estimate content card height (more realistic than 200px)
      const estimatedContentHeight = 250;

      // Position above the spotlight
      x = spotlightPos.x + spotlightPos.width / 2 - contentMaxWidth / 2 + offsetX;
      y = spotlightPos.y - estimatedContentHeight - contentGap + offsetY;

      // Ensure content stays within viewport horizontally
      if (x < 16) x = 16;
      if (x + contentMaxWidth > viewportWidth - 16) {
        x = viewportWidth - contentMaxWidth - 16;
      }

      // If content would overflow top, find best alternative position
      if (y < 16) {
        // Check if there's space below
        const spaceBelow = viewportHeight - (spotlightPos.y + spotlightPos.height + contentGap);
        const spaceAbove = spotlightPos.y - contentGap;

        if (spaceBelow >= estimatedContentHeight + 16) {
          // Position below if there's enough space
          y = spotlightPos.y + spotlightPos.height + contentGap + offsetY;
        } else if (spaceAbove >= 100) {
          // Position at top of viewport if there's some space above
          y = 16;
        } else {
          // For large elements, position content card floating over the top portion
          // This ensures visibility when the table takes up most of the viewport
          y = Math.max(16, spotlightPos.y + 32);
        }
      }
      break;

    case 'left':
      // Position to the left of spotlight
      x = spotlightPos.x - contentMaxWidth - contentGap + offsetX;
      y = spotlightPos.y + spotlightPos.height / 2 - 100 + offsetY;

      // Ensure content stays within viewport vertically
      if (y < 16) y = 16;
      if (y + 200 > viewportHeight - 16) {
        y = viewportHeight - 200 - 16;
      }

      // If content would overflow left, position right instead
      if (x < 16) {
        x = spotlightPos.x + spotlightPos.width + contentGap + offsetX;
      }
      break;

    case 'right':
      // Position to the right of spotlight
      x = spotlightPos.x + spotlightPos.width + contentGap + offsetX;
      y = spotlightPos.y + spotlightPos.height / 2 - 100 + offsetY;

      // Ensure content stays within viewport vertically
      if (y < 16) y = 16;
      if (y + 200 > viewportHeight - 16) {
        y = viewportHeight - 200 - 16;
      }

      // If content would overflow right, position left instead
      if (x + contentMaxWidth > viewportWidth - 16) {
        x = spotlightPos.x - contentMaxWidth - contentGap + offsetX;
      }
      break;

    case 'center':
      // Center on screen
      x = 'center';
      y = 'center';
      break;
  }

  return {
    x,
    y,
    maxWidth: contentMaxWidth,
  };
}

/**
 * Scroll target element into view if it's outside viewport
 * For large elements like tables, only scroll if they're completely out of view
 */
export function scrollIntoViewIfNeeded(target: RefObject<HTMLElement> | string): void {
  const element = getTargetElement(target);

  if (!element) {
    return;
  }

  const rect = element.getBoundingClientRect();
  const viewportHeight = window.innerHeight;
  const viewportWidth = window.innerWidth;

  // Check if element is large (>50% of viewport height)
  const isLargeElement = rect.height > viewportHeight * 0.5;

  // For large elements, only scroll if they're completely out of view
  if (isLargeElement) {
    const isCompletelyAbove = rect.bottom < 0;
    const isCompletelyBelow = rect.top > viewportHeight;
    const isCompletelyLeft = rect.right < 0;
    const isCompletelyRight = rect.left > viewportWidth;

    if (isCompletelyAbove || isCompletelyBelow || isCompletelyLeft || isCompletelyRight) {
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'nearest', // Use 'nearest' instead of 'center' for large elements
        inline: 'nearest',
      });
    }
  } else {
    // For smaller elements, scroll if any part is out of view
    const isInViewport =
      rect.top >= 0 &&
      rect.left >= 0 &&
      rect.bottom <= viewportHeight &&
      rect.right <= viewportWidth;

    if (!isInViewport) {
      element.scrollIntoView({
        behavior: 'smooth',
        block: 'center',
        inline: 'center',
      });
    }
  }
}

/**
 * Check if an element is visible (not hidden by display:none or visibility:hidden)
 */
export function isElementVisible(target: RefObject<HTMLElement> | string): boolean {
  const element = getTargetElement(target);

  if (!element) {
    return false;
  }

  const style = window.getComputedStyle(element);
  return style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
}

/**
 * Wait for an element to be available in DOM
 */
export function waitForElement(
  selector: string,
  timeout: number = 5000
): Promise<HTMLElement | null> {
  return new Promise((resolve) => {
    const element = document.querySelector<HTMLElement>(selector);

    if (element) {
      resolve(element);
      return;
    }

    const observer = new MutationObserver(() => {
      const element = document.querySelector<HTMLElement>(selector);
      if (element) {
        observer.disconnect();
        resolve(element);
      }
    });

    observer.observe(document.body, {
      childList: true,
      subtree: true,
    });

    // Timeout
    setTimeout(() => {
      observer.disconnect();
      resolve(null);
    }, timeout);
  });
}
