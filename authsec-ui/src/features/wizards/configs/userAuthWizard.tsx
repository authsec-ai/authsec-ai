import { KeyRound, Shield, Code2 } from "lucide-react";
import type { WizardConfig } from "../types";
import { toast } from "react-hot-toast";

export const userAuthWizard: WizardConfig = {
  wizardId: "user-auth-wizard",
  title: "User Authentication Setup",
  description: "Set up SSO authentication for your users in 4 easy steps",
  steps: [
    {
      id: "choose-auth-method",
      title: "Choose Authentication Method",
      description:
        "Select between OIDC (OAuth 2.0 / OpenID Connect) or SAML2 for SSO authentication.",
      briefDescription: "Choose OIDC or SAML2 authentication",
      icon: <Shield className="h-5 w-5" />,
      actionLabel: "Select Auth Method",
      actionType: "custom",
      actionPayload: {
        handler: "choose-auth-method",
      },
      completionTrigger: "manual",
    },
    {
      id: "configure-auth",
      title: "Configure SSO Authentication",
      description:
        "Set up your chosen authentication provider with client credentials and configuration.",
      briefDescription: "Configure authentication provider settings",
      icon: <Shield className="h-5 w-5" />,
      actionLabel: "Configure Authentication",
      actionType: "custom",
      actionPayload: {
        handler: "configure-auth",
      },
      completionTrigger: "navigation-return",
    },
    {
      id: "client-selection",
      title: "Select Client",
      description:
        "Choose an existing client or create a new one to configure authentication.",
      briefDescription: "Select or create a client for authentication",
      icon: <KeyRound className="h-5 w-5" />,
      actionLabel: "Continue with Selected Client",
      actionType: "custom",
      actionPayload: {
        handler: "client-selection",
      },
      completionTrigger: "manual",
    },
    {
      id: "integrate-sdk",
      title: "Integrate SDK",
      description:
        "Add the AuthSec SDK to your application to enable user authentication. On the next page, you'll find tabs for MCP Server and AI Agent - click each tab to view the SDK integration code specific to your use case.",
      briefDescription: "Add AuthSec SDK for user authentication",
      icon: <Code2 className="h-5 w-5" />,
      actionLabel: "View SDK Integration",
      actionType: "navigate",
      actionPayload: {
        route: "/sdk/clients/:clientId",
      },
      completionTrigger: "navigation-return",
    },
  ],
  onComplete: () => {
    toast.success("User authentication setup complete! 🎉");
  },
  onSkip: () => {
    toast("Wizard skipped. You can restart it anytime from the dashboard.", {
      icon: "👋",
    });
  },
};
