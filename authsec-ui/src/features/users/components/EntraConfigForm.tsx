import React from "react";
import { Input } from "@/components/ui/input";
import { PasswordInput } from "@/components/ui/password-input";
import { Label } from "@/components/ui/label";
import { Checkbox } from "@/components/ui/checkbox";
import { Textarea } from "@/components/ui/textarea";

export interface EntraConfigFormData {
  config_name: string;
  description?: string;
  tenant_id: string;
  client_id: string;
  client_secret: string;
  skip_verify?: boolean;
}

interface EntraConfigFormProps {
  config: EntraConfigFormData;
  onChange: (config: EntraConfigFormData) => void;
  touched?: Partial<Record<keyof EntraConfigFormData, boolean>>;
  onBlur?: (field: keyof EntraConfigFormData) => void;
  errors?: Partial<Record<keyof EntraConfigFormData, string>>;
}

export function EntraConfigForm({
  config,
  onChange,
  touched = {},
  onBlur,
  errors = {},
}: EntraConfigFormProps) {
  const updateField = <K extends keyof EntraConfigFormData>(
    field: K,
    value: EntraConfigFormData[K]
  ) => {
    onChange({ ...config, [field]: value });
  };

  const isErr = (k: keyof EntraConfigFormData) => touched[k] && errors[k];

  return (
    <div className="space-y-3">
      {/* Configuration Name and Description in a grid */}
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label htmlFor="config-name" className="text-sm">
            Configuration Name <span className="text-destructive">*</span>
          </Label>
          <Input
            id="config-name"
            value={config.config_name}
            onChange={(e) => updateField("config_name", e.target.value)}
            onBlur={() => onBlur?.("config_name")}
            aria-invalid={!!isErr("config_name")}
            placeholder="e.g., Azure Production"
            autoFocus
            className="h-9"
          />
          {isErr("config_name") && (
            <p className="text-xs text-destructive">{errors.config_name}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="config-description" className="text-sm">Description (Optional)</Label>
          <Input
            id="config-description"
            value={config.description || ""}
            onChange={(e) => updateField("description", e.target.value)}
            placeholder="e.g., Main Azure AD tenant"
            className="h-9"
          />
        </div>
      </div>

      {/* Tenant ID and Client ID in a grid */}
      <div className="grid grid-cols-2 gap-4">
        <div className="space-y-1.5">
          <Label htmlFor="entra-tenant-id" className="text-sm">
            Tenant ID <span className="text-destructive">*</span>
          </Label>
          <Input
            id="entra-tenant-id"
            value={config.tenant_id}
            onChange={(e) => updateField("tenant_id", e.target.value)}
            onBlur={() => onBlur?.("tenant_id")}
            aria-invalid={!!isErr("tenant_id")}
            placeholder="12345678-1234-5678-9012-123456789012"
            autoComplete="off"
            className="h-9"
          />
          <p className="text-[11px] text-muted-foreground">
            Azure tenant ID (Directory ID)
          </p>
          {isErr("tenant_id") && (
            <p className="text-xs text-destructive">{errors.tenant_id}</p>
          )}
        </div>

        <div className="space-y-1.5">
          <Label htmlFor="entra-client-id" className="text-sm">
            Client ID <span className="text-destructive">*</span>
          </Label>
          <Input
            id="entra-client-id"
            value={config.client_id}
            onChange={(e) => updateField("client_id", e.target.value)}
            onBlur={() => onBlur?.("client_id")}
            aria-invalid={!!isErr("client_id")}
            placeholder="87654321-4321-8765-2109-987654321098"
            autoComplete="off"
            className="h-9"
          />
          <p className="text-[11px] text-muted-foreground">
            Application (client) ID
          </p>
          {isErr("client_id") && (
            <p className="text-xs text-destructive">{errors.client_id}</p>
          )}
        </div>
      </div>

      {/* Client Secret - full width */}
      <div className="space-y-1.5">
        <Label htmlFor="entra-client-secret" className="text-sm">
          Client Secret <span className="text-destructive">*</span>
        </Label>
        <PasswordInput
          id="entra-client-secret"
          value={config.client_secret}
          onChange={(e) => updateField("client_secret", e.target.value)}
          onBlur={() => onBlur?.("client_secret")}
          aria-invalid={!!isErr("client_secret")}
          placeholder="Enter your application client secret"
          autoComplete="off"
          className="h-9"
        />
        <p className="text-[11px] text-muted-foreground">
          Client secret value (not the secret ID)
        </p>
        {isErr("client_secret") && (
          <p className="text-xs text-destructive">{errors.client_secret}</p>
        )}
      </div>

      {/* Skip Verify Option */}
      <div className="flex items-center space-x-2 pt-2">
        <Checkbox
          id="skip-verify"
          checked={config.skip_verify ?? false}
          onCheckedChange={(checked) =>
            updateField("skip_verify", checked as boolean)
          }
        />
        <Label htmlFor="skip-verify" className="text-sm font-normal cursor-pointer">
          Skip Certificate Verification <span className="text-[11px] text-muted-foreground">(testing only)</span>
        </Label>
      </div>
    </div>
  );
}
