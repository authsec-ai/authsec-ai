import React, { useEffect, useState } from "react";
import { Button } from "../../components/ui/button";
import { Badge } from "../../components/ui/badge";
import { Fingerprint, Smartphone, Shield, ChevronRight, CheckCircle } from "lucide-react";
import type { MFAMethod } from "../../app/api/webauthnApi";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface MFASelectionPageProps {
  contextType: "admin" | "oidc";
  availableMethods: MFAMethod[];
  onMethodSelect: (method: "webauthn" | "totp") => void;
  onGetMethods: () => void;
  email?: string;
  isLoading?: boolean;
}

/**
 * MFA Selection Page
 * 
 * Context-agnostic component that displays available MFA methods for first-time users.
 * Allows users to choose between WebAuthn (biometric) and TOTP (authenticator app).
 * 
 * Shown when: first_login: true and currentStep: "mfa_selection"
 */
export function MFASelectionPage({ 
  contextType,
  availableMethods,
  onMethodSelect,
  onGetMethods,
  email,
  isLoading = false
}: MFASelectionPageProps) {
  const [selectedMethod, setSelectedMethod] = useState<"webauthn" | "totp" | null>(null);

  // Load MFA methods on component mount
  useEffect(() => {
    if (!availableMethods || availableMethods.length === 0) {
      onGetMethods();
    }
  }, [availableMethods, onGetMethods]);

  const handleMethodSelect = (method: "webauthn" | "totp") => {
    setSelectedMethod(method);
    onMethodSelect(method);
  };

  const getMethodIcon = (type: string) => {
    switch (type) {
      case "webauthn":
        return <Fingerprint className="h-8 w-8" />;
      case "totp":
        return <Smartphone className="h-8 w-8" />;
      default:
        return <Shield className="h-8 w-8" />;
    }
  };

  const getMethodColor = (type: string, recommended: boolean) => {
    if (recommended) {
      return "text-blue-700 bg-blue-50 border-blue-200";
    }
    return "text-foreground bg-muted/30 border-border";
  };

  if (isLoading && (!availableMethods || availableMethods.length === 0)) {
    return (
      <div className="space-y-6">
        <AuthStepHeader
          align="center"
          title="Setting up security"
          subtitle="Loading available authentication methods..."
        />
        <div className="flex items-center justify-center py-8">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-slate-800" />
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <AuthStepHeader
        align="center"
        title="Secure your account"
        subtitle="Choose your preferred method to keep your account safe."
        meta={
          email ? (
            <>
              Setting up for: <span className="font-semibold text-slate-900">{email}</span>
            </>
          ) : undefined
        }
      />

      <div className="space-y-4">
        {availableMethods && availableMethods.map((method) => (
          <div key={method.type} className="relative">
            <div
              className={`relative cursor-pointer rounded-xl border px-5 py-4 transition-colors ${
                selectedMethod === method.type
                  ? "border-slate-900 bg-slate-50"
                  : "border-slate-300 bg-white hover:border-slate-500"
              }`}
              onClick={() => handleMethodSelect(method.type as "webauthn" | "totp")}
            >
              {method.recommended && (
                <div className="absolute -top-3 left-5 z-10">
                  <Badge className="bg-slate-900 text-white px-3 py-1 shadow-sm">
                    Recommended
                  </Badge>
                </div>
              )}

              <div className="flex items-start space-x-4">
                {/* Icon */}
                <div className={`p-3 rounded-xl border ${getMethodColor(method.type, method.recommended)}`}>
                  {getMethodIcon(method.type)}
                </div>

                {/* Content */}
                <div className="flex-1 min-w-0">
                  <div className="mb-2 flex items-center justify-between">
                    <h3 className="text-base font-semibold text-slate-900">
                      {method.display_name}
                    </h3>
                    {selectedMethod === method.type ? (
                      <CheckCircle className="h-5 w-5 text-slate-900" />
                    ) : (
                      <ChevronRight className="h-5 w-5 text-slate-400" />
                    )}
                  </div>

                  <p className="mb-3 text-sm leading-relaxed text-slate-600">
                    {method.description}
                  </p>

                  <div className="flex items-center space-x-2">
                    {method.enabled ? (
                      <Badge variant="outline" className="border-green-300 bg-green-50 text-green-700">
                        Available
                      </Badge>
                    ) : (
                      <Badge variant="outline" className="border-blue-300 bg-blue-50 text-blue-700">
                        Setup Required
                      </Badge>
                    )}

                    {method.recommended && (
                      <Badge variant="outline" className="border-blue-300 bg-blue-50 text-blue-700">
                        Most Secure
                      </Badge>
                    )}

                    {method.type === "totp" && (
                      <Badge variant="outline" className="border-gray-300 bg-gray-50 text-gray-700">
                        Works Offline
                      </Badge>
                    )}
                  </div>
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>

      {/* Continue Button */}
      {selectedMethod && (
        <div className="pt-4">
          <Button 
            className="w-full h-12 text-base rounded-xl"
            disabled={isLoading}
            onClick={() => selectedMethod && handleMethodSelect(selectedMethod)}
          >
            {isLoading ? (
              <>
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-current mr-2" />
                Setting up...
              </>
            ) : (
              <>
                Continue with {availableMethods?.find(m => m.type === selectedMethod)?.display_name}
                <ChevronRight className="h-4 w-4 ml-2" />
              </>
            )}
          </Button>
        </div>
      )}

      {/* Help Text */}
      <div className="text-center text-sm text-foreground">
        <p>
          You can always add more authentication methods later in your account settings.
        </p>
      </div>
    </div>
  );
}
