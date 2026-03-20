import { Input } from "../../../components/ui/input";
import { Label } from "../../../components/ui/label";
import { RadioGroup, RadioGroupItem } from "../../../components/ui/radio-group";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import type { ExternalServiceFormData } from "../types";

interface AdvancedOptionsProps {
  formData: ExternalServiceFormData;
  onUpdate: (updates: Partial<ExternalServiceFormData>) => void;
}

export function AdvancedOptions({ formData, onUpdate }: AdvancedOptionsProps) {
  const handleSyncIntervalChange = (value: string) => {
    onUpdate({
      advancedOptions: {
        ...formData.advancedOptions,
        syncInterval: value,
      },
    });
  };

  const handleTokenStorageRegionChange = (value: "us" | "eu") => {
    onUpdate({
      advancedOptions: {
        ...formData.advancedOptions,
        tokenStorageRegion: value,
      },
    });
  };

  const handleCustomEndpointChange = (
    field: keyof typeof formData.advancedOptions.customAuthEndpoints,
    value: string
  ) => {
    onUpdate({
      advancedOptions: {
        ...formData.advancedOptions,
        customAuthEndpoints: {
          ...formData.advancedOptions.customAuthEndpoints,
          [field]: value,
        },
      },
    });
  };

  // Common sync intervals as cron expressions
  const syncIntervals = [
    { label: "Every hour", value: "0 * * * *" },
    { label: "Every day at midnight", value: "0 0 * * *" },
    { label: "Every Monday at 9am", value: "0 9 * * 1" },
    { label: "First day of month", value: "0 0 1 * *" },
    { label: "Custom", value: "custom" },
  ];

  return (
    <div className="space-y-6 pt-4">
      {/* Sync Interval */}
      <div className="space-y-3">
        <Label htmlFor="syncInterval">Sync Interval (cron)</Label>
        <Select
          value={
            syncIntervals.some((si) => si.value === formData.advancedOptions.syncInterval)
              ? formData.advancedOptions.syncInterval
              : "custom"
          }
          onValueChange={(value) => {
            if (value !== "custom") {
              handleSyncIntervalChange(value);
            }
          }}
        >
          <SelectTrigger id="syncInterval">
            <SelectValue placeholder="Select sync interval" />
          </SelectTrigger>
          <SelectContent>
            {syncIntervals.map((interval) => (
              <SelectItem key={interval.value} value={interval.value}>
                {interval.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {(!syncIntervals.some((si) => si.value === formData.advancedOptions.syncInterval) ||
          formData.advancedOptions.syncInterval === "custom") && (
          <div className="pt-2">
            <Label htmlFor="customSyncInterval" className="text-sm">
              Custom Cron Expression
            </Label>
            <Input
              id="customSyncInterval"
              value={formData.advancedOptions.syncInterval}
              onChange={(e) => handleSyncIntervalChange(e.target.value)}
              placeholder="0 * * * *"
              className="mt-1"
            />
            <p className="text-xs text-foreground mt-1">
              Format: minute hour day month weekday (e.g., "0 * * * *" for every hour)
            </p>
          </div>
        )}
      </div>

      {/* Custom Auth Endpoints (only for Custom OAuth 2.0) */}
      {formData.provider === "custom_oauth2" && (
        <div className="space-y-4 border-t pt-4">
          <div className="text-sm font-medium">Custom OAuth 2.0 Endpoints</div>

          <div className="space-y-3">
            <Label htmlFor="authorizationUrl">Authorization URL</Label>
            <Input
              id="authorizationUrl"
              value={formData.advancedOptions.customAuthEndpoints.authorizationUrl}
              onChange={(e) => handleCustomEndpointChange("authorizationUrl", e.target.value)}
              placeholder="https://example.com/oauth2/authorize"
            />
          </div>

          <div className="space-y-3">
            <Label htmlFor="tokenUrl">Token URL</Label>
            <Input
              id="tokenUrl"
              value={formData.advancedOptions.customAuthEndpoints.tokenUrl}
              onChange={(e) => handleCustomEndpointChange("tokenUrl", e.target.value)}
              placeholder="https://example.com/oauth2/token"
            />
          </div>

          <div className="space-y-3">
            <Label htmlFor="userinfoUrl">User Info URL (optional)</Label>
            <Input
              id="userinfoUrl"
              value={formData.advancedOptions.customAuthEndpoints.userinfoUrl}
              onChange={(e) => handleCustomEndpointChange("userinfoUrl", e.target.value)}
              placeholder="https://example.com/oauth2/userinfo"
            />
          </div>
        </div>
      )}

      {/* Token Storage Region */}
      <div className="space-y-3 border-t pt-4">
        <Label className="text-sm font-medium">Token Storage Region</Label>
        <RadioGroup
          value={formData.advancedOptions.tokenStorageRegion}
          onValueChange={(value) => handleTokenStorageRegionChange(value as "us" | "eu")}
          className="flex flex-col space-y-2"
        >
          <div className="flex items-center space-x-2">
            <RadioGroupItem value="us" id="us" />
            <Label htmlFor="us" className="font-normal">
              US (Default)
            </Label>
          </div>
          <div className="flex items-center space-x-2">
            <RadioGroupItem value="eu" id="eu" />
            <Label htmlFor="eu" className="font-normal">
              EU (GDPR Compliant)
            </Label>
          </div>
        </RadioGroup>
        <p className="text-xs text-foreground">
          Select where user tokens will be stored. EU region provides enhanced GDPR compliance.
        </p>
      </div>
    </div>
  );
}
