import React, { useEffect, useState } from "react";
import { useDispatch, useSelector } from "react-redux";
import { motion } from "framer-motion";
import { Button } from "../../components/ui/button";
import {
  IconCircleCheck,
  IconAlertTriangle,
  IconCopy,
  IconCheck,
} from "@tabler/icons-react";
import {
  useHandleCallbackMutation,
  useExchangeCodeForTokensMutation,
  useSendTokenToOIDCLoginMutation,
  useSamlLoginMutation,
} from "../../app/api/oidcApi";
import { setLoginData, setCurrentStep } from "../slices/oidcWebAuthnSlice";
import type { RootState } from "../../app/store";
import { OIDCWebAuthnRouter } from "./OIDCWebAuthnRouter";
// OIDC WebAuthn functionality using dedicated OIDC context
import { EndUserAuthProvider, useEndUserAuth } from "../context/EndUserAuthContext";
import { decodeJWT } from "../../utils/jwt";
// Device Management
import { DeviceManagementPanel } from "./device-management";
import config from '../../config';
import { AuthSplitFrame } from "../components/AuthSplitFrame";
import { AuthValuePanel } from "../components/AuthValuePanel";
import { AuthActionPanel } from "../components/AuthActionPanel";
import { AuthStepHeader } from "../components/AuthStepHeader";

