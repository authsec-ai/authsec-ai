import React, { useState } from "react";
import {
  Activity,
  LayoutDashboard,
  Lock,
  Package,
  Shield,
  Users,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { toast } from "react-hot-toast";

import { SessionManager } from "../../utils/sessionManager";
import { useGetAllClientsQuery } from "../../app/api/clientApi";
import { useListDomainsQuery } from "../../app/api/domainApi";
import { useGetExternalServicesQuery } from "../../app/api/externalServiceApi";
import { useWizard } from "@/contexts/WizardContext";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";

import { useDashboardData } from "./hooks/useDashboardData";
import { useWizardStatus } from "./hooks/useWizardStatus";
import { useDashboardOverview } from "./hooks/useDashboardOverview";

import { ActivationSection } from "./components/ActivationSection";
import { SetupTourList } from "./components/SetupTourList";
import { DashboardSection } from "./components/DashboardSection";
import { DashboardActionTile } from "./components/DashboardActionTile";
import { DashboardOperationalStatusPanel } from "./components/DashboardOperationalStatusPanel";
import { DashboardTile } from "./components/DashboardTile";
import { ClientSelectionModal } from "./components/ClientSelectionModal";
import { ServiceSelectionModal } from "./components/ServiceSelectionModal";

import "./dashboard-theme.css";

export function DashboardPage() {
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;
  const navigate = useNavigate();
  const { audience } = useRbacAudience();
  const { isActive, activeWizard, completedSteps } = useWizard();

  const [showClientSelection, setShowClientSelection] = useState(false);
  const [showServiceSelection, setShowServiceSelection] = useState(false);

  const {
    isLoading: dashboardLoading,
    isError,
    error,
    quickActionsStatus,
  } = useDashboardData({
    tenantId: tenantId || "",
  });

  const { data: clientsData, isLoading: isClientsLoading } = useGetAllClientsQuery(
    { tenant_id: tenantId || "" },
    { skip: !tenantId },
  );

  const { data: domainsData, isLoading: isDomainsLoading } = useListDomainsQuery(
    { tenant_id: tenantId || "" },
    { skip: !tenantId },
  );

  const { data: servicesData, isLoading: isServicesLoading } = useGetExternalServicesQuery(
    undefined,
    {
      skip: !tenantId,
    },
  );

  const userAuth = useWizardStatus("user-auth-wizard");

  const completedWizards = React.useMemo(() => {
    try {
      const raw = localStorage.getItem("authsec_wizards");
      if (!raw) return [] as string[];
      const parsed = JSON.parse(raw) as { completedWizards?: unknown };
      if (!Array.isArray(parsed.completedWizards)) return [] as string[];
      return parsed.completedWizards.filter(
        (wizardId): wizardId is string => typeof wizardId === "string",
      );
    } catch {
      return [] as string[];
    }
  }, [isActive, activeWizard]);

  const hasClient = (clientsData?.clients?.length || 0) > 0;
  const customDomainAdded =
    (domainsData || []).some((domain) => domain.kind === "custom") || false;

  const isUserAuthWizard = activeWizard === "user-auth-wizard";
  const activationStep1Done =
    userAuth.isCompleted ||
    (isUserAuthWizard && completedSteps.includes("client-selection"));
  const activationStep2Done =
    userAuth.isCompleted ||
    (isUserAuthWizard && completedSteps.includes("integrate-sdk"));

  const setupSteps = React.useMemo(() => {
    const rbacConfigured = completedWizards.includes("rbac-wizard");
    const sdkDeployed = completedWizards.includes("m2m-workload-wizard");

    return [
      { id: "create-client", done: hasClient },
      { id: "configure-authentication", done: activationStep1Done },
      { id: "setup-rbac", done: rbacConfigured },
      { id: "add-domains", done: customDomainAdded },
      { id: "deploy-sdk", done: sdkDeployed },
    ];
  }, [hasClient, activationStep1Done, completedWizards, customDomainAdded]);

  const clientsCount = tenantId
    ? typeof clientsData?.clients?.length === "number"
      ? clientsData.clients.length
      : isClientsLoading
        ? null
        : 0
    : null;

  const customDomainCount = tenantId
    ? Array.isArray(domainsData)
      ? domainsData.filter((domain) => domain.kind === "custom").length
      : isDomainsLoading
        ? null
        : 0
    : null;

  const externalServiceCount = tenantId
    ? Array.isArray(servicesData)
      ? servicesData.length
      : isServicesLoading
        ? null
        : 0
    : null;

  const overview = useDashboardOverview({
    setupSteps,
    clientsCount,
    customDomainCount,
    externalServiceCount,
    quickActionsStatus,
    isLoading:
      dashboardLoading || isClientsLoading || isDomainsLoading || isServicesLoading,
  });

  const operationalMap = React.useMemo(() => {
    return new Map(overview.operationalItems.map((item) => [item.id, item]));
  }, [overview.operationalItems]);

  const adSyncOp = operationalMap.get("ad-sync");
  const authMethodsOp = operationalMap.get("auth-methods");
  const loggingOp = operationalMap.get("logging");

  const externalServicesTileStatus =
    externalServiceCount === null
      ? { label: "Unknown", tone: "muted" as const, meta: "Inventory status unavailable" }
      : externalServiceCount > 0
        ? {
            label: "Configured",
            tone: "success" as const,
            meta: `${externalServiceCount} service${externalServiceCount > 1 ? "s" : ""} connected`,
          }
        : {
            label: "Needs Setup",
            tone: "warning" as const,
            meta: "No services configured yet",
          };

  const authSdkMeta =
    clientsCount === null
      ? "Client inventory unavailable"
      : clientsCount > 0
        ? `${clientsCount} client${clientsCount > 1 ? "s" : ""} available for onboarding`
        : "Create a client before launching SDK onboarding";

  const servicesSdkMeta =
    externalServiceCount === null
      ? "Service inventory unavailable"
      : externalServiceCount > 0
        ? `${externalServiceCount} service${externalServiceCount > 1 ? "s" : ""} available for SDK setup`
        : "Add a service first to enable guided SDK setup";

  const handleAuthSDKClick = () => {
    const clients = clientsData?.clients || [];
    if (clients.length > 0) {
      setShowClientSelection(true);
    } else {
      toast.error("No clients present. Create a client first.");
      navigate("/clients/mcp");
    }
  };

  const handleServicesSDKClick = () => {
    const services = servicesData || [];
    if (services.length > 0) {
      setShowServiceSelection(true);
    } else {
      toast.error("No services present. Create a service first.");
      navigate("/external-services/add");
    }
  };

  const handleRecommendationClick = React.useCallback(() => {
    switch (overview.setup.nextStepId) {
      case "create-client":
        navigate("/clients/mcp");
        return;
      case "configure-authentication":
        userAuth.launch();
        return;
      case "setup-rbac":
        navigate(`/${audience}/permissions`);
        return;
      case "add-domains":
        navigate("/custom-domains");
        return;
      case "deploy-sdk":
        handleAuthSDKClick();
        return;
      default:
        navigate("/clients/mcp");
    }
  }, [
    overview.setup.nextStepId,
    navigate,
    userAuth,
    audience,
    handleAuthSDKClick,
  ]);

  const handleOperationalItemClick = React.useCallback(
    (item: { id: string }) => {
      switch (item.id) {
        case "ad-sync":
          navigate("/admin/users", {
            state: {
              openDirectorySync: true,
              provider: "ad",
              mode: "configure",
            },
          });
          return;
        case "auth-methods":
          navigate("/authentication");
          return;
        case "domains":
          navigate("/custom-domains");
          return;
        case "logging":
          navigate("/logs/auth");
          return;
        default:
          break;
      }
    },
    [navigate],
  );

  if (isError) {
    return (
      <div data-dashboard="overview" className="dash-page h-full">
        <div className="mx-auto max-w-[1600px] p-4 sm:p-6">
          <DashboardTile className="p-6 sm:p-8">
            <div className="flex flex-col items-center gap-4 text-center">
              <div className="dash-icon-chip" data-tone="danger">
                <LayoutDashboard className="h-5 w-5" />
              </div>
              <div className="space-y-1">
                <h2 className="text-base font-semibold dash-text-1">
                  Unable to Load Dashboard
                </h2>
                <p className="text-sm dash-text-2">
                  {error?.message || "Failed to fetch dashboard data"}
                </p>
              </div>
            </div>
          </DashboardTile>
        </div>
      </div>
    );
  }

  return (
    <div data-dashboard="overview" className="dash-page min-h-full">
      <div className="mx-auto max-w-[1600px] space-y-4 px-4 py-4 sm:px-6 sm:py-5">
        <ActivationSection
          step1Done={activationStep1Done}
          step2Done={activationStep2Done}
          onStart={userAuth.launch}
          isWizardActive={isActive}
        />

        <div className="grid gap-4 xl:grid-cols-12">
          <div className="space-y-4 xl:col-span-7">
            <SetupTourList />

            <DashboardSection
              label="Integrations"
              title="SDK Integrations"
              description="Launch client and service SDK onboarding from a compact integration workspace."
            >
              <div className="dash-group-panel divide-y dash-divider">
                <DashboardActionTile
                  icon={Shield}
                  iconTone="accent"
                  title="Auth SDK"
                  description="Add enterprise authentication to client apps with OAuth, SAML, and WebAuthn."
                  statusLabel="Recommended"
                  statusTone="accent"
                  meta={authSdkMeta}
                  primaryActionLabel="Open Setup"
                  onPrimaryAction={handleAuthSDKClick}
                  primaryActionPriority="inline"
                  layout="row"
                  framed={false}
                  revealActionsOnHover
                />
                <DashboardActionTile
                  icon={Package}
                  iconTone="success"
                  title="Services & Secrets SDK"
                  description="Integrate third-party services and manage API credentials through the SDK path."
                  statusLabel={
                    externalServiceCount !== null && externalServiceCount > 0
                      ? "Ready"
                      : externalServiceCount === null
                        ? "Unknown"
                        : "Blocked"
                  }
                  statusTone={
                    externalServiceCount !== null && externalServiceCount > 0
                      ? "success"
                      : externalServiceCount === null
                        ? "muted"
                        : "warning"
                  }
                  meta={servicesSdkMeta}
                  primaryActionLabel="Open Setup"
                  onPrimaryAction={handleServicesSDKClick}
                  primaryActionPriority="inline"
                  layout="row"
                  framed={false}
                  revealActionsOnHover
                />
                <DashboardActionTile
                  icon={Lock}
                  iconTone="muted"
                  title="RBAC SDK"
                  description="Fine-grained authorization SDK workflows for roles, resources, and permissions."
                  statusLabel="Soon"
                  statusTone="muted"
                  meta="This flow is planned but not available in the current dashboard build."
                  primaryActionLabel="Unavailable"
                  onPrimaryAction={() => {}}
                  primaryActionPriority="inline"
                  disabled
                  layout="row"
                  framed={false}
                />
              </div>
            </DashboardSection>
          </div>

          <div className="space-y-4 xl:col-span-5">
            <DashboardOperationalStatusPanel
              items={overview.operationalItems}
              recommendation={overview.recommendation}
              onRecommendationAction={handleRecommendationClick}
              onItemAction={handleOperationalItemClick}
            />

            <DashboardSection
              label="Actions"
              title="Quick Actions"
              description="Configuration shortcuts with current state visible before you click."
            >
              <div className="dash-group-panel divide-y dash-divider">
                <DashboardActionTile
                  icon={Users}
                  iconTone="accent"
                  title="Sync AD Users"
                  description="Connect Active Directory or Entra ID and keep user identities in sync."
                  statusLabel={adSyncOp?.statusLabel ?? "Unknown"}
                  statusTone={adSyncOp?.tone ?? "muted"}
                  meta={adSyncOp?.detail ?? "Directory sync status unavailable"}
                  primaryActionLabel="Configure"
                  onPrimaryAction={() =>
                    navigate("/admin/users", {
                      state: {
                        openDirectorySync: true,
                        provider: "ad",
                        mode: "configure",
                      },
                    })
                  }
                  primaryActionPriority="inline"
                  layout="row"
                  framed={false}
                  revealActionsOnHover
                />
                <DashboardActionTile
                  icon={Shield}
                  iconTone="accent"
                  title="Auth Methods"
                  description="Manage OIDC, SAML, and social providers for application sign-in."
                  statusLabel={authMethodsOp?.statusLabel ?? "Unknown"}
                  statusTone={authMethodsOp?.tone ?? "muted"}
                  meta={authMethodsOp?.detail ?? "Auth provider status unavailable"}
                  primaryActionLabel="Add Provider"
                  onPrimaryAction={() => navigate("/authentication/create")}
                  primaryActionPriority="inline"
                  layout="row"
                  framed={false}
                  revealActionsOnHover
                />
                <DashboardActionTile
                  icon={Package}
                  iconTone="success"
                  title="External Services"
                  description="Connect third-party services and manage API credentials and secrets."
                  statusLabel={externalServicesTileStatus.label}
                  statusTone={externalServicesTileStatus.tone}
                  meta={externalServicesTileStatus.meta}
                  primaryActionLabel={
                    externalServiceCount !== null && externalServiceCount > 0
                      ? "Manage"
                      : "Add Service"
                  }
                  onPrimaryAction={() => navigate("/external-services/add")}
                  primaryActionPriority="inline"
                  layout="row"
                  framed={false}
                  revealActionsOnHover
                />
                <DashboardActionTile
                  icon={Activity}
                  iconTone={loggingOp?.tone ?? "muted"}
                  title="Auth Logs"
                  description="Review authentication and audit log streams and validate pipeline health."
                  statusLabel={loggingOp?.statusLabel ?? "Unknown"}
                  statusTone={loggingOp?.tone ?? "muted"}
                  meta={loggingOp?.detail ?? "Logging status unavailable"}
                  primaryActionLabel="View Logs"
                  onPrimaryAction={() => navigate("/logs/auth")}
                  primaryActionPriority="inline"
                  layout="row"
                  framed={false}
                  revealActionsOnHover
                />
              </div>
            </DashboardSection>
          </div>
        </div>

        <ClientSelectionModal
          isOpen={showClientSelection}
          onClose={() => setShowClientSelection(false)}
        />
        <ServiceSelectionModal
          isOpen={showServiceSelection}
          onClose={() => setShowServiceSelection(false)}
        />
      </div>
    </div>
  );
}
