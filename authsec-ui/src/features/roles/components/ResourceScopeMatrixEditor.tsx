import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../../components/ui/dialog";
import { Button } from "../../../components/ui/button";
import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { Checkbox } from "../../../components/ui/checkbox";
import { Badge } from "../../../components/ui/badge";
import { Separator } from "../../../components/ui/separator";
import {
  Search,
  Save,
  Key,
  Shield,
  Lock,
  Unlock,
  CheckCircle,
  AlertCircle,
  CloudCog,
} from "lucide-react";
import { useGetExternalServicesQuery } from "@/app/api/externalServiceApi";
import {
  injectExternalServicesIntoResources,
  isExternalServiceResource,
} from "@/features/external-services/utils/external-role-utils";
import type { EnhancedRole, RolePermission } from "../../../types/entities";

interface ResourceScopeMatrixEditorProps {
  isOpen: boolean;
  onClose: () => void;
  role: EnhancedRole | null;
  onSave: (permissions: RolePermission[]) => void;
}

export function ResourceScopeMatrixEditor({
  isOpen,
  onClose,
  role,
  onSave,
}: ResourceScopeMatrixEditorProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const [permissions, setPermissions] = useState<RolePermission[]>([]);
  const [selectedResource, setSelectedResource] = useState<string | null>(null);
  const [showExternalServices, setShowExternalServices] = useState(true);

  // Fetch external services
  const { data: externalServices = [] } = useGetExternalServicesQuery();

  // Combine internal and external resources
  const allResources = injectExternalServicesIntoResources([], externalServices);

  useEffect(() => {
    if (role) {
      setPermissions(role.permissions);
    }
  }, [role]);

  const filteredResources = allResources.filter(
    (resource) =>
      (showExternalServices || !resource.isExternal) &&
      (resource.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        resource.description?.toLowerCase().includes(searchQuery.toLowerCase()) ||
        false)
  );

  const getResourcePermission = (resourceId: string): RolePermission | undefined => {
    return permissions.find((p) => p.resourceId === resourceId);
  };

  const toggleScope = (
    resourceId: string,
    resourceName: string,
    scope: string,
    scopeName: string
  ) => {
    setPermissions((prev) => {
      const existing = prev.find((p) => p.resourceId === resourceId);

      if (existing) {
        const scopeIndex = existing.scopes.indexOf(scope);
        if (scopeIndex >= 0) {
          // Remove scope
          const newScopes = [...existing.scopes];
          const newScopeNames = [...(existing.scopeNames || [])];
          newScopes.splice(scopeIndex, 1);
          newScopeNames.splice(scopeIndex, 1);

          if (newScopes.length === 0) {
            // Remove entire permission if no scopes left
            return prev.filter((p) => p.resourceId !== resourceId);
          } else {
            // Update permission
            return prev.map((p) =>
              p.resourceId === resourceId
                ? { ...p, scopes: newScopes, scopeNames: newScopeNames }
                : p
            );
          }
        } else {
          // Add scope
          return prev.map((p) =>
            p.resourceId === resourceId
              ? {
                  ...p,
                  scopes: [...p.scopes, scope],
                  scopeNames: [...(p.scopeNames || []), scopeName],
                }
              : p
          );
        }
      } else {
        // Create new permission
        return [
          ...prev,
          {
            resourceId,
            resourceName,
            scopes: [scope],
            scopeNames: [scopeName],
            isExternal: isExternalServiceResource(resourceId),
          },
        ];
      }
    });
  };

  const toggleAllScopes = (
    resourceId: string,
    resourceName: string,
    scopes: string[],
    scopeNames: string[]
  ) => {
    const existing = getResourcePermission(resourceId);
    const hasAllScopes = existing && existing.scopes.length === scopes.length;

    if (hasAllScopes) {
      // Remove all scopes
      setPermissions((prev) => prev.filter((p) => p.resourceId !== resourceId));
    } else {
      // Add all scopes
      setPermissions((prev) => {
        const filtered = prev.filter((p) => p.resourceId !== resourceId);
        return [
          ...filtered,
          {
            resourceId,
            resourceName,
            scopes: [...scopes],
            scopeNames: [...scopeNames],
            isExternal: isExternalServiceResource(resourceId),
          },
        ];
      });
    }
  };

  const getTotalScopeCount = () => {
    return permissions.reduce((sum, p) => sum + p.scopes.length, 0);
  };

  const getPermissionSeverity = (scope: string) => {
    if (scope.includes("admin") || scope.includes("delete")) return "high";
    if (scope.includes("write") || scope.includes("edit")) return "medium";
    return "low";
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case "high":
        return "text-red-600 bg-red-50 border-red-200";
      case "medium":
        return "text-orange-600 bg-orange-50 border-orange-200";
      default:
        return "text-green-600 bg-green-50 border-green-200";
    }
  };

  const handleSave = () => {
    onSave(permissions);
    onClose();
  };

  if (!role) return null;

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="  h-[80vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Key className="h-5 w-5" />
            Edit Permissions for "{role.name}"
          </DialogTitle>
          <DialogDescription>
            Configure resource-scope permissions for this role. Changes will create a new version.
          </DialogDescription>
        </DialogHeader>

        <div className="flex h-full gap-6 overflow-hidden">
          {/* Resource List */}
          <div className="w-1/3 flex flex-col border-r pr-4">
            <div className="space-y-4 mb-4">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground" />
                <Input
                  placeholder="Search resources..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-9"
                />
              </div>

              <div className="flex items-center justify-between">
                <Label className="text-sm font-medium">
                  Resources ({filteredResources.length})
                </Label>
                <Badge variant="outline" className="text-xs">
                  {permissions.length} configured
                </Badge>
              </div>

              <div className="flex items-center space-x-2">
                <Checkbox
                  id="show-external"
                  checked={showExternalServices}
                  onCheckedChange={(checked) => setShowExternalServices(!!checked)}
                />
                <Label htmlFor="show-external" className="text-sm flex items-center gap-1">
                  <CloudCog className="h-3 w-3" />
                  Include External services and secrets management
                </Label>
              </div>
            </div>

            <div className="flex-1 overflow-auto space-y-2">
              {filteredResources.map((resource) => {
                const permission = getResourcePermission(resource.id);
                const hasPermissions = !!permission;
                const scopeCount = permission?.scopes.length || 0;

                return (
                  <div
                    key={resource.id}
                    className={`p-3 border rounded-lg cursor-pointer transition-all ${
                      selectedResource === resource.id
                        ? "border-primary bg-primary/5"
                        : hasPermissions
                        ? "border-green-200 bg-green-50"
                        : resource.isExternal
                        ? "border-blue-200 bg-blue-50/30"
                        : "border-border hover:border-muted-foreground/50"
                    }`}
                    onClick={() => setSelectedResource(resource.id)}
                  >
                    <div className="flex items-start justify-between">
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm truncate flex items-center gap-1">
                          {resource.name}
                          {resource.isExternal && <CloudCog className="h-3 w-3 text-blue-600" />}
                        </div>
                        <div className="text-xs text-foreground truncate">
                          {resource.description}
                        </div>
                        {hasPermissions && (
                          <div className="flex items-center gap-1 mt-1">
                            <CheckCircle className="h-3 w-3 text-green-600" />
                            <span className="text-xs text-green-600">
                              {scopeCount} scope{scopeCount !== 1 ? "s" : ""}
                            </span>
                          </div>
                        )}
                      </div>
                      <div className="ml-2 flex-shrink-0">
                        <Badge
                          variant={hasPermissions ? "default" : "secondary"}
                          className="text-xs"
                        >
                          {resource.scopes.length}
                        </Badge>
                      </div>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <div className="flex-1 flex flex-col">
            {selectedResource ? (
              (() => {
                const resource = filteredResources.find((r) => r.id === selectedResource);
                if (!resource) return null;

                const permission = getResourcePermission(selectedResource);
                const hasAllScopes =
                  permission && permission.scopes.length === resource.scopes.length;

                return (
                  <div className="flex flex-col h-full">
                    <div className="space-y-4 mb-4">
                      <div>
                        <h3 className="font-medium flex items-center gap-2">
                          {resource.isExternal ? (
                            <CloudCog className="h-4 w-4 text-blue-600" />
                          ) : (
                            <Shield className="h-4 w-4" />
                          )}
                          {resource.name}
                          {resource.isExternal && (
                            <Badge
                              variant="outline"
                              className="text-xs bg-blue-50 text-blue-600 border-blue-200"
                            >
                              External Service
                            </Badge>
                          )}
                        </h3>
                        <p className="text-sm text-foreground">{resource.description}</p>
                      </div>

                      <div className="flex items-center justify-between">
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="text-xs">
                            {resource.scopes.length} available scopes
                          </Badge>
                          {permission && (
                            <Badge variant="default" className="text-xs">
                              {permission.scopes.length} granted
                            </Badge>
                          )}
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            toggleAllScopes(
                              resource.id,
                              resource.name,
                              resource.scopes.map((s) => s.id),
                              resource.scopes.map((s) => s.name)
                            )
                          }
                        >
                          {hasAllScopes ? (
                            <>
                              <Lock className="h-3 w-3 mr-1" /> Revoke All
                            </>
                          ) : (
                            <>
                              <Unlock className="h-3 w-3 mr-1" /> Grant All
                            </>
                          )}
                        </Button>
                      </div>
                    </div>

                    <div className="flex-1 overflow-auto">
                      <div className="grid grid-cols-1 gap-3">
                        {resource.scopes.map((scope) => {
                          const isSelected = permission?.scopes.includes(scope.id) || false;
                          const severity = getPermissionSeverity(scope.id);
                          const severityColor = getSeverityColor(severity);

                          return (
                            <div
                              key={scope.id}
                              className={`p-3 border rounded-lg transition-all ${
                                isSelected ? "border-primary bg-primary/5" : "border-border"
                              }`}
                            >
                              <div className="flex items-start gap-3">
                                <Checkbox
                                  checked={isSelected}
                                  onCheckedChange={() =>
                                    toggleScope(resource.id, resource.name, scope.id, scope.name)
                                  }
                                  className="mt-1"
                                />
                                <div className="flex-1 min-w-0">
                                  <div className="flex items-center gap-2">
                                    <span className="font-medium text-sm">{scope.name}</span>
                                    <Badge className={`text-xs ${severityColor}`}>{severity}</Badge>
                                  </div>
                                  <p className="text-xs text-foreground mt-1">
                                    {scope.description}
                                  </p>
                                  <div className="text-xs text-foreground mt-1 font-mono">
                                    {scope.id}
                                  </div>
                                </div>
                              </div>
                            </div>
                          );
                        })}
                      </div>
                    </div>
                  </div>
                );
              })()
            ) : (
              <div className="flex-1 flex items-center justify-center text-center">
                <div className="space-y-3">
                  <Shield className="h-12 w-12 text-foreground mx-auto" />
                  <div>
                    <h3 className="font-medium">Select a Resource</h3>
                    <p className="text-sm text-foreground">
                      Choose a resource from the left to configure its scopes
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        <Separator />

        <div className="flex items-center justify-between">
          <div className="flex items-center gap-4 text-sm text-foreground">
            <div className="flex items-center gap-1">
              <Key className="h-4 w-4" />
              <span>{permissions.length} resources configured</span>
            </div>
            <div className="flex items-center gap-1">
              <CheckCircle className="h-4 w-4" />
              <span>{getTotalScopeCount()} total scopes</span>
            </div>
            {permissions.filter((p) => p.isExternal).length > 0 && (
              <div className="flex items-center gap-1">
                <CloudCog className="h-4 w-4 text-blue-600" />
                <span>{permissions.filter((p) => p.isExternal).length} external services</span>
              </div>
            )}
            {getTotalScopeCount() > 0 && (
              <div className="flex items-center gap-1">
                <AlertCircle className="h-4 w-4 text-amber-500" />
                <span>Will create version {role.version + 1}</span>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={handleSave} disabled={permissions.length === 0}>
              <Save className="h-4 w-4 mr-2" />
              Save Permissions
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  );
}
