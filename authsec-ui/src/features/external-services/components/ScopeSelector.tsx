import { useState, useMemo } from "react";
import { Badge } from "../../../components/ui/badge";
import { Input } from "../../../components/ui/input";
import { Button } from "../../../components/ui/button";
import { X, Plus, AlertTriangle } from "lucide-react";
import { cn } from "../../../lib/utils";
import type { ExternalServiceFormData, ProviderScope } from "../types";

// Provider scopes mapping
const providerScopes: Record<string, ProviderScope[]> = {
  google_drive: [
    {
      id: "drive.readonly",
      name: "drive.readonly",
      description: "View files in your Google Drive",
    },
    {
      id: "drive.file",
      name: "drive.file",
      description: "View and manage files created by this app",
    },
    {
      id: "drive.appdata",
      name: "drive.appdata",
      description: "View and manage app data",
      isSensitive: true,
    },
    {
      id: "drive.metadata.readonly",
      name: "drive.metadata.readonly",
      description: "View metadata for files",
    },
    {
      id: "drive.activity",
      name: "drive.activity",
      description: "View activity history",
      isDeprecated: true,
    },
    { id: "drive.metadata", name: "drive.metadata", description: "View and edit file metadata" },
    {
      id: "drive",
      name: "drive",
      description: "Full access to files and documents",
      isSensitive: true,
    },
  ],
  salesforce: [
    { id: "api", name: "api", description: "Access the Salesforce API" },
    {
      id: "refresh_token",
      name: "refresh_token",
      description: "Get a refresh token for offline access",
    },
    { id: "chatter_api", name: "chatter_api", description: "Access Chatter API" },
    { id: "full", name: "full", description: "Full access to org data", isSensitive: true },
    {
      id: "custom_permissions",
      name: "custom_permissions",
      description: "Access custom permissions",
    },
    { id: "wave_api", name: "wave_api", description: "Access Analytics API", isDeprecated: true },
  ],
  microsoft_graph: [
    { id: "Files.Read", name: "Files.Read", description: "Read files" },
    { id: "Files.ReadWrite", name: "Files.ReadWrite", description: "Read and write files" },
    { id: "User.Read", name: "User.Read", description: "Read user profile" },
    { id: "Mail.Read", name: "Mail.Read", description: "Read mail", isSensitive: true },
    { id: "Calendars.Read", name: "Calendars.Read", description: "Read calendars" },
    {
      id: "Directory.Read.All",
      name: "Directory.Read.All",
      description: "Read directory data",
      isSensitive: true,
    },
    {
      id: "Sites.Read.All",
      name: "Sites.Read.All",
      description: "Read SharePoint sites",
      isDeprecated: true,
    },
  ],
  custom_oauth2: [{ id: "custom", name: "custom", description: "Custom scope" }],
};

interface ScopeSelectorProps {
  formData: ExternalServiceFormData;
  onUpdate: (updates: Partial<ExternalServiceFormData>) => void;
}

