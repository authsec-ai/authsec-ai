import { useState, useEffect, useMemo } from "react";
import { X, Plus } from "lucide-react";
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
import {
  useUpdateAdminApiOAuthScopeMutation,
  useGetAdminApiOAuthScopeQuery,
} from "@/app/api/admin/apiOAuthScopesApi";
import {
  useUpdateEndUserApiOAuthScopeMutation,
  useGetEndUserApiOAuthScopeQuery,
} from "@/app/api/enduser/apiOAuthScopesApi";
import {
  useGetPermissionsQuery,
} from "@/app/api/permissionsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { resolveTenantId } from "@/utils/workspace";
import { CreatePermissionModal } from "../../permissions/components/CreatePermissionModal";
import type { ApiOAuthScope } from "../types";

interface EditApiOAuthScopeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  scope: ApiOAuthScope | null;
}

export function EditApiOAuthScopeModal({
  open,
  onOpenChange,
  scope,
}: EditApiOAuthScopeModalProps) {
  const tenantId = resolveTenantId();
  const { audience, isAdmin } = useRbacAudience();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(new Set());
  const [searchQuery, setSearchQuery] = useState("");
  const [isUpdating, setIsUpdating] = useState(false);
  const [createPermissionModalOpen, setCreatePermissionModalOpen] = useState(false);

  // API hooks
  const { data: allPermissions = [], isLoading: permissionsLoading, refetch: refetchPermissions } = useGetPermissionsQuery({
    tenant_id: tenantId || "",
    audience,
  });

  // Fetch scope details to get permission_ids
  const { data: adminScopeDetails } = useGetAdminApiOAuthScopeQuery(
    scope?.id || "",
    { skip: !scope || !isAdmin }
  );
  const { data: endUserScopeDetails } = useGetEndUserApiOAuthScopeQuery(
    scope?.id || "",
    { skip: !scope || isAdmin }
  );

  const scopeDetails = isAdmin ? adminScopeDetails : endUserScopeDetails;

  const [updateAdminScope] = useUpdateAdminApiOAuthScopeMutation();
  const [updateEndUserScope] = useUpdateEndUserApiOAuthScopeMutation();

  // Initialize form with scope data and permissions
  useEffect(() => {
    if (scope) {
      setName(scope.name);
      setDescription(scope.description || "");
    }
    if (scopeDetails?.permission_ids) {
      setSelectedPermissions(new Set(scopeDetails.permission_ids));
    }
  }, [scope, scopeDetails]);

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setName("");
      setDescription("");
      setSelectedPermissions(new Set());
      setSearchQuery("");
      setIsUpdating(false);
    }
  }, [open]);

  // Filter permissions based on search
  const filteredPermissions = useMemo(() => {
    if (!searchQuery) return allPermissions;
    const query = searchQuery.toLowerCase();
    return allPermissions.filter(
      (p) =>
        p.full_permission_string?.toLowerCase().includes(query) ||
        p.action?.toLowerCase().includes(query) ||
        p.resource?.toLowerCase().includes(query)
    );
  }, [allPermissions, searchQuery]);

  // Toggle permission selection
  const togglePermission = (permissionId: string) => {
    const newSelected = new Set(selectedPermissions);
    if (newSelected.has(permissionId)) {
      newSelected.delete(permissionId);
    } else {
      newSelected.add(permissionId);
    }
    setSelectedPermissions(newSelected);
  };

  // Remove permission
  const removePermission = (permissionId: string) => {
    const newSelected = new Set(selectedPermissions);
    newSelected.delete(permissionId);
    setSelectedPermissions(newSelected);
  };

  // Get selected permission details for display
  const selectedPermissionDetails = useMemo(() => {
    return allPermissions.filter((p) => selectedPermissions.has(p.id));
  }, [allPermissions, selectedPermissions]);

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!scope) return;

    if (!name.trim()) {
      toast.error("Scope name is required");
      return;
    }

    if (selectedPermissions.size === 0) {
      toast.error("Select at least one permission");
      return;
    }

    setIsUpdating(true);

    try {
      if (isAdmin) {
        await updateAdminScope({
          scope_id: scope.id,
          name: name.trim(),
          description: description.trim() || "",
          mapped_permission_ids: Array.from(selectedPermissions),
        }).unwrap();
      } else {
        await updateEndUserScope({
          scope_id: scope.id,
          name: name.trim(),
          description: description.trim() || "",
          mapped_permission_ids: Array.from(selectedPermissions),
        }).unwrap();
      }

      toast.success(`API/OAuth scope updated successfully`);
      onOpenChange(false);
    } catch (error: any) {
      console.error("Error updating API/OAuth scope:", error);
      const apiMessage =
        error?.data?.message ||
        error?.data?.error ||
        error?.error ||
        error?.message;
      toast.error(apiMessage || "Failed to update API/OAuth scope. Please try again.");
    } finally {
      setIsUpdating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px] max-h-[90vh] overflow-y-auto">
        <DialogHeader className="space-y-2 pb-4 text-center flex flex-col items-center">
          <DialogTitle className="text-3xl font-bold tracking-tight text-center w-full">
            Edit Mapping
          </DialogTitle>
          <DialogDescription className="text-sm text-foreground text-center">
            Update the API/OAuth scope mapping details and permissions
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 pt-2">
          {/* Scope Name */}
          <div className="space-y-3">
            <Label htmlFor="edit-name" className="text-sm font-semibold">
              Scope Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id="edit-name"
              placeholder="e.g., project:read, api:admin"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="h-11"
              required
            />
          </div>

          {/* Description */}
          <div className="space-y-3">
            <Label htmlFor="edit-description" className="text-sm font-semibold">
              Description <span className="text-foreground text-xs font-normal">(optional)</span>
            </Label>
            <Input
              id="edit-description"
              placeholder="Describe this API scope..."
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="h-11"
            />
          </div>

          {/* Permissions Selection */}
          <div className="space-y-3">
            <Label className="text-sm font-semibold">
              Permissions <span className="text-destructive">*</span>
            </Label>

            {/* Search Input with Add Button */}
            <div className="flex gap-2">
              <Input
                placeholder="Search permissions..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="h-11 flex-1"
              />
              <Button
                type="button"
                variant="outline"
                size="icon"
                className="h-11 w-11 shrink-0"
                onClick={() => setCreatePermissionModalOpen(true)}
                title="Create new permission"
              >
                <Plus className="h-5 w-5" />
              </Button>
            </div>

            {/* Permissions List */}
            <div className="border rounded-lg max-h-[200px] overflow-y-auto">
              {permissionsLoading ? (
                <div className="p-4 text-center text-sm text-foreground">
                  Loading permissions...
                </div>
              ) : filteredPermissions.length === 0 ? (
                <div className="p-4 text-center text-sm text-foreground">
                  {searchQuery ? "No permissions found" : "No permissions available"}
                </div>
              ) : (
                <div className="p-2">
                  {filteredPermissions.map((permission) => (
                    <div
                      key={permission.id}
                      className={`flex items-center justify-between p-3 rounded-md hover:bg-muted/50 cursor-pointer transition-colors ${
                        selectedPermissions.has(permission.id) ? "bg-primary/5" : ""
                      }`}
                      onClick={() => togglePermission(permission.id)}
                    >
                      <div className="flex flex-col gap-1">
                        <code className="text-xs font-mono font-medium">
                          {permission.full_permission_string}
                        </code>
                        {permission.description && (
                          <span className="text-xs text-foreground">
                            {permission.description}
                          </span>
                        )}
                      </div>
                      {selectedPermissions.has(permission.id) && (
                        <Badge variant="default" className="text-xs">
                          Selected
                        </Badge>
                      )}
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>

          {/* Selected Permissions Preview */}
          {selectedPermissionDetails.length > 0 && (
            <div className="space-y-3 rounded-lg border-2 border-primary/20 bg-primary/5 p-4">
              <div className="flex items-center justify-between">
                <Label className="text-sm font-semibold">
                  Selected Permissions
                </Label>
                <span className="text-xs font-medium text-foreground bg-background px-2 py-1 rounded">
                  {selectedPermissionDetails.length} selected
                </span>
              </div>
              <div className="flex flex-wrap gap-2">
                {selectedPermissionDetails.map((permission) => (
                  <Badge
                    key={permission.id}
                    variant="default"
                    className="font-mono text-xs px-3 py-1 gap-2"
                  >
                    {permission.full_permission_string || `${permission.resource}:${permission.action}`}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation();
                        removePermission(permission.id);
                      }}
                      className="hover:bg-primary/20 rounded-full p-0.5"
                    >
                      <X className="h-3 w-3" />
                    </button>
                  </Badge>
                ))}
              </div>
            </div>
          )}

          <DialogFooter className="gap-3 sm:gap-3">
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isUpdating}
              className="h-11 flex-1 sm:flex-initial"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!name.trim() || isUpdating}
              className="h-11 flex-1 sm:flex-initial"
            >
              {isUpdating ? "Updating..." : "Update Mapping"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>

      {/* Create Permission Modal */}
      <CreatePermissionModal
        open={createPermissionModalOpen}
        onOpenChange={(isOpen) => {
          setCreatePermissionModalOpen(isOpen);
          // Refetch permissions when modal closes after creating a permission
          if (!isOpen) {
            refetchPermissions();
          }
        }}
      />
    </Dialog>
  );
}
