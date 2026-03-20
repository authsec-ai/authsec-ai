import { useEffect, useMemo, useState } from "react";
import { useGetAuthSecRolesQuery } from "@/app/api/rolesApi";
import { useGetAdminUsersQuery } from "@/app/api/admin/usersApi";
import { useGetEndUsersQuery } from "@/app/api/enduser/usersApi";
import { useCreateBindingMutation, type RbacAudience } from "@/app/api/bindingsApi";
import { useGetScopeMappingsQuery } from "@/app/api/scopesApi";
import { useGetEndUserScopeMappingsQuery } from "@/app/api/enduser/scopesApi";
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
import { Textarea } from "@/components/ui/textarea";
import { Loader2, Shield, UserRound, Layers } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { toast } from "@/lib/toast";
import { resolveTenantId } from "@/utils/workspace";
import { SessionManager } from "@/utils/sessionManager";
import { SearchableSelect, type SearchableSelectOption } from "@/components/ui/searchable-select";

type RoleOption = {
  id: string;
  name: string;
  description?: string;
};

type UserOption = {
  id: string;
  name: string;
  email?: string;
};

interface MapRoleToScopeModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSuccess?: () => void;
  preselectedUsers?: Array<{ id: string; name: string; email?: string }>;
  preselectedRoles?: Array<{ id: string; name: string }>;
  audience: RbacAudience;
}

