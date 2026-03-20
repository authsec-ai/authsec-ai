import { type FormEvent, useState, useMemo, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormContainer, FormInput, FormTextarea } from "@form";
import { ArrowLeft, Loader2, Key, Database } from "lucide-react";
import { SearchableSelect } from "@/components/ui/searchable-select";
import { Badge } from "@/components/ui/badge";
import { Label } from "@/components/ui/label";
import { toast } from "@/lib/toast";
import {
  useCreateScopesMutation,
  useUpdateScopeMutation,
} from "@/app/api/scopesApi";
import {
  useCreateEndUserScopeMutation,
  useUpdateEndUserScopeMutation,
  useGetEndUserScopeQuery,
} from "@/app/api/enduser/scopesApi";
import { useGetPermissionResourcesQuery } from "@/app/api/permissionsApi";
import { resolveTenantId } from "@/utils/workspace";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";

export default function CreateScopePage() {
  const navigate = useContextualNavigate();
  const { id: scopeId } = useParams<{ id: string }>();
  const isEditMode = Boolean(scopeId);
  const { isAdmin } = useRbacAudience();

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            pageTitle: isEditMode ? "Edit Admin Scope" : "Create Admin Scope",
            pageSubtitle: isEditMode
              ? "Update capabilities for your administrators."
              : "Define fine-grained capabilities for your administrators.",
            badgeLabel: "Admin RBAC",
            cardTitle: "Admin scope details",
            cardDescription:
              "Choose names that reflect the privileged actions this scope unlocks.",
            nameLabel: "Scope name",
            namePlaceholder: "e.g., manage:users, audit:logs",
            nameHelper:
              "Use lowercase with colons or hyphens for readability (e.g. `manage:users`).",
            descriptionHelper:
              "Explain which administrative workflows require this scope.",
            createCta: isEditMode ? "Update" : "Create",
            successNoun: "Admin scope",
            entityNoun: "scope",
            exampleTitle: "Admin Scope Ideas",
            examples: [
              {
                scope: "manage:users",
                description: "Invite, deactivate, and reset users",
              },
              {
                scope: "audit:logs",
                description: "View authentication and audit logs",
              },
              {
                scope: "configure:mfa",
                description: "Change multi-factor policies",
              },
              { scope: "admin:all", description: "Full administrator control" },
            ],
          }
        : {
            pageTitle: isEditMode
              ? "Edit End-user Scope"
              : "Create End-user Scope",
            pageSubtitle: isEditMode
              ? "Update permissions for your customers."
              : "Describe the permissions customers can earn across your apps.",
            badgeLabel: "End-user access",
            cardTitle: "End-user scope details",
            cardDescription:
              "Name the actions your customers, partners, or members can request.",
            nameLabel: "Scope name",
            namePlaceholder: "e.g., profile:read, billing:update",
            nameHelper:
              "Match the naming expected by your APIs (e.g. `profile:read`).",
            descriptionHelper:
              "Share when to grant this scope and what parts of the product it touches.",
            createCta: isEditMode ? "Update" : "Create",
            successNoun: "End-user scope",
            entityNoun: "scope",
            exampleTitle: "Popular End-user Scopes",
            examples: [
              {
                scope: "profile:read",
                description: "View personal profile data",
              },
              {
                scope: "billing:update",
                description: "Manage payment methods",
              },
              { scope: "support:create", description: "Open support tickets" },
              {
                scope: "notifications:manage",
                description: "Control notification preferences",
              },
            ],
          },
    [isAdmin, isEditMode]
  );

  // API hooks
  const [createAdminScopes] = useCreateScopesMutation();
  const [updateAdminScope] = useUpdateScopeMutation();
  const [createEndUserScope] = useCreateEndUserScopeMutation();
  const [updateEndUserScope] = useUpdateEndUserScopeMutation();

  // Resolve tenant ID from workspace context (required for end-user APIs)
  const tenantId = resolveTenantId();

  // Fetch existing scope data in edit mode (end-user only, admin uses scopesApi)
  const { data: endUserScopeData, isLoading: isLoadingScope } =
    useGetEndUserScopeQuery(
      scopeId || "",
      { skip: !isEditMode || isAdmin || !scopeId }
    );

  // Fetch available resources for end-user scopes
  const { data: availableResources = [], isLoading: isLoadingResources } =
    useGetPermissionResourcesQuery(
      { audience: isAdmin ? "admin" : "endUser" },
      { skip: isAdmin }
    );

  // Scope state
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [scopeName, setScopeName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedResources, setSelectedResources] = useState<string[]>([]);

  // Pre-fill form in edit mode
  useEffect(() => {
    if (isEditMode && endUserScopeData && !isAdmin) {
      setScopeName(endUserScopeData.name);
      setDescription(endUserScopeData.description || "");
      setSelectedResources(endUserScopeData.resources || []);
    }
  }, [isEditMode, endUserScopeData, isAdmin]);

  const handleSubmit = async (event?: FormEvent) => {
    event?.preventDefault();
    setIsSubmitting(true);

    if (!scopeName.trim()) {
      toast.error(`${audienceCopy.nameLabel} is required`);
      setIsSubmitting(false);
      return;
    }

    if (!isAdmin && selectedResources.length === 0) {
      toast.error("At least one resource is required");
      setIsSubmitting(false);
      return;
    }

    try {
      if (isEditMode && scopeId) {
        // Update existing scope
        if (isAdmin) {
          await updateAdminScope({
            id: scopeId,
            data: {
              tenant_id: tenantId || "",
              name: scopeName.trim(),
              description: description.trim() || undefined,
            },
          }).unwrap();
        } else {
          await updateEndUserScope({
            scope_id: scopeId,
            scope_name: scopeName.trim(),
            resources: selectedResources,
            description: description.trim() || undefined,
          }).unwrap();
        }
        toast.success(
          `${audienceCopy.successNoun} "${scopeName}" updated successfully`
        );
      } else {
        // Create new scope
        if (isAdmin) {
          await createAdminScopes({
            tenant_id: tenantId || "",
            scopes: [
              {
                name: scopeName.trim(),
                description: description.trim() || undefined,
              },
            ],
          }).unwrap();
        } else {
          await createEndUserScope({
            scope_name: scopeName.trim(),
            resources: selectedResources,
            description: description.trim() || undefined,
          }).unwrap();
        }
        toast.success(
          `${audienceCopy.successNoun} "${scopeName}" created successfully`
        );
      }

      navigate("/scopes");
    } catch (error: any) {
      console.error(
        `Error ${isEditMode ? "updating" : "creating"} scope:`,
        error
      );
      const message =
        error?.data?.message ||
        error?.message ||
        `Failed to ${
          isEditMode ? "update" : "create"
        } scope. Please try again.`;
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    navigate("/scopes");
  };

  const isFormValid = () => {
    if (isAdmin) return scopeName.trim().length > 0;
    return scopeName.trim().length > 0 && selectedResources.length > 0;
  };

  // Show loading state when fetching scope data in edit mode
  if (isEditMode && isLoadingScope && !isAdmin) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950 flex items-center justify-center">
        <div className="flex items-center gap-2">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>Loading scope...</span>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="container mx-auto max-w-8xl space-y-8 px-8 py-8">
        {/* Header */}
        <header className="bg-card border border-border rounded-sm p-6 shadow-sm">
          <div className="flex items-center justify-between gap-4">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCancel}
                className="h-8 px-2"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <div>
                <h1 className="text-2xl font-semibold">
                  {audienceCopy.pageTitle}
                </h1>
                <p className="text-sm text-foreground mt-1">
                  {audienceCopy.pageSubtitle}
                </p>
              </div>
            </div>
            <Button
              type="submit"
              form="create-scope-form"
              disabled={!isFormValid() || isSubmitting}
              className="min-w-[140px]"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {isEditMode ? "Updating..." : "Creating..."}
                </>
              ) : (
                audienceCopy.createCta
              )}
            </Button>
          </div>
        </header>

        {/* Main Form */}
        <form
          id="create-scope-form"
          onSubmit={handleSubmit}
          className="space-y-6"
        >
          <FormInput
            id="scope-name"
            label={audienceCopy.nameLabel}
            required
            placeholder={audienceCopy.namePlaceholder}
            value={scopeName}
            onChange={(e) => setScopeName(e.target.value)}
            disabled={isSubmitting}
            className="font-mono"
            helperText={audienceCopy.nameHelper}
          />

          <FormTextarea
            id="description"
            label="Description"
            placeholder={`Describe what this ${audienceCopy.entityNoun} allows...`}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={isSubmitting}
            rows={4}
            helperText={audienceCopy.descriptionHelper}
          />

          {/* Resources field for end-user scopes */}
          {!isAdmin && (
            <div className="space-y-3">
              <Label className="flex items-center gap-2 text-sm font-medium">
                <Database className="h-4 w-4 text-primary" />
                Resources <span className="text-destructive">*</span>
              </Label>

              <SearchableSelect
                multiple={true}
                options={availableResources.map(r => ({ value: r, label: r }))}
                value={selectedResources}
                onChange={(values) => setSelectedResources(
                  Array.isArray(values) ? values : [values].filter(Boolean) as string[]
                )}
                placeholder={isLoadingResources ? "Loading..." : "Select resources..."}
                searchPlaceholder="Search resources..."
                emptyText="No resources found"
                disabled={isLoadingResources || isSubmitting}
                showSelectAll={true}
                maxBadges={3}
                className="h-11"
              />

              {selectedResources.length > 0 && (
                <div className="flex flex-wrap gap-1.5 p-2 bg-primary/5 rounded-md border">
                  {selectedResources.map((resource) => (
                    <Badge key={resource} variant="secondary" className="h-6 text-xs">
                      {resource}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          )}

          {/* Example Scopes */}
          <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900 rounded-lg p-4">
            <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
              {audienceCopy.exampleTitle}
            </h4>
            <ul className="text-sm text-blue-800 dark:text-blue-200 space-y-1">
              {audienceCopy.examples.map((example) => (
                <li key={example.scope}>
                  <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">
                    {example.scope}
                  </code>{" "}
                  - {example.description}
                </li>
              ))}
            </ul>
          </div>
        </form>
      </div>
    </div>
  );
}
