import * as amplitude from "@amplitude/analytics-browser";
import clarity from "@microsoft/clarity";

/**
 * Centralized analytics utility for custom Amplitude event tracking.
 * All calls are fire-and-forget — errors are caught and logged, never thrown.
 */

function track(eventName: string, properties?: Record<string, unknown>): void {
  try {
    amplitude.track(eventName, properties);
  } catch (error) {
    console.error("[Analytics] Failed to track:", eventName, error);
  }
}

// ─── Auth Flow Events ───────────────────────────────────────────────

export function trackSignInAttempted(
  method: "password" | "oauth",
  provider?: string,
): void {
  track("auth_sign_in_attempted", { method, provider });
}

export function trackSignInSucceeded(): void {
  track("auth_sign_in_succeeded");
}

export function trackSignUpStarted(): void {
  track("auth_sign_up_started");
}

export function trackWorkspaceCreated(tenantDomain: string): void {
  track("auth_workspace_created", { tenant_domain: tenantDomain });
}

export function trackOtpVerified(): void {
  track("auth_otp_verified");
}

export function trackOAuthProviderClicked(providerName: string): void {
  track("auth_oauth_provider_clicked", { provider: providerName });
}

export function trackForgotPasswordRequested(): void {
  track("auth_forgot_password_requested");
}

export function trackPasswordResetCompleted(): void {
  track("auth_password_reset_completed");
}

// ─── User Management Events ─────────────────────────────────────────

export function trackUserDeleted(audience: "admin" | "enduser"): void {
  track("user_deleted", { audience });
}

export function trackUserStatusChanged(
  active: boolean,
  audience: "admin" | "enduser",
): void {
  track("user_status_changed", { active, audience });
}

export function trackUserPasswordReset(
  audience: "admin" | "enduser",
): void {
  track("user_password_reset", { audience });
}

export function trackUserRoleAssigned(): void {
  track("user_role_assigned");
}

// ─── Client Management Events ───────────────────────────────────────

export function trackClientCreated(clientName: string): void {
  track("client_created", { client_name: clientName });
}

export function trackClientDeleted(): void {
  track("client_deleted");
}

export function trackClientStatusToggled(active: boolean): void {
  track("client_status_toggled", { active });
}

export function trackClientPreviewLogin(): void {
  track("client_preview_login_opened");
}

export function trackVoiceAgentConfigured(): void {
  track("client_voice_agent_configured");
}

// ─── Role Management Events ─────────────────────────────────────────

export function trackRoleCreated(roleName: string): void {
  track("role_created", { role_name: roleName });
}

export function trackRoleDeleted(count: number): void {
  track("role_deleted", { count });
}

// ─── Auth Provider Events ───────────────────────────────────────────

export function trackAuthProviderDeleted(
  providerType: "oidc" | "saml",
): void {
  track("auth_provider_deleted", { provider_type: providerType });
}

// ─── Dashboard Events ────────────────────────────────────────────────

export function trackDashboardStartSetup(): void {
  track("dashboard_start_setup");
}

export function trackDashboardContinueSetup(stepsCompleted: number): void {
  track("dashboard_continue_setup", { steps_completed: stepsCompleted });
}

export function trackDashboardActivationCompleted(): void {
  track("dashboard_activation_completed");
}

export function trackDashboardActivationDismissed(isCompleted: boolean): void {
  track("dashboard_activation_dismissed", { is_completed: isCompleted });
}

export function trackDashboardTourCardClicked(
  wizard: string,
  isCompleted: boolean,
): void {
  track("dashboard_tour_card_clicked", { wizard, is_completed: isCompleted });
}

export function trackDashboardQuickActionClicked(action: string): void {
  track("dashboard_quick_action_clicked", { action });
}

// ─── Wizard Lifecycle Events ─────────────────────────────────────────

export function trackWizardStarted(wizardId: string): void {
  track("wizard_started", { wizard_id: wizardId });
}

export function trackWizardStepCompleted(
  wizardId: string,
  stepId: string,
): void {
  track("wizard_step_completed", { wizard_id: wizardId, step_id: stepId });
}

export function trackWizardCompleted(wizardId: string): void {
  track("wizard_completed", { wizard_id: wizardId });
}

export function trackWizardDismissed(
  wizardId: string,
  stepIndex: number,
): void {
  track("wizard_dismissed", { wizard_id: wizardId, step_index: stepIndex });
}

export function trackWizardSkipped(wizardId: string): void {
  track("wizard_skipped", { wizard_id: wizardId });
}

// ─── Wizard Step-Specific Events ────────────────────────────────────

export function trackWizardUserAuthMethodSelected(
  method: "oidc" | "saml2",
): void {
  track("wizard_user_auth_method_selected", { method });
}

export function trackWizardRbacContextSelected(
  context: "admin" | "end_user",
): void {
  track("wizard_rbac_context_selected", { context });
}

export function trackWizardScopesContextSelected(
  context: "admin" | "end_user",
): void {
  track("wizard_scopes_context_selected", { context });
}

export function trackWizardM2MSpireFaqOpened(): void {
  track("wizard_m2m_spire_faq_opened");
}

export function trackWizardM2MSpireRecheck(): void {
  track("wizard_m2m_spire_recheck");
}

// ─── User Identification ────────────────────────────────────────────

/**
 * Set the user ID in both Amplitude and Clarity so all events are associated with this user.
 * Call this after login succeeds.
 */
export function setAnalyticsUserId(userId: string): void {
  // Set in Amplitude
  try {
    amplitude.setUserId(userId);
  } catch (error) {
    console.error("[Analytics] Failed to set Amplitude userId:", error);
  }

  // Set in Clarity
  try {
    clarity.identify(userId);
  } catch (error) {
    console.error("[Analytics] Failed to set Clarity userId:", error);
  }
}

/**
 * Set the user ID in Amplitude so all events are associated with this user.
 * Call this after login succeeds.
 * @deprecated Use setAnalyticsUserId() instead for both Amplitude and Clarity tracking
 */
export function setAmplitudeUserId(userId: string): void {
  setAnalyticsUserId(userId);
}

// ─── Page View Tracking ─────────────────────────────────────────────

/**
 * Track a page view with a friendly page name.
 * The page name is included in the event name for immediate visibility in the dashboard.
 */
export function trackPageViewed(pageName: string, path: string): void {
  const eventName = `page_viewed_${pageName.toLowerCase().replace(/\s+/g, "_")}`;
  track(eventName, { path });
}
