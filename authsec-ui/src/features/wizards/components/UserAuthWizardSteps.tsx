import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { Label } from "@/components/ui/label";
import { OnboardClientModal } from "@/features/clients/components/OnboardClientModal";
import { useGetAllClientsQuery } from "@/app/api/clientApi";
import { SessionManager } from "@/utils/sessionManager";
import { cn } from "@/lib/utils";
import {
  Check,
  ChevronRight,
  Plus,
  CheckCircle,
  AlertCircle,
  ArrowLeft,
} from "lucide-react";
import { useUnifiedProviders } from "@/features/authentication/hooks/useUnifiedProviders";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { SearchableSelect } from "@/components/ui/searchable-select";

// Provider SVG Icons (from CreateAuthMethodPage.tsx)
const GoogleIcon = ({ className }: { className?: string }) => (
  <svg className={cn("h-6 w-6", className)} viewBox="0 0 24 24">
    <path
      fill="#4285f4"
      d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
    />
    <path
      fill="#34a853"
      d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
    />
    <path
      fill="#fbbc05"
      d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
    />
    <path
      fill="#ea4335"
      d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
    />
  </svg>
);

const MicrosoftIcon = ({ className }: { className?: string }) => (
  <svg
    className={cn("h-6 w-6", className)}
    viewBox="0 0 23 23"
    fill="currentColor"
  >
    <path d="M0 0h11v11H0z" fill="#f25022" />
    <path d="M12 0h11v11H12z" fill="#00a4ef" />
    <path d="M0 12h11v11H0z" fill="#ffb900" />
    <path d="M12 12h11v11H12z" fill="#7fba00" />
  </svg>
);

const GitHubIcon = ({ className }: { className?: string }) => (
  <svg
    className={cn("h-6 w-6", className)}
    fill="#054ddcff"
    viewBox="0 0 20 20"
  >
    <path
      fillRule="evenodd"
      d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
      clipRule="evenodd"
    />
  </svg>
);

// Step 1: Client Selection
interface ClientSelectionStepProps {
  onComplete: (clientId: string) => void;
}

