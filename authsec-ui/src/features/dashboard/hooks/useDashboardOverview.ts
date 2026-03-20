import { useMemo } from "react";
import type { QuickActionsStatus } from "@/app/api/dashboardApi";
import type { DashboardStatusTone } from "../components/DashboardStatusTag";
import type { DashboardSummaryRailItem } from "../components/DashboardSummaryRail";
import type { DashboardOperationalStatusItem } from "../components/DashboardOperationalStatusPanel";

interface SetupStepLike {
  id: string;
  done: boolean;
}

interface UseDashboardOverviewProps {
  setupSteps: SetupStepLike[];
  clientsCount: number | null;
  customDomainCount: number | null;
  externalServiceCount: number | null;
  quickActionsStatus?: QuickActionsStatus | null;
  isLoading?: boolean;
}

export interface DashboardOverview {
  setup: {
    completed: number;
    total: number;
    percentage: number;
    nextStepId: string | null;
    nextStepLabel: string | null;
    nextStepDescription: string | null;
    statusLabel: string;
    statusTone: DashboardStatusTone;
  };
  summaryItems: DashboardSummaryRailItem[];
  operationalItems: DashboardOperationalStatusItem[];
  recommendation: {
    title: string;
    detail: string;
  };
}

const NEXT_STEP_COPY: Record<
  string,
  { label: string; description: string }
> = {
  "create-client": {
    label: "Create your first client",
    description: "Add an application client before onboarding auth providers and SDK flows.",
  },
  "configure-authentication": {
    label: "Configure an authentication provider",
    description: "Add OIDC, SAML, or social sign-in so users can start authenticating.",
  },
  "setup-rbac": {
    label: "Set up RBAC roles and permissions",
    description: "Define permissions and roles before onboarding more apps and users.",
  },
  "add-domains": {
    label: "Add a custom domain",
    description: "Connect a production domain to move beyond default tenant-hosted URLs.",
  },
  "deploy-sdk": {
    label: "Deploy the SDK",
    description: "Integrate AuthSec SDK into your app and verify the login path end-to-end.",
  },
};

function countToDisplay(value: number | null): string {
  return value === null ? "—" : value.toLocaleString();
}

function authMethodsStatusCopy(
  quickActionsStatus?: QuickActionsStatus | null,
): { value: string; meta: string; tone: DashboardStatusTone } {
  const count = quickActionsStatus?.authMethods?.count;
  if (typeof count !== "number") {
    return { value: "—", meta: "Authentication methods status unavailable", tone: "muted" };
  }
  if (count === 0) {
    return { value: "0", meta: "No auth providers configured", tone: "warning" };
  }
  return {
    value: String(count),
    meta: `${count} provider${count > 1 ? "s" : ""} connected`,
    tone: "success",
  };
}

function loggingStatusCopy(
  quickActionsStatus?: QuickActionsStatus | null,
): { value: string; meta: string; tone: DashboardStatusTone } {
  const status = quickActionsStatus?.logging?.status;
  if (!status) {
    return {
      value: "—",
      meta: "Logging configuration status unavailable",
      tone: "muted",
    };
  }

  if (status === "active") {
    return { value: "Active", meta: "Auth and audit logs are enabled", tone: "success" };
  }

  return { value: "Inactive", meta: "Configure log pipelines", tone: "warning" };
}

