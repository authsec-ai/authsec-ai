import { Shield, Users, Key, Link2 } from "lucide-react";
import type { WizardConfig } from "../types";
import { toast } from "react-hot-toast";

export const rbacWizard: WizardConfig = {
  wizardId: "rbac-wizard",
  title: "RBAC Setup",
  description: "Set up role-based access control in 4 easy steps",
  steps: [
    {
      id: "select-context",
      title: "Select Context",
      description:
        "Choose whether to configure RBAC for Admin users or End Users.",
      briefDescription: "Choose Admin or End User context",
      icon: <Users className="h-5 w-5" />,
      actionLabel: "Select Context",
      actionType: "custom",
      actionPayload: {
        handler: "select-context",
      },
      completionTrigger: "manual",
    },
    {
      id: "create-permissions",
      title: "Create Permissions & Resources",
      description:
        "Define granular permissions for resources that control what actions can be performed.",
      briefDescription: "Define permissions for resources",
      icon: <Shield className="h-5 w-5" />,
      actionLabel: "Create Permissions",
      actionType: "navigate",
      actionPayload: {
        route: "permissions?openModal=create",
      },
      completionTrigger: "navigation-return",
    },
    {
      id: "create-roles",
      title: "Create Roles & Map Permissions",
      description:
        "Bundle permissions into reusable roles that can be assigned to users.",
      briefDescription: "Create roles and map permissions",
      icon: <Key className="h-5 w-5" />,
      actionLabel: "Create Roles",
      actionType: "navigate",
      actionPayload: {
        route: "roles?openModal=create",
      },
      completionTrigger: "navigation-return",
    },
    {
      id: "create-bindings",
      title: "Create Role Bindings",
      description:
        "Assign roles to users with specific scopes to complete the access control setup.",
      briefDescription: "Assign roles to users with scopes",
      icon: <Link2 className="h-5 w-5" />,
      actionLabel: "Create Bindings",
      actionType: "navigate",
      actionPayload: {
        route: "role-bindings?openModal=create",
      },
      completionTrigger: "navigation-return",
    },
  ],
  onComplete: () => {
    toast.success("RBAC setup complete! 🎉");
  },
  onSkip: () => {
    toast("Wizard skipped. You can restart it anytime from the dashboard.", {
      icon: "👋",
    });
  },
};
