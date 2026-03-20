import { useEffect } from "react";

/**
 * Persist & restore scroll position for a given key using sessionStorage.
 * Helpful for complex dashboards where navigating away & back should restore the list position.
 */
export function useScrollRestore<T extends HTMLElement>(
  ref: React.RefObject<T | null>,
  storageKey: string
) {
  useEffect(() => {
    const node = ref.current;
    if (!node) return;

    // Restore on mount
    const stored = sessionStorage.getItem(`scroll:${storageKey}`);
    if (stored) {
      try {
        const pos = parseInt(stored, 10);
        if (!Number.isNaN(pos)) {
          node.scrollTop = pos;
        }
      } catch (_) {}
    }

    const handle = () => {
      sessionStorage.setItem(`scroll:${storageKey}`, String(node.scrollTop));
    };

    node.addEventListener("scroll", handle);
    return () => node.removeEventListener("scroll", handle);
  }, [ref, storageKey]);
}
 