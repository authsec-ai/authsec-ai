import React, { useState } from "react";
import { Code2, Shield, Package, Lock } from "lucide-react";
import { SDKIntegrationCard } from "./SDKIntegrationCard";
import { ClientSelectionModal } from "./ClientSelectionModal";
import { ServiceSelectionModal } from "./ServiceSelectionModal";
import { useNavigate } from "react-router-dom";
import { useGetExternalServicesQuery } from "../../../app/api/externalServiceApi";
import { useGetAllClientsQuery } from "../../../app/api/clientApi";
import { SessionManager } from "../../../utils/sessionManager";
import { toast } from "react-hot-toast";

export function SDKIntegrationSections() {
  const navigate = useNavigate();
  const [showClientSelection, setShowClientSelection] = useState(false);
  const [showServiceSelection, setShowServiceSelection] = useState(false);

  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id || "";

  // Fetch external services to get a service ID for SDK page
  const { data: servicesData } = useGetExternalServicesQuery(undefined, {
    skip: !tenantId,
  });

  // Fetch clients for SDK integration
  const { data: clientsData } = useGetAllClientsQuery(
    { tenant_id: tenantId },
    { skip: !tenantId }
  );

  const handleAuthSDKClick = () => {
    const clients = clientsData?.clients || [];
    if (clients.length > 0) {
      // Show modal to select a client
      setShowClientSelection(true);
    } else {
      // Show toast and navigate to clients page
      toast.error("No clients present. Create a client first.");
      navigate("/clients/mcp");
    }
  };

  const handleExternalServicesClick = () => {
    const services = servicesData || [];
    if (services.length > 0) {
      // Show modal to select a service
      setShowServiceSelection(true);
    } else {
      // Show toast and navigate to add service page
      toast.error("No services present. Create a service first.");
      navigate("/external-services/add");
    }
  };

  const handleRBACClick = () => {
    navigate("/sdk/rbac?module=rbac");
  };

  const sdkCards = [
    {
      id: "auth-sdk",
      icon: Shield,
      title: "Integrate Client to Auth SDK",
      description:
        "Add enterprise authentication to your client applications with support for email/password, OAuth providers, and WebAuthn MFA",
      ctaLabel: "View SDK Integration",
      onClick: handleAuthSDKClick,
      gradient:
        "from-blue-50 to-cyan-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
      badge: "Quick Setup",
      badgeColor: "info" as const,
    },
    {
      id: "external-services-sdk",
      icon: Package,
      title: "Integrate Secrets & Services SDK",
      description:
        "Connect to third-party services and securely manage API secrets, credentials, and external integrations with our SDK",
      ctaLabel: "View Services SDK",
      onClick: handleExternalServicesClick,
      gradient:
        "from-green-50 to-emerald-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-green-500/10 dark:bg-green-400/20",
      iconColor: "text-green-600 dark:text-green-400",
    },
    {
      id: "rbac-sdk",
      icon: Lock,
      title: "Implement RBAC SDK Integration",
      description:
        "Fine-grained role-based access control with support for custom roles, resources, scopes, and permission management",
      ctaLabel: "View RBAC SDK",
      onClick: handleRBACClick,
      gradient:
        "from-blue-50 to-sky-50 dark:from-neutral-800/50 dark:to-neutral-700/30",
      iconBg: "bg-blue-500/10 dark:bg-blue-400/20",
      iconColor: "text-blue-600 dark:text-blue-400",
    },
  ];

  return (
    <>
      <div className="space-y-6">
        {/* Section Header */}
        <div className="flex items-center gap-3">
          <div className="p-2 bg-gradient-to-br from-blue-500 to-cyan-500 rounded-lg">
            <Code2 className="h-5 w-5 text-white" />
          </div>
          <div>
            <h2 className="text-xl font-semibold text-foreground">
              SDK Integration
            </h2>
            <p className="text-sm text-foreground">
              Integrate your applications with our SDKs and services
            </p>
          </div>
        </div>

        {/* SDK Cards Grid */}
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {sdkCards.map((card) => (
            <div key={card.id}>
              <SDKIntegrationCard {...card} />
            </div>
          ))}
        </div>

        {/* Helper Text */}
        <div className="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4">
          <p className="text-sm text-blue-800 dark:text-blue-200">
            <strong>Tip:</strong> For detailed SDK integration guides and code
            examples, select a client to view step-by-step instructions for your
            specific authentication setup.
          </p>
        </div>
      </div>

      {/* Client Selection Modal */}
      <ClientSelectionModal
        isOpen={showClientSelection}
        onClose={() => setShowClientSelection(false)}
      />

      {/* Service Selection Modal */}
      <ServiceSelectionModal
        isOpen={showServiceSelection}
        onClose={() => setShowServiceSelection(false)}
      />
    </>
  );
}
