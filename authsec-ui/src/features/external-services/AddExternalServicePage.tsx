import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Textarea } from "../../components/ui/textarea";
import { Card, CardContent } from "../../components/ui/card";
import { ArrowLeft, Copy, Check, Loader2 } from "lucide-react";
import { toast } from "react-hot-toast";
import {
  useCreateExternalServiceMutation,
  useGetExternalServicesQuery,
} from "@/app/api/externalServiceApi";
import type { RawExternalService } from "@/app/api/externalServiceApi";

interface ServiceFormData {
  name: string;
  type: string;
  url: string;
  description: string;
  tags: string;
  auth_type: "oauth2" | "api_key" | "basic_auth" | "bearer_token" | "none";
  agent_accessible: boolean;
  api_key: string;
  client_id: string;
  client_secret: string;
  webhook_secret: string;
}

export function AddExternalServicePage() {
  const navigate = useNavigate();
  const { serviceId } = useParams<{ serviceId: string }>();
  const [createService] = useCreateExternalServiceMutation();
  const { data: allServices } = useGetExternalServicesQuery();

  const [formData, setFormData] = useState<ServiceFormData>({
    name: "",
    type: "API",
    url: "",
    description: "",
    tags: "",
    auth_type: "api_key",
    agent_accessible: true,
    api_key: "",
    client_id: "",
    client_secret: "",
    webhook_secret: "",
  });
  const [isCreating, setIsCreating] = useState(false);
  const [createdService, setCreatedService] =
    useState<RawExternalService | null>(null);
  const [copiedSteps, setCopiedSteps] = useState<Set<string>>(new Set());

  // SDK View Mode: If serviceId is present, find the service and show SDK section
  const isSDKViewMode = !!serviceId;
  const existingService =
    isSDKViewMode && allServices
      ? allServices.find((s) => s.id === serviceId)
      : null;

  // Set createdService to existingService in SDK view mode
  useEffect(() => {
    if (isSDKViewMode && existingService) {
      setCreatedService(existingService);
      // Also populate formData for SDK code generation
      setFormData({
        name: existingService.name,
        type: existingService.type || "API",
        url: existingService.url,
        description: existingService.description || "",
        tags: existingService.tags?.join(", ") || "",
        auth_type: existingService.auth_type as
          | "oauth2"
          | "api_key"
          | "basic_auth"
          | "bearer_token"
          | "none",
        agent_accessible: existingService.agent_accessible,
        api_key: "",
        client_id: "",
        client_secret: "",
        webhook_secret: "",
      });
    }
  }, [isSDKViewMode, existingService]);

  const handleCopy = (text: string, stepId: string) => {
    navigator.clipboard.writeText(text);
    setCopiedSteps((prev) => new Set([...prev, stepId]));
    setTimeout(() => {
      setCopiedSteps((prev) => {
        const newSet = new Set(prev);
        newSet.delete(stepId);
        return newSet;
      });
    }, 2000);
  };

  const getInstallCommand = () => {
    return "pip install git+https://github.com/authsec-ai/sdk-authsec.git";
  };

  const getSDKUsageCode = () => {
    return `from AuthSec_SDK import ServiceAccessSDK, ServiceAccessError

# Initialize SDK in your function at the beginning
sdk = ServiceAccessSDK(session, timeout=10)

# Fetch ${formData.name || "service-name"} credentials & token
try:
    credentials = await sdk.get_service_credentials("${
      formData.name || "service-name"
    }")
    token = credentials.credentials.get("access_token")
except ServiceAccessError as e:
    return [{"type": "text", "text": f"Failed to fetch credentials: {str(e)}"}]

if not token:
    return [{"type": "text", "text": "${
      formData.name || "service-name"
    } token not available"}]

# Close SDK at the end of your function
finally:
    if sdk:
        await sdk.close()`;
  };

  const getExampleCode = () => {
    return `from AuthSec_SDK import ServiceAccessSDK, ServiceAccessError
import aiohttp, json

@protected_by_AuthSec("${
      formData.name.toLowerCase().replace(/\\s+/g, "_") || "test_service"
    }")
async def ${
      formData.name.toLowerCase().replace(/\\s+/g, "_") || "test_service"
    }(arguments: dict, session) -> list:
    sdk = ServiceAccessSDK(session, timeout=10)
    try:
        credentials = await sdk.get_service_credentials("${
          formData.name || "service-name"
        }")
        token = credentials.credentials["access_token"]
        
        async with aiohttp.ClientSession() as http_session:
            headers = {"Authorization": f"Bearer {token}"}
            async with http_session.get("${
              formData.url || "https://api.example.com"
            }/endpoint", headers=headers) as response:
                data = await response.json()
                return [{"type": "text", "text": json.dumps(data, indent=2)}]
    finally:
        await sdk.close()`;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsCreating(true);

    try {
      // Build secret_data based on auth_type
      const secret_data: any = {};
      if (formData.auth_type === "api_key" && formData.api_key) {
        secret_data.api_key = formData.api_key;
      }
      if (formData.auth_type === "oauth2") {
        if (formData.client_id) secret_data.client_id = formData.client_id;
        if (formData.client_secret)
          secret_data.client_secret = formData.client_secret;
      }
      if (formData.webhook_secret) {
        secret_data.webhook_secret = formData.webhook_secret;
      }

      const payload = {
        name: formData.name,
        type: formData.type,
        url: formData.url,
        description: formData.description,
        tags: formData.tags
          .split(",")
          .map((t) => t.trim())
          .filter(Boolean),
        resource_id: 1,
        auth_type: formData.auth_type,
        agent_accessible: formData.agent_accessible,
        secret_data,
      };

      const created = await createService(payload).unwrap();
      // Cast to RawExternalService since the API returns the raw data
      setCreatedService(created as any as RawExternalService);
      toast.success(`${formData.name} service created successfully!`);
    } catch (error: unknown) {
      console.error("Failed to create service:", error);
      const err = error as { data?: { message?: string }; error?: string };
      const msg =
        err?.data?.message || err?.error || "Failed to create service";
      toast.error(msg);
    } finally {
      setIsCreating(false);
    }
  };

  // CodeBlock component with slim headers
  const CodeBlock = ({
    code,
    label,
    onCopy,
    copied,
  }: {
    code: string;
    label?: string;
    onCopy?: () => void;
    copied?: boolean;
  }) => {
    return (
      <div className="border border-border/60 rounded-lg overflow-hidden bg-background/50 backdrop-blur-sm">
        <div className="flex items-center justify-between border-b border-border/50 bg-muted/40 px-2.5 py-0.5">
          <span className="text-[11px] font-semibold text-foreground/75">
            {label}
          </span>
          <div className="flex items-center">
            {onCopy && (
              <Button
                size="sm"
                variant="ghost"
                className="h-5 w-5 p-0 hover:bg-background/60 text-foreground/60 hover:text-foreground"
                onClick={onCopy}
              >
                {copied ? (
                  <Check className="h-2.5 w-2.5 text-green-600 dark:text-green-500" />
                ) : (
                  <Copy className="h-2.5 w-2.5" />
                )}
              </Button>
            )}
          </div>
        </div>
        <div className="p-3 overflow-x-auto-hidden bg-background/30">
          <pre className="text-sm font-mono text-foreground/90 whitespace-pre-wrap leading-relaxed">
            {code}
          </pre>
        </div>
      </div>
    );
  };

  if (createdService) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-background via-background to-muted/20">
        <div className="container mx-auto px-4 py-8 max-w-8xl">
          {/* Page Header */}
          <div className="flex items-center gap-3 mb-8 border-b pb-4">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => navigate("/external-services")}
              className="h-8 px-2"
            >
              <ArrowLeft className="h-4 w-4" />
            </Button>
            <div>
              <h1 className="text-xl font-semibold">
                Service Created - SDK Integration
              </h1>
              <p className="text-sm text-foreground">
                Now integrate the AuthSec SDK to access {createdService.name}
              </p>
            </div>
          </div>

          {/* Timeline Container */}
          <div className="relative">
            {/* Vertical Timeline Line */}
            <div className="absolute left-6 top-0 bottom-0 w-0.5 bg-gradient-to-b from-primary/50 via-primary/30 to-transparent" />

            {/* Step 1: Install SDK */}
            <div className="relative flex gap-6 mb-6">
              <div className="relative z-10 flex-shrink-0">
                <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center transition-all">
                  <span className="text-sm font-semibold text-foreground">
                    1
                  </span>
                </div>
              </div>
              <div className="flex-1 pt-2">
                <p className="text-sm font-medium text-foreground/70 mb-3">
                  Install AuthSec SDK
                </p>
                <CodeBlock
                  label="Terminal"
                  code={`$ ${getInstallCommand()}`}
                  onCopy={() => handleCopy(getInstallCommand(), "install")}
                  copied={copiedSteps.has("install")}
                />
              </div>
            </div>

            {/* Step 2: SDK Usage */}
            <div className="relative flex gap-6 mb-6">
              <div className="relative z-10 flex-shrink-0">
                <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center transition-all">
                  <span className="text-sm font-semibold text-foreground">
                    2
                  </span>
                </div>
              </div>
              <div className="flex-1 pt-2">
                <p className="text-sm font-medium text-foreground/70 mb-3">
                  Access {createdService.name} Credentials
                </p>
                <CodeBlock
                  label="SDK Usage Template"
                  code={getSDKUsageCode()}
                  onCopy={() => handleCopy(getSDKUsageCode(), "sdk-usage")}
                  copied={copiedSteps.has("sdk-usage")}
                />
              </div>
            </div>

            {/* Step 3: Complete Example */}
            <div className="relative flex gap-6 mb-6">
              <div className="relative z-10 flex-shrink-0">
                <div className="w-12 h-12 rounded-full border-2 bg-background border-primary/30 flex items-center justify-center transition-all">
                  <span className="text-sm font-semibold text-foreground">
                    3
                  </span>
                </div>
              </div>
              <div className="flex-1 pt-2">
                <p className="text-sm font-medium text-foreground/70 mb-3">
                  Complete Example
                </p>
                <CodeBlock
                  label={`example_${formData.name
                    .toLowerCase()
                    .replace(/\s+/g, "_")}.py`}
                  code={getExampleCode()}
                  onCopy={() => handleCopy(getExampleCode(), "example")}
                  copied={copiedSteps.has("example")}
                />
              </div>
            </div>
          </div>

          {/* Success Message */}
          <div className="bg-green-50 dark:bg-green-950/20 border border-green-200 dark:border-green-800 rounded-lg p-4 mb-6">
            <h4 className="font-semibold text-green-800 dark:text-green-200 mb-2">
              🎉 You're all set!
            </h4>
            <p className="text-green-700 dark:text-green-300 text-sm">
              Your {createdService.name} service is now configured and ready to
              use. Users will be securely authenticated before accessing the
              service through your functions.
            </p>
          </div>

          {/* Navigation buttons */}
          <div className="flex gap-4 justify-center">
            <Button
              onClick={() => navigate("/external-services")}
              variant="outline"
            >
              View All Services
            </Button>
            <Button
              onClick={() => {
                setCreatedService(null);
                setFormData({
                  name: "",
                  type: "API",
                  url: "",
                  description: "",
                  tags: "",
                  auth_type: "api_key",
                  agent_accessible: true,
                  api_key: "",
                  client_id: "",
                  client_secret: "",
                  webhook_secret: "",
                });
              }}
            >
              Add Another Service
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen">
      <div className="container mx-auto max-w-8xl space-y-8 px-8 py-8">
        {/* Page Header */}
        <header className="bg-card border border-border rounded-sm p-6 shadow-sm">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => navigate("/external-services")}
                className="h-8 px-2"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <div>
                <h1 className="text-2xl font-semibold">Add External Service</h1>
                <p className="text-sm text-foreground mt-1">
                  Create a service, then integrate the SDK
                </p>
              </div>
            </div>
            <Button
              type="submit"
              form="create-service-form"
              disabled={isCreating}
              className="min-w-[140px]"
            >
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Service"
              )}
            </Button>
          </div>
        </header>

        {/* Service Configuration Form */}
        <div className="flex justify-center">
          <Card className="!max-w-none w-[95vw]">
            <CardContent className="p-6">
              <h2 className="text-lg font-semibold mb-6">
                Service Configuration
              </h2>

              <form
                id="create-service-form"
                onSubmit={handleSubmit}
                className="space-y-4"
              >
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="name">Service Name *</Label>
                    <Input
                      id="name"
                      placeholder="e.g., Stripe Payment API"
                      value={formData.name}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          name: e.target.value,
                        }))
                      }
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="url">API URL *</Label>
                    <Input
                      id="url"
                      placeholder="e.g., https://api.stripe.com/v1"
                      value={formData.url}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          url: e.target.value,
                        }))
                      }
                      required
                    />
                  </div>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="description">Description</Label>
                  <Textarea
                    id="description"
                    placeholder="e.g., Payment processing service"
                    value={formData.description}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        description: e.target.value,
                      }))
                    }
                  />
                </div>

                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="tags">Tags (comma-separated)</Label>
                    <Input
                      id="tags"
                      placeholder="e.g., payment, billing, stripe"
                      value={formData.tags}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          tags: e.target.value,
                        }))
                      }
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="auth_type">Auth Type</Label>
                    <select
                      id="auth_type"
                      className="w-full px-3 py-2 border rounded-md dark:bg-neutral-900 dark:border-neutral-600"
                      value={formData.auth_type}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          auth_type: e.target.value as any,
                        }))
                      }
                    >
                      <option value="api_key">API Key</option>
                      <option value="oauth2">OAuth 2.0</option>
                      <option value="bearer_token">Bearer Token</option>
                      <option value="basic_auth">Basic Auth</option>
                      <option value="none">None</option>
                    </select>
                  </div>
                </div>

                {/* Conditional secret fields based on auth type */}
                {formData.auth_type === "api_key" && (
                  <div className="space-y-2">
                    <Label htmlFor="api_key">API Key</Label>
                    <Input
                      id="api_key"
                      placeholder="sk_test_your_key_here"
                      value={formData.api_key}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          api_key: e.target.value,
                        }))
                      }
                    />
                  </div>
                )}

                {formData.auth_type === "oauth2" && (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label htmlFor="client_id">Client ID</Label>
                      <Input
                        id="client_id"
                        placeholder="your_client_id"
                        value={formData.client_id}
                        onChange={(e) =>
                          setFormData((prev) => ({
                            ...prev,
                            client_id: e.target.value,
                          }))
                        }
                      />
                    </div>
                    <div className="space-y-2">
                      <Label htmlFor="client_secret">Client Secret</Label>
                      <Input
                        id="client_secret"
                        type="password"
                        placeholder="your_client_secret"
                        value={formData.client_secret}
                        onChange={(e) =>
                          setFormData((prev) => ({
                            ...prev,
                            client_secret: e.target.value,
                          }))
                        }
                      />
                    </div>
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="webhook_secret">
                    Webhook Secret (Optional)
                  </Label>
                  <Input
                    id="webhook_secret"
                    placeholder="whsec_your_webhook_secret"
                    value={formData.webhook_secret}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        webhook_secret: e.target.value,
                      }))
                    }
                  />
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
