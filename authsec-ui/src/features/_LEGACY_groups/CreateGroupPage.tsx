import { type FormEvent, useState, useMemo, useEffect } from "react";
import { useParams, useLocation } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Textarea } from "@/components/ui/textarea";
import { Badge } from "@/components/ui/badge";
import {
  ArrowLeft,
  CheckCircle2,
  Loader2,
  X, // 1. Import new icons for dropdown visual cues
  Users,
  ChevronDown,
  ChevronUp,
} from "lucide-react";
import { toast } from "@/lib/toast";
import {
  useCreateGroupsMutation,
  useGetGroupsByTenantQuery,
  useUpdateGroupMutation,
} from "@/app/api/admin/groupsApi";
// Import both admin and enduser user APIs
import { useGetAdminUsersQuery } from "@/app/api/admin/usersApi";
import { useGetEndUsersQuery } from "@/app/api/enduser/usersApi";
import { useAddUserToGroupsMutation } from "@/app/api/enduser/groupsApi";
import { ScrollArea } from "@/components/ui/scroll-area";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { resolveTenantId } from "@/utils/workspace";
import { useContextualNavigate } from "@/hooks/useContextualNavigate";
import { cn } from "@/lib/utils";

interface PrefillGroupState {
  suggestedName?: string;
  description?: string;
  selectedUserIds?: string[];
  selectedUserEmails?: string[];
  source?: string;
}

interface CreateGroupLocationState {
  prefillGroup?: PrefillGroupState;
}

const LOG_PREFIX = "[CreateGroupPage]";
const debug = (...args: any[]) => console.log(LOG_PREFIX, ...args);

