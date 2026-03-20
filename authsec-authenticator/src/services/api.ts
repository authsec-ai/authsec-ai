import axios from 'axios';
import { Platform } from 'react-native';

// Backend API URL
// PRODUCTION: Use your deployed API
export const BASE_URL = 'https://prod.api.authsec.ai';

// For local development, uncomment this:
// export const BASE_URL = 'http://192.168.1.XXX:7468';

const api = axios.create({
  baseURL: BASE_URL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface LoginResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

export interface PrecheckResponse {
  tenant_domain: string;
  tenant_id: string;
  client_id?: string; // Made optional
  has_password?: boolean;
  auth_methods?: string[];
}

export interface WebAuthnCallbackResponse {
  access_token: string;
  token_type: string;
  expires_in: number;
}

export interface RegisterDeviceResponse {
  success: boolean;
  device_id: string;
  message: string;
}

export interface RespondToCIBAResponse {
  success: boolean;
  message: string;
}

export interface EndUserLoginResponse {
  tenant_id: string;
  email: string;
  first_login: boolean;
  otp_required: boolean;
  mfa_required?: boolean;
  token?: string;
}

export interface TOTPRegistrationResponse {
  success: boolean;
  device_id: string;
  secret: string;
  qr_code_url: string;
  backup_codes?: string[];
  message: string;
}

export interface TOTPConfirmResponse {
  success: boolean;
  message: string;
}

export interface TOTPLoginResponse {
  success: boolean;
  access_token?: string;
  token_type?: string;
  expires_in?: number;
  error?: string;
  message?: string;
}

export interface OIDCProvider {
  name?: string; // Legacy support
  provider_name?: string;
  display_name?: string;
  logo_url?: string;
  auth_url?: string;
}

export interface OIDCAuthURLResponse {
  success?: boolean;
  auth_url: string;
}

export interface OIDCProviderConfig {
  additional_params: any;
  auth_url: string;
  client_id: string;
  client_secret: string;
  issuer_url: string;
  jwks_url: string;
  scopes: string[];
  token_url: string;
  type: string;
  user_info_url: string;
}

export interface OIDCProviderDetail {
  provider_name: string;
  display_name: string;
  is_active: boolean;
  sort_order: number;
  callback_url: string;
  config: OIDCProviderConfig;
}

export interface OIDCPageDataResponse {
  client_id: string;
  success: boolean;
  login_challenge: string;
  tenant_name: string;
  client_name: string;
  providers: OIDCProviderDetail[];
  base_url: string;
}

/**
 * Precheck to get tenant domain from email
 */
export const loginPrecheck = async (email: string): Promise<PrecheckResponse> => {
  try {
    const response = await api.post('/uflow/auth/admin/login/precheck', {
      email,
    });
    return response.data;
  } catch (error: any) {
    console.error('Precheck error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to verify email.',
    );
  }
};

/**
 * Login user with email and password
 * This is step 1 - validates credentials
 */
export const login = async (
  email: string,
  password: string,
  tenantDomain: string,
): Promise<LoginResponse> => {
  try {
    const response = await api.post('/uflow/login', {
      email,
      password,
      tenant_domain: tenantDomain,
    });
    return response.data;
  } catch (error: any) {
    console.error('Login error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Login failed. Please try again.',
    );
  }
};

/**
 * WebAuthn Callback - Final step to get JWT token
 * This is step 2 - called after login to get the actual access token
 */
export const webauthnCallback = async (
  email: string,
  tenantId: string,
): Promise<WebAuthnCallbackResponse> => {
  try {
    const response = await api.post('/uflow/login/webauthn-callback', {
      email,
      mfa_verified: true,
      tenant_id: tenantId,
      flow_context: 'admin',
    });
    return response.data;
  } catch (error: any) {
    console.error(
      'WebAuthn callback error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error || 'Failed to complete authentication.',
    );
  }
};

/**
 * Register device for push notifications
 */
export const registerDeviceToken = async (
  deviceToken: string,
  authToken: string,
): Promise<RegisterDeviceResponse> => {
  try {
    const response = await api.post(
      '/uflow/auth/tenant/ciba/register-device',
      {
        device_token: deviceToken,
        platform: Platform.OS,
        device_name: Platform.OS === 'ios' ? 'iPhone' : 'Android Phone',
        device_model: Platform.OS,
        app_version: '1.0.0',
        os_version: Platform.Version.toString(),
      },
      {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      },
    );
    return response.data;
  } catch (error: any) {
    console.error(
      'Register device error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error ||
      'Failed to register device. Please try again.',
    );
  }
};

/**
 * Respond to CIBA authentication request (approve or deny)
 */
export const respondToCIBA = async (
  authReqId: string,
  approved: boolean,
  biometricVerified: boolean,
  authToken: string,
): Promise<RespondToCIBAResponse> => {
  try {
    const response = await api.post(
      '/uflow/auth/tenant/ciba/respond',
      {
        auth_req_id: authReqId,
        approved,
        biometric_verified: biometricVerified,
      },
      {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      },
    );
    return response.data;
  } catch (error: any) {
    console.error(
      'Respond to CIBA error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error ||
      'Failed to respond to authentication request.',
    );
  }
};

/**
 * Precheck to get tenant domain from email for end-user
 */
export const loginPrecheckEndUser = async (email: string): Promise<PrecheckResponse> => {
  try {
    const response = await api.post('/uflow/auth/enduser/login/precheck', {
      email,
    });
    return response.data;
  } catch (error: any) {
    console.error('Precheck error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to verify email.',
    );
  }
};

/**
 * Login end-user with email and password
 */
export const loginEndUser = async (
  email: string,
  password: string,
  tenantDomain: string,
  clientId: string,
): Promise<EndUserLoginResponse> => {
  try {
    const response = await api.post('/uflow/user/login', {
      email,
      password,
      tenant_domain: tenantDomain,
      client_id: clientId,
    });
    return response.data;
  } catch (error: any) {
    console.error('Login error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Login failed. Please try again.',
    );
  }
};

/**
 * WebAuthn Callback for end-user
 */
export const webauthnCallbackEndUser = async (
  email: string,
  tenantId: string,
  clientId: string,
): Promise<WebAuthnCallbackResponse> => {
  try {
    const response = await api.post('/uflow/auth/enduser/webauthn-callback', {
      email,
      mfa_verified: true,
      tenant_id: tenantId,
      flow_context: 'enduser',
      client_id: clientId,
    });
    return response.data;
  } catch (error: any) {
    console.error(
      'WebAuthn callback error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error || 'Failed to complete authentication.',
    );
  }
};

/**
 * Register TOTP device for admin users
 */
export const registerTOTPDeviceAdmin = async (
  deviceName: string,
  authToken: string,
): Promise<TOTPRegistrationResponse> => {
  try {
    const response = await api.post(
      '/uflow/auth/totp/register',
      {
        device_name: deviceName,
        device_type: 'mobile',
      },
      {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      },
    );
    return response.data;
  } catch (error: any) {
    console.error(
      'Register TOTP device error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error || 'Failed to register TOTP device.',
    );
  }
};

/**
 * Confirm TOTP device registration for admin users
 */
export const confirmTOTPDeviceAdmin = async (
  deviceId: string,
  totpCode: string,
  authToken: string,
): Promise<TOTPConfirmResponse> => {
  try {
    const response = await api.post(
      '/uflow/auth/totp/confirm',
      {
        device_id: deviceId,
        totp_code: totpCode,
      },
      {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      },
    );
    return response.data;
  } catch (error: any) {
    console.error(
      'Confirm TOTP device error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error || 'Failed to confirm TOTP device.',
    );
  }
};

/**
 * Register TOTP device for tenant/enduser
 */
export const registerTOTPDeviceEndUser = async (
  deviceName: string,
  authToken: string,
): Promise<TOTPRegistrationResponse> => {
  try {
    const response = await api.post(
      '/uflow/auth/tenant/totp/register',
      {
        device_name: deviceName,
        device_type: 'mobile',
      },
      {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      },
    );
    return response.data;
  } catch (error: any) {
    console.error(
      'Register TOTP device error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error || 'Failed to register TOTP device.',
    );
  }
};

/**
 * Confirm TOTP device registration for tenant/enduser
 */
export const confirmTOTPDeviceEndUser = async (
  deviceId: string,
  totpCode: string,
  authToken: string,
): Promise<TOTPConfirmResponse> => {
  try {
    const response = await api.post(
      '/uflow/auth/tenant/totp/confirm',
      {
        device_id: deviceId,
        totp_code: totpCode,
      },
      {
        headers: {
          Authorization: `Bearer ${authToken}`,
        },
      },
    );
    return response.data;
  } catch (error: any) {
    console.error(
      'Confirm TOTP device error:',
      error.response?.data || error.message,
    );
    throw new Error(
      error.response?.data?.error || 'Failed to confirm TOTP device.',
    );
  }
};

/**
 * Get OIDC auth URLs for a client ID
 */
export const getOIDCAuthURL = async (clientId: string): Promise<OIDCAuthURLResponse> => {
  try {
    const response = await api.post('/uflow/oidc/auth-url', {
      client_id: clientId,
    });
    return response.data;
  } catch (error: any) {
    console.error('Get OIDC auth URL error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to get OIDC providers.',
    );
  }
};

/**
 * Get OIDC page data for a specific login challenge
 */
export const getOIDCPageData = async (loginChallenge: string): Promise<OIDCPageDataResponse> => {
  try {
    console.log('🌐 API Request - Getting OIDC page data');
    console.log('📍 Base URL:', api.defaults.baseURL);
    console.log('🔑 Login challenge:', loginChallenge.substring(0, 30) + '...');
    
    const response = await api.get('/hmgr/login/page-data', {
      params: {
        login_challenge: loginChallenge,
      },
    });
    
    console.log('✅ OIDC page data received successfully');
    return response.data;
  } catch (error: any) {
    console.error('❌ Get OIDC page data error (FULL):', error);
    console.error('Response data:', error.response?.data);
    console.error('Status:', error.response?.status);
    console.error('Message:', error.message);
    console.error('Network Error?', error.code === 'ENOTFOUND' || error.code === 'ECONNREFUSED');
    
    throw new Error(
      error.response?.data?.error || error.message || 'Failed to get OIDC page data.',
    );
  }
};

/**
 * Initiate OIDC authentication with a provider
 */
export const initiateOIDC = async (
  provider: string,
  loginChallenge: string,
  baseUrl: string = 'https://prod.api.authsec.ai/hmgr',
  redirectUri?: string
): Promise<any> => {
  try {
    const requestBody: any = {
      login_challenge: loginChallenge,
    };
    
    // Include redirect_uri if provided (for mobile custom schemes)
    if (redirectUri) {
      requestBody.redirect_uri = redirectUri;
    }
    
    const response = await api.post(
      `${baseUrl}/auth/initiate/${provider}`,
      requestBody
    );
    return response.data;
  } catch (error: any) {
    console.error('Initiate OIDC error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to initiate OIDC authentication.',
    );
  }
};

/**
 * Handle OIDC callback (after Google authentication)
 */
export const oidcCallback = async (code: string, state: string): Promise<any> => {
  try {
    console.log('📤 Calling /hmgr/auth/callback');
    console.log('  - Code length:', code.length);
    console.log('  - State length:', state.length);
    console.log('  - Code preview:', code.substring(0, 30) + '...');
    console.log('  - State preview:', state.substring(0, 50) + '...');
    
    const response = await api.post('/hmgr/auth/callback', {
      code,
      state,
    });
    
    console.log('✅ OIDC callback successful');
    return response.data;
  } catch (error: any) {
    console.error('❌ OIDC callback error (FULL):', error);
    console.error('Response status:', error.response?.status);
    console.error('Response data:', error.response?.data);
    console.error('Error message:', error.message);
    
    throw new Error(
      error.response?.data?.error || error.message || 'Failed to process OIDC callback.',
    );
  }
};

/**
 * Exchange OIDC code for access token
 */
export const exchangeOIDCToken = async (
  loginChallenge: string,
  code: string,
  state: string,
  provider: string,
  redirectUri: string
): Promise<any> => {
  try {
    const requestPayload = {
      login_challenge: loginChallenge,
      code,
      state,
      provider,
      redirect_uri: redirectUri,
    };
    
    console.log('📤 Exchange token request payload:');
    console.log('- login_challenge:', loginChallenge.substring(0, 30) + '...');
    console.log('- code length:', code.length, 'preview:', code.substring(0, 20) + '...');
    console.log('- code full:', code); // Log full code for debugging
    console.log('- state length:', state.length, 'preview:', state.substring(0, 30) + '...');
    console.log('- provider:', provider);
    console.log('- redirect_uri:', redirectUri);
    
    const response = await api.post('/hmgr/auth/exchange-token', requestPayload);
    
    console.log('✅ Exchange token response:', response.data);
    return response.data;
  } catch (error: any) {
    console.error('❌ Exchange token error details:');
    console.error('- Status:', error.response?.status);
    console.error('- Data:', error.response?.data);
    console.error('- Message:', error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to exchange OIDC token.',
    );
  }
};

/**
 * Login with OIDC access token
 */
export const oidcLogin = async (accessToken: string, expiresIn: number): Promise<any> => {
  try {
    const response = await api.post('/uflow/user/oidc/login', {
      access_token: accessToken,
      expires_in: expiresIn,
    });
    return response.data;
  } catch (error: any) {
    console.error('OIDC login error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to login with OIDC.',
    );
  }
};

/**
 * WebAuthn callback for OIDC flow
 */
export const webauthnCallbackOIDC = async (
  email: string,
  tenantId: string,
  clientId: string
): Promise<any> => {
  try {
    const response = await api.post('/uflow/auth/enduser/webauthn-callback', {
      email,
      mfa_verified: true,
      tenant_id: tenantId,
      flow_context: 'enduser',
      client_id: clientId,
    });
    return response.data;
  } catch (error: any) {
    console.error('WebAuthn callback error:', error.response?.data || error.message);
    throw new Error(
      error.response?.data?.error || 'Failed to complete authentication.',
    );
  }
};

export default api;
