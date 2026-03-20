import { useState } from "react";
import { Button } from "../../../components/ui/button";
import { Card } from "../../../components/ui/card";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { Copy } from "lucide-react";
import { toast } from "react-hot-toast";
import { cn } from "../../../lib/utils";
import type { ExternalServiceFormData, ProviderOption } from "../types";

// Provider templates
const providerTemplates: ProviderOption[] = [
  {
    id: "google_drive",
    name: "Google Drive",
    icon: "https://upload.wikimedia.org/wikipedia/commons/1/12/Google_Drive_icon_%282020%29.svg",
    scopes: [
      {
        id: "drive.readonly",
        name: "drive.readonly",
        description: "View files in your Google Drive",
      },
      {
        id: "drive.file",
        name: "drive.file",
        description: "View and manage files created by this app",
      },
      {
        id: "drive.appdata",
        name: "drive.appdata",
        description: "View and manage app data",
        isSensitive: true,
      },
      {
        id: "drive.metadata.readonly",
        name: "drive.metadata.readonly",
        description: "View metadata for files",
      },
      {
        id: "drive.activity",
        name: "drive.activity",
        description: "View activity history",
        isDeprecated: true,
      },
    ],
    resources: [
      { resource: "drive/files/*", scopes: ["read", "write"] },
      { resource: "drive/meta/*", scopes: ["read"] },
    ],
    authEndpoints: {
      authorizationUrl: "https://accounts.google.com/o/oauth2/auth",
      tokenUrl: "https://oauth2.googleapis.com/token",
      userinfoUrl: "https://www.googleapis.com/oauth2/v3/userinfo",
    },
  },
  {
    id: "salesforce",
    name: "Salesforce",
    icon: "https://upload.wikimedia.org/wikipedia/commons/f/f9/Salesforce.com_logo.svg",
    scopes: [
      { id: "api", name: "api", description: "Access the Salesforce API" },
      {
        id: "refresh_token",
        name: "refresh_token",
        description: "Get a refresh token for offline access",
      },
      { id: "chatter_api", name: "chatter_api", description: "Access Chatter API" },
    ],
    resources: [
      { resource: "salesforce/accounts/*", scopes: ["read", "write"] },
      { resource: "salesforce/contacts/*", scopes: ["read"] },
    ],
    authEndpoints: {
      authorizationUrl: "https://login.salesforce.com/services/oauth2/authorize",
      tokenUrl: "https://login.salesforce.com/services/oauth2/token",
      userinfoUrl: "https://login.salesforce.com/services/oauth2/userinfo",
    },
  },
  {
    id: "microsoft_graph",
    name: "Microsoft Graph",
    icon: "https://upload.wikimedia.org/wikipedia/commons/4/44/Microsoft_logo.svg",
    scopes: [
      { id: "Files.Read", name: "Files.Read", description: "Read files" },
      { id: "Files.ReadWrite", name: "Files.ReadWrite", description: "Read and write files" },
      { id: "User.Read", name: "User.Read", description: "Read user profile" },
      { id: "Mail.Read", name: "Mail.Read", description: "Read mail", isSensitive: true },
    ],
    resources: [
      { resource: "msgraph/files/*", scopes: ["read", "write"] },
      { resource: "msgraph/users/*", scopes: ["read"] },
    ],
    authEndpoints: {
      authorizationUrl: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
      tokenUrl: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
      userinfoUrl: "https://graph.microsoft.com/v1.0/me",
    },
  },
  {
    id: "custom_oauth2",
    name: "Custom OAuth 2.0",
    icon: "https://cdn-icons-png.flaticon.com/512/5968/5968756.png",
    scopes: [{ id: "custom", name: "custom", description: "Custom scope" }],
    resources: [{ resource: "custom/*", scopes: ["read", "write"] }],
    authEndpoints: {
      authorizationUrl: "",
      tokenUrl: "",
      userinfoUrl: "",
    },
  },
];

interface ProviderSelectorProps {
  formData: ExternalServiceFormData;
  onUpdate: (updates: Partial<ExternalServiceFormData>) => void;
  errors?: Record<string, string>;
}

