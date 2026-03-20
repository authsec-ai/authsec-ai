import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useDispatch, useSelector } from "react-redux";
import { motion } from "framer-motion";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Separator } from "../../components/ui/separator";
import {
  IconBrandGithub,
  IconBrandLinkedin,
  IconKey,
  IconCircleCheck,
  IconAlertTriangle,
  IconMail,
  IconEye,
  IconEyeOff,
  IconArrowLeft,
} from "@tabler/icons-react";
import { Loader2, ArrowRight } from "lucide-react";
import {
  useLazyGetLoginPageDataQuery,
  useInitiateAuthMutation,
  useCheckCustomLoginStatusMutation,
  useRegisterCustomUserMutation,
  useCompleteCustomUserRegistrationMutation,
  useSamlLoginMutation,
  type LoginPageData,
  type OIDCProvider,
} from "../../app/api/oidcApi";
import {
  setCurrentStep,
  setLoginData,
  setClientId,
} from "../slices/oidcWebAuthnSlice";
import type { RootState } from "../../app/store";
import { OIDCWebAuthnRouter } from "./OIDCWebAuthnRouter";
// OIDC WebAuthn functionality using dedicated OIDC context
import {
  EndUserAuthProvider,
  useEndUserAuth,
} from "../context/EndUserAuthContext";
import {
  useForgotPasswordMutation,
  useForgotPasswordVerifyOtpMutation,
  useForgotPasswordResetMutation,
} from "../../app/api/authApi";
import { useCustomLoginMutation } from "../../app/api/userAuthApi";
import { useTheme } from "next-themes";
import config from '../../config';
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

