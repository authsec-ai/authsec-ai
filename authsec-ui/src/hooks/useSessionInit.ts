import { useEffect } from "react";
import { useDispatch } from "react-redux";
import { checkSession } from "../auth/slices/authSlice";
import { SessionManager } from "../utils/sessionManager";

/**
 * Hook to initialize session on app start
 * Checks for existing valid tokens and restores authentication state
 */
export const useSessionInit = () => {
  const dispatch = useDispatch();

  useEffect(() => {
    const initializeSession = () => {
      // Check if we have a valid session and restore state if needed
      dispatch(checkSession());
    };

    // Initialize on mount
    initializeSession();

    // Set up automatic session validation
    const setupPeriodicValidation = () => {
      // Check session every 5 minutes
      const interval = setInterval(() => {
        dispatch(checkSession());
      }, 5 * 60 * 1000);

      // Check when tab becomes visible
      const handleVisibilityChange = () => {
        if (!document.hidden) {
          dispatch(checkSession());
        }
      };

      document.addEventListener("visibilitychange", handleVisibilityChange);

      return () => {
        clearInterval(interval);
        document.removeEventListener("visibilitychange", handleVisibilityChange);
      };
    };

    // Only set up periodic validation if we have a valid session
    let cleanup: (() => void) | undefined;
    if (SessionManager.isSessionValid()) {
      cleanup = setupPeriodicValidation();
    }

    return cleanup;
  }, [dispatch]);
};