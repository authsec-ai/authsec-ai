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
import { Separator } from "../../../components/ui/separator";
import { ScrollArea } from "../../../components/ui/scroll-area";
import {
  History,
  GitBranch,
  User,
  Calendar,
  RotateCcw,
  Eye,
  ChevronDown,
  ChevronRight,
  Plus,
  Minus,
  ArrowRight,
} from "lucide-react";
import type { EnhancedRole, RoleVersion, RolePermission } from "../../../types/entities";

interface VersionHistoryModalProps {
  isOpen: boolean;
  onClose: () => void;
  role: EnhancedRole | null;
  onRollback: (version: number) => void;
}

// Mock version history data - in real app this would come from API
const getMockVersionHistory = (role: EnhancedRole): RoleVersion[] => {
  const baseVersions: RoleVersion[] = [
    {
      version: role.version,
      timestamp: role.updatedAt,
      author: role.createdBy || "system",
      changes: ["Current version"],
      changeType: "update",
      permissions: role.permissions,
      changedBy: role.createdBy || "system",
      changedAt: role.updatedAt,
    },
  ];

  // Generate previous versions
  if (role.version > 1) {
    baseVersions.unshift({
      version: role.version - 1,
      timestamp: "2024-01-20T10:30:00Z",
      author: "admin",
      changes: [
        "Added Analytics API read permissions",
        "Removed User Management delete permissions",
      ],
      changeType: "permissions",
      permissions: role.permissions.slice(0, -1), // Simulate different permissions
      changedBy: "admin",
      changedAt: "2024-01-20T10:30:00Z",
    });
  }

  if (role.version > 2) {
    baseVersions.unshift({
      version: role.version - 2,
      timestamp: "2024-01-15T14:15:00Z",
      author: "admin",
      changes: ["Updated role description", "Added File Storage write permissions"],
      changeType: "update",
      permissions: role.permissions.slice(0, -2),
      changedBy: "admin",
      changedAt: "2024-01-15T14:15:00Z",
    });
  }

  return baseVersions.reverse();
};