export function MapRoleToScopeModal({
  open,
  onOpenChange,
  onSuccess,
  preselectedUsers,
  preselectedRoles,
  audience
}: MapRoleToScopeModalProps) {
  const sessionData = SessionManager.getSession();
  const tenantId =
    resolveTenantId() ??
    sessionData?.tenant_id ??
    (sessionData as any)?.tenantId ??
    sessionData?.jwtPayload?.tenant_id ??
    "";

  const [selectedUserId, setSelectedUserId] = useState("");
  const [selectedRoleId, setSelectedRoleId] = useState("");
  const [selectedScope, setSelectedScope] = useState("");
  const [conditions, setConditions] = useState("");
  const [formError, setFormError] = useState<string | null>(null);

  // Initialize preselected values
  useEffect(() => {
    if (open && preselectedUsers && preselectedUsers.length > 0) {
      setSelectedUserId(preselectedUsers[0].id);
    }
    if (open && preselectedRoles && preselectedRoles.length > 0) {
      setSelectedRoleId(preselectedRoles[0].id);
    }
  }, [open, preselectedUsers, preselectedRoles]);

  const isEndUserAudience = audience === "endUser";

  const {
    data: adminUsersResponse,
    isLoading: isLoadingAdminUsers,
    isFetching: isFetchingAdminUsers,
  } = useGetAdminUsersQuery(
    {
      page: 1,
      limit: 100,
    },
    { skip: isEndUserAudience }
  );

  const {
    data: endUsersResponse,
    isLoading: isLoadingEndUsers,
    isFetching: isFetchingEndUsers,
  } = useGetEndUsersQuery(
    {
      page: 1,
      limit: 100,
    },
    { skip: !isEndUserAudience }
  );

  const usersResponse = isEndUserAudience ? endUsersResponse : adminUsersResponse;
  const isLoadingUsers = isEndUserAudience ? isLoadingEndUsers : isLoadingAdminUsers;
  const isFetchingUsers = isEndUserAudience ? isFetchingEndUsers : isFetchingAdminUsers;

  const {
    data: rolesResponse = [],
    isLoading: isLoadingRoles,
    isFetching: isFetchingRoles,
  } = useGetAuthSecRolesQuery({
    tenant_id: tenantId,
    audience,
  });

  // Admin scope mappings - skip when endUser
  const { data: adminScopeMappings = [], isFetching: isFetchingAdminScopes } = useGetScopeMappingsQuery(undefined, {
    skip: audience === 'endUser',
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
  });

  // End-user scope mappings - skip when admin
  const { data: endUserScopeMappings = [], isFetching: isFetchingEndUserScopes } = useGetEndUserScopeMappingsQuery(undefined, {
    skip: audience === 'admin',
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
  });

  // Combined fetching state
  const isFetchingScopes = isFetchingAdminScopes || isFetchingEndUserScopes;

  // Use appropriate scope mappings based on audience
  const scopeMappings = useMemo(() => {
    if (audience === 'admin') {
      return adminScopeMappings;
    }
    return endUserScopeMappings;
  }, [audience, adminScopeMappings, endUserScopeMappings]);

  // Extract scope names from mappings
  const scopeNames = useMemo(() => {
    return scopeMappings.map((mapping) => mapping.scope_name);
  }, [scopeMappings]);

  const [createBinding, { isLoading: isCreatingBinding }] = useCreateBindingMutation();

  useEffect(() => {
    if (!open) {
      setSelectedUserId("");
      setSelectedRoleId("");
      setSelectedScope("");
      setConditions("");
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
        };
      })
      .filter(Boolean) as UserOption[];

    const unique = new Map<string, UserOption>();
    mapped.forEach((user) => {
      if (!unique.has(user.id)) unique.set(user.id, user);
    });
    return Array.from(unique.values());
  }, [usersResponse]);

  const roles = useMemo<RoleOption[]>(() => {
    const rawRoles = Array.isArray(rolesResponse)
      ? rolesResponse
      : Array.isArray((rolesResponse as any)?.roles)
        ? (rolesResponse as any).roles
        : [];

    const mapped = rawRoles
      .map((role: any, index: number) => {
        const rawId =
          role?.id ??
          role?.role_id ??
          role?.roleId ??
          role?.uuid ??
          role?.uid ??
          role?.external_id ??
          role?.slug ??
          role?.name ??
          role?.role_name ??
          `role-${index}`;
        const name =
          role?.name ??
          role?.role_name ??
          role?.roleName ??
          role?.label ??
          role?.display_name ??
          role?.role ??
          `Role ${index + 1}`;

        if (!rawId || !name) return null;

        const description =
          role?.description ||
          role?.details ||
          role?.summary ||
          role?.meta?.description ||
          undefined;

        return {
          id: String(rawId),
          name: String(name),
          description: description ? String(description) : undefined,
        };
      })
      .filter(Boolean) as RoleOption[];

    const unique = new Map<string, RoleOption>();
    mapped.forEach((role) => {
      if (!unique.has(role.id)) unique.set(role.id, role);
    });
    return Array.from(unique.values());
  }, [rolesResponse]);

  const busyLoading =
    isLoadingUsers || isLoadingRoles || isFetchingUsers || isFetchingRoles || isFetchingScopes;

  // Convert to SearchableSelectOption format
  const userOptions = useMemo<SearchableSelectOption[]>(() => {
    const options = users.map((user) => ({
      value: user.id,
      label: user.name.trim() || user.email || user.id,
      description: user.email,
    }));
    return options;
  }, [users]);

  const roleOptions = useMemo<SearchableSelectOption[]>(() => {
    return roles.map((role) => ({
      value: role.id,
      label: role.name.trim() || role.id,
      description: role.description,
    }));
  }, [roles]);

  const scopeOptions = useMemo<SearchableSelectOption[]>(() => {
    return scopeNames.map((scope) => ({
      value: scope,
      label: scope,
    }));
  }, [scopeNames]);

  const parseConditions = (): Record<string, any> | null => {
    if (!conditions.trim()) return {};

    try {
      const parsed = JSON.parse(conditions);
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed;
      }
      setFormError("Conditions must be a valid JSON object.");
      return null;
    } catch (error) {
      setFormError("Conditions must be valid JSON.");
      return null;
    }
  };

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    setFormError(null);

    if (!selectedUserId || !selectedRoleId) {
      setFormError("Please select both a user and a role.");
      return;
    }

    if (!selectedScope) {
      setFormError("Please select a scope.");
      return;
    }

    const parsedConditions = parseConditions();
    if (parsedConditions === null) return;

    try {
      // If preselected users, create bindings for all users
      if (preselectedUsers && preselectedUsers.length > 0) {
        const promises = preselectedUsers.map((user) =>
          createBinding({
            user_id: user.id,
            role_id: selectedRoleId,
            conditions: parsedConditions,
            scope: {
              id: "*",
              type: selectedScope,
            },
            audience,
          }).unwrap()
        );
        await Promise.all(promises);
        const userText = preselectedUsers.length === 1 ? `user "${preselectedUsers[0].name || preselectedUsers[0].email}"` : `${preselectedUsers.length} users`;
        toast.success(`Role mapped to scope "${selectedScope}" for ${userText} successfully.`);
      }
      // If preselected roles, create bindings for all roles
      else if (preselectedRoles && preselectedRoles.length > 0) {
        const promises = preselectedRoles.map((role) =>
          createBinding({
            user_id: selectedUserId,
            role_id: role.id,
            conditions: parsedConditions,
            scope: {
              id: "*",
              type: selectedScope,
            },
            audience,
          }).unwrap()
        );
        await Promise.all(promises);
        const roleText = preselectedRoles.length === 1 ? `role "${preselectedRoles[0].name}"` : `${preselectedRoles.length} roles`;
        toast.success(`${roleText} mapped to scope "${selectedScope}" successfully.`);
      }
      // Default single binding
      else {
        await createBinding({
          user_id: selectedUserId,
          role_id: selectedRoleId,
          conditions: parsedConditions,
          scope: {
            id: "*",
            type: selectedScope,
          },
          audience,
        }).unwrap();
        toast.success(`Role mapped to scope "${selectedScope}" successfully.`);
      }

      onOpenChange(false);
      onSuccess?.();
    } catch (error: any) {
      console.error("Failed to create binding:", error);
      setFormError(error?.data?.message || "Failed to map role to scope. Please try again.");
    }
  };

  // Copy based on audience
  const copy = useMemo(() => ({
    title: "Map roles to scope",
    description: audience === "admin"
      ? "Connect a role to a user. Scopes are applied internally while the scope format is being aligned."
      : "Assign a role to a user with specific scope permissions.",
  }), [audience]);

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <DialogHeader className="space-y-2 pb-2">
          <DialogTitle className="text-2xl font-semibold leading-tight">
            {copy.title}
          </DialogTitle>
          <DialogDescription>
            {copy.description}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Preselected Users Display */}
          {preselectedUsers && preselectedUsers.length > 0 && (
            <div className="flex flex-wrap gap-2 p-3 bg-muted/50 rounded-md">
              <Label className="w-full text-xs text-foreground mb-1">Assigning to:</Label>
              {preselectedUsers.slice(0, 5).map((user) => (
                <Badge key={user.id} variant="secondary" className="flex items-center gap-1">
                  <UserRound className="h-3 w-3" />
                  {user.name || user.email}
                </Badge>
              ))}
              {preselectedUsers.length > 5 && (
                <Badge variant="outline">+{preselectedUsers.length - 5} more</Badge>
              )}
            </div>
          )}

          <div className="space-y-3">
            <Label className="flex items-center gap-2 text-sm font-semibold">
              <UserRound className="h-4 w-4 text-primary" />
              User
            </Label>
            <SearchableSelect
              options={userOptions}
              value={selectedUserId || undefined}
              onChange={(val) => setSelectedUserId(val ?? "")}
              placeholder={busyLoading ? "Loading users..." : "Select a user..."}
              searchPlaceholder="Search users..."
              emptyText="No users found"
              disabled={busyLoading || !!(preselectedUsers && preselectedUsers.length > 0)}
              className="h-11"
            />
          </div>

          {/* Preselected Roles Display */}
          {preselectedRoles && preselectedRoles.length > 0 && (
            <div className="flex flex-wrap gap-2 p-3 bg-muted/50 rounded-md">
              <Label className="w-full text-xs text-foreground mb-1">Assigning to:</Label>
              {preselectedRoles.map((role) => (
                <Badge key={role.id} variant="secondary" className="flex items-center gap-1">
                  <Shield className="h-3 w-3" />
                  {role.name}
                </Badge>
              ))}
            </div>
          )}

          <div className="space-y-3">
            <Label className="flex items-center gap-2 text-sm font-semibold">
              <Shield className="h-4 w-4 text-primary" />
              Role
            </Label>
            <SearchableSelect
              options={roleOptions}
              value={selectedRoleId || undefined}
              onChange={(val) => setSelectedRoleId(val ?? "")}
              placeholder={busyLoading ? "Loading roles..." : "Select a role..."}
              searchPlaceholder="Search roles..."
              emptyText="No roles found"
              disabled={busyLoading || !!(preselectedRoles && preselectedRoles.length > 0)}
              className="h-11"
            />
          </div>

          <div className="space-y-3">
            <Label className="flex items-center gap-2 text-sm font-semibold">
              <Layers className="h-4 w-4 text-primary" />
              Scope
            </Label>
            <SearchableSelect
              options={scopeOptions}
              value={selectedScope || undefined}
              onChange={(val) => setSelectedScope(val ?? "")}
              placeholder={busyLoading ? "Loading scopes..." : "Select a scope..."}
              searchPlaceholder="Search scopes..."
              emptyText="No scopes found"
              disabled={busyLoading}
              className="h-11"
            />
          </div>

          <div className="space-y-2">
            <Label className="text-sm font-semibold">Conditions (optional JSON)</Label>
            <Textarea
              placeholder='e.g., {"region": "us-west"}'
              value={conditions}
              onChange={(e) => setConditions(e.target.value)}
              className="min-h-[100px]"
            />
            <p className="text-xs text-foreground">
              Leave blank to send empty conditions.
            </p>
          </div>

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
              disabled={!selectedUserId || !selectedRoleId || !selectedScope || isCreatingBinding}
            >
              {isCreatingBinding ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Mapping...
                </>
              ) : (
                "Map role to scope"
              )}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
