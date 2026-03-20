import { useState, useEffect, useMemo } from "react";
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
import { toast } from "@/lib/toast";
import { useCreateScopeMutation, useCreateScopesMutation } from "@/app/api/scopesApi";
import { useCreateEndUserScopeMutation } from "@/app/api/enduser/scopesApi";
import { useGetPermissionResourcesQuery } from "@/app/api/permissionsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { resolveTenantId } from "@/utils/workspace";
import { SessionManager } from "@/utils/sessionManager";
import { Badge } from "@/components/ui/badge";
import { Check, Loader2, Database, Asterisk } from "lucide-react";
import { cn } from "@/lib/utils";
import { SearchableSelect, type SearchableSelectOption } from "@/components/ui/searchable-select";

interface CreateScopeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
}

export function CreateScopeModal({ open, onOpenChange, onSuccess }: CreateScopeModalProps) {
  const tenantId = resolveTenantId() || SessionManager.getSession()?.tenant_id || "";
  const { audience } = useRbacAudience();
  const [name, setName] = useState("");
  const [selectedResources, setSelectedResources] = useState<string[]>([]);
  const [isCreating, setIsCreating] = useState(false);

  // NEW API: Use new createScope mutation for name-based scopes with resources
  const [createScope] = useCreateScopeMutation();
  // LEGACY API: Keep for fallback
  const [createScopes] = useCreateScopesMutation();
  // END-USER API
  const [createEndUserScope] = useCreateEndUserScopeMutation();

  // Fetch available resources
  const { data: availableResources = [], isLoading: isLoadingResources } = useGetPermissionResourcesQuery({
    audience,
  });

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setName("");
      setSelectedResources([]);
      setIsCreating(false);
    }
  }, [open]);

  // Special wildcard value
  const WILDCARD_VALUE = "*";

  // Convert resources to SearchableSelectOption format, including wildcard
  const resourceOptions = useMemo<SearchableSelectOption[]>(() => {
    const wildcardOption: SearchableSelectOption = {
      value: WILDCARD_VALUE,
      label: "All Resources (Wildcard)",
      description: "Grants access to all current and future resources",
      icon: <Asterisk className="h-4 w-4 text-primary" />,
    };

    const regularOptions = availableResources.map((resource) => ({
      value: resource,
      label: resource,
    }));

    return [wildcardOption, ...regularOptions];
  }, [availableResources]);

  // Check if wildcard is selected
  const isWildcardSelected = selectedResources.includes(WILDCARD_VALUE);

  // Handle resource selection changes
  const handleResourceChange = (values: string[]) => {
    // If wildcard is selected, only keep wildcard
    if (values.includes(WILDCARD_VALUE)) {
      setSelectedResources([WILDCARD_VALUE]);
    } else {
      setSelectedResources(values);
    }
  };

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      toast.error("Scope name is required");
      return;
    }

    if (selectedResources.length === 0) {
      toast.error("At least one resource is required");
      return;
    }

    setIsCreating(true);

    try {
      // If wildcard is selected, send an empty array
      const resourcesToSend = isWildcardSelected ? [] : selectedResources;

      // Use appropriate API based on audience
      if (audience === 'admin') {
        await createScope({
          scope_name: name.trim(),
          resources: resourcesToSend,
        }).unwrap();
      } else {
        await createEndUserScope({
          scope_name: name.trim(),
          resources: resourcesToSend,
          description: undefined,
        }).unwrap();
      }

      const resourceMessage = isWildcardSelected
        ? "all resources (wildcard)"
        : `${selectedResources.length} resource(s)`;
      toast.success(`Scope "${name}" created successfully with ${resourceMessage}`);
      onSuccess?.();
      onOpenChange(false);
    } catch (error: any) {
      console.error("Error creating scope:", error);
      toast.error(error?.data?.message || "Failed to create scope. Please try again.");
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[550px]">
        <DialogHeader>
          <DialogTitle className="text-xl font-semibold">Create Scope</DialogTitle>
          <DialogDescription>
            Define a scope that groups resources for access control
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5 mt-4">
          {/* Scope Name */}
          <div className="space-y-2">
            <Label htmlFor="name" className="text-sm font-medium">
              Scope Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id="name"
              placeholder="e.g., api_access, admin_panel"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="h-10"
              required
            />
          </div>

          {/* Resources Selection */}
          <div className="space-y-3">
            <Label className="flex items-center gap-2 text-sm font-medium">
              <Database className="h-4 w-4 text-primary" />
              Resources <span className="text-destructive">*</span>
            </Label>

            <SearchableSelect
              multiple={!isWildcardSelected}
              options={resourceOptions}
              value={isWildcardSelected ? WILDCARD_VALUE : selectedResources}
              onChange={(values) => {
                // Handle both single and multiple select
                const newValues = Array.isArray(values) ? values : [values].filter(Boolean) as string[];
                handleResourceChange(newValues);
              }}
              placeholder={isLoadingResources ? "Loading resources..." : "Select resources..."}
              searchPlaceholder="Search resources..."
              emptyText="No resources found"
              disabled={isLoadingResources}
              showSelectAll={!isWildcardSelected}
              maxBadges={3}
              className="h-11"
            />

            {/* Selected Resources Preview */}
            {selectedResources.length > 0 && (
              <div className="flex flex-wrap gap-1.5 p-2 bg-primary/5 rounded-md border border-primary/20">
                {isWildcardSelected ? (
                  <Badge variant="secondary" className="h-6 text-xs gap-1 px-2 bg-black/5 dark:bg-white/10 border-0">
                    <Asterisk className="h-3 w-3" />
                    All Resources
                  </Badge>
                ) : (
                  selectedResources.map((resource) => (
                    <Badge key={resource} variant="secondary" className="h-6 text-xs px-2 bg-black/5 dark:bg-white/10 border-0">
                      {resource}
                    </Badge>
                  ))
                )}
              </div>
            )}
          </div>

          <DialogFooter className="gap-2 sm:gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isCreating}
              className="flex-1 sm:flex-none"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!name.trim() || selectedResources.length === 0 || isCreating}
              className="flex-1 sm:flex-none"
            >
              {isCreating ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creating...
                </>
              ) : (
                "Create Scope"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
