import { useCallback, useState } from "react";
import {
  GripVertical,
  SkipForward,
  CheckCircle,
  ExternalLink,
  Eye,
  Shield,
  Key,
  Link2,
} from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useWizard } from "@/contexts/WizardContext";
import { WizardProgress } from "./WizardProgress";
import { Button } from "@/components/ui/button";
import { ClientAuthMethodsModal } from "@/features/clients/components/ClientAuthMethodsModal";
import { useGetAllClientsQuery } from "@/app/api/clientApi";
import { SessionManager } from "@/utils/sessionManager";
import { toast } from "react-hot-toast";
import type { ClientWithAuthMethods } from "@/types/entities";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import { generateOAuth2AuthorizationUrl } from "@/utils/oauthUtils";

interface AppRightSidebarWizardProps {
  onClose: () => void;
  onWidthChange?: (width: number) => void;
  initialWidth?: number;
}

// M2M Workload Wizard Completion View
function M2MCompletionView({ onClose }: { onClose: () => void }) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6 bg-background">
      <div className="max-w-md text-center space-y-4">
        <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
          <CheckCircle className="w-8 h-8 text-primary" />
        </div>
        <h2 className="text-2xl font-semibold text-foreground">
          Setup Complete!
        </h2>
        <p className="text-sm text-muted-foreground">
          You've successfully completed the M2M workload setup. You can now
          close this panel and continue exploring the features.
        </p>
        <Button
          onClick={onClose}
          className="mt-6 px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
        >
          Close
        </Button>
      </div>
    </div>
  );
}

// RBAC Wizard Completion View
function RbacCompletionView({ onClose }: { onClose: () => void }) {
  const contextualNavigate = useContextualNavigate();
  const { setIsAwaitingPlatformAction } = useWizard();

  const handleViewPermissions = () => {
    setIsAwaitingPlatformAction(true);
    contextualNavigate("permissions");
  };

  const handleViewRoles = () => {
    setIsAwaitingPlatformAction(true);
    contextualNavigate("roles");
  };

  const handleViewBindings = () => {
    setIsAwaitingPlatformAction(true);
    contextualNavigate("role-bindings");
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6 bg-background">
      <div className="max-w-md text-center space-y-4">
        <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
          <CheckCircle className="w-8 h-8 text-primary" />
        </div>
        <h2 className="text-2xl font-semibold text-foreground">
          RBAC Setup Complete!
        </h2>
        <p className="text-sm text-muted-foreground">
          You've successfully configured role-based access control. Your
          permissions, roles, and bindings are now set up and ready to use.
        </p>
        <div className="flex flex-col gap-2 mt-6">
          <Button
            onClick={handleViewPermissions}
            variant="outline"
            className="w-full justify-start"
          >
            <Shield className="mr-2 h-4 w-4" />
            View Permissions
          </Button>
          <Button
            onClick={handleViewRoles}
            variant="outline"
            className="w-full justify-start"
          >
            <Key className="mr-2 h-4 w-4" />
            View Roles
          </Button>
          <Button
            onClick={handleViewBindings}
            variant="outline"
            className="w-full justify-start"
          >
            <Link2 className="mr-2 h-4 w-4" />
            View Role Bindings
          </Button>
        </div>
        <Button
          onClick={onClose}
          className="mt-4 px-6 py-2 bg-primary text-primary-foreground rounded-lg hover:bg-primary/90 transition-colors"
        >
          Close
        </Button>
      </div>
    </div>
  );
}

// Scopes Wizard Completion View
function ScopesCompletionView({ onClose }: { onClose: () => void }) {
  const contextualNavigate = useContextualNavigate();
  const { setIsAwaitingPlatformAction } = useWizard();

  const handleViewScopes = () => {
    setIsAwaitingPlatformAction(true);
    contextualNavigate("scopes");
  };

  return (
    <div className="flex-1 flex flex-col items-center justify-center p-6 bg-background">
      <div className="max-w-md text-center space-y-4">
        {/* Success icon */}
        <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
          <CheckCircle className="w-8 h-8 text-primary" />
        </div>

        {/* Title and message */}
        <h2 className="text-2xl font-semibold text-foreground">
          Scopes Setup Complete!
        </h2>
        <p className="text-sm text-muted-foreground">
          You've successfully created a scope. Scopes define permission
          boundaries in your RBAC system.
        </p>

        {/* Action buttons */}
        <div className="flex flex-col gap-2 mt-6">
          <Button onClick={handleViewScopes} className="w-full">
            <Shield className="mr-2 h-4 w-4" />
            View Scopes
          </Button>
          <Button variant="outline" onClick={onClose} className="w-full">
            Close
          </Button>
        </div>
      </div>
    </div>
  );
}