export function ProviderSelector({ formData, onUpdate, errors = {} }: ProviderSelectorProps) {
  const [copied, setCopied] = useState(false);

  const handleProviderSelect = (provider: ProviderOption) => {
    onUpdate({
      provider: provider.id,
      providerName: provider.name,
      scopes: provider.scopes.filter((s) => !s.isDeprecated && !s.isSensitive).map((s) => s.id),
      externalResources: provider.resources,
      advancedOptions: {
        ...formData.advancedOptions,
        customAuthEndpoints: provider.authEndpoints,
      },
    });
  };

  const handleCopyRedirectUri = () => {
    if (formData.redirectUri) {
      navigator.clipboard.writeText(formData.redirectUri);
      setCopied(true);
      toast.success("Redirect URI copied to clipboard");

      setTimeout(() => {
        setCopied(false);
      }, 2000);
    }
  };

  return (
    <div className="space-y-6">
      {/* Provider Templates */}
      <div className="space-y-3">
        <Label>Provider Template</Label>
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
          {providerTemplates.map((provider) => (
            <Card
              key={provider.id}
              className={cn(
                "flex flex-col items-center justify-center p-4 cursor-pointer hover:border-primary/50 transition-colors",
                formData.provider === provider.id && "border-primary bg-primary/5"
              )}
              onClick={() => handleProviderSelect(provider)}
            >
              <div className="h-12 w-12 mb-2">
                <img
                  src={provider.icon}
                  alt={provider.name}
                  className="h-full w-full object-contain"
                />
              </div>
              <div className="text-center">
                <div className="font-medium">{provider.name}</div>
              </div>
            </Card>
          ))}
        </div>
        {errors.provider && <p className="text-sm text-destructive">{errors.provider}</p>}
      </div>

      {/* Service Name */}
      <div className="space-y-3">
        <Label htmlFor="serviceName">Service Name</Label>
        <Input
          id="serviceName"
          value={formData.serviceName}
          onChange={(e) => onUpdate({ serviceName: e.target.value })}
          placeholder="My Google Drive"
          className={errors.serviceName ? "border-destructive" : ""}
        />
        {errors.serviceName ? (
          <p className="text-sm text-destructive">{errors.serviceName}</p>
        ) : (
          <p className="text-sm text-foreground">
            A unique name for this service. Will be used to generate the service ID.
          </p>
        )}
      </div>

      {/* Service ID - auto-generated but shown */}
      <div className="space-y-3">
        <Label htmlFor="serviceId">Service ID</Label>
        <Input id="serviceId" value={formData.serviceId} readOnly className="bg-muted" />
        <p className="text-sm text-foreground">
          Auto-generated from service name. Used in API calls and redirect URIs.
        </p>
      </div>

      {/* Client ID */}
      <div className="space-y-3">
        <Label htmlFor="clientId">Client ID</Label>
        <Input
          id="clientId"
          value={formData.clientId}
          onChange={(e) => onUpdate({ clientId: e.target.value })}
          placeholder="OAuth Client ID from provider console"
          className={errors.clientId ? "border-destructive" : ""}
        />
        {errors.clientId && <p className="text-sm text-destructive">{errors.clientId}</p>}
      </div>

      {/* Client Secret */}
      <div className="space-y-3">
        <Label htmlFor="clientSecret">Client Secret</Label>
        <Input
          id="clientSecret"
          type="password"
          value={formData.clientSecret}
          onChange={(e) => onUpdate({ clientSecret: e.target.value })}
          placeholder="OAuth Client Secret from provider console"
          className={errors.clientSecret ? "border-destructive" : ""}
        />
        {errors.clientSecret && <p className="text-sm text-destructive">{errors.clientSecret}</p>}
      </div>

      {/* Redirect URI - read only */}
      <div className="space-y-3">
        <Label htmlFor="redirectUri">Redirect URI</Label>
        <div className="flex">
          <Input
            id="redirectUri"
            value={formData.redirectUri}
            readOnly
            className="bg-muted rounded-r-none flex-1"
          />
          <Button
            type="button"
            variant="secondary"
            className="rounded-l-none"
            onClick={handleCopyRedirectUri}
          >
            <Copy className="h-4 w-4 mr-2" />
            {copied ? "Copied!" : "Copy"}
          </Button>
        </div>
        <p className="text-sm text-foreground">
          Use this redirect URI in your OAuth application settings.
        </p>
      </div>
    </div>
  );
}
