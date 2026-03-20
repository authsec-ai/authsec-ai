import { useState, useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Label } from "@/components/ui/label";
import { SearchableSelect, type SearchableSelectOption } from "@/components/ui/searchable-select";
import { X, Users, UserPlus, Plus } from "lucide-react";
import type { RoleFormData, User as UserType, Group } from "../types";
import { mockUsers, mockGroups } from "../utils/mock-data";

interface AssignmentsPanelProps {
  formData: RoleFormData;
  onUpdate: (data: Partial<RoleFormData>) => void;
}

export function AssignmentsPanel({ formData, onUpdate }: AssignmentsPanelProps) {
  const handleRemoveUser = (userId: string) => {
    onUpdate({ assignedUsers: formData.assignedUsers.filter((id) => id !== userId) });
  };

  const handleRemoveGroup = (groupId: string) => {
    onUpdate({ assignedGroups: formData.assignedGroups.filter((id) => id !== groupId) });
  };

  const selectedUsers = mockUsers.filter((user) => formData.assignedUsers.includes(user.id));
  const selectedGroups = mockGroups.filter((group) => formData.assignedGroups.includes(group.id));

  // Convert users to SearchableSelectOption format
  const userOptions = useMemo<SearchableSelectOption[]>(() => {
    return mockUsers.map((user) => ({
      value: user.id,
      label: user.name,
      description: user.email,
    }));
  }, []);

  // Convert groups to SearchableSelectOption format
  const groupOptions = useMemo<SearchableSelectOption[]>(() => {
    return mockGroups.map((group) => ({
      value: group.id,
      label: group.name,
      description: `${group.memberCount} members`,
    }));
  }, []);

  return (
    <Card className="border rounded-xl bg-muted/30">
      <CardHeader className="pb-4">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10">
            <Users className="h-5 w-5 text-primary" />
          </div>
          <div>
            <CardTitle className="text-xl font-semibold text-foreground">Assignments</CardTitle>
            <p className="text-base text-foreground mt-1">
              Assign users and groups to this role (optional)
            </p>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Users Section */}
        <div className="space-y-3">
          <Label className="text-base font-semibold text-foreground flex items-center gap-2">
            <UserPlus className="h-4 w-4 text-primary" />
            Users
          </Label>
          <SearchableSelect
            multiple
            options={userOptions}
            value={formData.assignedUsers}
            onChange={(ids) => onUpdate({ assignedUsers: ids })}
            placeholder="Select users..."
            searchPlaceholder="Search users..."
            emptyText="No users found"
            maxBadges={5}
            className="h-11"
          />

          {selectedUsers.length === 0 ? (
            <div className="text-center py-4 text-foreground">
              <Users className="mx-auto h-8 w-8 mb-2" />
              <p className="text-sm">No users assigned</p>
            </div>
          ) : (
            <div className="space-y-2">
              {selectedUsers.map((user) => (
                <div
                  key={user.id}
                  className="flex items-center justify-between p-3 border rounded-lg bg-background"
                >
                  <div className="flex items-center gap-3">
                    <Avatar className="h-8 w-8">
                      <AvatarImage src={user.avatar} alt={user.name} />
                      <AvatarFallback className="text-xs">
                        {user.name
                          .split(" ")
                          .map((n) => n[0])
                          .join("")}
                      </AvatarFallback>
                    </Avatar>
                    <div>
                      <div className="text-sm font-medium">{user.name}</div>
                      <div className="text-xs text-foreground">{user.email}</div>
                    </div>
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => handleRemoveUser(user.id)}>
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Groups Section */}
        <div className="space-y-3">
          <Label className="text-base font-semibold text-foreground flex items-center gap-2">
            <Users className="h-4 w-4 text-primary" />
            Groups
          </Label>
          <SearchableSelect
            multiple
            options={groupOptions}
            value={formData.assignedGroups}
            onChange={(ids) => onUpdate({ assignedGroups: ids })}
            placeholder="Select groups..."
            searchPlaceholder="Search groups..."
            emptyText="No groups found"
            maxBadges={5}
            className="h-11"
          />

          {selectedGroups.length === 0 ? (
            <div className="text-center py-4 text-foreground">
              <Users className="mx-auto h-8 w-8 mb-2" />
              <p className="text-sm">No groups assigned</p>
            </div>
          ) : (
            <div className="space-y-2">
              {selectedGroups.map((group) => (
                <div
                  key={group.id}
                  className="flex items-center justify-between p-3 border rounded-lg bg-background"
                >
                  <div className="flex items-center gap-3">
                    <div className="w-8 h-8 bg-muted rounded-full flex items-center justify-center">
                      <Users className="h-4 w-4" />
                    </div>
                    <div>
                      <div className="text-sm font-medium">{group.name}</div>
                      <div className="text-xs text-foreground">
                        {group.memberCount} members
                      </div>
                    </div>
                  </div>
                  <Button variant="ghost" size="sm" onClick={() => handleRemoveGroup(group.id)}>
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
