import React, { useMemo, useState, useEffect } from "react";
import { useLocation, useNavigate } from "react-router-dom";
import { Loader2, ArrowLeft, Clock, CheckCircle2, XCircle } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { PasswordInput } from "@/components/ui/password-input";
import { OTPInput } from "@/components/ui/otp-input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import authsecLogoWhite from "@/logos/AuthSec Logo White.png";
import { useTheme } from "next-themes";
import {
  useAdminLoginPrecheckMutation,
  useAdminBootstrapAccountMutation,
  useRegisterVerifyMutation,
  useResendOtpMutation,
  useAdminForgotPasswordMutation,
  useAdminForgotPasswordVerifyOtpMutation,
  useAdminForgotPasswordResetMutation,
  type AdminLoginPrecheckResponse,
} from "@/app/api/authApi";
import { useAuth } from "../context/AuthContext";
import { toast } from "react-hot-toast";
import {
  useGetUFlowOIDCProvidersMutation,
  useInitiateUFlowOIDCMutation,
  useLazyCheckTenantDomainQuery,
  type UFlowOIDCProvider,
  type UFlowOIDCCallbackData,
} from "@/app/api/oidcApi";
import { TenantDomainSelectionModal } from "../components/TenantDomainSelectionModal";
import { encodeHandoff, decodeHandoff } from "@/utils/handoff";
import config from "../../config";
import {
  trackSignInAttempted,
  trackSignInSucceeded,
  trackSignUpStarted,
  trackWorkspaceCreated,
  trackOtpVerified,
  trackOAuthProviderClicked,
  trackForgotPasswordRequested,
  trackPasswordResetCompleted,
} from "@/utils/analytics";
import { trackXSignupCompleted } from "@/utils/xPixel";
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

type FlowStage = "idle" | "existing" | "register" | "otp";

interface StageTransitionOptions {
  emailOverride?: string;
  skipEmailRequirement?: boolean;
}

interface IdleNotice {
  tone: "info" | "error";
  message: string;
}

// Lightweight, URL-safe handoff token utilities for cross-tenant redirects
// (moved to utils/handoff)

const GithubIcon = () => (
  <svg
    className="h-5 w-5"
    viewBox="0 0 24 24"
    fill="currentColor"
    aria-hidden="true"
  >
    <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
  </svg>
);

const GoogleIcon = () => (
  <svg className="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
    <path
      fill="#EA4335"
      d="M12 10.2v4.07h5.7c-.25 1.32-1.69 3.87-5.7 3.87-3.43 0-6.23-2.83-6.23-6.33s2.8-6.33 6.23-6.33c1.96 0 3.28.84 4.03 1.57l2.74-2.65C16.64 2.09 14.51 1 12 1 6.76 1 2.5 5.27 2.5 10.5S6.76 20 12 20c6.98 0 8.51-6.03 7.94-9.8z"
    />
    <path
      fill="#34A853"
      d="M1.22 6.73 4.5 9.2c.78-1.97 2.73-3.5 5-3.5 1.44 0 2.56.62 3.25 1.28l2.65-2.57C14.64 2.16 12.75 1.3 10.5 1.3 6.38 1.3 2.88 3.9 1.22 6.73z"
    />
    <path
      fill="#4A90E2"
      d="M12 22.7c2.21 0 4.08-.72 5.43-1.95L14.9 18.4c-.72.49-1.75.8-2.9.8-2.24 0-4.15-1.53-4.82-3.57l-3.36 2.6C5.45 21.01 8.51 22.7 12 22.7z"
    />
    <path
      fill="#FBBC05"
      d="M22.28 9.13H21.5V9H12v4.2h5.83c-.25 1.58-1.06 2.82-2.2 3.68l3.39 2.64c1.98-1.83 3.13-4.53 3.13-7.92 0-.79-.08-1.55-.27-2.47z"
    />
  </svg>
);

const MicrosoftIcon = () => (
  <svg className="h-5 w-5" viewBox="0 0 24 24" aria-hidden="true">
    <path fill="#F25022" d="M11.5 11.5H2v-9h9.5v9z" />
    <path fill="#7FBA00" d="M22 11.5H12.5v-9H22v9z" />
    <path fill="#00A4EF" d="M11.5 22H2v-9h9.5v9z" />
    <path fill="#FFB900" d="M22 22H12.5v-9H22v9z" />
  </svg>
);

const PROVIDER_ORDER: Record<string, number> = {
  microsoft: 0,
  google: 1,
  github: 2,
};

const normalizeProviderKey = (value?: string | null) =>
  (value || "").trim().toLowerCase();

const getProviderDisplayName = (provider: UFlowOIDCProvider) => {
  const key = normalizeProviderKey(provider.provider_name);
  if (key === "google") return "Google Workspace";
  if (key === "microsoft") return "Microsoft";
  if (key === "github") return "GitHub";
  return provider.display_name;
};

