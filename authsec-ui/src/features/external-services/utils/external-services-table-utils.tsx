import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
  DropdownMenuSeparator,
} from "@/components/ui/dropdown-menu";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  MoreHorizontal,
  Edit,
  Trash2,
  Code,
  Eye,
  Copy,
  Check,
  Lock,
  ChevronDown,
} from "lucide-react";
import type { ResponsiveColumnDef } from "@/components/ui/responsive-data-table";
import type { RawExternalService } from "@/app/api/externalServiceApi";
import { useLazyGetExternalServiceCredentialsQuery } from "@/app/api/externalServiceApi";
import { toast } from "@/lib/toast";

export interface ExternalServiceTableActions {
  onEdit: (service: RawExternalService) => void;
  onDelete: (service: RawExternalService) => void;
  onViewSDK: (service: RawExternalService) => void;
  onViewSecret: (service: RawExternalService) => void;
}

// Service Name Cell with type badge
function ServiceNameCell({
  service,
  row,
}: {
  service: RawExternalService;
  row: any;
}) {
  // The `row` object from the table library (like TanStack Table) contains
  // properties like `getCanExpand()` to check if the row is expandable.
  const canExpand = row.getCanExpand();

  return (
    <div className="flex items-center gap-2 group">
      <Button
        variant="ghost"
        size="sm"
        className={`h-8 w-8 p-0 transition-opacity ${
          row.getIsExpanded() // If already expanded, show it
            ? "opacity-100"
            : "opacity-0 group-hover:opacity-100" // Otherwise, show only on hover
        }`}
        onClick={canExpand ? row.getToggleExpandedHandler() : undefined} // Toggle expand/collapse only if expandable
        disabled={!canExpand}
      >
        <ChevronDown
          className={`h-4 w-4 transform transition-transform ${
            row.getIsExpanded() ? "rotate-180" : "rotate-0" // Rotate if expanded
          }`}
        />
      </Button>
      <div className="flex flex-col gap-1 flex-1">
        <span className="font-medium truncate" title={service.name}>
          {service.name}
        </span>
        {service.description && (
          <span
            className="text-xs text-foreground line-clamp-1"
            title={service.description}
          >
            {service.description}
          </span>
        )}
      </div>
    </div>
  );
}
// View SDK Cell
function SDKCell({
  service,
  onViewSDK,
}: {
  service: RawExternalService;
  onViewSDK: ExternalServiceTableActions["onViewSDK"];
}) {
  return (
    <Button
      variant="outline"
      size="sm"
      className="admin-tonal-cta h-8 px-3 gap-2"
      onClick={() => onViewSDK(service)}
      data-tone="sdk"
    >
      <Code className="h-4 w-4" />
      <span className="text-xs">View SDK Code</span>
    </Button>
  );
}
// Type Badge Cell
function TypeCell({ type }: { type: string }) {
  return (
    <Badge variant="outline" className="capitalize">
      {type}
    </Badge>
  );
}

