import type { WizardConfig } from "../types";
import { toast } from "@/lib/toast";

export const scopesWizard: WizardConfig = {
  wizardId: "scopes-wizard",
  title: "Scopes Setup",
  description: "Set up permission scopes in 2 easy steps",
  steps: [
    {
      id: "select-context",
      title: "Select Context",
      briefDescription: "Choose whether to configure scopes for Admin or End Users",
      description:
        "Select the audience for your scope configuration. Admin scopes control operator access, while End User scopes manage application user permissions.",
      actionLabel: "Select Context",
      actionType: "custom",
      actionPayload: { handler: "select-context" },
      completionTrigger: "manual",
    },
    {
      id: "create-scope",
      title: "Create Scope",
      briefDescription: "Define a scope with associated resources",
      description:
        "Create a new scope by giving it a name and selecting the resources it can access. Scopes define permission boundaries in your RBAC system.",
      actionLabel: "Create Scope",
      actionType: "navigate",
      actionPayload: { route: "scopes?openModal=create" },
      completionTrigger: "navigation-return",
    },
  ],
  onComplete: () => {
    toast.success("Scopes setup complete! 🎉");
  },
  onSkip: () => {
    toast("Wizard skipped. You can restart it anytime from the dashboard.", {
      icon: "👋",
    });
  },
};
