import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Plus, Edit2, Trash2, ExternalLink, AlertTriangle, Shield } from "lucide-react";
import type { RoleFormData, RoleGrant, ClientOption, ResourceOption } from "../types";
import { mockClients } from "../utils/mock-data";

interface PermissionBuilderProps {
  formData: RoleFormData;
  onUpdate: (data: Partial<RoleFormData>) => void;
}

export function PermissionBuilder({ formData, onUpdate }: PermissionBuilderProps) {
  const [selectedClient, setSelectedClient] = useState<ClientOption | null>(null);
  const [selectedResource, setSelectedResource] = useState<ResourceOption | null>(null);
  const [selectedScopes, setSelectedScopes] = useState<string[]>([]);
  const [editingIndex, setEditingIndex] = useState<number | null>(null);

  const handleClientChange = (clientId: string) => {
    const client = mockClients.find((c) => c.id === clientId) || null;
    setSelectedClient(client);
    setSelectedResource(null);
    setSelectedScopes([]);
  };

  const handleResourceChange = (resourcePath: string) => {
    const resource = selectedClient?.resources.find((r) => r.path === resourcePath) || null;
    setSelectedResource(resource);
    setSelectedScopes([]);
  };

  const handleScopeToggle = (scope: string, checked: boolean) => {
    setSelectedScopes((prev) => (checked ? [...prev, scope] : prev.filter((s) => s !== scope)));
  };

  const handleAddGrant = () => {
    if (!selectedResource || selectedScopes.length === 0) return;

    const newGrant: RoleGrant = {
      resource: selectedResource.path,
      scopes: selectedScopes,
      client: selectedClient?.id,
      isExternal: selectedResource.isExternal,
    };

    const newGrants =
      editingIndex !== null
        ? formData.grants.map((grant, index) => (index === editingIndex ? newGrant : grant))
        : [...formData.grants, newGrant];

    onUpdate({ grants: newGrants });

    // Reset form
    setSelectedClient(null);
    setSelectedResource(null);
    setSelectedScopes([]);
    setEditingIndex(null);
  };

  const handleEditGrant = (index: number) => {
    const grant = formData.grants[index];
    const client = mockClients.find((c) => c.id === grant.client);
    const resource = client?.resources.find((r) => r.path === grant.resource);

    setSelectedClient(client || null);
    setSelectedResource(resource || null);
    setSelectedScopes(grant.scopes);
    setEditingIndex(index);
  };

  const handleDeleteGrant = (index: number) => {
    const newGrants = formData.grants.filter((_, i) => i !== index);
    onUpdate({ grants: newGrants });
  };

  const handleCancelEdit = () => {
    setSelectedClient(null);
    setSelectedResource(null);
    setSelectedScopes([]);
    setEditingIndex(null);
  };

  const getClientName = (clientId?: string) => {
    return mockClients.find((c) => c.id === clientId)?.name || "Unknown Client";
  };

  const getResourceLabel = (resourcePath: string, clientId?: string) => {
    const client = mockClients.find((c) => c.id === clientId);
    const resource = client?.resources.find((r) => r.path === resourcePath);
    return resource?.label || resourcePath;
  };

  const canAddGrant = selectedResource && selectedScopes.length > 0;

  return (
    <Card className="border rounded-xl bg-muted/30">
      <CardHeader className="pb-4">
        <div className="flex items-center gap-3">
          <div className="p-2 rounded-lg bg-primary/10">
            <Shield className="h-5 w-5 text-primary" />
          </div>
          <div>
            <CardTitle className="text-xl font-semibold text-foreground">
              Permission Builder
            </CardTitle>
            <p className="text-base text-foreground mt-1">
              Configure resource permissions and scopes for this role
            </p>
          </div>
        </div>
      </CardHeader>
      <CardContent className="space-y-6">
        {/* Add Permission Form */}
        <div className="border rounded-lg p-4 space-y-4">
          <div className="flex items-center justify-between">
            <h4 className="font-medium">
              {editingIndex !== null ? "Edit Permission" : "Add Permission"}
            </h4>
            {editingIndex !== null && (
              <Button variant="ghost" size="sm" onClick={handleCancelEdit}>
                Cancel
              </Button>
            )}
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* Client Selection */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Client</Label>
              <Select value={selectedClient?.id || ""} onValueChange={handleClientChange}>
                <SelectTrigger>
                  <SelectValue placeholder="Select client" />
                </SelectTrigger>
                <SelectContent>
                  {mockClients.map((client) => (
                    <SelectItem key={client.id} value={client.id}>
                      <div className="flex items-center gap-2">
                        <span>{client.name}</span>
                        {client.resources.some((r) => r.isExternal) && (
                          <Badge variant="secondary" className="text-xs">
                            External
                          </Badge>
                        )}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {/* Resource Selection */}
            <div className="space-y-2">
              <Label className="text-sm font-medium">Resource</Label>
              <Select
                value={selectedResource?.path || ""}
                onValueChange={handleResourceChange}
                disabled={!selectedClient}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Select resource" />
                </SelectTrigger>
                <SelectContent>
                  {selectedClient?.resources.map((resource) => (
                    <SelectItem key={resource.path} value={resource.path}>
                      <div className="flex items-center gap-2">
                        <code className="text-xs bg-muted px-1 rounded">{resource.path}</code>
                        <span className="text-sm">{resource.label}</span>
                        {resource.isExternal && (
                          <Badge variant="secondary" className="text-xs">
                            External
                          </Badge>
                        )}
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Scopes Selection */}
          {selectedResource && (
            <div className="space-y-2">
              <Label className="text-sm font-medium">Scopes</Label>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-2">
                {selectedResource.scopes.map((scope) => (
                  <div key={scope.name} className="flex items-center space-x-2">
                    <Checkbox
                      id={scope.name}
                      checked={selectedScopes.includes(scope.name)}
                      onCheckedChange={(checked) => handleScopeToggle(scope.name, !!checked)}
                    />
                    <label
                      htmlFor={scope.name}
                      className="text-sm flex items-center gap-2 cursor-pointer"
                    >
                      <span>{scope.name}</span>
                      {scope.isExternal && (
                        <Badge variant="secondary" className="text-xs">
                          External
                        </Badge>
                      )}
                      {scope.isDeprecated && (
                        <Badge variant="destructive" className="text-xs">
                          Deprecated
                        </Badge>
                      )}
                    </label>
                  </div>
                ))}
              </div>
            </div>
          )}

          <Button onClick={handleAddGrant} disabled={!canAddGrant} className="w-full">
            <Plus className="mr-2 h-4 w-4" />
            {editingIndex !== null ? "Update Permission" : "Add Permission"}
          </Button>

          {/* Need new resource/scope link */}
          <div className="text-center">
            <Button variant="link" size="sm" className="text-xs">
              <Plus className="mr-1 h-3 w-3" />
              Need a new resource or scope?
            </Button>
          </div>
        </div>

        {/* Grants Table */}
        <div className="space-y-3">
          <h4 className="font-medium">Current Permissions</h4>

          {formData.grants.length === 0 ? (
            <div className="text-center py-8 text-foreground">
              <p>No permissions configured yet</p>
              <p className="text-sm">Add your first permission above</p>
            </div>
          ) : (
            <div className="space-y-2">
              {formData.grants.map((grant, index) => (
                <div
                  key={index}
                  className="flex items-center justify-between p-3 border rounded-lg hover:bg-muted/50 transition-colors"
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2 mb-1">
                      <code className="text-xs bg-muted px-2 py-1 rounded">{grant.resource}</code>
                      {grant.isExternal && (
                        <Badge variant="secondary" className="text-xs">
                          External
                        </Badge>
                      )}
                    </div>
                    <div className="text-sm text-foreground mb-1">
                      {getClientName(grant.client)}
                    </div>
                    <div className="flex flex-wrap gap-1">
                      {grant.scopes.map((scope) => (
                        <Badge key={scope} variant="outline" className="text-xs">
                          {scope}
                        </Badge>
                      ))}
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    <Button size="sm" variant="ghost" onClick={() => handleEditGrant(index)}>
                      <Edit2 className="h-4 w-4" />
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => handleDeleteGrant(index)}>
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </CardContent>
    </Card>
  );
}
