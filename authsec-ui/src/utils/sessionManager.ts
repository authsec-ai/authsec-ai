import { decodeJWT, type JWTPayload } from "./jwt";

export interface SessionData {
  token: string | null;
  user: any;
  projects: any[];
  currentProject: any;
  jwtPayload?: JWTPayload;
  tenant_id: string;
  tenant_domain?: string;
  project_id: string;
  client_id: string;
  org_id?: string;
  user_id: string; // User ID for API calls
  expiresAt: number;
  // JWT-derived fields for easy access
  roles?: string[];
  resources?: string[];
  scopes?: string[];
  groups?: string[];
}

const SESSION_KEY = "authsec_session_v2";

/**
 * Session Manager for handling JWT token storage and validation
 * Stores tokens in localStorage only, sessionStorage reserved for cookies
 * Implements security best practices for token storage and expiration
 */
export class SessionManager {
  /**
   * Save session data to localStorage only
   * sessionStorage is reserved for cookies
   */
  static saveSession(data: SessionData): void {
    try {
      localStorage.setItem(SESSION_KEY, JSON.stringify(data));
    } catch (error) {
      console.error("Failed to save session:", error);
    }
  }

  /**
   * Get session data from localStorage only
   */
  static getSession(): SessionData | null {
    try {
      const stored = localStorage.getItem(SESSION_KEY);
      if (!stored) return null;

      const data: SessionData = JSON.parse(stored);
      return data;
    } catch (error) {
      console.error("Failed to parse session data:", error);
      SessionManager.clearSession();
      return null;
    }
  }

  /**
   * Check if current session is valid
   */
  static isSessionValid(): boolean {
    const session = SessionManager.getSession();
    if (!session) return false;

    // Check expiration
    const now = Date.now();
    if (session.expiresAt && now >= session.expiresAt) {
      SessionManager.clearSession();
      return false;
    }

    // If we have a JWT token, validate it
    if (session.token) {
      const jwtPayload = decodeJWT(session.token);
      if (!jwtPayload) {
        SessionManager.clearSession();
        return false;
      }

      // Check JWT expiration
      const jwtExpiry = jwtPayload.exp * 1000; // Convert to milliseconds
      if (now >= jwtExpiry) {
        SessionManager.clearSession();
        return false;
      }
    }

    return true;
  }

  /**
   * Clear all authentication-related storage including cookies and session data
   */
  static clearSession(): void {
    try {
      // Clear localStorage
      localStorage.removeItem(SESSION_KEY);
      localStorage.removeItem("authsec_verification_v2");
      localStorage.removeItem("jwt_token");
      localStorage.removeItem("auth_token");
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("authsec-ui-theme"); // Clear theme preference on logout

      // Clear sessionStorage
      // Preserve OIDC flow values during OIDC login/callback so redirects don't break
      let preserveOidc = false;
      try {
        const path = window.location?.pathname || "";
        const isOidcRoute =
          path.startsWith("/oidc/login") ||
          path.startsWith("/oidc/auth/callback");
        const params = new URLSearchParams(window.location?.search || "");
        const hasOidcParams =
          params.has("login_challenge") ||
          params.has("code") ||
          params.has("state");
        preserveOidc = isOidcRoute || hasOidcParams;
      } catch {
        // ignore if window is not available
      }

      if (!preserveOidc) {
        sessionStorage.removeItem("oauth_state");
        sessionStorage.removeItem("oauth_provider");
        sessionStorage.removeItem("login_challenge");
      }
      sessionStorage.removeItem("webauthn_callback_token");
      sessionStorage.removeItem("webauthn_callback_email");

      // Clear all cookies
      SessionManager.clearAllCookies();

      if (preserveOidc) {
        console.log("✅ Session cleared; OIDC flow values preserved");
      } else {
        console.log("✅ Session and all authentication storage cleared");
      }
    } catch (error) {
      console.error("Failed to clear session:", error);
    }
  }

  /**
   * Clear all cookies by setting them to expire
   */
  private static clearAllCookies(): void {
    try {
      // Get all cookies and set them to expire
      const cookies = document.cookie.split(";");

      for (let cookie of cookies) {
        const eqPos = cookie.indexOf("=");
        const name =
          eqPos > -1 ? cookie.substring(0, eqPos).trim() : cookie.trim();

        // Set cookie to expire (multiple domain/path combinations for thoroughness)
        document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/`;
        document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; domain=${window.location.hostname}`;
        document.cookie = `${name}=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/; domain=.${window.location.hostname}`;
      }
    } catch (error) {
      console.error("Failed to clear cookies:", error);
    }
  }

  /**
   * Get token from current session
   */
  static getToken(): string | null {
    const session = SessionManager.getSession();
    return session?.token || null;
  }

  /**
   * Check if token is close to expiry (within 5 minutes)
   */
  static isTokenNearExpiry(): boolean {
    const session = SessionManager.getSession();
    if (!session || !session.token) return false;

    const jwtPayload = decodeJWT(session.token);
    if (!jwtPayload) return true;

    const now = Date.now();
    const jwtExpiry = jwtPayload.exp * 1000;
    const fiveMinutes = 5 * 60 * 1000;

    return jwtExpiry - now <= fiveMinutes;
  }

  /**
   * Get time remaining until token expires (in milliseconds)
   */
  static getTimeUntilExpiry(): number {
    const session = SessionManager.getSession();
    if (!session || !session.token) return 0;

    const jwtPayload = decodeJWT(session.token);
    if (!jwtPayload) return 0;

    const now = Date.now();
    const jwtExpiry = jwtPayload.exp * 1000;

    return Math.max(0, jwtExpiry - now);
  }

  /**
   * Get user roles from session
   */
  static getUserRoles(): string[] {
    const session = SessionManager.getSession();
    return session?.roles || session?.jwtPayload?.roles || [];
  }

  /**
   * Get user resources from session
   */
  static getUserResources(): string[] {
    const session = SessionManager.getSession();
    return session?.resources || session?.jwtPayload?.resources || [];
  }

  /**
   * Get user scopes from session
   */
  static getUserScopes(): string[] {
    const session = SessionManager.getSession();
    return session?.scopes || session?.jwtPayload?.scopes || [];
  }

  /**
   * Get user groups from session
   */
  static getUserGroups(): string[] {
    const session = SessionManager.getSession();
    return session?.groups || session?.jwtPayload?.groups || [];
  }

  /**
   * Check if user has a specific role
   */
  static hasRole(role: string): boolean {
    return SessionManager.getUserRoles().includes(role);
  }

  /**
   * Check if user has access to a specific resource
   */
  static hasResource(resource: string): boolean {
    return SessionManager.getUserResources().includes(resource);
  }

  /**
   * Check if user has a specific scope
   */
  static hasScope(scope: string): boolean {
    return SessionManager.getUserScopes().includes(scope);
  }
}