export default function CreateGroupPage() {
  console.log("=== CreateGroupPage RENDER ===");
  const navigate = useContextualNavigate();
  const { id: groupId } = useParams<{ id: string }>();
  const location = useLocation();
  const { isAdmin } = useRbacAudience();
  const isEditMode = Boolean(groupId);
  const locationState = location.state as CreateGroupLocationState | undefined;
  const prefillGroup = !isEditMode ? locationState?.prefillGroup : undefined;
  console.log("Render info:", { isAdmin, isEditMode, groupId }); // Base path uses contextual navigation to prepend the correct audience segment

  const basePath = "/groups"; // API hooks

  const [createGroups] = useCreateGroupsMutation();
  const [updateGroup] = useUpdateGroupMutation();
  const [addUserToGroups] = useAddUserToGroupsMutation(); // Resolve tenant ID from workspace context

  const tenantId = resolveTenantId(); // Fetch existing groups for edit mode

  const { data: groupsData } = useGetGroupsByTenantQuery(tenantId || "", {
    skip: !tenantId || !isEditMode,
  }); // Fetch users based on audience context // Admin context: fetch admin users (staff) // End-user context: fetch end-users (customers)

  const {
    data: adminUsersResponse,
    isLoading: isLoadingAdminUsers,
    isFetching: isFetchingAdminUsers,
    error: adminUsersError,
  } = useGetAdminUsersQuery(
    {
      page: 1,
      limit: 1000, // Fetch all users for selection
      tenant_id: tenantId || "",
    },
    { skip: !isAdmin || !tenantId }
  );

  const {
    data: endUsersResponse,
    isLoading: isLoadingEndUsers,
    isFetching: isFetchingEndUsers,
    error: endUsersError,
  } = useGetEndUsersQuery(
    {
      page: 1,
      limit: 1000, // Fetch all users for selection
      tenant_id: tenantId || "",
    },
    { skip: isAdmin || !tenantId }
  ); // Select appropriate data based on context

  const usersResponse = isAdmin ? adminUsersResponse : endUsersResponse;
  const isLoadingUsers = isAdmin ? isLoadingAdminUsers : isLoadingEndUsers;
  const isFetchingUsers = isAdmin ? isFetchingAdminUsers : isFetchingEndUsers;
  const usersError = isAdmin ? adminUsersError : endUsersError; // Log API query states

  useEffect(() => {
    debug("API Query States:", {
      audience: isAdmin ? "admin" : "enduser",
      tenantId,
      isAdmin,
      adminUsers: {
        skip: !isAdmin || !tenantId,
        isLoading: isLoadingAdminUsers,
        isFetching: isFetchingAdminUsers,
        hasData: !!adminUsersResponse,
        error: adminUsersError,
      },
      endUsers: {
        skip: isAdmin || !tenantId,
        isLoading: isLoadingEndUsers,
        isFetching: isFetchingEndUsers,
        hasData: !!endUsersResponse,
        error: endUsersError,
      },
      selected: {
        isLoading: isLoadingUsers,
        isFetching: isFetchingUsers,
        hasData: !!usersResponse,
        error: usersError,
      },
    });
  }, [
    isAdmin,
    tenantId,
    isLoadingAdminUsers,
    isFetchingAdminUsers,
    adminUsersResponse,
    adminUsersError,
    isLoadingEndUsers,
    isFetchingEndUsers,
    endUsersResponse,
    endUsersError,
    isLoadingUsers,
    isFetchingUsers,
    usersResponse,
    usersError,
  ]); // Group state

  const [isSubmitting, setIsSubmitting] = useState(false);
  const [groupName, setGroupName] = useState(
    () => prefillGroup?.suggestedName || ""
  );
  const [groupDescription, setGroupDescription] = useState(
    () => prefillGroup?.description || ""
  );
  const [selectedUsers, setSelectedUsers] = useState<string[]>(
    () => prefillGroup?.selectedUserIds || []
  );
  const [userSearchTerm, setUserSearchTerm] = useState("");
  // 2. New state to control the visibility of the user list dropdown
  const [showUserList, setShowUserList] = useState(false); // Load group data in edit mode

  useEffect(() => {
    if (isEditMode && groupsData && groupId) {
      const existingGroup = groupsData.find((g) => g.id === groupId);
      if (existingGroup) {
        setGroupName(existingGroup.name);
        setGroupDescription(existingGroup.description || "");
      }
    }
  }, [isEditMode, groupsData, groupId]);

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            pageTitle: isEditMode ? "Edit admin group" : "Create admin group",
            pageSubtitle: isEditMode
              ? "Update the details of this admin group."
              : "Organize privileged teammates into assignment bundles you can reuse.",
            badgeLabel: "Admin RBAC",
            cardTitle: isEditMode ? "Edit admin group" : "Admin group details",
            cardDescription:
              "Choose a descriptive name that helps other admins understand who belongs here.",
            nameLabel: "Group name",
            namePlaceholder: "e.g. security-ops, compliance-admins",
            nameHelper: "Use kebab-case for consistency (e.g. `security-ops`).",
            addPeopleLabel: "Add admins",
            selectButtonLabel: "Select admins to add to group",
            personSingular: "admin",
            personPlural: "admins",
            createCta: isEditMode ? "Update" : "Create",
            successNoun: "Admin group",
            entityNoun: "group",
          }
        : {
            pageTitle: isEditMode ? "Edit user group" : "Create user group",
            pageSubtitle: isEditMode
              ? "Update the details of this user group."
              : "Segment end users into groups to tailor access and communications.",
            badgeLabel: "End-user access",
            cardTitle: isEditMode ? "Edit group" : "Group details",
            cardDescription:
              "Pick a name that reflects the customers or members included in this audience.",
            nameLabel: "Group name",
            namePlaceholder: "e.g. beta-testers, premium-plan",
            nameHelper:
              "Pick names your teammates will recognize (e.g. `vip-customers`).",
            addPeopleLabel: "Add end users",
            selectButtonLabel: "Select end users to include",
            personSingular: "end user",
            personPlural: "end users",
            createCta: isEditMode ? "Update" : "Create",
            successNoun: "Group",
            entityNoun: "group",
          },
    [isAdmin, isEditMode]
  );

  const prefillSummary = useMemo(() => {
    if (
      !prefillGroup?.selectedUserEmails ||
      prefillGroup.selectedUserEmails.length === 0
    ) {
      return null;
    }
    const preview = prefillGroup.selectedUserEmails.slice(0, 3).join(", ");
    const remaining = prefillGroup.selectedUserEmails.length - 3;
    return remaining > 0 ? `${preview} +${remaining} more` : preview;
  }, [prefillGroup?.selectedUserEmails]);

  const prefillCount = prefillGroup?.selectedUserIds?.length ?? 0; // Extract users from response

  const apiUsers = useMemo(() => {
    debug("Processing usersResponse:", {
      hasResponse: !!usersResponse,
      responseType: typeof usersResponse,
      responseKeys: usersResponse ? Object.keys(usersResponse) : [],
      rawResponse: usersResponse,
    });

    if (!usersResponse) {
      debug("No usersResponse - returning empty array");
      return [];
    }

    const candidates: any[] = [];
    const pushArray = (maybeArray: any) => {
      if (Array.isArray(maybeArray)) {
        debug("Found array with", maybeArray.length, "items");
        candidates.push(...maybeArray);
      }
    };

    const resp: any = usersResponse;
    pushArray(resp?.users);
    pushArray(resp?.data?.users);
    pushArray(resp?.data);
    pushArray(resp?.results);

    debug("Total candidates before filtering:", candidates.length);

    const processed = candidates
      .filter(Boolean)
      .map((user) => {
        const id = user.user_id || user.id || user.email || user.username;
        const name =
          user.name ||
          user.first_name ||
          user.username ||
          user.email ||
          "Unknown user";
        return {
          id: String(id),
          email: user.email || user.username || String(id),
          name: String(name),
        };
      })
      .filter(
        (user, index, self) =>
          user.id && self.findIndex((u) => u.id === user.id) === index
      );

    debug("Processed users:", {
      total: processed.length,
      sample: processed.slice(0, 3),
    });

    return processed;
  }, [usersResponse]);

  const prefilledUsers = useMemo(() => {
    if (
      !prefillGroup?.selectedUserIds ||
      prefillGroup.selectedUserIds.length === 0
    ) {
      return [];
    }
    return prefillGroup.selectedUserIds.map((id, index) => {
      const label = prefillGroup.selectedUserEmails?.[index] ?? String(id);
      return {
        id: String(id),
        email: label,
        name: label,
      };
    });
  }, [prefillGroup]);

  const users = useMemo(() => {
    const byId = new Map<string, { id: string; email: string; name: string }>();
    prefilledUsers.forEach((user) => {
      byId.set(user.id, user);
    });
    apiUsers.forEach((user) => {
      byId.set(user.id, user);
    });
    const result = Array.from(byId.values()).sort((a, b) => {
      const aKey = a.email || a.name || "";
      const bKey = b.email || b.name || "";
      return aKey.localeCompare(bKey);
    });

    debug("Users available for selection", {
      total: result.length,
      sample: result.slice(0, 5),
    });
    return result;
  }, [apiUsers, prefilledUsers]);

  const filteredUsers = useMemo(() => {
    const term = userSearchTerm.trim().toLowerCase();
    if (!term) return users;
    return users.filter((user) => {
      const name = user.name?.toLowerCase() ?? "";
      const email = user.email?.toLowerCase() ?? "";
      return name.includes(term) || email.includes(term);
    });
  }, [userSearchTerm, users]);

  useEffect(() => {
    debug("Users dataset ready", {
      audience: isAdmin ? "admin" : "enduser",
      totalOptions: users.length,
      isLoading: isLoadingUsers,
      isFetching: isFetchingUsers,
      hasError: !!usersError,
      buttonShouldBeDisabled: isLoadingUsers && users.length === 0,
      usersBreakdown: {
        apiUsers: apiUsers.length,
        prefilledUsers: prefilledUsers.length,
        totalUnique: users.length,
      },
    });
  }, [
    users.length,
    isLoadingUsers,
    isFetchingUsers,
    usersError,
    isAdmin,
    apiUsers.length,
    prefilledUsers.length,
  ]);

  const handleToggleUser = (userId: string) => {
    setSelectedUsers((prev) =>
      prev.includes(userId)
        ? prev.filter((id) => id !== userId)
        : [...prev, userId]
    );
  };

  const handleRemoveUser = (userId: string) => {
    debug("Removing user from selection", { userId });
    setSelectedUsers((prev) => prev.filter((id) => id !== userId));
  };

  const handleSubmit = async (event?: FormEvent) => {
    event?.preventDefault();
    setIsSubmitting(true);
    if (!groupName.trim()) {
      toast.error(`${audienceCopy.nameLabel} is required`);
      setIsSubmitting(false);
      return;
    }
    if (!tenantId) {
      toast.error("Tenant context missing; please sign in again.");
      setIsSubmitting(false);
      return;
    }

    debug("Submitting form", {
      audience: isAdmin ? "admin" : "enduser",
      isEditMode,
      groupId,
      groupName: groupName.trim(),
      selectedUsers,
      tenantId,
    });

    try {
      if (isEditMode && groupId) {
        // Update existing group
        await updateGroup({
          id: groupId,
          data: {
            tenant_id: tenantId,
            name: groupName.trim(),
            description: groupDescription.trim() || undefined,
          },
        }).unwrap();

        toast.success(
          `${audienceCopy.successNoun} "${groupName}" updated successfully`
        );
        navigate(basePath);
        return;
      } // Create new group using AuthSec API

      const createGroupsPayload = {
        tenant_id: tenantId,
        groups: [
          {
            name: groupName.trim(),
            description: groupDescription.trim() || undefined,
          },
        ],
      };

      const result = await createGroups(createGroupsPayload).unwrap(); // If users were selected, add them to the newly created group

      if (
        selectedUsers.length > 0 &&
        result.groups &&
        result.groups.length > 0
      ) {
        const createdGroup = result.groups[0];
        const createdGroupName = createdGroup.name || groupName.trim();
        debug("Created group response", {
          createdGroup,
          selectedUserCount: selectedUsers.length,
        }); // Add each selected user to the new group

        const mappingErrors: string[] = [];

        for (const userId of selectedUsers) {
          try {
            await addUserToGroups({
              tenant_id: tenantId,
              user_id: userId,
              groups: [createdGroupName],
            }).unwrap();
          } catch (mapError: any) {
            const userEmail =
              users.find((u) => u.id === userId)?.email || userId;
            mappingErrors.push(userEmail);
            console.error(
              `Failed to add user ${userEmail} to group:`,
              mapError
            );
          }
        }

        if (mappingErrors.length > 0) {
          toast.warning(
            `${
              audienceCopy.successNoun
            } "${groupName}" created, but failed to add ${
              mappingErrors.length
            } user(s): ${mappingErrors.join(", ")}`
          );
        } else {
          toast.success(
            `${audienceCopy.successNoun} "${groupName}" created with ${selectedUsers.length} user(s)`
          );
        }
      } else {
        debug("Group created without user assignments", {
          selectedUserCount: selectedUsers.length,
          groups: result.groups,
        });
        toast.success(
          `${audienceCopy.successNoun} "${groupName}" created successfully`
        );
      }

      navigate(basePath);
    } catch (error: any) {
      console.error("Error creating group:", error); // Handle RTK Query errors properly
      const message =
        error?.data?.message ||
        error?.message ||
        "Failed to create group. Please try again.";
      toast.error(message);
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleCancel = () => {
    navigate(basePath);
  };

  const isFormValid = () => {
    return groupName.trim().length > 0;
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="container mx-auto max-w-8xl space-y-8 px-8 py-8">
        {/* Header  */}
        <header className="bg-card border border-border rounded-sm p-5 shadow-sm">
          <div className="flex justify-between items-center gap-4">
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
              form="create-group-form"
              disabled={!isFormValid() || isSubmitting}
              className="min-w-[140px] h-10 px-4"
            >
              {isSubmitting ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {isEditMode ? "Saving..." : "Creating..."}
                </>
              ) : (
                <>
                  <CheckCircle2 className="mr-2 h-4 w-4" />
                  {audienceCopy.createCta}
                </>
              )}
            </Button>
          </div>
        </header>
        <form
          id="create-group-form"
          onSubmit={handleSubmit}
          className="grid grid-cols-1 lg:grid-cols-2 gap-6"
        >
          {/* Left Column: Group Name and Description */}
          <div className="space-y-6">
            <div className="space-y-2">
              <Label htmlFor="groupName" className="text-sm font-medium">
                {audienceCopy.nameLabel}
                <span className="ml-1 text-destructive">*</span>
              </Label>
              <Input
                id="groupName"
                placeholder={audienceCopy.namePlaceholder}
                value={groupName}
                onChange={(e) => setGroupName(e.target.value)}
              />
              <p className="text-xs text-foreground">
                {audienceCopy.nameHelper}
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="groupDescription" className="text-sm font-medium">
                Description
                <span className="text-foreground">(optional)</span>
              </Label>
              <Textarea
                id="groupDescription"
                placeholder={`Describe the purpose and scope of this ${audienceCopy.entityNoun}.`}
                value={groupDescription}
                onChange={(e) => setGroupDescription(e.target.value)}
                className="min-h-[200px] resize-none"
              />
              <p className="text-xs text-foreground">
                Help others understand what this {audienceCopy.entityNoun} is
                for and who should be in it.
              </p>
            </div>
          </div>
          {/* Right Column: Group Members Selection */}
          <div className="space-y-6">
            <div className="space-y-2">
              <Label className="text-sm font-medium">
                {audienceCopy.addPeopleLabel}
                <span className="text-foreground">(optional)</span>
              </Label>
              {!isEditMode && prefillCount > 0 && prefillSummary && (
                <div className="rounded-lg border border-dashed border-border/60 bg-muted/40 px-3 py-2 text-xs text-foreground">
                  Pulled {prefillCount}
                  {prefillCount === 1
                    ? audienceCopy.personSingular
                    : audienceCopy.personPlural}
                  from Users: {prefillSummary}. Review the selection or add more
                  before saving.
                </div>
              )}
              {/* 3. The main dropdown/combobox container */}
              <div className="relative">
                {/* 4. The Dropdown/Search Trigger */}
                <div className="relative">
                  <Input
                    value={userSearchTerm}
                    onChange={(event) => {
                      setUserSearchTerm(event.target.value);
                      setShowUserList(true); // Open list when typing
                    }}
                    onFocus={() => setShowUserList(true)} // Open list on focus
                    placeholder={`Search ${audienceCopy.personPlural} to add...`}
                    className={cn(
                      "pr-10",
                      showUserList && "rounded-b-none border-b-0"
                    )}
                  />
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    className="absolute right-0 top-1/2 -translate-y-1/2 h-8 w-8 p-0"
                    onClick={() => setShowUserList((prev) => !prev)}
                    aria-expanded={showUserList}
                  >
                    {showUserList ? (
                      <ChevronUp className="h-4 w-4" />
                    ) : (
                      <ChevronDown className="h-4 w-4" />
                    )}
                  </Button>
                </div>
                {/* 5. The Dropdown Content (User List) */}
                {showUserList && (
                  <div className="absolute z-10 w-full rounded-b-lg border border-t-0 border-border/60 bg-background/95 shadow-lg overflow-hidden">
                    {/* Search Info Bar */}
                    <div className="flex items-center justify-between text-xs text-foreground p-3 border-b border-border/60 bg-background">
                      <span className="font-medium">
                        {isLoadingUsers && users.length === 0
                          ? `Loading ${audienceCopy.personPlural.toLowerCase()}...`
                          : `${
                              filteredUsers.length
                            } ${audienceCopy.personPlural.toLowerCase()} available`}
                      </span>
                      {selectedUsers.length > 0 && (
                        <span className="flex items-center gap-1">
                          <Users className="h-3 w-3" />
                          {selectedUsers.length} selected
                        </span>
                      )}
                    </div>

                    {/* User List with ScrollArea */}
                    <ScrollArea className="h-64">
                      {/* User List */}
                      {isLoadingUsers && users.length === 0 ? (
                        <div className="flex items-center justify-center gap-2 py-8 text-sm text-foreground">
                          <Loader2 className="h-4 w-4 animate-spin" />
                          Loading {audienceCopy.personPlural}...
                        </div>
                      ) : filteredUsers.length === 0 ? (
                        <div className="py-6 text-center text-sm text-foreground">
                          No {audienceCopy.personPlural.toLowerCase()} match
                          your search.
                          {userSearchTerm && (
                            <Button
                              type="button"
                              variant="link"
                              size="sm"
                              className="h-auto px-1 text-xs"
                              onClick={() => setUserSearchTerm("")}
                            >
                              (Clear Search)
                            </Button>
                          )}
                        </div>
                      ) : (
                        <div className="divide-y divide-border/60">
                          {filteredUsers.map((user) => {
                            const isSelected = selectedUsers.includes(user.id);
                            return (
                              <button
                                key={user.id}
                                type="button"
                                className={cn(
                                  "flex w-full items-center justify-between gap-3 px-4 py-2 text-left text-sm transition hover:bg-muted/40 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-1",
                                  isSelected ? "bg-muted/50" : ""
                                )}
                                onClick={() => handleToggleUser(user.id)}
                              >
                                <div className="min-w-0">
                                  <p className="truncate font-medium">
                                    {user.name}
                                  </p>
                                  <p className="truncate text-xs text-foreground">
                                    {user.email}
                                  </p>
                                </div>
                                <CheckCircle2
                                  className={cn(
                                    "h-4 w-4 flex-shrink-0 text-primary transition-opacity duration-200",
                                    isSelected ? "opacity-100" : "opacity-0"
                                  )}
                                />
                              </button>
                            );
                          })}
                        </div>
                      )}
                    </ScrollArea>
                  </div>
                )}
              </div>
              {/* 6. Selected Users Tag List */}
              {selectedUsers.length > 0 && (
                <div className="flex flex-wrap gap-2 rounded-lg border border-border/60 bg-muted/40 p-3">
                  {selectedUsers.map((userId) => {
                    const user = users.find((u) => u.id === userId);
                    if (!user) return null;
                    return (
                      <Badge
                        key={userId}
                        variant="secondary"
                        className="flex items-center gap-1 px-2 py-1"
                      >
                        <span className="text-xs">{user.email}</span>
                        <button
                          type="button"
                          onClick={() => handleRemoveUser(userId)}
                          className="ml-1 rounded-full hover:bg-muted"
                        >
                          <X className="h-3 w-3" />
                        </button>
                      </Badge>
                    );
                  })}
                </div>
              )}
              <p className="text-xs text-foreground">
                You can add {audienceCopy.personPlural} to this{" "}
                {audienceCopy.entityNoun} now or later from the groups page.
              </p>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
