import React, { createContext, useContext, useEffect, useState } from "react";
import { useSelector, useDispatch } from "react-redux";
import { useLoginMutation, useRegisterInitiateMutation } from "../../app/api/authApi";
import { logout, checkSession, type AuthUser } from "../slices/authSlice";
import { setLoginData, setCurrentStep, setAuthenticationError } from "../slices/adminWebAuthnSlice";
import type { RootState } from "../../app/store";
import { toast } from "react-hot-toast";

export interface Project {
  id: string;
  name: string;
  description?: string;
  slug: string;
  ownerId: string;
  role: "owner" | "admin" | "member";
}

export interface UserProject {
  user: AuthUser;
  projects: Project[];
  currentProject: Project | null;
}

interface AuthContextType {
  user: AuthUser | null;
  currentProject: Project | null;
  projects: Project[];
  isLoading: boolean;
  isAuthenticated: boolean;
  signIn: (
    email: string,
    password: string,
    tenantDomainOverride?: string
  ) => Promise<{ success: boolean; requiresWebAuthn?: boolean; tenantId?: string; email?: string; firstLogin?: boolean }>;
  signUp: (
    email: string,
    password: string,
    firstName?: string,
    lastName?: string,
    tenantDomain?: string
  ) => Promise<{ success: boolean; email?: string }>;
  signOut: () => Promise<void>;
  createProject: (name: string, description?: string) => Promise<boolean>;
  switchProject: (projectId: string) => Promise<boolean>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | null>(null);

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error("useAuth must be used within an AuthProvider");
  }
  return context;
};


export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const dispatch = useDispatch();
  const auth = useSelector((state: RootState) => state.auth);
  const [loginMutation] = useLoginMutation();
  const [registerInitiateMutation] = useRegisterInitiateMutation();
  const [isLoading, setIsLoading] = useState(true);

  const { user, currentProject, projects, isAuthenticated } = auth;

  // Initialize auth state on mount
  useEffect(() => {
    initializeAuth();
  }, []);

  // Listen for storage changes (cross-tab sync)
  useEffect(() => {
    const handleStorageChange = (e: StorageEvent) => {
      if (e.key === "authsec_session_v2" && e.newValue !== e.oldValue) {
        // Session changed in another tab, reinitialize
        initializeAuth();
      }
    };

    window.addEventListener("storage", handleStorageChange);
    return () => window.removeEventListener("storage", handleStorageChange);
  }, []);

  // Periodic session validation
  useEffect(() => {
    if (!isAuthenticated) return;

    const validateSession = () => {
      dispatch(checkSession());
    };

    // Check session validity every 5 minutes
    const interval = setInterval(validateSession, 5 * 60 * 1000);

    // Also check when tab becomes visible (user returns to app)
    const handleVisibilityChange = () => {
      if (!document.hidden) {
        validateSession();
      }
    };

    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      clearInterval(interval);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
  }, [isAuthenticated, dispatch]);

  const initializeAuth = async () => {
    setIsLoading(true);
    try {
      // Use SessionManager to check session validity
      dispatch(checkSession());
    } catch (error) {
      console.error("Auth initialization error:", error);
      dispatch(logout());
    } finally {
      setIsLoading(false);
    }
  };

  const signIn = async (
    email: string,
    password: string,
    tenantDomainOverride?: string
  ): Promise<{ success: boolean; requiresWebAuthn?: boolean; tenantId?: string; email?: string; firstLogin?: boolean }> => {
    setIsLoading(true);
    try {
      const result = await loginMutation({ email, password, tenant_domain: tenantDomainOverride });
      
      if ('data' in result && result.data) {
        const { tenant_id, email: userEmail, first_login } = result.data;
        
        // Store WebAuthn flow data in Redux
        dispatch(setLoginData({
          tenantId: tenant_id,
          email: userEmail,
          isFirstLogin: first_login
        }));
        
        // Clear any existing authentication errors before starting WebAuthn flow
        dispatch(setAuthenticationError(null));
        
        // Route based on first login status
        if (first_login) {
          dispatch(setCurrentStep("mfa_selection"));
        } else {
          dispatch(setCurrentStep("authentication"));
        }
        
        toast.success("Login successful! Setting up authentication...");
        return {
          success: true,
          requiresWebAuthn: true,
          tenantId: tenant_id,
          email: userEmail,
          firstLogin: first_login
        };
      } else if ('error' in result) {
        const error = result.error as any;
        toast.error(error.data?.message || "Login failed");
        return { success: false };
      }
      return { success: false };
    } catch (error) {
      console.error("Sign in error:", error);
      toast.error("Failed to sign in");
      return { success: false };
    } finally {
      setIsLoading(false);
    }
  };

  const signUp = async (
    email: string,
    password: string,
    firstName?: string,
    lastName?: string,
    tenantDomain?: string
  ): Promise<{ success: boolean; email?: string }> => {
    setIsLoading(true);
    try {
      const result = await registerInitiateMutation({ 
        email, 
        password,
        first_name: firstName || "",
        last_name: lastName || "",
        tenant_domain: tenantDomain || ""
      });
      
      if ('data' in result) {
        toast.success("Verification code sent to your email!");
        return { success: true, email };
      } else if ('error' in result) {
        const error = result.error as any;
        toast.error(error.data?.message || "Registration failed");
        return { success: false };
      }
      return { success: false };
    } catch (error) {
      console.error("Sign up error:", error);
      toast.error("Failed to initiate registration");
      return { success: false };
    } finally {
      setIsLoading(false);
    }
  };


  const signOut = async (): Promise<void> => {
    setIsLoading(true);
    try {
      dispatch(logout());
      toast.success("Successfully signed out!");
    } catch (error) {
      console.error("Sign out error:", error);
      toast.error("Error signing out");
    } finally {
      setIsLoading(false);
    }
  };

  const createProject = async (name: string, description?: string): Promise<boolean> => {
    setIsLoading(true);
    try {
      // For now, project creation is not implemented with the new API
      toast.error("Project creation not yet implemented with the new API");
      return false;
    } catch (error) {
      console.error("Create project error:", error);
      return false;
    } finally {
      setIsLoading(false);
    }
  };

  const switchProject = async (projectId: string): Promise<boolean> => {
    try {
      // For now, project switching is not implemented with the new API
      toast.error("Project switching not yet implemented with the new API");
      return false;
    } catch (error) {
      console.error("Switch project error:", error);
      return false;
    }
  };

  const refreshUser = async (): Promise<void> => {
    try {
      // For now, user refresh is not implemented with the new API
      // The Redux state should handle this automatically
      console.log("User refresh - data is managed by Redux state");
    } catch (error) {
      console.error("Refresh user error:", error);
    }
  };

  const value: AuthContextType = {
    user,
    currentProject,
    projects,
    isLoading,
    isAuthenticated,
    signIn,
    signUp,
    signOut,
    createProject,
    switchProject,
    refreshUser,
  };

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
};
