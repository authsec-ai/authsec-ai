/**
 * Admin Auth Context - For /uflow/login admin authentication flow
 * 
 * Uses "Login" endpoints and stores tokens in localStorage/sessionStorage
 */

import React, { createContext, useContext, useCallback } from "react";
import { useSelector, useDispatch } from "react-redux";
import {
  useGetAdminMFAStatusMutation,
  useBeginAdminRegistrationMutation,
  useFinishAdminRegistrationMutation,
  useBeginAdminAuthenticationMutation,
  useFinishAdminAuthenticationMutation,
  useBeginTOTPLoginSetupMutation,  // Admin uses "Login" endpoints
  useConfirmTOTPLoginSetupMutation,
  useVerifyTOTPLoginMutation,
  useWebauthnCallbackMutation
} from "../../app/api/webauthnApi";
import { useNotifyNewUserRegistrationMutation } from "../../app/api/authApi";
import {
  setCurrentStep,
  setAvailableMFAMethods,
  setSelectedMFAMethod,
  setTOTPSecret,
  setTOTPSetupData,
  setAuthenticationError,
  setAuthToken,
  setMFARequired,
  resetAdminWebAuthnState,
  type MFAMethod as AdminMFAMethod
} from "../slices/adminWebAuthnSlice";
import type { RootState } from "../../app/store";
import { toast } from "react-hot-toast";
import { completeWebAuthnAuthentication } from "../slices/authSlice";
import { NIL } from "uuid";
import type { MFAStatusMethod } from "../../app/api/webauthnApi";

interface AdminAuthContextType {
  // State
  currentStep: string;
  isFirstLogin: boolean;
  tenantId: string | null;
  email: string | null;
  availableMFAMethods: AdminMFAMethod[];
  selectedMFAMethod: "webauthn" | "totp" | null;
  totpSetupData: any;
  isLoading: boolean;
  authenticationError: string | null;
  authToken: string | null;
  
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
}

const AdminAuthContext = createContext<AdminAuthContextType | null>(null);

export const useAdminAuth = () => {
  const context = useContext(AdminAuthContext);
  if (!context) {
    throw new Error("useAdminAuth must be used within an AdminAuthProvider");
  }
  return context;
};

// Backward compatibility export
export const useAdminWebAuthn = useAdminAuth;

// Helper functions for WebAuthn (same as before but in admin context)
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

