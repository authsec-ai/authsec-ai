/**
 * OIDC WebAuthn Slice - For /oidc-login end-user authentication flow
 * 
 * Handles OIDC-specific WebAuthn/TOTP authentication with token display (no storage)
 */

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export interface MFAMethod {
  type: "webauthn" | "totp";
  display_name: string;
  description: string;
  enabled: boolean;
  recommended: boolean;
}

interface OIDCWebAuthnState {
  // Core flow state
  currentStep: "login" | "mfa_selection" | "webauthn_setup" | "totp_setup" | "authentication" | "token_display";
  isFirstLogin: boolean;

  // User context
  tenantId: string | null;
  email: string | null;
  clientId: string | null; // Add client_id to capture from initial response

  // MFA configuration
  availableMFAMethods: MFAMethod[];
  selectedMFAMethod: "webauthn" | "totp" | null;
  
  // TOTP setup state
  totpSecret: string | null;
  totpSetupData: {
    account: string;
    issuer: string;
    qr_code: string;
    secret: string;
  } | null;
  
  // UI state
  isLoading: boolean;
  authenticationError: string | null;
  
  // OIDC-specific: display token (not stored)
  displayToken: string | null;
  tokenDisplayed: boolean;

  // Track if MFA is required (for new user notification)
  mfaRequired: boolean | null;
}

const initialState: OIDCWebAuthnState = {
  currentStep: "login",
  isFirstLogin: false,
  tenantId: null,
  email: null,
  clientId: null,
  availableMFAMethods: [],
  selectedMFAMethod: null,
  totpSecret: null,
  totpSetupData: null,
  isLoading: false,
  authenticationError: null,
  displayToken: null,
  tokenDisplayed: false,
  mfaRequired: null,
};

const oidcWebAuthnSlice = createSlice({
  name: "oidcWebAuthn",
  initialState,
  reducers: {
    // Flow control
    setCurrentStep: (state, action: PayloadAction<OIDCWebAuthnState["currentStep"]>) => {
      state.currentStep = action.payload;
    },
    
    // User context
    setLoginData: (state, action: PayloadAction<{ tenantId: string; email: string; isFirstLogin: boolean; clientId?: string }>) => {
      state.tenantId = action.payload.tenantId;
      state.email = action.payload.email;
      state.isFirstLogin = action.payload.isFirstLogin;
      if (action.payload.clientId) {
        state.clientId = action.payload.clientId;
      }
    },

    setClientId: (state, action: PayloadAction<string>) => {
      state.clientId = action.payload;
    },
    
    // MFA setup
    setAvailableMFAMethods: (state, action: PayloadAction<MFAMethod[]>) => {
      state.availableMFAMethods = action.payload;
    },
    
    setSelectedMFAMethod: (state, action: PayloadAction<"webauthn" | "totp">) => {
      state.selectedMFAMethod = action.payload;
    },
    
    // TOTP setup
    setTOTPSecret: (state, action: PayloadAction<string>) => {
      state.totpSecret = action.payload;
    },
    
    setTOTPSetupData: (state, action: PayloadAction<OIDCWebAuthnState["totpSetupData"]>) => {
      state.totpSetupData = action.payload;
    },
    
    // UI state
    setIsLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    
    setAuthenticationError: (state, action: PayloadAction<string | null>) => {
      state.authenticationError = action.payload;
    },
    
    // OIDC-specific: token display management
    setDisplayToken: (state, action: PayloadAction<string>) => {
      state.displayToken = action.payload;
      state.tokenDisplayed = true;
      state.currentStep = "token_display";
    },
    
    clearDisplayToken: (state) => {
      state.displayToken = null;
      state.tokenDisplayed = false;
    },

    // MFA requirement tracking (for new user notification)
    setMFARequired: (state, action: PayloadAction<boolean>) => {
      state.mfaRequired = action.payload;
    },

    // Reset state
    resetOIDCWebAuthnState: (state) => {
      Object.assign(state, initialState);
    },
  },
});

export const {
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
  clearDisplayToken,
  setMFARequired,
  resetOIDCWebAuthnState,
} = oidcWebAuthnSlice.actions;

export default oidcWebAuthnSlice.reducer;