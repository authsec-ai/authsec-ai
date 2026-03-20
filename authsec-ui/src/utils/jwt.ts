export interface JWTPayload {
  aud?: string;
  client_id: string;
  email?: string;
  email_id: string;
  exp: number;
  groups: string[];
  iat: number;
  iss: string;
  nbf?: number;
  project_id: string;
  resources: string[];
  roles: string[];
  scope?: string;
  scopes: string[];
  tenant_domain?: string;
  tenant_id: string;
  token_type: string;
  [key: string]: unknown;
}

export const decodeJWT = (token: string): JWTPayload | null => {
  try {
    // JWT structure: header.payload.signature
    const parts = token.split('.');
    if (parts.length !== 3) {
      console.error('Invalid JWT: Expected 3 parts, got', parts.length);
      return null;
    }

    // Decode the payload (base64url) - this is the main part we care about
    const payloadPart = parts[1];
    
    // Add padding if needed for base64url
    const paddedPayload = payloadPart + '='.repeat((4 - payloadPart.length % 4) % 4);
    
    // Convert base64url to base64
    const base64Payload = paddedPayload.replace(/-/g, '+').replace(/_/g, '/');
    
    // Decode and parse
    const decodedPayloadString = atob(base64Payload);
    const decoded = JSON.parse(decodedPayloadString);

    const scopes =
      Array.isArray(decoded.scopes)
        ? decoded.scopes
        : typeof decoded.scope === "string"
          ? decoded.scope.split(/\s+/).filter(Boolean)
          : [];

    const normalized: JWTPayload = {
      ...decoded,
      client_id: decoded.client_id || decoded.sub || "",
      email_id: decoded.email_id || decoded.email || "",
      project_id: decoded.project_id || "",
      tenant_id: decoded.tenant_id || "",
      token_type: decoded.token_type || "",
      scopes,
      roles: Array.isArray(decoded.roles) ? decoded.roles : [],
      resources: Array.isArray(decoded.resources) ? decoded.resources : [],
      groups: Array.isArray(decoded.groups) ? decoded.groups : [],
    };
    
    // Debug logging (can be removed in production)
    console.warn('📦 JWT Payload decoded:', normalized);
    
    return normalized;
  } catch (error) {
    console.error('Failed to decode JWT:', error);
    return null;
  }
};

export const isTokenExpired = (token: string): boolean => {
  const payload = decodeJWT(token);
  if (!payload) return true;
  
  const currentTime = Math.floor(Date.now() / 1000);
  return payload.exp < currentTime;
};

export const createUserFromJWT = (payload: JWTPayload): {
  id: string;
  email: string;
  first_name?: string;
  last_name?: string;
  avatar_url?: string;
} => {
  return {
    id: payload.client_id,
    email: payload.email_id,
    first_name: undefined, // Not available in JWT
    last_name: undefined, // Not available in JWT
    avatar_url: undefined, // Not available in JWT
  };
};