export function ScopeSelector({ formData, onUpdate }: ScopeSelectorProps) {
  const [customScope, setCustomScope] = useState("");

  // Get available scopes for the selected provider
  const availableScopes = useMemo(() => {
    return formData.provider ? providerScopes[formData.provider] || [] : [];
  }, [formData.provider]);

  // Add a scope
  const handleAddScope = (scope: string) => {
    if (!formData.scopes.includes(scope)) {
      onUpdate({ scopes: [...formData.scopes, scope] });
    }
  };

  // Remove a scope
  const handleRemoveScope = (scope: string) => {
    onUpdate({ scopes: formData.scopes.filter((s) => s !== scope) });
  };

  // Add custom scope
  const handleAddCustomScope = () => {
    if (customScope && !formData.scopes.includes(customScope)) {
      onUpdate({ scopes: [...formData.scopes, customScope] });
      setCustomScope("");
    }
  };

  // Get scope details by ID
  const getScopeDetails = (scopeId: string): ProviderScope | undefined => {
    return availableScopes.find((s) => s.id === scopeId);
  };

  return (
    <div className="space-y-4">
      {/* Selected Scopes */}
      <div>
        <div className="text-sm font-medium mb-2">Selected Scopes</div>
        <div className="flex flex-wrap gap-2 min-h-10">
          {formData.scopes.length === 0 ? (
            <div className="text-sm text-foreground">No scopes selected</div>
          ) : (
            formData.scopes.map((scope) => {
              const scopeDetails = getScopeDetails(scope);
              const isDeprecated = scopeDetails?.isDeprecated;
              const isSensitive = scopeDetails?.isSensitive;

              return (
                <Badge
                  key={scope}
                  variant={isDeprecated ? "outline" : isSensitive ? "secondary" : "default"}
                  className={cn(
                    "pl-3 pr-2 py-1 h-auto text-xs flex items-center gap-1",
                    isDeprecated && "border-orange-500 text-orange-500 bg-orange-500/10",
                    isSensitive && "border-blue-500 text-blue-500 bg-blue-500/10"
                  )}
                >
                  {scope}
                  {isDeprecated && <AlertTriangle className="h-3 w-3 text-orange-500 ml-1" />}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-4 w-4 p-0 ml-1 rounded-full hover:bg-muted"
                    onClick={() => handleRemoveScope(scope)}
                  >
                    <X className="h-3 w-3" />
                  </Button>
                </Badge>
              );
            })
          )}
        </div>
      </div>

      {/* Available Scopes */}
      {formData.provider && (
        <div>
          <div className="text-sm font-medium mb-2">Available Scopes</div>
          <div className="flex flex-wrap gap-2">
            {availableScopes.length === 0 ? (
              <div className="text-sm text-foreground">
                No scopes available for this provider
              </div>
            ) : (
              availableScopes.map((scope) => (
                <Badge
                  key={scope.id}
                  variant={
                    scope.isDeprecated ? "outline" : scope.isSensitive ? "secondary" : "default"
                  }
                  className={cn(
                    "pl-3 pr-2 py-1 h-auto text-xs flex items-center gap-1 cursor-pointer",
                    scope.isDeprecated && "border-orange-500 text-orange-500 bg-orange-500/10",
                    scope.isSensitive && "border-blue-500 text-blue-500 bg-blue-500/10",
                    formData.scopes.includes(scope.id) && "opacity-50"
                  )}
                  onClick={() => !formData.scopes.includes(scope.id) && handleAddScope(scope.id)}
                >
                  {scope.name}
                  {scope.isDeprecated && <AlertTriangle className="h-3 w-3 text-orange-500 ml-1" />}
                  {!formData.scopes.includes(scope.id) && <Plus className="h-3 w-3 ml-1" />}
                </Badge>
              ))
            )}
          </div>

          {/* Scope descriptions */}
          <div className="mt-4 text-sm text-foreground">
            <div className="flex items-center gap-2 mb-1">
              <Badge variant="default" className="h-4 w-4 p-0"></Badge>
              <span>Standard scopes</span>
            </div>
            <div className="flex items-center gap-2 mb-1">
              <Badge
                variant="secondary"
                className="h-4 w-4 p-0 border-blue-500 bg-blue-500/10"
              ></Badge>
              <span>Sensitive scopes (may require provider review)</span>
            </div>
            <div className="flex items-center gap-2">
              <Badge variant="outline" className="h-4 w-4 p-0 border-orange-500"></Badge>
              <span>Deprecated scopes (may stop working in future)</span>
            </div>
          </div>
        </div>
      )}

      {/* Custom Scope Input */}
      <div className="pt-4 border-t">
        <div className="text-sm font-medium mb-2">Add Custom Scope</div>
        <div className="flex gap-2">
          <Input
            value={customScope}
            onChange={(e) => setCustomScope(e.target.value)}
            placeholder="Enter custom scope"
            className="flex-1"
          />
          <Button
            variant="outline"
            onClick={handleAddCustomScope}
            disabled={!customScope || formData.scopes.includes(customScope)}
          >
            <Plus className="h-4 w-4 mr-2" />
            Add
          </Button>
        </div>
        <div className="text-xs text-foreground mt-2">
          Add custom scopes if they're not in the predefined list
        </div>
      </div>
    </div>
  );
}
