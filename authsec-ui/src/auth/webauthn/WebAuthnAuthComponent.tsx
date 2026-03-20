import React, { useState, useEffect } from "react";
// Card components removed - using clean div-based design
import { Button } from "../../components/ui/button";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { useWebAuthnAuth } from "./useWebAuthnAuth";
import { 
  Fingerprint, 
  Shield, 
  AlertCircle, 
  Loader2,
  Smartphone,
  RefreshCw
} from "lucide-react";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface WebAuthnAuthComponentProps {
  contextType: "admin" | "oidc";
  email: string;
  tenantId: string;
  onSuccess?: (token: string) => void;
  onError?: (error: string) => void;
  onAuthenticate?: (email: string, tenantId: string) => Promise<any>;
}

/**
 * WebAuthn Authentication Component
 * 
 * Handles biometric authentication for returning users.
 * Pure component - flow logic handled by parent page
 */
export function WebAuthnAuthComponent({ contextType, email, tenantId, onSuccess, onError, onAuthenticate }: WebAuthnAuthComponentProps) {
  const { 
    authenticateUser, 
    handleCallback,
    isAuthenticating,
    isHandlingCallback
  } = useWebAuthnAuth();

  const [authState, setAuthState] = useState<"ready" | "waiting" | "error">("ready");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);

  const handleAuthenticate = async () => {
    setAuthState("waiting");
    setErrorMessage(null);

    try {
      if (onAuthenticate) {
        // Prefer parent-provided auth flow (e.g., OIDC context handles callback + token display)
        const ok = await onAuthenticate(email, tenantId);
        if (!ok) {
          throw new Error('Authentication failed');
        }
      } else {
        // Local fallback: perform WebAuthn + callback via hook
        await authenticateUser(email, tenantId);
        const callbackResult = await handleCallback(email, tenantId);
        if (callbackResult.success && callbackResult.token) {
          onSuccess?.(callbackResult.token);
        } else {
          throw new Error(callbackResult.error || 'No token received');
        }
      }
      
    } catch (error: any) {
      setAuthState("error");
      setRetryCount(prev => prev + 1);
      
      const errorMsg = getErrorMessage(error);
      setErrorMessage(errorMsg);
      onError?.(errorMsg);
    }
  };

  const getErrorMessage = (error: any): string => {
    if (error.name === 'NotAllowedError') {
      return "Authentication was cancelled. Please try again and allow access to your biometric sensor.";
    } else if (error.name === 'InvalidStateError') {
      return "No registered credentials found. Please contact support or use an alternative authentication method.";
    } else if (error.name === 'AbortError') {
      return "Authentication timed out. Please try again.";
    } else if (error.name === 'NotSupportedError') {
      return "Biometric authentication is not supported on this device or browser.";
    } else {
      return error.message || "Authentication failed. Please try again or use an alternative method.";
    }
  };

  // Auto-trigger authentication on component mount
  useEffect(() => {
    // Small delay to ensure smooth UI transition
    const timer = setTimeout(() => {
      if (authState === "ready") {
        handleAuthenticate();
      }
    }, 500);

    return () => clearTimeout(timer);
  }, []); // Only run on mount

  const getAuthIcon = () => {
    switch (authState) {
      case "waiting":
        return <Loader2 className="h-12 w-12 animate-spin text-blue-600" />;
      case "error":
        return <AlertCircle className="h-12 w-12 text-red-600" />;
      default:
        return <Fingerprint className="h-12 w-12 text-blue-600" />;
    }
  };

  const getAuthTitle = () => {
    switch (authState) {
      case "waiting":
        return "Authenticating...";
      case "error":
        return "Authentication failed";
      default:
        return "Welcome back!";
    }
  };

  const getAuthDescription = () => {
    switch (authState) {
      case "waiting":
        return "Please use your biometric sensor or security key to continue.";
      case "error":
        return errorMessage || "Something went wrong during authentication.";
      default:
        return "Use your fingerprint, face, or security key to sign in securely.";
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex justify-center">{getAuthIcon()}</div>
      <AuthStepHeader
        align="center"
        title={getAuthTitle()}
        subtitle={getAuthDescription()}
        meta={
          email ? (
            <>
              Signing in as: <span className="font-semibold text-slate-900">{email}</span>
            </>
          ) : undefined
        }
      />

      {authState === "ready" && (
        <Button
          onClick={handleAuthenticate}
          className="w-full h-11 text-base rounded-xl"
          disabled={isAuthenticating || isHandlingCallback}
        >
          <Fingerprint className="h-4 w-4 mr-2" />
          Use biometric authentication
        </Button>
      )}

      {authState === "waiting" && (
        <div className="text-center space-y-4">
          <div className="flex items-center justify-center gap-3 border-t border-[var(--auth-shell-border)] pt-4 text-sm text-slate-800">
            <Smartphone className="h-5 w-5 text-slate-700" />
            <p>Check your device for authentication prompts</p>
          </div>

          <p className="text-sm text-slate-700">
            Touch your fingerprint sensor, look at your camera, or insert your security key.
          </p>

          <Button
            variant="outline"
            onClick={() => setAuthState("ready")}
            className="text-sm"
            disabled={isAuthenticating || isHandlingCallback}
          >
            Cancel
          </Button>
        </div>
      )}

      {authState === "error" && (
        <div className="space-y-4">
          <Alert className="border-red-200 bg-red-50">
            <AlertCircle className="h-4 w-4 text-red-600" />
            <AlertDescription className="text-red-800">
              {errorMessage}
            </AlertDescription>
          </Alert>

          <div className="space-y-3">
            <Button
              onClick={handleAuthenticate}
              className="w-full rounded-xl"
              disabled={isAuthenticating || isHandlingCallback}
            >
              <RefreshCw className="h-4 w-4 mr-2" />
              Try again
            </Button>

            {retryCount >= 2 && (
              <div className="text-center">
                <p className="text-sm text-slate-700 mb-3">
                  Having trouble with biometric authentication?
                </p>
                <Button
                  variant="outline"
                  className="w-full rounded-xl"
                  onClick={() => {
                    // This would need to trigger TOTP authentication instead
                    // For now, we'll just show the option
                  }}
                >
                  Use authenticator app instead
                </Button>
              </div>
            )}
          </div>
        </div>
      )}

      <div className="mt-2 border-t border-[var(--auth-shell-border)] pt-4">
        <div className="flex items-start space-x-3">
          <Shield className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
          <div className="text-sm">
            <p className="font-medium text-amber-900 mb-1">Secure authentication</p>
            <p className="text-amber-800">
              Your biometric data is processed locally on your device and never shared with our servers.
            </p>
          </div>
        </div>
      </div>

      {authState === "error" && retryCount >= 1 && (
        <div className="border-t border-[var(--auth-shell-border)] pt-4">
          <h4 className="font-medium text-slate-900 mb-3">Troubleshooting</h4>
          <ul className="text-sm text-slate-700 space-y-2">
            <li>• Make sure your device supports biometric authentication</li>
            <li>• Check that your browser allows biometric authentication</li>
            <li>• Ensure your biometric sensor is clean and working</li>
            <li>• Try refreshing the page if the issue persists</li>
          </ul>
        </div>
      )}
    </div>
  );
}