const OIDCLoginPageInner: React.FC = () => {
  const dispatch = useDispatch();
  const { executeCallback } = useEndUserAuth();

  // Force light theme on OIDC login page
  const { setTheme } = useTheme();
  useEffect(() => {
    setTheme("light");
  }, [setTheme]);

  // Get client_id from Redux state instead of local state
  const clientId = useSelector(
    (state: RootState) => state.oidcWebAuthn.clientId
  );
  const [loginPageData, setLoginPageData] = useState<LoginPageData | null>(
    null
  );
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [authenticating, setAuthenticating] = useState<string | null>(null);

  // RTK Query hooks
  const [customLogin] = useCustomLoginMutation();
  const [getLoginPageData] = useLazyGetLoginPageDataQuery();
  const [initiateAuth] = useInitiateAuthMutation();
  const [checkCustomLoginStatus] = useCheckCustomLoginStatusMutation();
  const [registerCustomUser] = useRegisterCustomUserMutation();
  const [completeCustomUserRegistration] =
    useCompleteCustomUserRegistrationMutation();

  // Custom login state
  const [showCustomLogin, setShowCustomLogin] = useState(false);
  const [email, setEmail] = useState("");
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [userExists, setUserExists] = useState<boolean | null>(null);
  const [showPassword, setShowPassword] = useState(false);
  const [showConfirmPassword, setShowConfirmPassword] = useState(false);
  const [customAuthenticating, setCustomAuthenticating] = useState(false);
  const [emailSubmitted, setEmailSubmitted] = useState(false);
  const [registrationStep, setRegistrationStep] = useState<"details" | "otp">(
    "details"
  );
  const [registrationOtp, setRegistrationOtp] = useState("");
  const [registrationMessage, setRegistrationMessage] = useState<string | null>(
    null
  );
  const tenantDomain =
    typeof window !== "undefined" ? window.location.hostname : undefined;

  // WebAuthn state
  const [status, setStatus] = useState<
    "idle" | "processing" | "webauthn" | "success" | "error"
  >("idle");
  const [webauthnData, setWebauthnData] = useState<{
    tenantId: string;
    email: string;
    firstLogin: boolean;
  } | null>(null);

  // Forgot Password state
  const [showForgotPassword, setShowForgotPassword] = useState(false);
  const [forgotPasswordStep, setForgotPasswordStep] = useState<
    "email" | "otp" | "reset" | "success"
  >("email");
  const [forgotPasswordEmail, setForgotPasswordEmail] = useState("");
  const [otp, setOtp] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmNewPassword, setConfirmNewPassword] = useState("");
  const [showNewPassword, setShowNewPassword] = useState(false);
  const [showConfirmNewPassword, setShowConfirmNewPassword] = useState(false);
  const [forgotLoading, setForgotLoading] = useState(false);
  const [forgotError, setForgotError] = useState<string | null>(null);
  const [forgotSuccess, setForgotSuccess] = useState<string | null>(null);

  // API mutations
  const [forgotPassword] = useForgotPasswordMutation();
  const [forgotPasswordVerifyOtp] = useForgotPasswordVerifyOtpMutation();
  const [forgotPasswordReset] = useForgotPasswordResetMutation();
  const activeProviders = useMemo(
    () =>
      (loginPageData?.providers || []).filter(
        (provider) =>
          provider.is_active &&
          provider.provider_name?.toLowerCase() !== "authsec"
      ),
    [loginPageData]
  );
  const sortedProviders = useMemo(
    () => activeProviders.slice().sort((a, b) => a.sort_order - b.sort_order),
    [activeProviders]
  );

  // Forgot Password handlers
  const handleOpenForgotPassword = () => {
    setShowForgotPassword(true);
    setForgotPasswordStep("email");
    setForgotPasswordEmail(email || "");
    setOtp("");
    setNewPassword("");
    setConfirmNewPassword("");
    setForgotError(null);
    setForgotSuccess(null);
  };

  const handleCloseForgotPassword = () => {
    setShowForgotPassword(false);
    setForgotPasswordStep("email");
    setOtp("");
    setNewPassword("");
    setConfirmNewPassword("");
    setForgotError(null);
    setForgotSuccess(null);
  };

  const handleForgotSendOtp = async () => {
    if (!forgotPasswordEmail || !forgotPasswordEmail.includes("@")) {
      setForgotError("Please enter a valid email address");
      return;
    }
    if (!clientId) {
      setForgotError("Client ID not available. Please refresh the page.");
      return;
    }
    setForgotLoading(true);
    setForgotError(null);
    setForgotSuccess(null);
    try {
      await forgotPassword({
        email: forgotPasswordEmail,
        client_id: clientId,
      }).unwrap();
      setForgotPasswordStep("otp");
      setForgotSuccess("If your email is registered, an OTP has been sent.");
    } catch (err: any) {
      const message = err?.data?.message || "Failed to send OTP";
      setForgotError(message);
    } finally {
      setForgotLoading(false);
    }
  };

  const handleForgotVerifyOtp = async () => {
    if (!otp || otp.trim().length < 4) {
      setForgotError("Please enter the OTP");
      return;
    }
    setForgotLoading(true);
    setForgotError(null);
    setForgotSuccess(null);
    try {
      await forgotPasswordVerifyOtp({
        email: forgotPasswordEmail,
        otp: otp.trim(),
      }).unwrap();
      setForgotPasswordStep("reset");
      setForgotSuccess("OTP verified. You can now reset your password.");
    } catch (err: any) {
      const message = err?.data?.message || "Invalid OTP. Please try again.";
      setForgotError(message);
    } finally {
      setForgotLoading(false);
    }
  };

  const handleForgotResetPassword = async () => {
    if (!newPassword || !confirmNewPassword) {
      setForgotError("Please fill in all fields");
      return;
    }
    if (newPassword !== confirmNewPassword) {
      setForgotError("Passwords do not match");
      return;
    }
    if (newPassword.length < 8) {
      setForgotError("Password must be at least 8 characters long");
      return;
    }
    if (!clientId) {
      setForgotError("Client ID not available. Please refresh the page.");
      return;
    }
    setForgotLoading(true);
    setForgotError(null);
    setForgotSuccess(null);
    try {
      await forgotPasswordReset({
        email: forgotPasswordEmail,
        new_password: newPassword,
        client_id: clientId,
      }).unwrap();
      setForgotPasswordStep("success");
      setForgotSuccess("Password reset successfully.");
    } catch (err: any) {
      const message = err?.data?.message || "Failed to reset password";
      setForgotError(message);
    } finally {
      setForgotLoading(false);
    }
  };

  const rawSearch = typeof window !== "undefined" ? window.location.search : "";
  const urlParams = new URLSearchParams(rawSearch);
  const loginChallenge = urlParams.get("login_challenge");

  // Extract SAML-related parameters from URL (if present after SAML callback)
  const samlClientId = urlParams.get("client_id");
  const samlUserEmail = urlParams.get("user_email");
  const samlTenantId = urlParams.get("tenant_id");
  const samlUserId = urlParams.get("user_id");
  const samlProvider = urlParams.get("provider");
  const samlProviderId = urlParams.get("provider_id");
  const samlProjectId = urlParams.get("project_id");
  const samlSuccess = urlParams.get("success");

  // Handle SAML callback immediately when parameters are detected
  const [samlLogin] = useSamlLoginMutation();

  useEffect(() => {
    const handleSamlCallback = async () => {
      if (samlClientId && samlUserEmail && samlSuccess === "true") {
        console.log(
          "🔐 SAML callback detected! Calling SAML login API immediately:",
          {
            client_id: samlClientId,
            user_email: samlUserEmail,
            tenant_id: samlTenantId,
            provider: samlProvider,
          }
        );

        try {
          // Call SAML login API to check if WebAuthn is needed
          const samlLoginResponse = await samlLogin({
            client_id: samlClientId,
            email: samlUserEmail,
          }).unwrap();

          console.log("✅ SAML login check response:", samlLoginResponse);

          // Check if WebAuthn is needed based on first_login
          if (samlLoginResponse.first_login !== undefined) {
            // Set up WebAuthn flow data
            const webauthnFlowData = {
              tenantId: samlLoginResponse.tenant_id,
              email: samlLoginResponse.email,
              firstLogin: samlLoginResponse.first_login,
            };

            console.log(
              "🔒 WebAuthn required for SAML user, initiating flow..."
            );

            // Initialize OIDC WebAuthn flow in Redux
            dispatch(
              setLoginData({
                tenantId: webauthnFlowData.tenantId,
                email: webauthnFlowData.email,
                isFirstLogin: webauthnFlowData.firstLogin,
                clientId: samlClientId,
              })
            );

            // Set appropriate WebAuthn step
            if (webauthnFlowData.firstLogin) {
              console.log("🆕 First-time SAML user → MFA setup");
              dispatch(setCurrentStep("mfa_selection"));
            } else {
              console.log("🔑 Returning SAML user → WebAuthn authentication");
              dispatch(setCurrentStep("authentication"));
            }

            // Switch UI into WebAuthn router
            setWebauthnData(webauthnFlowData);
            setStatus("webauthn");

            // Store SAML parameters for use after WebAuthn completion
            sessionStorage.setItem("saml_post_webauthn", "true");
            sessionStorage.setItem("saml_client_id", samlClientId);
            sessionStorage.setItem("saml_user_email", samlUserEmail);
            if (loginChallenge)
              sessionStorage.setItem("saml_login_challenge", loginChallenge);
            if (samlTenantId)
              sessionStorage.setItem("saml_tenant_id", samlTenantId);
            if (samlUserId) sessionStorage.setItem("saml_user_id", samlUserId);
            if (samlProvider)
              sessionStorage.setItem("saml_provider", samlProvider);
            if (samlProviderId)
              sessionStorage.setItem("saml_provider_id", samlProviderId);
            if (samlProjectId)
              sessionStorage.setItem("saml_project_id", samlProjectId);
          } else {
            console.log("⚠️ SAML login response missing first_login field");
          }
        } catch (error) {
          console.error("❌ SAML login check failed:", error);
          setError("Failed to verify SAML authentication. Please try again.");
        }
      }
    };

    handleSamlCallback();
  }, [
    samlClientId,
    samlUserEmail,
    samlSuccess,
    samlTenantId,
    samlUserId,
    samlProvider,
    samlProviderId,
    samlProjectId,
    samlLogin,
    dispatch,
  ]);

  // Handle SAML + WebAuthn completion redirect
  const samlWebAuthnComplete = urlParams.get("saml_webauthn_complete");
  useEffect(() => {
    const handleSamlWebAuthnCompletion = async () => {
      if (samlWebAuthnComplete === "true" && loginChallenge) {
        console.log(
          "✅ SAML + WebAuthn completion detected, finalizing OAuth flow..."
        );
        setStatus("processing");
        setError("Authentication completed! Finalizing login...");

        // At this point, the user has completed SAML authentication + WebAuthn
        // The backend should have a valid session with the WebAuthn token
        // We need to finalize the OAuth flow by redirecting to the backend endpoint
        // that will issue the OAuth code and redirect to Hydra

        // Option 1: Redirect to a SAML finalization endpoint
        // Option 2: Let the normal fetchLoginData handle it and auto-trigger provider selection
        // For now, we'll show a message and let fetchLoginData proceed normally
        // The backend should detect the authenticated session and handle accordingly

        console.log(
          "🔄 SAML WebAuthn flow completed, login page will load normally"
        );
      }
    };

    handleSamlWebAuthnCompletion();
  }, [samlWebAuthnComplete, loginChallenge]);

  const fetchLoginData = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);

      if (!loginChallenge) {
        console.error("❌ Missing login_challenge parameter");
        setError(
          "Missing login_challenge parameter. This page should be accessed via OAuth flow."
        );
        setLoading(false);
        return;
      }

      console.log("🔍 Fetching login data with challenge:", {
        login_challenge: loginChallenge.substring(0, 20) + "...",
        full_url: window.location.href,
        has_saml_params: !!(
          urlParams.get("client_id") || urlParams.get("user_email")
        ),
        saml_webauthn_complete: urlParams.get("saml_webauthn_complete"),
      });

      const data = await getLoginPageData({
        login_challenge: loginChallenge,
        extraQuery: rawSearch, // ✅ keep all other params
      }).unwrap();

      console.log("📥 Login data response:", {
        success: data.success,
        has_providers: !!data.providers,
        client_id: data.client_id,
        error: (data as any)?.error,
      });

      const ok = (data as any)?.success !== false;
      if (ok) {
        setLoginPageData(data);
        if (data.client_id) dispatch(setClientId(data.client_id));
      } else {
        console.error("❌ Login data fetch failed:", (data as any)?.error);
        setError((data as any)?.error || "Failed to load login data");
      }
    } catch (err) {
      console.error("❌ Login data fetch exception:", err);
      setError(
        err instanceof Error
          ? err.message
          : "Failed to load authentication providers"
      );
    } finally {
      setLoading(false);
    }
  }, [loginChallenge, rawSearch, getLoginPageData, dispatch]);

  // Call fetchLoginData on component mount
  useEffect(() => {
    fetchLoginData();
  }, [fetchLoginData]);

  // Step 3: Handle provider selection and initiate OAuth
  const handleProviderAuth = async (provider: OIDCProvider) => {
    if (!loginChallenge) return;

    try {
      setAuthenticating(provider.provider_name);
      setError(null);

      const isSaml =
        String((provider as any)?.config?.type || "").toLowerCase() ===
          "saml" || provider.provider_name.toLowerCase().includes("saml");

      // Build extraQuery: all current params except login_challenge
      const rawSearch = window.location.search || "";
      const extraQS = new URLSearchParams(rawSearch.replace(/^\?/, ""));
      extraQS.delete("login_challenge");
      const extraQuery = extraQS.toString(); // pass-through params

      const authResponse = await initiateAuth({
        provider: provider.provider_name.toLowerCase(),
        login_challenge: loginChallenge!,
        isSaml,
        extraQuery,
      }).unwrap();

      // store state/provider for callback verification
      sessionStorage.setItem("oauth_state", authResponse.state || "");
      sessionStorage.setItem(
        "oauth_provider",
        authResponse.provider || provider.provider_name
      );
      sessionStorage.setItem("login_challenge", loginChallenge!);

      // ------ SAML handling ------
      if (isSaml && (authResponse.sso_url || authResponse.auth_url)) {
        const target = authResponse.sso_url || authResponse.auth_url;

        // If backend indicates POST binding or you received SAMLRequest payload,
        // auto-submit a form in the SAME PAGE.
        if (authResponse.method === "POST" || authResponse.form_params) {
          const form = document.createElement("form");
          form.method = "POST";
          form.action = target;
          form.style.display = "none";

          const params = authResponse.form_params || {};
          Object.entries(params).forEach(([name, value]) => {
            const input = document.createElement("input");
            input.type = "hidden";
            input.name = name;
            input.value = value ?? "";
            form.appendChild(input);
          });

          document.body.appendChild(form);
          form.submit(); // same-tab POST redirect
          return;
        }

        // Default GET redirect in the SAME TAB
        window.location.replace(target); // or window.location.assign(target)
        return;
      }

      // ------ OIDC/other providers ------
      if (authResponse.auth_url) {
        window.location.replace(authResponse.auth_url); // same-tab
      } else {
        setError(
          (authResponse as any)?.error || "Failed to initiate authentication"
        );
        setAuthenticating(null);
      }
    } catch (err) {
      setError(
        err instanceof Error
          ? err.message
          : "Failed to start authentication process"
      );
      setAuthenticating(null);
    }
  };

  // Custom login handlers
  const handleEmailSubmit = async () => {
    if (!email || !email.includes("@")) {
      setError("Please enter a valid email address");
      return;
    }

    if (!clientId) {
      setError("Client ID not available. Please refresh the page.");
      return;
    }

    setCustomAuthenticating(true);

    try {
      const statusResponse = await checkCustomLoginStatus({
        client_id: clientId,
        email,
        ...(tenantDomain ? { tenant_domain: tenantDomain } : {}),
      }).unwrap();

      // Treat presence of a response field as authoritative
      if (typeof statusResponse.response !== "undefined") {
        setUserExists(
          statusResponse.response === "true" || statusResponse.response === true
        );
        setEmailSubmitted(true);
        setError(null);
        setRegistrationStep("details");
        setRegistrationOtp("");
        setRegistrationMessage(null);
      } else {
        setError(
          (statusResponse as any)?.error || "Failed to check user status"
        );
      }
    } catch (error) {
      console.error("User status check failed:", error);
      setError("Failed to check user status");
    } finally {
      setCustomAuthenticating(false);
    }
  };

  const handlePasswordSubmit = async () => {
    if (!password) {
      setError("Please enter your password");
      return;
    }

    if (!clientId) {
      setError("Client ID not available. Please refresh the page.");
      return;
    }

    setCustomAuthenticating(true);

    try {
      // Call the existing handleCustomLoginSuccess which uses customLogin
      await handleCustomLoginSuccess(clientId, email, password);
      setError(null);
    } catch (error) {
      console.error("Custom login failed:", error);
      setError("Login failed");
    } finally {
      setCustomAuthenticating(false);
    }
  };

  const handleSetupSubmit = async () => {
    if (!password || !confirmPassword || !name.trim()) {
      // UPDATED
      setError("Please fill in all fields");
      return;
    }

    if (name.trim().length < 2) {
      // NEW
      setError("Please enter your name");
      return;
    }

    if (password !== confirmPassword) {
      setError("Passwords do not match");
      return;
    }

    if (password.length < 8) {
      setError("Password must be at least 8 characters long");
      return;
    }

    if (!clientId) {
      setError("Client ID not available. Please refresh the page.");
      return;
    }

    setCustomAuthenticating(true);
    try {
      const registerResponse = await registerCustomUser({
        client_id: clientId,
        name: name.trim(),
        email,
        password,
        ...(tenantDomain ? { tenant_domain: tenantDomain } : {}),
      }).unwrap();

      if (!registerResponse?.error && registerResponse?.success) {
        setError(null);
        setRegistrationMessage(
          registerResponse?.message ||
            "Registration initiated. Please enter the OTP we sent to your email."
        );
        setRegistrationStep("otp");
        setRegistrationOtp("");
      } else {
        setError(
          registerResponse?.error ||
            registerResponse?.message ||
            "Registration failed"
        );
      }
    } catch (error) {
      console.error("Registration failed:", error);
      setError("Registration failed");
    } finally {
      setCustomAuthenticating(false);
    }
  };

  const handleRegistrationOtpSubmit = async () => {
    if (!registrationOtp.trim()) {
      setError("Please enter the OTP sent to your email");
      return;
    }

    if (!clientId) {
      setError("Client ID not available. Please refresh the page.");
      return;
    }

    if (!password) {
      setError("Password missing. Please go back and re-enter your password.");
      setRegistrationStep("details");
      return;
    }

    setCustomAuthenticating(true);
    try {
      const verifyResponse = await completeCustomUserRegistration({
        client_id: clientId,
        email,
        otp: registrationOtp.trim(),
      }).unwrap();

      if (!verifyResponse?.error && verifyResponse?.success) {
        setError(null);
        setRegistrationMessage(
          verifyResponse?.message || "Registration verified. Completing sign-in..."
        );
        await handleCustomLoginSuccess(clientId, email, password);
        setRegistrationStep("details");
        setRegistrationOtp("");
      } else {
        setError(
          verifyResponse?.error ||
            verifyResponse?.message ||
            "Failed to verify OTP"
        );
      }
    } catch (error) {
      console.error("OTP verification failed:", error);
      setError("Failed to verify OTP");
    } finally {
      setCustomAuthenticating(false);
    }
  };

  // Handle successful custom login/registration and initiate WebAuthn flow if needed
  const handleCustomLoginSuccess = async (
    clientId: string,
    email: string,
    password: string
  ) => {
    try {
      // Call the custom login API endpoint using RTK Query
      const result = await customLogin({
        client_id: clientId,
        email,
        password,
        ...(tenantDomain ? { tenant_domain: tenantDomain } : {}),
      }).unwrap();

      // Your API returns data directly: {tenant_id, email, first_login, otp_required, mfa_required}
      const userData = result;

      // Check if we have the expected fields
      if (
        userData &&
        userData.tenant_id &&
        userData.email !== undefined &&
        userData.first_login !== undefined
      ) {
        // If first_login is false, we need to initiate the WebAuthn flow
        if (!userData.first_login) {
          // Set WebAuthn data
          const webauthnFlowData = {
            tenantId: userData.tenant_id,
            email: userData.email,
            firstLogin: userData.first_login,
          };

          // Initialize OIDC WebAuthn flow in Redux first to provide context data
          dispatch(
            setLoginData({
              tenantId: webauthnFlowData.tenantId,
              email: webauthnFlowData.email,
              isFirstLogin: webauthnFlowData.firstLogin,
              clientId, // Include client_id in Redux state
            })
          );

          // Since first_login is false, go directly to authentication step
          dispatch(setCurrentStep("authentication"));

          // Now switch UI into WebAuthn router
          setWebauthnData(webauthnFlowData);
          setStatus("webauthn");
        } else {
          // First-time login - initiate MFA setup flow
          const webauthnFlowData = {
            tenantId: userData.tenant_id,
            email: userData.email,
            firstLogin: true,
          };

          // Initialize OIDC WebAuthn flow for first-time user first
          dispatch(
            setLoginData({
              tenantId: webauthnFlowData.tenantId,
              email: webauthnFlowData.email,
              isFirstLogin: webauthnFlowData.firstLogin,
              clientId, // Include client_id in Redux state
            })
          );

          // First-time login - go to MFA selection step
          dispatch(setCurrentStep("mfa_selection"));

          // Now switch UI into WebAuthn router
          setWebauthnData(webauthnFlowData);
          setStatus("webauthn");
        }
      } else {
        // If we don't have the expected structure, log it and show error
        console.error("Invalid custom login response structure:", result);
        console.error(
          "Expected: {tenant_id, email, first_login, otp_required, mfa_required}"
        );
        setError("Invalid response from login API. Please try again.");
      }
    } catch (error) {
      console.error("WebAuthn flow initiation failed:", error);
      setError(
        "Login successful but failed to initiate security flow. Please try again."
      );
    }
  };

  const handleBackToEmail = () => {
    setEmailSubmitted(false);
    setUserExists(null);
    setPassword("");
    setName("");
    setConfirmPassword("");
    setError(null);
    setRegistrationStep("details");
    setRegistrationOtp("");
    setRegistrationMessage(null);
  };

  const handleToggleCustomLogin = () => {
    setShowCustomLogin(!showCustomLogin);
    setError(null);
    // Reset custom login state
    setEmail("");
    setPassword("");
    setName("");
    setConfirmPassword("");
    setUserExists(null);
    setEmailSubmitted(false);
    setStatus("idle");
    setWebauthnData(null);
    setRegistrationStep("details");
    setRegistrationOtp("");
    setRegistrationMessage(null);
  };

  // Handle WebAuthn completion (redirect to OIDCCallbackPage with token)
  const handleWebAuthnComplete = async () => {
    try {
      // Use the integrated callback handler to prevent race conditions
      const result = await executeCallback(
        webauthnData?.email || email,
        webauthnData?.tenantId
      );

      if (result.success && result.token) {
        // Store the WebAuthn callback token in sessionStorage so OIDCCallbackPage can display it
        sessionStorage.setItem("webauthn_callback_token", result.token);
        sessionStorage.setItem(
          "webauthn_callback_email",
          webauthnData?.email || email
        );

        // Redirect to OIDCCallbackPage with a special parameter to indicate WebAuthn completion
        const callbackUrl = `/oidc/auth/callback?webauthn_complete=true`;
        window.location.href = callbackUrl;
      } else {
        setStatus("error");
        setError(
          result.error || "webauthn-callback failed without specific error"
        );
        console.error(
          "❌ Integrated WebAuthn callback handler failed:",
          result.error
        );
      }
    } catch (error) {
      setStatus("error");
      setError(
        `WebAuthn callback handler failed: ${
          error instanceof Error ? error.message : "Unknown error"
        }`
      );
      console.error("❌ Integrated WebAuthn callback handler error:", error);
    }
  };

  // Handle WebAuthn error
  const handleWebAuthnError = (error: string) => {
    setStatus("error");
    setError(`WebAuthn authentication failed: ${error}`);
  };

  const getProviderIcon = (providerName: string) => {
    const iconProps = { size: 20, className: "text-current" };

    switch (providerName.toLowerCase()) {
      case "github":
        return <IconBrandGithub {...iconProps} />;
      case "google":
        return (
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            className="text-current"
          >
            <path
              fill="currentColor"
              d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
            />
            <path
              fill="currentColor"
              d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
            />
            <path
              fill="currentColor"
              d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
            />
            <path
              fill="currentColor"
              d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
            />
          </svg>
        );
      case "microsoft":
        return (
          <svg
            width="20"
            height="20"
            viewBox="0 0 24 24"
            className="text-current"
          >
            <path
              fill="currentColor"
              d="M11.4 24H0V12.6h11.4V24zM24 24H12.6V12.6H24V24zM11.4 11.4H0V0h11.4v11.4zm12.6 0H12.6V0H24v11.4z"
            />
          </svg>
        );
      case "linkedin":
        return <IconBrandLinkedin {...iconProps} />;
      default:
        return <IconKey {...iconProps} />;
    }
  };

  // Removed provider-specific colors - using clean, consistent hover state like admin login

  if (loading) {
    return (
      <AuthSplitFrame
        valuePanel={
          <AuthValuePanel
            eyebrow="End-user Authentication"
            title="Secure sign-in is loading."
            subtitle="We are preparing provider and challenge details for this OIDC transaction."
            points={[
              "Provider catalog and tenant settings are being validated.",
              "Challenge context remains bound to your current session.",
            ]}
          />
        }
      >
        <AuthActionPanel className="space-y-6">
          <AuthStepHeader
            title="Preparing sign-in"
            subtitle="Loading authentication options..."
          />
          <div className="space-y-3 text-center auth-inline-note">
            <motion.div
              animate={{ rotate: 360 }}
              transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
              className="mx-auto h-10 w-10 rounded-full border-2 border-slate-300 border-t-slate-700"
            />
            <p className="text-sm text-slate-600">
              Loading authentication options...
            </p>
            {loginChallenge && (
              <p className="text-xs text-slate-500">
                Challenge: {loginChallenge.substring(0, 10)}...
              </p>
            )}
          </div>
        </AuthActionPanel>
      </AuthSplitFrame>
    );
  }

  if (error) {
    return (
      <AuthSplitFrame
        valuePanel={
          <AuthValuePanel
            eyebrow="End-user Authentication"
            title="Sign-in options could not be loaded."
            subtitle="The authentication flow hit an error before method selection."
            points={[
              "Retry to reload tenant and provider configuration.",
              "Return to the login route if the challenge expired.",
            ]}
          />
        }
      >
        <AuthActionPanel className="space-y-6">
          <AuthStepHeader
            title="Authentication error"
            subtitle="We couldn't load your sign-in options."
          />
          <div className="space-y-4 text-center">
            <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-red-50">
              <IconAlertTriangle className="h-5 w-5 text-red-600" />
            </div>
            <p className="text-sm text-red-700">{error}</p>
            <div className="space-y-2">
              <Button onClick={fetchLoginData} className="w-full">
                Retry
              </Button>
              <Button
                variant="outline"
                onClick={() => window.history.back()}
                className="w-full"
              >
                Go Back
              </Button>
            </div>

            {import.meta.env.DEV && (
              <details className="auth-inline-note text-left">
                <summary className="cursor-pointer text-sm text-slate-600 hover:text-slate-900">
                  Debug Information
                </summary>
                <div className="mt-2 space-y-1 text-xs text-slate-600">
                  <div>Challenge: {loginChallenge}</div>
                  <div>API URL: {config.VITE_API_URL}</div>
                  <div>Error: {error}</div>
                </div>
              </details>
            )}
          </div>
        </AuthActionPanel>
      </AuthSplitFrame>
    );
  }

  if (!loginPageData) return null;

  // If WebAuthn flow is active, render the WebAuthn Router instead
  if (status === "webauthn" && webauthnData) {
    return (
      <OIDCWebAuthnRouter
        onAuthComplete={handleWebAuthnComplete}
        onAuthError={handleWebAuthnError}
      />
    );
  }

  return (
    <AuthSplitFrame
      valuePanel={
        <AuthValuePanel
          eyebrow="End-user Authentication"
          title="Deliver secure sign-in to every customer."
          subtitle="Unified social login, password-based access, registration, and MFA verification for your OIDC clients."
          points={[
            "Supports provider-based and custom email/password login.",
            "Built-in registration, OTP confirmation, and password recovery.",
            "Routes users into WebAuthn or TOTP when required.",
            "Preserves challenge context through callback handoff.",
          ]}
          trustLabel={loginPageData.tenant_name ? `Tenant: ${loginPageData.tenant_name}` : undefined}
        />
      }
    >
      <AuthActionPanel className="space-y-6">
        <AuthStepHeader
          title="Sign in"
          subtitle={
            loginPageData.tenant_name
              ? `Tenant: ${loginPageData.tenant_name}`
              : "Choose a method to continue."
          }
        />
        <motion.div
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.15, duration: 0.35 }}
          className="space-y-6"
        >
            {!showCustomLogin ? (
              <>
                {/* Title Section */}
                <div className="space-y-1">
                  <h2 className="text-lg font-semibold text-slate-900">
                    Sign In
                  </h2>
                  <p className="text-sm text-slate-600">
                    Pick a method to continue
                  </p>
                </div>

                {sortedProviders.length > 0 && (
                  <>
                    <div className="space-y-3">
                      {sortedProviders.map((provider) => {
                        const isLoading =
                          authenticating === provider.provider_name;

                        return (
                          <Button
                            key={provider.provider_name}
                            onClick={() => handleProviderAuth(provider)}
                            disabled={authenticating !== null}
                            variant="outline"
                            size="lg"
                            className="h-12 w-full justify-center gap-3 rounded-xl border-slate-300 bg-white font-medium text-slate-700 shadow-sm transition-all hover:bg-slate-50 hover:shadow-md dark:border-slate-700 dark:bg-slate-900/50 dark:text-slate-200 dark:hover:bg-slate-800/50"
                            title={`Sign in with ${provider.display_name}`}
                          >
                            {isLoading ? (
                              <>
                                <Loader2 className="h-5 w-5 animate-spin" />
                                Connecting...
                              </>
                            ) : (
                              <>
                                <span className="flex h-5 w-5 items-center justify-center">
                                  {getProviderIcon(provider.provider_name)}
                                </span>
                                {provider.display_name}
                              </>
                            )}
                          </Button>
                        );
                      })}
                    </div>

                    <Separator className="my-4" />
                  </>
                )}

                <form
                  className="space-y-4"
                  onSubmit={(e) => {
                    e.preventDefault();
                    setShowCustomLogin(true);
                    handleEmailSubmit();
                  }}
                >
                  <div className="space-y-2">
                    <Label
                      htmlFor="inline-email"
                      className="text-sm font-semibold text-slate-700 dark:text-slate-300"
                    >
                      Email
                    </Label>
                    <div className="relative">
                      <Input
                        id="inline-email"
                        type="email"
                        required
                        value={email}
                        onChange={(e) => setEmail(e.target.value)}
                        className="h-12 rounded-xl border-slate-300 bg-white pr-14 text-base shadow-sm transition-all focus:border-blue-500 focus:ring-2 focus:ring-blue-500/20 dark:border-slate-700 dark:bg-slate-900/50 dark:focus:border-blue-500"
                        placeholder="you@company.com"
                      />
                      <button
                        type="submit"
                        disabled={customAuthenticating}
                        className="absolute right-2 top-1/2 flex h-8 w-8 -translate-y-1/2 items-center justify-center rounded-lg bg-blue-600 text-white transition-all hover:bg-blue-700 disabled:opacity-50 dark:bg-blue-600 dark:hover:bg-blue-700"
                        aria-label="Continue"
                      >
                        {customAuthenticating ? (
                          <Loader2 className="h-4 w-4 animate-spin" />
                        ) : (
                          <ArrowRight className="h-4 w-4" />
                        )}
                      </button>
                    </div>
                  </div>
                </form>
              </>
            ) : (
              <>
                <div className="space-y-4">
                  <div className="flex items-center justify-between">
                    <p className="text-sm font-medium text-slate-700 dark:text-slate-300">
                      {!emailSubmitted
                        ? "Enter your email address"
                        : userExists
                        ? "Welcome back!"
                        : "Create your account"}
                    </p>
                    <Button
                      variant="ghost"
                      size="icon"
                      onClick={handleToggleCustomLogin}
                      className="text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300 h-8 w-8"
                    >
                      <IconArrowLeft className="h-4 w-4" />
                    </Button>
                  </div>

                  {!emailSubmitted ? (
                    <>
                      <div className="space-y-2">
                        <Label
                          htmlFor="email"
                          className="text-sm font-medium text-slate-700 dark:text-slate-300"
                        >
                          Email Address
                        </Label>
                        <Input
                          id="email"
                          type="email"
                          value={email}
                          onChange={(e) => setEmail(e.target.value)}
                          placeholder="Enter your email address"
                          className="h-10 rounded-xl border-slate-300 shadow-sm dark:border-slate-700"
                          onKeyDown={(e) =>
                            e.key === "Enter" && handleEmailSubmit()
                          }
                        />
                      </div>
                      <Button
                        onClick={handleEmailSubmit}
                        className="h-10 w-full rounded-xl"
                        disabled={!email || customAuthenticating}
                      >
                        {customAuthenticating ? (
                          <motion.div
                            animate={{ rotate: 360 }}
                            transition={{
                              duration: 1,
                              repeat: Infinity,
                              ease: "linear",
                            }}
                            className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                          />
                        ) : null}
                        Next
                      </Button>
                    </>
                  ) : (
                    <>
                      <div className="rounded-lg bg-slate-50 p-3 dark:bg-slate-800/50">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center">
                            <IconMail className="mr-2 h-4 w-4 text-slate-500" />
                            <span className="text-sm text-slate-700 dark:text-slate-300">
                              {email}
                            </span>
                          </div>
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={handleBackToEmail}
                            className="h-8 px-2 text-slate-500 hover:text-slate-700 dark:text-slate-400 dark:hover:text-slate-300"
                          >
                            Change
                          </Button>
                        </div>
                      </div>

                      {userExists ? (
                        <>
                          {!showForgotPassword ? (
                            <>
                              <div className="space-y-2">
                                <Label
                                  htmlFor="password"
                                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                >
                                  Password
                                </Label>
                                <div className="relative">
                                  <Input
                                    id="password"
                                    type={showPassword ? "text" : "password"}
                                    value={password}
                                    onChange={(e) =>
                                      setPassword(e.target.value)
                                    }
                                    placeholder="Enter your password"
                                    className="h-10 rounded-xl border-slate-300 pr-10 shadow-sm dark:border-slate-700"
                                    onKeyDown={(e) =>
                                      e.key === "Enter" &&
                                      handlePasswordSubmit()
                                    }
                                  />
                                  <Button
                                    type="button"
                                    variant="ghost"
                                    size="sm"
                                    className="absolute right-0 top-0 h-11 px-3 py-2 hover:bg-transparent"
                                    onClick={() =>
                                      setShowPassword(!showPassword)
                                    }
                                  >
                                    {showPassword ? (
                                      <IconEyeOff className="h-4 w-4 text-slate-500" />
                                    ) : (
                                      <IconEye className="h-4 w-4 text-slate-500" />
                                    )}
                                  </Button>
                                </div>
                              </div>
                              <div className="space-y-2">
                                <Button
                                  onClick={handlePasswordSubmit}
                                  className="h-10 w-full rounded-xl"
                                  disabled={!password || customAuthenticating}
                                >
                                  {customAuthenticating ? (
                                    <motion.div
                                      animate={{ rotate: 360 }}
                                      transition={{
                                        duration: 1,
                                        repeat: Infinity,
                                        ease: "linear",
                                      }}
                                      className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                                    />
                                  ) : null}
                                  Sign In
                                </Button>
                                <div className="flex justify-end">
                                  <Button
                                    type="button"
                                    variant="link"
                                    size="sm"
                                    className="h-auto p-0 text-sm text-slate-600 dark:text-slate-300"
                                    onClick={handleOpenForgotPassword}
                                  >
                                    Forgot password?
                                  </Button>
                                </div>
                              </div>
                            </>
                          ) : (
                            <>
                              {forgotPasswordStep === "email" && (
                                <div className="space-y-3">
                                  <Label
                                    htmlFor="forgot-email"
                                    className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                  >
                                    Enter your email
                                  </Label>
                                  <Input
                                    id="forgot-email"
                                    type="email"
                                    value={forgotPasswordEmail}
                                    onChange={(e) =>
                                      setForgotPasswordEmail(e.target.value)
                                    }
                                    placeholder="you@example.com"
                                    className="h-10 rounded-xl border-slate-300 shadow-sm dark:border-slate-700"
                                    onKeyDown={(e) =>
                                      e.key === "Enter" && handleForgotSendOtp()
                                    }
                                  />
                                  <div className="flex gap-2">
                                    <Button
                                      onClick={handleForgotSendOtp}
                                      className="h-10 flex-1 rounded-xl"
                                      disabled={forgotLoading}
                                    >
                                      {forgotLoading ? (
                                        <motion.div
                                          animate={{ rotate: 360 }}
                                          transition={{
                                            duration: 1,
                                            repeat: Infinity,
                                            ease: "linear",
                                          }}
                                          className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                                        />
                                      ) : null}
                                      Send OTP
                                    </Button>
                                    <Button
                                      variant="outline"
                                      onClick={handleCloseForgotPassword}
                                      className="h-11 rounded-xl"
                                    >
                                      Cancel
                                    </Button>
                                  </div>
                                </div>
                              )}

                              {forgotPasswordStep === "otp" && (
                                <div className="space-y-3">
                                  <Label
                                    htmlFor="forgot-otp"
                                    className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                  >
                                    Enter OTP sent to {forgotPasswordEmail}
                                  </Label>
                                  <Input
                                    id="forgot-otp"
                                    type="text"
                                    inputMode="numeric"
                                    value={otp}
                                    onChange={(e) => setOtp(e.target.value)}
                                    placeholder="Enter OTP"
                                    className="h-10 rounded-xl border-slate-300 shadow-sm dark:border-slate-700"
                                    onKeyDown={(e) =>
                                      e.key === "Enter" &&
                                      handleForgotVerifyOtp()
                                    }
                                  />
                                  <div className="flex gap-2">
                                    <Button
                                      onClick={handleForgotVerifyOtp}
                                      className="h-10 flex-1 rounded-xl"
                                      disabled={forgotLoading}
                                    >
                                      {forgotLoading ? (
                                        <motion.div
                                          animate={{ rotate: 360 }}
                                          transition={{
                                            duration: 1,
                                            repeat: Infinity,
                                            ease: "linear",
                                          }}
                                          className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                                        />
                                      ) : null}
                                      Verify OTP
                                    </Button>
                                    <Button
                                      variant="outline"
                                      onClick={() =>
                                        setForgotPasswordStep("email")
                                      }
                                      className="h-11 rounded-xl"
                                    >
                                      Back
                                    </Button>
                                  </div>
                                </div>
                              )}

                              {forgotPasswordStep === "reset" && (
                                <div className="space-y-4">
                                  <div className="space-y-2">
                                    <Label
                                      htmlFor="new-password-reset"
                                      className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                    >
                                      New Password
                                    </Label>
                                    <div className="relative">
                                      <Input
                                        id="new-password-reset"
                                        type={
                                          showNewPassword ? "text" : "password"
                                        }
                                        value={newPassword}
                                        onChange={(e) =>
                                          setNewPassword(e.target.value)
                                        }
                                        placeholder="Enter new password"
                                        className="h-10 rounded-xl border-slate-300 pr-10 shadow-sm dark:border-slate-700"
                                        onKeyDown={(e) =>
                                          e.key === "Enter" &&
                                          handleForgotResetPassword()
                                        }
                                      />
                                      <Button
                                        type="button"
                                        variant="ghost"
                                        size="sm"
                                        className="absolute right-0 top-0 h-11 px-3 py-2 hover:bg-transparent"
                                        onClick={() =>
                                          setShowNewPassword(!showNewPassword)
                                        }
                                      >
                                        {showNewPassword ? (
                                          <IconEyeOff className="h-4 w-4 text-slate-500" />
                                        ) : (
                                          <IconEye className="h-4 w-4 text-slate-500" />
                                        )}
                                      </Button>
                                    </div>
                                  </div>

                                  <div className="space-y-2">
                                    <Label
                                      htmlFor="confirm-new-password-reset"
                                      className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                    >
                                      Confirm New Password
                                    </Label>
                                    <div className="relative">
                                      <Input
                                        id="confirm-new-password-reset"
                                        type={
                                          showConfirmNewPassword
                                            ? "text"
                                            : "password"
                                        }
                                        value={confirmNewPassword}
                                        onChange={(e) =>
                                          setConfirmNewPassword(e.target.value)
                                        }
                                        placeholder="Confirm new password"
                                        className="h-10 rounded-xl border-slate-300 pr-10 shadow-sm dark:border-slate-700"
                                        onKeyDown={(e) =>
                                          e.key === "Enter" &&
                                          handleForgotResetPassword()
                                        }
                                      />
                                      <Button
                                        type="button"
                                        variant="ghost"
                                        size="sm"
                                        className="absolute right-0 top-0 h-11 px-3 py-2 hover:bg-transparent"
                                        onClick={() =>
                                          setShowConfirmNewPassword(
                                            !showConfirmNewPassword
                                          )
                                        }
                                      >
                                        {showConfirmNewPassword ? (
                                          <IconEyeOff className="h-4 w-4 text-slate-500" />
                                        ) : (
                                          <IconEye className="h-4 w-4 text-slate-500" />
                                        )}
                                      </Button>
                                    </div>
                                  </div>

                                  <div className="flex gap-2">
                                    <Button
                                      onClick={handleForgotResetPassword}
                                      className="h-10 flex-1 rounded-xl"
                                      disabled={forgotLoading}
                                    >
                                      {forgotLoading ? (
                                        <motion.div
                                          animate={{ rotate: 360 }}
                                          transition={{
                                            duration: 1,
                                            repeat: Infinity,
                                            ease: "linear",
                                          }}
                                          className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                                        />
                                      ) : null}
                                      Reset Password
                                    </Button>
                                    <Button
                                      variant="outline"
                                      onClick={() =>
                                        setForgotPasswordStep("otp")
                                      }
                                      className="h-10 rounded-xl"
                                    >
                                      Back
                                    </Button>
                                  </div>
                                </div>
                              )}

                              {forgotPasswordStep === "success" && (
                                <div className="space-y-3 text-center">
                                  <div className="mx-auto flex h-10 w-10 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/20">
                                    <IconCircleCheck className="h-6 w-6 text-green-600 dark:text-green-400" />
                                  </div>
                                  <p className="text-sm text-slate-700 dark:text-slate-300">
                                    {forgotSuccess ||
                                      "Password reset successfully."}
                                  </p>
                                  <Button
                                    onClick={handleCloseForgotPassword}
                                    className="h-10 w-full rounded-xl"
                                  >
                                    Back to sign in
                                  </Button>
                                </div>
                              )}

                              {(forgotError ||
                                (forgotSuccess &&
                                  forgotPasswordStep !== "success")) && (
                                <div
                                  className={`mt-3 rounded-lg border p-3 ${
                                    forgotError
                                      ? "bg-red-50 border-red-200 dark:bg-red-900/20 dark:border-red-800"
                                      : "bg-green-50 border-green-200 dark:bg-green-900/20 dark:border-green-800"
                                  }`}
                                >
                                  <div className="flex items-center">
                                    {forgotError ? (
                                      <IconAlertTriangle className="mr-2 h-4 w-4 text-red-600 dark:text-red-400" />
                                    ) : (
                                      <IconCircleCheck className="mr-2 h-4 w-4 text-green-600 dark:text-green-400" />
                                    )}
                                    <p
                                      className={`text-sm ${
                                        forgotError
                                          ? "text-red-700 dark:text-red-300"
                                          : "text-green-700 dark:text-green-300"
                                      }`}
                                    >
                                      {forgotError || forgotSuccess}
                                    </p>
                                  </div>
                                </div>
                              )}

                              {forgotPasswordStep !== "success" && (
                                <div className="mt-2 flex justify-end">
                                  <Button
                                    type="button"
                                    variant="link"
                                    size="sm"
                                    className="h-auto p-0 text-sm"
                                    onClick={handleCloseForgotPassword}
                                  >
                                    Back to sign in
                                  </Button>
                                </div>
                              )}
                            </>
                          )}
                        </>
                      ) : (
                        <>
                          {registrationStep === "details" ? (
                            <>
                              <div className="space-y-2">
                                <Label
                                  htmlFor="full-name"
                                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                >
                                  Full Name
                                </Label>
                                <Input
                                  id="full-name"
                                  type="text"
                                  value={name}
                                  onChange={(e) => setName(e.target.value)}
                                  placeholder="Your full name"
                                  className="h-10 rounded-xl border-slate-300 shadow-sm dark:border-slate-700"
                                  onKeyDown={(e) =>
                                    e.key === "Enter" && handleSetupSubmit()
                                  }
                                  autoComplete="name"
                                />
                              </div>

                              <div className="space-y-4">
                                <div className="space-y-2">
                                  <Label
                                    htmlFor="new-password"
                                    className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                  >
                                    Password
                                  </Label>
                                  <div className="relative">
                                    <Input
                                      id="new-password"
                                      type={showPassword ? "text" : "password"}
                                      value={password}
                                      onChange={(e) =>
                                        setPassword(e.target.value)
                                      }
                                      placeholder="Create a password"
                                      className="h-10 rounded-xl border-slate-300 pr-10 shadow-sm dark:border-slate-700"
                                    />
                                    <Button
                                      type="button"
                                      variant="ghost"
                                      size="sm"
                                      className="absolute right-0 top-0 h-11 px-3 py-2 hover:bg-transparent"
                                      onClick={() =>
                                        setShowPassword(!showPassword)
                                      }
                                    >
                                      {showPassword ? (
                                        <IconEyeOff className="h-4 w-4 text-slate-500" />
                                      ) : (
                                        <IconEye className="h-4 w-4 text-slate-500" />
                                      )}
                                    </Button>
                                  </div>
                                </div>

                                <div className="space-y-2">
                                  <Label
                                    htmlFor="confirm-password"
                                    className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                  >
                                    Confirm Password
                                  </Label>
                                  <div className="relative">
                                    <Input
                                      id="confirm-password"
                                      type={
                                        showConfirmPassword
                                          ? "text"
                                          : "password"
                                      }
                                      value={confirmPassword}
                                      onChange={(e) =>
                                        setConfirmPassword(e.target.value)
                                      }
                                      placeholder="Confirm your password"
                                      className="h-10 rounded-xl border-slate-300 pr-10 shadow-sm dark:border-slate-700"
                                      onKeyDown={(e) =>
                                        e.key === "Enter" && handleSetupSubmit()
                                      }
                                    />
                                    <Button
                                      type="button"
                                      variant="ghost"
                                      size="sm"
                                      className="absolute right-0 top-0 h-11 px-3 py-2 hover:bg-transparent"
                                      onClick={() =>
                                        setShowConfirmPassword(
                                          !showConfirmPassword
                                        )
                                      }
                                    >
                                      {showConfirmPassword ? (
                                        <IconEyeOff className="h-4 w-4 text-slate-500" />
                                      ) : (
                                        <IconEye className="h-4 w-4 text-slate-500" />
                                      )}
                                    </Button>
                                  </div>
                                </div>
                              </div>

                              <Button
                                onClick={handleSetupSubmit}
                                className="h-10 w-full rounded-xl"
                                disabled={
                                  !password ||
                                  !confirmPassword ||
                                  customAuthenticating
                                }
                              >
                                {customAuthenticating ? (
                                  <motion.div
                                    animate={{ rotate: 360 }}
                                    transition={{
                                      duration: 1,
                                      repeat: Infinity,
                                      ease: "linear",
                                    }}
                                    className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                                  />
                                ) : null}
                                Setup Account
                              </Button>
                            </>
                          ) : (
                            <>
                              {registrationMessage && (
                                <div className="mb-3 rounded-lg border border-blue-200 bg-blue-50 p-3 dark:border-blue-800 dark:bg-blue-900/20">
                                  <div className="flex items-center gap-2">
                                    <IconCircleCheck className="h-4 w-4 text-blue-600 dark:text-blue-300" />
                                    <p className="text-sm text-blue-700 dark:text-blue-200">
                                      {registrationMessage}
                                    </p>
                                  </div>
                                </div>
                              )}

                              <div className="space-y-2">
                                <Label
                                  htmlFor="registration-otp"
                                  className="text-sm font-medium text-slate-700 dark:text-slate-300"
                                >
                                  Enter OTP
                                </Label>
                                <Input
                                  id="registration-otp"
                                  type="text"
                                  inputMode="numeric"
                                  value={registrationOtp}
                                  onChange={(e) =>
                                    setRegistrationOtp(e.target.value)
                                  }
                                  placeholder="Enter the code sent to your email"
                                  className="h-10 rounded-xl border-slate-300 shadow-sm dark:border-slate-700"
                                  onKeyDown={(e) =>
                                    e.key === "Enter" &&
                                    handleRegistrationOtpSubmit()
                                  }
                                />
                                <p className="text-xs text-slate-500 dark:text-slate-400">
                                  We sent a verification code to {email}. Enter
                                  it here to finish creating your account.
                                </p>
                              </div>

                              <div className="mt-4 space-y-2">
                                <Button
                                  onClick={handleRegistrationOtpSubmit}
                                  className="h-10 w-full rounded-xl"
                                  disabled={
                                    !registrationOtp.trim() ||
                                    customAuthenticating
                                  }
                                >
                                  {customAuthenticating ? (
                                    <motion.div
                                      animate={{ rotate: 360 }}
                                      transition={{
                                        duration: 1,
                                        repeat: Infinity,
                                        ease: "linear",
                                      }}
                                      className="mr-2 h-4 w-4 rounded-full border-2 border-white border-t-transparent"
                                    />
                                  ) : null}
                                  Verify OTP & Continue
                                </Button>
                                <Button
                                  type="button"
                                  variant="ghost"
                                  className="h-10 w-full rounded-xl"
                                  onClick={() => {
                                    setRegistrationStep("details");
                                    setRegistrationMessage(null);
                                    setRegistrationOtp("");
                                  }}
                                >
                                  Back
                                </Button>
                              </div>
                            </>
                          )}
                        </>
                      )}
                    </>
                  )}
                </div>
              </>
            )}

            {error && (
              <motion.div
                initial={{ opacity: 0, y: -10 }}
                animate={{ opacity: 1, y: 0 }}
                className="auth-inline-note border-red-200 bg-red-50"
              >
                <div className="flex items-center gap-2">
                  <IconAlertTriangle className="h-4 w-4 text-red-600" />
                  <p className="text-sm text-red-700">{error}</p>
                </div>
              </motion.div>
            )}
        </motion.div>

        <p className="text-center text-xs text-slate-500">
          Secure authentication powered by AuthSec
        </p>
      </AuthActionPanel>
    </AuthSplitFrame>
  );
};

export const OIDCLoginPage: React.FC = () => {
  return (
    <EndUserAuthProvider>
      <OIDCLoginPageInner />
    </EndUserAuthProvider>
  );
};
