import React, { useEffect, useState, useCallback, useRef } from "react";
import { createPortal } from "react-dom";

interface NewClientSpotlightOverlayProps {
  isActive: boolean;
  onDismiss: () => void;
}

const PADDING = 4;   // small padding around row edges
const RADIUS = 4;    // match table row corner radius

/**
 * Full-screen dark overlay with a spotlight cutout on the current step button.
 * Uses the same SVG-mask technique as GuidedTourOverlay.
 * All pointer-events are `none` — a document-level click listener detects
 * outside-spotlight clicks and calls onDismiss.
 */
export function NewClientSpotlightOverlay({
  isActive,
  onDismiss,
}: NewClientSpotlightOverlayProps) {
  const [rect, setRect] = useState<DOMRect | null>(null);
  const [mounted, setMounted] = useState(false);
  const rectRef = useRef<DOMRect | null>(null);
  const onDismissRef = useRef(onDismiss);
  onDismissRef.current = onDismiss;

  useEffect(() => {
    setMounted(true);
  }, []);

  const updateRect = useCallback(() => {
    // Find the actions cell, then walk up to the <tr> for the full row highlight
    const cell = document.querySelector("[data-new-client='true']");
    const target = cell?.closest("tr") ?? cell;
    if (target) {
      const r = target.getBoundingClientRect();
      rectRef.current = r;
      setRect(r);
    } else {
      rectRef.current = null;
      setRect(null);
    }
  }, []);

  // Recalculate spotlight position when active state changes or window resizes/scrolls
  useEffect(() => {
    if (!isActive) {
      setRect(null);
      rectRef.current = null;
      return;
    }
    const t = setTimeout(updateRect, 150);
    window.addEventListener("resize", updateRect);
    window.addEventListener("scroll", updateRect, true);
    return () => {
      clearTimeout(t);
      window.removeEventListener("resize", updateRect);
      window.removeEventListener("scroll", updateRect, true);
    };
  }, [isActive, updateRect]);

  // Dismiss when user clicks outside the spotlight area
  useEffect(() => {
    if (!isActive) return;

    const handleClick = (e: MouseEvent) => {
      const target = e.target as HTMLElement;

      // Don't dismiss if clicking inside the step guidance popovers
      if (target.closest("[data-new-client-popover]")) return;

      const r = rectRef.current;
      if (!r) {
        onDismissRef.current();
        return;
      }

      const isInsideSpotlight =
        e.clientX >= r.left - PADDING &&
        e.clientX <= r.right + PADDING &&
        e.clientY >= r.top - PADDING &&
        e.clientY <= r.bottom + PADDING;

      if (!isInsideSpotlight) {
        onDismissRef.current();
      }
    };

    document.addEventListener("click", handleClick);
    return () => document.removeEventListener("click", handleClick);
  }, [isActive]);

  if (!mounted || !isActive || !rect) return null;

  const x = rect.left - PADDING;
  const y = rect.top - PADDING;
  const w = rect.width + PADDING * 2;
  const h = rect.height + PADDING * 2;

  return createPortal(
    <div
      className="fixed inset-0 pointer-events-none"
      style={{ zIndex: 48 }}
      aria-hidden="true"
    >
      <svg
        className="absolute inset-0"
        style={{ width: "100%", height: "100%" }}
      >
        <defs>
          <mask id="new-client-spotlight-mask">
            {/* White = show overlay; black = cut out (transparent window) */}
            <rect width="100%" height="100%" fill="white" />
            <rect x={x} y={y} width={w} height={h} rx={RADIUS} fill="black" />
          </mask>
          <filter id="new-client-backdrop-blur">
            <feGaussianBlur in="SourceGraphic" stdDeviation="3" />
          </filter>
        </defs>

        {/* Blurred dark layer */}
        <rect
          width="100%"
          height="100%"
          fill="rgba(23, 37, 84, 0.75)"
          filter="url(#new-client-backdrop-blur)"
          mask="url(#new-client-spotlight-mask)"
        />

        {/* Solid tint on top for depth */}
        <rect
          width="100%"
          height="100%"
          fill="rgba(23, 37, 84, 0.3)"
          mask="url(#new-client-spotlight-mask)"
        />
      </svg>

      {/* Blue glow border around the spotlight cutout */}
      <div
        className="absolute rounded-md"
        style={{
          left: x - 2,
          top: y - 2,
          width: w + 4,
          height: h + 4,
          boxShadow:
            "0 0 0 2px rgba(59,130,246,0.9), 0 0 0 5px rgba(59,130,246,0.25), 0 0 18px rgba(59,130,246,0.45)",
          borderRadius: RADIUS + 2,
          pointerEvents: "none",
        }}
      />
    </div>,
    document.body,
  );
}
