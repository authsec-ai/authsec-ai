// Single page authentication method creation form

import React, { useState, useEffect, useCallback, useMemo } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import {
  ArrowLeft,
  CheckCircle,
  Eye,
  EyeOff,
  Loader2,
  Check,
  ChevronRight,
  Settings,
  X,
} from "lucide-react";
import { toast } from "../../lib/toast";
import { cn } from "../../lib/utils";
import { FormField, FormInput, FormBadge, FormCopyField } from "../../theme";

// Types and utilities
import type { AuthMethodConfigData } from "./types";
import { DEFAULT_AUTH_METHOD_CONFIG } from "./utils/defaults";
import { useAddOidcProviderMutation } from "../../app/api/authMethodApi";
import { SessionManager } from "../../utils/sessionManager";

// Provider templates
interface ProviderTemplate {
  id: string;
  name: string;
  icon: React.ComponentType<{ className?: string }>;
  accent: string;
  scopes: string[];
  description: string;
  auth_url: string;
  token_url: string;
  user_info_url: string;
  sort_order: number;
}

type ProviderOptionProps = {
  template: ProviderTemplate;
  selected: boolean;
  onSelect: () => void;
};

const ProviderOption = ({
  template,
  selected,
  onSelect,
}: ProviderOptionProps) => {
  const Icon = template.icon;

  const iconContainerClasses = selected
    ? "bg-primary/15 text-primary"
    : template.accent
      ? cn(template.accent, "text-[color:var(--color-text-secondary)]")
      : "bg-muted text-[color:var(--color-text-secondary)]";

  return (
    <button
      type="button"
      onClick={onSelect}
      className={cn(
        "group relative flex flex-col items-center gap-2 overflow-hidden rounded-lg border px-4 py-3 text-center transition-all duration-300",
        "border-[var(--component-form-section-border)] bg-[color-mix(in_oklab,var(--component-card-background) 94%,transparent)] shadow-sm",
        "hover:border-primary/60 hover:shadow-md focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary/60",
        selected &&
          "border-primary bg-primary/8 ring-2 ring-primary/25 shadow-md",
      )}
    >
      <div
        className={cn(
          "flex h-10 w-10 shrink-0 items-center justify-center rounded-lg transition-colors",
          iconContainerClasses,
        )}
      >
        <Icon className="h-5 w-5" />
      </div>
      <div className="text-sm font-semibold text-[color:var(--color-text-primary)]">
        {template.name}
      </div>
      {selected && (
        <span className="absolute right-2 top-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-primary text-primary-foreground shadow-sm">
          <Check className="h-3 w-3" />
        </span>
      )}
    </button>
  );
};

// Provider template list
const OIDC_TEMPLATES: ProviderTemplate[] = [
  {
    id: "google",
    name: "Google",
    icon: ({ className }) => (
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
    ),
    accent: "bg-gradient-to-br from-[#FDE5EE] via-[#E8EDFF] to-white",
    scopes: ["openid", "profile", "email"],
    description: "Google OAuth 2.0 / OpenID Connect",
    auth_url: "https://accounts.google.com/o/oauth2/v2/auth",
    token_url: "https://oauth2.googleapis.com/token",
    user_info_url: "https://www.googleapis.com/oauth2/v2/userinfo",
    sort_order: 2,
  },
  {
    id: "microsoft",
    name: "Microsoft",
    icon: ({ className }) => (
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
    ),
    accent: "bg-gradient-to-br from-[#E4F2FF] via-[#E9EDFF] to-white",
    scopes: ["openid", "profile", "email"],
    description: "Azure AD / Microsoft 365",
    auth_url: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
    token_url: "https://login.microsoftonline.com/common/oauth2/v2.0/token",
    user_info_url: "https://graph.microsoft.com/v1.0/me",
    sort_order: 3,
  },
  {
    id: "github",
    name: "GitHub",
    icon: ({ className }) => (
      <svg
        className={cn("h-6 w-6", className)}
        fill="#054ddcff"
        viewBox="0 0 20 20"
      >
        <path
          // fill="#054ddcff"
          fillRule="evenodd"
          d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
          clipRule="evenodd"
        />
      </svg>
    ),
    accent: "bg-gradient-to-br from-[#EFE3FF] via-[#F3E8FF] to-white",
    scopes: ["user:email"],
    description: "GitHub OAuth Apps",
    auth_url: "https://github.com/login/oauth/authorize",
    token_url: "https://github.com/login/oauth/access_token",
    user_info_url: "https://api.github.com/user",
    sort_order: 1,
  },
];

