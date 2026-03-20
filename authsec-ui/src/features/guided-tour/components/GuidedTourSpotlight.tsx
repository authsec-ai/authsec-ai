import { motion } from 'framer-motion';
import type { SpotlightPosition } from '../types';

interface GuidedTourSpotlightProps {
  position: SpotlightPosition;
}

/**
 * Spotlight component that highlights the target element
 * Uses framer-motion for smooth transitions between positions
 */
export function GuidedTourSpotlight({ position }: GuidedTourSpotlightProps) {
  return (
    <motion.div
      className="absolute pointer-events-none"
      style={{
        left: position.x,
        top: position.y,
        width: position.width,
        height: position.height,
        borderRadius: position.borderRadius,
      }}
      initial={{ scale: 0.9, opacity: 0 }}
      animate={{ scale: 1, opacity: 1 }}
      exit={{ scale: 0.9, opacity: 0 }}
      transition={{
        duration: 0.4,
        ease: [0.4, 0, 0.2, 1], // Material Design easing
      }}
    >
      {/* Highlight border with glow effect */}
      <div
        className="absolute inset-0 rounded-[inherit] ring-2 ring-white/60 ring-offset-2 ring-offset-blue-950/90 shadow-lg shadow-white/20"
        style={{
          borderRadius: position.borderRadius,
        }}
      />

      {/* Pulsing animation for extra emphasis */}
      <motion.div
        className="absolute inset-0 rounded-[inherit] ring-1 ring-white/40"
        style={{
          borderRadius: position.borderRadius,
        }}
        animate={{
          scale: [1, 1.05, 1],
          opacity: [0.6, 0.3, 0.6],
        }}
        transition={{
          duration: 2,
          repeat: Infinity,
          ease: 'easeInOut',
        }}
      />
    </motion.div>
  );
}
