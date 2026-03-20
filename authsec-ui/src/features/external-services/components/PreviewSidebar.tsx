import { Button } from "../../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "../../../components/ui/card";
import { Badge } from "../../../components/ui/badge";
import { Copy, AlertTriangle, Database } from "lucide-react";
import { toast } from "react-hot-toast";
import { cn } from "../../../lib/utils";
import type { ExternalServiceFormData, ExternalServiceWarning } from "../types";

interface PreviewSidebarProps {
  formData: ExternalServiceFormData;
  onCreateService: () => void;
  isCreating: boolean;
}

export function PreviewSidebar({ formData, onCreateService, isCreating }: PreviewSidebarProps) {
  // Generate warnings based on the form data
  const getWarnings = (): ExternalServiceWarning[] => {
    const warnings: ExternalServiceWarning[] = [];

    // Check for sensitive scopes
    if (formData.provider) {
      const providerMap: Record<string, string[]> = {
        google_drive: ["drive.appdata", "drive"],
        salesforce: ["full"],
        microsoft_graph: ["Mail.Read", "Directory.Read.All"],
      };

      const sensitiveScopes = providerMap[formData.provider] || [];
      const hasSensitiveScopes = formData.scopes.some((scope) => sensitiveScopes.includes(scope));

      if (hasSensitiveScopes) {
        warnings.push({
          type: "warning",
          message: `Scope ${formData.scopes.find((scope) =>
            sensitiveScopes.includes(scope)
          )} is sensitive (provider review needed)`,
        });
      }
    }

    // Check for deprecated scopes
    if (formData.provider) {
      const deprecatedMap: Record<string, string[]> = {
        google_drive: ["drive.activity"],
        salesforce: ["wave_api"],
        microsoft_graph: ["Sites.Read.All"],
      };

      const deprecatedScopes = deprecatedMap[formData.provider] || [];
      const hasDeprecatedScopes = formData.scopes.some((scope) => deprecatedScopes.includes(scope));

      if (hasDeprecatedScopes) {
        warnings.push({
          type: "warning",
          message: `Scope ${formData.scopes.find((scope) =>
            deprecatedScopes.includes(scope)
          )} is deprecated and may stop working`,
        });
      }
    }

    // No clients linked
    if (formData.linkedClients.length === 0) {
      warnings.push({
        type: "info",
        message: "No clients linked - service won't be available until linked",
      });
    }

    // Custom OAuth without endpoints
    if (formData.provider === "custom_oauth2") {
      const { authorizationUrl, tokenUrl } = formData.advancedOptions.customAuthEndpoints;
      if (!authorizationUrl || !tokenUrl) {
        warnings.push({
          type: "error",
          message: "Custom OAuth 2.0 requires authorization and token URLs",
        });
      }
    }

    return warnings;
  };

  const handleCopyRedirectUri = () => {
    if (formData.redirectUri) {
      navigator.clipboard.writeText(formData.redirectUri);
      toast.success("Redirect URI copied to clipboard");
    }
  };

  const warnings = getWarnings();
  const isValid =
    formData.serviceName && formData.provider && formData.clientId && formData.clientSecret;

  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="pb-2">
          <CardTitle className="text-lg">Service Preview</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Provider */}
          <div className="space-y-1">
            <div className="text-sm text-foreground">Provider</div>
            <div className="font-medium">{formData.providerName || "Not selected"}</div>
          </div>

          {/* External Resources */}
          <div className="space-y-1">
            <div className="text-sm text-foreground">External Resources</div>
            <div>
              {formData.externalResources.length > 0 ? (
                <div className="space-y-1">
                  {formData.externalResources.map((resource, index) => (
                    <div key={index} className="text-sm">
                      {resource.resource}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-sm text-foreground">No resources defined</div>
              )}
            </div>
          </div>

          {/* Scopes */}
          <div className="space-y-1">
            <div className="text-sm text-foreground">Scopes</div>
            <div>
              {formData.scopes.length > 0 ? (
                <div className="flex flex-wrap gap-1">
                  {formData.scopes.slice(0, 3).map((scope) => (
                    <Badge key={scope} variant="outline" className="text-xs">
                      {scope}
                    </Badge>
                  ))}
                  {formData.scopes.length > 3 && (
                    <Badge variant="outline" className="text-xs">
                      +{formData.scopes.length - 3} more
                    </Badge>
                  )}
                </div>
              ) : (
                <div className="text-sm text-foreground">No scopes selected</div>
              )}
            </div>
          </div>

          {/* Linked Clients */}
          <div className="space-y-1">
            <div className="text-sm text-foreground">Linked Clients</div>
            <div className="font-medium">{formData.linkedClients.length}</div>
          </div>

          {/* Redirect URI */}
          <div className="space-y-1">
            <div className="text-sm text-foreground">Redirect URI</div>
            <div className="flex items-center gap-1">
              <div className="text-sm truncate max-w-[200px]">
                {formData.redirectUri || "Not generated yet"}
              </div>
              {formData.redirectUri && (
                <Button
                  variant="ghost"
                  size="sm"
                  className="h-6 w-6 p-0"
                  onClick={handleCopyRedirectUri}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              )}
            </div>
          </div>

          {/* Warnings */}
          {warnings.length > 0 && (
            <div className="space-y-2 border-t pt-3 mt-3">
              <div className="text-sm font-medium">Warnings</div>
              {warnings.map((warning, index) => (
                <div
                  key={index}
                  className={cn(
                    "flex items-start gap-2 text-xs p-2 rounded-md",
                    warning.type === "error" && "bg-destructive/10 text-destructive",
                    warning.type === "warning" && "bg-orange-500/10 text-orange-500",
                    warning.type === "info" && "bg-blue-500/10 text-blue-500"
                  )}
                >
                  <AlertTriangle className="h-3 w-3 mt-0.5 flex-shrink-0" />
                  <span>{warning.message}</span>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="flex flex-col gap-2">
        <Button onClick={onCreateService} disabled={!isValid || isCreating} className="w-full">
          {isCreating ? (
            <>Creating Service...</>
          ) : (
            <>
              <Database className="h-4 w-4 mr-2" />
              Create Service
            </>
          )}
        </Button>
        <Button variant="outline" className="w-full" onClick={() => window.history.back()}>
          Cancel
        </Button>
      </div>
    </div>
  );
}