export function AdminLoginHubPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const from =
    (location.state as { from?: { pathname?: string } } | null)?.from
      ?.pathname || "/dashboard";
  const { signIn } = useAuth();
  const [flowStage, setFlowStage] = useState<FlowStage>("idle");
  const [emailInput, setEmailInput] = useState("");
  const [checkedEmail, setCheckedEmail] = useState<string>("");
  const [existingPassword, setExistingPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [tenantDomain, setTenantDomain] = useState("");
  const [domainCheckStatus, setDomainCheckStatus] = useState<
    "idle" | "checking" | "available" | "taken" | "error"
  >("idle");
  const [domainCheckMessage, setDomainCheckMessage] = useState<string | null>(
    null,
  );
  const [checkTenantDomain] = useLazyCheckTenantDomainQuery();
  const [isPasswordSubmitting, setIsPasswordSubmitting] = useState(false);
  const [isPrecheckInProgress, setIsPrecheckInProgress] = useState(false);

  // OTP-related state
  const [otp, setOtp] = useState("");
  const [isVerifyingOtp, setIsVerifyingOtp] = useState(false);
  const [isResending, setIsResending] = useState(false);
  const [timeLeft, setTimeLeft] = useState(60);
  const [canResend, setCanResend] = useState(false);
  const [isForgotPasswordOpen, setIsForgotPasswordOpen] = useState(false);
  const [forgotPasswordStep, setForgotPasswordStep] = useState<
    "email" | "otp" | "reset"
  >("email");
  const [forgotPasswordEmail, setForgotPasswordEmail] = useState("");
  const [forgotPasswordOtp, setForgotPasswordOtp] = useState("");
  const [forgotPasswordNew, setForgotPasswordNew] = useState("");
  const [forgotPasswordConfirm, setForgotPasswordConfirm] = useState("");
  const [isForgotPasswordSubmitting, setIsForgotPasswordSubmitting] =
    useState(false);

  const [adminLoginPrecheck, { isLoading: isPrecheckLoading }] =
    useAdminLoginPrecheckMutation();
  const [bootstrapAccount, { isLoading: isBootstrapLoading }] =
    useAdminBootstrapAccountMutation();
  const [verifyOtp] = useRegisterVerifyMutation();
  const [resendOtpMutation] = useResendOtpMutation();
  const [adminForgotPassword] = useAdminForgotPasswordMutation();
  const [adminVerifyForgotOtp] = useAdminForgotPasswordVerifyOtpMutation();
  const [adminResetForgotPassword] = useAdminForgotPasswordResetMutation();

  // UFlow OAuth state
  const [uflowProviders, setUflowProviders] = useState<UFlowOIDCProvider[]>([]);
  const [showDomainModal, setShowDomainModal] = useState(false);
  const [uflowCallbackData, setUflowCallbackData] =
    useState<UFlowOIDCCallbackData | null>(null);
  const [authenticatingProvider, setAuthenticatingProvider] = useState<
    string | null
  >(null);
  const [idleNotice, setIdleNotice] = useState<IdleNotice | null>(null);
  const [ssoProviderName, setSsoProviderName] = useState<string | null>(null);

  const [getUFlowOIDCProviders] = useGetUFlowOIDCProvidersMutation();
  const [initiateUFlowOIDC] = useInitiateUFlowOIDCMutation();

  const currentEmail = useMemo(() => {
    // Prefer checked email (server-validated) over input
    if (checkedEmail) return checkedEmail;
    if (emailInput) return emailInput;
    return "";
  }, [checkedEmail, emailInput]);

  const orderedUflowProviders = useMemo(
    () =>
      [...uflowProviders].sort((left, right) => {
        const leftRank =
          PROVIDER_ORDER[normalizeProviderKey(left.provider_name)] ?? 99;
        const rightRank =
          PROVIDER_ORDER[normalizeProviderKey(right.provider_name)] ?? 99;

        if (leftRank !== rightRank) return leftRank - rightRank;
        return getProviderDisplayName(left).localeCompare(
          getProviderDisplayName(right),
        );
      }),
    [uflowProviders],
  );

  const ssoProvider = useMemo(() => {
    if (!ssoProviderName) return null;
    return (
      orderedUflowProviders.find(
        (provider) =>
          normalizeProviderKey(provider.provider_name) === ssoProviderName,
      ) || null
    );
  }, [orderedUflowProviders, ssoProviderName]);

  // Countdown timer for OTP resend
  useEffect(() => {
    if (flowStage === "otp" && timeLeft > 0 && !canResend) {
      const timer = setTimeout(() => setTimeLeft(timeLeft - 1), 1000);
      return () => clearTimeout(timer);
    } else if (timeLeft === 0) {
      setCanResend(true);
    }
  }, [timeLeft, canResend, flowStage]);

  // Real-time domain availability check
  useEffect(() => {
    if (!tenantDomain || tenantDomain.length < 2) {
      setDomainCheckStatus("idle");
      setDomainCheckMessage(null);
      return;
    }
    setDomainCheckStatus("checking");
    const timer = setTimeout(async () => {
      try {
        const result = await checkTenantDomain(tenantDomain).unwrap();
        if (result.exists) {
          setDomainCheckStatus("taken");
          setDomainCheckMessage("This domain is already taken");
        } else {
          setDomainCheckStatus("available");
          setDomainCheckMessage("Available");
        }
      } catch {
        setDomainCheckStatus("error");
        setDomainCheckMessage("Could not check availability");
      }
    }, 500);
    return () => clearTimeout(timer);
  }, [tenantDomain, checkTenantDomain]);

  // Debug logging (development only)
  useEffect(() => {
    if (process.env.NODE_ENV === "development") {
      console.log("AdminLoginHub State:", {
        flowStage,
        emailInput,
        checkedEmail,
        currentEmail,
        tenantDomain,
        isPrecheckInProgress,
        urlParams: location.search,
      });
    }
  }, [
    flowStage,
    emailInput,
    checkedEmail,
    currentEmail,
    tenantDomain,
    isPrecheckInProgress,
    location.search,
  ]);

  // Fetch UFlow OAuth providers on mount
  useEffect(() => {
    console.log(
      "[AdminLogin] 🚀 useEffect triggered - fetching UFlow providers",
    );
    console.log("[AdminLogin] 📍 API Base URL:", config.VITE_API_URL);
    console.log(
      "[AdminLogin] 🔧 Full endpoint:",
      `${config.VITE_API_URL}/uflow/oidc/providers`,
    );

    const fetchUFlowProviders = async () => {
      try {
        console.log("[AdminLogin] 📤 Calling getUFlowOIDCProviders API...");
        const result = await getUFlowOIDCProviders({ email: "" }).unwrap();
        console.log("[AdminLogin] 📥 API Response:", result);

        if (result?.providers) {
          console.log("[AdminLogin] ✅ Setting providers:", result.providers);
          setUflowProviders(result.providers);
        } else {
          console.warn("[AdminLogin] ⚠️ No providers in response");
        }
      } catch (error) {
        console.error(
          "[AdminLogin] ❌ Failed to fetch UFlow OAuth providers:",
          error,
        );
        console.error(
          "[AdminLogin] 💥 Error details:",
          JSON.stringify(error, null, 2),
        );
        // Show error to user for debugging
        toast.error(
          "Failed to load login providers. Please check console for details.",
        );
      }
    };

    fetchUFlowProviders();
  }, [getUFlowOIDCProviders]);

  // Handle UFlow OAuth callback parameters
  useEffect(() => {
    const handleUFlowCallback = () => {
      const params = new URLSearchParams(location.search);
      const showDomainModalParam = params.get("show_domain_modal");

      if (showDomainModalParam === "true") {
        // Retrieve stored OAuth callback data from sessionStorage
        const storedEmail = sessionStorage.getItem("uflow_user_email");
        const storedName = sessionStorage.getItem("uflow_user_name");
        const storedPicture = sessionStorage.getItem("uflow_user_picture");
        const storedProvider = sessionStorage.getItem("uflow_provider");
        const storedProviderUserId = sessionStorage.getItem(
          "uflow_provider_user_id",
        );

        if (
          storedEmail &&
          storedName &&
          storedProvider &&
          storedProviderUserId
        ) {
          setUflowCallbackData({
            email: storedEmail,
            name: storedName,
            picture: storedPicture || "",
            provider: storedProvider,
            provider_user_id: storedProviderUserId,
            needs_domain: true,
            success: false,
          });
          setShowDomainModal(true);

          // Clear URL parameter
          navigate("/admin/login", { replace: true });
        }
      }
    };

    handleUFlowCallback();
  }, [location.search, navigate]);

  // Email validation helper
  const isValidEmail = (email: string | null): email is string => {
    if (!email || email === "undefined" || email === "null") return false;
    // Basic email validation
    return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
  };

  // Apply cross-tenant handoff (redirect payload) if present
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const handoff = params.get("handoff");
    if (!handoff) return;

    const payload = decodeHandoff<{
      email: string;
      tenant_domain?: string;
      flow_stage?: FlowStage;
      verified?: boolean;
    }>(handoff);

    if (!payload) {
      // Remove invalid handoff token from URL
      params.delete("handoff");
      navigate(
        {
          pathname: location.pathname,
          search: params.toString() ? `?${params.toString()}` : "",
        },
        { replace: true },
      );
      return;
    }

    if (payload.email) {
      setEmailInput(payload.email);
      setCheckedEmail(payload.email);
    }
    if (payload.tenant_domain) {
      setTenantDomain(payload.tenant_domain);
    }

    if (payload.flow_stage === "existing") {
      safeSetFlowStage("existing", {
        emailOverride: payload.email,
        skipEmailRequirement: true,
      });
    } else if (payload.flow_stage === "otp") {
      safeSetFlowStage("otp", {
        emailOverride: payload.email,
        skipEmailRequirement: true,
      });
      setTimeLeft(60);
      setCanResend(false);
    }

    // Clean up query params after applying
    params.delete("handoff");
    navigate(
      {
        pathname: location.pathname,
        search: params.toString() ? `?${params.toString()}` : "",
      },
      { replace: true },
    );
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Pre-fill email from URL params and handle verified redirects
  useEffect(() => {
    console.log("[AdminLogin/useEffect] 🚀 Effect triggered");
    console.log("[AdminLogin/useEffect] 📍 Current URL:", window.location.href);
    console.log("[AdminLogin/useEffect] 📊 Current State:", {
      checkedEmail,
      flowStage,
      isPrecheckInProgress,
      tenantDomain,
    });

    const params = new URLSearchParams(location.search);
    const emailFromUrl = params.get("email");
    const verified = params.get("verified");

    console.log("[AdminLogin/useEffect] 🔍 URL Params:", {
      email: emailFromUrl,
      verified,
      allParams: Object.fromEntries(params.entries()),
    });

    // Handle invalid email
    if (emailFromUrl && !isValidEmail(emailFromUrl)) {
      console.log(
        "[AdminLogin/useEffect] ❌ Invalid email detected:",
        emailFromUrl,
      );
      console.log(
        "[AdminLogin/useEffect] 🔄 Clearing URL and redirecting to clean login",
      );
      // Clear invalid email from URL
      navigate("/admin/login", { replace: true });
      return;
    }

    // Only proceed if we have a valid email and haven't already processed it
    if (
      isValidEmail(emailFromUrl) &&
      !checkedEmail &&
      flowStage === "idle" &&
      !isPrecheckInProgress
    ) {
      console.log(
        "[AdminLogin/useEffect] ✅ Valid email, proceeding with flow:",
        emailFromUrl,
      );
      console.log(
        "[AdminLogin/useEffect] 📝 Setting emailInput to:",
        emailFromUrl,
      );
      setEmailInput(emailFromUrl);

      // If verified flag is present, skip precheck and go straight to password field
      if (verified === "true") {
        console.log(
          "[AdminLogin/useEffect] 🎯 VERIFIED FLAG DETECTED - Skipping precheck",
        );
        console.log(
          "[AdminLogin/useEffect] 📧 Setting checkedEmail to:",
          emailFromUrl,
        );
        console.log(
          '[AdminLogin/useEffect] 🔄 Transitioning to "existing" stage',
        );
        // User was already verified on another domain, just set up state
        setCheckedEmail(emailFromUrl);
        toast.success("Welcome back! Please enter your password.");
        safeSetFlowStage("existing", { emailOverride: emailFromUrl });
        return;
      }

      console.log(
        "[AdminLogin/useEffect] 🔍 No verified flag - Starting AUTO-PRECHECK",
      );
      // No verified flag - do precheck (handles direct links, bookmarks, etc.)
      const doPrecheck = async () => {
        console.log(
          "[AdminLogin/AutoPrecheck] 🚀 Starting auto-precheck for:",
          emailFromUrl,
        );
        setIsPrecheckInProgress(true);
        console.log(
          "[AdminLogin/AutoPrecheck] 🔒 Set isPrecheckInProgress = true",
        );

        try {
          const payload = { email: emailFromUrl.trim().toLowerCase() };
          console.log(
            "[AdminLogin/AutoPrecheck] 📤 Calling API with payload:",
            payload,
          );

          const response = await adminLoginPrecheck(payload).unwrap();

          console.log(
            "[AdminLogin/AutoPrecheck] 📥 API Response received:",
            response,
          );

          // Only update state if response is valid
          if (response) {
            const validatedEmail = emailFromUrl.trim().toLowerCase();
            console.log(
              "[AdminLogin/AutoPrecheck] ✅ Valid response, using input email:",
              validatedEmail,
            );
            handlePrecheckResponse(response, validatedEmail);
          } else {
            console.log(
              "[AdminLogin/AutoPrecheck] ❌ Invalid response:",
              response,
            );
          }
        } catch (error: unknown) {
          const apiError = error as { data?: { message?: string } };
          const message = getPrecheckErrorMessage(apiError);
          console.log("[AdminLogin/AutoPrecheck] 💥 API Error:", error);
          console.log("[AdminLogin/AutoPrecheck] 📛 Error message:", message);
          toast.error(message);
          // Clear invalid email from URL
          console.log("[AdminLogin/AutoPrecheck] 🔄 Clearing URL due to error");
          navigate("/admin/login", { replace: true });
        } finally {
          console.log(
            "[AdminLogin/AutoPrecheck] 🔓 Set isPrecheckInProgress = false",
          );
          setIsPrecheckInProgress(false);
        }
      };
      doPrecheck();
    } else {
      console.log("[AdminLogin/useEffect] ⏭️ Skipping - conditions not met:", {
        hasValidEmail: isValidEmail(emailFromUrl),
        checkedEmail,
        flowStage,
        isPrecheckInProgress,
      });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.search]);

  // Redirect helper functions
  const buildRedirectUrl = (
    targetDomain: string,
    email: string,
    path: string = "/admin/login",
    verified: boolean = true,
    handoffData?: { flow_stage?: FlowStage; tenant_domain?: string },
  ): string => {
    console.log("[Redirect/buildUrl] 🔨 Building redirect URL");
    console.log("[Redirect/buildUrl] 📊 Inputs:", {
      targetDomain,
      email,
      path,
      verified,
      currentDomain: window.location.hostname,
      currentProtocol: window.location.protocol,
    });

    // Validate inputs
    if (!targetDomain || !email) {
      console.error("[Redirect/buildUrl] ❌ Invalid redirect parameters:", {
        targetDomain,
        email,
      });
      return "";
    }

    // Ensure email is properly encoded
    const params = new URLSearchParams();
    params.set("email", email.trim().toLowerCase());
    console.log(
      "[Redirect/buildUrl] 📧 Email param set:",
      email.trim().toLowerCase(),
    );

    // Add verified flag to skip precheck on redirect
    if (verified) {
      params.set("verified", "true");
      console.log("[Redirect/buildUrl] ✅ Added verified=true flag");
    }

    // Attach a compact handoff payload so the target domain can restore state instantly
    if (handoffData) {
      const handoffToken = encodeHandoff({
        ...handoffData,
        email: email.trim().toLowerCase(),
        ts: Date.now(),
      });
      if (handoffToken) {
        params.set("handoff", handoffToken);
        console.log("[Redirect/buildUrl] 🎁 Added handoff token");
      }
    }

    // Handle different environments
    const protocol = window.location.protocol; // Preserve http/https
    const url = `${protocol}//${targetDomain}${path}?${params.toString()}`;

    console.log("[Redirect/buildUrl] 🎯 Final redirect URL:", url);
    return url;
  };

  const performRedirect = (url: string, delay: number = 800) => {
    console.log("[Redirect/perform] 🚀 Initiating redirect");
    console.log("[Redirect/perform] 🎯 Target URL:", url);
    console.log("[Redirect/perform] ⏱️ Delay:", delay, "ms");
    console.log("[Redirect/perform] 📍 Current URL:", window.location.href);

    if (!url) {
      console.error("[Redirect/perform] ❌ Empty URL provided");
      toast.error("Unable to redirect. Please try again.");
      return;
    }

    console.log("[Redirect/perform] ⏳ Setting timeout for redirect...");
    setTimeout(() => {
      console.log("[Redirect/perform] 🏃 Executing redirect NOW");
      console.log(
        "[Redirect/perform] 🌐 Changing window.location.href to:",
        url,
      );
      window.location.href = url;
    }, delay);
  };

  // State machine validation
  const canTransitionTo = (
    newStage: FlowStage,
    options: StageTransitionOptions = {},
  ): boolean => {
    // Check actual state values instead of derived currentEmail to avoid race condition
    const hasEmail = options.skipEmailRequirement
      ? true
      : !!(options.emailOverride || checkedEmail || emailInput);

    // Validate email exists before allowing transitions
    if (newStage !== "idle" && !hasEmail) {
      console.error("Cannot transition without email. State:", {
        checkedEmail,
        emailInput,
      });
      return false;
    }

    // Validate state transitions
    const validTransitions: Record<FlowStage, FlowStage[]> = {
      idle: ["existing", "register"],
      existing: ["idle", "otp"], // Can go to OTP if password reset needed
      register: ["idle", "otp"],
      otp: ["idle", "existing"], // Can go back after verification
    };

    if (!validTransitions[flowStage].includes(newStage)) {
      console.warn("Invalid state transition:", flowStage, "->", newStage);
    }

    return true;
  };

  const safeSetFlowStage = (
    newStage: FlowStage,
    options: StageTransitionOptions = {},
  ) => {
    console.log("[FlowStage] 🔄 Attempting to transition:", {
      from: flowStage,
      to: newStage,
      currentEmail,
      options,
    });

    if (canTransitionTo(newStage, options)) {
      console.log(
        "[FlowStage] ✅ Transition allowed, setting flowStage to:",
        newStage,
      );
      setFlowStage(newStage);
    } else {
      console.log("[FlowStage] ❌ Transition BLOCKED");
    }
  };

  const clearIdleState = () => {
    setIdleNotice(null);
    setSsoProviderName(null);
  };

  const getSsoRequiredProvider = (
    response: AdminLoginPrecheckResponse,
  ): string | null => {
    const providers = response.available_providers ?? [];
    const nonEmailProviders = providers.filter(
      (provider) => provider !== "email",
    );
    const allowsPassword =
      response.requires_password !== false && providers.includes("email");

    if (!response.exists || allowsPassword || nonEmailProviders.length !== 1) {
      return null;
    }

    return normalizeProviderKey(nonEmailProviders[0]);
  };

  const getPrecheckErrorMessage = (error: any) => {
    const rawMessage =
      error?.data?.message ||
      error?.error ||
      "Unable to verify email right now";
    const status = error?.status || error?.originalStatus;

    if (status === 429 || /too many|rate limit/i.test(rawMessage)) {
      return "Too many attempts. Try again later.";
    }

    return rawMessage;
  };

  const handlePrecheckResponse = (
    response: AdminLoginPrecheckResponse,
    validatedEmail: string,
  ) => {
    setCheckedEmail(validatedEmail);

    if (response.tenant_domain) {
      setTenantDomain(response.tenant_domain);
    }

    const availableProviders = response.available_providers ?? [];
    const nonEmailProviders = availableProviders.filter(
      (provider) => provider !== "email",
    );
    const passwordAllowed =
      response.requires_password !== false &&
      availableProviders.includes("email");
    const requiredSsoProvider = getSsoRequiredProvider(response);

    if (response.exists && !passwordAllowed && nonEmailProviders.length > 0) {
      setSsoProviderName(requiredSsoProvider);
      setIdleNotice({
        tone: "info",
        message:
          nonEmailProviders.length > 1
            ? "This organization requires SSO. Use your identity provider below."
            : "This organization requires SSO.",
      });
      return;
    }

    clearIdleState();

    if (response.exists && response.tenant_domain) {
      const currentDomain = window.location.hostname;
      const targetDomain = response.tenant_domain;
      const shouldRedirect =
        currentDomain !== targetDomain &&
        !currentDomain.includes("localhost") &&
        !currentDomain.includes("127.0.0.1");

      if (shouldRedirect) {
        const redirectUrl = buildRedirectUrl(
          targetDomain,
          validatedEmail,
          "/admin/login",
          true,
          {
            flow_stage: "existing",
            tenant_domain: targetDomain,
          },
        );

        if (redirectUrl) {
          toast.success("Redirecting to your workspace...");
          performRedirect(redirectUrl);
        } else {
          toast.error("Invalid redirect configuration");
        }
        return;
      }

      toast.success("Account found. Enter your password to continue.");
      safeSetFlowStage("existing", { emailOverride: validatedEmail });
      return;
    }

    if (response.exists) {
      toast.success("Account found. Enter your password to continue.");
      safeSetFlowStage("existing", { emailOverride: validatedEmail });
      return;
    }

    toast("Let's create your workspace.");
    safeSetFlowStage("register", { emailOverride: validatedEmail });
  };

  const flowCopy = useMemo(() => {
    switch (flowStage) {
      case "existing":
        return {
          title: "Sign in",
          description: "Enter your password to continue.",
        };
      case "register":
        return {
          title: "Create your workspace",
          description: "Set your workspace domain and password to continue.",
        };
      case "otp":
        return {
          title: "Check your inbox",
          description: `Enter the 6-digit code we sent to ${currentEmail}.`,
        };
      default:
        return {
          title: "Get started",
          description: "Enter your work email to continue.",
        };
    }
  }, [flowStage, currentEmail]);

  const resetFlow = () => {
    setFlowStage("idle");
    setEmailInput("");
    setCheckedEmail("");
    setExistingPassword("");
    setTenantDomain("");
    setDomainCheckStatus("idle");
    setDomainCheckMessage(null);
    setNewPassword("");
    setConfirmPassword("");
    setOtp("");
    setTimeLeft(60);
    setCanResend(false);
    setIsPrecheckInProgress(false); // Clear the guard
    clearIdleState();

    // Clear URL params to prevent auto-precheck loop
    navigate("/admin/login", { replace: true });
  };

  const resetForgotPasswordState = () => {
    setIsForgotPasswordOpen(false);
    setForgotPasswordStep("email");
    setForgotPasswordEmail("");
    setForgotPasswordOtp("");
    setForgotPasswordNew("");
    setForgotPasswordConfirm("");
    setIsForgotPasswordSubmitting(false);
  };

  const handleForgotPasswordOpen = () => {
    const defaultEmail = (currentEmail || emailInput || "").trim();
    setForgotPasswordEmail(defaultEmail);
    setForgotPasswordStep("email");
    setForgotPasswordOtp("");
    setForgotPasswordNew("");
    setForgotPasswordConfirm("");
    setIsForgotPasswordSubmitting(false);
    setIsForgotPasswordOpen(true);
  };

  const handleForgotPasswordDialogChange = (open: boolean) => {
    if (open) {
      if (!forgotPasswordEmail) {
        const defaultEmail = (currentEmail || emailInput || "").trim();
        setForgotPasswordEmail(defaultEmail);
      }
      setIsForgotPasswordOpen(true);
      return;
    }

    resetForgotPasswordState();
  };

  const handleForgotPasswordRequest = async (event: React.FormEvent) => {
    event.preventDefault();

    if (!forgotPasswordEmail) {
      toast.error("Please enter the email associated with your account");
      return;
    }

    setIsForgotPasswordSubmitting(true);

    try {
      const email = forgotPasswordEmail.trim().toLowerCase();
      const response = await adminForgotPassword({ email }).unwrap();
      toast.success(
        response?.message ||
          "If the email is registered, we'll send you an OTP to reset your password.",
      );
      trackForgotPasswordRequested();
      setForgotPasswordStep("otp");
    } catch (error: unknown) {
      const apiError = error as { data?: { message?: string } };
      toast.error(apiError?.data?.message || "Failed to send reset email");
    } finally {
      setIsForgotPasswordSubmitting(false);
    }
  };

  const handleForgotPasswordOtpVerification = async (
    event: React.FormEvent,
  ) => {
    event.preventDefault();

    if (forgotPasswordOtp.length !== 6) {
      toast.error("Enter the 6-digit OTP sent to your email");
      return;
    }

    setIsForgotPasswordSubmitting(true);

    try {
      const email = forgotPasswordEmail.trim().toLowerCase();
      const response = await adminVerifyForgotOtp({
        email,
        otp: forgotPasswordOtp,
      }).unwrap();
      toast.success(
        response?.message ||
          "OTP verified successfully. You can now reset your password.",
      );
      setForgotPasswordStep("reset");
    } catch (error: unknown) {
      const apiError = error as { data?: { message?: string } };
      toast.error(apiError?.data?.message || "Invalid OTP. Please try again.");
    } finally {
      setIsForgotPasswordSubmitting(false);
    }
  };

  const handleForgotPasswordReset = async (event: React.FormEvent) => {
    event.preventDefault();

    if (forgotPasswordNew !== forgotPasswordConfirm) {
      toast.error("Passwords don't match");
      return;
    }

    if (forgotPasswordNew.length < 10) {
      toast.error("Password must be at least 10 characters long");
      return;
    }

    setIsForgotPasswordSubmitting(true);
    const normalizedEmail = forgotPasswordEmail.trim().toLowerCase();

    try {
      await adminResetForgotPassword({
        email: normalizedEmail,
        new_password: forgotPasswordNew,
      }).unwrap();

      toast.success(
        "Password reset successfully. Please sign in with your new password.",
      );
      trackPasswordResetCompleted();
      resetForgotPasswordState();
      setEmailInput(normalizedEmail);
      setCheckedEmail(normalizedEmail);
      setExistingPassword("");

      if (flowStage !== "existing") {
        safeSetFlowStage("existing", {
          emailOverride: normalizedEmail,
          skipEmailRequirement: true,
        });
      }
    } catch (error: unknown) {
      const apiError = error as { data?: { message?: string } };
      toast.error(apiError?.data?.message || "Failed to reset password");
    } finally {
      setIsForgotPasswordSubmitting(false);
    }
  };

  // UFlow OAuth handler
  const handleUFlowProviderAuth = async (provider: UFlowOIDCProvider) => {
    try {
      setIdleNotice(null);
      trackOAuthProviderClicked(provider.provider_name);
      setAuthenticatingProvider(provider.provider_name);

      const response = await initiateUFlowOIDC({
        provider: provider.provider_name.toLowerCase(),
      }).unwrap();

      // Store provider info and state in sessionStorage
      sessionStorage.setItem("uflow_oauth_provider", provider.provider_name);
      sessionStorage.setItem("uflow_oauth_state", response.state);
      sessionStorage.setItem("uflow_oauth_type", "admin"); // Flag for admin login

      // Redirect to OAuth provider
      if (response.redirect_url) {
        window.location.href = response.redirect_url;
      } else {
        toast.error("Failed to initiate OAuth authentication");
        setAuthenticatingProvider(null);
      }
    } catch (err) {
      console.error("UFlow OAuth initiation error:", err);
      toast.error(
        err instanceof Error
          ? err.message
          : "Failed to start OAuth authentication",
      );
      setAuthenticatingProvider(null);
    }
  };

  // Handle domain modal success
  const handleDomainModalSuccess = async (data: {
    tenant_id: string;
    client_id: string;
    tenant_domain: string;
  }) => {
    console.log("Domain registration successful:", data);

    // Clear session storage
    sessionStorage.removeItem("uflow_user_email");
    sessionStorage.removeItem("uflow_user_name");
    sessionStorage.removeItem("uflow_user_picture");
    sessionStorage.removeItem("uflow_provider");
    sessionStorage.removeItem("uflow_provider_user_id");
    sessionStorage.removeItem("uflow_oauth_type");

    if (uflowCallbackData) {
      // Set the email and tenant domain for OTP verification
      setEmailInput(uflowCallbackData.email);
      setCheckedEmail(uflowCallbackData.email);
      setTenantDomain(data.tenant_domain);

      toast.success(
        "Workspace created! Check your email for verification code.",
      );

      // Transition to OTP stage
      safeSetFlowStage("otp");
      setTimeLeft(60);
      setCanResend(false);
      setShowDomainModal(false);
    }
  };

  const handleEmailPrecheck = async () => {
    console.log("[ManualPrecheck] 🚀 Manual precheck triggered");
    console.log("[ManualPrecheck] 📊 Current state:", {
      emailInput,
      isPrecheckInProgress,
      checkedEmail,
      flowStage,
    });

    if (!emailInput || isPrecheckInProgress) {
      console.log("[ManualPrecheck] ⏭️ Skipping - guard conditions:", {
        hasEmailInput: !!emailInput,
        isPrecheckInProgress,
      });
      return;
    }

    // Validate email format
    if (!isValidEmail(emailInput)) {
      console.log("[ManualPrecheck] ❌ Invalid email format:", emailInput);
      toast.error("Please enter a valid email address");
      return;
    }

    if (ssoProvider) {
      clearIdleState();
      await handleUFlowProviderAuth(ssoProvider);
      return;
    }

    console.log("[ManualPrecheck] 🔒 Setting isPrecheckInProgress = true");
    setIsPrecheckInProgress(true);
    setIdleNotice(null);

    try {
      const payload = { email: emailInput.trim().toLowerCase() };
      console.log("[ManualPrecheck] 📤 Calling API with payload:", payload);

      const response = await adminLoginPrecheck(payload).unwrap();

      console.log("[ManualPrecheck] 📥 API Response:", response);

      // Validate response
      if (!response) {
        console.log("[ManualPrecheck] ❌ Invalid response:", response);
        toast.error("Invalid response from server");
        return;
      }

      const validatedEmail = emailInput.trim().toLowerCase();
      console.log(
        "[ManualPrecheck] ✅ Valid response, using input email:",
        validatedEmail,
      );
      handlePrecheckResponse(response, validatedEmail);
    } catch (error: any) {
      console.log("[ManualPrecheck] 💥 API Error:", error);
      const message = getPrecheckErrorMessage(error);
      console.log("[ManualPrecheck] 📛 Error message:", message);
      setIdleNotice({ tone: "error", message });
      toast.error(message);
    } finally {
      console.log("[ManualPrecheck] 🔓 Setting isPrecheckInProgress = false");
      setIsPrecheckInProgress(false);
    }
  };

  const handleInlineEmailSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    await handleEmailPrecheck();
  };

  const handleExistingSignIn = async (event: React.FormEvent) => {
    event.preventDefault();

    // Validate before proceeding
    if (!currentEmail || !existingPassword) {
      toast.error("Email and password are required");
      return;
    }

    // Validate email format
    if (!isValidEmail(currentEmail)) {
      toast.error("Invalid email format");
      return;
    }

    try {
      setIsPasswordSubmitting(true);
      trackSignInAttempted("password");
      const tenantOverride = tenantDomain?.trim() || undefined;
      const result = await signIn(
        currentEmail,
        existingPassword,
        tenantOverride,
      );

      if (result.success) {
        trackSignInSucceeded();
        // Already on the correct tenant domain (redirected in precheck), navigate locally
        if (result.requiresWebAuthn) {
          navigate("/admin/webauthn", { replace: true });
        } else {
          navigate(from, { replace: true });
        }
      }
    } catch (error) {
      console.error("Sign in error:", error);
      toast.error("Failed to sign in");
    } finally {
      setIsPasswordSubmitting(false);
    }
  };

  const handleBootstrap = async (event: React.FormEvent) => {
    event.preventDefault();

    // Validate email
    if (!currentEmail) {
      toast.error("Email is required");
      return;
    }

    if (!isValidEmail(currentEmail)) {
      toast.error("Invalid email format");
      return;
    }

    // Validate passwords
    if (!newPassword || !confirmPassword) {
      toast.error("Please enter and confirm your password");
      return;
    }

    if (newPassword !== confirmPassword) {
      toast.error("Passwords do not match");
      return;
    }

    if (newPassword.length < 8) {
      toast.error("Password must be at least 8 characters");
      return;
    }

    // Validate tenant domain
    if (!tenantDomain || !tenantDomain.trim()) {
      toast.error("Workspace domain is required");
      return;
    }

    try {
      trackSignUpStarted();
      const response = await bootstrapAccount({
        email: currentEmail,
        password: newPassword,
        confirm_password: confirmPassword,
        tenant_domain: tenantDomain.trim(),
      }).unwrap();

      // Validate response
      if (!response) {
        toast.error("Invalid response from server");
        return;
      }

      // Store the tenant domain from response
      if (response.tenant_domain) {
        setTenantDomain(response.tenant_domain);
      }

      toast.success(
        "Account created! Check your inbox for the verification code.",
      );
      trackWorkspaceCreated(tenantDomain.trim());
      safeSetFlowStage("otp");
      setTimeLeft(60);
      setCanResend(false);
    } catch (error: any) {
      const message =
        error?.data?.message ||
        error?.data?.error ||
        "Failed to create account";
      toast.error(message);
    }
  };

  const handleVerifyOtp = async () => {
    if (!currentEmail || otp.length !== 6) return;

    setIsVerifyingOtp(true);
    try {
      const result = await verifyOtp({ email: currentEmail, otp });

      if ("data" in result) {
        toast.success("Account verified successfully!");
        trackOtpVerified();
        trackXSignupCompleted(currentEmail);

        // Redirect to tenant domain for login
        if (tenantDomain && window.location.hostname !== tenantDomain) {
          const redirectUrl = buildRedirectUrl(
            tenantDomain,
            currentEmail,
            "/admin/login",
            true,
            {
              flow_stage: "existing",
              tenant_domain: tenantDomain,
            },
          );
          if (redirectUrl) {
            toast.success("Redirecting to login...");
            performRedirect(redirectUrl, 1000);
          } else {
            toast.error(
              "Account verified, but redirect failed. Please login manually.",
            );
            navigate("/admin/login", { replace: true });
          }
        } else {
          // Already on correct domain or no tenant domain
          navigate("/admin/login", {
            replace: true,
            state: { email: currentEmail, verificationComplete: true },
          });
        }
      } else if ("error" in result) {
        const error = result.error as any;
        toast.error(error.data?.message || "Invalid OTP. Please try again.");
        setOtp("");
      }
    } catch (error) {
      console.error("OTP verification error:", error);
      toast.error("Failed to verify OTP");
      setOtp("");
    } finally {
      setIsVerifyingOtp(false);
    }
  };

  const handleResendOtp = async () => {
    if (!currentEmail || isResending) return;

    setIsResending(true);
    try {
      const result = await resendOtpMutation({ email: currentEmail });

      if ("data" in result) {
        toast.success("OTP sent successfully!");
        setTimeLeft(60);
        setCanResend(false);
        setOtp("");
      } else if ("error" in result) {
        const error = result.error as any;
        toast.error(error.data?.message || "Failed to resend OTP");
      }
    } catch (error) {
      console.error("Resend OTP error:", error);
      toast.error("Failed to resend OTP");
    } finally {
      setIsResending(false);
    }
  };

  const handleOtpComplete = (value: string) => {
    setOtp(value);
    if (value.length === 6) {
      setTimeout(() => handleVerifyOtp(), 100);
    }
  };

  // Force light theme on login page
  const { setTheme } = useTheme();
  useEffect(() => {
    setTheme("light");
  }, [setTheme]);

  return (
    <AuthSplitFrame
      className="auth-shell--admin"
      valuePanel={
        <div className="auth-value-panel--dark">
          {/* Main content — vertically centered */}
          <div className="flex-1 flex flex-col justify-center">
            <div className="flex items-center gap-2.5 brand-logo">
              <img
                src={authsecLogoWhite}
                alt="AuthSec"
                className="h-8 w-8 object-contain"
              />
              <span className="text-white font-bold text-2xl tracking-tight">
                AuthSec
              </span>
            </div>
            <AuthValuePanel
              eyebrow="Admin Console"
              title="Agentic auth for AI-native teams"
              subtitle="The fastest way to add authentication to your AI agents, MCP servers, and voice interfaces."
              points={[
                "OAuth, SAML SSO, and social login in minutes",
                "Headless & voice auth for AI agents and MCP servers",
                "Full RBAC with audit logs and policy controls",
                "Agent first. Developer first. Fully open source.",
              ]}
            />
          </div>

          {/* Footer — pinned to bottom */}
          <div className="mt-auto pt-6 flex items-center justify-between mx-3">
            <span className="text-white/35 text-xs">
              © {new Date().getFullYear()} AuthSec. All rights reserved.
            </span>
            <div className="flex items-center gap-3">
              <a
                href="https://x.com/authsec"
                target="_blank"
                rel="noopener noreferrer"
                className="text-white/35 hover:text-white/70 transition-colors"
                aria-label="X (Twitter)"
              >
                <svg
                  className="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path d="M18.244 2.25h3.308l-7.227 8.26 8.502 11.24H16.17l-4.714-6.231-5.401 6.231H2.744l7.73-8.835L1.254 2.25H8.08l4.253 5.622 5.911-5.622Zm-1.161 17.52h1.833L7.084 4.126H5.117z" />
                </svg>
              </a>
              <a
                href="https://github.com/authsec-ai"
                target="_blank"
                rel="noopener noreferrer"
                className="text-white/35 hover:text-white/70 transition-colors"
                aria-label="GitHub"
              >
                <svg
                  className="h-4 w-4"
                  viewBox="0 0 24 24"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path d="M12 2C6.477 2 2 6.484 2 12.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0112 6.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.202 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.943.359.309.678.92.678 1.855 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0022 12.017C22 6.484 17.522 2 12 2z" />
                </svg>
              </a>
            </div>
          </div>
        </div>
      }
    >
      <div className="space-y-4">
        <AuthActionPanel>
          <div className="w-4/5 mx-auto mb-7">
            {flowStage === "idle" && (
              <h2 className="text-[1.6rem] mb-3 text-center font-bold tracking-tight text-slate-900 leading-tight">
                Continue with your email
              </h2>
            )}
            {flowStage === "idle" && (
              <p className="mt-1.5 text-center text-sm text-slate-700">
                Enter your work email to sign in or create an account
              </p>
            )}
          </div>

          {flowStage !== "idle" && (
            <AuthStepHeader
              className="w-4/5 mx-auto text-center"
              title={flowCopy.title}
              subtitle={flowCopy.description}
            />
          )}

          <div className="space-y-6">
            {flowStage === "idle" && (
              <>
                <form className="space-y-4" onSubmit={handleInlineEmailSubmit}>
                  <div className="w-4/5 mx-auto space-y-4">
                    <div className="space-y-2">
                      <Label htmlFor="inline-email">Work email</Label>
                      <Input
                        id="inline-email"
                        type="email"
                        required
                        value={emailInput}
                        onChange={(event) => {
                          clearIdleState();
                          setEmailInput(event.target.value);
                        }}
                        className="h-11 rounded-[12px] px-4 text-[15px]"
                        placeholder="name@company.com"
                      />
                    </div>
                    <Button
                      type="submit"
                      className="h-11 w-full rounded-[12px] font-semibold"
                      disabled={
                        isPrecheckLoading ||
                        (!!ssoProviderName && !ssoProvider) ||
                        (idleNotice?.tone === "info" && !ssoProviderName)
                      }
                    >
                      {isPrecheckLoading ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Checking settings...
                        </>
                      ) : ssoProviderName ? (
                        "Continue with SSO"
                      ) : idleNotice?.tone === "info" ? (
                        "Use your identity provider"
                      ) : (
                        "Continue"
                      )}
                    </Button>
                  </div>
                </form>

                {idleNotice ? (
                  <div
                    className={`auth-inline-alert ${
                      idleNotice.tone === "error"
                        ? "auth-inline-alert--error"
                        : "auth-inline-alert--info"
                    }`}
                    role={idleNotice.tone === "error" ? "alert" : "status"}
                  >
                    {idleNotice.message}
                  </div>
                ) : null}

                <div className="auth-divider-row">
                  <div className="auth-panel-divider" />
                  <span className="auth-divider-copy">
                    Or use your identity provider
                  </span>
                  <div className="auth-panel-divider" />
                </div>

                <div className="flex flex-col items-center gap-3">
                  {orderedUflowProviders.map((provider) => {
                    const isLoading =
                      authenticatingProvider === provider.provider_name;
                    const Icon =
                      provider.provider_name.toLowerCase() === "github"
                        ? GithubIcon
                        : provider.provider_name.toLowerCase() === "google"
                          ? GoogleIcon
                          : provider.provider_name.toLowerCase() === "microsoft"
                            ? MicrosoftIcon
                            : null;

                    return (
                      <Button
                        key={provider.provider_name}
                        type="button"
                        size="lg"
                        className="h-11 w-4/5 mx-auto justify-center gap-3 rounded-[12px] px-4 font-semibold cursor-pointer"
                        variant="outline"
                        onClick={() => handleUFlowProviderAuth(provider)}
                        disabled={authenticatingProvider !== null}
                      >
                        {isLoading ? (
                          <>
                            <Loader2 className="h-5 w-5 animate-spin" />
                            Connecting...
                          </>
                        ) : (
                          <>
                            {Icon ? (
                              <Icon />
                            ) : provider.icon_url ? (
                              <img
                                src={provider.icon_url}
                                alt={provider.display_name}
                                className="h-5 w-5"
                              />
                            ) : null}
                            {getProviderDisplayName(provider)}
                          </>
                        )}
                      </Button>
                    );
                  })}
                </div>
              </>
            )}

            {flowStage === "existing" && (
              <form className="space-y-5" onSubmit={handleExistingSignIn}>
                <div className="w-4/5 mx-auto space-y-2">
                  <Label htmlFor="existing-email">Email</Label>
                  <div className="auth-inline-note text-sm font-medium">
                    {currentEmail}
                  </div>
                </div>
                <div className="w-4/5 mx-auto space-y-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="login-password">Password</Label>
                    <button
                      type="button"
                      onClick={handleForgotPasswordOpen}
                      className="text-xs font-semibold text-slate-600 hover:text-slate-900"
                    >
                      Forgot password?
                    </button>
                  </div>
                  <PasswordInput
                    id="login-password"
                    required
                    value={existingPassword}
                    onChange={(event) =>
                      setExistingPassword(event.target.value)
                    }
                    className="h-11 rounded-xl"
                    placeholder="Enter your password"
                  />
                </div>
                <div className="flex items-center gap-3 pt-2 w-4/5 mx-auto">
                  <Button
                    type="submit"
                    className="h-11 flex-1 rounded-xl font-semibold"
                    disabled={isPasswordSubmitting}
                  >
                    {isPasswordSubmitting && (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    Sign in
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    className="h-11 rounded-xl"
                    onClick={resetFlow}
                  >
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Back
                  </Button>
                </div>
              </form>
            )}

            {flowStage === "register" && (
              <form className="space-y-5" onSubmit={handleBootstrap}>
                <div className="w-4/5 mx-auto space-y-2">
                  <Label htmlFor="register-email">Email</Label>
                  <div className="auth-inline-note text-sm font-medium">
                    {currentEmail}
                  </div>
                </div>
                <div className="w-4/5 mx-auto space-y-2">
                  <Label htmlFor="tenant-domain">Workspace Domain</Label>
                  <Input
                    id="tenant-domain"
                    placeholder="acme-cloud"
                    value={tenantDomain}
                    onChange={(event) => {
                      const value = event.target.value.replace(
                        /[^a-zA-Z0-9]/g,
                        "",
                      );
                      setTenantDomain(value);
                    }}
                    required
                    className={`h-11 rounded-xl ${
                      domainCheckStatus === "available"
                        ? "border-green-500"
                        : domainCheckStatus === "taken" ||
                            domainCheckStatus === "error"
                          ? "border-red-500"
                          : ""
                    }`}
                  />
                  {domainCheckStatus === "checking" && (
                    <div className="flex items-center gap-1.5 text-sm text-slate-500">
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      <span>Checking availability...</span>
                    </div>
                  )}
                  {domainCheckStatus === "available" && (
                    <div className="flex items-center gap-1.5 text-sm text-green-600">
                      <CheckCircle2 className="h-3.5 w-3.5" />
                      <span>Available</span>
                    </div>
                  )}
                  {(domainCheckStatus === "taken" ||
                    domainCheckStatus === "error") &&
                    domainCheckMessage && (
                      <div className="flex items-center gap-1.5 text-sm text-red-600">
                        <XCircle className="h-3.5 w-3.5" />
                        <span>{domainCheckMessage}</span>
                      </div>
                    )}
                </div>
                <div className="w-4/5 mx-auto space-y-2">
                  <Label htmlFor="new-password">Password</Label>
                  <PasswordInput
                    id="new-password"
                    required
                    value={newPassword}
                    onChange={(event) => setNewPassword(event.target.value)}
                    className="h-11 rounded-xl"
                    placeholder="Choose a password"
                  />
                </div>
                <div className="w-4/5 mx-auto space-y-2">
                  <Label htmlFor="confirm-password">Confirm Password</Label>
                  <PasswordInput
                    id="confirm-password"
                    required
                    value={confirmPassword}
                    onChange={(event) => setConfirmPassword(event.target.value)}
                    className="h-11 rounded-xl"
                    placeholder="Confirm your password"
                  />
                </div>
                <div className="flex items-center gap-3 pt-2 w-4/5 mx-auto">
                  <Button
                    type="submit"
                    className="h-11 flex-1 rounded-xl font-semibold"
                    disabled={isBootstrapLoading}
                  >
                    {isBootstrapLoading && (
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    )}
                    Create workspace
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    className="h-11 rounded-xl"
                    onClick={resetFlow}
                  >
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Back
                  </Button>
                </div>
              </form>
            )}

            {flowStage === "otp" && (
              <div className="w-4/5 mx-auto space-y-5">
                <div className="space-y-2">
                  <Label htmlFor="otp-email">Email</Label>
                  <div className="auth-inline-note text-sm font-medium">
                    {currentEmail}
                  </div>
                </div>
                <div className="space-y-4">
                  <Label htmlFor="otp">Verification Code</Label>
                  <OTPInput
                    value={otp}
                    onChange={setOtp}
                    onComplete={handleOtpComplete}
                    disabled={isVerifyingOtp}
                    length={6}
                  />
                  <p className="text-center text-xs text-slate-600">
                    {flowCopy.description}
                  </p>
                </div>

                <div className="text-center space-y-2">
                  {!canResend ? (
                    <div className="flex items-center justify-center gap-2 text-sm text-slate-600">
                      <Clock className="h-4 w-4" />
                      <span>Resend code in {timeLeft}s</span>
                    </div>
                  ) : (
                    <button
                      type="button"
                      onClick={handleResendOtp}
                      disabled={isResending}
                      className="text-sm font-medium text-slate-700 hover:text-slate-950"
                    >
                      {isResending ? "Sending..." : "Resend code"}
                    </button>
                  )}
                </div>

                <div className="flex items-center gap-3 pt-2">
                  <Button
                    onClick={handleVerifyOtp}
                    className="h-11 flex-1 rounded-xl font-semibold"
                    disabled={isVerifyingOtp || otp.length !== 6}
                  >
                    {isVerifyingOtp ? (
                      <>
                        <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                        Verifying...
                      </>
                    ) : (
                      "Verify code"
                    )}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    className="h-11 rounded-xl"
                    onClick={resetFlow}
                  >
                    <ArrowLeft className="mr-2 h-4 w-4" />
                    Use different email
                  </Button>
                </div>
              </div>
            )}
          </div>
        </AuthActionPanel>

        <Dialog
          open={isForgotPasswordOpen}
          onOpenChange={handleForgotPasswordDialogChange}
        >
          <DialogContent className="sm:max-w-lg border-none bg-transparent shadow-none p-0">
            <div className="auth-action-panel space-y-5">
              <DialogHeader className="space-y-2 text-left">
                <DialogTitle className="text-2xl font-semibold text-slate-900">
                  {forgotPasswordStep === "email" && "Reset your password"}
                  {forgotPasswordStep === "otp" && "Verify it’s you"}
                  {forgotPasswordStep === "reset" && "Choose a new password"}
                </DialogTitle>
                <DialogDescription className="text-sm text-slate-600">
                  {forgotPasswordStep === "email" &&
                    "Enter your admin email and we'll send you a secure verification code."}
                  {forgotPasswordStep === "otp" &&
                    "Type the six-digit code we sent to your inbox to continue."}
                  {forgotPasswordStep === "reset" &&
                    "Use at least 10 characters with a mix of upper, lower, number, and symbol."}
                </DialogDescription>
              </DialogHeader>

              {forgotPasswordStep === "email" && (
                <form
                  className="space-y-4"
                  onSubmit={handleForgotPasswordRequest}
                >
                  <div className="space-y-2">
                    <Label htmlFor="forgot-email">Email</Label>
                    <Input
                      id="forgot-email"
                      type="email"
                      placeholder="you@company.com"
                      value={forgotPasswordEmail}
                      onChange={(event) =>
                        setForgotPasswordEmail(event.target.value)
                      }
                      disabled={isForgotPasswordSubmitting}
                      required
                      className="h-11 rounded-xl"
                    />
                  </div>
                  <div className="flex gap-3 pt-1">
                    <Button
                      type="button"
                      variant="outline"
                      className="flex-1 rounded-xl"
                      onClick={resetForgotPasswordState}
                      disabled={isForgotPasswordSubmitting}
                    >
                      Cancel
                    </Button>
                    <Button
                      type="submit"
                      className="flex-1 rounded-xl font-semibold"
                      disabled={
                        isForgotPasswordSubmitting || !forgotPasswordEmail
                      }
                    >
                      {isForgotPasswordSubmitting ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Sending...
                        </>
                      ) : (
                        "Send OTP"
                      )}
                    </Button>
                  </div>
                </form>
              )}

              {forgotPasswordStep === "otp" && (
                <form
                  className="space-y-4"
                  onSubmit={handleForgotPasswordOtpVerification}
                >
                  <div className="space-y-2">
                    <Label htmlFor="forgot-otp">Verification code</Label>
                    <Input
                      id="forgot-otp"
                      type="text"
                      inputMode="numeric"
                      pattern="[0-9]*"
                      placeholder="Enter 6-digit OTP"
                      value={forgotPasswordOtp}
                      onChange={(event) =>
                        setForgotPasswordOtp(
                          event.target.value.replace(/\D/g, "").slice(0, 6),
                        )
                      }
                      disabled={isForgotPasswordSubmitting}
                      required
                      maxLength={6}
                      className="h-11 rounded-xl"
                    />
                    <p className="text-xs text-slate-500">
                      OTP sent to {forgotPasswordEmail || "your email"}
                    </p>
                  </div>
                  <div className="flex gap-3 pt-1">
                    <Button
                      type="button"
                      variant="outline"
                      className="flex-1 rounded-xl"
                      onClick={() => {
                        setForgotPasswordStep("email");
                        setForgotPasswordOtp("");
                      }}
                      disabled={isForgotPasswordSubmitting}
                    >
                      Back
                    </Button>
                    <Button
                      type="submit"
                      className="flex-1 rounded-xl font-semibold"
                      disabled={
                        isForgotPasswordSubmitting ||
                        forgotPasswordOtp.length !== 6
                      }
                    >
                      {isForgotPasswordSubmitting ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Verifying...
                        </>
                      ) : (
                        "Verify OTP"
                      )}
                    </Button>
                  </div>
                </form>
              )}

              {forgotPasswordStep === "reset" && (
                <form
                  className="space-y-4"
                  onSubmit={handleForgotPasswordReset}
                >
                  <div className="space-y-2">
                    <Label htmlFor="forgot-new-password">New password</Label>
                    <PasswordInput
                      id="forgot-new-password"
                      placeholder="Enter new password"
                      value={forgotPasswordNew}
                      onChange={(event) =>
                        setForgotPasswordNew(event.target.value)
                      }
                      disabled={isForgotPasswordSubmitting}
                      minLength={10}
                      required
                      className="h-11 rounded-xl"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="forgot-confirm-password">
                      Confirm password
                    </Label>
                    <PasswordInput
                      id="forgot-confirm-password"
                      placeholder="Confirm new password"
                      value={forgotPasswordConfirm}
                      onChange={(event) =>
                        setForgotPasswordConfirm(event.target.value)
                      }
                      disabled={isForgotPasswordSubmitting}
                      minLength={10}
                      required
                      className="h-11 rounded-xl"
                    />
                  </div>
                  <div className="flex gap-3 pt-1">
                    <Button
                      type="button"
                      variant="outline"
                      className="flex-1 rounded-xl"
                      onClick={() => setForgotPasswordStep("otp")}
                      disabled={isForgotPasswordSubmitting}
                    >
                      Back
                    </Button>
                    <Button
                      type="submit"
                      className="flex-1 rounded-xl font-semibold"
                      disabled={
                        isForgotPasswordSubmitting ||
                        !forgotPasswordNew ||
                        !forgotPasswordConfirm ||
                        forgotPasswordNew !== forgotPasswordConfirm ||
                        forgotPasswordNew.length < 10
                      }
                    >
                      {isForgotPasswordSubmitting ? (
                        <>
                          <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                          Resetting...
                        </>
                      ) : (
                        "Reset password"
                      )}
                    </Button>
                  </div>
                </form>
              )}
            </div>
          </DialogContent>
        </Dialog>

        <div className="auth-card-footer pb-3" aria-label="Trust footer">
          <span>
            <a href="https://authsec.ai/security" target="__blank">
              Security & Compliance
            </a>
          </span>
          <span>
            <a href="https://authsec.ai/terms-and-conditions" target="__blank">
              Terms
            </a>
          </span>
          <span>
            <a href="https://authsec.ai/privacy-policy" target="__blank">
              Privacy
            </a>
          </span>
          <span>
            <a href="https://authsec.ai/contact" target="__blank">
              Support
            </a>
          </span>
        </div>
      </div>

      {uflowCallbackData && (
        <TenantDomainSelectionModal
          open={showDomainModal}
          onOpenChange={setShowDomainModal}
          userData={{
            email: uflowCallbackData.email,
            name: uflowCallbackData.name,
            picture: uflowCallbackData.picture,
            provider: uflowCallbackData.provider,
            provider_user_id: uflowCallbackData.provider_user_id,
          }}
          onSuccess={handleDomainModalSuccess}
        />
      )}
    </AuthSplitFrame>
  );
}

export default AdminLoginHubPage;
