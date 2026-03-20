import { useState } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { Textarea } from "../../../components/ui/textarea";
import { Checkbox } from "../../../components/ui/checkbox";
import { ScrollArea } from "../../../components/ui/scroll-area";
import { Separator } from "../../../components/ui/separator";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../../components/ui/card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "../../../components/ui/collapsible";
import {
  ChevronRight,
  ChevronLeft,
  Check,
  Users,
  Shield,
  Settings,
  ChevronDown,
  Plus,
  X,
  Search,
  Building,
  Key,
  UserCheck,
  UserPlus,
} from "lucide-react";
import type { Resource, RolePermission } from "../../../types/entities";

// Mock data imports

interface AddRoleWizardProps {
  isOpen: boolean;
  onClose: () => void;
  onRoleCreated: (role: any) => void;
}

interface RoleBasics {
  name: string;
  description: string;
  type: "system" | "custom";
}

interface PermissionBuilder {
  selectedClient: string;
  selectedResource: string;
  permissions: RolePermission[];
}

interface RoleAssignments {
  selectedUsers: string[];
  selectedGroups: string[];
}

const WIZARD_STEPS = [
  {
    id: "basics",
    title: "Role Basics",
    description: "Define the role name and description",
    icon: Settings,
  },
  {
    id: "permissions",
    title: "Permissions",
    description: "Configure resource access and scopes",
    icon: Shield,
  },
  {
    id: "assignments",
    title: "Assignments",
    description: "Assign users and groups to this role",
    icon: Users,
  },
];