export function useDashboardOverview({
  setupSteps,
  clientsCount,
  customDomainCount,
  externalServiceCount,
  quickActionsStatus,
  isLoading = false,
}: UseDashboardOverviewProps): DashboardOverview {
  return useMemo(() => {
    const total = setupSteps.length;
    const completed = setupSteps.filter((step) => step.done).length;
    const percentage = total > 0 ? Math.round((completed / total) * 100) : 0;
    const nextStep = setupSteps.find((step) => !step.done) ?? null;
    const nextCopy = nextStep ? NEXT_STEP_COPY[nextStep.id] : null;

    const authMethods = authMethodsStatusCopy(quickActionsStatus);
    const logging = loggingStatusCopy(quickActionsStatus);

    const adSyncStatus = quickActionsStatus?.adSync?.status;
    const adSyncTone: DashboardStatusTone =
      adSyncStatus === "connected"
        ? "success"
        : adSyncStatus === "syncing"
          ? "accent"
          : adSyncStatus === "error"
            ? "danger"
            : adSyncStatus === "not_configured"
              ? "warning"
              : "muted";

    const adSyncLabel =
      adSyncStatus === "connected"
        ? quickActionsStatus?.adSync?.type
          ? `${quickActionsStatus.adSync.type} Connected`
          : "Connected"
        : adSyncStatus === "syncing"
          ? "Syncing"
          : adSyncStatus === "error"
            ? "Error"
            : adSyncStatus === "not_configured"
              ? "Not Configured"
              : "Unknown";

    const setupStatusTone: DashboardStatusTone =
      completed === total && total > 0
        ? "success"
        : completed > 0
          ? "accent"
          : "neutral";
    const setupStatusLabel =
      completed === total && total > 0
        ? "Complete"
        : completed > 0
          ? "In Progress"
          : "Ready";

    const summaryItems: DashboardSummaryRailItem[] = [
      {
        id: "setup-progress",
        label: "Setup Progress",
        value: `${completed}/${total}`,
        meta:
          nextCopy?.label ??
          (isLoading ? "Syncing dashboard status" : "All setup milestones complete"),
        tagLabel: setupStatusLabel,
        tagTone: setupStatusTone,
      },
      {
        id: "clients",
        label: "Clients",
        value: countToDisplay(clientsCount),
        meta:
          clientsCount === null
            ? "Client count unavailable"
            : `${clientsCount} registered application client${clientsCount === 1 ? "" : "s"}`,
      },
      {
        id: "auth-methods",
        label: "Auth Providers",
        value: authMethods.value,
        meta: authMethods.meta,
        tagLabel: authMethods.tone === "success" ? "Healthy" : authMethods.tone === "warning" ? "Needs Setup" : authMethods.tone === "muted" ? "Unknown" : "Review",
        tagTone: authMethods.tone,
      },
      {
        id: "external-services",
        label: "External Services",
        value: countToDisplay(externalServiceCount),
        meta:
          externalServiceCount === null
            ? "Service inventory unavailable"
            : `${externalServiceCount} integrated service${externalServiceCount === 1 ? "" : "s"}`,
      },
      {
        id: "logging",
        label: "Logging",
        value: logging.value,
        meta: logging.meta,
        tagLabel:
          logging.tone === "success"
            ? "Healthy"
            : logging.tone === "warning"
              ? "Attention"
              : "Unknown",
        tagTone: logging.tone,
      },
    ];

    const authProviderCount =
      typeof quickActionsStatus?.authMethods?.count === "number"
        ? quickActionsStatus.authMethods.count
        : null;

    const operationalItems: DashboardOperationalStatusItem[] = [
      {
        id: "ad-sync",
        label: "Directory Sync",
        detail:
          quickActionsStatus?.adSync?.type
            ? `${quickActionsStatus.adSync.type} sync path configured`
            : "Connect AD or Entra ID for user sync",
        statusLabel: adSyncLabel,
        tone: adSyncTone,
      },
      {
        id: "auth-methods",
        label: "Auth Methods",
        detail:
          authProviderCount === null
            ? "Provider count unavailable"
            : authProviderCount === 0
              ? "No providers configured"
              : `${authProviderCount} provider${authProviderCount > 1 ? "s" : ""} available`,
        statusLabel:
          authProviderCount === null
            ? "Unknown"
            : authProviderCount > 0
              ? "Configured"
              : "Needs Setup",
        tone:
          authProviderCount === null
            ? "muted"
            : authProviderCount > 0
              ? "success"
              : "warning",
      },
      {
        id: "domains",
        label: "Custom Domains",
        detail:
          customDomainCount === null
            ? "Domain data unavailable"
            : customDomainCount === 0
              ? "Using default tenant domain only"
              : `${customDomainCount} custom domain${customDomainCount > 1 ? "s" : ""} configured`,
        statusLabel:
          customDomainCount === null
            ? "Unknown"
            : customDomainCount > 0
              ? "Configured"
              : "Default Only",
        tone:
          customDomainCount === null
            ? "muted"
            : customDomainCount > 0
              ? "success"
              : "neutral",
      },
      {
        id: "logging",
        label: "Log Streams",
        detail: logging.meta,
        statusLabel: logging.value,
        tone: logging.tone,
      },
    ];

    return {
      setup: {
        completed,
        total,
        percentage,
        nextStepId: nextStep?.id ?? null,
        nextStepLabel: nextCopy?.label ?? null,
        nextStepDescription: nextCopy?.description ?? null,
        statusLabel: setupStatusLabel,
        statusTone: setupStatusTone,
      },
      summaryItems,
      operationalItems,
      recommendation: {
        title: nextCopy?.label ?? "Dashboard setup is complete",
        detail:
          nextCopy?.description ??
          "Use quick actions to iterate on providers, services, and logging configuration.",
      },
    };
  }, [
    setupSteps,
    clientsCount,
    customDomainCount,
    externalServiceCount,
    quickActionsStatus,
    isLoading,
  ]);
}

