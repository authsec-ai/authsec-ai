import { useMemo } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Progress } from "@/components/ui/progress";
import {
  Check,
  AlertCircle,
  AlertTriangle,
  Users,
  ShieldCheck,
  Clock,
  ExternalLink,
} from "lucide-react";
import type { RoleFormData, RolePreview } from "../types";
import { validateFormData } from "../utils/role-context";
import { mockClients } from "../utils/mock-data";

interface PreviewSidebarProps {
  formData: RoleFormData;
  onCreateRole: () => void;
  isCreating?: boolean;
}

export function PreviewSidebar({
  formData,
  onCreateRole,
  isCreating = false,
}: PreviewSidebarProps) {
  const preview = useMemo((): RolePreview => {
    const validation = validateFormData(formData);
    const warnings: string[] = [];

    // Check for deprecated scopes
    formData.grants.forEach((grant) => {
      const client = mockClients.find((c) => c.id === grant.client);
      const resource = client?.resources.find((r) => r.path === grant.resource);
      grant.scopes.forEach((scope) => {
        const scopeConfig = resource?.scopes.find((s) => s.name === scope);
        if (scopeConfig?.isDeprecated) {
          warnings.push(`Scope "${scope}" is deprecated`);
        }
      });
    });

    // Check for external resources
    const externalGrants = formData.grants.filter((g) => g.isExternal);
    if (externalGrants.length > 0) {
      warnings.push(`${externalGrants.length} external resource(s) included`);
    }

    // Check for empty assignments
    if (formData.assignedUsers.length === 0 && formData.assignedGroups.length === 0) {
      warnings.push("No initial assignments configured");
    }

    return {
      roleId: formData.roleId,
      resourceCount: formData.grants.length,
      scopeCount: formData.grants.reduce((sum, grant) => sum + grant.scopes.length, 0),
      userCount: formData.assignedUsers.length,
      groupCount: formData.assignedGroups.length,
      warnings,
      isValid: validation.isValid,
    };
  }, [formData]);

  const getValidationProgress = () => {
    const checks = [
      { key: "roleId", valid: !!formData.roleId, label: "Role ID" },
      { key: "displayName", valid: !!formData.displayName, label: "Display Name" },
      { key: "grants", valid: formData.grants.length > 0, label: "Permissions" },
    ];

    const validChecks = checks.filter((c) => c.valid).length;
    return { progress: (validChecks / checks.length) * 100, checks };
  };

  const validationStatus = getValidationProgress();

  return (
    <div className="space-y-4">
      {/* Role Preview */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Role Preview</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          <div>
            <div className="font-medium text-lg">{preview.roleId || "—"}</div>
            <div className="text-sm text-foreground">
              {formData.displayName || "No display name"}
            </div>
            {formData.description && (
              <div className="text-xs text-foreground mt-1">{formData.description}</div>
            )}
          </div>

          <div className="grid grid-cols-2 gap-3 text-sm">
            <div className="text-center p-2 bg-muted/50 rounded">
              <div className="font-medium">{preview.resourceCount}</div>
              <div className="text-xs text-foreground">Resources</div>
            </div>
            <div className="text-center p-2 bg-muted/50 rounded">
              <div className="font-medium">{preview.scopeCount}</div>
              <div className="text-xs text-foreground">Scopes</div>
            </div>
            <div className="text-center p-2 bg-muted/50 rounded">
              <div className="font-medium">{preview.userCount}</div>
              <div className="text-xs text-foreground">Users</div>
            </div>
            <div className="text-center p-2 bg-muted/50 rounded">
              <div className="font-medium">{preview.groupCount}</div>
              <div className="text-xs text-foreground">Groups</div>
            </div>
          </div>

          <div className="text-sm">
            <div className="flex items-center justify-between mb-1">
              <span className="text-foreground">Completion</span>
              <span className="text-xs">{Math.round(validationStatus.progress)}%</span>
            </div>
            <Progress value={validationStatus.progress} className="h-2" />
          </div>
        </CardContent>
      </Card>

      {/* Validation Status */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Validation</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {validationStatus.checks.map((check) => (
            <div key={check.key} className="flex items-center gap-2 text-sm">
              {check.valid ? (
                <Check className="h-4 w-4 text-green-600" />
              ) : (
                <AlertCircle className="h-4 w-4 text-foreground" />
              )}
              <span className={check.valid ? "text-foreground" : "text-foreground"}>
                {check.label}
              </span>
            </div>
          ))}

          {preview.isValid && (
            <div className="flex items-center gap-2 text-sm text-green-600 pt-2 border-t">
              <ShieldCheck className="h-4 w-4" />
              <span>Ready to create</span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Warnings */}
      {preview.warnings.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base flex items-center gap-2">
              <AlertTriangle className="h-4 w-4 text-amber-500" />
              Warnings
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {preview.warnings.map((warning, index) => (
                <div key={index} className="text-sm text-amber-700 dark:text-amber-400">
                  • {warning}
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Grants Summary */}
      {formData.grants.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Permissions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {formData.grants.map((grant, index) => (
                <div key={index} className="text-sm">
                  <div className="flex items-center gap-2">
                    <code className="text-xs bg-muted px-1 rounded">{grant.resource}</code>
                    {grant.isExternal && (
                      <Badge variant="secondary" className="text-xs">
                        External
                      </Badge>
                    )}
                  </div>
                  <div className="text-xs text-foreground ml-1">
                    {grant.scopes.join(", ")}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Create Role Button */}
      <Card>
        <CardContent className="pt-6">
          <Button
            onClick={onCreateRole}
            disabled={!preview.isValid || isCreating}
            className="w-full"
            size="lg"
          >
            {isCreating ? (
              <>
                <Clock className="mr-2 h-4 w-4 animate-spin" />
                Creating Role...
              </>
            ) : (
              <>
                <ShieldCheck className="mr-2 h-4 w-4" />
                Create Role
              </>
            )}
          </Button>

          {!preview.isValid && (
            <div className="text-xs text-foreground text-center mt-2">
              Complete required fields to enable creation
            </div>
          )}
        </CardContent>
      </Card>

      {/* Assignment Summary */}
      {(formData.assignedUsers.length > 0 || formData.assignedGroups.length > 0) && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Initial Assignments</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="space-y-2 text-sm">
              {formData.assignedUsers.length > 0 && (
                <div className="flex items-center gap-2">
                  <Users className="h-4 w-4 text-foreground" />
                  <span>{formData.assignedUsers.length} user(s)</span>
                </div>
              )}
              {formData.assignedGroups.length > 0 && (
                <div className="flex items-center gap-2">
                  <Users className="h-4 w-4 text-foreground" />
                  <span>{formData.assignedGroups.length} group(s)</span>
                </div>
              )}
            </div>
          </CardContent>
        </Card>
      )}

      {/* Help */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Need Help?</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="space-y-2 text-sm">
            <Button variant="link" size="sm" className="p-0 h-auto text-left">
              <ExternalLink className="mr-1 h-3 w-3" />
              Role Management Guide
            </Button>
            <Button variant="link" size="sm" className="p-0 h-auto text-left">
              <ExternalLink className="mr-1 h-3 w-3" />
              Permission Scopes
            </Button>
            <Button variant="link" size="sm" className="p-0 h-auto text-left">
              <ExternalLink className="mr-1 h-3 w-3" />
              SDK Documentation
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