export function ClientSelectionStep({ onComplete }: ClientSelectionStepProps) {
  const session = SessionManager.getSession();
  const tenantId = session?.tenant_id;

  const {
    data: clientsResponse,
    isLoading,
    refetch,
  } = useGetAllClientsQuery(
    {
      tenant_id: tenantId || "",
      active_only: false,
    },
    { skip: !tenantId },
  );

  const [selectionMode, setSelectionMode] = useState<
    "default" | "existing" | "new"
  >("default");
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [selectedClientId, setSelectedClientId] = useState<string>("");

  const clients = clientsResponse?.clients || [];
  const defaultClient = clients[0];

  const handleModeChange = (mode: "default" | "existing" | "new") => {
    setSelectionMode(mode);
    // Clear selectedClientId when switching to default mode
    if (mode === "default") {
      setSelectedClientId("");
    }
  };

  const handleContinue = () => {
    let clientId = "";

    if (selectionMode === "default") {
      clientId = defaultClient?.client_id || "";
    } else if (selectionMode === "existing" || selectionMode === "new") {
      clientId = selectedClientId;
    }

    if (clientId) {
      onComplete(clientId);
    }
  };

  const handleClientCreated = (clientId: string) => {
    // Refetch clients to get the newly created one
    refetch();
    setShowCreateModal(false);
    setSelectionMode("new");
    setSelectedClientId(clientId);

    // Auto-complete step after client creation
    setTimeout(() => {
      if (clientId) {
        onComplete(clientId);
      }
    }, 500);
  };

  return (
    <div className="space-y-4">
      <RadioGroup
        value={selectionMode}
        onValueChange={(v) =>
          handleModeChange(v as "default" | "existing" | "new")
        }
      >
        {/* Use Default Client Option */}
        <div
          className={cn(
            "flex items-start space-x-3 rounded-lg border p-4 cursor-pointer transition-all",
            selectionMode === "default" && "border-primary bg-primary/5",
          )}
          onClick={() => handleModeChange("default")}
        >
          <RadioGroupItem value="default" id="default" />
          <div className="flex-1">
            <Label htmlFor="default" className="cursor-pointer font-medium">
              Use Default Client
            </Label>
            {defaultClient ? (
              <div className="mt-2 p-3 bg-muted rounded-md text-sm space-y-1">
                <p className="text-xs text-muted-foreground">Client ID:</p>
                <p className="font-mono text-xs break-all">
                  {defaultClient.client_id}
                </p>
                <p className="text-xs text-muted-foreground mt-2">Name:</p>
                <p className="font-medium text-xs">{defaultClient.name}</p>
              </div>
            ) : (
              <p className="text-sm text-muted-foreground mt-1">
                {isLoading ? "Loading..." : "No clients available"}
              </p>
            )}
          </div>
          {selectionMode === "default" && (
            <Check className="h-5 w-5 text-primary shrink-0" />
          )}
        </div>

        {/* Select from Existing Clients Option */}
        {clients.length > 1 && (
          <div
            className={cn(
              "flex items-start space-x-3 rounded-lg border p-4 cursor-pointer transition-all",
              selectionMode === "existing" && "border-primary bg-primary/5",
            )}
            onClick={() => handleModeChange("existing")}
          >
            <RadioGroupItem value="existing" id="existing" />
            <div className="flex-1">
              <Label htmlFor="existing" className="cursor-pointer font-medium">
                Select Existing Client
              </Label>
              <p className="text-sm text-muted-foreground mt-1">
                Choose from {clients.length - 1} other available client
                {clients.length > 2 ? "s" : ""}
              </p>

              {selectionMode === "existing" && (
                <div className="mt-3" onClick={(e) => e.stopPropagation()}>
                  <SearchableSelect
                    options={clients.slice(1).map((client) => ({
                      value: client.client_id,
                      label: client.name,
                      description: client.client_id,
                    }))}
                    value={selectedClientId}
                    onChange={(value) => setSelectedClientId(value || "")}
                    placeholder="Search and select a client..."
                    searchPlaceholder="Search clients..."
                    emptyText="No clients found"
                    clearable={true}
                    className="w-full"
                  />
                </div>
              )}
            </div>
            {selectionMode === "existing" && selectedClientId && (
              <Check className="h-5 w-5 text-primary shrink-0" />
            )}
          </div>
        )}

        {/* Create New Client Option */}
        <div
          className={cn(
            "flex items-start space-x-3 rounded-lg border p-4 cursor-pointer transition-all",
            selectionMode === "new" && "border-primary bg-primary/5",
          )}
          onClick={() => handleModeChange("new")}
        >
          <RadioGroupItem value="new" id="new" />
          <div className="flex-1">
            <Label htmlFor="new" className="cursor-pointer font-medium">
              Create New Client
            </Label>
            <p className="text-sm text-muted-foreground mt-1">
              Register a new MCP Server or AI Agent client
            </p>
            {selectedClientId && selectionMode === "new" && (
              <div className="mt-2 p-3 bg-muted rounded-md text-sm">
                <p className="text-xs text-muted-foreground">Client ID:</p>
                <p className="font-mono text-xs break-all mt-1">
                  {selectedClientId}
                </p>
              </div>
            )}
            {selectionMode === "new" && !selectedClientId && (
              <Button
                onClick={(e) => {
                  e.stopPropagation();
                  setShowCreateModal(true);
                }}
                size="sm"
                variant="outline"
                className="mt-3"
              >
                <Plus className="h-4 w-4 mr-2" />
                Create Client
              </Button>
            )}
          </div>
          {selectionMode === "new" && selectedClientId && (
            <Check className="h-5 w-5 text-primary shrink-0" />
          )}
        </div>
      </RadioGroup>

      <div className="flex justify-center">
        <Button
          onClick={handleContinue}
          variant="outline"
          disabled={
            (selectionMode === "default" && !defaultClient) ||
            (selectionMode === "existing" && !selectedClientId) ||
            (selectionMode === "new" && !selectedClientId)
          }
          className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
        >
          Continue
          <ChevronRight className="ml-2 h-4 w-4" />
        </Button>
      </div>

      {/* Client Creation Modal */}
      <OnboardClientModal
        isOpen={showCreateModal}
        onClose={() => setShowCreateModal(false)}
        onSuccess={handleClientCreated}
        preventNavigation={true}
      />
    </div>
  );
}

// Step 2: Choose Auth Method Type
interface ChooseAuthMethodStepProps {
  onComplete: (methodType: "oidc" | "saml2") => void;
}

