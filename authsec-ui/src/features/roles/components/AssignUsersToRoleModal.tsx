import { useEffect, useMemo, useState } from "react";
import { useGetAdminUsersQuery } from "@/app/api/admin/usersApi";
import { useCreateBindingMutation } from "@/app/api/bindingsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
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
import { Badge } from "@/components/ui/badge";
import { Loader2, UserPlus, Shield, User } from "lucide-react";
import { toast } from "@/lib/toast";
import { resolveTenantId } from "@/utils/workspace";
import { SessionManager } from "@/utils/sessionManager";
import { SearchableSelect, type SearchableSelectOption } from "@/components/ui/searchable-select";

type UserOption = {
  id: string;
  name: string;
  email?: string;
  avatar?: string;
};

type RoleInfo = {
  id: string;
  name: string;
};

interface AssignUsersToRoleModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedRoles: RoleInfo[];
  onSuccess?: () => void;
}

export function AssignUsersToRoleModal({
  open,
  onOpenChange,
  selectedRoles,
  onSuccess,
}: AssignUsersToRoleModalProps) {
  const { isAdmin, audience } = useRbacAudience();
  const sessionData = SessionManager.getSession();
  const tenantId =
    resolveTenantId() ??
    sessionData?.tenant_id ??
    (sessionData as any)?.tenantId ??
    sessionData?.jwtPayload?.tenant_id ??
    "";

  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  const [formError, setFormError] = useState<string | null>(null);

  const {
    data: usersResponse,
    isLoading: isLoadingUsers,
    isFetching: isFetchingUsers,
  } = useGetAdminUsersQuery(
    {
      page: 1,
      limit: 100,
    },
    {
      skip: !open,
    }
  );

  const [createBinding, { isLoading: isCreatingBinding }] = useCreateBindingMutation();

  useEffect(() => {
    if (!open) {
      setSelectedUserIds([]);
      setFormError(null);
    }
  }, [open]);

  const users = useMemo<UserOption[]>(() => {
    const rawUsers =
      Array.isArray((usersResponse as any)?.users) && (usersResponse as any)?.users.length > 0
        ? (usersResponse as any).users
        : Array.isArray(usersResponse)
          ? (usersResponse as any)
          : [];

    const mapped = rawUsers
      .map((user: any, index: number) => {
        const rawId =
          user?.id ||
          user?.user_id ||
          user?.uid ||
          user?.external_id ||
          user?.email ||
          user?.username ||
          `user-${index}`;
        const name =
          user?.name ||
          [user?.first_name, user?.last_name].filter(Boolean).join(" ").trim() ||
          user?.username ||
          user?.email ||
          `User ${index + 1}`;

        if (!rawId || !name) return null;

        return {
          id: String(rawId),
          name: String(name),
          email: user?.email ? String(user.email) : undefined,
          avatar: user?.provider_data?.picture || user?.provider_data?.avatar_url || undefined,
        };
      })
      .filter(Boolean) as UserOption[];

    const unique = new Map<string, UserOption>();
    mapped.forEach((user) => {
      if (!unique.has(user.id)) unique.set(user.id, user);
    });
    return Array.from(unique.values());
  }, [usersResponse]);

  // Convert users to SearchableSelectOption format
  const userOptions = useMemo<SearchableSelectOption[]>(() => {
    return users.map((user) => ({
      value: user.id,
      label: user.name,
      description: user.email,
    }));
  }, [users]);

  const busyLoading = isLoadingUsers || isFetchingUsers;

  // Get selected users for preview
  const selectedUsersData = useMemo(() => {
    return users.filter((u) => selectedUserIds.includes(u.id));
  }, [users, selectedUserIds]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setFormError(null);

    if (selectedUserIds.length === 0) {
      setFormError("Please select at least one user.");
      return;
    }

    if (selectedRoles.length === 0) {
      setFormError("No roles selected.");
      return;
    }

    try {
      // Create bindings for each user-role combination
      const promises: Promise<any>[] = [];
      for (const role of selectedRoles) {
        for (const userId of selectedUserIds) {
          promises.push(
            createBinding({
              user_id: userId,
              role_id: role.id,
              scope: {
                id: "*",
                type: "*",
              },
              audience,
            }).unwrap()
          );
        }
      }

      await Promise.all(promises);

      const roleText = selectedRoles.length === 1 ? `role "${selectedRoles[0].name}"` : `${selectedRoles.length} roles`;
      const userText = selectedUserIds.length === 1 ? "1 user" : `${selectedUserIds.length} users`;
      toast.success(`Successfully assigned ${userText} to ${roleText}`);
      onOpenChange(false);
      onSuccess?.();
    } catch (error: any) {
      console.error("Failed to create bindings:", error);
      setFormError(error?.data?.message || "Failed to assign users. Please try again.");
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px]">
        <DialogHeader className="space-y-2 pb-2">
          <DialogTitle className="text-2xl font-semibold leading-tight flex items-center gap-2">
            <UserPlus className="h-6 w-6 text-primary" />
            Assign Users to Role
          </DialogTitle>
          <DialogDescription>
            Select users to assign to{" "}
            {selectedRoles.length === 1 ? (
              <span className="font-medium text-foreground">{selectedRoles[0].name}</span>
            ) : (
              <span className="font-medium text-foreground">{selectedRoles.length} roles</span>
            )}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-4">
          {/* Selected Roles Display */}
          {selectedRoles.length > 0 && (
            <div className="flex flex-wrap gap-2 p-3 bg-muted/50 rounded-md">
              <Label className="w-full text-xs text-foreground mb-1">Assigning to:</Label>
              {selectedRoles.map((role) => (
                <Badge key={role.id} variant="secondary" className="flex items-center gap-1 bg-black/5 dark:bg-white/10 border-0">
                  <Shield className="h-3 w-3" />
                  {role.name}
                </Badge>
              ))}
            </div>
          )}

          {/* User Selection */}
          <div className="space-y-3">
            <Label className="flex items-center gap-2 text-sm font-semibold">
              <User className="h-4 w-4 text-primary" />
              Users
            </Label>
            <SearchableSelect
              multiple
              options={userOptions}
              value={selectedUserIds}
              onChange={(ids) => setSelectedUserIds(ids)}
              placeholder={busyLoading ? "Loading users..." : "Select users..."}
              searchPlaceholder="Search users..."
              emptyText="No users found"
              disabled={busyLoading}
              showSelectAll
              maxBadges={3}
              className="h-11"
            />
          </div>

          {/* Selected Users Preview */}
          {selectedUsersData.length > 0 && (
            <div className="flex flex-wrap gap-2 p-3 bg-primary/5 rounded-md border border-primary/20">
              <Label className="w-full text-xs text-foreground mb-1">
                Selected users ({selectedUsersData.length}):
              </Label>
              {selectedUsersData.map((user) => (
                <Badge
                  key={user.id}
                  variant="secondary"
                  className="flex items-center gap-1 bg-black/5 dark:bg-white/10 border-0"
                >
                  <User className="h-3 w-3" />
                  {user.name}
                </Badge>
              ))}
            </div>
          )}

          {formError && (
            <div className="rounded-md border border-destructive/50 bg-destructive/5 px-3 py-2 text-sm text-destructive">
              {formError}
            </div>
          )}

          <DialogFooter className="flex flex-col gap-2 sm:flex-row sm:justify-end">
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
              className="w-full sm:w-auto"
              disabled={isCreatingBinding}
            >
              Cancel
            </Button>
            <Button
              type="submit"
              className="w-full sm:w-auto"
              disabled={selectedUserIds.length === 0 || isCreatingBinding}
            >
              {isCreatingBinding ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Assigning...
                </>
              ) : (
                <>
                  <UserPlus className="mr-2 h-4 w-4" />
                  Assign {selectedUserIds.length > 0 ? `(${selectedUserIds.length})` : "Users"}
                </>
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
