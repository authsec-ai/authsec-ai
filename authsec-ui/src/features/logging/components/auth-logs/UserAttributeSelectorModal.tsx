import { useState, useEffect, useMemo } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "../../../../components/ui/dialog";
import { Button } from "../../../../components/ui/button";
import { Checkbox } from "../../../../components/ui/checkbox";
import { Badge } from "../../../../components/ui/badge";
import { Input } from "../../../../components/ui/input";
import { Search, User, Hash, ChevronLeft, Loader2 } from "lucide-react";
import { useGetAdminUsersQuery } from "../../../../app/api/admin/usersApi";
import { SessionManager } from "../../../../utils/sessionManager";

export type UserAttribute = "userId" | "username";

export interface UserSelection {
  type: UserAttribute;
  values: string[];
}

interface UserAttributeSelectorModalProps {
  isOpen: boolean;
  onClose: () => void;
  onApply: (selection: UserSelection) => void;
}

type Step = "select-type" | "select-users";

export function UserAttributeSelectorModal({
  isOpen,
  onClose,
  onApply,
}: UserAttributeSelectorModalProps) {
  // Get tenant ID from session
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  // Fetch users from API
  const {
    data: usersResponse,
    isLoading: isLoadingUsers,
    isError: isErrorUsers,
    error: usersError,
  } = useGetAdminUsersQuery(
    {
      page: 1,
      limit: 50,
      tenant_id: tenantId || "",
      active: true,
    },
    {
      skip: !tenantId || !isOpen,
      refetchOnMountOrArgChange: true,
    }
  );

  // Transform API response to user list
  const users = useMemo(() => {
    const rawUsers = Array.isArray(usersResponse?.users)
      ? usersResponse.users
      : Array.isArray(usersResponse)
      ? usersResponse
      : [];

    return rawUsers
      .map((user: any) => ({
        id: user.id || user.user_id,
        username: user.username,
        email: user.email,
        name: user.name,
      }))
      .filter((user) => user.id && user.username)
      .filter(
        (user, index, self) => self.findIndex((u) => u.id === user.id) === index
      );
  }, [usersResponse]);

  const [step, setStep] = useState<Step>("select-type");
  const [selectedType, setSelectedType] = useState<UserAttribute | null>(null);
  const [selectedValues, setSelectedValues] = useState<string[]>([]);
  const [searchQuery, setSearchQuery] = useState("");

  useEffect(() => {
    if (!isOpen) {
      setTimeout(() => {
        setStep("select-type");
        setSelectedType(null);
        setSelectedValues([]);
        setSearchQuery("");
      }, 200);
    }
  }, [isOpen]);

  const handleTypeSelect = (type: UserAttribute) => {
    setSelectedType(type);
    setStep("select-users");
  };

  const handleBack = () => {
    setStep("select-type");
    setSelectedValues([]);
    setSearchQuery("");
  };

  const handleToggleValue = (value: string) => {
    setSelectedValues((prev) =>
      prev.includes(value) ? prev.filter((v) => v !== value) : [...prev, value]
    );
  };

  const handleSelectAll = () => {
    const filtered = getFilteredUsers();
    const allValues = filtered.map((u) =>
      selectedType === "userId" ? u.id : u.username
    );
    setSelectedValues(allValues);
  };

  const handleDeselectAll = () => {
    setSelectedValues([]);
  };

  const handleApply = () => {
    if (selectedType && selectedValues.length > 0) {
      onApply({ type: selectedType, values: selectedValues });
      onClose();
    }
  };

  const handleCancel = () => {
    onClose();
  };

  const getFilteredUsers = () => {
    if (!searchQuery.trim()) return users;

    const query = searchQuery.toLowerCase();
    return users.filter(
      (user) =>
        user.username?.toLowerCase().includes(query) ||
        user.id?.toLowerCase().includes(query) ||
        user.email?.toLowerCase().includes(query)
    );
  };

  const filteredUsers = getFilteredUsers();

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="sm:max-w-[600px] bg-white dark:bg-neutral-900 border-slate-200 dark:border-neutral-800">
        {step === "select-type" ? (
          <>
            <DialogHeader>
              <DialogTitle className="text-slate-900 dark:text-zinc-100">
                Group By Users
              </DialogTitle>
              <DialogDescription className="text-slate-600 dark:text-zinc-400">
                Select how you want to group the auth logs by users
              </DialogDescription>
            </DialogHeader>

            <div className="py-6">
              <div className="space-y-3">
                <button
                  onClick={() => handleTypeSelect("username")}
                  className="w-full flex items-start gap-4 p-4 rounded-lg border-2 border-slate-200 dark:border-neutral-800 hover:border-emerald-500 dark:hover:border-emerald-600 hover:bg-emerald-50 dark:hover:bg-emerald-950/20 transition-all cursor-pointer text-left"
                >
                  <div className="p-2 bg-emerald-100 dark:bg-emerald-900/30 rounded-lg">
                    <User className="h-5 w-5 text-emerald-700 dark:text-emerald-400" />
                  </div>
                  <div className="flex-1">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-zinc-100 mb-1">
                      Username
                    </h4>
                    <p className="text-xs text-slate-600 dark:text-zinc-500">
                      Group logs by user's login name (e.g., john.doe,
                      sarah.johnson)
                    </p>
                  </div>
                </button>

                <button
                  onClick={() => handleTypeSelect("userId")}
                  className="w-full flex items-start gap-4 p-4 rounded-lg border-2 border-slate-200 dark:border-neutral-800 hover:border-emerald-500 dark:hover:border-emerald-600 hover:bg-emerald-50 dark:hover:bg-emerald-950/20 transition-all cursor-pointer text-left"
                >
                  <div className="p-2 bg-blue-100 dark:bg-blue-900/30 rounded-lg">
                    <Hash className="h-5 w-5 text-blue-700 dark:text-blue-400" />
                  </div>
                  <div className="flex-1">
                    <h4 className="text-sm font-semibold text-slate-900 dark:text-zinc-100 mb-1">
                      User ID
                    </h4>
                    <p className="text-xs text-slate-600 dark:text-zinc-500">
                      Group logs by unique user identifier (e.g., user_001,
                      user_002)
                    </p>
                  </div>
                </button>
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={handleCancel}>
                Cancel
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <div className="flex items-center gap-2">
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleBack}
                  className="h-8 w-8 p-0"
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>
                <div className="flex-1">
                  <DialogTitle className="text-slate-900 dark:text-zinc-100">
                    Select{" "}
                    {selectedType === "userId" ? "User IDs" : "Usernames"}
                  </DialogTitle>
                  <DialogDescription className="text-slate-600 dark:text-zinc-400">
                    Choose one or more{" "}
                    {selectedType === "userId" ? "user IDs" : "usernames"} to
                    filter logs
                  </DialogDescription>
                </div>
              </div>
            </DialogHeader>

            <div className="py-4">
              {/* Search */}
              <div className="relative mb-4">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400 dark:text-zinc-500" />
                <Input
                  placeholder={`Search by ${
                    selectedType === "userId"
                      ? "ID, username, or email"
                      : "username, ID, or email"
                  }...`}
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10 bg-white dark:bg-neutral-800 border-slate-300 dark:border-neutral-600"
                />
              </div>

              {/* Select All / Deselect All */}
              <div className="flex items-center justify-between mb-3">
                <p className="text-sm text-slate-600 dark:text-zinc-400">
                  {selectedValues.length} of {filteredUsers.length} selected
                </p>
                <div className="flex gap-2">
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={handleSelectAll}
                    className="h-8 text-xs"
                  >
                    Select All
                  </Button>
                  {selectedValues.length > 0 && (
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={handleDeselectAll}
                      className="h-8 text-xs"
                    >
                      Deselect All
                    </Button>
                  )}
                </div>
              </div>

              {/* User List */}
              <div className="border border-slate-200 dark:border-neutral-800 rounded-lg max-h-[400px] overflow-y-auto">
                {isLoadingUsers ? (
                  <div className="p-8 text-center">
                    <Loader2 className="h-8 w-8 mx-auto mb-3 text-emerald-600 dark:text-emerald-400 animate-spin" />
                    <p className="text-sm text-slate-600 dark:text-zinc-500">
                      Loading users...
                    </p>
                  </div>
                ) : isErrorUsers ? (
                  <div className="p-8 text-center text-rose-600 dark:text-rose-400">
                    <p className="font-medium mb-2">Failed to load users</p>
                    <p className="text-sm text-slate-600 dark:text-zinc-500">
                      {(usersError as any)?.data?.message ||
                        "Please try again later"}
                    </p>
                  </div>
                ) : filteredUsers.length === 0 ? (
                  <div className="p-8 text-center text-slate-500 dark:text-zinc-500">
                    {searchQuery ? (
                      <>No users found matching "{searchQuery}"</>
                    ) : (
                      <>No users available</>
                    )}
                  </div>
                ) : (
                  <div className="divide-y divide-slate-200 dark:divide-neutral-800">
                    {filteredUsers.map((user) => {
                      const value =
                        selectedType === "userId" ? user.id : user.username;
                      const isSelected = selectedValues.includes(value);

                      return (
                        <div
                          key={user.id}
                          className={`flex items-center gap-3 p-3 hover:bg-slate-50 dark:hover:bg-neutral-800/50 cursor-pointer transition-colors ${
                            isSelected
                              ? "bg-emerald-50 dark:bg-emerald-950/20"
                              : ""
                          }`}
                          onClick={() => handleToggleValue(value)}
                        >
                          <Checkbox
                            checked={isSelected}
                            onCheckedChange={() => handleToggleValue(value)}
                          />
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-2">
                              <p className="text-sm font-medium text-slate-900 dark:text-zinc-100">
                                {selectedType === "userId"
                                  ? user.id
                                  : user.username}
                              </p>
                              {isSelected && (
                                <Badge className="bg-emerald-500 dark:bg-emerald-600 text-white text-xs">
                                  Selected
                                </Badge>
                              )}
                            </div>
                            <p className="text-xs text-slate-600 dark:text-zinc-500 truncate">
                              {user.email}
                            </p>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            </div>

            <DialogFooter>
              <Button variant="outline" onClick={handleCancel}>
                Cancel
              </Button>
              <Button
                onClick={handleApply}
                disabled={selectedValues.length === 0}
                className="bg-emerald-600 hover:bg-emerald-700 text-white disabled:opacity-50"
              >
                Apply ({selectedValues.length})
              </Button>
            </DialogFooter>
          </>
        )}
      </DialogContent>
    </Dialog>
  );
}
