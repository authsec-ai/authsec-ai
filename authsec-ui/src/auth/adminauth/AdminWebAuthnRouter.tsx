/**
 * Admin WebAuthn Router - For admin authentication flow
 * 
 * Orchestrates the admin WebAuthn authentication flow with token storage
 */

import React, { useEffect } from "react";
import { useDispatch } from "react-redux";
import { setCurrentStep, setLoginData } from "../slices/adminWebAuthnSlice";
import { AdminAuthProvider, useAdminAuth } from "../context/AdminAuthContext";
import { MFASelectionPage } from "../webauthn/MFASelectionPage";
import { WebAuthnSetupComponent } from "../webauthn/WebAuthnSetupComponent";
import { TOTPSetupComponent } from "../webauthn/TOTPSetupComponent";
import { WebAuthnAuthComponent } from "../webauthn/WebAuthnAuthComponent";
import { TOTPAuthComponent } from "../webauthn/TOTPAuthComponent";
import { AuthSuccessComponent } from "../webauthn/AuthSuccessComponent";
import { useLocation } from "react-router-dom";
import { decodeHandoff } from "@/utils/handoff";
import { AuthActionPanel } from "../components/AuthActionPanel";

interface AdminWebAuthnRouterProps {
  onAuthComplete: () => void;
  onAuthError?: (error: string) => void;
}

/**
 * Inner router component that uses Admin context
 */
