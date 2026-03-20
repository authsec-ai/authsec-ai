import React, { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import {
  ArrowLeft,
  CheckCircle,
  Loader2,
  Check,
  ChevronRight,
  Cloud,
  Settings,
  PlayCircle,
  X,
} from "lucide-react";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import { EntraConfigForm, type EntraConfigFormData } from "./EntraConfigForm";
import { useCreateSyncConfigMutation, useUpdateSyncConfigMutation, type SyncConfig } from "@/app/api/syncConfigsApi";
import { useSyncEntraIDMutation } from "@/app/api/enduser/invitesApi";

const WIZARD_STEPS = [
  {
    id: "configuration",
    label: "Configuration",
    icon: Settings,
  },
  {
    id: "test",
    label: "Test",
    icon: PlayCircle,
  },
  {
    id: "complete",
    label: "Complete",
    icon: CheckCircle,
  },
];

interface EntraSyncInlineFormProps {
  onClose: () => void;
  onSuccess: () => void;
  editConfig?: SyncConfig | null;
}

export function EntraSyncInlineForm({ onClose, onSuccess, editConfig }: EntraSyncInlineFormProps) {
  const isEditing = !!editConfig;

  const createDefaultConfig = (): EntraConfigFormData => ({
    config_name: "",
    description: "",
    tenant_id: "",
    client_id: "",
    client_secret: "",
    skip_verify: true,
  });

  const hydrateConfig = (source: SyncConfig): EntraConfigFormData => {
    const defaults = createDefaultConfig();
    const entra = source.entra_config || ({} as any);
    const fallback = source as any;

    return {
      ...defaults,
      config_name: source.config_name || defaults.config_name,
      description: source.description || '',
      tenant_id: entra.tenant_id || fallback.tenant_id || fallback.entra_tenant_id || defaults.tenant_id,
      client_id: entra.client_id || fallback.entra_client_id || fallback.client_id || defaults.client_id,
      client_secret: entra.client_secret || fallback.client_secret || defaults.client_secret,
      skip_verify: entra.skip_verify ?? fallback.skip_verify ?? defaults.skip_verify,
    };
  };

  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [entraConfig, setEntraConfig] = useState<EntraConfigFormData>(() =>
    editConfig ? hydrateConfig(editConfig) : createDefaultConfig()
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [testingConnection, setTestingConnection] = useState(false);
  const [connectionTested, setConnectionTested] = useState(false);

  const [createConfig, { isLoading: isCreating }] = useCreateSyncConfigMutation();
  const [updateConfig, { isLoading: isUpdating }] = useUpdateSyncConfigMutation();
  const [syncEntra] = useSyncEntraIDMutation();

  // Update config when editConfig prop changes
  React.useEffect(() => {
    if (editConfig) {
      setEntraConfig(hydrateConfig(editConfig));
    } else {
      setEntraConfig(createDefaultConfig());
    }
    setCurrentStepIndex(0);
    setErrors({});
    setConnectionTested(false);
  }, [editConfig]);

  const currentStep = WIZARD_STEPS[currentStepIndex];

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!entraConfig.config_name.trim()) newErrors.config_name = "Configuration name is required";
    if (!entraConfig.tenant_id.trim()) newErrors.tenant_id = "Tenant ID is required";
    if (!entraConfig.client_id.trim()) newErrors.client_id = "Client ID is required";
    if (!entraConfig.client_secret.trim()) newErrors.client_secret = "Client Secret is required";
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleBack = useCallback(() => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1);
    } else {
      onClose();
    }
  }, [currentStepIndex, onClose]);

  const handleNext = useCallback(() => {
    if (currentStepIndex === 0) {
      if (!validateForm()) {
        toast.error("Please fill in all required fields");
        return;
      }
    }

    if (currentStepIndex < WIZARD_STEPS.length - 1) {
      setCurrentStepIndex(currentStepIndex + 1);
    }
  }, [currentStepIndex]);

  const handleTestConnection = async () => {
    if (!validateForm()) {
      toast.error("Please fill in all required fields");
      return;
    }

    setTestingConnection(true);
    setConnectionTested(false);

    try {
      await syncEntra({
        provider: "entra",
        config: {
          tenant_id: entraConfig.tenant_id,
          client_id: entraConfig.client_id,
          client_secret: entraConfig.client_secret,
          skip_verify: entraConfig.skip_verify,
        },
        dry_run: true,
      }).unwrap();

      setConnectionTested(true);
      toast.success("Connection verified successfully");
    } catch (error: any) {
      toast.error(
        `Connection test failed: ${error.data?.message || error.message || "Unknown error"}`
      );
    } finally {
      setTestingConnection(false);
    }
  };

  const handleFinish = async () => {
    if (!connectionTested) {
      toast.error("Please test the connection before finishing");
      return;
    }

    try {
      const configData = {
        sync_type: "entra_id" as const,
        config_name: entraConfig.config_name,
        description: entraConfig.description,
        entra_config: {
          tenant_id: entraConfig.tenant_id,
          client_id: entraConfig.client_id,
          client_secret: entraConfig.client_secret,
          skip_verify: entraConfig.skip_verify,
        },
        is_active: true,
      };

      if (isEditing && editConfig?.id) {
        await updateConfig({
          id: editConfig.id,
          ...configData,
        }).unwrap();
        toast.success("Entra ID sync updated successfully");
      } else {
        await createConfig(configData).unwrap();
        toast.success("Entra ID sync configured successfully");
      }

      onSuccess();
    } catch (error: any) {
      toast.error(`Failed to ${isEditing ? 'update' : 'create'} configuration: ${error.data?.message || error.message}`);
    }
  };

  const canProceed = () => {
    if (currentStepIndex === 0) {
      return (
        entraConfig.config_name.trim() &&
        entraConfig.tenant_id.trim() &&
        entraConfig.client_id.trim() &&
        entraConfig.client_secret.trim()
      );
    }
    if (currentStepIndex === 1) {
      return connectionTested;
    }
    return true;
  };

  return (
    <div className="flex flex-col h-[90vh] w-full">
      {/* Header */}
      <div className="flex-shrink-0 border-b py-4 px-8">
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <h2 className="text-lg font-semibold">Configure Entra ID Sync</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              {currentStep.id === "configuration" &&
                "Configure your Microsoft Entra ID connection settings"}
              {currentStep.id === "test" && "Test and verify the Entra ID connection"}
              {currentStep.id === "complete" && "Review and finalize setup"}
            </p>
          </div>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClose}
            className="h-8 w-8 rounded-full bg-red-50 text-red-600 hover:bg-red-100 hover:text-red-700 dark:bg-red-950 dark:text-red-400 dark:hover:bg-red-900 dark:hover:text-red-300"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Content - scrollable area */}
      <div className="flex-1 overflow-y-auto px-8 py-4 min-h-0">
        <div className="w-full max-w-6xl">
          {currentStepIndex === 0 && (
            <div className="space-y-3">
              <div className="mb-4">
                <h3 className="text-base font-semibold">Entra ID Configuration</h3>
                <p className="text-xs text-muted-foreground mt-0.5">Configure your Microsoft Entra ID (Azure AD) connection</p>
              </div>
              <EntraConfigForm config={entraConfig} onChange={setEntraConfig} errors={errors} />
            </div>
          )}

          {currentStepIndex === 1 && (
            <div className="space-y-3 max-w-2xl">
              <div className="mb-3">
                <h3 className="text-base font-semibold">Test Connection</h3>
                <p className="text-xs text-muted-foreground mt-0.5">Verify your Entra ID connection</p>
              </div>

              <div className="rounded-lg border bg-muted/50 p-3">
                <div className="flex items-start gap-3">
                  <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-blue-500/10">
                    <Cloud className="h-3.5 w-3.5 text-blue-600" />
                  </div>
                  <div className="flex-1 space-y-0.5">
                    <h4 className="font-medium text-xs">Connection Details</h4>
                    <div className="space-y-0.5 text-[11px] text-muted-foreground">
                      <div className="flex justify-between gap-4">
                        <span>Tenant ID:</span>
                        <span className="font-mono truncate">{entraConfig.tenant_id}</span>
                      </div>
                      <div className="flex justify-between gap-4">
                        <span>Client ID:</span>
                        <span className="font-mono truncate">{entraConfig.client_id}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <Button
                onClick={handleTestConnection}
                disabled={testingConnection}
                className="w-full"
                size="default"
              >
                {testingConnection ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Testing...
                  </>
                ) : connectionTested ? (
                  <>
                    <Check className="mr-2 h-4 w-4" />
                    Verified
                  </>
                ) : (
                  <>
                    <PlayCircle className="mr-2 h-4 w-4" />
                    Test Connection
                  </>
                )}
              </Button>

              {connectionTested && (
                <div className="rounded-lg border border-green-200 bg-green-50 p-2.5 dark:border-green-900/50 dark:bg-green-950/20">
                  <div className="flex items-center gap-2">
                    <CheckCircle className="h-3.5 w-3.5 text-green-600 dark:text-green-500" />
                    <p className="text-xs font-medium text-green-900 dark:text-green-100">
                      Connection verified successfully!
                    </p>
                  </div>
                </div>
              )}
            </div>
          )}

          {currentStepIndex === 2 && (
            <div className="space-y-3 max-w-2xl">
              <div className="mb-3">
                <h3 className="text-base font-semibold">Review & Complete</h3>
                <p className="text-xs text-muted-foreground mt-0.5">Review configuration and finish setup</p>
              </div>

              <div className="rounded-lg border bg-muted/50 p-3 space-y-2.5">
                <div>
                  <h4 className="font-medium text-xs mb-0.5">Configuration Name</h4>
                  <p className="text-xs text-muted-foreground">{entraConfig.config_name}</p>
                </div>
                {entraConfig.description && (
                  <div>
                    <h4 className="font-medium text-xs mb-0.5">Description</h4>
                    <p className="text-xs text-muted-foreground">{entraConfig.description}</p>
                  </div>
                )}
                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1">Connection Details</h4>
                  <div className="space-y-0.5 text-[11px] text-muted-foreground">
                    <div className="flex justify-between gap-4">
                      <span>Tenant ID:</span>
                      <span className="font-mono truncate">{entraConfig.tenant_id}</span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span>Client ID:</span>
                      <span className="font-mono truncate">{entraConfig.client_id}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Footer Actions - fixed at bottom, never scrolls */}
      <div className="flex-shrink-0 border-t bg-background pt-4 pb-4 mt-auto px-8">
        <div className="flex items-center justify-between gap-4">
          {/* Back/Cancel Button on Left */}
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

          {/* Progress Steps in Center */}
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
                      isCompleted && "opacity-60"
                    )}
                  >
                    <div
                      className={cn(
                        "flex h-6 w-6 items-center justify-center rounded-full text-xs",
                        isCompleted && "bg-primary text-primary-foreground",
                        isActive && "bg-primary/20 text-primary",
                        !isActive && !isCompleted && "bg-muted text-muted-foreground"
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
                        !isActive && "text-muted-foreground"
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

          {/* Next/Finish Button on Right */}
          <div className="flex items-center gap-2 min-w-[120px] justify-end">
            {currentStepIndex < WIZARD_STEPS.length - 1 ? (
              <Button onClick={handleNext} disabled={!canProceed()} size="default">
                Next
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            ) : (
              <Button
                onClick={handleFinish}
                disabled={!connectionTested || isCreating}
                size="default"
              >
                {isCreating ? (
                  <>
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  <>
                    <Check className="mr-2 h-4 w-4" />
                    Finish
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
