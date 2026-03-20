/**
 * WebAuthn Operations Hook - Pure WebAuthn browser API integration
 * 
 * Handles WebAuthn browser API operations and credential conversions
 * Pages handle their own flow logic and navigation
 */

import { useCallback } from 'react';
import { 
  useWebauthnCallbackMutation,
  useBeginWebAuthnAuthMutation,
  useFinishWebAuthnAuthMutation,
  useBeginWebAuthnRegistrationMutation,
  useFinishWebAuthnRegistrationMutation,
} from '../../app/api/webauthnApi';
import type { WebAuthnCredential, WebAuthnRegistrationCredential } from '../../app/api/webauthnApi';

export interface UseWebAuthnAuthResult {
  // Authentication flow
  authenticateUser: (email: string, tenantId: string) => Promise<boolean>;
  
  // Registration flow
  registerUser: (email: string, tenantId: string) => Promise<boolean>;
  
  // Callback handling
  handleCallback: (email: string, tenantId?: string, flowContext?: 'admin' | 'oidc') => Promise<{ success: boolean; token?: string; error?: string }>;
  
  // Loading states
  isAuthenticating: boolean;
  isRegistering: boolean;
  isHandlingCallback: boolean;
}

// Helpers for base64url conversions
const arrayBufferToBase64Url = (buffer: ArrayBuffer): string =>
  btoa(String.fromCharCode(...new Uint8Array(buffer)))
    .replace(/\+/g, '-')
    .replace(/\//g, '_')
    .replace(/=/g, '');

export function useWebAuthnAuth(): UseWebAuthnAuthResult {
  // RTK Query mutations
  const [beginAuth, { isLoading: isBeginningAuth }] = useBeginWebAuthnAuthMutation();
  const [finishAuth, { isLoading: isFinishingAuth }] = useFinishWebAuthnAuthMutation();
  const [beginRegistration, { isLoading: isBeginningRegistration }] = useBeginWebAuthnRegistrationMutation();
  const [finishRegistration, { isLoading: isFinishingRegistration }] = useFinishWebAuthnRegistrationMutation();
  const [webauthnCallback, { isLoading: isHandlingCallback }] = useWebauthnCallbackMutation();
  
  // Authentication flow
  const authenticateUser = useCallback(async (email: string, tenantId: string): Promise<boolean> => {
    try {
      // Begin authentication
      const beginResult = await beginAuth({ email, tenant_id: tenantId }).unwrap();
      // Response shape: { publicKey: <assertionOptions> }
      const publicKeyOpts = (beginResult && (beginResult.publicKey || beginResult));
      const pk = publicKeyOpts?.publicKey ? publicKeyOpts.publicKey : publicKeyOpts;
      
      if (!pk?.challenge) {
        throw new Error('Invalid authentication options received');
      }

      // Get credential from browser
      const credential = await navigator.credentials.get({
        publicKey: {
          challenge: Uint8Array.from(atob((pk.challenge as string).replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0)).buffer,
          rpId: pk.rpId || "app.authsec.dev",
          allowCredentials: pk.allowCredentials?.map((cred: any) => ({
            ...cred,
            id: Uint8Array.from(atob((cred.id as string).replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0)).buffer,
          })),
          timeout: pk.timeout,
          userVerification: pk.userVerification,
        }
      }) as PublicKeyCredential;
      
      if (!credential) {
        throw new Error('No credential received from browser');
      }
      
      // Convert credential to expected format
      const webauthnCredential: WebAuthnCredential = {
        id: credential.id,
        rawId: arrayBufferToBase64Url(credential.rawId),
        type: 'public-key',
        response: {
          clientDataJSON: arrayBufferToBase64Url((credential.response as AuthenticatorAssertionResponse).clientDataJSON),
          authenticatorData: arrayBufferToBase64Url((credential.response as AuthenticatorAssertionResponse).authenticatorData),
          signature: arrayBufferToBase64Url((credential.response as AuthenticatorAssertionResponse).signature),
          userHandle: (credential.response as AuthenticatorAssertionResponse).userHandle ? 
            arrayBufferToBase64Url((credential.response as AuthenticatorAssertionResponse).userHandle!) : null
        }
      };
      
      // Finish authentication
      await finishAuth({
        email,
        tenant_id: tenantId,
        credential: webauthnCredential
      }).unwrap();
      
      return true;
      
    } catch (error) {
      console.error('WebAuthn authentication error:', error);
      throw error;
    }
  }, [beginAuth, finishAuth]);
  
  // Registration flow
  const registerUser = useCallback(async (email: string, tenantId: string): Promise<boolean> => {
    try {
      // Begin registration
      const beginResult = await beginRegistration({ email, tenant_id: tenantId }).unwrap();
      // Response shape: { publicKey: <creationOptions> }
      const publicKeyOpts = (beginResult && (beginResult.publicKey || beginResult));
      const pk = publicKeyOpts?.publicKey ? publicKeyOpts.publicKey : publicKeyOpts;

      if (!pk?.challenge || !pk?.user?.id || !pk?.rp) {
        throw new Error('Invalid registration options received');
      }

      // Get credential from browser
      const credential = await navigator.credentials.create({
        publicKey: {
          challenge: Uint8Array.from(atob((pk.challenge as string).replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0)).buffer,
          rp: {
            ...pk.rp
          },
          user: {
            id: Uint8Array.from(atob((pk.user.id as string).replace(/-/g, '+').replace(/_/g, '/')), c => c.charCodeAt(0)).buffer,
            name: pk.user.name,
            displayName: pk.user.displayName,
          },
          pubKeyCredParams: pk.pubKeyCredParams,
          authenticatorSelection: pk.authenticatorSelection,
          timeout: pk.timeout,
          attestation: pk.attestation,
        }
      }) as PublicKeyCredential;
      
      if (!credential) {
        throw new Error('No credential received from browser');
      }
      
      // Convert credential to expected format
      const webauthnCredential: WebAuthnRegistrationCredential = {
        id: credential.id,
        rawId: arrayBufferToBase64Url(credential.rawId),
        type: 'public-key',
        response: {
          attestationObject: arrayBufferToBase64Url((credential.response as AuthenticatorAttestationResponse).attestationObject),
          clientDataJSON: arrayBufferToBase64Url((credential.response as AuthenticatorAttestationResponse).clientDataJSON)
        }
      };
      
      // Finish registration
      await finishRegistration({
        email,
        tenant_id: tenantId,
        credential: webauthnCredential
      }).unwrap();
      
      return true;
      
    } catch (error) {
      console.error('WebAuthn registration error:', error);
      throw error;
    }
  }, [beginRegistration, finishRegistration]);
  
  // Callback handling
  const handleCallback = useCallback(async (
    email: string, 
    tenantId?: string,
    flowContext?: 'admin' | 'oidc'
  ): Promise<{ success: boolean; token?: string; error?: string }> => {
    try {
      const result = await webauthnCallback({
        email,
        mfa_verified: true,
        tenant_id: tenantId,
        flow_context: flowContext
      }).unwrap();
      
      return result;
      
    } catch (error) {
      console.error('WebAuthn callback error:', error);
      return {
        success: false,
        error: error instanceof Error ? error.message : 'Callback failed'
      };
    }
  }, [webauthnCallback]);
  
  return {
    authenticateUser,
    registerUser,
    handleCallback,
    isAuthenticating: isBeginningAuth || isFinishingAuth,
    isRegistering: isBeginningRegistration || isFinishingRegistration,
    isHandlingCallback,
  };
}
