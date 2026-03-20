import { useEffect, useRef } from "react";
import { useSidebar } from "@/components/ui/sidebar";
import { useResponsiveLayout } from "@/hooks/use-mobile";

interface ResponsiveSidebarControllerProps {
  isRightSidebarOpen?: boolean;
  rightSidebarWidth?: number;
}

/**
 * Invisible component that handles responsive sidebar behavior
 * without interfering with manual sidebar controls
 */
export function ResponsiveSidebarController({
  isRightSidebarOpen = false,
  rightSidebarWidth = 400,
}: ResponsiveSidebarControllerProps) {
  const { width, isMobile } = useResponsiveLayout();
  const { open, setOpen } = useSidebar();
  const lastManualToggleRef = useRef<number>(0);

  // Calculate effective available width when right sidebar is open
  const effectiveWidth = isRightSidebarOpen ? width - rightSidebarWidth : width;

  // Determine if we should auto-collapse based on effective width
  const shouldAutoCollapseSidebar = effectiveWidth < 1200;
  const hasConstrainedSpace = effectiveWidth < 1400;

  // Track manual toggles to avoid interfering with user actions
  useEffect(() => {
    const handleManualToggle = () => {
      lastManualToggleRef.current = Date.now();
    };

    // Listen for manual sidebar triggers
    const triggers = document.querySelectorAll('[data-sidebar="trigger"]');
    triggers.forEach((trigger) => {
      trigger.addEventListener("click", handleManualToggle);
    });

    return () => {
      triggers.forEach((trigger) => {
        trigger.removeEventListener("click", handleManualToggle);
      });
    };
  }, []);

  // Auto-collapse sidebar when screen space becomes constrained
  useEffect(() => {
    // Don't interfere if user manually toggled recently (within 2 seconds)
    const timeSinceManualToggle = Date.now() - lastManualToggleRef.current;
    if (timeSinceManualToggle < 2000) {
      return;
    }

    if (!isMobile) {
      if (shouldAutoCollapseSidebar && open) {
        setOpen(false);
      } else if (!shouldAutoCollapseSidebar && !hasConstrainedSpace && !open) {
        // Only auto-expand if we have plenty of space (>1400px effective width)
        setOpen(true);
      }
    }
  }, [
    shouldAutoCollapseSidebar,
    hasConstrainedSpace,
    isMobile,
    open,
    setOpen,
    isRightSidebarOpen,
    rightSidebarWidth,
  ]);

  // This component renders nothing
  return null;
}
