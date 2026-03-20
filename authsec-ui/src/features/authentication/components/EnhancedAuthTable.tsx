import React, { useState } from "react";
import type { AuthMethod } from "../types";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { Checkbox } from "../../../components/ui/checkbox";
import { cn } from "../../../lib/utils";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "../../../components/ui/table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Edit,
  Copy,
  Shield,
  Users,
  Globe,
  Settings,
  Pause,
  Play,
  Plus,
  Code2,
} from "lucide-react";
import { ViewSDKModal, generateAuthMethodSDKCode } from "@/features/sdk";

// Get provider icon
const getProviderIcon = (providerType?: string) => {
  switch (providerType?.toLowerCase()) {
    case "oidc":
    case "oauth2":
      return Globe;
    case "saml":
      return Shield;
    case "email-pass":
      return Users;
    default:
      return Settings;
  }
};

// Get status badge color
const STATUS_BADGE_STYLES: Record<string, string> = {
  active:
    "border-transparent text-[color:var(--color-success)] bg-[color-mix(in_oklab,var(--color-success)_18%,transparent)]",
  inactive:
    "border-transparent text-[color:var(--color-text-secondary)] bg-[color-mix(in_oklab,var(--color-muted)_65%,transparent)]",
  default:
    "border-transparent text-[color:var(--color-primary)] bg-[color-mix(in_oklab,var(--color-primary)_18%,transparent)]",
};

const getStatusBadgeClass = (status: string) =>
  STATUS_BADGE_STYLES[status] ?? STATUS_BADGE_STYLES.default;

const ENVIRONMENT_DOT: Record<string, string> = {
  production: "bg-[color:var(--color-danger)]",
  staging: "bg-[color:var(--color-warning)]",
  development: "bg-[color:var(--color-success)]",
};

interface EnhancedAuthTableProps {
  data: AuthMethod[];
  selectedMethods: string[];
  onSelectAll: () => void;
  onSelectMethod: (methodId: string) => void;
  onEditMethod: (methodId: string) => void;
  onDuplicateMethod: (methodId: string) => void;
  onToggleStatus: (methodId: string) => void;
  onCreateMethod: () => void;
}

