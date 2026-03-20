import { motion } from "framer-motion";
import { X } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { ContentPosition } from "../types";

interface GuidedTourContentProps {
  heading: string;
  description: string;
  currentStep: number;
  totalSteps: number;
  position: ContentPosition;
  onNext: () => void;
  onSkip: () => void;
  onClose: () => void;
  showPrevious?: boolean;
  onPrevious?: () => void;
}

/**
 * Content card component that displays tour information and controls
 */
export function GuidedTourContent({
  heading,
  description,
  currentStep,
  totalSteps,
  position,
  onNext,
  onSkip,
  onClose,
  showPrevious = false,
  onPrevious,
}: GuidedTourContentProps) {
  const isLastStep = currentStep === totalSteps - 1;

  // Calculate positioning styles
  const positionStyles: React.CSSProperties = {
    maxWidth: position.maxWidth,
  };

  if (position.x === "center" && position.y === "center") {
    positionStyles.left = "50%";
    positionStyles.top = "50%";
    positionStyles.transform = "translate(-50%, -50%)";
  } else {
    if (position.x === "center") {
      positionStyles.left = "50%";
      positionStyles.transform = "translateX(-50%)";
    } else {
      positionStyles.left = position.x;
    }

    if (position.y === "center") {
      positionStyles.top = "50%";
      positionStyles.transform = positionStyles.transform
        ? `${positionStyles.transform} translateY(-50%)`
        : "translateY(-50%)";
    } else {
      positionStyles.top = position.y;
    }
  }

  return (
    <motion.div
      className="absolute z-10"
      style={positionStyles}
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: 10 }}
      transition={{
        duration: 0.35,
        delay: 0.15,
        ease: "easeOut",
      }}
    >
      <div className="bg-card border border-border/20 rounded-xl shadow-2xl shadow-black/50 p-6 relative">
        {/* Close button */}
        <button
          type="button"
          onClick={onSkip}
          className="absolute top-4 right-4 text-muted-foreground hover:text-foreground transition-colors"
          aria-label="Close tour"
        >
          <X className="h-4 w-4" />
        </button>

        {/* Step indicator */}
        {totalSteps > 1 && (
          <div className="text-xs text-muted-foreground mb-3 font-medium">
            Step {currentStep + 1} of {totalSteps}
          </div>
        )}

        {/* Heading */}
        <h3 className="text-xl font-semibold mb-3 pr-6 text-foreground">
          {heading}
        </h3>

        {/* Description */}
        <p className="text-sm text-muted-foreground mb-6 leading-relaxed">
          {description}
        </p>

        {/* Action buttons */}
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-2">
            {showPrevious && currentStep > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onPrevious}
                className="text-xs"
              >
                Previous
              </Button>
            )}
            <Button
              variant="ghost"
              size="sm"
              onClick={onSkip}
              className="text-xs text-muted-foreground hover:text-foreground"
            >
              Skip Tour
            </Button>
          </div>

          <Button
            size="sm"
            onClick={onNext}
            className="bg-[var(--brand-blue-600)] text-white hover:bg-[var(--brand-blue-700)]"
          >
            {isLastStep ? "Got it" : "Next"}
          </Button>
        </div>
      </div>
    </motion.div>
  );
}