export function ChooseAuthMethodStep({
  onComplete,
}: ChooseAuthMethodStepProps) {
  const [selectedMethod, setSelectedMethod] = useState<"oidc" | "saml2" | null>(
    null,
  );

  const methods = [
    {
      id: "oidc" as const,
      title: "OIDC / OAuth 2.0",
      description:
        "OpenID Connect with support for Google, GitHub, Microsoft, and custom providers",
      icon: "🔐",
      tooltip: [
        "Ideal for modern web apps, SPAs, and developer tools.",
        "Supports PKCE, refresh tokens, and social login flows.",
      ],
    },
    {
      id: "saml2" as const,
      title: "SAML 2.0",
      description:
        "Security Assertion Markup Language for enterprise SSO providers",
      icon: "🛡️",
      tooltip: [
        "Widely used with enterprise IdPs like Okta, ADFS, and Azure AD.",
        "Uses XML-based assertions for secure identity federation.",
      ],
    },
  ];

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4">
        {methods.map((method) => (
          <Tooltip key={method.id}>
            <TooltipTrigger asChild>
              <div
                className={cn(
                  "flex items-start gap-4 p-4 rounded-lg border cursor-pointer transition-all hover:border-primary/50",
                  selectedMethod === method.id &&
                    "border-primary bg-primary/5 ring-2 ring-primary ring-offset-2",
                )}
                onClick={() => setSelectedMethod(method.id)}
              >
                <div className="text-2xl">{method.icon}</div>
                <div className="flex-1">
                  <h4 className="font-semibold text-sm">{method.title}</h4>
                  <p className="text-xs text-muted-foreground mt-1">
                    {method.description}
                  </p>
                </div>
                {selectedMethod === method.id && (
                  <Check className="h-5 w-5 text-primary shrink-0" />
                )}
              </div>
            </TooltipTrigger>
            <TooltipContent
              side="left"
              className="max-w-[250px] text-[10px] leading-relaxed"
            >
              <p className="text-white">{method.tooltip[0]}</p>
              <p className="mt-1 opacity-80 text-white">{method.tooltip[1]}</p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>

      <div className="flex justify-center">
        <Button
          onClick={() => selectedMethod && onComplete(selectedMethod)}
          disabled={!selectedMethod}
          variant="outline"
          className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
        >
          Continue with {selectedMethod?.toUpperCase()}
          <ChevronRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}

// Step 3: Configure Auth (with provider selection for OIDC)
interface ConfigureAuthStepProps {
  authMethodType: "oidc" | "saml2";
  onNavigate: (provider?: string) => void;
  onComplete: () => void;
}

export function ConfigureAuthStep({
  authMethodType,
  onNavigate,
  onComplete,
}: ConfigureAuthStepProps) {
  const session = SessionManager.getSession();
  const tenantId = session?.tenant_id;

  // State for provider selection and existing config display
  const [selectedProvider, setSelectedProvider] = useState<string | null>(null);
  const [showExistingConfig, setShowExistingConfig] = useState(false);
  const [selectedProviderData, setSelectedProviderData] = useState<any>(null);

  // Check for existing auth methods
  const { providers, isLoading } = useUnifiedProviders({
    tenant_id: tenantId || "",
    client_id: undefined, // Tenant-wide check
  });

  // Filter providers by selected auth method type
  const relevantProviders = useMemo(() => {
    return providers.filter((p) => p.provider_type === authMethodType);
  }, [providers, authMethodType]);

  const oidcProviders = [
    {
      id: "google",
      name: "Google",
      description: "Google OAuth 2.0 / OpenID Connect",
      icon: <GoogleIcon />,
      tooltip: [
        "Sign in with Google accounts via OAuth 2.0.",
        "Requires a Google Cloud Console OAuth app.",
      ],
    },
    {
      id: "github",
      name: "GitHub",
      description: "GitHub OAuth Apps",
      icon: <GitHubIcon />,
      tooltip: [
        "Authenticate developers via GitHub accounts.",
        "Requires a GitHub OAuth App with callback URL.",
      ],
    },
    {
      id: "microsoft",
      name: "Microsoft",
      description: "Azure AD / Microsoft 365",
      icon: <MicrosoftIcon />,
      tooltip: [
        "Enterprise login via Azure AD / Microsoft 365.",
        "Supports both personal and work/school accounts.",
      ],
    },
  ];

  // Handle "Configure Provider" button click
  const handleConfigureProvider = () => {
    if (!selectedProvider) return;
    const existingProvider = relevantProviders.find(
      (p) => p.provider_name === selectedProvider,
    );

    if (existingProvider) {
      setSelectedProviderData(existingProvider);
      setShowExistingConfig(true);
    } else {
      onNavigate(selectedProvider);
    }
  };

  // Loading state
  if (isLoading) {
    return (
      <div className="flex items-center justify-center p-6">
        <div className="text-sm text-muted-foreground">
          Loading providers...
        </div>
      </div>
    );
  }

  // Show existing provider config details
  if (showExistingConfig && selectedProviderData) {
    return (
      <div className="space-y-4">
        <Alert>
          <CheckCircle className="h-4 w-4 text-green-600" />
          <AlertDescription>
            {selectedProviderData.display_name ||
              selectedProviderData.provider_name}{" "}
            provider is already configured.
          </AlertDescription>
        </Alert>

        {/* Show existing config details */}
        <div className="p-4 rounded-lg border border-border bg-muted/30 space-y-2">
          <div className="text-sm">
            <span className="font-medium">Provider:</span>{" "}
            <span className="text-muted-foreground">
              {selectedProviderData.display_name}
            </span>
          </div>
          <div className="text-sm">
            <span className="font-medium">Status:</span>{" "}
            <Badge
              variant={selectedProviderData.is_active ? "default" : "secondary"}
            >
              {selectedProviderData.is_active ? "Active" : "Inactive"}
            </Badge>
          </div>
          {selectedProviderData.provider_type === "oidc" && (
            <>
              <div className="text-sm">
                <span className="font-medium">Client ID:</span>{" "}
                <span className="font-mono text-xs text-muted-foreground">
                  {selectedProviderData.client_id}
                </span>
              </div>
              <div className="text-sm">
                <span className="font-medium">Callback URL:</span>{" "}
                <span className="font-mono text-xs text-muted-foreground break-all">
                  {selectedProviderData.callback_url}
                </span>
              </div>
            </>
          )}
        </div>

        <div className="flex justify-center gap-2">
          <Button
            onClick={() => {
              setShowExistingConfig(false);
              setSelectedProvider(null);
            }}
            variant="outline"
            size="sm"
          >
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back
          </Button>
          <Button
            onClick={() => onComplete()}
            variant="outline"
            className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
          >
            Continue
            <ChevronRight className="ml-2 h-4 w-4" />
          </Button>
        </div>
      </div>
    );
  }

  // SAML2 flow
  if (authMethodType === "saml2") {
    const samlProvider = relevantProviders.find(
      (p) => p.provider_type === "saml",
    );

    if (samlProvider) {
      // Show existing SAML config
      return (
        <div className="space-y-4">
          <Alert>
            <CheckCircle className="h-4 w-4 text-green-600" />
            <AlertDescription>
              SAML 2.0 provider found and configured.
            </AlertDescription>
          </Alert>

          <div className="p-4 rounded-lg border border-border bg-muted/30 space-y-2">
            <div className="text-sm">
              <span className="font-medium">Provider:</span>{" "}
              <span className="text-muted-foreground">
                {samlProvider.display_name}
              </span>
            </div>
            <div className="text-sm">
              <span className="font-medium">Status:</span>{" "}
              <Badge variant={samlProvider.is_active ? "default" : "secondary"}>
                {samlProvider.is_active ? "Active" : "Inactive"}
              </Badge>
            </div>
          </div>

          <div className="flex justify-center">
            <Button
              onClick={() => onComplete()}
              variant="outline"
              className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
            >
              Continue to Client Selection
              <ChevronRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </div>
      );
    } else {
      // No SAML - show "Configure SAML 2.0" button
      return (
        <div className="space-y-4">
          <div className="flex justify-center">
            <Button
              onClick={() => onNavigate()}
              variant="outline"
              className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
            >
              Configure SAML 2.0
              <ChevronRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </div>
      );
    }
  }

  // OIDC provider selection (default view)
  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">
        Select an OAuth 2.0 / OpenID Connect provider to configure:
      </p>
      <div className="grid grid-cols-1 gap-3">
        {oidcProviders.map((provider) => (
          <Tooltip key={provider.id}>
            <TooltipTrigger asChild>
              <div
                className={cn(
                  "flex items-center gap-3 p-3 rounded-lg border cursor-pointer transition-all hover:border-primary/50",
                  selectedProvider === provider.id &&
                    "border-primary bg-primary/5 ring-2 ring-primary ring-offset-2",
                )}
                onClick={() => setSelectedProvider(provider.id)}
              >
                <div className="flex items-center justify-center w-10 h-10 rounded-lg bg-background border border-border">
                  {provider.icon}
                </div>
                <div className="flex-1">
                  <h4 className="font-medium text-sm">{provider.name}</h4>
                  <p className="text-xs text-muted-foreground">
                    {provider.description}
                  </p>
                </div>
                {selectedProvider === provider.id && (
                  <Check className="h-4 w-4 text-primary shrink-0" />
                )}
              </div>
            </TooltipTrigger>
            <TooltipContent
              side="left"
              className="max-w-[220px] text-[10px] leading-relaxed"
            >
              <p className="text-white">{provider.tooltip[0]}</p>
              <p className="mt-1 opacity-80 text-white">{provider.tooltip[1]}</p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>

      <div className="flex justify-center">
        <Button
          onClick={handleConfigureProvider}
          disabled={!selectedProvider}
          variant="outline"
          className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
        >
          Configure{" "}
          {selectedProvider
            ? oidcProviders.find((p) => p.id === selectedProvider)?.name
            : "Provider"}
          <ChevronRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