export function VersionHistoryModal({
  isOpen,
  onClose,
  role,
  onRollback,
}: VersionHistoryModalProps) {
  const [selectedVersion, setSelectedVersion] = useState<number | null>(null);
  const [compareMode, setCompareMode] = useState(false);
  const [compareVersion, setCompareVersion] = useState<number | null>(null);

  if (!role) return null;

  const versions = getMockVersionHistory(role);
  const currentVersion = versions.find((v) => v.version === role.version);
  const selectedVersionData = selectedVersion
    ? versions.find((v) => v.version === selectedVersion)
    : null;
  const compareVersionData = compareVersion
    ? versions.find((v) => v.version === compareVersion)
    : null;

  const getChangeTypeColor = (changeType: string) => {
    switch (changeType) {
      case "create":
        return "bg-green-100 text-green-800 border-green-200";
      case "permissions":
        return "bg-blue-100 text-blue-800 border-blue-200";
      case "update":
        return "bg-orange-100 text-orange-800 border-orange-200";
      case "rollback":
        return "bg-blue-100 text-blue-800 border-blue-200";
      default:
        return "bg-gray-100 text-gray-800 border-gray-200";
    }
  };

  const getPermissionDiff = (oldPerms: RolePermission[], newPerms: RolePermission[]) => {
    const oldPermMap = new Map(oldPerms.map((p) => [p.resourceId, p.scopes]));
    const newPermMap = new Map(newPerms.map((p) => [p.resourceId, p.scopes]));

    const added: string[] = [];
    const removed: string[] = [];
    const modified: string[] = [];

    // Check for additions and modifications
    newPerms.forEach((newPerm) => {
      const oldScopes = oldPermMap.get(newPerm.resourceId) || [];
      const newScopes = newPerm.scopes;

      newScopes.forEach((scope: string) => {
        if (!oldScopes.includes(scope)) {
          added.push(`${newPerm.resourceName}: ${scope}`);
        }
      });
    });

    // Check for removals
    oldPerms.forEach((oldPerm) => {
      const newScopes = newPermMap.get(oldPerm.resourceId) || [];
      const oldScopes = oldPerm.scopes;

      oldScopes.forEach((scope: string) => {
        if (!newScopes.includes(scope)) {
          removed.push(`${oldPerm.resourceName}: ${scope}`);
        }
      });
    });

    return { added, removed, modified };
  };

  const handleRollback = (version: number) => {
    onRollback(version);
    onClose();
  };

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className="max-w-5xl h-[80vh] overflow-hidden">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <History className="h-5 w-5" />
            Version History - "{role.name}"
          </DialogTitle>
          <DialogDescription>
            View version history, compare changes, and rollback to previous versions.
          </DialogDescription>
        </DialogHeader>

        <div className="flex h-full gap-6 overflow-hidden">
          {/* Version List */}
          <div className="w-1/3 flex flex-col border-r pr-4">
            <div className="mb-4">
              <div className="flex items-center justify-between mb-2">
                <h3 className="font-medium">Versions ({versions.length})</h3>
                <Button variant="outline" size="sm" onClick={() => setCompareMode(!compareMode)}>
                  {compareMode ? "Exit Compare" : "Compare"}
                </Button>
              </div>
              {compareMode && (
                <p className="text-xs text-foreground">Select two versions to compare</p>
              )}
            </div>

            <ScrollArea className="flex-1">
              <div className="space-y-3">
                {versions.map((version) => {
                  const isSelected = selectedVersion === version.version;
                  const isCompared = compareVersion === version.version;
                  const isCurrent = version.version === role.version;

                  return (
                    <div
                      key={version.version}
                      className={`p-3 border rounded-lg cursor-pointer transition-all ${
                        isSelected
                          ? "border-primary bg-primary/5"
                          : isCompared
                          ? "border-blue-500 bg-blue-50"
                          : "border-border hover:border-muted-foreground/50"
                      }`}
                      onClick={() => {
                        if (compareMode) {
                          if (compareVersion === version.version) {
                            setCompareVersion(null);
                          } else {
                            setCompareVersion(version.version);
                          }
                        } else {
                          setSelectedVersion(version.version);
                        }
                      }}
                    >
                      <div className="space-y-2">
                        <div className="flex items-center justify-between">
                          <div className="flex items-center gap-2">
                            <Badge variant="outline" className="text-xs">
                              v{version.version}
                            </Badge>
                            {isCurrent && (
                              <Badge className="text-xs bg-green-100 text-green-800">Current</Badge>
                            )}
                          </div>
                          <Badge className={`text-xs ${getChangeTypeColor(version.changeType)}`}>
                            {version.changeType}
                          </Badge>
                        </div>

                        <div className="text-sm space-y-1">
                          <div className="flex items-center gap-1 text-foreground">
                            <User className="h-3 w-3" />
                            <span className="text-xs">{version.author}</span>
                          </div>
                          <div className="flex items-center gap-1 text-foreground">
                            <Calendar className="h-3 w-3" />
                            <span className="text-xs">
                              {new Date(version.timestamp).toLocaleString()}
                            </span>
                          </div>
                        </div>

                        <div className="text-xs">
                          {version.changes.slice(0, 2).map((change, idx) => (
                            <div key={idx} className="text-foreground truncate">
                              • {change}
                            </div>
                          ))}
                          {version.changes.length > 2 && (
                            <div className="text-foreground">
                              ... +{version.changes.length - 2} more
                            </div>
                          )}
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            </ScrollArea>
          </div>

          {/* Version Details / Comparison */}
          <div className="flex-1 flex flex-col">
            {compareMode && selectedVersion && compareVersion ? (
              // Comparison View
              <div className="flex flex-col h-full">
                <div className="mb-4">
                  <h3 className="font-medium flex items-center gap-2 mb-2">
                    <GitBranch className="h-4 w-4" />
                    Comparing v{Math.min(selectedVersion, compareVersion)}
                    <ArrowRight className="h-4 w-4" />v{Math.max(selectedVersion, compareVersion)}
                  </h3>
                </div>

                <ScrollArea className="flex-1">
                  {(() => {
                    const oldVersion =
                      selectedVersion < compareVersion ? selectedVersionData : compareVersionData;
                    const newVersion =
                      selectedVersion < compareVersion ? compareVersionData : selectedVersionData;

                    if (!oldVersion || !newVersion) return null;

                    const diff = getPermissionDiff(oldVersion.permissions, newVersion.permissions);

                    return (
                      <div className="space-y-4">
                        {diff.added.length > 0 && (
                          <div className="border rounded-lg p-3 bg-green-50 border-green-200">
                            <h4 className="font-medium text-green-800 flex items-center gap-2 mb-2">
                              <Plus className="h-4 w-4" />
                              Added Permissions ({diff.added.length})
                            </h4>
                            <div className="space-y-1">
                              {diff.added.map((perm, idx) => (
                                <div key={idx} className="text-sm text-green-700 font-mono">
                                  + {perm}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {diff.removed.length > 0 && (
                          <div className="border rounded-lg p-3 bg-red-50 border-red-200">
                            <h4 className="font-medium text-red-800 flex items-center gap-2 mb-2">
                              <Minus className="h-4 w-4" />
                              Removed Permissions ({diff.removed.length})
                            </h4>
                            <div className="space-y-1">
                              {diff.removed.map((perm, idx) => (
                                <div key={idx} className="text-sm text-red-700 font-mono">
                                  - {perm}
                                </div>
                              ))}
                            </div>
                          </div>
                        )}

                        {diff.added.length === 0 && diff.removed.length === 0 && (
                          <div className="text-center text-foreground py-8">
                            <GitBranch className="h-8 w-8 mx-auto mb-2" />
                            <p>No permission changes between these versions</p>
                          </div>
                        )}
                      </div>
                    );
                  })()}
                </ScrollArea>
              </div>
            ) : selectedVersionData ? (
              // Single Version View
              <div className="flex flex-col h-full">
                <div className="mb-4">
                  <div className="flex items-center justify-between mb-2">
                    <h3 className="font-medium flex items-center gap-2">
                      <Eye className="h-4 w-4" />
                      Version {selectedVersionData.version} Details
                    </h3>
                    {selectedVersionData.version !== role.version && (
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => handleRollback(selectedVersionData.version)}
                      >
                        <RotateCcw className="h-4 w-4 mr-2" />
                        Rollback to v{selectedVersionData.version}
                      </Button>
                    )}
                  </div>

                  <div className="grid grid-cols-3 gap-4 text-sm">
                    <div>
                      <span className="text-foreground">Author:</span>
                      <div className="font-medium">{selectedVersionData.author}</div>
                    </div>
                    <div>
                      <span className="text-foreground">Date:</span>
                      <div className="font-medium">
                        {new Date(selectedVersionData.timestamp).toLocaleString()}
                      </div>
                    </div>
                    <div>
                      <span className="text-foreground">Type:</span>
                      <Badge
                        className={`text-xs ${getChangeTypeColor(selectedVersionData.changeType)}`}
                      >
                        {selectedVersionData.changeType}
                      </Badge>
                    </div>
                  </div>
                </div>

                <Separator className="mb-4" />

                <ScrollArea className="flex-1">
                  <div className="space-y-4">
                    {/* Changes */}
                    <div>
                      <h4 className="font-medium mb-2">Changes</h4>
                      <div className="space-y-1">
                        {selectedVersionData.changes.map((change, idx) => (
                          <div key={idx} className="text-sm text-foreground">
                            • {change}
                          </div>
                        ))}
                      </div>
                    </div>

                    {/* Permissions */}
                    <div>
                      <h4 className="font-medium mb-2">
                        Permissions ({selectedVersionData.permissions.length} resources)
                      </h4>
                      <div className="space-y-3">
                        {selectedVersionData.permissions.map((permission) => (
                          <div
                            key={permission.resourceId}
                            className="border rounded-lg p-3 bg-background"
                          >
                            <div className="font-medium text-sm mb-2">
                              {permission.resourceName}
                            </div>
                            <div className="flex flex-wrap gap-1">
                              {permission.scopes.map((scope, index) => (
                                <Badge key={scope} variant="outline" className="text-xs">
                                  {permission.scopeNames?.[index] || scope}
                                </Badge>
                              ))}
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  </div>
                </ScrollArea>
              </div>
            ) : (
              // Empty State
              <div className="flex-1 flex items-center justify-center text-center">
                <div className="space-y-3">
                  <History className="h-12 w-12 text-foreground mx-auto" />
                  <div>
                    <h3 className="font-medium">Select a Version</h3>
                    <p className="text-sm text-foreground">
                      Choose a version from the left to view its details
                    </p>
                  </div>
                </div>
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            Close
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
