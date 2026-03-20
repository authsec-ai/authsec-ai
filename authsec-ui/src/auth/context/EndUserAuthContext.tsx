/**
 * EndUser Auth Context - For end-user authentication flows (OIDC/OAuth)
 * Updated to support universal callback URL approach
 * 
 * Handles OIDC/OAuth flows and WebAuthn for end users
 */

import React, { createContext, useContext, useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";
import {
  useGetMFAStatusForLoginMutation,
  useBeginWebAuthnRegistrationMutation,
  useFinishWebAuthnRegistrationMutation,
  useBeginWebAuthnAuthMutation,
  useFinishWebAuthnAuthMutation,
  useBeginTOTPSetupMutation,
  useConfirmTOTPSetupMutation,
  useVerifyTOTPMutation,
  useWebauthnEnduserCallbackMutation,
  type WebAuthnCredential,
  type MFAMethod,
  type MFAStatusMethod
} from "../../app/api/webauthnApi";
import { useNotifyNewUserRegistrationMutation } from "../../app/api/authApi";
import {
  setCurrentStep,
  setLoginData,
  setClientId,
  setAvailableMFAMethods,
  setSelectedMFAMethod,
  setTOTPSecret,
  setTOTPSetupData,
  setIsLoading,
  setAuthenticationError,
  setDisplayToken,
  setMFARequired,
  resetOIDCWebAuthnState,
  type MFAMethod as OIDCMFAMethod
} from "../slices/oidcWebAuthnSlice";
import type { RootState } from "../../app/store";
import { toast } from "react-hot-toast";

interface EndUserAuthContextType {
  // State
  currentStep: string;
  isFirstLogin: boolean;
  tenantId: string | null;
  email: string | null;
  clientId: string | null;
  availableMFAMethods: OIDCMFAMethod[];
  selectedMFAMethod: "webauthn" | "totp" | null;
  totpSetupData: any;
  isLoading: boolean;
  authenticationError: string | null;
  displayToken: string | null;
  tokenDisplayed: boolean;
  
  // Actions
  getMFAMethods: () => Promise<boolean>;
  selectMFAMethod: (method: "webauthn" | "totp") => void;
  setupWebAuthn: () => Promise<boolean>;
  setupTOTP: () => Promise<boolean>;
  confirmTOTPSetup: (code: string) => Promise<boolean>;
  authenticateWithWebAuthn: () => Promise<boolean>;
  authenticateWithTOTP: (code: string) => Promise<boolean>;
  resetFlow: () => void;
  executeCallback: (email: string, tenantId?: string) => Promise<{ success: boolean; token?: string; error?: string }>;
  backToSelection: () => void;
  // Helper to capture client_id from initial responses
  captureClientId: (clientId: string) => void;
  // New helper for universal callback URL
  getUniversalCallbackURL: () => string;
}

const EndUserAuthContext = createContext<EndUserAuthContextType | null>(null);

export const useEndUserAuth = () => {
  const context = useContext(EndUserAuthContext);
  if (!context) {
    throw new Error("useEndUserAuth must be used within an EndUserAuthProvider");
  }
  return context;
};

// Backward compatibility exports
export const useEndUserWebAuthn = useEndUserAuth;
export const useOIDCWebAuthn = useEndUserAuth;

// Helper functions for WebAuthn
const arrayBufferToBase64Url = (buffer: ArrayBuffer): string => {
  try {
    return btoa(String.fromCharCode(...new Uint8Array(buffer)))
      .replace(/\+/g, '-')
      .replace(/\//g, '_')
      .replace(/=/g, '');
  } catch (error) {
    console.error(`Error converting ArrayBuffer to base64url:`, error);
    throw new Error(`Failed to encode ArrayBuffer: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
};

const base64UrlToArrayBuffer = (base64url: string): ArrayBuffer => {
  try {
    base64url = base64url.trim();
    if (!base64url) {
      throw new Error('Empty base64url string provided');
    }
    
    const padding = '='.repeat((4 - base64url.length % 4) % 4);
    const base64 = (base64url + padding).replace(/-/g, '+').replace(/_/g, '/');
    
    const binary = atob(base64);
    const buffer = new ArrayBuffer(binary.length);
    const bytes = new Uint8Array(buffer);
    for (let i = 0; i < binary.length; i++) {
      bytes[i] = binary.charCodeAt(i);
    }
    
    return buffer;
  } catch (error) {
    console.error(`Error converting base64url to ArrayBuffer: ${error instanceof Error ? error.message : 'Unknown error'}`);
    throw new Error(`Failed to decode base64url: ${error instanceof Error ? error.message : 'Unknown error'}`);
  }
};

// OIDC callback handler using RTK Query (displays token, doesn't store)
class OIDCCallbackHandler {
  private pendingCallbacks = new Map<string, Promise<any>>();
  private completedResults = new Map<string, { ts: number; result: { success: boolean; token?: string; error?: string } }>();
  
  constructor(private webauthnCallbackMutation: any) {}
  
  async executeCallback(email: string, tenantId?: string, clientId?: string) {
    const sessionKey = `oidc-${email}-${tenantId ?? 'none'}-${clientId ?? 'none'}`;

    if (this.pendingCallbacks.has(sessionKey)) {
      return this.pendingCallbacks.get(sessionKey);
    }

    const completed = this.completedResults.get(sessionKey);
    if (completed && Date.now() - completed.ts < 15000) {
      return completed.result;
    }

    const callbackPromise = this.performCallback(email, tenantId, clientId);
    this.pendingCallbacks.set(sessionKey, callbackPromise);
    
    try {
      const result = await callbackPromise;
      if (result?.success && result?.token) {
        this.completedResults.set(sessionKey, { ts: Date.now(), result });
      }
      return result;
    } finally {
      this.pendingCallbacks.delete(sessionKey);
    }
  }
  
  private async performCallback(email: string, tenantId?: string, clientId?: string) {
    try {
      const requestBody = {
        email,
        mfa_verified: true,
        tenant_id: tenantId,
        flow_context: 'enduser' as const,
        ...(clientId && { client_id: clientId })
      };
      
      const result = await this.webauthnCallbackMutation(requestBody).unwrap();
      const token = (result && (result.token || (result as any).access_token || (result as any).jwt || (result as any).jwt_token || (result as any).id_token)) as string | undefined;

      if (token) {
        return { success: true, token };
      } else {
        const successFlag = (result as any)?.success;
        return { success: !!successFlag, error: (result as any)?.error || 'No token received' };
      }
      
    } catch (error) {
      return { 
        success: false, 
        error: error instanceof Error ? error.message : 'Unknown error' 
      };
    }
  }
}

export const EndUserAuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const dispatch = useDispatch();
  const oidcWebauthn = useSelector((state: RootState) => state.oidcWebAuthn);
  
  // API hooks
  const [mfaStatusCheck] = useGetMFAStatusForLoginMutation();
  const [beginWebAuthnRegistration] = useBeginWebAuthnRegistrationMutation();
  const [finishWebAuthnRegistration] = useFinishWebAuthnRegistrationMutation();
  const [beginWebAuthnAuth] = useBeginWebAuthnAuthMutation();
  const [finishWebAuthnAuth] = useFinishWebAuthnAuthMutation();
  const [totpBeginSetup] = useBeginTOTPSetupMutation();
  const [totpConfirmSetup] = useConfirmTOTPSetupMutation();
  const [totpVerify] = useVerifyTOTPMutation();
  const [webauthnCallbackMutation] = useWebauthnEnduserCallbackMutation();
  const [notifyNewUser] = useNotifyNewUserRegistrationMutation();

  const callbackHandler = React.useMemo(() => new OIDCCallbackHandler(webauthnCallbackMutation), [webauthnCallbackMutation]);

  // Ref to guard setMFARequired — only set on first getMFAMethods call, not refreshes.
  const mfaRequiredSetRef = React.useRef(false);

  // Universal callback URL helper
  const getUniversalCallbackURL = useCallback((): string => {
    const baseUrl = window.location.origin;
    return `${baseUrl}/oidc/auth/callback`;
  }, []);

  const getMFAMethods = useCallback(async (): Promise<boolean> => {
    if (!oidcWebauthn.tenantId || !oidcWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    // Log client_id availability for debugging
    if (!oidcWebauthn.clientId) {
      console.warn("⚠️ Client ID not available for MFA status check");
    }

    try {
      // Include client_id in MFA status check if available
      const requestPayload = {
        email: oidcWebauthn.email!,
        tenant_id: oidcWebauthn.tenantId!,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId })
      };

      const res = await mfaStatusCheck(requestPayload);
      
      if (!('data' in res) || !res.data) {
        const status = 'error' in res ? (res as any).error?.status : undefined;
        if (status === 404) {
          const methods404: OIDCMFAMethod[] = [
            {
              type: "webauthn",
              display_name: "WebAuthn (Biometric)",
              description: "Use fingerprint/face/security key",
              enabled: false,
              recommended: true
            },
            {
              type: "totp",
              display_name: "TOTP (Authenticator App)",
              description: "Google Authenticator / Authy",
              enabled: false,
              recommended: false
            }
          ];
          dispatch(setAvailableMFAMethods(methods404));
          if (!mfaRequiredSetRef.current) {
            mfaRequiredSetRef.current = true;
            dispatch(setMFARequired(false)); // No methods configured = new user
          }
          return true;
        }
        toast.error("Failed to get MFA methods");
        return false;
      }

      const configured = Array.isArray(res.data.configured_methods) ? res.data.configured_methods : [];
      const apiMethods: MFAStatusMethod[] = Array.isArray(res.data.methods) ? res.data.methods : [];

      const findMethodByType = (type: "webauthn" | "totp") =>
        apiMethods.find((method) => (method?.method_type ?? "").toLowerCase() === type);

      const isConfigured = (type: "webauthn" | "totp") =>
        configured.some((m: any) => (m?.type ?? "").toLowerCase() === type && (m?.enabled ?? true));

      const webauthnMethod = findMethodByType("webauthn");
      const totpMethod = findMethodByType("totp");

      const methods: OIDCMFAMethod[] = [
        {
          type: "webauthn",
          display_name: (webauthnMethod?.display_name || "").trim() || "WebAuthn (Biometric)",
          description: (webauthnMethod?.description || "").trim() || "Use fingerprint/face/security key",
          enabled: typeof webauthnMethod?.enabled === "boolean" ? webauthnMethod.enabled : isConfigured("webauthn"),
          recommended: typeof webauthnMethod?.recommended === "boolean" ? webauthnMethod.recommended : true
        },
        {
          type: "totp",
          display_name: (totpMethod?.display_name || "").trim() || "TOTP (Authenticator App)",
          description: (totpMethod?.description || "").trim() || "Google Authenticator / Authy",
          enabled: typeof totpMethod?.enabled === "boolean" ? totpMethod.enabled : isConfigured("totp"),
          recommended: typeof totpMethod?.recommended === "boolean" ? totpMethod.recommended : false
        }
      ];

      dispatch(setAvailableMFAMethods(methods));

      // Store MFA required status on first call only (not on refreshes after setup)
      if (!mfaRequiredSetRef.current) {
        mfaRequiredSetRef.current = true;
        console.log("[Auth] MFA status response mfa_required:", res.data.mfa_required);
        dispatch(setMFARequired(res.data.mfa_required === true ? true : false));
      }

      return true;
    } catch (e) {
      toast.error("Failed to get MFA methods");
      return false;
    }
  }, [oidcWebauthn.tenantId, oidcWebauthn.email, oidcWebauthn.clientId, mfaStatusCheck, dispatch]);

  const selectMFAMethod = useCallback((method: "webauthn" | "totp") => {
    console.log("🔄 OIDC method selected:", method);
    dispatch(setSelectedMFAMethod(method));
    // For first login, always take the setup route for the selected method
    if (oidcWebauthn.isFirstLogin) {
      const nextStep = method === 'totp' ? 'totp_setup' : 'webauthn_setup';
      console.log("➡️ OIDC advancing to step:", nextStep, "(first_login)");
      dispatch(setCurrentStep(nextStep));
      return;
    }

    const isEnabled = (oidcWebauthn.availableMFAMethods || []).some(m => m.type === method && m.enabled);
    const nextStep = method === "totp"
      ? (isEnabled ? "authentication" : "totp_setup")
      : (isEnabled ? "authentication" : "webauthn_setup");
    console.log("➡️ OIDC advancing to step:", nextStep, "(enabled:", isEnabled, ")");
    dispatch(setCurrentStep(nextStep));
  }, [dispatch, oidcWebauthn.availableMFAMethods, oidcWebauthn.isFirstLogin]);

  const setupWebAuthn = useCallback(async (): Promise<boolean> => {
    if (!oidcWebauthn.tenantId || !oidcWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    if (typeof window === 'undefined' || !('credentials' in navigator)) {
      toast.error("WebAuthn unsupported in this browser");
      return false;
    }
    if (window.isSecureContext !== true) {
      toast.error("Requires HTTPS origin");
      return false;
    }

    // Log client_id availability for debugging
    if (!oidcWebauthn.clientId) {
      console.warn("⚠️ Client ID not available for WebAuthn setup");
    }

    try {
      // End-User Setup: begin registration
      const requestPayload = {
        email: oidcWebauthn.email!,
        tenant_id: oidcWebauthn.tenantId!,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId })
      };

      const challengeData = await beginWebAuthnRegistration(requestPayload).unwrap();
      let publicKeyData = challengeData.publicKey || challengeData;
      if (publicKeyData.publicKey) publicKeyData = publicKeyData.publicKey;

      if (!publicKeyData.challenge || !publicKeyData.user?.id || !publicKeyData.rp) {
        throw new Error('Invalid registration options received');
      }

      const credential = await navigator.credentials.create({
        publicKey: {
          challenge: base64UrlToArrayBuffer(publicKeyData.challenge),
          rp: {
            ...publicKeyData.rp
          },
          user: {
            id: base64UrlToArrayBuffer(publicKeyData.user.id),
            name: publicKeyData.user.name || oidcWebauthn.email!,
            displayName: publicKeyData.user.displayName || oidcWebauthn.email!,
          },
          pubKeyCredParams: publicKeyData.pubKeyCredParams,
          authenticatorSelection: publicKeyData.authenticatorSelection,
          timeout: publicKeyData.timeout,
          attestation: publicKeyData.attestation,
        }
      });

      if (!credential) {
        throw new Error('Failed to create credential');
      }

      const publicKeyCredential = credential as PublicKeyCredential;
      const response = publicKeyCredential.response as AuthenticatorAttestationResponse;

      // Finish registration
      const finishPayload = {
        email: oidcWebauthn.email!,
        tenant_id: oidcWebauthn.tenantId!,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId }),
        credential: {
          id: credential.id,
          rawId: arrayBufferToBase64Url(publicKeyCredential.rawId),
          type: "public-key",
          response: {
            attestationObject: arrayBufferToBase64Url(response.attestationObject),
            clientDataJSON: arrayBufferToBase64Url(response.clientDataJSON),
          }
        }
      };

      await finishWebAuthnRegistration(finishPayload).unwrap();

      // Refresh MFA methods so WebAuthn shows enabled immediately
      await getMFAMethods();

      toast.success("Biometric setup complete. Please verify to finish sign-in.");
      dispatch(setSelectedMFAMethod("webauthn"));
      dispatch(setCurrentStep("authentication"));
      return true;
    } catch (error) {
      console.error("❌ OIDC WebAuthn setup error:", error);
      const errorMsg = error instanceof Error ? error.message : 'Setup failed';
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [oidcWebauthn.tenantId, oidcWebauthn.email, oidcWebauthn.clientId, beginWebAuthnRegistration, finishWebAuthnRegistration, dispatch, getMFAMethods]);

  const setupTOTP = useCallback(async (): Promise<boolean> => {
    if (!oidcWebauthn.tenantId || !oidcWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    // Log client_id availability for debugging
    if (!oidcWebauthn.clientId) {
      console.warn("⚠️ Client ID not available for TOTP setup");
    }

    try {
      const requestPayload = {
        email: oidcWebauthn.email,
        tenant_id: oidcWebauthn.tenantId,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId })
      };

      const result = await totpBeginSetup(requestPayload);

      if ('data' in result && result.data) {
        dispatch(setTOTPSetupData(result.data));
        dispatch(setTOTPSecret(result.data.secret));
        dispatch(setCurrentStep("totp_setup"));
        return true;
      } else {
        throw new Error('Failed to setup TOTP');
      }
    } catch (error) {
      console.error("❌ OIDC TOTP setup error:", error);
      const errorMsg = error instanceof Error ? error.message : 'TOTP setup failed';
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [oidcWebauthn.tenantId, oidcWebauthn.email, oidcWebauthn.clientId, totpBeginSetup, dispatch]);

  const confirmTOTPSetup = useCallback(async (code: string): Promise<boolean> => {
    if (!oidcWebauthn.tenantId || !oidcWebauthn.email || !oidcWebauthn.totpSecret) {
      toast.error("Missing TOTP setup data");
      return false;
    }

    // Log client_id availability for debugging
    if (!oidcWebauthn.clientId) {
      console.warn("⚠️ Client ID not available for TOTP setup confirmation");
    }

    try {
      const requestPayload = {
        email: oidcWebauthn.email,
        tenant_id: oidcWebauthn.tenantId,
        secret: oidcWebauthn.totpSecret,
        code,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId })
      };

      const result = await totpConfirmSetup(requestPayload);

      if ('data' in result) {
        // Refresh MFA methods to ensure TOTP now shows as enabled
        await getMFAMethods();

        toast.success("TOTP setup confirmed. Please verify with a code to finish sign-in.");
        dispatch(setSelectedMFAMethod("totp"));
        dispatch(setCurrentStep("authentication"));
        return true;
      } else {
        throw new Error('TOTP confirmation failed');
      }
    } catch (error) {
      console.error("❌ OIDC TOTP confirm error:", error);
      const errorMsg = error instanceof Error ? error.message : 'Invalid TOTP code';
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [oidcWebauthn.tenantId, oidcWebauthn.email, oidcWebauthn.clientId, oidcWebauthn.totpSecret, totpConfirmSetup, dispatch, getMFAMethods]);

  const authenticateWithWebAuthn = useCallback(async (): Promise<boolean> => {
    if (!oidcWebauthn.tenantId || !oidcWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    if (typeof window === 'undefined' || !('credentials' in navigator)) {
      toast.error("WebAuthn unsupported in this browser");
      return false;
    }
    if (window.isSecureContext !== true) {
      toast.error("Requires HTTPS origin");
      return false;
    }

    // Log client_id availability for debugging
    if (!oidcWebauthn.clientId) {
      console.warn("⚠️ Client ID not available for WebAuthn authentication");
    }

    try {
      // Begin authentication
      const requestPayload = {
        email: oidcWebauthn.email!,
        tenant_id: oidcWebauthn.tenantId!,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId })
      };

      const challengeData = await beginWebAuthnAuth(requestPayload).unwrap();

      let publicKeyData = challengeData.publicKey || challengeData;
      if (publicKeyData.publicKey) publicKeyData = publicKeyData.publicKey;

      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: base64UrlToArrayBuffer(publicKeyData.challenge),
          rpId: publicKeyData.rpId || "app.authsec.dev",
          allowCredentials: publicKeyData.allowCredentials?.map((cred: any) => ({
            ...cred,
            id: base64UrlToArrayBuffer(cred.id)
          })),
          timeout: publicKeyData.timeout,
          userVerification: publicKeyData.userVerification,
        }
      });

      if (!credential) {
        throw new Error('Authentication cancelled or failed');
      }

      const publicKeyCredential = credential as PublicKeyCredential;
      const response = publicKeyCredential.response as AuthenticatorAssertionResponse;

      // Finish authentication
      const finishPayload = {
        email: oidcWebauthn.email!,
        tenant_id: oidcWebauthn.tenantId!,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId }),
        credential: {
          id: credential.id,
          rawId: arrayBufferToBase64Url(publicKeyCredential.rawId),
          type: "public-key",
          response: {
            clientDataJSON: arrayBufferToBase64Url(response.clientDataJSON),
            authenticatorData: arrayBufferToBase64Url(response.authenticatorData),
            signature: arrayBufferToBase64Url(response.signature),
            userHandle: response.userHandle ? arrayBufferToBase64Url(response.userHandle) : null
          }
        }
      };

      await finishWebAuthnAuth(finishPayload).unwrap();

      toast.success("Authentication successful!");

      const callbackResult = await callbackHandler.executeCallback(oidcWebauthn.email!, oidcWebauthn.tenantId!, oidcWebauthn.clientId || undefined);
      if (callbackResult.success && callbackResult.token) {
        // Notify new user before displaying token (await to ensure it completes)
        if (oidcWebauthn.mfaRequired === false) {
          try {
            await notifyNewUser({}).unwrap();
            console.log("[Auth] New user registration notification sent");
          } catch (err) {
            console.error("[Auth] Failed to send new user notification:", err);
          }
        }

        dispatch(setDisplayToken(callbackResult.token));
        return true;
      } else {
        throw new Error(callbackResult.error || 'Callback failed');
      }
    } catch (error: any) {
      console.error("❌ OIDC WebAuthn auth error:", error);
      const apiMsg = error?.data?.error || error?.data?.message;
      const errorMsg = apiMsg || (error instanceof Error ? error.message : 'Authentication failed');
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [oidcWebauthn.tenantId, oidcWebauthn.email, oidcWebauthn.clientId, oidcWebauthn.mfaRequired, beginWebAuthnAuth, finishWebAuthnAuth, dispatch, callbackHandler, notifyNewUser]);

  const authenticateWithTOTP = useCallback(async (code: string): Promise<boolean> => {
    if (!oidcWebauthn.tenantId || !oidcWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    // Log client_id availability for debugging
    if (!oidcWebauthn.clientId) {
      console.warn("⚠️ Client ID not available for TOTP authentication");
    }

    try {
      const requestPayload = {
        email: oidcWebauthn.email,
        tenant_id: oidcWebauthn.tenantId,
        code,
        ...(oidcWebauthn.clientId && { client_id: oidcWebauthn.clientId })
      };

      const result = await totpVerify(requestPayload);

      if ('data' in result) {
        toast.success("Authentication successful!");
        
        const callbackResult = await callbackHandler.executeCallback(oidcWebauthn.email, oidcWebauthn.tenantId, oidcWebauthn.clientId || undefined);

        if (callbackResult.success && callbackResult.token) {
          // Notify new user before displaying token (await to ensure it completes)
          if (oidcWebauthn.mfaRequired === false) {
            try {
              await notifyNewUser({}).unwrap();
              console.log("[Auth] New user registration notification sent");
            } catch (err) {
              console.error("[Auth] Failed to send new user notification:", err);
            }
          }

          dispatch(setDisplayToken(callbackResult.token));
          return true;
        } else {
          throw new Error(callbackResult.error || 'Callback failed');
        }
      } else {
        const errorMsg = "Invalid TOTP code";
        dispatch(setAuthenticationError(errorMsg));
        toast.error(errorMsg);
        return false;
      }
    } catch (error) {
      console.error("❌ OIDC TOTP auth error:", error);
      const errorMsg = "Failed to authenticate with TOTP";
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [oidcWebauthn.tenantId, oidcWebauthn.email, oidcWebauthn.clientId, oidcWebauthn.mfaRequired, totpVerify, dispatch, callbackHandler, notifyNewUser]);

  const resetFlow = useCallback(() => {
    dispatch(resetOIDCWebAuthnState());
  }, [dispatch]);

  const executeCallback = useCallback(async (email: string, tenantId?: string) => {
    return callbackHandler.executeCallback(email, tenantId, oidcWebauthn.clientId || undefined);
  }, [callbackHandler, oidcWebauthn.clientId]);

  const backToSelection = useCallback(() => {
    dispatch(setCurrentStep("mfa_selection"));
  }, [dispatch]);

  const captureClientId = useCallback((clientId: string) => {
    dispatch(setClientId(clientId));
  }, [dispatch]);

  const value: EndUserAuthContextType = {
    // State
    currentStep: oidcWebauthn.currentStep,
    isFirstLogin: oidcWebauthn.isFirstLogin,
    tenantId: oidcWebauthn.tenantId,
    email: oidcWebauthn.email,
    clientId: oidcWebauthn.clientId,
    availableMFAMethods: oidcWebauthn.availableMFAMethods,
    selectedMFAMethod: oidcWebauthn.selectedMFAMethod,
    totpSetupData: oidcWebauthn.totpSetupData,
    isLoading: oidcWebauthn.isLoading,
    authenticationError: oidcWebauthn.authenticationError,
    displayToken: oidcWebauthn.displayToken,
    tokenDisplayed: oidcWebauthn.tokenDisplayed,
    
    // Actions
    getMFAMethods,
    selectMFAMethod,
    setupWebAuthn,
    setupTOTP,
    confirmTOTPSetup,
    authenticateWithWebAuthn,
    authenticateWithTOTP,
    resetFlow,
    executeCallback,
    backToSelection,
    captureClientId,
    getUniversalCallbackURL,
  };

  return <EndUserAuthContext.Provider value={value}>{children}</EndUserAuthContext.Provider>;
};

// Backward compatibility exports
export const EndUserWebAuthnProvider = EndUserAuthProvider;
export const OIDCWebAuthnProvider = EndUserAuthProvider;
