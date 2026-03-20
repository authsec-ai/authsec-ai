import React from "react";
import { useNavigate } from "react-router-dom";
import { AdminWebAuthnRouter } from "../adminauth/AdminWebAuthnRouter";
import { toast } from "react-hot-toast";
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";

/**
 * Admin WebAuthn Page
 * 
 * Handles the complete admin WebAuthn authentication flow after initial login.
 * Uses AdminWebAuthnRouter to route users through MFA selection, setup, and authentication.
 */
export function WebAuthnPage() {
  const navigate = useNavigate();

  const handleAuthComplete = () => {
    // Handle successful admin WebAuthn completion
    console.log("🎉 Admin WebAuthn authentication completed, redirecting to dashboard");
    console.log("🔑 JWT token in localStorage:", localStorage.getItem('jwt_token'));
    
    toast.success("Authentication completed successfully!");
    
    // Navigate to dashboard after successful authentication
    setTimeout(() => {
      console.log("🚀 Attempting to navigate to dashboard...");
      navigate("/dashboard", { replace: true });
      console.log("✅ navigate() call completed");
    }, 500);
  };

  const handleAuthError = (error: string) => {
    // Handle admin WebAuthn errors
    console.error("❌ Admin WebAuthn authentication error:", error);
    toast.error(`Authentication failed: ${error}`);
    
    // Wait a moment before redirecting to prevent jarring transitions
    setTimeout(() => {
      console.log("🔄 Redirecting to login due to auth error");
      if (error.includes("invalid") || error.includes("expired")) {
        toast.error("Your session has expired. Please sign in again.");
      }
      navigate("/admin/login", { replace: true });
    }, 1500);
  };

  return (
    <AuthSplitFrame
      valuePanel={
        <AuthValuePanel
          eyebrow="Adaptive MFA"
          title="Complete secure sign-in."
          subtitle="Use a biometric authenticator or one-time code to finish identity verification."
          points={[
            "Security keys and platform biometrics supported.",
            "Fallback authenticator app flow available.",
            "Verification state is tied to your tenant session.",
          ]}
        />
      }
    >
      <div className="w-full">
        <AdminWebAuthnRouter 
          onAuthComplete={handleAuthComplete}
          onAuthError={handleAuthError}
        />
      </div>
    </AuthSplitFrame>
  );
}
