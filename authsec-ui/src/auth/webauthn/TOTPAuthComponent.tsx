import React, { useState, useEffect, useRef } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { 
  Smartphone, 
  AlertCircle, 
  RefreshCw,
  Shield,
  Clock
} from "lucide-react";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface TOTPAuthComponentProps {
  contextType: "admin" | "oidc";
  email: string;
  tenantId: string;
  onSuccess?: () => void;
  onError?: (error: string) => void;
  onAuthenticate?: (code: string) => Promise<boolean>;
}

/**
 * TOTP Authentication Component
 * 
 * Context-agnostic component that handles TOTP (Time-based One-Time Password) authentication for returning users.
 * Provides code input and verification with helpful UI feedback.
 * 
 * Shown when: first_login: false, currentStep: "authentication", and using TOTP
 */
export function TOTPAuthComponent({ 
  contextType,
  email,
  tenantId,
  onSuccess,
  onError,
  onAuthenticate
}: TOTPAuthComponentProps) {

  const [verificationCode, setVerificationCode] = useState("");
  const [isVerifying, setIsVerifying] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);
  const [timeLeft, setTimeLeft] = useState(30);
  const inputRef = useRef<HTMLInputElement>(null);

  // Focus input on mount
  useEffect(() => {
    if (inputRef.current) {
      inputRef.current.focus();
    }
  }, []);

  // Handle authentication errors - removed undefined variable
  // Errors are now handled through the onError prop when authentication fails

  // TOTP countdown timer (approximate)
  useEffect(() => {
    const interval = setInterval(() => {
      setTimeLeft(prev => {
        if (prev <= 1) {
          return 30; // Reset to 30 seconds
        }
        return prev - 1;
      });
    }, 1000);

    return () => clearInterval(interval);
  }, []);

  const handleVerifyCode = async () => {
    if (!verificationCode.trim()) {
      setErrorMessage("Please enter the verification code from your authenticator app");
      return;
    }

    if (verificationCode.length !== 6) {
      setErrorMessage("Verification code must be 6 digits");
      return;
    }

    setIsVerifying(true);
    setErrorMessage(null);

    try {
      if (onAuthenticate) {
        const success = await onAuthenticate(verificationCode);
        if (success) {
          onSuccess?.();
        } else {
          setErrorMessage("Invalid verification code. Please check your authenticator app and try again.");
          onError?.("Invalid verification code. Please check your authenticator app and try again.");
          setRetryCount(prev => prev + 1);
          setVerificationCode("");
          // Re-focus input for next attempt
          setTimeout(() => {
            if (inputRef.current) {
              inputRef.current.focus();
            }
          }, 100);
        }
      } else {
        setErrorMessage("Authentication not available");
        onError?.("Authentication not available");
      }
    } catch (error: any) {
      const errorMsg = error.message || "Authentication failed. Please try again.";
      setErrorMessage(errorMsg);
      onError?.(errorMsg);
      setRetryCount(prev => prev + 1);
      setVerificationCode("");
    } finally {
      setIsVerifying(false);
    }
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && verificationCode.length === 6 && !isVerifying) {
      handleVerifyCode();
    }
  };

  const handleCodeChange = (value: string) => {
    // Only allow numbers and limit to 6 digits
    const numericValue = value.replace(/\D/g, '').slice(0, 6);
    setVerificationCode(numericValue);
    setErrorMessage(null);
  };

  const getTimeLeftColor = () => {
    if (timeLeft <= 10) return "text-red-600";
    if (timeLeft <= 20) return "text-amber-600";
    return "text-green-600";
  };

  return (
    <div className="space-y-8">
      <div className="flex justify-center">
        <Smartphone className="h-10 w-10 text-slate-900" />
      </div>
      <AuthStepHeader
        align="center"
        title="Enter authenticator code"
        subtitle="Open your authenticator app and type the 6-digit code."
        meta={
          email ? (
            <>
              Signing in as: <span className="font-semibold text-slate-900">{email}</span>
            </>
          ) : undefined
        }
      />

      <div className="space-y-6">
        <div className="space-y-3">
          <div className="flex items-center justify-between text-sm">
            <Label htmlFor="verification-code" className="text-slate-700">
              6-digit code
            </Label>
            <span className="text-slate-700">From your authenticator app</span>
          </div>
          <Input
            ref={inputRef}
            id="verification-code"
            type="text"
            placeholder="000000"
            value={verificationCode}
            onChange={(e) => handleCodeChange(e.target.value)}
            onKeyPress={handleKeyPress}
            className="text-center text-3xl font-mono tracking-[0.45em] h-16 border-2 bg-background/80 shadow-inner"
            maxLength={6}
            disabled={isVerifying}
            autoComplete="one-time-code"
          />

          <div className="flex flex-col gap-3 border-t border-[var(--auth-shell-border)] pt-3 text-sm text-slate-600 sm:flex-row sm:items-center sm:justify-between">
            <div className="flex items-center gap-2">
              <Clock className="h-4 w-4 text-slate-700" />
              <span>Code refreshes in</span>
            </div>
            <div className="flex items-center gap-3">
              <div className="h-2 w-28 overflow-hidden rounded-full bg-slate-200/70">
                <div
                  className="h-full bg-gradient-to-r from-slate-600 via-slate-800 to-black transition-all duration-200"
                  style={{ width: `${Math.max(10, (timeLeft / 30) * 100)}%` }}
                />
              </div>
              <span className={`font-mono text-base font-semibold ${getTimeLeftColor()}`}>
                {timeLeft}s
              </span>
            </div>
          </div>
        </div>

        {errorMessage && (
          <Alert className="border-red-200 bg-red-50">
            <AlertCircle className="h-4 w-4 text-red-600" />
            <AlertDescription className="text-red-800">
              {errorMessage}
            </AlertDescription>
          </Alert>
        )}

        <Button
          onClick={handleVerifyCode}
          className="w-full h-12 text-base rounded-xl"
          disabled={verificationCode.length !== 6 || isVerifying}
        >
          {isVerifying ? (
            <>
              <RefreshCw className="h-4 w-4 mr-2 animate-spin" />
              Verifying...
            </>
          ) : (
            "Verify and sign in"
          )}
        </Button>

        <div className="grid gap-4 border-t border-[var(--auth-shell-border)] pt-4 sm:grid-cols-2">
          <div className="space-y-2">
            <div className="flex items-center gap-2 text-sm font-medium text-slate-900">
              <RefreshCw className="h-4 w-4 text-slate-700" />
              Quick tips
            </div>
            <ul className="mt-2 space-y-2 text-sm text-slate-700">
              <li>• Use the freshest code; wait for the next one if the timer is low.</li>
              <li>• Ensure your phone clock is set to automatic time.</li>
              <li>• Type digits without spaces exactly as shown in your authenticator.</li>
            </ul>
          </div>

          <div className="space-y-2">
            <div className="flex items-start gap-3">
              <Shield className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
              <div className="text-sm">
                <p className="font-medium text-amber-900 mb-1">
                  Keep your codes secure
                </p>
                <p className="text-amber-800">
                  Codes rotate every 30 seconds and should never be shared. If you suspect compromise, reset MFA in your account.
                </p>
              </div>
            </div>
          </div>
        </div>

        {retryCount >= 2 && (
          <div className="space-y-2 border-t border-[var(--auth-shell-border)] pt-4">
            <h4 className="mb-3 flex items-center text-sm font-medium text-slate-900">
              <AlertCircle className="mr-2 h-4 w-4 text-amber-600" />
              Still not working?
            </h4>
            <ul className="space-y-2 text-sm text-slate-700">
              <li>• Double-check you selected the right account inside your authenticator app.</li>
              <li>• Wait for the next code cycle and try again.</li>
              <li>• If the issue persists, restart your authenticator app and confirm device time sync.</li>
            </ul>
          </div>
        )}
      </div>
    </div>
  );
}
