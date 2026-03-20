import React from "react";
import { Shield } from "lucide-react";
import { QuickActionCard } from "./QuickActionCard";
import { useWizard } from "@/contexts/WizardContext";
import { getWizardConfig } from "@/features/wizards";

interface IntegrationGuideGridProps {
  isLoading?: boolean;
}

export function IntegrationGuideGrid({
  isLoading = false,
}: IntegrationGuideGridProps) {
  const { startWizard, isActive } = useWizard();
  const [, forceUpdate] = React.useReducer((x) => x + 1, 0);

  // Force update when wizard becomes inactive (completed/dismissed)
  React.useEffect(() => {
    if (!isActive) {
      forceUpdate();
    }
  }, [isActive]);

  // Helper to clear wizard completion state
  const clearWizardCompletion = (wizardId: string) => {
    const stored = localStorage.getItem("authsec_wizards");
    if (stored) {
      const data = JSON.parse(stored);
      data.completedWizards =
        data.completedWizards?.filter((id: string) => id !== wizardId) || [];
      localStorage.setItem("authsec_wizards", JSON.stringify(data));
    }
  };

  // Handler for User Auth wizard
  const handleStartUserAuthWizard = () => {
    // If already completed, clear completion state to allow restart
    if (isUserAuthWizardCompleted()) {
      clearWizardCompletion("user-auth-wizard");
    }

    const wizardConfig = getWizardConfig("user-auth-wizard");
    if (wizardConfig) {
      startWizard("user-auth-wizard", wizardConfig);
    }
  };

  // Check if User Auth wizard is completed
  const isUserAuthWizardCompleted = () => {
    try {
      const stored = localStorage.getItem("authsec_wizards");
      if (!stored) return false;
      const data = JSON.parse(stored);
      return data.completedWizards?.includes("user-auth-wizard") || false;
    } catch {
      return false;
    }
  };

  const getUserAuthBadge = () => {
    if (isUserAuthWizardCompleted()) return "Completed";
    return "Get Started";
  };

  const getUserAuthColor = (): "success" | "warning" | "info" | "default" => {
    if (isUserAuthWizardCompleted()) return "success";
    return "default";
  };
  // Handler for M2M wizard
  const handleStartM2MWizard = () => {
    // If already completed, clear completion state to allow restart
    if (isM2MWizardCompleted()) {
      clearWizardCompletion("m2m-workload-wizard");
    }

    const wizardConfig = getWizardConfig("m2m-workload-wizard");
    if (wizardConfig) {
      startWizard("m2m-workload-wizard", wizardConfig);
    }
  };

  // Check if M2M wizard is completed
  const isM2MWizardCompleted = () => {
    try {
      const stored = localStorage.getItem("authsec_wizards");
      if (!stored) return false;
      const data = JSON.parse(stored);
      return data.completedWizards?.includes("m2m-workload-wizard") || false;
    } catch {
      return false;
    }
  };

  const getM2MBadge = () => {
    if (isM2MWizardCompleted()) return "Completed";
    return "Get Started";
  };

  const getM2MColor = (): "success" | "warning" | "info" | "default" => {
    if (isM2MWizardCompleted()) return "success";
    return "default";
  };

  // Handler for RBAC wizard
  const handleStartRBACWizard = () => {
    // If already completed, clear completion state to allow restart
    if (isRBACWizardCompleted()) {
      clearWizardCompletion("rbac-wizard");
    }

    const wizardConfig = getWizardConfig("rbac-wizard");
    if (wizardConfig) {
      startWizard("rbac-wizard", wizardConfig);
    }
  };

  // Check if RBAC wizard is completed
  const isRBACWizardCompleted = () => {
    try {
      const stored = localStorage.getItem("authsec_wizards");
      if (!stored) return false;
      const data = JSON.parse(stored);
      return data.completedWizards?.includes("rbac-wizard") || false;
    } catch {
      return false;
    }
  };

  const getRBACBadge = () => {
    if (isRBACWizardCompleted()) return "Completed";
    return "Get Started";
  };

  const getRBACColor = (): "success" | "warning" | "info" | "default" => {
    if (isRBACWizardCompleted()) return "success";
    return "default";
  };

  // Handler for Scopes wizard
  const handleStartScopesWizard = () => {
    // If already completed, clear completion state to allow restart
    if (isScopesWizardCompleted()) {
      clearWizardCompletion("scopes-wizard");
    }

    const wizardConfig = getWizardConfig("scopes-wizard");
    if (wizardConfig) {
      startWizard("scopes-wizard", wizardConfig);
    }
  };

  // Check if Scopes wizard is completed
  const isScopesWizardCompleted = () => {
    try {
      const stored = localStorage.getItem("authsec_wizards");
      if (!stored) return false;
      const data = JSON.parse(stored);
      return data.completedWizards?.includes("scopes-wizard") || false;
    } catch {
      return false;
    }
  };

  const getScopesBadge = () => {
    if (isScopesWizardCompleted()) return "Completed";
    return "Get Started";
  };

  const getScopesColor = (): "success" | "warning" | "info" | "default" => {
    if (isScopesWizardCompleted()) return "success";
    return "default";
  };

  const integrationGuides = [
    {
      id: "user-auth-setup",
      icon: Shield,
      title: "User Authentication Setup",
      description:
        "Configure SSO authentication with OIDC or SAML2 providers for your users",
      ctaLabel: isUserAuthWizardCompleted()
        ? "Setup Another"
        : "Setup User Authentication",
      ctaLink: "#",
      onCustomAction: handleStartUserAuthWizard,
      statusBadge: isLoading ? "..." : getUserAuthBadge(),
      statusColor: getUserAuthColor(),
      secondaryAction: isUserAuthWizardCompleted()
        ? {
            label: "View Clients",
            link: "/clients/mcp",
          }
        : undefined,
      gradient:
        "from-green-50 to-emerald-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-green-500/10 dark:bg-green-400/20",
      iconColor: "text-green-600 dark:text-green-400",
    },
    {
      id: "m2m-auth",
      icon: Shield,
      title: "M2M Auth",
      description:
        "Set up machine-to-machine authentication with workload identities and SDK integration",
      ctaLabel: isM2MWizardCompleted() ? "Setup Another" : "Setup M2M Auth",
      ctaLink: "#",
      onCustomAction: handleStartM2MWizard,
      statusBadge: isLoading ? "..." : getM2MBadge(),
      statusColor: getM2MColor(),
      secondaryAction: isM2MWizardCompleted()
        ? {
            label: "View Workloads",
            link: "/clients/workloads",
          }
        : undefined,
      gradient:
        "from-blue-50 to-cyan-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
    },
    {
      id: "rbac-setup",
      icon: Shield,
      title: "RBAC Setup",
      description:
        "Configure role-based access control with permissions, roles, and role bindings",
      ctaLabel: isRBACWizardCompleted() ? "Setup Another" : "Setup RBAC",
      ctaLink: "#",
      onCustomAction: handleStartRBACWizard,
      statusBadge: isLoading ? "..." : getRBACBadge(),
      statusColor: getRBACColor(),
      secondaryAction: isRBACWizardCompleted()
        ? {
            label: "View Permissions",
            link: "/admin/permissions",
          }
        : undefined,
      gradient:
        "from-blue-50 to-cyan-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
    },
    {
      id: "scopes-setup",
      icon: Shield,
      title: "Scopes Setup",
      description:
        "Define permission scopes and boundaries for your RBAC system",
      ctaLabel: isScopesWizardCompleted() ? "Setup Another" : "Setup Scopes",
      ctaLink: "#",
      onCustomAction: handleStartScopesWizard,
      statusBadge: isLoading ? "..." : getScopesBadge(),
      statusColor: getScopesColor(),
      secondaryAction: isScopesWizardCompleted()
        ? {
            label: "View Scopes",
            link: "/admin/scopes",
          }
        : undefined,
      gradient:
        "from-amber-50 to-yellow-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-amber-500/10 dark:bg-amber-400/20",
      iconColor: "text-amber-600 dark:text-amber-400",
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="text-xl font-semibold text-foreground">
          Integration Tours
        </h2>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {integrationGuides.map((guide) => (
          <div key={guide.id}>
            <QuickActionCard {...guide} />
          </div>
        ))}
      </div>
    </div>
  );
}