const base64UrlToArrayBuffer = (base64url: string | undefined): ArrayBuffer => {
  try {
    // Validate input
    if (base64url === undefined || base64url === null) {
      throw new Error('base64url parameter is undefined or null');
    }
    
    if (typeof base64url !== 'string') {
      throw new Error(`Expected string, got ${typeof base64url}`);
    }
    
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

// Admin callback handler using RTK Query
class AdminCallbackHandler {
  private pendingCallbacks = new Map<string, Promise<any>>();
  private completedResults = new Map<string, { ts: number; result: { success: boolean; token?: string; error?: string } }>();
  
  constructor(private webauthnCallbackMutation: any) {}
  
  async executeCallback(email: string, tenantId?: string) {
    const sessionKey = `admin-${email}-${tenantId ?? 'none'}`;
    
    if (this.pendingCallbacks.has(sessionKey)) {
      return this.pendingCallbacks.get(sessionKey);
    }
    const completed = this.completedResults.get(sessionKey);
    if (completed && Date.now() - completed.ts < 15000) {
      return completed.result;
    }
    
    const callbackPromise = this.performCallback(email, tenantId);
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
  
  private async performCallback(email: string, tenantId?: string) {
    try {
      const requestBody = {
        email,
        mfa_verified: true,
        tenant_id: tenantId,
        flow_context: 'admin' as const
      };
      
      const result = await this.webauthnCallbackMutation(requestBody).unwrap();
      // Accept various token key formats just in case
      const token = (result && (result.token || (result as any).access_token || (result as any).jwt || (result as any).jwt_token || (result as any).id_token)) as string | undefined;

      // Treat presence of a token as success even if "success" flag is absent
      if (token) {
        // Store in localStorage for admin flow
        if (typeof window !== 'undefined') {
          localStorage.setItem('jwt_token', token);
        }
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

export const AdminAuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const dispatch = useDispatch();
  const adminWebauthn = useSelector((state: RootState) => state.adminWebAuthn);
  
  // API hooks - Using new admin endpoints
  const [mfaStatusCheck] = useGetAdminMFAStatusMutation();
  const [beginAdminRegistration] = useBeginAdminRegistrationMutation();
  const [finishAdminRegistration] = useFinishAdminRegistrationMutation();
  const [beginAdminAuthentication] = useBeginAdminAuthenticationMutation();
  const [finishAdminAuthentication] = useFinishAdminAuthenticationMutation();
  const [totpBeginLoginSetup] = useBeginTOTPLoginSetupMutation(); // Admin Login endpoints
  const [totpConfirmLoginSetup] = useConfirmTOTPLoginSetupMutation();
  const [totpVerifyLogin] = useVerifyTOTPLoginMutation();
  const [webauthnCallbackMutation] = useWebauthnCallbackMutation();
  const [notifyNewUser] = useNotifyNewUserRegistrationMutation();

  const callbackHandler = React.useMemo(() => new AdminCallbackHandler(webauthnCallbackMutation), [webauthnCallbackMutation]);

  // Ref to guard setMFARequired — only set on first getMFAMethods call, not refreshes.
  // Using a ref avoids stale closure issues when setupWebAuthn calls getMFAMethods again.
  const mfaRequiredSetRef = React.useRef(false);
  const notifiedUsersRef = React.useRef<Set<string>>(new Set());

  const notifyNewUserIfNeeded = useCallback(async (token?: string) => {
    const emailKey = (adminWebauthn.email ?? "").trim().toLowerCase();
    if (!emailKey || notifiedUsersRef.current.has(emailKey)) {
      return;
    }
    if (adminWebauthn.mfaRequired !== false || !token) {
      return;
    }

    try {
      await notifyNewUser({ token }).unwrap();
      notifiedUsersRef.current.add(emailKey);
      console.log("[Auth] New user registration notification sent");
    } catch (err) {
      console.error("[Auth] Failed to send new user notification:", err);
    }
  }, [adminWebauthn.email, adminWebauthn.mfaRequired, notifyNewUser]);

  const getMFAMethods = useCallback(async (): Promise<boolean> => {
    if (!adminWebauthn.email) {
      toast.error("Missing authentication data");
      dispatch(setAuthenticationError("Missing authentication data"));
      return false;
    }

    // Clear any existing authentication errors before proceeding
    dispatch(setAuthenticationError(null));

    try {
      const res = await mfaStatusCheck({
        email: adminWebauthn.email
      });
      
      if (!('data' in res) || !res.data) {
        const status = 'error' in res ? (res as any).error?.status : undefined;
        // Treat 404 as "no methods configured"
        if (status === 404) {
          const methods404: AdminMFAMethod[] = [
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
          dispatch(setAuthenticationError(null)); // Clear any existing errors
          if (!mfaRequiredSetRef.current) {
            mfaRequiredSetRef.current = true;
            dispatch(setMFARequired(false)); // No methods configured = new user
          }
          return true;
        }
        const errorMsg = "Failed to get MFA methods";
        toast.error(errorMsg);
        dispatch(setAuthenticationError(errorMsg));
        return false;
      }

      const apiMethods: MFAStatusMethod[] = Array.isArray(res.data.methods) ? res.data.methods : [];
      const configured =
        Array.isArray(res.data.configured_methods) && res.data.configured_methods.length > 0
          ? res.data.configured_methods
          : apiMethods.map((method) => ({
              type: method.method_type,
              enabled: typeof method.enabled === "boolean" ? method.enabled : true,
            }));

      const findMethodByType = (type: "webauthn" | "totp") =>
        apiMethods.find((method) => (method?.method_type ?? "").toLowerCase() === type);

      const isConfigured = (type: "webauthn" | "totp") =>
        configured.some(
          (m: any) => (m?.type ?? "").toLowerCase() === type && (m?.enabled ?? true)
        );

      const webauthnMethod = findMethodByType("webauthn");
      const totpMethod = findMethodByType("totp");

      if (res.data.requires_registration) {
        const message =
          (res.data.message || "").trim() ||
          "WebAuthn registration required. Please complete setup.";
        const forcedMethods: AdminMFAMethod[] = [
          {
            type: "webauthn",
            display_name:
              (webauthnMethod?.display_name || "").trim() || "WebAuthn (Biometric)",
            description: message,
            enabled: false,
            recommended: true,
          },
        ];
        dispatch(setAvailableMFAMethods(forcedMethods));
        dispatch(setSelectedMFAMethod("webauthn"));
        dispatch(setCurrentStep("webauthn_setup"));
        dispatch(setAuthenticationError(null));
        if (!mfaRequiredSetRef.current) {
          mfaRequiredSetRef.current = true;
          dispatch(setMFARequired(res.data.mfa_required === true ? true : false));
        }
        toast(message);
        return true;
      }

      const methods: AdminMFAMethod[] = [
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
          description: (totpMethod?.description || "").trim() || "Google Authenticator / Auth",
          enabled: typeof totpMethod?.enabled === "boolean" ? totpMethod.enabled : isConfigured("totp"),
          recommended: typeof totpMethod?.recommended === "boolean" ? totpMethod.recommended : false
        }
      ];
      
      dispatch(setAvailableMFAMethods(methods));
      dispatch(setAuthenticationError(null)); // Clear any existing errors

      // Store MFA required status on first call only (not on refreshes after setup)
      if (!mfaRequiredSetRef.current) {
        mfaRequiredSetRef.current = true;
        console.log("[Auth] MFA status response mfa_required:", res.data.mfa_required);
        dispatch(setMFARequired(res.data.mfa_required === true ? true : false));
      }

      return true;
    } catch (e) {
      const errorMsg = `Failed to get MFA methods: ${e instanceof Error ? e.message : 'Unknown error'}`;
      console.error("❌ Admin getMFAMethods error:", e);
      toast.error("Failed to get MFA methods");
      dispatch(setAuthenticationError(errorMsg));
      return false;
    }
  }, [adminWebauthn.email, mfaStatusCheck, dispatch]);

  const selectMFAMethod = useCallback((method: "webauthn" | "totp") => {
    console.log("🔄 Admin method selected:", method);
    dispatch(setSelectedMFAMethod(method));

    // For first login, always go to setup for the selected method
    if (adminWebauthn.isFirstLogin) {
      const nextStep = method === 'totp' ? 'totp_setup' : 'webauthn_setup';
      console.log("➡️ Admin advancing to step:", nextStep, "(first_login)");
      dispatch(setCurrentStep(nextStep));
      return;
    }

    // Otherwise, advance based on whether the method is already enabled
    const isEnabled = (adminWebauthn.availableMFAMethods || []).some(m => m.type === method && m.enabled);
    const nextStep = method === "totp"
      ? (isEnabled ? "authentication" : "totp_setup")
      : (isEnabled ? "authentication" : "webauthn_setup");
    console.log("➡️ Admin advancing to step:", nextStep, "(enabled:", isEnabled, ")");
    dispatch(setCurrentStep(nextStep));
  }, [dispatch, adminWebauthn.availableMFAMethods, adminWebauthn.isFirstLogin]);

  const setupWebAuthn = useCallback(async (): Promise<boolean> => {
    if (!adminWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    // WebAuthn preflight checks
    if (typeof window === 'undefined' || !('credentials' in navigator)) {
      toast.error("WebAuthn unsupported in this browser");
      return false;
    }
    if (window.isSecureContext !== true) {
      toast.error("Requires HTTPS origin");
      return false;
    }

    try {
      // Admin Setup: begin registration
      const challengeData = await beginAdminRegistration({
        email: adminWebauthn.email!
      }).unwrap();

      // Response shape may be: { publicKey: <creationOptions> }
      let publicKeyData = challengeData.publicKey || challengeData;
      
      // If we still have a nested publicKey, extract it
      if (publicKeyData.publicKey) {
        publicKeyData = publicKeyData.publicKey;
      }
      
      // Validate required fields before processing
      if (!publicKeyData.challenge) {
        console.error('❌ Challenge validation failed');
        console.error('❌ Challenge data structure:', challengeData);
        console.error('❌ Public key data:', publicKeyData);
        throw new Error('Missing challenge in response');
      }
      if (!publicKeyData.user?.id) {
        console.error('❌ Challenge data structure:', challengeData);
        throw new Error('Missing user.id in response');
      }
      if (!publicKeyData.rp) {
        console.error('❌ Challenge data structure:', challengeData);
        throw new Error('Missing rp in response');
      }

      // WebAuthn credential creation - use server options directly
      const credential = await navigator.credentials.create({
        publicKey: {
          challenge: base64UrlToArrayBuffer(publicKeyData.challenge),
          rp: {
            ...publicKeyData.rp
          },
          user: {
            id: base64UrlToArrayBuffer(publicKeyData.user.id),
            name: publicKeyData.user.name || adminWebauthn.email!,
            displayName: publicKeyData.user.displayName || adminWebauthn.email!,
          },
          pubKeyCredParams: publicKeyData.pubKeyCredParams || [
            { type: "public-key", alg: -7 },
            { type: "public-key", alg: -257 }
          ],
          authenticatorSelection: publicKeyData.authenticatorSelection,
          timeout: publicKeyData.timeout || 60000,
          attestation: publicKeyData.attestation || "none",
        }
      });

      if (!credential) {
        throw new Error('Failed to create credential');
      }

      // Confirm registration
      const publicKeyCredential = credential as PublicKeyCredential;
      const response = publicKeyCredential.response as AuthenticatorAttestationResponse;

      const credentialPayload = {
        email: adminWebauthn.email,
        credential: {
          id: credential.id,
          rawId: arrayBufferToBase64Url(publicKeyCredential.rawId),
          type: "public-key" as const,
          response: {
            attestationObject: arrayBufferToBase64Url(response.attestationObject),
            clientDataJSON: arrayBufferToBase64Url(response.clientDataJSON),
          }
        }
      };

      // Finish admin registration
      await finishAdminRegistration(credentialPayload).unwrap();

      // Refresh MFA methods to ensure WebAuthn now shows as enabled
      await getMFAMethods();

      toast.success("Biometric setup complete. Please verify to finish sign-in.");
      dispatch(setSelectedMFAMethod("webauthn"));
      dispatch(setCurrentStep("authentication"));
      return true;
    } catch (error) {
      console.error("❌ Admin WebAuthn setup error:", error);
      const errorMsg = error instanceof Error ? error.message : 'Setup failed';
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [adminWebauthn.email, beginAdminRegistration, finishAdminRegistration, dispatch, getMFAMethods]);

  const setupTOTP = useCallback(async (): Promise<boolean> => {
    if (!adminWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    try {
      // Admin uses "Login" TOTP endpoints
      const result = await totpBeginLoginSetup({
        email: adminWebauthn.email,
        tenant_id: adminWebauthn.tenantId
      });

      if ('data' in result && result.data) {
        dispatch(setTOTPSetupData(result.data));
        dispatch(setTOTPSecret(result.data.secret));
        dispatch(setCurrentStep("totp_setup"));
        return true;
      } else {
        throw new Error('Failed to setup TOTP');
      }
    } catch (error) {
      console.error("❌ Admin TOTP setup error:", error);
      const errorMsg = error instanceof Error ? error.message : 'TOTP setup failed';
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [adminWebauthn.tenantId, adminWebauthn.email, totpBeginLoginSetup, dispatch]);

  const confirmTOTPSetup = useCallback(async (code: string): Promise<boolean> => {
    if (!adminWebauthn.tenantId || !adminWebauthn.email || !adminWebauthn.totpSecret) {
      toast.error("Missing TOTP setup data");
      return false;
    }

    try {
      const result = await totpConfirmLoginSetup({
        email: adminWebauthn.email,
        tenant_id: adminWebauthn.tenantId,
        secret: adminWebauthn.totpSecret,
        code
      });

      if ('data' in result) {
        // Do NOT execute callback here.
        // Per flow: after confirm, user must verify again (login) before callback/token.
        toast.success("TOTP setup confirmed. Please verify with a code to sign in.");
        // Advance to authentication step so user can enter a fresh TOTP code
        dispatch(setCurrentStep("authentication"));
        // Ensure TOTP is the selected method
        dispatch(setSelectedMFAMethod("totp"));
        return true;
      } else {
        throw new Error('TOTP confirmation failed');
      }
    } catch (error) {
      console.error("❌ Admin TOTP confirm error:", error);
      const errorMsg = error instanceof Error ? error.message : 'Invalid TOTP code';
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [adminWebauthn.tenantId, adminWebauthn.email, adminWebauthn.totpSecret, totpConfirmLoginSetup, dispatch, callbackHandler]);

  const authenticateWithWebAuthn = useCallback(async (): Promise<boolean> => {
    if (!adminWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    // WebAuthn preflight checks
    if (typeof window === 'undefined' || !('credentials' in navigator)) {
      toast.error("WebAuthn unsupported in this browser");
      return false;
    }
    if (window.isSecureContext !== true) {
      toast.error("Requires HTTPS origin");
      return false;
    }

    try {
      // Admin Verify: begin authentication
      const challengeData = await beginAdminAuthentication({
        email: adminWebauthn.email!
      }).unwrap();
      // Response shape: { publicKey: <assertionOptions> }
      let publicKeyData = challengeData.publicKey || challengeData;
      
      // If we still have a nested publicKey, extract it
      if (publicKeyData.publicKey) {
        publicKeyData = publicKeyData.publicKey;
      }
      
      // Validate required fields
      if (!publicKeyData.challenge) {
        console.error('❌ Authentication challenge data structure:', challengeData);
        throw new Error('Missing challenge in authentication response');
      }

      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: base64UrlToArrayBuffer(publicKeyData.challenge),
          rpId: publicKeyData.rpId || "app.authsec.dev",
          allowCredentials: publicKeyData.allowCredentials?.map((cred: any) => ({
            ...cred,
            id: base64UrlToArrayBuffer(cred.id)
          })),
          timeout: publicKeyData.timeout || 60000,
          userVerification: publicKeyData.userVerification || "preferred",
        }
      });

      if (!credential) {
        throw new Error('Authentication cancelled or failed');
      }

      const publicKeyCredential = credential as PublicKeyCredential;
      const response = publicKeyCredential.response as AuthenticatorAssertionResponse;

      // Finish admin authentication
      await finishAdminAuthentication({
        email: adminWebauthn.email!,
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
      }).unwrap();

      toast.success("Authentication successful!");

      // Admin flow: Execute callback and store token
      const callbackResult = await callbackHandler.executeCallback(adminWebauthn.email!, adminWebauthn.tenantId!);
      
      if (callbackResult.success && callbackResult.token) {
        dispatch(setAuthToken(callbackResult.token));
        dispatch(completeWebAuthnAuthentication({
          tenantId: adminWebauthn.tenantId!,
          email: adminWebauthn.email!,
          token: callbackResult.token,
        }));

        await notifyNewUserIfNeeded(callbackResult.token);
        dispatch(setCurrentStep("completed"));

        return true;
      } else {
        console.error('❌ WebAuthn auth callback failed:', callbackResult.error);
        throw new Error(callbackResult.error || 'Callback failed');
      }
    } catch (error: any) {
      console.error("❌ Admin WebAuthn auth error:", error);
      const apiMsg = error?.data?.error || error?.data?.message;
      const errorMsg = apiMsg || (error instanceof Error ? error.message : 'Authentication failed');
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [adminWebauthn.tenantId, adminWebauthn.email, beginAdminAuthentication, finishAdminAuthentication, dispatch, callbackHandler, notifyNewUserIfNeeded]);

  const authenticateWithTOTP = useCallback(async (code: string): Promise<boolean> => {
    if (!adminWebauthn.tenantId || !adminWebauthn.email) {
      toast.error("Missing authentication data");
      return false;
    }

    try {
      const result = await totpVerifyLogin({
        email: adminWebauthn.email,
        tenant_id: adminWebauthn.tenantId,
        code
      });

      if ('data' in result && result.data) {
        toast.success("Authentication successful!");
        
        // Check if the TOTP verification already returned a token
        const directToken = (result.data.token || (result.data as any).access_token || (result.data as any).jwt || (result.data as any).jwt_token || (result.data as any).id_token) as string | undefined;
        if (directToken) {
          // TOTP verification returned a token directly, use it
          const token = directToken;
          
          // Store in localStorage for admin flow
          if (typeof window !== 'undefined') {
            localStorage.setItem('jwt_token', token);
          }
          
          dispatch(setAuthToken(token));
          dispatch(completeWebAuthnAuthentication({
            tenantId: adminWebauthn.tenantId!,
            email: adminWebauthn.email!,
            token: token,
          }));

          await notifyNewUserIfNeeded(token);
          dispatch(setCurrentStep("completed"));
          return true;
        } else {
          // Fallback: Execute callback if no direct token
          const callbackResult = await callbackHandler.executeCallback(adminWebauthn.email, adminWebauthn.tenantId);

          if (callbackResult.success && callbackResult.token) {
            dispatch(setAuthToken(callbackResult.token));
            dispatch(completeWebAuthnAuthentication({
              tenantId: adminWebauthn.tenantId!,
              email: adminWebauthn.email!,
              token: callbackResult.token,
            }));

            await notifyNewUserIfNeeded(callbackResult.token);
            dispatch(setCurrentStep("completed"));

            return true;
          } else {
            throw new Error(callbackResult.error || 'No token received');
          }
        }
      } else {
        const errorMsg = "Invalid TOTP code";
        dispatch(setAuthenticationError(errorMsg));
        toast.error(errorMsg);
        return false;
      }
    } catch (error) {
      console.error("❌ Admin TOTP auth error:", error);
      const errorMsg = "Failed to authenticate with TOTP";
      dispatch(setAuthenticationError(errorMsg));
      toast.error(errorMsg);
      return false;
    }
  }, [adminWebauthn.tenantId, adminWebauthn.email, totpVerifyLogin, dispatch, callbackHandler, notifyNewUserIfNeeded]);

  const resetFlow = useCallback(() => {
    dispatch(resetAdminWebAuthnState());
  }, [dispatch]);

  const executeCallback = useCallback(async (email: string, tenantId?: string) => {
    return callbackHandler.executeCallback(email, tenantId);
  }, [callbackHandler]);

  // Navigate back to MFA selection list
  const backToSelection = useCallback(() => {
    dispatch(setCurrentStep("mfa_selection"));
  }, [dispatch]);

  const value: AdminAuthContextType = {
    // State
    currentStep: adminWebauthn.currentStep,
    isFirstLogin: adminWebauthn.isFirstLogin,
    tenantId: adminWebauthn.tenantId,
    email: adminWebauthn.email,
    availableMFAMethods: adminWebauthn.availableMFAMethods,
    selectedMFAMethod: adminWebauthn.selectedMFAMethod,
    totpSetupData: adminWebauthn.totpSetupData,
    isLoading: adminWebauthn.isLoading,
    authenticationError: adminWebauthn.authenticationError,
    authToken: adminWebauthn.authToken,
    
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
  };

  return <AdminAuthContext.Provider value={value}>{children}</AdminAuthContext.Provider>;
};

// Backward compatibility export
export const AdminWebAuthnProvider = AdminAuthProvider;