export function EnhancedAuthTable({
  data = [],
  selectedMethods = [],
  onSelectAll,
  onSelectMethod,
  onEditMethod,
  onDuplicateMethod,
  onToggleStatus,
  onCreateMethod,
}: EnhancedAuthTableProps) {
  const [expandedRows, setExpandedRows] = useState<Set<string>>(new Set());
  const [sdkModalOpen, setSdkModalOpen] = useState(false);
  const [selectedMethodForSDK, setSelectedMethodForSDK] = useState<AuthMethod | null>(null);

  const handleViewSDK = (method: AuthMethod) => {
    setSelectedMethodForSDK(method);
    setSdkModalOpen(true);
  };

  const toggleRowExpansion = (methodId: string) => {
    const newExpanded = new Set(expandedRows);
    if (newExpanded.has(methodId)) {
      newExpanded.delete(methodId);
    } else {
      newExpanded.add(methodId);
    }
    setExpandedRows(newExpanded);
  };

  // Handle empty state
  if (!data || data.length === 0) {
    return (
      <div className="border rounded-sm">
        <div className="flex flex-col items-center justify-center py-12 text-center">
          <Shield className="w-12 h-12 text-slate-400 dark:text-neutral-500 mb-4" />
          <h3 className="text-lg font-semibold text-slate-900 dark:text-neutral-100 mb-2">
            No authentication methods found
          </h3>
          <p className="text-slate-500 dark:text-neutral-400 mb-4 max-w-sm">
            Get started by creating your first authentication method to enable user login.
          </p>
          <Button onClick={onCreateMethod} className="flex items-center gap-2">
            <Plus className="w-4 h-4" />
            Add Authentication Method
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="border rounded-sm overflow-hidden">
      <Table>
        <TableHeader>
          <TableRow className="bg-[var(--component-table-header-bg)]">
            <TableHead className="w-12">
              <Checkbox
                checked={selectedMethods.length === data.length && data.length > 0}
                indeterminate={selectedMethods.length > 0 && selectedMethods.length < data.length}
                onCheckedChange={onSelectAll}
                aria-label="Select all methods"
              />
            </TableHead>
            <TableHead>Method</TableHead>
            <TableHead>Provider</TableHead>
            <TableHead>Status</TableHead>
            <TableHead>Environment</TableHead>
            <TableHead>Users</TableHead>
            <TableHead className="w-12"></TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {data.map((method) => {
            const ProviderIcon = getProviderIcon(method.providerType);
            const isSelected = selectedMethods.includes(method.id);
            const isExpanded = expandedRows.has(method.id);

            return (
              <React.Fragment key={method.id}>
                <TableRow
                  className="cursor-pointer transition-colors"
                  data-state={isSelected ? "selected" : undefined}
                  onClick={() => toggleRowExpansion(method.id)}
                >
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <Checkbox
                      checked={isSelected}
                      onCheckedChange={() => onSelectMethod(method.id)}
                      aria-label={`Select ${method.displayName}`}
                    />
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-3">
                      <div className="flex-shrink-0">
                        <div className="flex h-8 w-8 items-center justify-center rounded-full bg-muted text-foreground">
                          <ProviderIcon className="h-4 w-4" />
                        </div>
                      </div>
                      <div className="min-w-0 flex-1">
                        <div className="truncate font-medium text-foreground">
                          {method.displayName || method.name}
                        </div>
                        <div className="truncate text-sm text-foreground">
                          {method.providerType?.toUpperCase()} • {method.methodKey}
                        </div>
                      </div>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline" className="font-mono text-xs">
                      {method.provider || method.providerType}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="secondary"
                      className={cn("font-medium", getStatusBadgeClass(method.status))}
                    >
                      {method.status?.charAt(0).toUpperCase() + method.status?.slice(1)}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <div
                        className={cn(
                          "h-2 w-2 rounded-full",
                          ENVIRONMENT_DOT[method.environment || ""] ?? ENVIRONMENT_DOT.development
                        )}
                      />
                      <span className="text-sm capitalize text-foreground">
                        {method.environment || "—"}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center space-x-2">
                      <Users className="h-4 w-4 text-foreground" />
                      <span className="font-medium text-foreground">
                        {method.usersCount || 0}
                      </span>
                    </div>
                  </TableCell>
                  <TableCell onClick={(e) => e.stopPropagation()}>
                    <DropdownMenu>
                      <DropdownMenuTrigger asChild>
                        <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
                          <MoreHorizontal className="h-4 w-4" />
                          <span className="sr-only">Open menu</span>
                        </Button>
                      </DropdownMenuTrigger>
                      <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
                        <DropdownMenuItem onClick={() => handleViewSDK(method)}>
                          <Code2 className="mr-2 h-4 w-4" />
                          View SDK Code
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onEditMethod(method.id)}>
                          <Edit className="mr-2 h-4 w-4" />
                          Edit
                        </DropdownMenuItem>
                        <DropdownMenuItem onClick={() => onDuplicateMethod(method.id)}>
                          <Copy className="mr-2 h-4 w-4" />
                          Duplicate
                        </DropdownMenuItem>
                        <DropdownMenuSeparator />
                        <DropdownMenuItem onClick={() => onToggleStatus(method.id)}>
                          {method.status === "active" ? (
                            <>
                              <Pause className="mr-2 h-4 w-4" />
                              Disable
                            </>
                          ) : (
                            <>
                              <Play className="mr-2 h-4 w-4" />
                              Enable
                            </>
                          )}
                        </DropdownMenuItem>
                      </DropdownMenuContent>
                    </DropdownMenu>
                  </TableCell>
                </TableRow>
                
                {/* Expanded Row */}
                {isExpanded && (
                  <TableRow>
                    <TableCell colSpan={8} className="p-0">
                      <div className="border-t border-border/60 bg-muted/40 p-6">
                        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
                          {/* Configuration Details */}
                          <div className="space-y-3">
                            <h4 className="flex items-center gap-2 font-semibold text-foreground">
                              <Settings className="w-4 h-4" />
                              Configuration
                            </h4>
                            <div className="space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-foreground">Method Key:</span>
                                <code className="rounded bg-muted px-2 py-1 text-xs">
                                  {method.methodKey}
                                </code>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-foreground">Type:</span>
                                <span className="font-medium text-foreground">{method.type}</span>
                              </div>
                            </div>
                          </div>

                          {/* Provider Details */}
                          <div className="space-y-3">
                            <h4 className="flex items-center gap-2 font-semibold text-foreground">
                              <Globe className="w-4 h-4" />
                              Provider Details
                            </h4>
                            <div className="space-y-2 text-sm">
                              {method.providerConfig?.issuerUrl && (
                                <div className="flex flex-col space-y-1">
                                  <span className="text-foreground">Issuer URL:</span>
                                  <a
                                    href={method.providerConfig.issuerUrl}
                                    target="_blank"
                                    rel="noopener noreferrer"
                                    className="text-xs font-mono text-primary underline-offset-4 hover:underline"
                                  >
                                    {method.providerConfig.issuerUrl}
                                  </a>
                                </div>
                              )}
                              {method.providerConfig?.clientId && (
                                <div className="flex justify-between">
                                  <span className="text-foreground">Client ID:</span>
                                  <code className="max-w-32 truncate rounded bg-muted px-2 py-1 text-xs">
                                    {method.providerConfig.clientId}
                                  </code>
                                </div>
                              )}
                            </div>
                          </div>

                          {/* Statistics */}
                          <div className="space-y-3">
                            <h4 className="flex items-center gap-2 font-semibold text-foreground">
                              <Users className="w-4 h-4" />
                              Statistics
                            </h4>
                            <div className="space-y-2 text-sm">
                              <div className="flex justify-between">
                                <span className="text-foreground">Connected Users:</span>
                                <span className="font-medium text-foreground">{method.usersCount || 0}</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-foreground">Created:</span>
                                <span className="font-medium text-foreground">{formatDate(method.createdAt)}</span>
                              </div>
                              <div className="flex justify-between">
                                <span className="text-foreground">Updated:</span>
                                <span className="font-medium text-foreground">{formatDate(method.updatedAt)}</span>
                              </div>
                            </div>
                          </div>
                        </div>

                        {/* Description */}
                        {method.description && (
                          <div className="mt-6 border-t border-border/60 pt-4">
                            <h4 className="mb-2 font-semibold text-foreground">Description</h4>
                            <p className="text-sm leading-relaxed text-foreground">
                              {method.description}
                            </p>
                          </div>
                        )}
                      </div>
                    </TableCell>
                  </TableRow>
                )}
              </React.Fragment>
            );
          })}
        </TableBody>
      </Table>

      {selectedMethodForSDK && (
        <ViewSDKModal
          open={sdkModalOpen}
          onOpenChange={setSdkModalOpen}
          title={`SDK Code for ${selectedMethodForSDK.displayName || selectedMethodForSDK.name}`}
          description="Use this code to integrate this authentication method into your application."
          entityType="Auth Method"
          entityName={selectedMethodForSDK.displayName || selectedMethodForSDK.name || "Auth Method"}
          pythonCode={generateAuthMethodSDKCode({
            id: selectedMethodForSDK.id,
            name: selectedMethodForSDK.name,
            displayName: selectedMethodForSDK.displayName,
            providerType: selectedMethodForSDK.providerType,
            provider: selectedMethodForSDK.provider,
            methodKey: selectedMethodForSDK.methodKey,
            type: selectedMethodForSDK.type,
          }).python}
          typescriptCode={[]}
          docsLink="/docs/sdk/authentication"
        />
      )}
    </div>
  );
}
