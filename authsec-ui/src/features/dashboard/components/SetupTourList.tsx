import React from "react";
import { ScanSearch, Server, ShieldCheck, User } from "lucide-react";
import { useWizardStatus } from "../hooks/useWizardStatus";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { DashboardSection } from "./DashboardSection";
import { DashboardTaskRow } from "./DashboardTaskRow";

export function SetupTourList() {
  const { audience } = useRbacAudience();
  const m2m = useWizardStatus("m2m-workload-wizard");
  const rbac = useWizardStatus("rbac-wizard");
  const scopes = useWizardStatus("scopes-wizard");
  const userAuth = useWizardStatus("user-auth-wizard");

  const toursConfig = [
    {
      id: "m2m-auth",
      icon: Server,
      title: "M2M Auth",
      description:
        "Set up machine-to-machine authentication with workload identities",
      wizard: m2m,
      route: "/clients/workloads",
      routeLabel: "Workloads",
      color: "blue" as const,
      sortOrder: 1,
    },
    {
      id: "rbac",
      icon: ShieldCheck,
      title: "RBAC Setup",
      description:
        "Configure role-based access control with permissions and roles",
      wizard: rbac,
      route: `/${audience}/permissions`,
      routeLabel: "Permissions",
      color: "purple" as const,
      sortOrder: 2,
    },
    {
      id: "scopes",
      icon: ScanSearch,
      title: "Scopes Setup",
      description:
        "Define permission scopes and boundaries for your RBAC system",
      wizard: scopes,
      route: `/${audience}/scopes`,
      routeLabel: "Scopes",
      color: "amber" as const,
      sortOrder: 3,
    },
    {
      id: "user-auth",
      icon: User,
      title: "User Auth",
      description:
        "Set up user authentication with OAuth, SAML, and social providers",
      wizard: userAuth,
      route: "/clients/mcp",
      routeLabel: "Clients",
      color: "green" as const,
      sortOrder: 4,
    },
  ];

  // Sort: incomplete first, completed last (maintain order within groups)
  const sortedTours = React.useMemo(() => {
    return [...toursConfig].sort((a, b) => {
      if (a.wizard.isCompleted !== b.wizard.isCompleted) {
        return a.wizard.isCompleted ? 1 : -1; // incomplete first
      }
      return a.sortOrder - b.sortOrder; // maintain order
    });
  }, [
    m2m.isCompleted,
    rbac.isCompleted,
    scopes.isCompleted,
    userAuth.isCompleted,
  ]);

  // Map to card props
  const tours = sortedTours.map((config) => ({
    id: config.id,
    icon: config.icon,
    title: config.title,
    description: config.description,
    isCompleted: config.wizard.isCompleted,
    onPrimaryAction: () => {
      config.wizard.launch();
    },
    color: config.color,
  }));

  return (
    <DashboardSection
      label="Setup"
      title="Guided Setup Queue"
      description="Priority-ordered workflows for activation, access control, and onboarding."
      contentClassName="space-y-2"
    >
      <div className="dash-group-panel divide-y dash-divider">
        {tours.map((tour) => (
          <DashboardTaskRow
            key={tour.id}
            icon={tour.icon}
            iconTone={
              tour.color === "blue"
                ? "accent"
                : tour.color === "purple"
                  ? "accent"
                  : tour.color === "amber"
                    ? "warning"
                    : "success"
            }
            title={tour.title}
            description={tour.description}
            isCompleted={tour.isCompleted}
            statusLabel={tour.isCompleted ? "Complete" : "Ready"}
            statusTone={tour.isCompleted ? "success" : "neutral"}
            primaryActionLabel={tour.primaryActionLabel}
            onPrimaryAction={tour.onPrimaryAction}
            secondaryActionLabel={tour.secondaryActionLabel}
            onSecondaryAction={tour.onSecondaryAction}
            framed={false}
            primaryActionStyle="inline"
            revealActionsOnHover
          />
        ))}
      </div>
    </DashboardSection>
  );
}
