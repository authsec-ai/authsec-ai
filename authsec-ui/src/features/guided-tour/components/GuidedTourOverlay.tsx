import { useEffect, useState, useCallback } from 'react';
import { createPortal } from 'react-dom';
import { AnimatePresence, motion } from 'framer-motion';
import { useGuidedTour } from '../hooks/useGuidedTour';
import { GuidedTourSpotlight } from './GuidedTourSpotlight';
import { GuidedTourContent } from './GuidedTourContent';
import type { SpotlightPosition, ContentPosition } from '../types';
import {
  calculateSpotlightPosition,
  calculateContentPosition,
  scrollIntoViewIfNeeded,
  isElementVisible,
} from '../utils/positioning';

/**
 * Main overlay component that renders the guided tour
 * Portal-based, full-screen overlay with backdrop, spotlight, and content
 */
export function GuidedTourOverlay() {
  const {
    isActive,
    currentStep,
    nextStep,
    skipTour,
    dismissTour,
    previousStep,
    getCurrentTourConfig,
    getCurrentStepConfig,
  } = useGuidedTour();

  const [spotlightPos, setSpotlightPos] = useState<SpotlightPosition | null>(null);
  const [contentPos, setContentPos] = useState<ContentPosition | null>(null);
  const [mounted, setMounted] = useState(false);

  const tourConfig = getCurrentTourConfig();
  const stepConfig = getCurrentStepConfig();

  /**
   * Calculate and update positions
   */
  const updatePositions = useCallback(() => {
    if (!stepConfig) return;

    // Check if target element is visible
    if (!isElementVisible(stepConfig.target)) {
      console.warn('[GuidedTour] Target element is not visible');
      // Skip to next step or complete tour if element not found
      nextStep();
      return;
    }

    // Scroll target into view if needed
    scrollIntoViewIfNeeded(stepConfig.target);

    // Calculate spotlight position
    const spotlight = calculateSpotlightPosition(
      stepConfig.target,
      stepConfig.spotlightPadding || 8
    );

    if (!spotlight) {
      console.warn('[GuidedTour] Could not calculate spotlight position');
      // Skip to next step if target not found
      nextStep();
      return;
    }

    // Calculate content position
    const content = calculateContentPosition(
      stepConfig,
      spotlight,
      window.innerWidth,
      window.innerHeight
    );

    setSpotlightPos(spotlight);
    setContentPos(content);
  }, [stepConfig, nextStep]);

  /**
   * Update positions when step changes or window resizes
   */
  useEffect(() => {
    if (!isActive || !stepConfig) {
      return;
    }

    // Initial position calculation with small delay to ensure DOM is ready
    const timeoutId = setTimeout(() => {
      updatePositions();
    }, 50);

    // Recalculate on window resize
    const handleResize = () => {
      updatePositions();
    };

    window.addEventListener('resize', handleResize);

    return () => {
      clearTimeout(timeoutId);
      window.removeEventListener('resize', handleResize);
    };
  }, [isActive, stepConfig, currentStep, updatePositions]);

  /**
   * Handle ESC key to dismiss tour
   */
  useEffect(() => {
    if (!isActive) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        skipTour();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
    };
  }, [isActive, skipTour]);

  /**
   * Prevent body scroll when tour is active
   */
  useEffect(() => {
    if (isActive) {
      document.body.style.overflow = 'hidden';
    } else {
      document.body.style.overflow = '';
    }

    return () => {
      document.body.style.overflow = '';
    };
  }, [isActive]);

  /**
   * Set mounted state for portal
   */
  useEffect(() => {
    setMounted(true);
  }, []);

  /**
   * Handle backdrop click (dismiss tour)
   * Only dismiss if clicking on the dark overlay, not the spotlight area
   */
  const handleBackdropClick = useCallback((e: React.MouseEvent) => {
    if (!spotlightPos) return;

    const clickX = e.clientX;
    const clickY = e.clientY;

    // Check if click is inside spotlight area
    const isInsideSpotlight =
      clickX >= spotlightPos.x &&
      clickX <= spotlightPos.x + spotlightPos.width &&
      clickY >= spotlightPos.y &&
      clickY <= spotlightPos.y + spotlightPos.height;

    // Only skip tour if clicking outside spotlight
    if (!isInsideSpotlight) {
      skipTour();
    }
  }, [skipTour, spotlightPos]);

  /**
   * Prevent clicks on content card from closing overlay
   */
  const handleContentClick = useCallback((e: React.MouseEvent) => {
    e.stopPropagation();
  }, []);

  if (!mounted) return null;

  return createPortal(
    <AnimatePresence>
      {isActive && tourConfig && stepConfig && spotlightPos && contentPos && (
        <motion.div
          className="fixed inset-0 z-[60] flex items-center justify-center pointer-events-none"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          transition={{ duration: 0.3, ease: 'easeOut' }}
          role="dialog"
          aria-modal="true"
          aria-labelledby="guided-tour-heading"
          aria-describedby="guided-tour-description"
        >
          {/* Clickable backdrop - dismiss tour when clicking dark areas */}
          <div
            className="absolute inset-0 pointer-events-auto"
            onClick={handleBackdropClick}
          >
            {/* SVG mask for spotlight cutout effect with backdrop filter */}
            <svg
              className="absolute inset-0 pointer-events-none"
              style={{ width: '100%', height: '100%' }}
            >
              <defs>
                <mask id="spotlight-mask">
                  {/* White background - will show the overlay */}
                  <rect width="100%" height="100%" fill="white" />
                  {/* Black cutout - will be transparent (shows page content) */}
                  <rect
                    x={spotlightPos.x}
                    y={spotlightPos.y}
                    width={spotlightPos.width}
                    height={spotlightPos.height}
                    rx={spotlightPos.borderRadius}
                    fill="black"
                  />
                </mask>
                {/* Blur filter for backdrop */}
                <filter id="backdrop-blur">
                  <feGaussianBlur in="SourceGraphic" stdDeviation="3" />
                </filter>
              </defs>

              {/* Blurred background layer */}
              <rect
                width="100%"
                height="100%"
                fill="rgba(23, 37, 84, 0.75)"
                filter="url(#backdrop-blur)"
                mask="url(#spotlight-mask)"
              />

              {/* Solid overlay on top for darker effect */}
              <rect
                width="100%"
                height="100%"
                fill="rgba(23, 37, 84, 0.3)"
                mask="url(#spotlight-mask)"
              />
            </svg>
          </div>

          {/* Transparent area over spotlight - allows clicks through to button */}
          <div
            className="absolute"
            style={{
              left: spotlightPos.x,
              top: spotlightPos.y,
              width: spotlightPos.width,
              height: spotlightPos.height,
              pointerEvents: 'none',
              zIndex: 1,
            }}
          />

          {/* Spotlight border highlight */}
          <div onClick={handleContentClick} style={{ pointerEvents: 'none' }}>
            <GuidedTourSpotlight position={spotlightPos} />
          </div>

          {/* Content Card */}
          <div onClick={handleContentClick} style={{ pointerEvents: 'auto' }}>
            <GuidedTourContent
              heading={stepConfig.heading}
              description={stepConfig.description}
              currentStep={currentStep}
              totalSteps={tourConfig.steps.length}
              position={contentPos}
              onNext={nextStep}
              onSkip={skipTour}
              onClose={() => dismissTour(false)}
              showPrevious={currentStep > 0}
              onPrevious={previousStep}
            />
          </div>
        </motion.div>
      )}
    </AnimatePresence>,
    document.body
  );
}