function AdminWebAuthnRouterInner({ onAuthComplete, onAuthError }: AdminWebAuthnRouterProps) {
  const adminWebauthn = useAdminAuth();
  const dispatch = useDispatch();
  const location = useLocation();

  // Apply cross-tenant handoff context if present (when redirected from OIDC callback)
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const handoff = params.get("handoff");
    if (!handoff) return;

    const payload = decodeHandoff<{
      email?: string;
      tenant_id?: string;
      first_login?: boolean;
      target?: "login" | "webauthn";
    }>(handoff);

    if (payload?.email && payload?.tenant_id) {
      dispatch(
        setCurrentStep("login") // reset to login so auto-routing can advance
      );
      dispatch(
        setLoginData({
          tenantId: payload.tenant_id,
          email: payload.email,
          isFirstLogin: !!payload.first_login,
        })
      );

      // Clean up the URL to avoid re-processing
      params.delete("handoff");
      const next = `${location.pathname}${params.toString() ? `?${params.toString()}` : ""}`;
      window.history.replaceState({}, "", next);
    }
  }, [location.pathname, location.search, dispatch]);

  // Handle authentication completion
  useEffect(() => {
    console.log("🔄 Admin WebAuthn step changed to:", adminWebauthn.currentStep);
    
    if (adminWebauthn.currentStep === "completed" && adminWebauthn.authToken) {
      console.log("🎯 Admin flow completed with stored token");
      onAuthComplete();
    }
  }, [adminWebauthn.currentStep, adminWebauthn.authToken, onAuthComplete]);

  // Handle authentication errors
  useEffect(() => {
    if (adminWebauthn.authenticationError && onAuthError) {
      onAuthError(adminWebauthn.authenticationError);
    }
  }, [adminWebauthn.authenticationError, onAuthError]);

  // Automatic routing based on login status - matches OIDC router behavior
  useEffect(() => {
    if (
      adminWebauthn.currentStep === 'login' &&
      adminWebauthn.email &&
      adminWebauthn.tenantId
    ) {
      const next = adminWebauthn.isFirstLogin ? 'mfa_selection' : 'authentication';
      console.log("🔄 Admin auto-routing:", adminWebauthn.isFirstLogin ? "first login -> mfa_selection" : "returning user -> authentication");
      dispatch(setCurrentStep(next));
    }
  }, [adminWebauthn.currentStep, adminWebauthn.email, adminWebauthn.tenantId, adminWebauthn.isFirstLogin, dispatch]);

  // Prefetch MFA methods when entering authentication or selection
  useEffect(() => {
    if (!adminWebauthn.email || !adminWebauthn.tenantId) return;
    if (
      (adminWebauthn.currentStep === 'authentication' || adminWebauthn.currentStep === 'mfa_selection') &&
      (!adminWebauthn.availableMFAMethods || adminWebauthn.availableMFAMethods.length === 0)
    ) {
      // Fire and forget; context handles errors/toasts
      void adminWebauthn.getMFAMethods();
    }
  }, [adminWebauthn.currentStep, adminWebauthn.email, adminWebauthn.tenantId, adminWebauthn.getMFAMethods, adminWebauthn.availableMFAMethods]);

  // Render lightweight loader if step is still 'login' to avoid blank screen during transition
  if (adminWebauthn.currentStep === "login") {
    return (
      <div className="w-full flex items-center justify-center p-4">
        <div className="flex items-center space-x-3 text-foreground">
          <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-foreground" />
          <span>Preparing authentication...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="w-full">
      <div className="w-full max-w-[560px] mx-auto space-y-6">
        <AuthActionPanel>
          {adminWebauthn.currentStep === "mfa_selection" && (
            <MFASelectionPage 
              contextType="admin"
              availableMethods={adminWebauthn.availableMFAMethods}
              onMethodSelect={adminWebauthn.selectMFAMethod}
              onGetMethods={adminWebauthn.getMFAMethods}
            />
          )}
          
          {adminWebauthn.currentStep === "webauthn_setup" && (
            <WebAuthnSetupComponent 
              contextType="admin"
              email={adminWebauthn.email || ""}
              tenantId={adminWebauthn.tenantId || ""}
              onSuccess={() => {}} // Handled by context
              onError={(error) => onAuthError?.(error)}
              onBack={adminWebauthn.backToSelection}
              onSetup={adminWebauthn.setupWebAuthn}
            />
          )}
          
          {adminWebauthn.currentStep === "totp_setup" && (
            <TOTPSetupComponent 
              contextType="admin"
              email={adminWebauthn.email || ""}
              tenantId={adminWebauthn.tenantId || ""}
              totpData={adminWebauthn.totpSetupData}
              onSuccess={() => {}} // Handled by context
              onError={(error) => onAuthError?.(error)}
              onBack={adminWebauthn.backToSelection}
              onSetup={adminWebauthn.setupTOTP}
              onConfirm={adminWebauthn.confirmTOTPSetup}
            />
          )}
          
          {adminWebauthn.currentStep === "authentication" && (
            <>
              {/* Prefer explicitly selected method if present */}
              {adminWebauthn.selectedMFAMethod === 'webauthn' ? (
                <WebAuthnAuthComponent 
                  contextType="admin"
                  email={adminWebauthn.email || ""}
                  tenantId={adminWebauthn.tenantId || ""}
                  onSuccess={() => {}} // Handled by context
                  onError={(error) => onAuthError?.(error)}
                  onAuthenticate={adminWebauthn.authenticateWithWebAuthn}
                />
              ) : adminWebauthn.selectedMFAMethod === 'totp' ? (
                <TOTPAuthComponent 
                  contextType="admin"
                  email={adminWebauthn.email || ""}
                  tenantId={adminWebauthn.tenantId || ""}
                  onSuccess={() => {}} // Handled by context
                  onError={(error) => onAuthError?.(error)}
                  onAuthenticate={adminWebauthn.authenticateWithTOTP}
                />
              ) : (
                // When nothing is selected yet, wait for methods to load before choosing
                <>
                  {adminWebauthn.availableMFAMethods.length > 0 ? (
                    adminWebauthn.availableMFAMethods.some(method => method.type === "webauthn" && method.enabled) ? (
                      <WebAuthnAuthComponent 
                        contextType="admin"
                        email={adminWebauthn.email || ""}
                        tenantId={adminWebauthn.tenantId || ""}
                        onSuccess={() => {}} // Handled by context
                        onError={(error) => onAuthError?.(error)}
                        onAuthenticate={adminWebauthn.authenticateWithWebAuthn}
                      />
                    ) : (
                      <TOTPAuthComponent 
                        contextType="admin"
                        email={adminWebauthn.email || ""}
                        tenantId={adminWebauthn.tenantId || ""}
                        onSuccess={() => {}} // Handled by context
                        onError={(error) => onAuthError?.(error)}
                        onAuthenticate={adminWebauthn.authenticateWithTOTP}
                      />
                    )
                  ) : (
                    // Minimal loader to avoid TOTP -> biometric flicker
                    <div className="flex items-center justify-center py-10 text-slate-700">
                      <div className="animate-spin rounded-full h-5 w-5 border-b-2 border-slate-700 mr-2" />
                      Checking authentication methods...
                    </div>
                  )}
                </>
              )}
            </>
          )}
          
          {adminWebauthn.currentStep === "completed" && (
            <AuthSuccessComponent />
          )}
        </AuthActionPanel>
      </div>
      
      {/* Loading State */}
      {adminWebauthn.isLoading && (
        <div className="auth-loading-overlay">
          <div className="auth-loading-box">
            <div className="flex items-center space-x-3">
              <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-slate-800" />
              <span className="text-slate-800">Processing...</span>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

/**
 * Admin WebAuthn Router with provider wrapper
 */
export function AdminWebAuthnRouter(props: AdminWebAuthnRouterProps) {
  return (
    <AdminAuthProvider>
      <AdminWebAuthnRouterInner {...props} />
    </AdminAuthProvider>
  );
}
