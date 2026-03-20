import { createSlice } from "@reduxjs/toolkit";
import type { PayloadAction } from "@reduxjs/toolkit";
import { authApi } from "../../app/api/authApi";
import { decodeJWT } from "../../utils/jwt";
import { SessionManager } from "../../utils/sessionManager";

export interface AuthUser {
  id: string;
  email: string;
  first_name?: string;
  last_name?: string;
  avatar_url?: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  slug: string;
  ownerId: string;
  role: "owner" | "admin" | "member";
}

interface AuthState {
  isAuthenticated: boolean;
  user: AuthUser | null;
  token: string | null;
  projects: Project[];
  currentProject: Project | null;
  jwtPayload: any | null; // Store full JWT payload for additional info
}

// Initialize state from session if valid
const loadInitialState = (): AuthState => {
  if (SessionManager.isSessionValid()) {
    const session = SessionManager.getSession();
    if (session) {
      return {
        isAuthenticated: true,
        user: session.user,
        token: session.token,
        projects: session.projects,
        currentProject: session.currentProject,
        jwtPayload: session.jwtPayload || null,
      };
    }
  }

  return {
    isAuthenticated: false,
    user: null,
    token: null,
    projects: [],
    currentProject: null,
    jwtPayload: null,
  };
};

const initialState: AuthState = loadInitialState();

/**
 * Auth slice for managing authentication state
 * Handles user login, logout, and token management with RTK Query integration
 */
const authSlice = createSlice({
  name: "auth",
  initialState,
  reducers: {
    logout: (state) => {
      state.isAuthenticated = false;
      state.user = null;
      state.token = null;
      state.projects = [];
      state.currentProject = null;
      state.jwtPayload = null;
      // Clear session using SessionManager
      SessionManager.clearSession();
    },
    updateToken: (state, action: PayloadAction<string>) => {
      state.token = action.payload;
    },
    updateUser: (state, action: PayloadAction<AuthState["user"]>) => {
      state.user = action.payload;
    },
    setProjects: (state, action: PayloadAction<Project[]>) => {
      state.projects = action.payload;
    },
    setCurrentProject: (state, action: PayloadAction<Project | null>) => {
      state.currentProject = action.payload;
    },
    // Handle successful WebAuthn authentication
    completeWebAuthnAuthentication: (
      state,
      action: PayloadAction<{ tenantId: string; email: string; token?: string | null }>
    ) => {
      const { tenantId, email, token } = action.payload;

      console.log("🔐 completeWebAuthnAuthentication called:", {
        tenantId,
        email,
        hasToken: !!token,
      });

      // Decode JWT token to get correct project_id if available
      let jwtPayload = null;
      let actualProjectId = tenantId; // fallback to tenantId
      let actualClientId = tenantId; // fallback to tenantId
      
      if (token) {
        jwtPayload = decodeJWT(token);
        if (jwtPayload) {
          actualProjectId = jwtPayload.project_id || tenantId;
          actualClientId = jwtPayload.client_id || tenantId;
        }
      }

      // Create user object from WebAuthn data
      const user: AuthUser = {
        id: actualClientId,
        email,
        first_name: undefined,
        last_name: undefined,
        avatar_url: undefined,
      };

      // Create a default project using correct project_id
      const project: Project = {
        id: actualProjectId,
        name: "Default Project",
        description: "Your default project",
        slug: `project-${actualProjectId.slice(0, 8)}`,
        ownerId: actualClientId,
        role: "owner",
      };

      // Set authenticated state
      state.isAuthenticated = true;
      state.user = user;
      state.token = token || null; // Use JWT token from WebAuthn callback if available
      state.projects = [project];
      state.currentProject = project;
      state.jwtPayload = jwtPayload;

      // Store session data with correct IDs from JWT
      SessionManager.saveSession({
        token: token || null,
        user,
        projects: [project],
        currentProject: project,
        tenant_id: tenantId,
        tenant_domain: jwtPayload?.tenant_domain,
        project_id: actualProjectId, // Use project_id from JWT
        client_id: actualClientId, // Use client_id from JWT
        user_id: user?.id || actualClientId, // Use user ID or client_id as fallback
        jwtPayload,
        // Extract JWT fields for easy access
        roles: jwtPayload?.roles || [],
        resources: jwtPayload?.resources || [],
        scopes: jwtPayload?.scopes || [],
        groups: jwtPayload?.groups || [],
        expiresAt: Date.now() + 30 * 24 * 60 * 60 * 1000, // 30 days
      });

      console.log("✅ Authentication state updated:", {
        isAuthenticated: state.isAuthenticated,
        userId: state.user?.id,
        hasToken: !!state.token,
      });
    },
    checkSession: (state) => {
      // Check if current session is still valid
      if (!SessionManager.isSessionValid()) {
        // Session expired or invalid, logout user
        state.isAuthenticated = false;
        state.user = null;
        state.token = null;
        state.projects = [];
        state.currentProject = null;
        state.jwtPayload = null;
        SessionManager.clearSession();
      } else {
        // Session is valid, restore authentication state from localStorage
        const sessionData = SessionManager.getSession();
        if (sessionData && !state.isAuthenticated) {
          console.log("🔄 Restoring authentication state from session storage");
          state.isAuthenticated = true;
          state.user = sessionData.user;
          state.token = sessionData.token;
          state.projects = sessionData.projects || [];
          state.currentProject = sessionData.currentProject;
          state.jwtPayload = sessionData.jwtPayload || null;
          console.log("✅ Authentication state restored successfully");
        }
      }
    },
  },
  extraReducers: (builder) => {
    // Handle login success - new format with WebAuthn flow
    builder.addMatcher(authApi.endpoints.login.matchFulfilled, (state, action) => {
      // New login response only contains tenant_id, email, first_login
      // Don't authenticate user here - authentication happens after WebAuthn flow
      const { tenant_id, email, first_login } = action.payload;

      // Just log the login attempt, but don't authenticate yet
      console.log("Login successful, WebAuthn flow required:", { tenant_id, email, first_login });

      // Clear any existing auth state since this is just step 1
      state.isAuthenticated = false;
      state.user = null;
      state.token = null;
      state.projects = [];
      state.currentProject = null;
      state.jwtPayload = null;
    });

    // Handle register verify success (OTP verification)
    // Note: Don't authenticate user here - they need to login to get JWT token with full profile
    builder.addMatcher(authApi.endpoints.registerVerify.matchFulfilled, (state, action) => {
      // Just store the verification data temporarily, don't authenticate
      // User will be authenticated after login with JWT token
      const verificationData = {
        tenant_id: action.payload.tenant_id,
        project_id: action.payload.project_id,
        client_id: action.payload.client_id,
        email_id: action.payload.email_id,
        verified_at: Date.now(),
      };

      // Store verification completion but don't authenticate
      sessionStorage.setItem("authsec_verification_v2", JSON.stringify(verificationData));
    });

  },
});

export const {
  logout,
  updateToken,
  updateUser,
  setProjects,
  setCurrentProject,
  completeWebAuthnAuthentication,
  checkSession,
} = authSlice.actions;

export default authSlice.reducer;
