import React, { useState, useEffect, useMemo, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/button";
import {
  ArrowLeft,
  CheckCircle,
  Loader2,
  Check,
  ChevronRight,
  Settings,
  X,
} from "lucide-react";
import { toast } from "../../lib/toast";
import { cn } from "../../lib/utils";
import { FormField, FormInput, FormCopyField } from "../../theme";
import {
  useAddSamlProviderMutation,
  useLazyGetSamlMetadataQuery,
} from "../../app/api/samlApi";
import { useGetClientsQuery } from "../../app/api/clientApi";
import { SessionManager } from "../../utils/sessionManager";
import { current } from "@reduxjs/toolkit";

const NAME_ID_FORMATS = [
  {
    value: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    label: "Email Address",
  },
  {
    value: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified",
    label: "Unspecified",
  },
  {
    value: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent",
    label: "Persistent",
  },
  {
    value: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient",
    label: "Transient",
  },
];

// Wizard steps
const WIZARD_STEPS = [
  { id: "configuration", label: "Configuration", icon: Settings },
  { id: "Identity-Provider", label: "Identity Provider", icon: Settings },
  { id: "review", label: "Review", icon: CheckCircle },
];

export function CreateSamlMethodPage() {
  const navigate = useNavigate();
  const session = SessionManager.getSession();
  const tenantId = session?.tenant_id || "";

  const [selectedClientId, setSelectedClientId] = useState("");
  const [metadata, setMetadata] = useState<{
    entity_id: string;
    acs_url: string;
  } | null>(null);
  const [currentStepIndex, setCurrentStepIndex] = useState(0);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [formData, setFormData] = useState({
    provider_name: "",
    display_name: "",
    entity_id: "",
    sso_url: "",
    certificate: "",
    metadata_url: "",
    name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    attribute_email: "",
    attribute_first_name: "",
    attribute_last_name: "",
    is_active: true,
    sort_order: 1,
  });

  const { data: clientsResponse } = useGetClientsQuery(
    { tenant_id: tenantId, active_only: false },
    { skip: !tenantId }
  );
  const [fetchMetadata, { isLoading: loadingMetadata }] =
    useLazyGetSamlMetadataQuery();
  const [addSamlProvider, { isLoading: isCreating }] =
    useAddSamlProviderMutation();

  const clients = clientsResponse?.clients || [];
  const currentStep = WIZARD_STEPS[currentStepIndex];

  useEffect(() => {
    if (selectedClientId && tenantId) {
      fetchMetadata({ tenant_id: tenantId, client_id: selectedClientId })
        .unwrap()
        .then((data) => {
          setMetadata({ entity_id: data.entity_id, acs_url: data.acs_url });
        })
        .catch(() => {
          toast.error("Failed to fetch SAML metadata");
        });
    }
  }, [selectedClientId, tenantId, fetchMetadata]);

  const handleBack = useCallback(() => {
    if (currentStepIndex > 0) {
      setCurrentStepIndex(currentStepIndex - 1);
    } else {
      navigate("/authentication");
    }
  }, [currentStepIndex, navigate]);

  const handleNext = useCallback(() => {
    if (currentStepIndex === 0) {
      // Validate all required fields at once
      if (!selectedClientId) {
        toast.error("Please select a client");
        return;
      }
      if (!formData.provider_name || !formData.display_name) {
        toast.error("Please fill in provider information");
        return;
      }
    }
    if (currentStepIndex === 1) {
      if (!formData.entity_id || !formData.sso_url || !formData.certificate) {
        toast.error("Please fill in all required IDP fields");
        return;
      }
    }

    if (currentStepIndex < WIZARD_STEPS.length - 1) {
      setErrors({});
      setCurrentStepIndex(currentStepIndex + 1);
    }
  }, [currentStepIndex, selectedClientId, formData]);

  const canProceed = () => {
    if (currentStepIndex === 0) {
      // All required fields must be filled
      return Boolean(
        selectedClientId && formData.provider_name && formData.display_name
      );
    }
    if (currentStepIndex === 1) {
      // Identity Provider step
      return Boolean(
        formData.entity_id && formData.sso_url && formData.certificate
      );
    }
    if (currentStepIndex === 2) {
      // Review step
      return Boolean(
        selectedClientId && formData.provider_name && formData.entity_id
      );
    }
    return false;
  };

  const handleFinish = async () => {
    if (!canProceed()) {
      toast.error("Please fill in all required fields");
      return;
    }

    try {
      const payload = {
        tenant_id: tenantId,
        client_id: selectedClientId,
        provider_name: formData.provider_name,
        display_name: formData.display_name,
        entity_id: formData.entity_id,
        sso_url: formData.sso_url,
        certificate: formData.certificate,
        metadata_url: formData.metadata_url || undefined,
        name_id_format: formData.name_id_format,
        attribute_mapping: {
          email: formData.attribute_email,
          first_name: formData.attribute_first_name,
          last_name: formData.attribute_last_name,
        },
        is_active: formData.is_active,
        sort_order: formData.sort_order,
      };

      await addSamlProvider(payload).unwrap();
      toast.success("SAML provider created successfully!");
      navigate("/", {
        state: {
          authProviderCreated: true,
          clientId: selectedClientId
        }
      });
    } catch (error: any) {
      console.error("Failed to create SAML provider:", error);
      toast.error(error?.data?.message || "Failed to create SAML provider");
    }
  };

  const getStepSubtitle = () => {
    switch (currentStep.id) {
      case "configuration":
        return "Configure your SAML authentication settings";
      case "Identity-Provider":
        return "Identity Provider settings for your SAML method";
      case "review":
        return "Review and finalize your SAML configuration";
      default:
        return "";
    }
  };
  const getSteptitle = () => {
    switch (currentStep.id) {
      case "configuration":
        return "Configure";
      case "Identity-Provider":
        return "Identity Provider";
      case "review":
        return "Review";
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
            <h2 className="text-lg font-semibold">{getSteptitle()}</h2>
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
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Left Column */}
              <div className="space-y-6">
                {/* Client Selection Section */}
                <div>
                  <h3 className="text-base font-semibold mb-3">
                    Client Application
                  </h3>

                  <FormField label="Client" htmlFor="client" required>
                    <select
                      id="client"
                      value={selectedClientId}
                      onChange={(e) => setSelectedClientId(e.target.value)}
                      className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      <option value="">Select a client...</option>
                      {clients.map((client) => (
                        <option key={client.client_id} value={client.client_id}>
                          {client.name} ({client.client_id})
                        </option>
                      ))}
                    </select>
                  </FormField>

                  {loadingMetadata && (
                    <div className="flex items-center gap-2 text-sm text-muted-foreground mt-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span>Loading SP metadata...</span>
                    </div>
                  )}

                  {metadata && (
                    <div className="mt-4 space-y-3">
                      <div className="mb-2">
                        <h4 className="text-sm font-semibold">
                          Service Provider Metadata
                        </h4>
                        <p className="text-xs text-muted-foreground">
                          Configure these in your Identity Provider
                        </p>
                      </div>

                      <FormField label="Entity ID (Audience URI)">
                        <FormCopyField
                          value={metadata.entity_id}
                          onCopy={() =>
                            toast.success("Entity ID copied to clipboard!")
                          }
                          className="font-mono text-sm"
                        />
                      </FormField>

                      <FormField label="ACS URL (Assertion Consumer Service)">
                        <FormCopyField
                          value={metadata.acs_url}
                          onCopy={() =>
                            toast.success("ACS URL copied to clipboard!")
                          }
                          className="font-mono text-sm"
                        />
                      </FormField>
                    </div>
                  )}
                </div>

                {/* Provider Information Section */}
                <div>
                  <h3 className="text-base font-semibold mb-3">
                    Provider Information
                  </h3>

                  <div className="space-y-3">
                    <FormField
                      label="Provider Name"
                      htmlFor="provider_name"
                      required
                    >
                      <FormInput
                        id="provider_name"
                        placeholder="e.g., okta-saml"
                        value={formData.provider_name}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            provider_name: e.target.value,
                          })
                        }
                        className="h-9"
                      />
                    </FormField>

                    <FormField
                      label="Display Name"
                      htmlFor="display_name"
                      required
                    >
                      <FormInput
                        id="display_name"
                        placeholder="e.g., Okta SAML"
                        value={formData.display_name}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            display_name: e.target.value,
                          })
                        }
                        className="h-9"
                      />
                    </FormField>
                  </div>
                </div>
              </div>

              {/* Right Column */}
              <div className="space-y-6">
                {/* Attribute Mapping Section */}
                <div>
                  <h3 className="text-base font-semibold mb-3">
                    Attribute Mapping
                  </h3>

                  <div className="space-y-3">
                    <FormField
                      label="Email Attribute"
                      htmlFor="attribute_email"
                    >
                      <FormInput
                        id="attribute_email"
                        placeholder="email"
                        value={formData.attribute_email}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            attribute_email: e.target.value,
                          })
                        }
                        className="h-9"
                      />
                    </FormField>

                    <FormField
                      label="First Name Attribute"
                      htmlFor="attribute_first_name"
                    >
                      <FormInput
                        id="attribute_first_name"
                        placeholder="firstName"
                        value={formData.attribute_first_name}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            attribute_first_name: e.target.value,
                          })
                        }
                        className="h-9"
                      />
                    </FormField>

                    <FormField
                      label="Last Name Attribute"
                      htmlFor="attribute_last_name"
                    >
                      <FormInput
                        id="attribute_last_name"
                        placeholder="lastName"
                        value={formData.attribute_last_name}
                        onChange={(e) =>
                          setFormData({
                            ...formData,
                            attribute_last_name: e.target.value,
                          })
                        }
                        className="h-9"
                      />
                    </FormField>
                  </div>
                </div>
              </div>
            </div>
          )}

          {currentStepIndex === 1 && (
            <div>
              <h3 className="text-base font-semibold mb-3">
                Identity Provider Configuration
              </h3>

              <div className="space-y-3">
                <FormField
                  label="Entity ID (Issuer ID)"
                  htmlFor="entity_id"
                  required
                >
                  <FormInput
                    id="entity_id"
                    placeholder="https://your-idp.com/entity-id"
                    value={formData.entity_id}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        entity_id: e.target.value,
                      })
                    }
                    className="h-9"
                  />
                </FormField>

                <FormField label="SSO URL" htmlFor="sso_url" required>
                  <FormInput
                    id="sso_url"
                    placeholder="https://your-idp.com/sso/saml"
                    value={formData.sso_url}
                    onChange={(e) =>
                      setFormData({ ...formData, sso_url: e.target.value })
                    }
                    className="h-9"
                  />
                </FormField>

                <FormField
                  label="X.509 Certificate"
                  htmlFor="certificate"
                  required
                >
                  <textarea
                    id="certificate"
                    placeholder="MIIDtDCCApygAwIBAgIGAZp327n/MA0GCSqGSIb3DQEBC..."
                    value={formData.certificate}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        certificate: e.target.value,
                      })
                    }
                    rows={6}
                    className="flex w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 font-mono resize-vertical"
                  />
                </FormField>

                <FormField label="Metadata URL" htmlFor="metadata_url">
                  <FormInput
                    id="metadata_url"
                    placeholder="https://your-idp.com/metadata (optional)"
                    value={formData.metadata_url}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        metadata_url: e.target.value,
                      })
                    }
                    className="h-9"
                  />
                </FormField>

                <FormField label="Name ID Format" htmlFor="name_id_format">
                  <select
                    id="name_id_format"
                    value={formData.name_id_format}
                    onChange={(e) =>
                      setFormData({
                        ...formData,
                        name_id_format: e.target.value,
                      })
                    }
                    className="flex h-9 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {NAME_ID_FORMATS.map((format) => (
                      <option key={format.value} value={format.value}>
                        {format.label}
                      </option>
                    ))}
                  </select>
                </FormField>
              </div>
            </div>
          )}

          {/* Step 1: Review & Create */}
          {currentStepIndex === 2 && (
            <div className="space-y-3">
              <h3 className="text-base font-semibold mb-3">Review & Create</h3>

              <div className="rounded-lg border bg-muted/50 p-3 space-y-2.5">
                <div>
                  <h4 className="font-medium text-xs mb-0.5">
                    Client Application
                  </h4>
                  <p className="text-xs text-muted-foreground font-mono">
                    {selectedClientId || "—"}
                  </p>
                </div>

                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1">
                    Provider Information
                  </h4>
                  <div className="space-y-0.5 text-[11px] text-muted-foreground">
                    <div className="flex justify-between gap-4">
                      <span>Provider Name:</span>
                      <span className="font-mono">
                        {formData.provider_name || "—"}
                      </span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span>Display Name:</span>
                      <span>{formData.display_name || "—"}</span>
                    </div>
                  </div>
                </div>

                {metadata && (
                  <div className="border-t pt-2.5">
                    <h4 className="font-medium text-xs mb-1">
                      Service Provider
                    </h4>
                    <div className="space-y-0.5 text-[11px] text-muted-foreground">
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium">Entity ID:</span>
                        <span className="font-mono break-all">
                          {metadata.entity_id}
                        </span>
                      </div>
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium">ACS URL:</span>
                        <span className="font-mono break-all">
                          {metadata.acs_url}
                        </span>
                      </div>
                    </div>
                  </div>
                )}

                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1">
                    Identity Provider
                  </h4>
                  <div className="space-y-0.5 text-[11px] text-muted-foreground">
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">Entity ID (Issuer):</span>
                      <span className="font-mono break-all">
                        {formData.entity_id || "—"}
                      </span>
                    </div>
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">SSO URL:</span>
                      <span className="font-mono break-all">
                        {formData.sso_url || "—"}
                      </span>
                    </div>
                    <div className="flex flex-col gap-0.5">
                      <span className="font-medium">Name ID Format:</span>
                      <span className="text-xs">
                        {NAME_ID_FORMATS.find(
                          (f) => f.value === formData.name_id_format
                        )?.label || formData.name_id_format}
                      </span>
                    </div>
                    {formData.metadata_url && (
                      <div className="flex flex-col gap-0.5">
                        <span className="font-medium">Metadata URL:</span>
                        <span className="font-mono break-all">
                          {formData.metadata_url}
                        </span>
                      </div>
                    )}
                  </div>
                </div>

                <div className="border-t pt-2.5">
                  <h4 className="font-medium text-xs mb-1">
                    Attribute Mapping
                  </h4>
                  <div className="space-y-0.5 text-[11px] text-muted-foreground">
                    <div className="flex justify-between gap-4">
                      <span>Email:</span>
                      <span className="font-mono">
                        {formData.attribute_email}
                      </span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span>First Name:</span>
                      <span className="font-mono">
                        {formData.attribute_first_name}
                      </span>
                    </div>
                    <div className="flex justify-between gap-4">
                      <span>Last Name:</span>
                      <span className="font-mono">
                        {formData.attribute_last_name}
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
                      isCompleted && "opacity-60"
                    )}
                  >
                    <div
                      className={cn(
                        "flex h-6 w-6 items-center justify-center rounded-full text-xs",
                        isCompleted && "bg-primary text-primary-foreground",
                        isActive && "bg-primary/20 text-primary",
                        !isActive &&
                          !isCompleted &&
                          "bg-muted text-muted-foreground"
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