export function AddRoleWizard({ isOpen, onClose, onRoleCreated }: AddRoleWizardProps) {
  const [currentStep, setCurrentStep] = useState(0);
  const [roleBasics, setRoleBasics] = useState<RoleBasics>({
    name: "",
    description: "",
    type: "custom",
  });
  const [permissionBuilder, setPermissionBuilder] = useState<PermissionBuilder>({
    selectedClient: "",
    selectedResource: "",
    permissions: [],
  });
  const [roleAssignments, setRoleAssignments] = useState<RoleAssignments>({
    selectedUsers: [],
    selectedGroups: [],
  });

  const [searchUsers, setSearchUsers] = useState("");
  const [searchGroups, setSearchGroups] = useState("");

  const isStepComplete = (stepIndex: number) => {
    switch (stepIndex) {
      case 0:
        return roleBasics.name.trim() !== "";
      case 1:
        return permissionBuilder.permissions.length > 0;
      case 2:
        return true; // Assignments are optional
      default:
        return false;
    }
  };

  const canProceed = () => {
    return isStepComplete(currentStep);
  };

  const handleNext = () => {
    if (currentStep < WIZARD_STEPS.length - 1) {
      setCurrentStep(currentStep + 1);
    }
  };

  const handlePrevious = () => {
    if (currentStep > 0) {
      setCurrentStep(currentStep - 1);
    }
  };

  const handleFinish = () => {
    const newRole = {
      id: `role_${Date.now()}`,
      name: roleBasics.name,
      description: roleBasics.description,
      type: roleBasics.type,
      permissions: permissionBuilder.permissions,
      userIds: roleAssignments.selectedUsers,
      groupIds: roleAssignments.selectedGroups,
      userCount: roleAssignments.selectedUsers.length,
      groupCount: roleAssignments.selectedGroups.length,
      isBuiltIn: false,
      version: 1,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
      createdBy: "admin",
    };

    onRoleCreated(newRole);
    onClose();

    // Reset wizard state
    setCurrentStep(0);
    setRoleBasics({ name: "", description: "", type: "custom" });
    setPermissionBuilder({ selectedClient: "", selectedResource: "", permissions: [] });
    setRoleAssignments({ selectedUsers: [], selectedGroups: [] });
  };

  const handleClose = () => {
    onClose();
    // Reset wizard state
    setCurrentStep(0);
    setRoleBasics({ name: "", description: "", type: "custom" });
    setPermissionBuilder({ selectedClient: "", selectedResource: "", permissions: [] });
    setRoleAssignments({ selectedUsers: [], selectedGroups: [] });
  };

  const addPermission = (resource: Resource, selectedScopes: string[]) => {
    const newPermission: RolePermission = {
      resourceId: resource.id,
      resourceName: resource.name,
      scopes: selectedScopes,
      scopeNames: selectedScopes,
    };

    setPermissionBuilder((prev) => ({
      ...prev,
      permissions: [...prev.permissions.filter((p) => p.resourceId !== resource.id), newPermission],
    }));
  };

  const removePermission = (resourceId: string) => {
    setPermissionBuilder((prev) => ({
      ...prev,
      permissions: prev.permissions.filter((p) => p.resourceId !== resourceId),
    }));
  };

  const toggleUserSelection = (userId: string) => {
    setRoleAssignments((prev) => ({
      ...prev,
      selectedUsers: prev.selectedUsers.includes(userId)
        ? prev.selectedUsers.filter((id) => id !== userId)
        : [...prev.selectedUsers, userId],
    }));
  };

  const toggleGroupSelection = (groupId: string) => {
    setRoleAssignments((prev) => ({
      ...prev,
      selectedGroups: prev.selectedGroups.includes(groupId)
        ? prev.selectedGroups.filter((id) => id !== groupId)
        : [...prev.selectedGroups, groupId],
    }));
  };

  const filteredUsers = [].filter(
    (user: any) =>
      user.name.toLowerCase().includes(searchUsers.toLowerCase()) ||
      user.email.toLowerCase().includes(searchUsers.toLowerCase())
  );

  const filteredGroups = [].filter((group) =>
    group.name.toLowerCase().includes(searchGroups.toLowerCase())
  );

  const renderStepContent = () => {
    switch (currentStep) {
      case 0:
        return (
          <div className="space-y-6">
            <div className="text-center">
              <div className="mx-auto w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center mb-4">
                <Settings className="h-8 w-8 text-blue-600" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Role Basics</h3>
              <p className="text-foreground">
                Define the fundamental properties of your new role
              </p>
            </div>

            <div className="space-y-4">
              <div className="grid grid-cols-1 gap-4">
                <div>
                  <Label htmlFor="role-name">Role Name*</Label>
                  <Input
                    id="role-name"
                    value={roleBasics.name}
                    onChange={(e) => setRoleBasics((prev) => ({ ...prev, name: e.target.value }))}
                    placeholder="e.g., Content Manager, Data Analyst"
                    className="mt-1"
                  />
                </div>

                <div>
                  <Label htmlFor="role-description">Description</Label>
                  <Textarea
                    id="role-description"
                    value={roleBasics.description}
                    onChange={(e) =>
                      setRoleBasics((prev) => ({ ...prev, description: e.target.value }))
                    }
                    placeholder="Describe the role's purpose and responsibilities"
                    className="mt-1"
                    rows={3}
                  />
                </div>

                <div>
                  <Label htmlFor="role-type">Role Type</Label>
                  <Select
                    value={roleBasics.type}
                    onValueChange={(value: "system" | "custom") =>
                      setRoleBasics((prev) => ({ ...prev, type: value }))
                    }
                  >
                    <SelectTrigger className="mt-1">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="custom">Custom Role</SelectItem>
                      <SelectItem value="system">System Role</SelectItem>
                    </SelectContent>
                  </Select>
                  <p className="text-sm text-foreground mt-1">
                    System roles have elevated privileges and are managed by administrators
                  </p>
                </div>
              </div>
            </div>
          </div>
        );

      case 1:
        return (
          <div className="space-y-6">
            <div className="text-center">
              <div className="mx-auto w-16 h-16 bg-green-100 rounded-full flex items-center justify-center mb-4">
                <Shield className="h-8 w-8 text-green-600" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Configure Permissions</h3>
              <p className="text-foreground">Select resources and define the access scopes</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Resource Selection */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="flex items-center gap-2">
                    <Building className="h-4 w-4" />
                    Available Resources
                  </CardTitle>
                  <CardDescription>
                    Select resources and configure their access scopes
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ScrollArea className="h-[300px]">
                    <div className="space-y-3">
                      {[].map((resource) => (
                        <ResourcePermissionCard
                          key={resource.id}
                          resource={resource}
                          onPermissionChange={addPermission}
                          existingPermission={permissionBuilder.permissions.find(
                            (p) => p.resourceId === resource.id
                          )}
                        />
                      ))}
                    </div>
                  </ScrollArea>
                </CardContent>
              </Card>

              {/* Selected Permissions */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="flex items-center gap-2">
                    <Key className="h-4 w-4" />
                    Selected Permissions
                  </CardTitle>
                  <CardDescription>
                    {permissionBuilder.permissions.length} permission
                    {permissionBuilder.permissions.length !== 1 ? "s" : ""} configured
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <ScrollArea className="h-[300px]">
                    <div className="space-y-3">
                      {permissionBuilder.permissions.length === 0 ? (
                        <div className="text-center py-8 text-foreground">
                          <Shield className="h-12 w-12 mx-auto mb-2 opacity-50" />
                          <p>No permissions selected yet</p>
                          <p className="text-sm">Configure resource access on the left</p>
                        </div>
                      ) : (
                        permissionBuilder.permissions.map((permission) => (
                          <div key={permission.resourceId} className="border rounded-lg p-3">
                            <div className="flex items-start justify-between">
                              <div className="flex-1">
                                <div className="flex items-center gap-2 mb-2">
                                  <h4 className="font-medium">{permission.resourceName}</h4>
                                  <Badge variant="secondary" className="text-xs">
                                    {permission.scopes.length} scope
                                    {permission.scopes.length !== 1 ? "s" : ""}
                                  </Badge>
                                </div>
                                <div className="flex flex-wrap gap-1">
                                  {permission.scopes.map((scope) => (
                                    <Badge key={scope} variant="outline" className="text-xs">
                                      {scope}
                                    </Badge>
                                  ))}
                                </div>
                              </div>
                              <Button
                                variant="ghost"
                                size="sm"
                                onClick={() => removePermission(permission.resourceId)}
                                className="h-8 w-8 p-0 text-destructive hover:text-destructive"
                              >
                                <X className="h-4 w-4" />
                              </Button>
                            </div>
                          </div>
                        ))
                      )}
                    </div>
                  </ScrollArea>
                </CardContent>
              </Card>
            </div>
          </div>
        );

      case 2:
        return (
          <div className="space-y-6">
            <div className="text-center">
              <div className="mx-auto w-16 h-16 bg-blue-100 rounded-full flex items-center justify-center mb-4">
                <Users className="h-8 w-8 text-blue-600" />
              </div>
              <h3 className="text-lg font-semibold mb-2">Assign Users & Groups</h3>
              <p className="text-foreground">Choose who will receive this role</p>
            </div>

            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              {/* Users */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="flex items-center gap-2">
                    <UserCheck className="h-4 w-4" />
                    Users
                  </CardTitle>
                  <CardDescription>
                    {roleAssignments.selectedUsers.length} user
                    {roleAssignments.selectedUsers.length !== 1 ? "s" : ""} selected
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-foreground" />
                      <Input
                        placeholder="Search users..."
                        value={searchUsers}
                        onChange={(e) => setSearchUsers(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                    <ScrollArea className="h-[250px]">
                      <div className="space-y-2">
                        {filteredUsers.map((user: any) => (
                          <div
                            key={user.id}
                            className={`flex items-center space-x-3 p-2 rounded-lg cursor-pointer transition-colors ${
                              roleAssignments.selectedUsers.includes(user.id)
                                ? "bg-primary/10 border border-primary/20"
                                : "hover:bg-muted"
                            }`}
                            onClick={() => toggleUserSelection(user.id)}
                          >
                            <Checkbox
                              checked={roleAssignments.selectedUsers.includes(user.id)}
                              onChange={() => toggleUserSelection(user.id)}
                            />
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <p className="font-medium text-sm">{user.name}</p>
                                <Badge variant="outline" className="text-xs">
                                  {user.status}
                                </Badge>
                              </div>
                              <p className="text-xs text-foreground truncate">{user.email}</p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </ScrollArea>
                  </div>
                </CardContent>
              </Card>

              {/* Groups */}
              <Card>
                <CardHeader className="pb-3">
                  <CardTitle className="flex items-center gap-2">
                    <UserPlus className="h-4 w-4" />
                    Groups
                  </CardTitle>
                  <CardDescription>
                    {roleAssignments.selectedGroups.length} group
                    {roleAssignments.selectedGroups.length !== 1 ? "s" : ""} selected
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-3">
                    <div className="relative">
                      <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-foreground" />
                      <Input
                        placeholder="Search groups..."
                        value={searchGroups}
                        onChange={(e) => setSearchGroups(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                    <ScrollArea className="h-[250px]">
                      <div className="space-y-2">
                        {filteredGroups.map((group) => (
                          <div
                            key={group.id}
                            className={`flex items-center space-x-3 p-2 rounded-lg cursor-pointer transition-colors ${
                              roleAssignments.selectedGroups.includes(group.id)
                                ? "bg-primary/10 border border-primary/20"
                                : "hover:bg-muted"
                            }`}
                            onClick={() => toggleGroupSelection(group.id)}
                          >
                            <Checkbox
                              checked={roleAssignments.selectedGroups.includes(group.id)}
                              onChange={() => toggleGroupSelection(group.id)}
                            />
                            <div className="flex-1 min-w-0">
                              <div className="flex items-center gap-2">
                                <p className="font-medium text-sm">{group.name}</p>
                                {group.isSystem && (
                                  <Badge variant="outline" className="text-xs">
                                    System
                                  </Badge>
                                )}
                              </div>
                              <p className="text-xs text-foreground">
                                {group.memberCount} member{group.memberCount !== 1 ? "s" : ""}
                              </p>
                            </div>
                          </div>
                        ))}
                      </div>
                    </ScrollArea>
                  </div>
                </CardContent>
              </Card>
            </div>
          </div>
        );

      default:
        return null;
    }
  };

  return (
    <Dialog open={isOpen} onOpenChange={handleClose}>
      <DialogContent className="  h-[85vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Plus className="h-5 w-5" />
            Create New Role
          </DialogTitle>
          <DialogDescription>
            Follow the steps to create a new role with permissions and assignments
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col h-full overflow-hidden">
          {/* Progress Steps */}
          <div className="flex items-center justify-between mb-6 px-2">
            {WIZARD_STEPS.map((step, index) => {
              const isActive = index === currentStep;
              const isCompleted =
                index < currentStep || (index === currentStep && isStepComplete(index));
              const StepIcon = step.icon;

              return (
                <div key={step.id} className="flex items-center">
                  <div
                    className={`flex items-center justify-center w-10 h-10 rounded-full border-2 ${
                      isActive
                        ? "border-primary bg-primary text-primary-foreground"
                        : isCompleted
                        ? "border-green-500 bg-green-500 text-white"
                        : "border-muted-foreground bg-muted text-foreground"
                    }`}
                  >
                    {isCompleted && index !== currentStep ? (
                      <Check className="h-5 w-5" />
                    ) : (
                      <StepIcon className="h-5 w-5" />
                    )}
                  </div>
                  <div className="ml-3">
                    <div
                      className={`font-medium ${
                        isActive ? "text-foreground" : "text-foreground"
                      }`}
                    >
                      {step.title}
                    </div>
                    <div className="text-sm text-foreground">{step.description}</div>
                  </div>
                  {index < WIZARD_STEPS.length - 1 && (
                    <div className="flex-1 mx-4 h-px bg-border" />
                  )}
                </div>
              );
            })}
          </div>

          {/* Step Content */}
          <div className="flex-1 overflow-hidden">
            <ScrollArea className="h-full">
              <div className="p-1">{renderStepContent()}</div>
            </ScrollArea>
          </div>

          {/* Footer */}
          <Separator />
          <DialogFooter className="flex items-center justify-between px-6 py-4">
            <div className="flex items-center gap-2">
              <span className="text-sm text-foreground">
                Step {currentStep + 1} of {WIZARD_STEPS.length}
              </span>
            </div>
            <div className="flex items-center gap-2">
              <Button variant="outline" onClick={handlePrevious} disabled={currentStep === 0}>
                <ChevronLeft className="h-4 w-4 mr-2" />
                Previous
              </Button>
              {currentStep === WIZARD_STEPS.length - 1 ? (
                <Button onClick={handleFinish} disabled={!canProceed()}>
                  <Check className="h-4 w-4 mr-2" />
                  Create Role
                </Button>
              ) : (
                <Button onClick={handleNext} disabled={!canProceed()}>
                  Next
                  <ChevronRight className="h-4 w-4 ml-2" />
                </Button>
              )}
            </div>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}

// Helper component for resource permission configuration
function ResourcePermissionCard({
  resource,
  onPermissionChange,
  existingPermission,
}: {
  resource: Resource;
  onPermissionChange: (resource: Resource, scopes: string[]) => void;
  existingPermission?: RolePermission;
}) {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedScopes, setSelectedScopes] = useState<string[]>(existingPermission?.scopes || []);

  const handleScopeToggle = (scopeName: string) => {
    const newScopes = selectedScopes.includes(scopeName)
      ? selectedScopes.filter((s) => s !== scopeName)
      : [...selectedScopes, scopeName];

    setSelectedScopes(newScopes);
    if (newScopes.length > 0) {
      onPermissionChange(resource, newScopes);
    }
  };

  return (
    <Collapsible open={isOpen} onOpenChange={setIsOpen}>
      <CollapsibleTrigger asChild>
        <div className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted cursor-pointer">
          <div className="flex items-center gap-3">
            <div className="flex items-center gap-2">
              <Building className="h-4 w-4 text-foreground" />
              <div>
                <h4 className="font-medium">{resource.name}</h4>
                <p className="text-xs text-foreground">{resource.clientName}</p>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {existingPermission && (
              <Badge variant="secondary" className="text-xs">
                {existingPermission.scopes.length} scope
                {existingPermission.scopes.length !== 1 ? "s" : ""}
              </Badge>
            )}
            <ChevronDown className={`h-4 w-4 transition-transform ${isOpen ? "rotate-180" : ""}`} />
          </div>
        </div>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <div className="p-3 border-l border-r border-b rounded-b-lg bg-muted/50">
          <div className="space-y-2">
            <p className="text-sm font-medium">Available Scopes:</p>
            <div className="grid grid-cols-1 gap-2">
              {resource.scopes.map((scope) => (
                <div
                  key={scope.id}
                  className="flex items-center space-x-2 p-2 rounded hover:bg-background"
                >
                  <Checkbox
                    id={`scope-${scope.id}`}
                    checked={selectedScopes.includes(scope.name)}
                    onCheckedChange={() => handleScopeToggle(scope.name)}
                  />
                  <label htmlFor={`scope-${scope.id}`} className="flex-1 cursor-pointer">
                    <div className="flex items-center justify-between">
                      <span className="font-medium text-sm">{scope.name}</span>
                      {scope.isDeprecated && (
                        <Badge variant="destructive" className="text-xs">
                          Deprecated
                        </Badge>
                      )}
                    </div>
                    {scope.description && (
                      <p className="text-xs text-foreground mt-1">{scope.description}</p>
                    )}
                  </label>
                </div>
              ))}
            </div>
          </div>
        </div>
      </CollapsibleContent>
    </Collapsible>
  );
}
