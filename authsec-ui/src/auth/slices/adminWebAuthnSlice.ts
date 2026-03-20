/**
 * Admin WebAuthn Slice - For /uflow/login admin authentication flow
 * 
 * Handles admin-specific WebAuthn/TOTP authentication with token storage
 */

import { createSlice, type PayloadAction } from "@reduxjs/toolkit";

export interface MFAMethod {
  type: "webauthn" | "totp";
  display_name: string;
  description: string;
  enabled: boolean;
  recommended: boolean;
}

interface AdminWebAuthnState {
  // Core flow state
  currentStep: "login" | "mfa_selection" | "webauthn_setup" | "totp_setup" | "authentication" | "completed";
  isFirstLogin: boolean;
  
  // User context
  tenantId: string | null;
  email: string | null;
  
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
  
  // Admin-specific: stored token
  authToken: string | null;

  // Track if MFA is required (for new user notification)
  mfaRequired: boolean | null;
}

const initialState: AdminWebAuthnState = {
  currentStep: "login",
  isFirstLogin: false,
  tenantId: null,
  email: null,
  availableMFAMethods: [],
  selectedMFAMethod: null,
  totpSecret: null,
  totpSetupData: null,
  isLoading: false,
  authenticationError: null,
  authToken: null,
  mfaRequired: null,
};

const adminWebAuthnSlice = createSlice({
  name: "adminWebAuthn",
  initialState,
  reducers: {
    // Flow control
    setCurrentStep: (state, action: PayloadAction<AdminWebAuthnState["currentStep"]>) => {
      state.currentStep = action.payload;
    },
    
    // User context
    setLoginData: (state, action: PayloadAction<{ tenantId: string; email: string; isFirstLogin: boolean }>) => {
      state.tenantId = action.payload.tenantId;
      state.email = action.payload.email;
      state.isFirstLogin = action.payload.isFirstLogin;
      // Reset flow-specific state on new login context
      state.availableMFAMethods = [];
      state.selectedMFAMethod = null;
      state.totpSecret = null;
      state.totpSetupData = null;
      state.authenticationError = null;
      state.authToken = null;
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
    
    setTOTPSetupData: (state, action: PayloadAction<AdminWebAuthnState["totpSetupData"]>) => {
      state.totpSetupData = action.payload;
    },
    
    // UI state
    setIsLoading: (state, action: PayloadAction<boolean>) => {
      state.isLoading = action.payload;
    },
    
    setAuthenticationError: (state, action: PayloadAction<string | null>) => {
      state.authenticationError = action.payload;
    },
    
    // Admin-specific: auth token management
    setAuthToken: (state, action: PayloadAction<string>) => {
      state.authToken = action.payload;
    },
    
    clearAuthToken: (state) => {
      state.authToken = null;
    },

    // MFA requirement tracking (for new user notification)
    setMFARequired: (state, action: PayloadAction<boolean>) => {
      state.mfaRequired = action.payload;
    },

    // Reset state
    resetAdminWebAuthnState: (state) => {
      Object.assign(state, initialState);
    },
  },
});

export const {
  setCurrentStep,
  setLoginData,
  setAvailableMFAMethods,
  setSelectedMFAMethod,
  setTOTPSecret,
  setTOTPSetupData,
  setIsLoading,
  setAuthenticationError,
  setAuthToken,
  clearAuthToken,
  setMFARequired,
  resetAdminWebAuthnState,
} = adminWebAuthnSlice.actions;

export default adminWebAuthnSlice.reducer;
