import { useState, useEffect, useMemo } from "react";
import { Database, Shield, Plus } from "lucide-react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Checkbox } from "@/components/ui/checkbox";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/lib/toast";
import { SearchableSelect, type SearchableSelectOption } from "@/components/ui/searchable-select";
import {
  useCreatePermissionMutation,
  useGetPermissionsQuery,
  useGetPermissionResourcesQuery,
} from "@/app/api/permissionsApi";
import { resolveTenantId } from "@/utils/workspace";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";

interface CreatePermissionModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

// Standard permission actions
const STANDARD_ACTIONS = [
  "read",
  "write",
  "create",
  "update",
  "delete",
  "list",
  "manage",
  "admin",
  "execute",
  "export",
  "import",
];

export function CreatePermissionModal({ open, onOpenChange, onSuccess }: CreatePermissionModalProps) {
  const tenantId = resolveTenantId();
  const { audience } = useRbacAudience();
  const [selectedResource, setSelectedResource] = useState<string>("");
  const [selectedActions, setSelectedActions] = useState<Set<string>>(new Set());
  const [customAction, setCustomAction] = useState("");
  const [description, setDescription] = useState("");
  const [isCreating, setIsCreating] = useState(false);
  const [customResources, setCustomResources] = useState<string[]>([]);
  const [showAddResourceModal, setShowAddResourceModal] = useState(false);
  const [newResourceName, setNewResourceName] = useState("");

  // API hooks - Fetch resources and permissions
  const { data: resources = [], isLoading: resourcesLoading } = useGetPermissionResourcesQuery({
    audience,
  }, {
    skip: !open,
  });
  const { data: existingPermissions = [] } = useGetPermissionsQuery({
    tenant_id: tenantId || "",
    audience,
  }, {
    skip: !open,
  });
  const [createPermission] = useCreatePermissionMutation();

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setSelectedResource("");
      setSelectedActions(new Set());
      setCustomAction("");
      setDescription("");
      setIsCreating(false);
      setCustomResources([]);
      setShowAddResourceModal(false);
      setNewResourceName("");
    }
  }, [open]);

  // Special value for creating new resource
  const CREATE_NEW_RESOURCE = "__CREATE_NEW__";

  // Convert resources to SearchableSelectOption format
  const resourceOptions = useMemo<SearchableSelectOption[]>(() => {
    const createNewOption: SearchableSelectOption = {
      value: CREATE_NEW_RESOURCE,
      label: "Create New Resource",
      description: "Add a custom resource to the list",
      icon: <Plus className="h-4 w-4 text-primary" />,
    };

    const apiResources = resources.map((resource) => ({
      value: resource,
      label: resource,
    }));

    const customResourceOptions = customResources.map((resource) => ({
      value: resource,
      label: resource,
      description: "Custom resource",
    }));

    return [createNewOption, ...customResourceOptions, ...apiResources];
  }, [resources, customResources]);

  // Toggle action selection
  const toggleAction = (action: string) => {
    const newActions = new Set(selectedActions);
    if (newActions.has(action)) {
      newActions.delete(action);
    } else {
      newActions.add(action);
    }
    setSelectedActions(newActions);
  };

  // Add custom action
  const addCustomAction = () => {
    const trimmed = customAction.trim().toLowerCase();
    if (trimmed && !selectedActions.has(trimmed)) {
      const newActions = new Set(selectedActions);
      newActions.add(trimmed);
      setSelectedActions(newActions);
      setCustomAction("");
    }
  };

  // Handle Enter key in custom action input
  const handleCustomActionKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      addCustomAction();
    }
  };

  // Handle adding custom resource
  const handleAddCustomResource = () => {
    const trimmed = newResourceName.trim().toLowerCase();
    if (!trimmed) {
      toast.error("Resource name cannot be empty");
      return;
    }

    // Check if resource already exists
    if (resources.includes(trimmed) || customResources.includes(trimmed)) {
      toast.error("This resource already exists");
      return;
    }

    // Add to custom resources and select it
    setCustomResources((prev) => [...prev, trimmed]);
    setSelectedResource(trimmed);
    setNewResourceName("");
    setShowAddResourceModal(false);
    toast.success(`Resource "${trimmed}" added`);
  };

  // Handle resource selection
  const handleResourceChange = (val: string | undefined) => {
    if (val === CREATE_NEW_RESOURCE) {
      setShowAddResourceModal(true);
    } else {
      setSelectedResource(val ?? "");
    }
  };

  // Generate permission strings
  const permissionStrings = selectedResource
    ? Array.from(selectedActions).map((action) => `${selectedResource}:${action}`)
    : [];

  // Check for duplicates
  const duplicatePermissions = permissionStrings.filter((permString) =>
    existingPermissions.some((p) => p.full_permission_string === permString)
  );

  const hasDuplicates = duplicatePermissions.length > 0;

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!selectedResource) {
      toast.error("Please select a resource");
      return;
    }

    if (selectedActions.size === 0) {
      toast.error("Please select at least one action");
      return;
    }

    if (hasDuplicates) {
      toast.error(
        `The following permissions already exist: ${duplicatePermissions.join(", ")}`
      );
      return;
    }

    setIsCreating(true);

    try {
      // Create multiple permissions (one for each action)
      const creationPromises = Array.from(selectedActions).map((action) =>
        createPermission({
          resource: selectedResource,
          action,
          description: description || undefined,
          audience,
        }).unwrap()
      );

      const results = await Promise.allSettled(creationPromises);

      // Check results
      const successCount = results.filter((r) => r.status === "fulfilled").length;
      const failureCount = results.filter((r) => r.status === "rejected").length;

      if (failureCount === 0) {
        toast.success(
          `Successfully created ${successCount} permission${successCount > 1 ? "s" : ""}`
        );
        onSuccess?.();
        onOpenChange(false);
      } else if (successCount > 0) {
        toast.error(
          `Created ${successCount} permission(s), but ${failureCount} failed. Please check and try again.`
        );
      } else {
        toast.error("Failed to create permissions. Please try again.");
      }
    } catch (error) {
      console.error("Error creating permissions:", error);
      toast.error("An unexpected error occurred. Please try again.");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[650px]">
        <DialogHeader className="space-y-2 pb-4 text-center flex flex-col items-center">
          <DialogTitle className="text-3xl font-bold tracking-tight text-center w-full">
            Create Permissions
          </DialogTitle>
          <DialogDescription className="text-sm text-foreground text-center">
            Create atomic permissions by selecting a resource and one or more actions. Multiple
            permissions will be created if you select multiple actions.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 pt-2">
          {/* Resource Selection */}
          <div className="space-y-3">
            <Label className="flex items-center gap-2 text-sm font-semibold">
              <Database className="h-4 w-4 text-primary" />
              Resource
            </Label>
            <SearchableSelect
              options={resourceOptions}
              value={selectedResource}
              onChange={handleResourceChange}
              placeholder={resourcesLoading ? "Loading..." : "Select a resource..."}
              searchPlaceholder="Search resources..."
              emptyText="No resources found"
              disabled={resourcesLoading}
              className="h-11"
            />
          </div>

          {/* Actions Multi-Select */}
          <div className="space-y-3">
            <Label className="text-sm font-semibold">Actions</Label>
            <div className="space-y-3 p-4 bg-black/[0.02] dark:bg-white/[0.02] rounded-lg border border-black/10 dark:border-white/10">
              {/* Standard Actions Grid */}
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
                {STANDARD_ACTIONS.map((action) => (
                  <div key={action} className="flex items-center space-x-2">
                    <Checkbox
                      id={`action-${action}`}
                      checked={selectedActions.has(action)}
                      onCheckedChange={() => toggleAction(action)}
                    />
                    <label
                      htmlFor={`action-${action}`}
                      className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70 cursor-pointer capitalize"
                    >
                      {action}
                    </label>
                  </div>
                ))}
              </div>

              {/* Custom Action Input */}
              <div className="pt-3 border-t border-black/10 dark:border-white/10">
                <Label className="text-xs text-foreground mb-2 block">
                  Or add a custom action
                </Label>
                <div className="flex gap-2">
                  <Input
                    placeholder="Type custom action..."
                    value={customAction}
                    onChange={(e) => setCustomAction(e.target.value)}
                    onKeyDown={handleCustomActionKeyDown}
                    className="h-9 flex-1"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={addCustomAction}
                    disabled={!customAction.trim()}
                    className="h-9"
                  >
                    <Plus className="h-4 w-4 mr-1" />
                    Add
                  </Button>
                </div>
              </div>

              {/* Show custom actions that were added */}
              {Array.from(selectedActions).filter(a => !STANDARD_ACTIONS.includes(a)).length > 0 && (
                <div className="pt-3 border-t border-black/10 dark:border-white/10">
                  <Label className="text-xs text-foreground mb-2 block">
                    Custom actions
                  </Label>
                  <div className="flex flex-wrap gap-2">
                    {Array.from(selectedActions)
                      .filter(a => !STANDARD_ACTIONS.includes(a))
                      .map((action) => (
                        <Badge
                          key={action}
                          variant="secondary"
                          className="pl-2 pr-1 py-1 flex items-center gap-1"
                        >
                          {action}
                          <button
                            type="button"
                            onClick={() => toggleAction(action)}
                            className="ml-1 hover:bg-black/10 dark:hover:bg-white/10 rounded-sm p-0.5"
                          >
                            ✕
                          </button>
                        </Badge>
                      ))}
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Description (Optional) */}
          <div className="space-y-3">
            <Label htmlFor="description" className="text-sm font-semibold">
              Description <span className="text-foreground text-xs font-normal">(optional)</span>
            </Label>
            <Input
              id="description"
              placeholder="Brief description of this permission..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="h-11"
            />
          </div>

          {/* Permission Preview */}
          {permissionStrings.length > 0 && (
            <div className="space-y-3 rounded-lg border-2 border-primary/20 bg-primary/5 p-4">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-semibold flex items-center gap-2">
                  <Shield className="h-4 w-4 text-primary" />
                  Permission Preview
                </Label>
                <span className="text-xs font-medium text-foreground bg-black/5 dark:bg-white/10 px-2 py-1 rounded">
                  {permissionStrings.length} permission{permissionStrings.length > 1 ? "s" : ""}
                </span>
              </div>
              <div className="flex flex-wrap gap-2">
                {permissionStrings.map((permString) => {
                  const isDuplicate = duplicatePermissions.includes(permString);
                  const action = permString.split(':')[1];
                  return (
                    <Badge
                      key={permString}
                      variant={isDuplicate ? "destructive" : "default"}
                      className="font-mono text-xs pl-3 pr-1 py-1 flex items-center gap-1"
                    >
                      {permString}
                      <button
                        type="button"
                        onClick={() => toggleAction(action)}
                        className="ml-1 hover:bg-destructive/20 rounded-sm p-0.5"
                      >
                        ✕
                      </button>
                    </Badge>
                  );
                })}
              </div>
              {hasDuplicates && (
                <p className="text-xs text-destructive font-medium flex items-center gap-1 mt-2 bg-destructive/10 px-3 py-2 rounded">
                  <span>⚠️</span>
                  Some permissions already exist and will not be created.
                </p>
              )}
            </div>
          )}

          <DialogFooter className="gap-3 sm:gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isCreating}
              className="h-11 flex-1 sm:flex-initial"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={
                !selectedResource ||
                selectedActions.size === 0 ||
                hasDuplicates ||
                isCreating ||
                resourcesLoading
              }
              className="h-11 flex-1 sm:flex-initial"
            >
              {isCreating
                ? "Creating..."
                : permissionStrings.length > 0
                ? `Create ${permissionStrings.length} Permission${permissionStrings.length > 1 ? "s" : ""}`
                : "Create Permission"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>

      {/* Add Custom Resource Modal */}
      <Dialog open={showAddResourceModal} onOpenChange={setShowAddResourceModal}>
        <DialogContent className="sm:max-w-[450px]">
          <DialogHeader>
            <DialogTitle className="text-xl font-semibold flex items-center gap-2">
              <Plus className="h-5 w-5 text-primary" />
              Create Custom Resource
            </DialogTitle>
            <DialogDescription>
              Enter a custom resource name to add to the list
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="newResource" className="text-sm font-medium">
                Resource Name <span className="text-destructive">*</span>
              </Label>
              <Input
                id="newResource"
                placeholder="e.g., projects, documents, tasks"
                value={newResourceName}
                onChange={(e) => setNewResourceName(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") {
                    e.preventDefault();
                    handleAddCustomResource();
                  }
                }}
                className="h-11"
                autoFocus
              />
              <p className="text-xs text-foreground/60">
                Resource names will be converted to lowercase
              </p>
            </div>
          </div>

          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => {
                setShowAddResourceModal(false);
                setNewResourceName("");
              }}
              className="flex-1 sm:flex-none"
            >
              Cancel
            </Button>
            <Button
              type="button"
              onClick={handleAddCustomResource}
              disabled={!newResourceName.trim()}
              className="flex-1 sm:flex-none"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Resource
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Dialog>
  );
}