// Credentials Cell - Shows API Key and Webhook Secret
function CredentialsCell({ service }: { service: RawExternalService }) {
  const [copiedField, setCopiedField] = React.useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = React.useState(false);
  const [fetchedCredentials, setFetchedCredentials] = React.useState<Record<
    string,
    any
  > | null>(null);
  const [fetchCredentials, { isLoading: isLoadingCredentials }] =
    useLazyGetExternalServiceCredentialsQuery();

  const handleCopy = (text: string, fieldName: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    toast.success("Copied to clipboard");
    setTimeout(() => setCopiedField(null), 2000);
  };

  const handleShowSecret = async () => {
    try {
      const result = await fetchCredentials(service.id).unwrap();
      setFetchedCredentials(result.credentials || result);
      setIsModalOpen(true);
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to retrieve credentials");
      console.error("Failed to fetch credentials:", error);
    }
  };

  return (
    <>
      <div className="flex items-center gap-2">
        <Button
          variant="ghost"
          size="sm"
          className="h-8 px-3 gap-2"
          onClick={handleShowSecret}
          disabled={isLoadingCredentials}
        >
          {isLoadingCredentials ? (
            <span className="text-xs">Loading...</span>
          ) : (
            <>
              <Lock className="h-4 w-4" />
              <span className="text-xs">Show Secret</span>
            </>
          )}
        </Button>
      </div>

      <Dialog open={isModalOpen} onOpenChange={setIsModalOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>Service Credentials</DialogTitle>
            <DialogDescription>
              {service.name} - {service.type}
            </DialogDescription>
          </DialogHeader>

          {fetchedCredentials && (
            <div className="space-y-4 mt-4">
              {fetchedCredentials.api_key && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-foreground">
                      API Key
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 px-2"
                      onClick={() =>
                        handleCopy(fetchedCredentials.api_key, "api_key")
                      }
                    >
                      {copiedField === "api_key" ? (
                        <>
                          <Check className="h-4 w-4 text-green-500 mr-2" />
                          <span className="text-xs">Copied</span>
                        </>
                      ) : (
                        <>
                          <Copy className="h-4 w-4 mr-2" />
                          <span className="text-xs">Copy</span>
                        </>
                      )}
                    </Button>
                  </div>
                  <div className="rounded-md bg-muted px-3 py-2 text-sm font-mono break-all">
                    {fetchedCredentials.api_key}
                  </div>
                </div>
              )}

              {fetchedCredentials.webhook_secret && (
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <span className="text-sm font-medium text-foreground">
                      Webhook Secret
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-8 px-2"
                      onClick={() =>
                        handleCopy(fetchedCredentials.webhook_secret, "webhook")
                      }
                    >
                      {copiedField === "webhook" ? (
                        <>
                          <Check className="h-4 w-4 text-green-500 mr-2" />
                          <span className="text-xs">Copied</span>
                        </>
                      ) : (
                        <>
                          <Copy className="h-4 w-4 mr-2" />
                          <span className="text-xs">Copy</span>
                        </>
                      )}
                    </Button>
                  </div>
                  <div className="rounded-md bg-muted px-3 py-2 text-sm font-mono break-all">
                    {fetchedCredentials.webhook_secret}
                  </div>
                </div>
              )}

              {fetchedCredentials.vault_path && (
                <div className="space-y-2">
                  <span className="text-sm font-medium text-foreground">
                    Vault Path
                  </span>
                  <div className="rounded-md bg-muted px-3 py-2 text-sm font-mono break-all">
                    {fetchedCredentials.vault_path}
                  </div>
                </div>
              )}
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  );
}

// Actions Cell
function ActionsCell({
  service,
  actions,
}: {
  service: RawExternalService;
  actions: ExternalServiceTableActions;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" visualVariant="row-actions" className="w-40">
        {/* <DropdownMenuItem onClick={() => actions.onViewSecret(service)}>
          <Eye className="mr-2 h-4 w-4" />
          View Secret
        </DropdownMenuItem> */}
        {/* <DropdownMenuItem onClick={() => actions.onViewSDK(service)}>
          <Code className="mr-2 h-4 w-4" />
          View SDK
        </DropdownMenuItem>
        <DropdownMenuSeparator /> */}
        <DropdownMenuItem onClick={() => actions.onEdit(service)}>
          <Edit className="mr-2 h-4 w-4" />
          Edit
        </DropdownMenuItem>
        <DropdownMenuItem
          onClick={() => actions.onDelete(service)}
          className="text-destructive focus:text-destructive"
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Delete
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// Expanded Row Content
export function ExternalServiceExpandedRow({
  service,
}: {
  service: RawExternalService;
}) {
  const [copiedField, setCopiedField] = React.useState<string | null>(null);
  const [showingSecret, setShowingSecret] = React.useState(false);
  const [fetchedCredentials, setFetchedCredentials] = React.useState<Record<
    string,
    any
  > | null>(null);
  const [fetchCredentials, { isLoading: isLoadingCredentials }] =
    useLazyGetExternalServiceCredentialsQuery();

  const handleCopy = (text: string, fieldName: string) => {
    navigator.clipboard.writeText(text);
    setCopiedField(fieldName);
    setTimeout(() => setCopiedField(null), 2000);
  };

  const handleShowSecret = async () => {
    if (showingSecret && fetchedCredentials) {
      setShowingSecret(false);
      return;
    }

    try {
      const result = await fetchCredentials(service.id).unwrap();
      setFetchedCredentials(result.credentials || result);
      setShowingSecret(true);
      toast.success("Credentials retrieved successfully");
    } catch (error: any) {
      toast.error(error?.data?.message || "Failed to retrieve credentials");
      console.error("Failed to fetch credentials:", error);
    }
  };

  return (
    <div className="px-0 py-0">
      <div className="grid gap-6 lg:grid-cols-2">
        <div className="space-y-5">
          <div className="space-y-3 rounded-lg border border-border bg-background/80 p-4">
            <h4 className="text-sm font-semibold text-foreground">
              Service Details
            </h4>
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-4 text-sm">
                <span className="text-foreground">Type</span>
                <span className="font-medium text-foreground">
                  {service.type}
                </span>
              </div>
              <div className="flex items-center justify-between gap-4 text-sm">
                <span className="text-foreground">Agent Accessible</span>
                <Badge
                  variant={service.agent_accessible ? "default" : "secondary"}
                >
                  {service.agent_accessible ? "Yes" : "No"}
                </Badge>
              </div>
            </div>
          </div>

          {service.url && (
            <div className="space-y-3 rounded-lg border border-border bg-background/80 p-4">
              <h4 className="text-sm font-semibold text-foreground">
                Endpoint
              </h4>
              <div className="space-y-2 text-xs font-mono text-foreground">
                <button
                  type="button"
                  onClick={() => handleCopy(service.url, "url")}
                  className="w-full truncate rounded-md bg-black/5 dark:bg-white/10 px-3 py-2 text-left hover:bg-black/10 dark:hover:bg-white/15 flex items-center justify-between gap-2"
                  title={service.url}
                >
                  <span className="truncate">{service.url}</span>
                  {copiedField === "url" ? (
                    <Check className="h-3 w-3 flex-shrink-0 text-green-500" />
                  ) : (
                    <Copy className="h-3 w-3 flex-shrink-0 text-foreground" />
                  )}
                </button>
              </div>
            </div>
          )}

          {service.tags && service.tags.length > 0 && (
            <div className="space-y-3 rounded-lg border border-border bg-background/80 p-4">
              <h4 className="text-sm font-semibold text-foreground">Tags</h4>
              <div className="flex flex-wrap gap-2">
                {service.tags.map((tag) => (
                  <Badge key={tag} variant="outline" className="text-[11px]">
                    {tag}
                  </Badge>
                ))}
              </div>
            </div>
          )}
        </div>

        <div className="space-y-5">
          <div className="space-y-3 rounded-lg border border-border bg-background/80 p-4">
            <h4 className="text-sm font-semibold text-foreground">Metadata</h4>
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-4 text-sm">
                <span className="text-foreground">Updated</span>
                <span className="font-medium text-foreground">
                  {new Date(service.updated_at).toLocaleDateString()}
                </span>
              </div>
              {service.resource_id && (
                <div className="flex items-center justify-between gap-4 text-sm">
                  <span className="text-foreground">Resource ID</span>
                  <span className="font-mono text-xs text-foreground">
                    {service.resource_id}
                  </span>
                </div>
              )}
            </div>
          </div>

          {/* Credentials Section */}
          {service.vault_path && (
            <div className="space-y-3 rounded-lg border border-amber-200 dark:border-amber-800 bg-amber-50/50 dark:bg-amber-950/20 p-4">
              <div className="flex items-center justify-between">
                <h4 className="text-sm font-semibold text-foreground">
                  Credentials
                </h4>
                <Badge
                  variant="outline"
                  className="text-[10px] border-amber-300 dark:border-amber-700"
                >
                  Protected
                </Badge>
              </div>

              <div className="space-y-3">
                <Button
                  variant="outline"
                  size="sm"
                  className="w-full"
                  onClick={handleShowSecret}
                  disabled={isLoadingCredentials}
                >
                  <Lock className="mr-2 h-4 w-4" />
                  {isLoadingCredentials
                    ? "Loading..."
                    : showingSecret
                    ? "Hide Secret"
                    : "Show Secret"}
                </Button>

                {showingSecret && fetchedCredentials && (
                  <div className="space-y-2 animate-in fade-in duration-300">
                    {fetchedCredentials.api_key && (
                      <div className="space-y-1">
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-medium text-foreground">
                            API Key
                          </span>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 px-2"
                            onClick={() =>
                              handleCopy(
                                fetchedCredentials.api_key,
                                "credential-api-key"
                              )
                            }
                          >
                            {copiedField === "credential-api-key" ? (
                              <Check className="h-3 w-3 text-green-500" />
                            ) : (
                              <Copy className="h-3 w-3" />
                            )}
                          </Button>
                        </div>
                        <div className="rounded-md bg-black/5 dark:bg-white/10 px-3 py-2 text-xs font-mono break-all">
                          {fetchedCredentials.api_key}
                        </div>
                      </div>
                    )}
                    {fetchedCredentials.webhook_secret && (
                      <div className="space-y-1">
                        <div className="flex items-center justify-between">
                          <span className="text-xs font-medium text-foreground">
                            Webhook Secret(Optional)
                          </span>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="h-6 px-2"
                            onClick={() =>
                              handleCopy(
                                fetchedCredentials.webhook_secret,
                                "credential-webhook"
                              )
                            }
                          >
                            {copiedField === "credential-webhook" ? (
                              <Check className="h-3 w-3 text-green-500" />
                            ) : (
                              <Copy className="h-3 w-3" />
                            )}
                          </Button>
                        </div>
                        <div className="rounded-md bg-black/5 dark:bg-white/10 px-3 py-2 text-xs font-mono break-all">
                          {fetchedCredentials.webhook_secret}
                        </div>
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// Create table columns - Only showing credentials
export function createExternalServiceTableColumns(
  actions: ExternalServiceTableActions
): ResponsiveColumnDef<RawExternalService, any>[] {
  return [
    {
      id: "name",
      accessorKey: "name",
      header: "Service Name",
      cell: ({ row }) => <ServiceNameCell service={row.original} row={row} />,
      enableSorting: true,
      responsive: true,
      className: "min-w-[200px]",
    },
    {
      id: "type",
      accessorKey: "type",
      header: "Type",
      cell: ({ row }) => <TypeCell type={row.original.type} />,
      enableSorting: true,
      responsive: true,
      className: "min-w-[100px]",
    },
    {
      id: "credentials",
      header: "Credentials",
      cell: ({ row }) => <CredentialsCell service={row.original} />,
      enableSorting: false,
      responsive: true,
      className: "min-w-[200px]",
    },
    {
      id: "sdk",
      header: "SDK's",
      cell: ({ row }) => (
        <SDKCell service={row.original} onViewSDK={actions.onViewSDK} />
      ),
      enableSorting: false,
      responsive: true,
      className: "min-w-[100px]",
    },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => (
        <ActionsCell service={row.original} actions={actions} />
      ),
      enableSorting: false,
      enableHiding: false,
      responsive: false,
      className: "w-[100px]",
      cellClassName: "text-center",
    },
  ];
}
