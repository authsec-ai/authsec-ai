import React, { useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { useDispatch } from "react-redux";
import type { FetchBaseQueryError } from "@reduxjs/toolkit/query";
import type { SerializedError } from "@reduxjs/toolkit";
import { Loader2, AlertCircle, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { TenantDomainSelectionModal } from "../components/TenantDomainSelectionModal";
import {
  useExchangeAdminOIDCCodeMutation,
  type AdminOIDCExchangeErrorResponse,
  type AdminOIDCExchangeSuccessResponse,
} from "@/app/api/oidcApi";
import {
  resetAdminWebAuthnState,
  setAuthenticationError,
  setCurrentStep,
  setLoginData,
} from "../slices/adminWebAuthnSlice";
import { toast } from "react-hot-toast";
import { encodeHandoff } from "@/utils/handoff";
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

type ProviderData = NonNullable<AdminOIDCExchangeErrorResponse["provider_data"]>;

export const AdminOIDCCallbackPage: React.FC = () => {
  const navigate = useNavigate();
  const dispatch = useDispatch();
  const [exchangeCode] = useExchangeAdminOIDCCodeMutation();

  const [status, setStatus] = useState<"processing" | "needs_domain" | "error">("processing");
  const [statusMessage, setStatusMessage] = useState("Completing sign-in with your provider...");
  const [providerData, setProviderData] = useState<ProviderData | null>(null);
  const [showDomainModal, setShowDomainModal] = useState(false);

  const urlParams = useMemo(() => new URLSearchParams(window.location.search), []);
  const code = urlParams.get("code") || "";
  const state = urlParams.get("state") || "";
  const errorParam = urlParams.get("error");
  const errorDescription = urlParams.get("error_description");

  useEffect(() => {
    // Clear transient OAuth hint keys once callback processing begins.
    sessionStorage.removeItem("uflow_oauth_type");
    sessionStorage.removeItem("uflow_oauth_provider");
    sessionStorage.removeItem("uflow_oauth_state");

    // Provider redirected with an explicit error
    if (errorParam) {
      setStatus("error");
      setStatusMessage(errorDescription || "Authentication was cancelled or failed.");
      return;
    }

    if (!code || !state) {
      // Backward compatibility for legacy callback payloads delivered as query params.
      const legacyEmail = urlParams.get("email") || "";
      const legacyName = urlParams.get("name") || "";
      const legacyPicture = urlParams.get("picture") || "";
      const legacyProvider = urlParams.get("provider") || "";
      const legacyProviderUserId = urlParams.get("provider_user_id") || "";
      const legacyTenantId = urlParams.get("tenant_id") || "";
      const legacyTenantDomain = urlParams.get("tenant_domain") || "";
      const legacyFirstLogin = urlParams.get("first_login") === "true";
      const legacyNeedsDomain = urlParams.get("needs_domain") === "true";
      const legacySuccess = urlParams.get("success") === "true";
      const legacyMessage = urlParams.get("message");

      if (
        legacyNeedsDomain &&
        legacyEmail &&
        legacyName &&
        legacyProvider &&
        legacyProviderUserId
      ) {
        setProviderData({
          email: legacyEmail,
          name: legacyName,
          picture: legacyPicture,
          provider: legacyProvider,
          provider_user_id: legacyProviderUserId,
        });
        setStatus("needs_domain");
        setStatusMessage(legacyMessage || "Create your workspace to finish signing in.");
        setShowDomainModal(true);
        return;
      }

      if (legacySuccess && legacyEmail) {
        if (!legacyTenantId) {
          setStatus("error");
          setStatusMessage("Callback is missing tenant information. Please try signing in again.");
          return;
        }

        handleExistingUser({
          tenant_id: legacyTenantId,
          email: legacyEmail,
          first_login: legacyFirstLogin,
          otp_required: false,
          mfa_required: true,
          tenant_domain: legacyTenantDomain || undefined,
          client_id: urlParams.get("client_id") || undefined,
        });
        return;
      }

      setStatus("error");
      setStatusMessage("Missing authorization parameters (code/state).");
      return;
    }

    const runExchange = async () => {
      try {
        const response = await exchangeCode({ code, state }).unwrap();
        handleExistingUser(response);
      } catch (err) {
        handleExchangeError(err as FetchBaseQueryError | SerializedError);
      }
    };

    void runExchange();
  }, [code, state, errorParam, errorDescription, exchangeCode]);

  const handleExistingUser = (data: AdminOIDCExchangeSuccessResponse) => {
    const tenantDomain = data.tenant_domain || "";
    const currentHost = window.location.hostname;
    const shouldRedirect =
      tenantDomain &&
      tenantDomain !== currentHost &&
      !currentHost.includes("localhost") &&
      !currentHost.includes("127.0.0.1");

    if (shouldRedirect) {
      const handoffToken = encodeHandoff({
        email: data.email,
        tenant_domain: tenantDomain,
        tenant_id: data.tenant_id,
        first_login: data.first_login,
        target: "webauthn",
      });
      const url = `${window.location.protocol}//${tenantDomain}/admin/webauthn?handoff=${handoffToken}`;
      toast.success("Redirecting to your workspace...");
      window.location.href = url;
      return;
    }

    dispatch(resetAdminWebAuthnState());
    dispatch(setAuthenticationError(null));
    dispatch(
      setLoginData({
        tenantId: data.tenant_id,
        email: data.email,
        isFirstLogin: data.first_login,
      })
    );
    // Router will advance once email + tenantId are present
    dispatch(setCurrentStep("login"));

    toast.success("Signed in with provider. Complete security verification.");
    setStatus("processing");
    setStatusMessage("Redirecting to security verification...");

    navigate(`/admin/webauthn?email=${encodeURIComponent(data.email)}`, { replace: true });
  };

  const handleExchangeError = (error: FetchBaseQueryError | SerializedError) => {
    const fetchError = error as FetchBaseQueryError;
    const data = (fetchError?.data ?? null) as AdminOIDCExchangeErrorResponse | null;

    if (fetchError?.status === 404 && data?.needs_domain && data.provider_data) {
      setProviderData(data.provider_data);
      setStatus("needs_domain");
      setStatusMessage(data.message || "Create your workspace to finish signing in.");
      setShowDomainModal(true);
      return;
    }

    const message =
      (data?.message as string | undefined) ||
      (typeof fetchError?.data === "string" ? fetchError.data : undefined);

    setStatus("error");
    setStatusMessage(message || "Failed to complete sign-in. Please try again.");
  };

  const handleDomainSuccess = (result: { tenant_id: string; client_id: string; tenant_domain: string }) => {
    if (!providerData) return;

    const tenantDomain = result.tenant_domain;
    const currentHost = window.location.hostname;
    const shouldRedirect =
      tenantDomain &&
      tenantDomain !== currentHost &&
      !currentHost.includes("localhost") &&
      !currentHost.includes("127.0.0.1");

    if (shouldRedirect) {
      const handoffToken = encodeHandoff({
        email: providerData.email,
        tenant_domain: tenantDomain,
        tenant_id: result.tenant_id,
        first_login: true,
        target: "webauthn",
      });
      const url = `${window.location.protocol}//${tenantDomain}/admin/webauthn?handoff=${handoffToken}`;
      toast.success("Redirecting to your new workspace...");
      window.location.href = url;
      return;
    }

    toast.success("Workspace created. Complete security verification.");
    setShowDomainModal(false);
    setStatus("processing");
    setStatusMessage("Redirecting to security verification...");

    dispatch(resetAdminWebAuthnState());
    dispatch(setAuthenticationError(null));
    dispatch(
      setLoginData({
        tenantId: result.tenant_id,
        email: providerData.email,
        isFirstLogin: true,
      })
    );
    dispatch(setCurrentStep("login"));

    navigate(`/admin/webauthn?email=${encodeURIComponent(providerData.email)}`, { replace: true });
  };

  const getStatusTitle = () => {
    if (status === "needs_domain") return "Set your workspace domain";
    if (status === "error") return "Sign-in could not be completed";
    return "Completing secure sign-in";
  };

  const getStatusTone = () => {
    if (status === "needs_domain") return "auth-status-block--success";
    if (status === "error") return "auth-status-block--error";
    return "auth-status-block--processing";
  };

  return (
    <AuthSplitFrame
      valuePanel={
        <AuthValuePanel
          eyebrow="Admin Callback"
          title="Completing provider sign-in."
          subtitle="We’re validating your provider response and preparing secure workspace authentication."
          points={[
            "Provider identity is exchanged for tenant-safe context.",
            "Cross-tenant handoff is signed before redirect.",
            "MFA routing is preserved before admin access.",
          ]}
        />
      }
    >
      <AuthActionPanel className="space-y-5">
        <AuthStepHeader
          align="center"
          title={getStatusTitle()}
          subtitle={statusMessage}
        />
        <div className={`auth-status-block ${getStatusTone()}`}>
          {status === "processing" && <Loader2 className="h-8 w-8 animate-spin text-slate-800" />}

          {status === "needs_domain" && (
            <>
              <CheckCircle2 className="h-8 w-8 text-emerald-500" />
              <p className="text-sm text-slate-700 text-center max-w-[32ch]">
                We just need a workspace domain to finish setting up your account.
              </p>
            </>
          )}

          {status === "error" && (
            <>
              <AlertCircle className="h-8 w-8 text-red-500" />
              <Button variant="outline" onClick={() => navigate("/admin/login")}>
                Back to login
              </Button>
            </>
          )}
        </div>
      </AuthActionPanel>

      {providerData && (
        <TenantDomainSelectionModal
          open={showDomainModal}
          onOpenChange={setShowDomainModal}
          userData={providerData}
          onSuccess={handleDomainSuccess}
        />
      )}
    </AuthSplitFrame>
  );
};

export default AdminOIDCCallbackPage;