// User Auth Wizard Completion View
function UserAuthCompletionView({
  clientId,
  authMethodType,
  onClose,
}: {
  clientId: string | undefined;
  authMethodType?: "oidc" | "saml2";
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const { setIsAwaitingPlatformAction } = useWizard();
  const session = SessionManager.getSession();
  const [authModalOpen, setAuthModalOpen] = useState(false);
  const [authModalClient, setAuthModalClient] =
    useState<ClientWithAuthMethods | null>(null);

  // Fetch clients for Edit Auth Methods action
  const { data: clientsData } = useGetAllClientsQuery(
    { tenant_id: session?.tenant_id || "", active_only: false },
    { skip: !session?.tenant_id },
  );

  const handleViewClient = () => {
    setIsAwaitingPlatformAction(true);
    navigate("/clients/mcp");
  };

  const handlePreviewLogin = async () => {
    if (!clientId) return;

    // Priority: 1) session tenant_domain, 2) extract from current hostname
    let tenantDomainForOAuth = session?.tenant_domain;
    let tenantDomainFromHostname: string | undefined;

    // Only extract from hostname if not found in session
    if (!tenantDomainForOAuth) {
      // Extract from hostname: dec10.app.authsec.dev -> dec10
      const hostname = window.location.hostname;
      const hostParts = hostname.split(".");
      if (
        hostParts.length >= 4 &&
        hostParts[0] !== "app" &&
        hostParts[0] !== "www"
      ) {
        tenantDomainFromHostname = hostParts[0];
        tenantDomainForOAuth = tenantDomainFromHostname;
      }
    }

    // eslint-disable-next-line no-console
    console.log("[PreviewLogin] 🔐 Generating OAuth URL with:", {
      clientId,
      tenantDomainFromSession: session?.tenant_domain,
      tenantDomainFromHostname,
      finalTenantDomain: tenantDomainForOAuth,
    });

    try {
      // Generate the OAuth2 authorization URL with PKCE
      const { authorizationUrl } = await generateOAuth2AuthorizationUrl({
        clientId,
        tenantDomain: tenantDomainForOAuth,
        scopes: ["openid", "profile", "email"],
      });

      window.open(authorizationUrl, "_blank", "noopener,noreferrer");
      toast.success("Opening login preview in a new tab");
    } catch (error) {
      console.error("Failed to generate OAuth2 URL:", error);
      toast.error("Failed to generate login preview URL");
    }
  };

  const handleEditAuthMethods = () => {
    if (!clientId || !clientsData?.clients) return;

    const client = clientsData.clients.find((c) => c.client_id === clientId);
    if (client) {
      // Transform to ClientWithAuthMethods format
      const clientWithAuthMethods: ClientWithAuthMethods = {
        id: client.client_id,
        name: client.name,
        workspace_id: client.tenant_id,
        secret_id: null,
        description: null,
        type: "mcp_server",
        tags: "",
        authentication_type: "sso",
        metadata: {
          raw_client: client,
        },
        roles: [],
        mfa_config: null,
        successful_authentications: null,
        denied_authentications: null,
        endpoint: "",
        access_status: "active",
        created_at: client.created_at || new Date().toISOString(),
        updated_at: client.updated_at || new Date().toISOString(),
        last_accessed: null,
        attachedMethods: [],
      };

      setAuthModalClient(clientWithAuthMethods);
      setAuthModalOpen(true);
    }
  };

  return (
    <>
      <div className="flex-1 flex flex-col items-center justify-center p-6 bg-background">
        <div className="max-w-md text-center space-y-6">
          {/* Success Icon */}
          <div className="w-16 h-16 bg-primary/10 rounded-full flex items-center justify-center mx-auto">
            <CheckCircle className="w-8 h-8 text-primary" />
          </div>

          {/* Title & Description */}
          <div className="space-y-2">
            <h2 className="text-2xl font-semibold text-foreground">
              Authentication Setup Complete!
            </h2>
            <p className="text-sm text-muted-foreground">
              Your user authentication is now configured. Choose what to do
              next:
            </p>
          </div>

          {/* Action Buttons - Centered in column */}
          <div className="flex flex-col gap-3 w-full max-w-sm mx-auto pt-4">
            {/* Button 1: View Client */}
            <Button
              onClick={handleViewClient}
              variant="outline"
              className="w-full justify-start h-12 text-sm"
            >
              <ExternalLink className="mr-3 h-4 w-4" />
              View Client
            </Button>

            {/* Button 2: Preview Login Page */}
            <Button
              onClick={handlePreviewLogin}
              variant="outline"
              className="w-full justify-start h-12 text-sm"
              disabled={!clientId}
            >
              <Eye className="mr-3 h-4 w-4" />
              Preview Login Page
            </Button>

            {/* Button 3: Edit Auth Methods - Hidden for SAML */}
            {authMethodType !== "saml2" && (
              <Button
                onClick={handleEditAuthMethods}
                variant="outline"
                className="w-full justify-start h-12 text-sm"
                disabled={!clientId}
              >
                <Shield className="mr-3 h-4 w-4" />
                Edit Auth Methods
              </Button>
            )}

            {/* Divider */}
            <div className="border-t border-border my-2" />

            {/* Button 4: Close */}
            <Button
              onClick={onClose}
              className="w-full h-12 bg-[var(--brand-blue-600)] text-white hover:bg-[var(--brand-blue-700)]"
            >
              Close Wizard
            </Button>
          </div>
        </div>
      </div>

      {/* Auth Methods Modal */}
      {authModalOpen && authModalClient && (
        <ClientAuthMethodsModal
          client={authModalClient}
          open={authModalOpen}
          onClose={() => setAuthModalOpen(false)}
        />
      )}
    </>
  );
}

export function AppRightSidebarWizard({
  onClose,
  onWidthChange,
  initialWidth = 520,
}: AppRightSidebarWizardProps) {
  const {
    wizardConfig,
    currentStep,
    completedSteps,
    isCompleted,
    skipWizard,
    dismissWizard,
    completeStep,
    getStepStatus,
    wizardCompletionData,
  } = useWizard();

  const [width, setWidth] = useState(initialWidth);
  const [isResizing, setIsResizing] = useState(false);

  const handleMouseDown = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      setIsResizing(true);

      const startX = e.clientX;
      const startWidth = width;

      const handleMouseMove = (e: MouseEvent) => {
        e.preventDefault();
        const deltaX = startX - e.clientX;
        const newWidth = Math.max(480, Math.min(800, startWidth + deltaX));
        setWidth(newWidth);
        onWidthChange?.(newWidth);
      };

      const handleMouseUp = () => {
        setIsResizing(false);
        document.body.style.cursor = "";
        document.body.style.userSelect = "";
        document.removeEventListener("mousemove", handleMouseMove);
        document.removeEventListener("mouseup", handleMouseUp);
      };

      document.body.style.cursor = "ew-resize";
      document.body.style.userSelect = "none";
      document.addEventListener("mousemove", handleMouseMove);
      document.addEventListener("mouseup", handleMouseUp);
    },
    [width, onWidthChange],
  );

  const handleClose = () => {
    // Only call dismissWizard if wizard is still active (not completed)
    if (!isCompleted) {
      dismissWizard();
    }
    onClose();
  };

  const handleSkip = () => {
    skipWizard();
    onClose();
  };

  // If wizard is completed, show wizard-specific completion view
  if (isCompleted) {
    const wizardId = wizardCompletionData?.wizardId;
    const clientId = wizardCompletionData?.clientId;

    return (
      <div
        className="h-full bg-background border-l border-border flex shadow-2xl"
        style={{ width: `${width}px` }}
      >
        {/* Resize Handle */}
        <div
          className="w-1 bg-border hover:bg-primary/50 cursor-ew-resize flex items-center justify-center group transition-all duration-200 relative"
          onMouseDown={handleMouseDown}
        >
          <div className="absolute inset-y-0 -left-2 -right-2 flex items-center justify-center">
            <GripVertical className="w-3 h-3 text-muted-foreground group-hover:text-primary transition-colors" />
          </div>
        </div>

        {/* Conditional Completion Views Based on Wizard ID */}
        {wizardId === "user-auth-wizard" ? (
          <UserAuthCompletionView
            clientId={clientId}
            authMethodType={wizardCompletionData?.authMethodType}
            onClose={onClose}
          />
        ) : wizardId === "m2m-workload-wizard" ? (
          <M2MCompletionView onClose={onClose} />
        ) : wizardId === "rbac-wizard" ? (
          <RbacCompletionView onClose={onClose} />
        ) : wizardId === "scopes-wizard" ? (
          <ScopesCompletionView onClose={onClose} />
        ) : (
          /* Fallback for unknown wizards */
          <M2MCompletionView onClose={onClose} />
        )}
      </div>
    );
  }

  if (!wizardConfig) {
    return null;
  }

  const totalSteps = wizardConfig.steps.length;

  return (
    <div
      className="h-full bg-background border-l border-border flex shadow-2xl"
      style={{ width: `${width}px` }}
    >
      {/* Resize Handle */}
      <div
        className="w-1 bg-border hover:bg-primary/50 cursor-ew-resize flex items-center justify-center group transition-all duration-200 relative"
        onMouseDown={handleMouseDown}
      >
        <div className="absolute inset-y-0 -left-2 -right-2 flex items-center justify-center">
          <GripVertical className="w-3 h-3 text-muted-foreground group-hover:text-primary transition-colors" />
        </div>
        {isResizing && (
          <div className="absolute -top-8 left-1/2 transform -translate-x-1/2 bg-popover text-popover-foreground px-2 py-1 rounded text-xs shadow-lg z-50 border">
            {width}px
          </div>
        )}
      </div>

      {/* Wizard Interface */}
      <div className="flex-1 flex flex-col bg-background overflow-y-auto">
        {/* Scrollable Step Timeline with Inline Content */}
        <WizardProgress
          totalSteps={totalSteps}
          currentStep={currentStep}
          completedSteps={completedSteps}
          steps={wizardConfig.steps}
          title={wizardConfig.title}
          onClose={handleClose}
          onCompleteStep={completeStep}
          getStepStatus={getStepStatus}
        />

        {/* Bottom: Footer with Skip */}
        {/* <div className="px-4 py-3 border-t border-border mt-auto">
          <Button
            variant="ghost"
            size="sm"
            onClick={handleSkip}
            className="w-full text-muted-foreground hover:text-foreground"
          >
            <SkipForward className="mr-2 h-4 w-4" />
            Skip Tour
          </Button>
        </div> */}
      </div>
    </div>
  );
}
