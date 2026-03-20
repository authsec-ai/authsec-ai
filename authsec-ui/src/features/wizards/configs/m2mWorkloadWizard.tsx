import { ServerCog, Code2, Server } from "lucide-react";
import type { WizardConfig } from "../types";
import { toast } from "react-hot-toast";

export const m2mWorkloadWizard: WizardConfig = {
  wizardId: "m2m-workload-wizard",
  title: "M2M Workload Setup",
  description: "Set up machine-to-machine authentication in 3 easy steps",
  steps: [
    {
      id: "check-spire-agent",
      title: "SPIRE Agent Setup",
      description:
        "Verify that at least one SPIRE agent is available in your infrastructure to attest workload identities.",
      briefDescription: "Ensure SPIRE agent is deployed and running",
      icon: <Server className="h-5 w-5" />,
      actionLabel: "Check Agents",
      actionType: "custom",
      actionPayload: {
        handler: "check-spire-agent",
      },
      completionTrigger: "manual",
    },
    {
      id: "create-workload",
      title: "Create M2M Workload",
      description:
        "Register your workload identity to receive a unique SPIFFE ID and X.509 certificate for secure authentication.",
      briefDescription: "Register your workload identity with SPIFFE ID",
      icon: <ServerCog className="h-5 w-5" />,
      actionLabel: "Create Workload",
      actionType: "navigate",
      actionPayload: {
        route: "/clients/workloads/create",
      },
      completionTrigger: "navigation-return",
    },
    {
      id: "integrate-sdk",
      title: "Integrate SDK",
      description:
        "Add the AuthSec SDK to your application to attest workload identity and access secure APIs.",
      briefDescription: "Add AuthSec SDK to access secure APIs",
      icon: <Code2 className="h-5 w-5" />,
      actionLabel: "View Integration Guide",
      actionType: "dialog",
      actionPayload: {
        contentId: "sdk-attestation",
      },
      completionTrigger: "auto",
    },
  ],
  onComplete: () => {
    toast.success("M2M workload setup complete! 🎉");
  },
  onSkip: () => {
    toast("Wizard skipped. You can restart it anytime from the dashboard.", {
      icon: "👋",
    });
  },
};
