import { useEffect } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { Label } from "@/components/ui/label";
import { AlertCircle, RefreshCw, User } from "lucide-react";
import type { RoleFormData } from "../types";
import { generateRoleId, validateRoleId } from "../utils/role-context";

interface BasicsPanelProps {
  formData: RoleFormData;
  onUpdate: (data: Partial<RoleFormData>) => void;
  errors: Record<string, string>;
}

export function BasicsPanel({ formData, onUpdate, errors }: BasicsPanelProps) {
  const roleIdValidation = validateRoleId(formData.roleId);
  const displayNameLength = formData.displayName.length;
  const descriptionLength = formData.description.length;

  // Auto-generate role ID when display name changes
  useEffect(() => {
    if (formData.displayName && !formData.roleId) {
      const generatedId = generateRoleId(formData.displayName);
      onUpdate({ roleId: generatedId });
    }
  }, [formData.displayName, formData.roleId, onUpdate]);

  const handleRegenerateRoleId = () => {
    if (formData.displayName) {
      const generatedId = generateRoleId(formData.displayName);
      onUpdate({ roleId: generatedId });
    }
  };

  return (
    <Card className="border rounded-xl bg-muted/30">
      <CardHeader className="pb-4">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10">
            <User className="h-5 w-5 text-primary" />
          </div>
          <div>
            <CardTitle className="text-xl font-semibold text-foreground">Role Basics</CardTitle>
            <p className="text-base text-foreground mt-1">
              Configure the basic information for this role
            </p>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Role ID */}
        <div className="space-y-2">
          <Label htmlFor="roleId" className="text-base font-semibold text-foreground">
            Role ID (machine)
            <span className="text-destructive ml-1">*</span>
          </Label>
          <div className="flex gap-2">
            <div className="flex-1">
              <Input
                id="roleId"
                value={formData.roleId}
                onChange={(e) => onUpdate({ roleId: e.target.value.toUpperCase() })}
                placeholder="e.g., ORDER_ADMIN"
                className={`font-mono h-12 text-base border-border focus:border-foreground/40 ${
                  !roleIdValidation.isValid ? "border-destructive" : ""
                }`}
              />
              {!roleIdValidation.isValid && (
                <div className="flex items-center gap-2 mt-1 text-sm text-destructive">
                  <AlertCircle className="h-4 w-4" />
                  {roleIdValidation.error}
                </div>
              )}
            </div>
            <button
              type="button"
              onClick={handleRegenerateRoleId}
              disabled={!formData.displayName}
              className="px-3 py-2 text-sm text-foreground hover:text-foreground disabled:opacity-50 disabled:cursor-not-allowed border border-border rounded-md"
              title="Regenerate from display name"
            >
              <RefreshCw className="h-4 w-4" />
            </button>
          </div>
          <p className="text-xs text-foreground">
            Upper-snake format (A-Z, 0-9, _). Must be unique.
          </p>
        </div>

        {/* Display Name */}
        <div className="space-y-2">
          <Label htmlFor="displayName" className="text-base font-semibold text-foreground">
            Display Name
            <span className="text-destructive ml-1">*</span>
          </Label>
          <Input
            id="displayName"
            value={formData.displayName}
            onChange={(e) => onUpdate({ displayName: e.target.value })}
            placeholder="e.g., Order Administrator"
            className={`h-12 text-base border-border focus:border-foreground/40 ${
              displayNameLength > 50 ? "border-destructive" : ""
            }`}
            maxLength={50}
          />
          <div className="flex items-center justify-between">
            <p className="text-xs text-foreground">Human-readable name shown in dashboards</p>
            <span
              className={`text-xs ${
                displayNameLength > 50 ? "text-destructive" : "text-foreground"
              }`}
            >
              {displayNameLength}/50
            </span>
          </div>
        </div>

        {/* Description */}
        <div className="space-y-2">
          <Label htmlFor="description" className="text-base font-semibold text-foreground">
            Description
            <span className="text-foreground font-normal ml-1">(optional)</span>
          </Label>
          <Textarea
            id="description"
            value={formData.description}
            onChange={(e) => onUpdate({ description: e.target.value })}
            placeholder="Describe what this role is responsible for..."
            className={`min-h-[100px] text-base border-border focus:border-foreground/40 resize-none ${
              descriptionLength > 200 ? "border-destructive" : ""
            }`}
            maxLength={200}
          />
          <div className="flex items-center justify-between">
            <p className="text-xs text-foreground">Optional description for documentation</p>
            <span
              className={`text-xs ${
                descriptionLength > 200 ? "text-destructive" : "text-foreground"
              }`}
            >
              {descriptionLength}/200
            </span>
          </div>
        </div>

        {/* Validation Summary */}
        {Object.keys(errors).length > 0 && (
          <div className="p-4 bg-destructive/10 border border-destructive/20 rounded-lg">
            <div className="flex items-center gap-2 text-sm text-destructive">
              <AlertCircle className="h-4 w-4" />
              <span className="font-medium">Validation Issues:</span>
            </div>
            <ul className="mt-2 text-sm text-destructive space-y-1">
              {Object.entries(errors).map(([field, error]) => (
                <li key={field} className="ml-6">
                  • {error}
                </li>
              ))}
            </ul>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
