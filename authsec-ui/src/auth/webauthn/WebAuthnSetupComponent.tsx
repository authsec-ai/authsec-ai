import React, { useState, useEffect } from "react";
// Card components removed - using clean div-based design
import { Button } from "../../components/ui/button";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { useWebAuthnAuth } from "./useWebAuthnAuth";
import { 
  Fingerprint, 
  Smartphone, 
  Shield, 
  CheckCircle, 
  AlertCircle, 
  ArrowLeft,
  Loader2
} from "lucide-react";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface WebAuthnSetupComponentProps {
  contextType: "admin" | "oidc";
  email: string;
  tenantId: string;
  onSuccess?: (token: string) => void;
  onError?: (error: string) => void;
  onBack?: () => void;
  onSetup?: (email: string, tenantId: string) => Promise<any>;
}

/**
 * WebAuthn Setup Component
 * 
 * Handles biometric/security key registration for new users.
 * Pure component - flow logic handled by parent page
 */
export function WebAuthnSetupComponent({ contextType, email, tenantId, onSuccess, onError, onBack, onSetup }: WebAuthnSetupComponentProps) {
  const { 
    registerUser, 
    handleCallback,
    isRegistering,
    isHandlingCallback
  } = useWebAuthnAuth();

  const [setupState, setSetupState] = useState<"ready" | "waiting" | "success" | "error">("ready");
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Reset state when component mounts
  useEffect(() => {
    setSetupState("ready");
    setErrorMessage(null);
  }, []);

  const handleSetupWebAuthn = async () => {
    setSetupState("waiting");
    setErrorMessage(null);

    try {
      if (onSetup) {
        // Use the context-specific setup function
        await onSetup(email, tenantId);
        setSetupState("success");
      } else {
        // Fallback to local setup logic
        await registerUser(email, tenantId);
        
        const callbackResult = await handleCallback(email, tenantId);
        
        if (callbackResult.success && callbackResult.token) {
          setSetupState("success");
          onSuccess?.(callbackResult.token);
        } else {
          throw new Error(callbackResult.error || 'No token received');
        }
      }
      
    } catch (error: any) {
      setSetupState("error");
      let errorMsg: string;
      
      if (error.name === 'NotSupportedError') {
        errorMsg = "Biometric authentication is not supported on this device or browser";
      } else if (error.name === 'NotAllowedError') {
        errorMsg = "Permission was denied. Please try again and allow access to your biometric sensor";
      } else if (error.name === 'InvalidStateError') {
        errorMsg = "A credential already exists for this device";
      } else if (error.name === 'AbortError') {
        errorMsg = "The operation was cancelled";
      } else {
        errorMsg = error.message || "Failed to setup biometric authentication. Please try again";
      }
      
      setErrorMessage(errorMsg);
      onError?.(errorMsg);
    }
  };

  const handleBack = () => {
    onBack && onBack(); // Go back to previous step
  };

  const getSetupIcon = () => {
    switch (setupState) {
      case "waiting":
        return <Loader2 className="h-12 w-12 animate-spin text-blue-600" />;
      case "success":
        return <CheckCircle className="h-12 w-12 text-green-600" />;
      case "error":
        return <AlertCircle className="h-12 w-12 text-red-600" />;
      default:
        return <Fingerprint className="h-12 w-12 text-blue-600" />;
    }
  };

  const getSetupTitle = () => {
    switch (setupState) {
      case "waiting":
        return "Setting up biometric authentication...";
      case "success":
        return "Biometric authentication enabled!";
      case "error":
        return "Setup failed";
      default:
        return "Enable biometric authentication";
    }
  };

  const getSetupDescription = () => {
    switch (setupState) {
      case "waiting":
        return "Please follow the prompts on your device to complete the setup.";
      case "success":
        return "You can now use your fingerprint, face, or security key to sign in securely.";
      case "error":
        return errorMessage || "Something went wrong during setup.";
      default:
        return "Use your device's built-in biometric sensor or security key for secure, password-free authentication.";
    }
  };

  return (
    <div className="space-y-5">
      <Button
        variant="ghost"
        onClick={handleBack}
        className="p-0 h-auto font-normal text-slate-600 hover:text-slate-900"
        disabled={isRegistering || isHandlingCallback || setupState === "waiting"}
      >
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to authentication methods
      </Button>

      <div className="flex justify-center">{getSetupIcon()}</div>

      <AuthStepHeader
        align="center"
        title={getSetupTitle()}
        subtitle={getSetupDescription()}
        meta={
          email ? (
            <>
              Setting up for: <span className="font-semibold text-slate-900">{email}</span>
            </>
          ) : undefined
        }
      />

      {setupState === "ready" && (
        <div className="space-y-4">
          <div className="space-y-3 border-t border-[var(--auth-shell-border)] pt-4">
            <h4 className="font-semibold text-slate-900 mb-3 flex items-center">
              <Shield className="h-4 w-4 mr-2" />
              What happens next
            </h4>
            <ul className="space-y-2 text-sm text-slate-700">
              <li className="flex items-start gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-slate-900 text-white rounded-full flex items-center justify-center text-xs font-medium">
                  1
                </span>
                Your browser will ask for permission to use biometric authentication.
              </li>
              <li className="flex items-start gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-slate-900 text-white rounded-full flex items-center justify-center text-xs font-medium">
                  2
                </span>
                Follow the prompts to register your fingerprint, face, or security key.
              </li>
              <li className="flex items-start gap-3">
                <span className="flex-shrink-0 w-6 h-6 bg-slate-900 text-white rounded-full flex items-center justify-center text-xs font-medium">
                  3
                </span>
                Your device will securely store your biometric data locally.
              </li>
            </ul>
          </div>

          <Button
            onClick={handleSetupWebAuthn}
            className="w-full h-11 text-base rounded-xl"
            disabled={isRegistering || isHandlingCallback}
          >
            <Fingerprint className="h-4 w-4 mr-2" />
            Set up biometric authentication
          </Button>
        </div>
      )}

      {setupState === "waiting" && (
        <div className="space-y-4 text-center">
          <div className="flex items-center justify-center gap-2 border-t border-[var(--auth-shell-border)] pt-4 text-sm text-amber-900">
            <Smartphone className="h-4 w-4" />
            <p>Check your device for biometric authentication prompts.</p>
          </div>
          <p className="text-sm text-slate-700">This may take a few seconds...</p>
          <Button
            variant="outline"
            onClick={handleBack}
            className="text-sm"
            disabled={isRegistering || isHandlingCallback}
          >
            Cancel
          </Button>
        </div>
      )}

      {setupState === "error" && (
        <div className="space-y-3">
          <Alert className="bg-red-50 border-red-200 text-red-800">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>{errorMessage}</AlertDescription>
          </Alert>
          <div className="flex gap-3">
            <Button
              variant="outline"
              onClick={handleBack}
              className="w-full rounded-xl"
              disabled={isRegistering || isHandlingCallback}
            >
              Back
            </Button>
            <Button
              onClick={handleSetupWebAuthn}
              className="w-full rounded-xl"
              disabled={isRegistering || isHandlingCallback}
            >
              Try again
            </Button>
          </div>
        </div>
      )}

      {setupState === "success" && (
        <div className="text-center space-y-3">
          <Alert className="bg-green-50 border-green-200 text-green-800">
            <CheckCircle className="h-4 w-4" />
            <AlertDescription>
              Biometric authentication has been set up successfully. You can now use it to sign in.
            </AlertDescription>
          </Alert>
          <Button onClick={handleBack} className="w-full rounded-xl">
            Continue
          </Button>
        </div>
      )}

      <div className="auth-callout mt-4">
        <div className="flex items-start space-x-3">
          <Shield className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
          <div className="text-sm">
            <p className="font-medium text-amber-900 mb-1">Your privacy is protected</p>
            <p className="text-amber-800">
              Your biometric data never leaves your device and is not stored on our servers.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
