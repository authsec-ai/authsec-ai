/**
 * OIDC WebAuthn Router - For end-user authentication flow
 *
 * Orchestrates the OIDC WebAuthn authentication flow with token display
 */

import React, { useEffect } from "react";
import { useDispatch } from "react-redux";
import { setCurrentStep } from "../slices/oidcWebAuthnSlice";
import { useEndUserAuth } from "../context/EndUserAuthContext";
import { MFASelectionPage } from "../webauthn/MFASelectionPage";
import { WebAuthnSetupComponent } from "../webauthn/WebAuthnSetupComponent";
import { TOTPSetupComponent } from "../webauthn/TOTPSetupComponent";
import { WebAuthnAuthComponent } from "../webauthn/WebAuthnAuthComponent";
import { TOTPAuthComponent } from "../webauthn/TOTPAuthComponent";
import { OIDCTokenDisplayComponent } from "./OIDCTokenDisplayComponent";
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

interface OIDCWebAuthnRouterProps {
  onAuthComplete: () => void;
  onAuthError?: (error: string) => void;
  onTokenDisplay?: (token: string) => void;
}

/**
 * Inner router component that uses OIDC context
 */
export function OIDCWebAuthnRouter({
  onAuthComplete: _onAuthComplete,
  onAuthError,
  onTokenDisplay,
}: OIDCWebAuthnRouterProps) {
  const oidcWebauthn = useEndUserAuth();
  const dispatch = useDispatch();

  // Handle token display without re-triggering callback
  useEffect(() => {
    console.log("🔄 OIDC WebAuthn step changed to:", oidcWebauthn.currentStep);

    if (oidcWebauthn.currentStep === "token_display" && oidcWebauthn.displayToken) {
      console.log("🎯 OIDC flow completed with token display");
      if (onTokenDisplay) {
        onTokenDisplay(oidcWebauthn.displayToken);
      }
      // Do not call onAuthComplete here to avoid duplicate webauthn-callback calls
    }
  }, [oidcWebauthn.currentStep, oidcWebauthn.displayToken, onTokenDisplay]);

  // Handle authentication errors
  useEffect(() => {
    if (oidcWebauthn.authenticationError && onAuthError) {
      onAuthError(oidcWebauthn.authenticationError);
    }
  }, [oidcWebauthn.authenticationError, onAuthError]);

  // If we arrive with user context set but step is still 'login', route to proper step
  useEffect(() => {
    if (oidcWebauthn.currentStep === "login" && oidcWebauthn.email && oidcWebauthn.tenantId) {
      const next = oidcWebauthn.isFirstLogin ? "mfa_selection" : "authentication";
      dispatch(setCurrentStep(next));
    }
  }, [
    oidcWebauthn.currentStep,
    oidcWebauthn.email,
    oidcWebauthn.tenantId,
    oidcWebauthn.isFirstLogin,
    dispatch,
  ]);

  // Prefetch MFA methods when entering authentication or selection
  useEffect(() => {
    if (!oidcWebauthn.email || !oidcWebauthn.tenantId) return;
    if (
      (oidcWebauthn.currentStep === "authentication" ||
        oidcWebauthn.currentStep === "mfa_selection") &&
      (!oidcWebauthn.availableMFAMethods || oidcWebauthn.availableMFAMethods.length === 0)
    ) {
      // Fire and forget; context handles errors/toasts
      void oidcWebauthn.getMFAMethods();
    }
  }, [oidcWebauthn.currentStep, oidcWebauthn.email, oidcWebauthn.tenantId]);

  // Render lightweight loader if step is still 'login' to avoid blank screen during transition
  if (oidcWebauthn.currentStep === "login") {
    return (
      <AuthSplitFrame
        valuePanel={
          <AuthValuePanel
            eyebrow="MFA Security"
            title="Preparing multi-factor authentication."
            subtitle="Routing your callback into available verification methods."
            points={[
              "Available WebAuthn/TOTP methods are fetched per user.",
              "Step routing remains inside the OIDC auth context.",
            ]}
          />
        }
      >
        <AuthActionPanel className="space-y-4">
          <AuthStepHeader
            title="Preparing authentication"
            subtitle="Loading your MFA context..."
          />
          <div className="auth-inline-note flex items-center gap-3 text-sm text-slate-700">
            <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-slate-700" />
            <span>Preparing authentication...</span>
          </div>
        </AuthActionPanel>
      </AuthSplitFrame>
    );
  }

  return (
    <AuthSplitFrame
      valuePanel={
        <AuthValuePanel
          eyebrow="MFA Security"
          title="Verify identity with strong factors."
          subtitle="This step secures callback completion with WebAuthn or TOTP and then hands back the access token."
          points={[
            "Method availability is tenant and user specific.",
            "Passkey and authenticator app setup are both supported.",
            "Successful verification transitions to token output.",
          ]}
          trustLabel={oidcWebauthn.email ? `User: ${oidcWebauthn.email}` : undefined}
        />
      }
    >
      <AuthActionPanel className="relative space-y-4">
        {oidcWebauthn.currentStep === "mfa_selection" && (
          <MFASelectionPage
            contextType="oidc"
            availableMethods={oidcWebauthn.availableMFAMethods}
            onMethodSelect={oidcWebauthn.selectMFAMethod}
            onGetMethods={oidcWebauthn.getMFAMethods}
          />
        )}

        {oidcWebauthn.currentStep === "webauthn_setup" && (
          <WebAuthnSetupComponent
            contextType="oidc"
            email={oidcWebauthn.email || ""}
            tenantId={oidcWebauthn.tenantId || ""}
            onSuccess={() => {}} // Handled by context
            onError={onAuthError}
            onBack={oidcWebauthn.backToSelection}
            onSetup={oidcWebauthn.setupWebAuthn}
          />
        )}

        {oidcWebauthn.currentStep === "totp_setup" && (
          <TOTPSetupComponent
            contextType="oidc"
            email={oidcWebauthn.email || ""}
            tenantId={oidcWebauthn.tenantId || ""}
            totpData={oidcWebauthn.totpSetupData}
            onSuccess={() => {}} // Handled by context
            onError={onAuthError}
            onBack={oidcWebauthn.backToSelection}
            onSetup={oidcWebauthn.setupTOTP}
            onConfirm={oidcWebauthn.confirmTOTPSetup}
          />
        )}

        {oidcWebauthn.currentStep === "authentication" && (
          <>
            {/* Prefer explicitly selected method if present */}
            {oidcWebauthn.selectedMFAMethod === "webauthn" ? (
              <WebAuthnAuthComponent
                contextType="oidc"
                email={oidcWebauthn.email || ""}
                tenantId={oidcWebauthn.tenantId || ""}
                onSuccess={() => {}} // Handled by context
                onError={(error) => onAuthError?.(error)}
                onAuthenticate={oidcWebauthn.authenticateWithWebAuthn}
              />
            ) : oidcWebauthn.selectedMFAMethod === "totp" ? (
              <TOTPAuthComponent
                contextType="oidc"
                email={oidcWebauthn.email || ""}
                tenantId={oidcWebauthn.tenantId || ""}
                onSuccess={() => {}} // Handled by context
                onError={(error) => onAuthError?.(error)}
                onAuthenticate={oidcWebauthn.authenticateWithTOTP}
              />
            ) : (
              // Fallback to checking enabled methods when nothing is selected
              <>
                {oidcWebauthn.availableMFAMethods.length > 0 ? (
                  oidcWebauthn.availableMFAMethods.some(
                    (method) => method.type === "webauthn" && method.enabled,
                  ) ? (
                    <WebAuthnAuthComponent
                      contextType="oidc"
                      email={oidcWebauthn.email || ""}
                      tenantId={oidcWebauthn.tenantId || ""}
                      onSuccess={() => {}} // Handled by context
                      onError={(error) => onAuthError?.(error)}
                      onAuthenticate={oidcWebauthn.authenticateWithWebAuthn}
                    />
                  ) : (
                    <TOTPAuthComponent
                      contextType="oidc"
                      email={oidcWebauthn.email || ""}
                      tenantId={oidcWebauthn.tenantId || ""}
                      onSuccess={() => {}} // Handled by context
                      onError={(error) => onAuthError?.(error)}
                      onAuthenticate={oidcWebauthn.authenticateWithTOTP}
                    />
                  )
                ) : (
                  <TOTPAuthComponent
                    contextType="oidc"
                    email={oidcWebauthn.email || ""}
                    tenantId={oidcWebauthn.tenantId || ""}
                    onSuccess={() => {}} // Handled by context
                    onError={(error) => onAuthError?.(error)}
                    onAuthenticate={oidcWebauthn.authenticateWithTOTP}
                  />
                )}
              </>
            )}
          </>
        )}

        {oidcWebauthn.currentStep === "token_display" && (
          <OIDCTokenDisplayComponent
            token={oidcWebauthn.displayToken || ""}
            email={oidcWebauthn.email || ""}
          />
        )}

        {oidcWebauthn.isLoading && (
          <div className="auth-loading-overlay">
            <div className="auth-loading-box">
              <div className="flex items-center space-x-3">
                <div className="h-6 w-6 animate-spin rounded-full border-2 border-slate-300 border-t-slate-700" />
                <span className="text-slate-900">Processing...</span>
              </div>
            </div>
          </div>
        )}
      </AuthActionPanel>
    </AuthSplitFrame>
  );
}
