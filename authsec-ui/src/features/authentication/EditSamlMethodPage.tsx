import React, { useState, useEffect, useMemo } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../components/ui/dialog";
import {
  ArrowLeft,
  CheckCircle,
  Loader2,
} from "lucide-react";
import { toast } from "../../lib/toast";
import {
  FormRoot,
  FormBody,
  FormSection,
  FormSectionHeader,
  FormField,
  FormGrid,
  FormDivider,
  FormInput,
  FormCopyField,
} from "../../theme";
import {
  useGetSamlProviderQuery,
  useUpdateSamlProviderMutation,
  useLazyGetSamlMetadataQuery
} from "../../app/api/samlApi";
import { SessionManager } from "../../utils/sessionManager";

const NAME_ID_FORMATS = [
  { value: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress", label: "Email Address" },
  { value: "urn:oasis:names:tc:SAML:1.1:nameid-format:unspecified", label: "Unspecified" },
  { value: "urn:oasis:names:tc:SAML:2.0:nameid-format:persistent", label: "Persistent" },
  { value: "urn:oasis:names:tc:SAML:2.0:nameid-format:transient", label: "Transient" },
];

export function EditSamlMethodPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const session = SessionManager.getSession();
  const tenantId = session?.tenant_id || "";

  const [metadata, setMetadata] = useState<{ entity_id: string; acs_url: string } | null>(null);
  const [formData, setFormData] = useState({
    provider_name: "",
    display_name: "",
    entity_id: "",
    sso_url: "",
    certificate: "",
    metadata_url: "",
    name_id_format: "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
    attribute_email: "email",
    attribute_first_name: "firstName",
    attribute_last_name: "lastName",
    is_active: true,
    sort_order: 1,
  });
  const [showConfirmDialog, setShowConfirmDialog] = useState(false);

  // Fetch existing SAML provider data
  const { data: providerData, isLoading: isLoadingProvider } = useGetSamlProviderQuery(
    { tenant_id: tenantId, provider_id: id || "" },
    { skip: !tenantId || !id }
  );

  const [fetchMetadata, { isLoading: loadingMetadata }] = useLazyGetSamlMetadataQuery();
  const [updateSamlProvider, { isLoading: isUpdating }] = useUpdateSamlProviderMutation();

  // Pre-populate form with existing provider data
  useEffect(() => {
    if (providerData?.provider) {
      const provider = providerData.provider;
      setFormData({
        provider_name: provider.provider_name || "",
        display_name: provider.display_name || "",
        entity_id: provider.entity_id || "",
        sso_url: provider.sso_url || "",
        certificate: provider.certificate || "",
        metadata_url: provider.metadata_url || "",
        name_id_format: provider.name_id_format || "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress",
        attribute_email: provider.attribute_mapping?.email || "email",
        attribute_first_name: provider.attribute_mapping?.first_name || "firstName",
        attribute_last_name: provider.attribute_mapping?.last_name || "lastName",
        is_active: provider.is_active ?? true,
        sort_order: provider.sort_order ?? 1,
      });

      // Set metadata from existing provider
      setMetadata({
        entity_id: provider.entity_id || "",
        acs_url: "", // This should come from metadata endpoint if needed
      });
    }
  }, [providerData]);

  // Fetch metadata when client_id is available
  useEffect(() => {
    const clientId = providerData?.provider?.client_id;
    if (clientId && tenantId) {
      fetchMetadata({ tenant_id: tenantId, client_id: clientId })
        .unwrap()
        .then((data) => {
          setMetadata({ entity_id: data.entity_id, acs_url: data.acs_url });
        })
        .catch(() => {
          // If metadata fetch fails, keep the entity_id from provider
          console.log("Could not fetch fresh metadata, using existing values");
        });
    }
  }, [providerData?.provider?.client_id, tenantId, fetchMetadata]);

  const canComplete = useMemo(
    () =>
      Boolean(
        formData.provider_name &&
          formData.display_name &&
          formData.entity_id &&
          formData.sso_url &&
          formData.certificate
      ),
    [formData]
  );

  const handleComplete = () => {
    if (canComplete) setShowConfirmDialog(true);
    else toast.error("Please fill all required fields");
  };

  const handleConfirmUpdate = async () => {
    if (!canComplete || !id) {
      toast.error("Please fill in all required fields");
      return;
    }

    try {
      const payload = {
        tenant_id: tenantId,
        provider_id: id,
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

      await updateSamlProvider(payload).unwrap();
      toast.success("SAML provider updated successfully!");
      setShowConfirmDialog(false);
      navigate("/authentication");
    } catch (error: any) {
      console.error("Failed to update SAML provider:", error);
      toast.error(error?.data?.message || "Failed to update SAML provider");
    }
  };

  if (isLoadingProvider) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
        <div className="container mx-auto max-w-8xl space-y-8 px-8 py-8">
          <div className="flex items-center justify-center h-64">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
            <span className="ml-2 text-foreground">Loading provider data...</span>
          </div>
        </div>
      </div>
    );
  }

  if (!providerData?.provider) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
        <div className="container mx-auto max-w-8xl space-y-8 px-8 py-8">
          <div className="flex flex-col items-center justify-center h-64 space-y-4">
            <p className="text-foreground">Provider not found</p>
            <Button onClick={() => navigate("/authentication")}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Authentication
            </Button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="container mx-auto max-w-8xl space-y-8 px-8 py-8">
        <header className="bg-card border border-border rounded-sm p-6 shadow-sm">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => navigate("/authentication")}
                className="h-8 px-2"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <div>
                <h1 className="text-2xl font-semibold">
                  Edit SAML Authentication Method
                </h1>
                <p className="text-sm text-foreground mt-1">
                  Update SAML 2.0 provider configuration and user attribute mapping.
                </p>
              </div>
            </div>
            <Button
              onClick={handleComplete}
              disabled={!canComplete || isUpdating}
              className="min-w-[140px]"
            >
              {isUpdating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Updating...
                </>
              ) : (
                <>
                  <CheckCircle className="mr-2 h-4 w-4" />
                  Review & Update
                </>
              )}
            </Button>
          </div>
        </header>

        <FormRoot className="px-0" maxWidth="96rem">
          <FormBody>
            {/* Service Provider Metadata */}
            {metadata && (
              <FormSection>
                <FormSectionHeader
                  title="Service Provider Metadata"
                  description="Provide these values to your Identity Provider (IdP)."
                />
                <FormGrid columns={2}>
                  <FormCopyField
                    label="Entity ID (Audience URL)"
                    value={metadata.entity_id}
                    description="Unique identifier for this service provider"
                  />
                  <FormCopyField
                    label="ACS URL (Reply URL)"
                    value={metadata.acs_url}
                    description="URL where SAML assertions are sent"
                  />
                </FormGrid>
              </FormSection>
            )}

            <FormDivider />

            {/* Basic Configuration */}
            <FormSection>
              <FormSectionHeader
                title="Provider Configuration"
                description="Basic settings for the SAML provider."
              />
              <FormGrid columns={2}>
                <FormField label="Provider Name" htmlFor="provider_name" required>
                  <FormInput
                    id="provider_name"
                    value={formData.provider_name}
                    onChange={(e) => setFormData({ ...formData, provider_name: e.target.value })}
                    placeholder="e.g., okta, azure, google"
                  />
                </FormField>

                <FormField label="Display Name" htmlFor="display_name" required>
                  <FormInput
                    id="display_name"
                    value={formData.display_name}
                    onChange={(e) => setFormData({ ...formData, display_name: e.target.value })}
                    placeholder="e.g., Okta SSO, Azure AD"
                  />
                </FormField>
              </FormGrid>
            </FormSection>

            <FormDivider />

            {/* Identity Provider Configuration */}
            <FormSection>
              <FormSectionHeader
                title="Identity Provider Settings"
                description="Configuration provided by your IdP."
              />
              <FormGrid columns={1}>
                <FormField label="Entity ID (Issuer ID)" htmlFor="entity_id" required>
                  <FormInput
                    id="entity_id"
                    value={formData.entity_id}
                    onChange={(e) => setFormData({ ...formData, entity_id: e.target.value })}
                    placeholder="https://your-idp.com/entity-id"
                  />
                </FormField>

                <FormField label="SSO URL" htmlFor="sso_url" required>
                  <FormInput
                    id="sso_url"
                    value={formData.sso_url}
                    onChange={(e) => setFormData({ ...formData, sso_url: e.target.value })}
                    placeholder="https://idp.example.com/sso/saml"
                  />
                </FormField>

                <FormField label="X.509 Certificate" htmlFor="certificate" required>
                  <textarea
                    id="certificate"
                    value={formData.certificate}
                    onChange={(e) => setFormData({ ...formData, certificate: e.target.value })}
                    placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----"
                    className="flex min-h-[120px] w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background placeholder:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 font-mono"
                  />
                </FormField>

                <FormField label="Metadata URL (Optional)" htmlFor="metadata_url">
                  <FormInput
                    id="metadata_url"
                    value={formData.metadata_url}
                    onChange={(e) => setFormData({ ...formData, metadata_url: e.target.value })}
                    placeholder="https://idp.example.com/metadata.xml"
                  />
                </FormField>

                <FormField label="Name ID Format" htmlFor="name_id_format" required>
                  <select
                    id="name_id_format"
                    value={formData.name_id_format}
                    onChange={(e) => setFormData({ ...formData, name_id_format: e.target.value })}
                    className="flex h-12 w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    {NAME_ID_FORMATS.map((format) => (
                      <option key={format.value} value={format.value}>
                        {format.label}
                      </option>
                    ))}
                  </select>
                </FormField>
              </FormGrid>
            </FormSection>

            <FormDivider />

            {/* Attribute Mapping */}
            <FormSection>
              <FormSectionHeader
                title="Attribute Mapping"
                description="Map SAML attributes to user properties."
              />
              <FormGrid columns={3}>
                <FormField label="Email Attribute" htmlFor="attribute_email" required>
                  <FormInput
                    id="attribute_email"
                    value={formData.attribute_email}
                    onChange={(e) => setFormData({ ...formData, attribute_email: e.target.value })}
                    placeholder="email"
                  />
                </FormField>

                <FormField label="First Name Attribute" htmlFor="attribute_first_name" required>
                  <FormInput
                    id="attribute_first_name"
                    value={formData.attribute_first_name}
                    onChange={(e) => setFormData({ ...formData, attribute_first_name: e.target.value })}
                    placeholder="firstName"
                  />
                </FormField>

                <FormField label="Last Name Attribute" htmlFor="attribute_last_name" required>
                  <FormInput
                    id="attribute_last_name"
                    value={formData.attribute_last_name}
                    onChange={(e) => setFormData({ ...formData, attribute_last_name: e.target.value })}
                    placeholder="lastName"
                  />
                </FormField>
              </FormGrid>
            </FormSection>

            <FormDivider />

            {/* Advanced Settings */}
            <FormSection>
              <FormSectionHeader
                title="Advanced Settings"
                description="Additional configuration options."
              />
              <FormGrid columns={2}>
                <FormField label="Sort Order" htmlFor="sort_order">
                  <FormInput
                    id="sort_order"
                    type="number"
                    value={formData.sort_order}
                    onChange={(e) => setFormData({ ...formData, sort_order: parseInt(e.target.value) || 1 })}
                    min={1}
                  />
                </FormField>

                <FormField label="Active" htmlFor="is_active">
                  <div className="flex items-center h-12">
                    <input
                      id="is_active"
                      type="checkbox"
                      checked={formData.is_active}
                      onChange={(e) => setFormData({ ...formData, is_active: e.target.checked })}
                      className="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
                    />
                    <label htmlFor="is_active" className="ml-2 text-sm text-foreground">
                      Enable this SAML provider
                    </label>
                  </div>
                </FormField>
              </FormGrid>
            </FormSection>
          </FormBody>
        </FormRoot>
      </div>

      {/* Confirmation Dialog */}
      <Dialog open={showConfirmDialog} onOpenChange={setShowConfirmDialog}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Confirm SAML Provider Update</DialogTitle>
            <DialogDescription>
              Please review your SAML provider configuration before updating.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div>
              <div className="text-sm font-medium">Provider Name</div>
              <div className="text-sm text-foreground">{formData.provider_name}</div>
            </div>
            <div>
              <div className="text-sm font-medium">Display Name</div>
              <div className="text-sm text-foreground">{formData.display_name}</div>
            </div>
            <div>
              <div className="text-sm font-medium">Entity ID (Issuer ID)</div>
              <div className="text-sm text-foreground font-mono">{formData.entity_id}</div>
            </div>
            <div>
              <div className="text-sm font-medium">SSO URL</div>
              <div className="text-sm text-foreground">{formData.sso_url}</div>
            </div>
            <div>
              <div className="text-sm font-medium">Status</div>
              <div className="text-sm text-foreground">
                {formData.is_active ? "Active" : "Inactive"}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setShowConfirmDialog(false)}>
              Cancel
            </Button>
            <Button onClick={handleConfirmUpdate} disabled={isUpdating}>
              {isUpdating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Updating...
                </>
              ) : (
                "Confirm & Update"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