// Wizard steps
const WIZARD_STEPS = [
  { id: "configuration", label: "Configuration", icon: Settings },
  { id: "review", label: "Review", icon: CheckCircle },
];

export function CreateAuthMethodPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const [configData, setConfigData] = useState<AuthMethodConfigData>(() => ({
    ...DEFAULT_AUTH_METHOD_CONFIG,
    providerType: "oidc",
  }));
  const [selectedTemplate, setSelectedTemplate] = useState<string>("");
  const [showClientSecret, setShowClientSecret] = useState(false);
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [addOidcProvider, { isLoading: isCreating }] =
    useAddOidcProviderMutation();

  const currentStep = WIZARD_STEPS[currentStepIndex];

  // ✅ make tenant reactive
  const [tenantId, setTenantId] = useState<string | null>(
    SessionManager.getSession()?.tenant_id || null,
  );
  useEffect(() => {
    const session = SessionManager.getSession();
    setTenantId(session?.tenant_id || null);
  }, []);

  // Auto-select provider from URL query parameter
  useEffect(() => {
    const preSelectedProvider = searchParams.get("provider");
    if (preSelectedProvider && !selectedTemplate) {
      const template = OIDC_TEMPLATES.find(
        (t) => t.id === preSelectedProvider.toLowerCase(),
      );
      if (template) {
        console.log(`[Auth] Auto-selecting provider: ${template.name}`);
        handleTemplateSelect(template);
      } else {
        console.warn(`[Auth] Invalid provider in URL: ${preSelectedProvider}`);
        toast.error(`Invalid provider: ${preSelectedProvider}`);
      }
    }
  }, [searchParams, selectedTemplate]);

  const selectedProviderTemplate = useMemo(
    () => OIDC_TEMPLATES.find((template) => template.id === selectedTemplate),
    [selectedTemplate],
  );
  const SelectedProviderIcon = selectedProviderTemplate?.icon;

  // ✅ stable functional update to prevent stale state
  const updateProviderConfig = useCallback((updates: any) => {
    setConfigData((prev) => ({
      ...prev,
      providerConfig: {
        ...prev.providerConfig,
        ...updates,
      },
    }));
  }, []);

  const handleTemplateSelect = (template: ProviderTemplate) => {
    setSelectedTemplate(template.id);
    updateProviderConfig({
      providerName: template.name,
      scopes: template.scopes,
      auth_url: template.auth_url,
      token_url: template.token_url,
      user_info_url: template.user_info_url,
      sort_order: template.sort_order,
    });
  };

  const handleBack = useCallback(() => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1);
    } else {
      navigate("/authentication");
    }
  }, [currentStepIndex, navigate]);

  const handleNext = useCallback(() => {
    if (currentStepIndex === 0) {
      if (!selectedTemplate) {
        toast.error("Please select a provider");
        return;
      }
      if (!configData.providerConfig?.clientId?.trim()) {
        setErrors({ clientId: "Client ID is required" });
        toast.error("Please fill in all required fields");
        return;
      }
    }

    if (currentStepIndex < WIZARD_STEPS.length - 1) {
      setErrors({});
      setCurrentStepIndex(currentStepIndex + 1);
    }
  }, [currentStepIndex, selectedTemplate, configData.providerConfig]);

  const canProceed = () => {
    if (currentStepIndex === 0) {
      // All configuration must be complete
      return Boolean(
        selectedTemplate && configData.providerConfig?.clientId?.trim(),
      );
    }
    if (currentStepIndex === 1) {
      // Review step - same validation
      return Boolean(
        tenantId && selectedTemplate && configData.providerConfig?.clientId,
      );
    }
    return false;
  };

  const handleFinish = async () => {
    const session = SessionManager.getSession();
    const tenant = session?.tenant_id;
    const userEmail = session?.user?.email;

    if (!tenant || !userEmail) {
      toast.error("Missing authentication session. Please sign in again.");
      return;
    }

    const providerConfig = configData.providerConfig;
    if (!providerConfig) {
      toast.error("Provider configuration is missing");
      return;
    }

    try {
      const oidcPayload = {
        tenant_id: tenant,
        org_id: tenant,
        client_id: tenant,
        react_app_url: window.location.origin,
        provider: {
          provider_name: providerConfig.providerName?.toLowerCase() || "",
          display_name: providerConfig.providerName || "",
          client_id: providerConfig.clientId || "",
          client_secret: providerConfig.clientSecret || "",
          auth_url: providerConfig.auth_url || "",
          token_url: providerConfig.token_url || "",
          user_info_url: providerConfig.user_info_url || "",
          scopes: providerConfig.scopes || [],
          is_active: true,
        },
        created_by: userEmail,
      };

      await addOidcProvider(oidcPayload).unwrap();
      toast.success("OIDC provider created successfully!");

      // Navigate back to dashboard with success state for wizard
      navigate("/", {
        state: {
          authProviderCreated: true,
          from: "/authentication/create",
        },
      });
    } catch (error) {
      console.error("Failed to create OIDC provider:", error);
      const ErrorMsg = error?.data?.error || "Failed to create OIDC provider.";
      toast.error(ErrorMsg);
    }
  };

  const getStepSubtitle = () => {
    switch (currentStep.id) {
      case "configuration":
        return "Configure your OAuth 2.0 / OpenID Connect provider";
      case "review":
        return "Review and finalize your authentication method";
      default:
        return "";
    }
  };

  return (
    <div className="flex flex-col h-[90vh] w-full">
      {/* Fixed Header */}
      <div className="flex-shrink-0 border-b py-4 px-8">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <h2 className="text-lg font-semibold">
              Create Authentication Method
            </h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              {getStepSubtitle()}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate("/authentication")}
            className="h-8 w-8 rounded-full bg-red-50 text-red-600 hover:bg-red-100 hover:text-red-700 dark:bg-red-950 dark:text-red-400 dark:hover:bg-red-900 dark:hover:text-red-300"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Scrollable Content Area */}
      <div className="flex-1 overflow-y-auto px-8 py-4 min-h-0">
        <div className="w-full">
          {/* Step 0: Configuration */}
          {currentStepIndex === 0 && (
            <div className="space-y-6">
              {/* Provider Selection Section */}
              <div>
                {selectedProviderTemplate && (
                  <div className="flex items-center gap-3 rounded-md border bg-muted/30 px-4 py-3 mb-3">
                    {SelectedProviderIcon && (
                      <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-background shadow-sm">
                        <SelectedProviderIcon className="h-6 w-6" />
                      </div>
                    )}
                    <div className="flex-1">
                      <div className="font-medium">
                        {selectedProviderTemplate.name}
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {selectedProviderTemplate.description}
                      </div>
                    </div>
                    <FormBadge variant="secondary">Selected</FormBadge>
                  </div>
                )}

                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
                  {[...OIDC_TEMPLATES]
                    .sort((a, b) => a.sort_order - b.sort_order)
                    .map((template) => (
                      <ProviderOption
                        key={template.id}
                        template={template}
                        selected={selectedTemplate === template.id}
                        onSelect={() => handleTemplateSelect(template)}
                      />
                    ))}
                </div>
              </div>
              {/* Callback URL Section */}
              {selectedProviderTemplate && (
                <div>
                  {/* <div className="rounded-lg border bg-muted/50 p-3 mb-3">
                    <div className="flex items-start gap-3">
                      <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-500/10">
                        {SelectedProviderIcon && (
                          <SelectedProviderIcon className="h-3.5 w-3.5 text-blue-600" />
                        )}
                      </div>
                      <div className="flex-1">
                        <h4 className="font-medium text-xs">
                          {selectedProviderTemplate?.name}
                        </h4>
                        <p className="text-[11px] text-muted-foreground">
                          {selectedProviderTemplate?.description}
                        </p>
                      </div>
                    </div>
                  </div> */}

                  <FormField label="Callback URL">
                    <FormCopyField
                      value={`${window.location.origin}/oidc/auth/callback`}
                      onCopy={() =>
                        toast.success("Callback URL copied to clipboard!")
                      }
                      className="font-mono text-sm"
                    />
                  </FormField>
                </div>
              )}

              {/* Client Credentials Section */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
                <div className="space-y-6">
                  {selectedProviderTemplate && (
                    <div>
                      <div className="space-y-3">
                        <FormField
                          label="Client ID"
                          htmlFor="providerClientId"
                          required
                        >
                          <FormInput
                            id="providerClientId"
                            placeholder="Your OAuth client ID"
                            value={configData.providerConfig?.clientId || ""}
                            onChange={(
                              e: React.ChangeEvent<HTMLInputElement>,
                            ) => {
                              updateProviderConfig({
                                clientId: e.target.value,
                              });
                              setErrors({});
                            }}
                            className="h-9 font-mono"
                          />
                          {errors.clientId && (
                            <p className="text-xs text-destructive mt-1">
                              {errors.clientId}
                            </p>
                          )}
                        </FormField>
                      </div>
                    </div>
                  )}
                </div>
                <div className="space-y-6">
                  {selectedProviderTemplate && (
                    <div>
                      <div className="space-y-3">
                        <FormField label="Client Secret" htmlFor="clientSecret">
                          <div className="relative">
                            <FormInput
                              id="clientSecret"
                              type={showClientSecret ? "text" : "password"}
                              placeholder="Your OAuth client secret (optional)"
                              value={
                                configData.providerConfig?.clientSecret || ""
                              }
                              onChange={(
                                e: React.ChangeEvent<HTMLInputElement>,
                              ) =>
                                updateProviderConfig({
                                  clientSecret: e.target.value,
                                })
                              }
                              className="h-9 font-mono pr-12"
                            />
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              className="absolute right-1 top-1/2 -translate-y-1/2 h-8 w-8 p-0"
                              onClick={() =>
                                setShowClientSecret(!showClientSecret)
                              }
                            >
                              {showClientSecret ? (
                                <EyeOff className="h-4 w-4" />
                              ) : (
                                <Eye className="h-4 w-4" />
                              )}
                            </Button>
                          </div>
                        </FormField>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}

          {currentStepIndex === 1 && (
            <div className="space-y-3">
              <div className="rounded-lg border bg-muted/50 p-3 space-y-2.5">
                <div>
                  <h4 className="font-medium text-xs mb-0.5">Tenant</h4>
                  <p className="text-xs text-muted-foreground font-mono">
                    {tenantId || "—"}
                  </p>
                </div>

                <div>
                  <h4 className="font-medium text-xs mb-0.5">Provider</h4>
                  <div className="flex items-center gap-2">
                    {SelectedProviderIcon && (
                      <div className="flex h-6 w-6 items-center justify-center rounded bg-background">
                        <SelectedProviderIcon className="h-4 w-4" />
                      </div>
                    )}
                    <p className="text-xs text-muted-foreground">
                      {selectedProviderTemplate?.name}
                    </p>
                  </div>
                </div>

                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1">
                    OAuth Configuration
                  </h4>
                  <div className="space-y-0.5 text-[11px] text-muted-foreground">
                    <div className="flex justify-between gap-4">
                      <span>Client ID:</span>
                      <span className="font-mono truncate">
                        {configData.providerConfig?.clientId || "—"}
                      </span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span>Callback URL:</span>
                      <span className="font-mono truncate text-right">
                        {window.location.origin}/oidc/auth/callback
                      </span>
                    </div>
                  </div>
                </div>

                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1.5">Scopes</h4>
                  <div className="flex flex-wrap gap-1.5">
                    {(configData.providerConfig?.scopes || []).map((scope) => (
                      <FormBadge key={scope} variant="outline">
                        {scope}
                      </FormBadge>
                    ))}
                  </div>
                </div>

                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1">Endpoints</h4>
                  <div className="space-y-1 text-[11px] text-muted-foreground">
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">Auth URL:</span>
                      <span className="font-mono break-all">
                        {configData.providerConfig?.auth_url}
                      </span>
                    </div>
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">Token URL:</span>
                      <span className="font-mono break-all">
                        {configData.providerConfig?.token_url}
                      </span>
                    </div>
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">UserInfo URL:</span>
                      <span className="font-mono break-all">
                        {configData.providerConfig?.user_info_url}
                      </span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Fixed Footer with Navigation */}
      <div className="flex-shrink-0 border-t bg-background pt-4 pb-4 mt-auto px-8">
        <div className="flex items-center justify-between gap-4">
          {/* Back/Cancel Button */}
          <div className="flex items-center gap-2 min-w-[120px]">
            <Button variant="outline" onClick={handleBack} size="default">
              {currentStepIndex > 0 ? (
                <>
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back
                </>
              ) : (
                "Cancel"
              )}
            </Button>
          </div>

          {/* Progress Stepper */}
          <div className="flex items-center gap-2 flex-1 justify-center">
            {WIZARD_STEPS.map((step, index) => {
              const StepIcon = step.icon;
              const isActive = index === currentStepIndex;
              const isCompleted = index < currentStepIndex;

              return (
                <React.Fragment key={step.id}>
                  <div
                    className={cn(
                      "flex items-center gap-2 rounded-lg px-3 py-2",
                      isActive && "bg-primary/10",
                      isCompleted && "opacity-60",
                    )}
                  >
                    <div
                      className={cn(
                        "flex h-6 w-6 items-center justify-center rounded-full text-xs",
                        isCompleted && "bg-primary text-primary-foreground",
                        isActive && "bg-primary/20 text-primary",
                        !isActive &&
                          !isCompleted &&
                          "bg-muted text-muted-foreground",
                      )}
                    >
                      {isCompleted ? (
                        <Check className="h-3 w-3" />
                      ) : (
                        <StepIcon className="h-3 w-3" />
                      )}
                    </div>
                    <span
                      className={cn(
                        "text-sm font-medium",
                        isActive && "text-foreground",
                        !isActive && "text-muted-foreground",
                      )}
                    >
                      {step.label}
                    </span>
                  </div>
                  {index < WIZARD_STEPS.length - 1 && (
                    <ChevronRight className="h-4 w-4 text-muted-foreground" />
                  )}
                </React.Fragment>
              );
            })}
          </div>

          {/* Next/Finish Button */}
          <div className="flex items-center gap-2 min-w-[120px] justify-end">
            {currentStepIndex < WIZARD_STEPS.length - 1 ? (
              <Button
                onClick={handleNext}
                disabled={!canProceed()}
                size="default"
              >
                Next
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            ) : (
              <Button
                onClick={handleFinish}
                disabled={!canProceed() || isCreating}
                size="default"
              >
                {isCreating ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <CheckCircle className="mr-2 h-4 w-4" />
                    Create Method
                  </>
                )}
              </Button>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
