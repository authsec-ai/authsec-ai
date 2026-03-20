import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { FormInput, FormTextarea } from "@form";
import { ArrowLeft, Database, Loader2 } from "lucide-react";
import { toast } from "@/lib/toast";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { resolveTenantId } from "@/utils/workspace";
import {
  useCreateAdminResourceMutation,
  useGetAdminResourceQuery,
  useUpdateAdminResourceMutation,
} from "@/app/api/admin/resourcesApi";
import {
  useCreateEndUserResourceMutation,
  useGetEndUserResourceQuery,
  useUpdateEndUserResourceMutation,
} from "@/app/api/enduser/resourcesApi";

/**
 * Add/Edit Resource Page - Minimal resource creation and editing flow
 *
 * Features:
 * - Simple form with name and description fields
 * - Supports both admin and end-user resource creation/editing
 * - Audience-aware (switches between admin/end-user APIs)
 * - Edit mode: Loads existing resource data and updates on save
 */
export function AddResourcePage() {
  const navigate = useContextualNavigate();
  const { isAdmin } = useRbacAudience();
  const { id: resourceId } = useParams<{ id: string }>();
  const tenantId = resolveTenantId();

  const isEditMode = Boolean(resourceId);

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  // CREATE mutations
  const [createAdminResource] = useCreateAdminResourceMutation();
  const [createEndUserResource] = useCreateEndUserResourceMutation();

  // UPDATE mutations
  const [updateAdminResource] = useUpdateAdminResourceMutation();
  const [updateEndUserResource] = useUpdateEndUserResourceMutation();

  // GET queries for edit mode
  const { data: adminResource, isLoading: adminResourceLoading } =
    useGetAdminResourceQuery(resourceId!, {
      skip: !isEditMode || !isAdmin || !resourceId,
    });

  const { data: endUserResource, isLoading: endUserResourceLoading } =
    useGetEndUserResourceQuery(
      { tenant_id: tenantId!, resource_id: resourceId! },
      { skip: !isEditMode || isAdmin || !tenantId || !resourceId }
    );

  // Populate form when resource data loads in edit mode
  useEffect(() => {
    if (isEditMode) {
      const resource = isAdmin ? adminResource : endUserResource;
      if (resource) {
        setName(resource.name);
        setDescription(resource.description || "");
      }
    }
  }, [isEditMode, isAdmin, adminResource, endUserResource]);

  const handleSubmit = async (e?: React.FormEvent) => {
    e?.preventDefault();

    // Validation
    if (!name.trim()) {
      toast.error("Resource name is required");
      return;
    }

    if (!isAdmin && !tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      return;
    }

    setIsLoading(true);
    try {
      if (isEditMode) {
        // UPDATE existing resource
        if (isAdmin) {
          await updateAdminResource({
            id: resourceId!,
            data: {
              name: name.trim(),
              description: description.trim() || undefined,
            },
          }).unwrap();
        } else {
          await updateEndUserResource({
            tenant_id: tenantId!,
            id: resourceId!,
            data: {
              name: name.trim(),
              description: description.trim() || undefined,
            },
          }).unwrap();
        }
        toast.success(`Resource "${name}" updated successfully!`);
      } else {
        // CREATE new resource
        if (isAdmin) {
          await createAdminResource({
            name: name.trim(),
            description: description.trim() || undefined,
          }).unwrap();
        } else {
          await createEndUserResource({
            tenant_id: tenantId!,
            data: {
              name: name.trim(),
              description: description.trim() || undefined,
            },
          }).unwrap();
        }
        toast.success(`Resource "${name}" created successfully!`);
      }

      navigate("/resources");
    } catch (error: any) {
      toast.error(
        error?.data?.message ||
          `Failed to ${isEditMode ? "update" : "create"} resource`
      );
    } finally {
      setIsLoading(false);
    }
  };

  const handleCancel = () => {
    navigate("/resources");
  };

  const isFormValid = () => {
    return name.trim().length > 0;
  };

  const audienceLabel = isAdmin ? "Admin" : "End-user";
  const resourceLoading =
    isEditMode &&
    ((isAdmin && adminResourceLoading) || (!isAdmin && endUserResourceLoading));

  // Show loading state when fetching resource data in edit mode
  if (isEditMode && resourceLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950 flex items-center justify-center">
        <div className="flex items-center gap-2">
          <Loader2 className="h-5 w-5 animate-spin" />
          <span>Loading resource...</span>
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
                  {isEditMode ? "Edit" : "Create"} {audienceLabel} Resource
                </h1>
                <p className="text-sm text-foreground mt-1">
                  {isEditMode
                    ? "Update the resource name and description."
                    : "Define a new resource with a name and optional description."}
                </p>
              </div>
            </div>
            <Button
              type="submit"
              form="create-resource-form"
              disabled={!isFormValid() || isLoading}
              className="min-w-[140px]"
            >
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {isEditMode ? "Updating..." : "Creating..."}
                </>
              ) : isEditMode ? (
                "Update"
              ) : (
                "Create"
              )}
            </Button>
          </div>
        </header>

        {/* Main Form */}
        <form
          id="create-resource-form"
          onSubmit={handleSubmit}
          className="space-y-6"
        >
          <FormInput
            id="resource-name"
            label="Resource Name"
            required
            placeholder="e.g., admin_users, customer_api, billing_service"
            value={name}
            onChange={(e) => setName(e.target.value)}
            disabled={isLoading}
            className="font-mono"
            helperText="A unique identifier for this resource"
          />

          <FormTextarea
            id="description"
            label="Description"
            placeholder="Describe what this resource is used for..."
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            disabled={isLoading}
            rows={4}
            helperText="Optional description to help others understand this resource"
          />

          {/* Example Resources */}
          <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-900 rounded-lg p-4">
            <h4 className="text-sm font-medium text-blue-900 dark:text-blue-100 mb-2">
              Example Resources
            </h4>
            <ul className="text-sm text-blue-800 dark:text-blue-200 space-y-1">
              <li>
                <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">
                  admin_users
                </code>{" "}
                - User management system
              </li>
              <li>
                <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">
                  customer_api
                </code>{" "}
                - External API endpoints
              </li>
              <li>
                <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">
                  billing_service
                </code>{" "}
                - Payment processing
              </li>
              <li>
                <code className="bg-blue-100 dark:bg-blue-900 px-1 rounded">
                  audit_logs
                </code>{" "}
                - Activity tracking
              </li>
            </ul>
          </div>
        </form>
      </div>
    </div>
  );
}

export default AddResourcePage;
