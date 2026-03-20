import React, { useState, useEffect, useRef } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Alert, AlertDescription } from "../../components/ui/alert";
import { 
  Smartphone, 
  Copy, 
  CheckCircle, 
  AlertCircle, 
  ArrowLeft,
  Eye,
  EyeOff,
  QrCode,
} from "lucide-react";
import { toast } from "react-hot-toast";
import type { TOTPSetupData } from "../../types/webauthn";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface TOTPSetupComponentProps {
  contextType: "admin" | "oidc";
  email: string;
  tenantId: string;
  totpData?: TOTPSetupData | null;
  onSuccess?: () => void;
  onError?: (error: string) => void;
  onBack?: () => void;
  onSetup?: () => Promise<boolean>;
  onConfirm?: (code: string) => Promise<boolean>;
}

/**
 * TOTP Setup Component
 * 
 * Context-agnostic component that handles TOTP (Time-based One-Time Password) setup for authenticator apps.
 * Displays QR code and manual entry secret, then verifies setup with user code.
 * 
 * Shown when: selectedMFAMethod: "totp" and currentStep: "totp_setup"
 */
export function TOTPSetupComponent({ 
  contextType,
  email,
  tenantId,
  totpData,
  onSuccess,
  onError,
  onBack,
  onSetup,
  onConfirm
}: TOTPSetupComponentProps) {

  const [setupStep, setSetupStep] = useState<"loading" | "scan" | "verify" | "success">("loading");
  const [verificationCode, setVerificationCode] = useState("");
  const [showSecret, setShowSecret] = useState(false);
  const [isVerifying, setIsVerifying] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  // Initialize TOTP setup on component mount (guard against repeated calls)
  const initializedRef = useRef(false);
  useEffect(() => {
    // Only kick off setup once per mount
    if (!initializedRef.current) {
      initializedRef.current = true;
      if (!totpData && onSetup) {
        onSetup()
          .then((success) => {
            if (success) {
              setSetupStep("scan");
            } else {
              setSetupStep("loading");
              setErrorMessage("Failed to initialize TOTP setup");
              onError?.("Failed to initialize TOTP setup");
            }
          })
          .catch((error) => {
            setSetupStep("loading");
            setErrorMessage(error.message || "Failed to initialize TOTP setup");
            onError?.(error.message || "Failed to initialize TOTP setup");
          });
      } else if (totpData) {
        setSetupStep("scan");
      }
    } else if (totpData) {
      // If setup data arrives later, advance to scan once
      setSetupStep((prev) => (prev === "loading" ? "scan" : prev));
    }
    // Intentionally exclude function props to avoid effect retriggers on each render
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [totpData]);

  // Handle prop-based error messages
  useEffect(() => {
    // Error handling is done through onError callback
  }, []);

  const handleCopySecret = async () => {
    if (totpData?.manual_entry) {
      try {
        await navigator.clipboard.writeText(totpData.manual_entry);
        toast.success("Secret copied to clipboard");
      } catch (error) {
        toast.error("Failed to copy secret");
      }
    }
  };

  const handleVerifyCode = async () => {
    if (!verificationCode.trim()) {
      setErrorMessage("Please enter the verification code");
      return;
    }

    if (verificationCode.length !== 6) {
      setErrorMessage("Verification code must be 6 digits");
      return;
    }

    setIsVerifying(true);
    setErrorMessage(null);

    try {
      if (onConfirm) {
        const success = await onConfirm(verificationCode);
        if (success) {
          setSetupStep("success");
          onSuccess?.();
        } else {
          setErrorMessage("Invalid verification code. Please try again.");
          onError?.("Invalid verification code. Please try again.");
        }
      } else {
        setErrorMessage("Verification not available");
        onError?.("Verification not available");
      }
    } catch (error: any) {
      const errorMsg = error.message || "Failed to verify code. Please try again.";
      setErrorMessage(errorMsg);
      onError?.(errorMsg);
    } finally {
      setIsVerifying(false);
    }
  };

  const handleBack = () => {
    onBack?.(); // Use the provided back handler
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && verificationCode.length === 6) {
      handleVerifyCode();
    }
  };

  if (setupStep === "loading") {
    return (
      <div className="space-y-6">
        <Button
          variant="ghost"
          onClick={handleBack}
          className="mb-4 p-0 h-auto font-normal text-slate-600 hover:text-slate-900"
        >
          <ArrowLeft className="h-4 w-4 mr-2" />
          Back to authentication methods
        </Button>

        <div className="flex justify-center">
          <Smartphone className="h-12 w-12 text-slate-900 animate-pulse" />
        </div>
        <AuthStepHeader
          align="center"
          title="Setting up authenticator..."
          subtitle="Preparing your TOTP setup..."
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Back Button */}
      <Button
        variant="ghost"
        onClick={handleBack}
        className="mb-4 p-0 h-auto font-normal text-slate-600 hover:text-slate-900"
        disabled={isVerifying}
      >
        <ArrowLeft className="h-4 w-4 mr-2" />
        Back to authentication methods
      </Button>

      <div className="flex justify-center">
        <Smartphone className="h-12 w-12 text-slate-900" />
      </div>
      <AuthStepHeader
        align="center"
        title={
          setupStep === "scan"
            ? "Set up authenticator app"
            : setupStep === "verify"
              ? "Verify your setup"
              : "Authenticator app enabled!"
        }
        subtitle={
          setupStep === "scan"
            ? "Scan the QR code with your authenticator app or enter the secret manually."
            : setupStep === "verify"
              ? "Enter the 6-digit code from your authenticator app to complete setup."
              : "You can now use your authenticator app to sign in securely."
        }
        meta={
          email ? (
            <>
              Setting up for: <span className="font-semibold text-slate-900">{email}</span>
            </>
          ) : undefined
        }
      />

      {setupStep === "scan" && totpData && (
        <div className="space-y-6">
          <div className="text-center">
            <div className="inline-block p-4 bg-white rounded-lg border">
              <img 
                src={`data:image/png;base64,${totpData.qr_code}`}
                alt="TOTP QR Code"
                className="w-48 h-48 mx-auto"
              />
            </div>
            <p className="text-sm text-slate-700 mt-3">
              Scan this QR code with your authenticator app
            </p>
          </div>

          <div className="space-y-3 border-t border-[var(--auth-shell-border)] pt-4">
            <div className="flex items-center justify-between mb-3">
              <Label className="text-sm font-medium">Manual entry key</Label>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowSecret(!showSecret)}
                className="h-8 px-2"
              >
                {showSecret ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </Button>
            </div>
            
            <div className="flex items-center space-x-2">
              <Input
                value={showSecret ? totpData.manual_entry : "••••••••••••••••••••••••••••••••"}
                readOnly
                className="font-mono text-xs bg-background"
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handleCopySecret}
                className="flex-shrink-0"
              >
                <Copy className="h-4 w-4" />
              </Button>
            </div>
            
            <div className="mt-3 text-xs text-slate-700 space-y-1">
              <p><strong>Account:</strong> {totpData.account}</p>
              <p><strong>Issuer:</strong> {totpData.issuer}</p>
            </div>
          </div>

          <Button 
            onClick={() => setSetupStep("verify")}
            className="w-full h-12 text-base rounded-xl"
          >
            I've added the account
          </Button>
        </div>
      )}

      {setupStep === "verify" && (
        <div className="space-y-6">
          <div className="space-y-4">
            <div>
              <Label htmlFor="verification-code" className="text-sm font-medium">
                Enter the 6-digit code from your app
              </Label>
              <Input
                id="verification-code"
                type="text"
                placeholder="000000"
                value={verificationCode}
                onChange={(e) => {
                  const value = e.target.value.replace(/\D/g, '').slice(0, 6);
                  setVerificationCode(value);
                  setErrorMessage(null);
                }}
                onKeyPress={handleKeyPress}
                className="mt-2 text-center text-2xl font-mono tracking-widest h-14"
                maxLength={6}
              />
            </div>

            {errorMessage && (
              <Alert className="border-red-200 bg-red-50">
                <AlertCircle className="h-4 w-4 text-red-600" />
                <AlertDescription className="text-red-800">
                  {errorMessage}
                </AlertDescription>
              </Alert>
            )}

            <div className="space-y-3">
              <Button 
                onClick={handleVerifyCode}
                className="w-full h-12 text-base rounded-xl"
                disabled={verificationCode.length !== 6 || isVerifying}
              >
                {isVerifying ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current mr-2" />
                    Verifying...
                  </>
                ) : (
                  "Verify and complete setup"
                )}
              </Button>

              <Button 
                onClick={() => setSetupStep("scan")}
                variant="outline"
                className="w-full rounded-xl"
                disabled={isVerifying}
              >
                Back to QR code
              </Button>
            </div>
          </div>
        </div>
      )}

      {setupStep === "success" && (
        <div className="text-center space-y-4">
          <div className="mx-auto w-16 h-16 bg-green-50 rounded-full flex items-center justify-center mb-4">
            <CheckCircle className="h-8 w-8 text-green-600" />
          </div>
          <div className="bg-green-50 border border-green-200 rounded-lg p-4">
            <p className="text-green-800 text-sm">
              Your authenticator app is now set up and ready to use!
            </p>
          </div>
        </div>
      )}

      {setupStep === "scan" && (
        <div className="border-t border-[var(--auth-shell-border)] pt-4">
          <div className="flex items-start space-x-3">
            <QrCode className="h-5 w-5 text-amber-700 flex-shrink-0 mt-0.5" />
            <div className="text-sm">
              <p className="font-medium text-amber-900 mb-2">
                Popular authenticator apps:
              </p>
              <ul className="text-amber-800 space-y-1">
                <li>• Google Authenticator</li>
                <li>• Microsoft Authenticator</li>
                <li>• Authy</li>
                <li>• 1Password</li>
              </ul>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
