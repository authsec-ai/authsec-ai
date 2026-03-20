import React, { useState, useEffect } from "react";
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
import { Badge } from "@/components/ui/badge";
import { toast } from "@/lib/toast";
import { useUpdateScopeResourcesMutation, useGetScopeMappingsQuery } from "@/app/api/scopesApi";
import { useUpdateEndUserScopeMutation, useGetEndUserScopeMappingsQuery } from "@/app/api/enduser/scopesApi";
import { useGetPermissionResourcesQuery } from "@/app/api/permissionsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { SessionManager } from "@/utils/sessionManager";
import { X, Plus, Loader2, Search } from "lucide-react";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { ScrollArea } from "@/components/ui/scroll-area";
import type { Scope } from "@/app/api/scopesApi";

interface EditScopeModalProps {
  scope: Scope | { name: string; id?: string }; // Support both legacy Scope and new format
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function EditScopeModal({ scope, open, onOpenChange }: EditScopeModalProps) {
  const { audience } = useRbacAudience();
  const tenantId = SessionManager.getSession()?.tenant_id || "";
  const [selectedResources, setSelectedResources] = useState<string[]>([]);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [resourceSelectorOpen, setResourceSelectorOpen] = useState(false);

  // NEW API: Use new updateScopeResources mutation for PUT endpoint
  const [updateScopeResources] = useUpdateScopeResourcesMutation();
  const [updateEndUserScope] = useUpdateEndUserScopeMutation();

  // Fetch scope mappings to get current resources
  const { data: scopeMappings = [], isLoading: isLoadingAdminMappings } = useGetScopeMappingsQuery(
    undefined,
    { skip: audience === 'endUser' }
  );
  const { data: endUserScopeMappings = [], isLoading: isLoadingEndUserMappings } = useGetEndUserScopeMappingsQuery(
    undefined,
    { skip: audience === 'admin' }
  );

  const isLoadingMappings = audience === 'admin' ? isLoadingAdminMappings : isLoadingEndUserMappings;

  // Fetch available resources
  const { data: availableResources = [], isLoading: isLoadingResources } =
    useGetPermissionResourcesQuery({
      audience,
    });
  const [resourceSearch, setResourceSearch] = useState("");

  // Find current scope's resources from mappings
  useEffect(() => {
    if (audience === 'admin') {
      if (scopeMappings.length > 0 && scope.name) {
        const currentScope = scopeMappings.find((m) => m.scope_name === scope.name);
        if (currentScope) {
          setSelectedResources(currentScope.resources || []);
        }
      }
    } else {
      if (endUserScopeMappings.length > 0 && scope.name) {
        const currentScope = endUserScopeMappings.find((m: any) => m.scope_name === scope.name);
        if (currentScope) {
          setSelectedResources(currentScope.resources || []);
        }
      }
    }
  }, [scopeMappings, endUserScopeMappings, scope.name, audience]);

  // Handle adding a resource
  const handleAddResource = (resource: string) => {
    const trimmed = (resource ?? "").toString().trim();
    if (!trimmed) return;
    setSelectedResources((prev) => (prev.includes(trimmed) ? prev : [...prev, trimmed]));
    setResourceSelectorOpen(false);
    setResourceSearch("");
  };

  // Handle removing a resource
  const handleRemoveResource = (resource: string) => {
    setSelectedResources(selectedResources.filter((r) => r !== resource));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (selectedResources.length === 0) {
      toast.error("At least one resource is required");
      return;
    }

    setIsSubmitting(true);

    try {
      // Use appropriate API based on audience
      if (audience === 'admin') {
        await updateScopeResources({
          scope_name: scope.name,
          resources: selectedResources,
        }).unwrap();
      } else {
        await updateEndUserScope({
          scope_id: scope.id || scope.name,
          resources: selectedResources,
          scope_name: scope.name,
        }).unwrap();
      }

      toast.success(
        `Scope "${scope.name}" updated successfully with ${selectedResources.length} resource(s)`
      );
      onOpenChange(false);
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to update scope");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[550px]">
        <DialogHeader>
          <DialogTitle className="text-2xl">Edit Scope: {scope.name}</DialogTitle>
          <DialogDescription>Update the resources associated with this scope</DialogDescription>
        </DialogHeader>

        {isLoadingMappings ? (
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-8 w-8 animate-spin text-foreground" />
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-6">
            {/* Scope Name (Read-only) */}
            <div className="space-y-2">
              <Label className="text-sm font-semibold">Scope Name</Label>
              <div className="px-3 py-2 bg-muted rounded-md border font-mono text-sm">
                {scope.name}
              </div>
              <p className="text-xs text-foreground">
                Scope name cannot be changed. Create a new scope if needed.
              </p>
            </div>

            {/* Resources Selection */}
            <div className="space-y-3">
              <Label className="text-sm font-semibold">
                Associated Resources <span className="text-destructive">*</span>
              </Label>
              <div className="space-y-2">
                {/* Selected Resources */}
                {selectedResources.length > 0 ? (
                  <div className="flex flex-wrap gap-2 p-3 bg-muted/50 rounded-md border min-h-[60px]">
                    {selectedResources.map((resource) => (
                      <Badge key={resource} variant="secondary" className="pl-2 pr-1 py-1 text-sm">
                        {resource}
                        <button
                          type="button"
                          onClick={() => handleRemoveResource(resource)}
                          className="ml-1 hover:bg-destructive/20 rounded-sm p-0.5"
                          disabled={isSubmitting}
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    ))}
                  </div>
                ) : (
                  <div className="p-4 bg-muted/30 rounded-md border border-dashed text-center text-sm text-foreground">
                    No resources selected. Add at least one resource.
                  </div>
                )}

                {/* Add Resource Button */}
                <Popover open={resourceSelectorOpen} onOpenChange={setResourceSelectorOpen}>
                  <PopoverTrigger asChild>
                    <Button
                      type="button"
                      variant="outline"
                      className="w-full h-11 justify-start text-left font-normal"
                      disabled={isLoadingResources || isSubmitting}
                    >
                      <Plus className="mr-2 h-4 w-4" />
                      {isLoadingResources
                        ? "Loading resources..."
                        : selectedResources.length === 0
                        ? "Select resources for this scope"
                        : "Add more resources"}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-[450px] p-3" align="start">
                    <div className="flex items-center gap-2 mb-3 px-2 py-1.5 rounded-md bg-muted/60">
                      <Search className="h-4 w-4 text-foreground" />
                      <Input
                        autoFocus
                        placeholder="Search resources..."
                        value={resourceSearch}
                        onChange={(e) => setResourceSearch(e.target.value)}
                        className="h-8 border-0 bg-transparent shadow-none px-0 text-sm focus-visible:ring-0"
                      />
                    </div>
                    <ScrollArea className="max-h-72">
                      <div className="space-y-1">
                        {availableResources
                          .filter((resource) => !selectedResources.includes(resource))
                          .filter((resource) =>
                            resource.toLowerCase().includes(resourceSearch.toLowerCase())
                          )
                          .map((resource) => (
                            <button
                              key={resource}
                              type="button"
                              onClick={() => handleAddResource(resource)}
                              className="w-full text-left px-3 py-2 text-sm rounded-md hover:bg-muted transition-colors"
                            >
                              {resource}
                            </button>
                          ))}
                        {availableResources
                          .filter((resource) => !selectedResources.includes(resource))
                          .filter((resource) =>
                            resource.toLowerCase().includes(resourceSearch.toLowerCase())
                          ).length === 0 && (
                          <div className="px-3 py-2 text-sm text-foreground">
                            No resources found.
                          </div>
                        )}
                      </div>
                    </ScrollArea>
                  </PopoverContent>
                </Popover>
              </div>
              <p className="text-xs text-foreground">
                Manage which resources this scope grants access to
              </p>
            </div>

            <DialogFooter className="gap-3 sm:gap-3">
              <Button
                type="button"
                variant="outline"
                onClick={() => onOpenChange(false)}
                disabled={isSubmitting}
                className="h-11 flex-1 sm:flex-initial"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                disabled={selectedResources.length === 0 || isSubmitting}
                className="h-11 flex-1 sm:flex-initial"
              >
                {isSubmitting ? "Saving..." : "Save Changes"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
