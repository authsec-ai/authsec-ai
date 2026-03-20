import React, { useEffect, useState } from "react";
import { Loader2 } from "lucide-react";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface AuthSuccessComponentProps {
  onContinue?: () => void;
  redirectDelay?: number;
}

/**
 * Authentication Success Component
 * 
 * Displays success confirmation after completing WebAuthn setup or authentication.
 * Provides a smooth transition to the main application with automatic redirect.
 * 
 * Shown when: currentStep: "completed"
 */
export function AuthSuccessComponent({ onContinue, redirectDelay = 3000 }: AuthSuccessComponentProps) {
  const [secondsLeft, setSecondsLeft] = useState(Math.ceil(redirectDelay / 1000));

  useEffect(() => {
    if (redirectDelay <= 0) {
      onContinue?.();
      return;
    }
    const tick = setInterval(() => {
      setSecondsLeft((s) => (s > 0 ? s - 1 : 0));
    }, 1000);
    const timer = setTimeout(() => {
      onContinue?.();
    }, redirectDelay);
    return () => {
      clearInterval(tick);
      clearTimeout(timer);
    };
  }, [redirectDelay, onContinue]);

  return (
    <div className="space-y-4 py-6 text-center">
      <AuthStepHeader
        align="center"
        title="Login successful"
        subtitle={
          <>Redirecting{secondsLeft > 0 ? ` in ${secondsLeft}s` : "..."}.</>
        }
      />
      <div className="flex items-center justify-center text-slate-700">
        <Loader2 className="h-5 w-5 animate-spin mr-2" />
        Finalizing secure session...
      </div>
    </div>
  );
}
