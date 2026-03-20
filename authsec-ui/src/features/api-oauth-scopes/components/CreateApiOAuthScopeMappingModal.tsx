import { useState, useEffect, useMemo } from "react";
import { X, Plus, Shield } from "lucide-react";
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
import { useCreateAdminApiOAuthScopeMutation } from "@/app/api/admin/apiOAuthScopesApi";
import { useCreateEndUserApiOAuthScopeMutation } from "@/app/api/enduser/apiOAuthScopesApi";
import { useGetPermissionsQuery } from "@/app/api/permissionsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { resolveTenantId } from "@/utils/workspace";
import { CreatePermissionModal } from "../../permissions/components/CreatePermissionModal";
import {
  SearchableSelect,
  type SearchableSelectOption,
} from "@/components/ui/searchable-select";

interface CreateApiOAuthScopeMappingModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function CreateApiOAuthScopeMappingModal({
  open,
  onOpenChange,
}: CreateApiOAuthScopeMappingModalProps) {
  const tenantId = resolveTenantId();
  const { audience, isAdmin } = useRbacAudience();
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [selectedPermissions, setSelectedPermissions] = useState<Set<string>>(
    new Set()
  );
  const [isCreating, setIsCreating] = useState(false);
  const [createPermissionModalOpen, setCreatePermissionModalOpen] =
    useState(false);

  // API hooks
  const {
    data: allPermissions = [],
    isLoading: permissionsLoading,
    refetch: refetchPermissions,
  } = useGetPermissionsQuery({
    tenant_id: tenantId || "",
    audience,
  });
  const [createAdminScope] = useCreateAdminApiOAuthScopeMutation();
  const [createEndUserScope] = useCreateEndUserApiOAuthScopeMutation();

  // Audience-aware copy
  const copy = useMemo(
    () => ({
      title: isAdmin
        ? "Add API/OAuth Scope Mapping"
        : "Add API/OAuth Scope Mapping",
      description: isAdmin
        ? "Create a new API/OAuth scope mapping with specific permissions."
        : "Create a new API/OAuth scope mapping with specific permissions.",
      namePlaceholder: isAdmin
        ? "e.g., project:read, api:admin"
        : "e.g., user:read, profile:edit",
      descriptionPlaceholder: isAdmin
        ? "Describe this API scope..."
        : "Describe this API scope...",
    }),
    [isAdmin]
  );

  // Reset form when modal closes
  useEffect(() => {
    if (!open) {
      setName("");
      setDescription("");
      setSelectedPermissions(new Set());
      setIsCreating(false);
    }
  }, [open]);

  // Convert permissions to SearchableSelectOption format
  const permissionOptions = useMemo<SearchableSelectOption[]>(() => {
    return allPermissions.map((p) => ({
      value: p.id,
      label: p.full_permission_string || `${p.resource}:${p.action}`,
      description: p.description,
    }));
  }, [allPermissions]);

  // Get selected permission details for display
  const selectedPermissionDetails = useMemo(() => {
    return allPermissions.filter((p) => selectedPermissions.has(p.id));
  }, [allPermissions, selectedPermissions]);

  // Handle form submission
  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!name.trim()) {
      toast.error("Scope name is required");
      return;
    }

    if (selectedPermissions.size === 0) {
      toast.error("Select at least one permission");
      return;
    }

    setIsCreating(true);

    try {
      if (isAdmin) {
        await createAdminScope({
          name: name.trim(),
          description: description.trim() || "",
          mapped_permission_ids: Array.from(selectedPermissions),
        }).unwrap();
      } else {
        await createEndUserScope({
          name: name.trim(),
          description: description.trim() || "",
          mapped_permission_ids: Array.from(selectedPermissions),
        }).unwrap();
      }

      toast.success(`API/OAuth scope "${name}" created successfully`);
      onOpenChange(false);
    } catch (error: any) {
      console.error("Error creating API/OAuth scope:", error);
      const apiMessage =
        error?.data?.message ||
        error?.data?.error ||
        error?.error ||
        error?.message;
      toast.error(
        apiMessage || "Failed to create API/OAuth scope. Please try again."
      );
    } finally {
      setIsCreating(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[700px]">
        <DialogHeader className="space-y-2 pb-4 text-center flex flex-col items-center">
          <DialogTitle className="text-3xl font-bold tracking-tight text-center w-full">
            {copy.title}
          </DialogTitle>
          <DialogDescription className="text-sm text-foreground text-center">
            {copy.description}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6 pt-2">
          {/* Scope Name */}
          <div className="space-y-3">
            <Label htmlFor="name" className="text-sm font-semibold">
              Name <span className="text-destructive">*</span>
            </Label>
            <Input
              id="name"
              placeholder={copy.namePlaceholder}
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="h-11"
              required
            />
          </div>

          {/* Description */}
          <div className="space-y-3">
            <Label htmlFor="description" className="text-sm font-semibold">
              Description{" "}
              <span className="text-foreground text-xs font-normal">
                (optional)
              </span>
            </Label>
            <Input
              id="description"
              placeholder={copy.descriptionPlaceholder}
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="h-11"
            />
          </div>

          {/* Permissions Selection */}
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <Label className="flex items-center gap-2 text-sm font-semibold">
                <Shield className="h-4 w-4 text-primary" />
                Permissions <span className="text-destructive">*</span>
              </Label>
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-8 gap-1"
                onClick={() => setCreatePermissionModalOpen(true)}
              >
                <Plus className="h-4 w-4" />
                New Permission
              </Button>
            </div>

            <SearchableSelect
              multiple
              options={permissionOptions}
              value={Array.from(selectedPermissions)}
              onChange={(ids) => setSelectedPermissions(new Set(ids))}
              placeholder={
                permissionsLoading
                  ? "Loading permissions..."
                  : "Select permissions..."
              }
              searchPlaceholder="Search permissions..."
              emptyText="No permissions found"
              disabled={permissionsLoading}
              showSelectAll
              maxBadges={3}
              className="h-11"
            />
          </div>

          {/* Selected Permissions Preview */}
          {selectedPermissionDetails.length > 0 && (
            <div className="space-y-3 rounded-lg border border-primary/20 bg-primary/5 p-4">
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
                    variant="secondary"
                    className="font-mono text-xs px-3 py-1 bg-black/5 dark:bg-white/10 border-0"
                  >
                    {permission.full_permission_string ||
                      `${permission.resource}:${permission.action}`}
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
              disabled={isCreating}
              className="h-11 flex-1 sm:flex-initial"
            >
              Cancel
            </Button>
            <Button
              type="submit"
              disabled={!name.trim() || isCreating}
              className="h-11 flex-1 sm:flex-initial"
            >
              {isCreating ? "Creating..." : "Add Mapping"}
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
