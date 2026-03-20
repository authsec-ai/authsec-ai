import React from "react";
import { Users, Shield, Package, Activity } from "lucide-react";
import { QuickActionCard } from "./QuickActionCard";
import { useWizard } from "@/contexts/WizardContext";
import { getWizardConfig } from "@/features/wizards";

interface QuickActionsStatus {
  adSync?: {
    configured: boolean;
    type: "AD" | "Entra" | null;
    status: "connected" | "syncing" | "error" | "not_configured";
  };
  authMethods?: {
    count: number;
    providers: string[];
  };
  externalServices?: {
    count: number;
    services: string[];
  };
  logging?: {
    configured: boolean;
    status: "active" | "inactive";
  };
}

interface QuickActionsGridProps {
  status?: QuickActionsStatus;
  isLoading?: boolean;
}

export function QuickActionsGrid({
  status,
  isLoading = false,
}: QuickActionsGridProps) {
  const { startWizard, isActive } = useWizard();
  const [, forceUpdate] = React.useReducer((x) => x + 1, 0);

  const getAuthMethodsBadge = () => {
    const count = status?.authMethods?.count || 0;
    if (count === 0) return "No Providers";
    return `${count} Provider${count > 1 ? "s" : ""}`;
  };

  const getAuthMethodsColor = ():
    | "success"
    | "warning"
    | "info"
    | "default" => {
    const count = status?.authMethods?.count || 0;
    return count > 0 ? "success" : "default";
  };

  // Helper functions to determine status
  const getADSyncBadge = () => {
    if (!status?.adSync) return "Not Connected";
    if (status.adSync.status === "connected") {
      return status.adSync.type
        ? `${status.adSync.type} Connected`
        : "Connected";
    }
    if (status.adSync.status === "syncing") return "Syncing...";
    if (status.adSync.status === "error") return "Error";
    return "Not Connected";
  };

  const getADSyncColor = (): "success" | "warning" | "info" | "default" => {
    if (!status?.adSync) return "default";
    if (status.adSync.status === "connected") return "success";
    if (status.adSync.status === "syncing") return "info";
    if (status.adSync.status === "error") return "warning";
    return "default";
  };

  const getExternalServicesBadge = () => {
    const count = status?.externalServices?.count || 0;
    if (count === 0) return "No Services";
    return `${count} Service${count > 1 ? "s" : ""}`;
  };

  const getExternalServicesColor = ():
    | "success"
    | "warning"
    | "info"
    | "default" => {
    const count = status?.externalServices?.count || 0;
    return count > 0 ? "success" : "default";
  };

  const getLoggingBadge = () => {
    if (!status?.logging) return "Configure Required";
    return status.logging.status === "active" ? "Active" : "Inactive";
  };

  const getLoggingColor = (): "success" | "warning" | "info" | "default" => {
    if (!status?.logging) return "default";
    return status.logging.status === "active" ? "success" : "warning";
  };

  const quickActions = [
    {
      id: "sync-ad-users",
      icon: Users,
      title: "Sync AD Users",
      description:
        "Connect to Active Directory or Azure Entra ID to sync user identities",
      ctaLabel: "Configure Sync",
      ctaLink: "/admin/users",
      navigationState: {
        openDirectorySync: true,
        provider: "ad",
        mode: "configure",
      },
      statusBadge: isLoading ? "..." : getADSyncBadge(),
      statusColor: getADSyncColor(),
      gradient:
        "from-blue-50 to-sky-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
    },
    {
      id: "auth-methods",
      icon: Shield,
      title: "Authentication Methods",
      description:
        "Configure OIDC, SAML, or social auth providers for your applications",
      ctaLabel: "Add Provider",
      ctaLink: "/authentication/create",
      statusBadge: isLoading ? "..." : getAuthMethodsBadge(),
      statusColor: getAuthMethodsColor(),
      gradient:
        "from-blue-50 to-cyan-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
    },
    {
      id: "external-services",
      icon: Package,
      title: "Manage External Services & Secrets",
      description:
        "Connect to Google Drive, Microsoft services, and securely manage API secrets and credentials",
      ctaLabel: "Add Service",
      ctaLink: "/external-services/add",
      statusBadge: isLoading ? "..." : getExternalServicesBadge(),
      statusColor: getExternalServicesColor(),
      gradient:
        "from-green-50 to-emerald-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-green-500/10 dark:bg-green-400/20",
      iconColor: "text-green-600 dark:text-green-400",
    },
    {
      id: "auth-logs",
      icon: Activity,
      title: "Authentication Logs",
      description: "Monitor and configure authentication and audit log streams",
      ctaLabel: "View Logs",
      ctaLink: "/logs/auth",
      statusBadge: isLoading ? "..." : getLoggingBadge(),
      statusColor: getLoggingColor(),
      gradient:
        "from-orange-50 to-amber-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-orange-500/10 dark:bg-orange-400/20",
      iconColor: "text-orange-600 dark:text-orange-400",
      secondaryAction: {
        label: "Configure",
        link: "/logs/configure",
      },
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-foreground">Quick Actions</h2>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {quickActions.map((action) => (
          <div key={action.id}>
            <QuickActionCard {...action} />
          </div>
        ))}
      </div>
    </div>
  );
}
