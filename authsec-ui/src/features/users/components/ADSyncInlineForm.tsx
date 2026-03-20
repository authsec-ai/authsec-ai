import React, { useState, useCallback } from "react";
import { Button } from "@/components/ui/button";
import { ArrowLeft, CheckCircle, Loader2, Check, ChevronRight, Server, Settings, PlayCircle, X } from "lucide-react";
import { toast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  FormRoot,
  FormBody,
  FormSection,
  FormSectionHeader,
  FormDivider,
} from "@/theme";
import { ADConfigForm, type ADConfigFormData } from "./ADConfigForm";
import { useCreateSyncConfigMutation, useUpdateSyncConfigMutation, type SyncConfig } from "@/app/api/syncConfigsApi";
import { useSyncActiveDirectoryMutation } from "@/app/api/enduser/invitesApi";

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

interface ADSyncInlineFormProps {
  onClose: () => void;
  onSuccess: () => void;
  editConfig?: SyncConfig | null;
}

export function ADSyncInlineForm({ onClose, onSuccess, editConfig }: ADSyncInlineFormProps) {
  const isEditing = !!editConfig;

  const createDefaultConfig = (): ADConfigFormData => ({
    config_name: '',
    description: '',
    server: '',
    username: '',
    password: '',
    base_dn: 'CN=Users,DC=company,DC=com',
    filter: '(&(objectClass=user)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))',
    use_ssl: false,
    skip_verify: true,
  });

  const hydrateConfig = (source: SyncConfig): ADConfigFormData => {
    const defaults = createDefaultConfig();
    const ad = source.ad_config || ({} as any);
    const fallback = source as any;

    return {
      ...defaults,
      config_name: source.config_name || defaults.config_name,
      description: source.description || '',
      server: ad.server || fallback.server || fallback.ad_server || defaults.server,
      username: ad.username || fallback.username || fallback.ad_username || defaults.username,
      password: ad.password || fallback.password || defaults.password,
      base_dn: ad.base_dn || fallback.base_dn || fallback.ad_base_dn || defaults.base_dn,
      filter: ad.filter || fallback.filter || defaults.filter,
      use_ssl: ad.use_ssl ?? fallback.use_ssl ?? defaults.use_ssl,
      skip_verify: ad.skip_verify ?? fallback.skip_verify ?? defaults.skip_verify,
    };
  };

  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [adConfig, setAdConfig] = useState<ADConfigFormData>(() =>
    editConfig ? hydrateConfig(editConfig) : createDefaultConfig()
  );
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [testingConnection, setTestingConnection] = useState(false);
  const [connectionTested, setConnectionTested] = useState(false);

  const [createConfig, { isLoading: isCreating }] = useCreateSyncConfigMutation();
  const [updateConfig, { isLoading: isUpdating }] = useUpdateSyncConfigMutation();
  const [syncAD] = useSyncActiveDirectoryMutation();

  // Update config when editConfig prop changes
  React.useEffect(() => {
    if (editConfig) {
      setAdConfig(hydrateConfig(editConfig));
    } else {
      setAdConfig(createDefaultConfig());
    }
    setCurrentStepIndex(0);
    setErrors({});
    setConnectionTested(false);
  }, [editConfig]);

  const currentStep = WIZARD_STEPS[currentStepIndex];

  const validateForm = (): boolean => {
    const newErrors: Record<string, string> = {};
    if (!adConfig.config_name.trim()) newErrors.config_name = 'Configuration name is required';
    if (!adConfig.server.trim()) newErrors.server = 'Server address is required';
    if (!adConfig.username.trim()) newErrors.username = 'Username is required';
    if (!adConfig.password.trim()) newErrors.password = 'Password is required';
    if (!adConfig.base_dn.trim()) newErrors.base_dn = 'Base DN is required';
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
      toast.error('Please fill in all required fields');
      return;
    }

    setTestingConnection(true);
    setConnectionTested(false);

    try {
      await syncAD({
        provider: 'ad',
        config: {
          server: adConfig.server,
          username: adConfig.username,
          password: adConfig.password,
          base_dn: adConfig.base_dn,
          use_ssl: adConfig.use_ssl,
          skip_verify: adConfig.skip_verify,
        },
        dry_run: true,
      }).unwrap();

      setConnectionTested(true);
      toast.success('Connection verified successfully');
    } catch (error: any) {
      toast.error(`Connection test failed: ${error.data?.message || error.message || 'Unknown error'}`);
    } finally {
      setTestingConnection(false);
    }
  };

  const handleFinish = async () => {
    if (!connectionTested) {
      toast.error('Please test the connection before finishing');
      return;
    }

    try {
      const configData = {
        sync_type: 'active_directory' as const,
        config_name: adConfig.config_name,
        description: adConfig.description,
        ad_config: {
          server: adConfig.server,
          username: adConfig.username,
          password: adConfig.password,
          base_dn: adConfig.base_dn,
          filter: adConfig.filter,
          use_ssl: adConfig.use_ssl,
          skip_verify: adConfig.skip_verify,
        },
        is_active: true,
      };

      if (isEditing && editConfig?.id) {
        await updateConfig({
          id: editConfig.id,
          ...configData,
        }).unwrap();
        toast.success('Active Directory sync updated successfully');
      } else {
        await createConfig(configData).unwrap();
        toast.success('Active Directory sync configured successfully');
      }

      onSuccess();
    } catch (error: any) {
      toast.error(`Failed to ${isEditing ? 'update' : 'create'} configuration: ${error.data?.message || error.message}`);
    }
  };

  const canProceed = () => {
    if (currentStepIndex === 0) {
      return adConfig.config_name.trim() && adConfig.server.trim() && adConfig.username.trim() && adConfig.password.trim() && adConfig.base_dn.trim();
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
            <h2 className="text-lg font-semibold">Configure Active Directory Sync</h2>
            <p className="text-xs text-muted-foreground mt-0.5">
              {currentStep.id === 'configuration' && 'Configure your Active Directory connection settings'}
              {currentStep.id === 'test' && 'Test and verify the AD connection'}
              {currentStep.id === 'complete' && 'Review and finalize setup'}
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

      {/* Content - scrollable area with padding at bottom */}
      <div className="flex-1 overflow-y-auto px-8 py-6 min-h-0">
        <div className="w-full max-w-7xl">
          {currentStepIndex === 0 && (
            <>
             
              <ADConfigForm
                config={adConfig}
                onChange={setAdConfig}
                errors={errors}
              />
            </>
          )}

          {currentStepIndex === 1 && (
            <>
              <FormSectionHeader
                title="Test Connection"
                description="Verify your Active Directory connection"
              />
              <div className="space-y-4">
                <div className="rounded-lg border bg-muted/50 p-4">
                  <div className="flex items-start gap-3">
                    <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-primary/10">
                      <Server className="h-4 w-4 text-primary" />
                    </div>
                    <div className="flex-1 space-y-1">
                      <h3 className="font-medium text-sm">Connection Details</h3>
                      <div className="space-y-1 text-xs text-muted-foreground">
                        <div className="flex justify-between">
                          <span>Server:</span>
                          <span className="font-mono">{adConfig.server}</span>
                        </div>
                        <div className="flex justify-between">
                          <span>Base DN:</span>
                          <span className="font-mono">{adConfig.base_dn}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <Button
                  onClick={handleTestConnection}
                  disabled={testingConnection}
                  className="w-full"
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
                  <div className="rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900/50 dark:bg-green-950/20">
                    <div className="flex items-center gap-2">
                      <CheckCircle className="h-4 w-4 text-green-600 dark:text-green-500" />
                      <p className="text-xs font-medium text-green-900 dark:text-green-100">
                        Connection verified successfully!
                      </p>
                    </div>
                  </div>
                )}
              </div>
            </>
          )}

          {currentStepIndex === 2 && (
            <>
              <FormSectionHeader
                title="Review & Complete"
                description="Review configuration and finish setup"
              />
              <div className="space-y-4">
                <div className="rounded-lg border bg-muted/50 p-4 space-y-3">
                  <div>
                    <h3 className="font-medium text-sm mb-1">Configuration Name</h3>
                    <p className="text-sm text-muted-foreground">{adConfig.config_name}</p>
                  </div>
                  {adConfig.description && (
                    <div>
                      <h3 className="font-medium text-sm mb-1">Description</h3>
                      <p className="text-sm text-muted-foreground">{adConfig.description}</p>
                    </div>
                  )}
                  <FormDivider />
                  <div>
                    <h3 className="font-medium text-sm mb-2">Connection Details</h3>
                    <div className="space-y-1 text-xs text-muted-foreground">
                      <div className="flex justify-between">
                        <span>Server:</span>
                        <span className="font-mono">{adConfig.server}</span>
                      </div>
                      <div className="flex justify-between">
                        <span>Base DN:</span>
                        <span className="font-mono">{adConfig.base_dn}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </>
          )}
        </div>
      </div>

      {/* Footer Actions - sticky at bottom */}
      <div className="flex-shrink-0 border-t bg-background pt-4 pb-4 mt-auto px-8">
        <div className="flex items-center justify-between gap-4">
          {/* Back/Cancel Button on Left */}
          <div className="flex items-center gap-2 min-w-[120px]">
            <Button
              variant="outline"
              onClick={handleBack}
              size="default"
            >
              {currentStepIndex > 0 ? (
                <>
                  <ArrowLeft className="mr-2 h-4 w-4" />
                  Back
                </>
              ) : (
                'Cancel'
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
                    <span className={cn(
                      "text-sm font-medium",
                      isActive && "text-foreground",
                      !isActive && "text-muted-foreground"
                    )}>
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