const OIDCCallbackPageInner: React.FC = () => {
  // No longer extract provider from URL path - it will come from state
  const urlParams = new URLSearchParams(window.location.search);
  const dispatch = useDispatch();
  // Get client_id from Redux state (set during initial login page load)
  const reduxClientId = useSelector((state: RootState) => state.oidcWebAuthn.clientId);
  const { executeCallback, captureClientId } = useEndUserAuth();

  // RTK Query hooks
  const [handleCallback] = useHandleCallbackMutation();
  const [exchangeCodeForTokens] = useExchangeCodeForTokensMutation();
  const [sendTokenToOIDCLogin] = useSendTokenToOIDCLoginMutation();
  const [samlLogin] = useSamlLoginMutation();

  const [status, setStatus] = useState<"processing" | "success" | "error" | "webauthn">(
    "processing",
  );
  const [message, setMessage] = useState<string>("Processing authentication...");
  const [debugInfo, setDebugInfo] = useState<any>(null);
  const [tokenResponse, setTokenResponse] = useState<any>(null);
  const [webauthnCallbackToken, setWebauthnCallbackToken] = useState<string | null>(null);
  const [copyStatus, setCopyStatus] = useState<"idle" | "copied">("idle");
  const [webauthnData, setWebauthnData] = useState<{
    tenantId: string;
    email: string;
    firstLogin: boolean;
  } | null>(null);
  const [isCustomLoginWebAuthn, setIsCustomLoginWebAuthn] = useState(false);
  const [isSamlWebAuthn, setIsSamlWebAuthn] = useState(false);
  const [detectedProvider, setDetectedProvider] = useState<string | null>(null);
  const [samlRedirectTo, setSamlRedirectTo] = useState<string | null>(null);

  useEffect(() => {
    console.log("Universal CallbackPage mounted");
    console.log("URL params:", Object.fromEntries(urlParams.entries()));
    console.log("Full URL:", window.location.href);

    processCallback();
  }, []);

  // Helper function to extract provider from state
  const extractProviderFromState = (state: string): string | null => {
    try {
      // Check if state looks like valid base64 (contains only valid chars and proper padding)
      if (!/^[A-Za-z0-9+/=]+$/.test(state) || state.length % 4 !== 0) {
        console.warn("State does not appear to be base64-encoded. Skipping extraction.");
        return null;
      }
      const stateBytes = atob(state.replace(/-/g, "+").replace(/_/g, "/"));
      const stateData = JSON.parse(stateBytes);
      return stateData.provider || null;
    } catch (error) {
      console.error("Failed to parse state for provider:", error);
      return null;
    }
  };

  const processCallback = async () => {
    try {
      // Admin OAuth callbacks should always be processed by the dedicated admin callback scene.
      // This protects against backend/provider configs still pointing to /oidc/auth/callback.
      const uflowOAuthType = sessionStorage.getItem("uflow_oauth_type");
      if (uflowOAuthType === "admin") {
        const adminCallbackUrl = `/authsec/uflow/oidc/callback${window.location.search || ""}`;
        console.log(
          "↪️ Admin OAuth callback detected on end-user route. Redirecting to admin callback:",
          adminCallbackUrl,
        );
        window.location.replace(adminCallbackUrl);
        return;
      }

      // Check if this is a UFlow OAuth callback
      const isUFlowOAuth = uflowOAuthType === "true" || uflowOAuthType === "admin";

      if (isUFlowOAuth) {
        const isAdminFlow = uflowOAuthType === "admin";
        console.log(
          `🔐 UFlow OAuth callback detected (${isAdminFlow ? "Admin" : "End-User"} flow)`,
        );

        // Parse callback URL to extract user data
        // The URL should contain query parameters with user data
        const email = urlParams.get("email");
        const name = urlParams.get("name");
        const picture = urlParams.get("picture");
        const provider = urlParams.get("provider");
        const providerUserId = urlParams.get("provider_user_id");
        const needsDomain = urlParams.get("needs_domain") === "true";
        const success = urlParams.get("success") === "true";
        const message = urlParams.get("message");

        // Check for errors
        const error = urlParams.get("error");
        if (error) {
          setStatus("error");
          setMessage(`OAuth authentication failed: ${message || error}`);
          sessionStorage.removeItem("uflow_oauth_type");
          sessionStorage.removeItem("uflow_oauth_provider");
          sessionStorage.removeItem("uflow_oauth_state");
          return;
        }

        if (needsDomain && email && name && provider && providerUserId) {
          console.log("📋 New UFlow OAuth user - needs domain selection");

          // Store user data in sessionStorage
          sessionStorage.setItem("uflow_user_email", email);
          sessionStorage.setItem("uflow_user_name", name);
          sessionStorage.setItem("uflow_user_picture", picture || "");
          sessionStorage.setItem("uflow_provider", provider);
          sessionStorage.setItem("uflow_provider_user_id", providerUserId);

          // Clean up OAuth session data
          sessionStorage.removeItem("uflow_oauth_type");
          sessionStorage.removeItem("uflow_oauth_provider");
          sessionStorage.removeItem("uflow_oauth_state");

          // Redirect back to login page to show domain modal
          setStatus("processing");
          setMessage("Redirecting to complete registration...");

          const loginPath = isAdminFlow ? "/admin/login" : "/oidc/login";

          setTimeout(() => {
            window.location.href = `${loginPath}?show_domain_modal=true`;
          }, 1000);

          return;
        } else if (success && email) {
          console.log("✅ Existing UFlow OAuth user - proceeding to login");

          // Extract tenant and client info
          const tenantId = urlParams.get("tenant_id");
          const tenantDomain = urlParams.get("tenant_domain");

          if (isAdminFlow) {
            // Admin flow - redirect to admin WebAuthn
            console.log("🔐 Admin OAuth flow - redirecting to admin WebAuthn");

            // Clean up OAuth session data
            sessionStorage.removeItem("uflow_oauth_type");
            sessionStorage.removeItem("uflow_oauth_provider");
            sessionStorage.removeItem("uflow_oauth_state");

            // Store login data for admin flow
            sessionStorage.setItem("admin_oauth_email", email);
            sessionStorage.setItem("admin_oauth_tenant", tenantDomain || "");

            setStatus("processing");
            setMessage("OAuth authentication successful. Redirecting to complete login...");

            setTimeout(() => {
              // Redirect to admin WebAuthn page
              window.location.href = `/admin/webauthn?email=${encodeURIComponent(email)}`;
            }, 1000);

            return;
          } else {
            // End-user flow - WebAuthn authentication
            const clientId = urlParams.get("client_id");

            if (!tenantId || !clientId) {
              setStatus("error");
              setMessage("Missing tenant or client information from OAuth callback");
              sessionStorage.removeItem("uflow_oauth_type");
              sessionStorage.removeItem("uflow_oauth_provider");
              sessionStorage.removeItem("uflow_oauth_state");
              return;
            }

            // Clean up OAuth session data
            sessionStorage.removeItem("uflow_oauth_type");
            sessionStorage.removeItem("uflow_oauth_provider");
            sessionStorage.removeItem("uflow_oauth_state");

            // Initialize WebAuthn flow
            const webauthnFlowData = {
              tenantId,
              email,
              firstLogin: false, // Existing user
            };

            setMessage("OAuth authentication successful. Initiating security authentication...");
            setWebauthnData(webauthnFlowData);
            setStatus("webauthn");

            // Initialize WebAuthn flow in Redux
            dispatch(
              setLoginData({
                tenantId: webauthnFlowData.tenantId,
                email: webauthnFlowData.email,
                isFirstLogin: webauthnFlowData.firstLogin,
                clientId: clientId,
              }),
            );

            // Set to authentication step for existing users
            dispatch(setCurrentStep("authentication"));

            setDebugInfo({
              flow_type: "UFlow OAuth - Existing User",
              email,
              tenant_id: tenantId,
              client_id: clientId,
            });

            return;
          }
        } else {
          setStatus("error");
          setMessage("Invalid OAuth callback response");
          sessionStorage.removeItem("uflow_oauth_type");
          sessionStorage.removeItem("uflow_oauth_provider");
          sessionStorage.removeItem("uflow_oauth_state");
          return;
        }
      }

      // Check if this is a WebAuthn completion from custom login
      const webauthnComplete = urlParams.get("webauthn_complete");
      if (webauthnComplete === "true") {
        // Handle WebAuthn completion from custom login
        const storedToken = sessionStorage.getItem("webauthn_callback_token");
        const storedEmail = sessionStorage.getItem("webauthn_callback_email");

        // Check if this is a SAML + WebAuthn flow from OIDCLoginPage
        const isSamlPostWebAuthn = sessionStorage.getItem("saml_post_webauthn") === "true";

        if (isSamlPostWebAuthn) {
          console.log("🔐 SAML + WebAuthn flow detected from OIDCLoginPage");

          // Get stored SAML parameters
          const samlLoginChallenge = sessionStorage.getItem("saml_login_challenge");
          const samlClientId = sessionStorage.getItem("saml_client_id");
          const samlUserEmail = sessionStorage.getItem("saml_user_email");

          console.log("📋 SAML post-WebAuthn parameters:", {
            login_challenge: samlLoginChallenge,
            client_id: samlClientId,
            email: samlUserEmail,
            token_received: !!storedToken,
          });

          if (storedToken && samlLoginChallenge) {
            setWebauthnCallbackToken(storedToken);
            setStatus("processing");
            setMessage("SAML authentication with WebAuthn completed! Finalizing login...");

            setDebugInfo({
              flow_type: "SAML + WebAuthn from OIDCLoginPage",
              email: samlUserEmail,
              client_id: samlClientId,
              login_challenge: samlLoginChallenge
                ? `${samlLoginChallenge.substring(0, 20)}...`
                : null,
              token_received: true,
              token_preview: `${storedToken.substring(0, 20)}...`,
              webauthn_callback_response: { token: storedToken },
            });

            // Clean up SAML and WebAuthn session storage
            sessionStorage.removeItem("webauthn_callback_token");
            sessionStorage.removeItem("webauthn_callback_email");
            sessionStorage.removeItem("saml_post_webauthn");
            sessionStorage.removeItem("saml_login_challenge");
            sessionStorage.removeItem("saml_client_id");
            sessionStorage.removeItem("saml_user_email");
            sessionStorage.removeItem("saml_tenant_id");
            sessionStorage.removeItem("saml_user_id");
            sessionStorage.removeItem("saml_provider");
            sessionStorage.removeItem("saml_provider_id");
            sessionStorage.removeItem("saml_project_id");

            // For SAML flow, we need to redirect back to continue the OAuth flow
            // Since we have the webauthn token, the backend should recognize the authenticated session
            // and issue the OAuth code. Redirect to a SAML continuation endpoint or back to login page.
            setTimeout(() => {
              const continuationUrl = `/oidc/login?login_challenge=${samlLoginChallenge}&saml_webauthn_complete=true`;
              console.log("🔄 Redirecting to continue SAML OAuth flow:", continuationUrl);
              window.location.href = continuationUrl;
            }, 1500);

            return;
          } else {
            console.error("❌ SAML post-WebAuthn flow missing required data:", {
              has_token: !!storedToken,
              has_login_challenge: !!samlLoginChallenge,
            });

            // Clean up even on error
            sessionStorage.removeItem("webauthn_callback_token");
            sessionStorage.removeItem("webauthn_callback_email");
            sessionStorage.removeItem("saml_post_webauthn");
            sessionStorage.removeItem("saml_login_challenge");
            sessionStorage.removeItem("saml_client_id");
            sessionStorage.removeItem("saml_user_email");
            sessionStorage.removeItem("saml_tenant_id");
            sessionStorage.removeItem("saml_user_id");
            sessionStorage.removeItem("saml_provider");
            sessionStorage.removeItem("saml_provider_id");
            sessionStorage.removeItem("saml_project_id");

            setStatus("error");
            setMessage("SAML WebAuthn flow missing required parameters. Please try again.");
            return;
          }
        }

        // Regular custom login WebAuthn flow (not SAML)
        if (storedToken) {
          setWebauthnCallbackToken(storedToken);
          setStatus("success");
          setIsCustomLoginWebAuthn(true);
          setMessage("Custom login WebAuthn authentication completed! Token received.");

          setDebugInfo({
            flow_type: "Custom Login WebAuthn",
            email: storedEmail,
            token_received: true,
            token_preview: `${storedToken.substring(0, 20)}...`,
            webauthn_callback_response: { token: storedToken },
          });

          // Clean up stored data
          sessionStorage.removeItem("webauthn_callback_token");
          sessionStorage.removeItem("webauthn_callback_email");

          return;
        } else {
          setStatus("error");
          setMessage("WebAuthn completion detected but no token found in storage");
          return;
        }
      }

      // Step 1: Extract OAuth response parameters
      const code = urlParams.get("code");
      const state = urlParams.get("state");
      const error = urlParams.get("error");
      const errorDescription = urlParams.get("error_description");

      // Step 2: Extract provider from state parameter ONLY if not a Hydra code
      let provider: string | null = null;
      const isHydraCode = code?.startsWith("ory_ac_") ?? false;
      if (state && !isHydraCode) {
        provider = extractProviderFromState(state);
        setDetectedProvider(provider);
      }

      console.log("Universal OAuth callback parameters:", {
        code: code ? `${code.substring(0, 10)}...` : null,
        state: state ? `${state.substring(0, 10)}...` : null,
        error,
        errorDescription,
        detectedProvider: provider,
      });

      // Check for OAuth errors first
      if (error) {
        setStatus("error");
        setMessage(`Authentication failed: ${errorDescription || error}`);
        setDebugInfo({ error, errorDescription, provider });
        return;
      }

      // Validate required parameters
      if (!code || !state) {
        setStatus("error");
        setMessage("Missing required parameters from OAuth provider");
        setDebugInfo({ code: !!code, state: !!state, provider, url: window.location.href });
        return;
      }

      if (!isHydraCode && !provider) {
        setStatus("error");
        setMessage("Could not determine provider from state parameter");
        setDebugInfo({ state, provider });
        return;
      }

      // Step 3: Validate state parameter against stored value (skip for Hydra codes)
      const storedState = sessionStorage.getItem("oauth_state");
      const storedProvider = sessionStorage.getItem("oauth_provider");
      const storedLoginChallenge = sessionStorage.getItem("login_challenge");

      if (isHydraCode) {
        setMessage("Exchanging Hydra authorization code for tokens...");

        // Use universal callback URL
        const redirectUri = `${window.location.origin}/oidc/auth/callback`;

        if (!storedLoginChallenge) {
          setStatus("error");
          setMessage("Missing login challenge - cannot exchange token");
          setDebugInfo({ hasLoginChallenge: false });
          return;
        }

        try {
          const response = await exchangeCodeForTokens({
            login_challenge: storedLoginChallenge,
            code,
            state,
            provider: provider || "hydra",
            redirect_uri: redirectUri,
          }).unwrap();

          if (response) {
            setStatus("processing");
            setMessage("Token exchange successful. Finalizing login...");

            const normalizedTokens = response.tokens ?? {
              access_token: (response as any).access_token,
              token_type: (response as any).token_type || "Bearer",
              expires_in: (response as any).expires_in || 3600,
              refresh_token: (response as any).refresh_token,
            };
            setTokenResponse(normalizedTokens);

            // Send access token to OIDC login endpoint
            const accessToken = normalizedTokens?.access_token;
            if (accessToken) {
              let oidcResponse: any | undefined;
              try {
                oidcResponse = await sendTokenToOIDCLogin({
                  access_token: accessToken,
                  expires_in: normalizedTokens?.expires_in || 3600,
                }).unwrap();
              } catch (e) {
                console.warn("OIDC login endpoint returned no body or non-standard shape");
              }

              const oidcData: any = oidcResponse
                ? ((oidcResponse as any).data ?? oidcResponse)
                : undefined;
              let tenantId: string | undefined = oidcData?.tenant_id;
              let email: string | undefined = oidcData?.email;
              const firstLogin = Boolean(oidcData?.first_login);

              // Extract tenant/email from exchanged access token, prioritize Redux client_id
                let clientId: string | null = reduxClientId; // Use Redux state first
                const decoded: any = response.tokens?.access_token ? decodeJWT(response.tokens.access_token) : null;
                if (decoded) {
                tenantId = decoded?.ext?.tenant_id || decoded?.tenant_id || tenantId;
                email = decoded?.ext?.email || decoded?.email_id || email;

                // Only extract client_id from JWT as fallback if not in Redux
                if (!clientId) {
                  clientId = decoded?.client_id || null;
                  // Remove "-main-client" suffix if present (consistent with API transform)
                  if (clientId && clientId.endsWith("-main-client")) {
                    clientId = clientId.replace("-main-client", "");
                  }
                }
              }

              if (tenantId && email) {
                const webauthnFlowData = { tenantId, email, firstLogin };
                setMessage("OIDC login processed. Initiating MFA (WebAuthn/TOTP)...");
                setWebauthnData(webauthnFlowData);
                setStatus("webauthn");

                // Ensure client_id is captured (from Redux or JWT fallback)
                if (clientId && clientId !== reduxClientId) {
                  captureClientId(clientId);
                }

                // Initialize WebAuthn flow in Redux
                dispatch(
                  setLoginData({
                    tenantId: webauthnFlowData.tenantId,
                    email: webauthnFlowData.email,
                    isFirstLogin: webauthnFlowData.firstLogin,
                    clientId: clientId || undefined,
                  }),
                );

                // Set appropriate WebAuthn step
                if (webauthnFlowData.firstLogin) {
                  dispatch(setCurrentStep("mfa_selection"));
                } else {
                  dispatch(setCurrentStep("authentication"));
                }

                setDebugInfo({
                  codeType: "Hydra Authorization Code",
                  code: `${code.substring(0, 20)}...`,
                  state,
                  provider,
                  tokens: normalizedTokens,
                  oidc_login_response: oidcResponse,
                  webauthn_flow: "initiated",
                  webauthn_data: webauthnFlowData,
                  client_id: clientId,
                  client_id_source: reduxClientId ? "Redux state" : "JWT fallback",
                  jwt_decoded: decoded,
                });
              } else {
                setStatus("error");
                setMessage("OIDC login did not return user info and token lacks required claims");
                setDebugInfo({
                  reason: "missing_tenant_or_email",
                  oidc_login_response: oidcResponse,
                  access_token_preview: `${accessToken.substring(0, 20)}...`,
                });
              }
            }

            console.log("Token exchange successful:", normalizedTokens);
            sessionStorage.removeItem("login_challenge");
          }
        } catch (exchangeError) {
          console.error("Token exchange error:", exchangeError);
          setStatus("error");
          setMessage("Token exchange failed");
          setDebugInfo({
            exchangeError: exchangeError instanceof Error ? exchangeError.message : "Unknown error",
          });
          sessionStorage.removeItem("login_challenge");
        }
        return;
      } else {
        console.log("State validation:", {
          storedState: storedState ? `${storedState.substring(0, 10)}...` : null,
          receivedState: state ? `${state.substring(0, 10)}...` : null,
          stateMatch: storedState === state,
          providerMatch: storedProvider === provider,
          hasLoginChallenge: !!storedLoginChallenge,
        });

        if (!storedState || storedState !== state) {
          setStatus("error");
          setMessage("Invalid state parameter - possible security issue");
          setDebugInfo({
            storedState: storedState ? `${storedState.substring(0, 10)}...` : null,
            receivedState: state ? `${state.substring(0, 10)}...` : null,
            stateMatch: storedState === state,
          });
          return;
        }

        if (!storedProvider || storedProvider !== provider) {
          setStatus("error");
          setMessage("Provider mismatch - possible security issue");
          setDebugInfo({
            storedProvider,
            receivedProvider: provider,
            providerMatch: storedProvider === provider,
          });
          return;
        }

        if (!storedLoginChallenge) {
          setStatus("error");
          setMessage("Missing login challenge - authentication flow error");
          setDebugInfo({ hasLoginChallenge: false });
          return;
        }

        // Clear stored session data
        sessionStorage.removeItem("oauth_state");
        sessionStorage.removeItem("oauth_provider");

        setMessage("Validating provider response with server...");

        try {
          // Step 5: Send callback data to API server for processing
          // Note: No provider in URL path now, provider comes from state
          const response = await handleCallback({
            code,
            state,
            error: error || undefined,
          }).unwrap();

          console.log("API callback response:", response);

          if (response) {
            setStatus("processing");
            setMessage("Provider credentials accepted.");
            setDebugInfo({
              redirectTo: response.redirect_to,
              userInfo: response.user_info,
            });

            // Defensive check: ensure we're not redirecting back to the login page (infinite loop prevention)
            const currentOrigin = window.location.origin;
            if (!response.redirect_to) {
              setStatus("error");
              setMessage("Authentication flow error: No redirect URL provided by backend.");
              setDebugInfo({
                ...debugInfo,
                error: "Missing redirect_to in response",
              });
              return;
            }
            const redirectUrl = new URL(response.redirect_to, currentOrigin);
            const isLoopingBack =
              redirectUrl.pathname.includes("/oidc/login") ||
              redirectUrl.pathname.includes("/oidc/auth/callback");

            if (isLoopingBack) {
              console.error(
                "⚠️ Potential infinite loop detected! Backend returned redirect_to pointing back to OIDC flow:",
                response.redirect_to,
              );
              setStatus("error");
              setMessage(
                "Authentication flow error: Backend returned invalid redirect URL. Please contact support.",
              );
              setDebugInfo({
                ...debugInfo,
                error: "Infinite loop prevented",
                redirect_to: response.redirect_to,
                detected_loop: true,
              });
              return;
            }

            // Check if this is a SAML provider that needs WebAuthn check
            const isSamlProvider =
              storedProvider?.toLowerCase().includes("saml") ||
              provider?.toLowerCase().includes("saml");

            if (isSamlProvider) {
              console.log("🔐 SAML provider detected, checking for WebAuthn requirements...");
              setMessage("Checking authentication requirements...");

              try {
                // Get SAML parameters from sessionStorage (stored by OIDCLoginPage from URL params)
                const samlStoredClientId = sessionStorage.getItem("saml_client_id");
                const samlStoredEmail = sessionStorage.getItem("saml_user_email");

                // Fallback to other sources if not in sessionStorage
                const clientIdForSaml =
                  samlStoredClientId || reduxClientId || sessionStorage.getItem("client_id");
                const userEmail = samlStoredEmail || response.user_info?.email;

                console.log("📋 SAML parameters retrieved:", {
                  client_id_from_saml_storage: samlStoredClientId,
                  email_from_saml_storage: samlStoredEmail,
                  client_id_final: clientIdForSaml,
                  email_final: userEmail,
                  fallback_to_redux: !samlStoredClientId && reduxClientId,
                  fallback_to_response: !samlStoredEmail && response.user_info?.email,
                });

                if (!clientIdForSaml) {
                  console.error(
                    "❌ Cannot proceed with SAML WebAuthn check: client_id not available",
                  );
                  // Clean up SAML parameters
                  sessionStorage.removeItem("saml_client_id");
                  sessionStorage.removeItem("saml_user_email");
                  sessionStorage.removeItem("saml_tenant_id");
                  sessionStorage.removeItem("saml_user_id");
                  sessionStorage.removeItem("saml_provider");
                  sessionStorage.removeItem("saml_provider_id");
                  sessionStorage.removeItem("saml_project_id");
                  sessionStorage.removeItem("saml_success");
                  // Fallback to direct redirect if client_id is missing
                  setTimeout(() => {
                    console.log(
                      "Redirecting to Hydra (no client_id for WebAuthn check):",
                      response.redirect_to,
                    );
                    if (response.redirect_to) {
                    window.location.href = response.redirect_to;
                  }
                  }, 1500);
                  return;
                }

                if (!userEmail) {
                  console.error("❌ Cannot proceed with SAML WebAuthn check: email not available");
                  // Clean up SAML parameters
                  sessionStorage.removeItem("saml_client_id");
                  sessionStorage.removeItem("saml_user_email");
                  sessionStorage.removeItem("saml_tenant_id");
                  sessionStorage.removeItem("saml_user_id");
                  sessionStorage.removeItem("saml_provider");
                  sessionStorage.removeItem("saml_provider_id");
                  sessionStorage.removeItem("saml_project_id");
                  sessionStorage.removeItem("saml_success");
                  // Fallback to direct redirect if email is missing
                  setTimeout(() => {
                    console.log(
                      "Redirecting to Hydra (no email for WebAuthn check):",
                      response.redirect_to,
                    );
                    if (response.redirect_to) {
                    window.location.href = response.redirect_to;
                  }
                  }, 1500);
                  return;
                }

                console.log(
                  `🔍 Calling SAML login check for email: ${userEmail}, client_id: ${clientIdForSaml}`,
                );

                // Call SAML login check API
                const samlLoginResponse = await samlLogin({
                  client_id: clientIdForSaml,
                  email: userEmail,
                }).unwrap();

                console.log("✅ SAML login check response:", samlLoginResponse);

                // Check if WebAuthn is needed based on first_login
                if (samlLoginResponse.first_login !== undefined) {
                  // Store the redirect_to URL for later use after WebAuthn
                  setSamlRedirectTo(response.redirect_to);
                  setIsSamlWebAuthn(true);

                  // Set up WebAuthn flow data
                  const webauthnFlowData = {
                    tenantId: samlLoginResponse.tenant_id,
                    email: samlLoginResponse.email,
                    firstLogin: samlLoginResponse.first_login,
                  };

                  setMessage("Initiating multi-factor authentication...");
                  setWebauthnData(webauthnFlowData);
                  setStatus("webauthn");

                  // Ensure client_id is captured
                  if (clientIdForSaml !== reduxClientId) {
                    captureClientId(clientIdForSaml);
                  }

                  // Initialize WebAuthn flow in Redux
                  dispatch(
                    setLoginData({
                      tenantId: webauthnFlowData.tenantId,
                      email: webauthnFlowData.email,
                      isFirstLogin: webauthnFlowData.firstLogin,
                      clientId: clientIdForSaml,
                    }),
                  );

                  // Set appropriate WebAuthn step
                  if (webauthnFlowData.firstLogin) {
                    console.log("🆕 First-time SAML user → MFA setup");
                    dispatch(setCurrentStep("mfa_selection"));
                  } else {
                    console.log("🔑 Returning SAML user → WebAuthn authentication");
                    dispatch(setCurrentStep("authentication"));
                  }

                  return; // Don't redirect yet, wait for WebAuthn completion
                } else {
                  console.log(
                    "⚠️ SAML login response missing first_login field, proceeding without WebAuthn",
                  );
                  // Clean up SAML parameters
                  sessionStorage.removeItem("saml_client_id");
                  sessionStorage.removeItem("saml_user_email");
                  sessionStorage.removeItem("saml_tenant_id");
                  sessionStorage.removeItem("saml_user_id");
                  sessionStorage.removeItem("saml_provider");
                  sessionStorage.removeItem("saml_provider_id");
                  sessionStorage.removeItem("saml_project_id");
                  sessionStorage.removeItem("saml_success");
                  // Fallback to direct redirect if response is incomplete
                  setTimeout(() => {
                    console.log("Redirecting to Hydra:", response.redirect_to);
                    if (response.redirect_to) {
                    window.location.href = response.redirect_to;
                  }
                  }, 1500);
                }
              } catch (samlLoginError) {
                console.error("❌ SAML login check failed:", samlLoginError);
                // Clean up SAML parameters
                sessionStorage.removeItem("saml_client_id");
                sessionStorage.removeItem("saml_user_email");
                sessionStorage.removeItem("saml_tenant_id");
                sessionStorage.removeItem("saml_user_id");
                sessionStorage.removeItem("saml_provider");
                sessionStorage.removeItem("saml_provider_id");
                sessionStorage.removeItem("saml_project_id");
                sessionStorage.removeItem("saml_success");
                // Fallback: proceed with redirect even if SAML login check fails
                setMessage("Provider credentials accepted. Redirecting to authorization server...");
                setTimeout(() => {
                  console.log("Redirecting to Hydra (SAML check failed):", response.redirect_to);
                  if (response.redirect_to) {
                    window.location.href = response.redirect_to;
                  }
                }, 1500);
              }
            } else {
              // Not a SAML provider or no email - proceed with normal redirect
              setMessage("Provider credentials accepted. Redirecting to authorization server...");
              setTimeout(() => {
                console.log("Redirecting to Hydra:", response.redirect_to);
                if (response.redirect_to) {
                  window.location.href = response.redirect_to;
                }
              }, 1500);
            }
          }
        } catch (err) {
          console.error("Callback processing error:", err);
          setStatus("error");
          setMessage(
            err instanceof Error ? err.message : "Failed to process authentication callback",
          );
          setDebugInfo({ error: err });
        }
      }
    } catch (outerErr) {
      console.error("Callback processing error (outer):", outerErr);
      setStatus("error");
      setMessage(
        outerErr instanceof Error ? outerErr.message : "Failed to process authentication callback",
      );
      setDebugInfo({ error: outerErr });
    }
  };

  const handleRetry = () => {
    setStatus("processing");
    setMessage("Retrying authentication...");
    setDebugInfo(null);
    setTokenResponse(null);
    processCallback();
  };

  const handleGoBack = () => {
    // Try to go back to login page with the original challenge if available
    const storedChallenge = sessionStorage.getItem("login_challenge");
    if (storedChallenge) {
      const loginUrl = `/oidc/login?login_challenge=${storedChallenge}`;
      console.log("Going back to login:", loginUrl);
      window.location.href = loginUrl;
    } else {
      window.history.back();
    }
  };

  // Copy webauthn-callback token to clipboard
  const copyTokenToClipboard = async () => {
    if (webauthnCallbackToken) {
      try {
        await navigator.clipboard.writeText(webauthnCallbackToken);
        setCopyStatus("copied");
        setTimeout(() => setCopyStatus("idle"), 2000);
      } catch (err) {
        console.error("Failed to copy to clipboard:", err);
        // Fallback for older browsers
        const textArea = document.createElement("textarea");
        textArea.value = webauthnCallbackToken;
        document.body.appendChild(textArea);
        textArea.select();
        document.execCommand("copy");
        document.body.removeChild(textArea);
        setCopyStatus("copied");
        setTimeout(() => setCopyStatus("idle"), 2000);
      }
    }
  };

  // Handle WebAuthn completion
  const handleWebAuthnComplete = async () => {
    console.log("WebAuthn flow completed successfully");
    setMessage("Calling webauthn-callback to get final token...");

    try {
      // Use the integrated callback handler to prevent race conditions
      const result = await executeCallback(
        webauthnData?.email || "unknown",
        webauthnData?.tenantId,
      );

      if (result.success && result.token) {
        setWebauthnCallbackToken(result.token);

        // Check if this is a SAML + WebAuthn flow
        if (isSamlWebAuthn && samlRedirectTo) {
          console.log(
            "✅ SAML + WebAuthn flow complete, redirecting to Hydra with stored URL:",
            samlRedirectTo,
          );
          setStatus("processing");
          setMessage("Authentication completed! Redirecting to authorization server...");

          setDebugInfo({
            ...(debugInfo || {}),
            webauthn_callback_service: "Integrated WebAuthn Callback Handler",
            callback_flow_type: "saml-oidc-callback",
            token_received: true,
            token_preview: `${result.token.substring(0, 20)}...`,
            saml_redirect_to: samlRedirectTo,
            service_stats: {
              callback_duration: 0,
              callback_timestamp: Date.now(),
            },
          });

          // Clean up SAML parameters from sessionStorage
          sessionStorage.removeItem("saml_client_id");
          sessionStorage.removeItem("saml_user_email");
          sessionStorage.removeItem("saml_tenant_id");
          sessionStorage.removeItem("saml_user_id");
          sessionStorage.removeItem("saml_provider");
          sessionStorage.removeItem("saml_provider_id");
          sessionStorage.removeItem("saml_project_id");
          sessionStorage.removeItem("saml_success");

          // Redirect to Hydra with the stored SAML redirect_to URL
          setTimeout(() => {
            console.log("Redirecting to Hydra (SAML + WebAuthn):", samlRedirectTo);
            window.location.href = samlRedirectTo;
          }, 1500);
        } else {
          // Regular OIDC flow - show token on page
          setStatus("success");
          setMessage("Authentication completed! Token received from webauthn-callback.");

          setDebugInfo({
            ...(debugInfo || {}),
            webauthn_callback_service: "Integrated WebAuthn Callback Handler",
            callback_flow_type: "oidc-callback",
            token_received: true,
            token_preview: `${result.token.substring(0, 20)}...`,
            service_stats: {
              callback_duration: 0,
              callback_timestamp: Date.now(),
            },
          });
        }
      } else {
        setStatus("error");
        setMessage(result.error || "webauthn-callback failed without specific error");
        setDebugInfo({
          ...(debugInfo || {}),
          webauthn_callback_service: "Integrated WebAuthn Callback Handler",
          callback_error: result.error,
        });
      }
    } catch (error) {
      console.error("❌ Integrated WebAuthn callback handler error:", error);
      setStatus("error");
      setMessage(
        `WebAuthn callback handler failed: ${error instanceof Error ? error.message : "Unknown error"}`,
      );
      setDebugInfo({
        ...(debugInfo || {}),
        webauthn_callback_service: "Integrated WebAuthn Callback Handler",
        service_error: error instanceof Error ? error.message : "Unknown error",
      });
    }
  };

  // Handle WebAuthn error
  const handleWebAuthnError = (error: string) => {
    console.error("WebAuthn flow error:", error);
    setStatus("error");
    setMessage(`WebAuthn authentication failed: ${error}`);
    setDebugInfo({
      ...debugInfo,
      webauthn_error: error,
    });
  };

  const getStatusIcon = () => {
    switch (status) {
      case "processing":
        return (
          <motion.div
            animate={{ rotate: 360 }}
            transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
            className="h-12 w-12 rounded-full border-2 border-slate-300 border-t-slate-600"
          />
        );
      case "webauthn":
        return (
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
            className="flex h-12 w-12 items-center justify-center rounded-full bg-blue-100"
          >
            <IconCircleCheck className="h-6 w-6 text-blue-600" />
          </motion.div>
        );
      case "success":
        return (
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
            className="flex h-12 w-12 items-center justify-center rounded-full bg-green-100"
          >
            <IconCircleCheck className="h-6 w-6 text-green-600" />
          </motion.div>
        );
      case "error":
        return (
          <motion.div
            initial={{ scale: 0 }}
            animate={{ scale: 1 }}
            transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
            className="flex h-12 w-12 items-center justify-center rounded-full bg-red-100"
          >
            <IconAlertTriangle className="h-6 w-6 text-red-600" />
          </motion.div>
        );
    }
  };

  const getStatusColor = () => {
    switch (status) {
      case "processing":
        return "text-slate-700";
      case "webauthn":
        return "text-blue-700";
      case "success":
        return "text-green-700";
      case "error":
        return "text-red-700";
    }
  };

  const getStatusTitle = () => {
    switch (status) {
      case "processing":
        return "Processing Authentication";
      case "webauthn":
        return "WebAuthn Authentication";
      case "success":
        return "Authentication Successful!";
      case "error":
        return "Authentication Failed";
    }
  };

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
          eyebrow="OIDC Callback"
          title="Completing the authentication handshake."
          subtitle="This step validates provider response, exchanges code/tokens, and routes into MFA when required."
          points={[
            "State and challenge are validated before issuing session credentials.",
            "Custom login and social providers use the same callback contract.",
            "WebAuthn completion can return directly into this callback.",
          ]}
        />
      }
    >
      <AuthActionPanel className="space-y-6">
        <AuthStepHeader
          title={getStatusTitle()}
          subtitle="Processing your authentication callback."
          meta={`Provider: ${detectedProvider || "Unknown"}`}
        />

        <div className="space-y-6">
          <div className="flex justify-center">{getStatusIcon()}</div>

          <motion.h2
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
            className={`text-center text-xl font-semibold ${getStatusColor()}`}
          >
            {getStatusTitle()}
          </motion.h2>

          <motion.p
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.25 }}
            className="text-center text-sm text-slate-600"
          >
            {message}
          </motion.p>

          <motion.div
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            transition={{ delay: 0.3 }}
            className="auth-inline-note"
          >
            <p className="text-sm text-slate-700">
              <span className="font-medium">Provider:</span> {detectedProvider || "Unknown"}
            </p>
            <p className="text-xs text-slate-500">
              Processing universal OAuth callback...
            </p>
          </motion.div>

          {status === "processing" && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.35 }}
              className="flex items-center justify-center gap-2"
            >
              <motion.div
                animate={{ y: [0, -6, 0] }}
                transition={{ duration: 0.55, repeat: Infinity, repeatDelay: 0.25 }}
                className="h-2 w-2 rounded-full bg-slate-600"
              />
              <motion.div
                animate={{ y: [0, -6, 0] }}
                transition={{ duration: 0.55, repeat: Infinity, repeatDelay: 0.25, delay: 0.1 }}
                className="h-2 w-2 rounded-full bg-slate-600"
              />
              <motion.div
                animate={{ y: [0, -6, 0] }}
                transition={{ duration: 0.55, repeat: Infinity, repeatDelay: 0.25, delay: 0.2 }}
                className="h-2 w-2 rounded-full bg-slate-600"
              />
            </motion.div>
          )}

          {status === "error" && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
              className="space-y-3"
            >
              <div className="flex flex-col justify-center gap-2 sm:flex-row">
                <Button onClick={handleRetry} className="w-full sm:w-auto">
                  Try Again
                </Button>
                <Button variant="outline" onClick={handleGoBack} className="w-full sm:w-auto">
                  Go Back to Login
                </Button>
              </div>

              {import.meta.env.DEV && debugInfo && (
                <details className="auth-inline-note text-left">
                  <summary className="cursor-pointer text-sm text-slate-600 hover:text-slate-900">
                    Debug Information
                  </summary>
                  <pre className="mt-2 max-h-40 overflow-auto text-xs text-slate-700">
                    {JSON.stringify(debugInfo, null, 2)}
                  </pre>
                </details>
              )}
            </motion.div>
          )}

          {status === "success" && webauthnCallbackToken && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.25 }}
              className="auth-callout"
            >
              <div className="mb-2 flex items-center justify-between">
                <h3 className="text-sm font-medium text-slate-800">
                  Token from webauthn-callback
                </h3>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={copyTokenToClipboard}
                  className="h-8 px-2"
                >
                  {copyStatus === "copied" ? (
                    <>
                      <IconCheck className="mr-1 h-3 w-3" />
                      <span className="text-xs">Copied</span>
                    </>
                  ) : (
                    <>
                      <IconCopy className="mr-1 h-3 w-3" />
                      <span className="text-xs">Copy</span>
                    </>
                  )}
                </Button>
              </div>
              <pre className="max-h-60 overflow-auto whitespace-pre-wrap break-all rounded-md bg-white p-3 text-xs text-slate-700">
                {webauthnCallbackToken}
              </pre>
            </motion.div>
          )}

          {status === "success" && tokenResponse && !webauthnCallbackToken && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.25 }}
              className="auth-callout"
            >
              <h3 className="mb-2 text-sm font-medium text-slate-800">Token Response</h3>
              <pre className="max-h-60 overflow-auto rounded-md bg-white p-3 text-xs text-slate-700">
                {JSON.stringify(tokenResponse, null, 2)}
              </pre>
            </motion.div>
          )}

          {status === "success" && !isCustomLoginWebAuthn && !webauthnCallbackToken && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.3 }}
              className="flex justify-center"
            >
              <div className="inline-flex items-center text-sm text-slate-500">
                <motion.div
                  animate={{ rotate: 360 }}
                  transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
                  className="mr-2 h-4 w-4 rounded-full border-2 border-slate-300 border-t-slate-500"
                />
                Redirecting to application...
              </div>
            </motion.div>
          )}

          {status === "success" && isCustomLoginWebAuthn && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.3 }}
              className="text-center text-sm font-medium text-green-700"
            >
              Authentication completed successfully.
            </motion.div>
          )}

          {status === "success" && webauthnCallbackToken && (
            <motion.div
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.3 }}
              className="space-y-2"
            >
              <DeviceManagementPanel token={webauthnCallbackToken} />
            </motion.div>
          )}

          {import.meta.env.DEV && (
            <motion.div
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              transition={{ delay: 0.45 }}
              className="auth-inline-note space-y-1 text-xs text-slate-700"
            >
              <div className="font-medium text-slate-900">Development Info</div>
              <div>URL: {window.location.pathname}</div>
              <div>Detected Provider: {detectedProvider || "None"}</div>
              <div>Code: {urlParams.get("code") ? "present" : "missing"}</div>
              <div>State: {urlParams.get("state") ? "present" : "missing"}</div>
              <div>Error: {urlParams.get("error") || "none"}</div>
              <div>API URL: {config.VITE_API_URL}</div>
              <div>Universal Callback: enabled</div>
            </motion.div>
          )}
        </div>
      </AuthActionPanel>
    </AuthSplitFrame>
  );
};

const OIDCCallbackPage: React.FC = () => {
  return (
    <EndUserAuthProvider>
      <OIDCCallbackPageInner />
    </EndUserAuthProvider>
  );
};

export default OIDCCallbackPage;
