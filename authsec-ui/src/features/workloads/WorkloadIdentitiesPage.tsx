import { useState, useEffect } from "react";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { Card, CardContent } from "../../components/ui/card";

import { TableCard } from "@/theme/components/cards";
import { Badge } from "../../components/ui/badge";
import {
  Plus,
  X,
  Search,
  RefreshCw,
  Info,
  ArrowLeft,
  Edit,
  Loader2,
  Copy,
  Clock,
  Trash2,
} from "lucide-react";
import { toast } from "react-hot-toast";
import { SessionManager } from "../../utils/sessionManager";
import { useNavigate, useParams, useLocation } from "react-router-dom";
import { detectSubdomain } from "../../utils/subdomainUtils";
import {
  useRegisterEntryMutation,
  useUpdateEntryMutation,
  useDeleteEntryMutation,
  useListEntriesQuery,
  useGetEntryQuery,
  useListAgentsQuery,
} from "../../app/api/workloadsApi";

const getErrorMessage = (error: unknown): string => {
  if (!error) return "Operation failed.";

  if (typeof error === "string") {
    return error;
  }

  if (typeof error === "object") {
    const err = error as { status?: number; data?: unknown; error?: string };
    if (err.data && typeof err.data === "object") {
      const data = err.data as {
        message?: string;
        error?: string;
        detail?: string;
      };
      if (data.message) return data.message;
      if (data.error) return data.error;
      if (data.detail) return data.detail;
    }

    if (err.error) {
      return err.error;
    }

    if (typeof err.status === "number") {
      return `Request failed (${err.status})`;
    }
  }

  return "Operation failed.";
};

type SelectorField = {
  id: string;
  key: string;
  value: string;
};

type WorkloadEntry = {
  id: string;
  spiffe_id: string;
  parent_id: string;
  selectors: Record<string, string>;
  ttl: number;
  admin?: boolean;
  downstream?: boolean;
  created_at: string;
};

const PLATFORMS = [
  { value: "kubernetes", label: "Kubernetes" },
  { value: "docker", label: "Docker" },
  { value: "unix", label: "Unix" },
];

const PLATFORM_TEMPLATES: Record<string, SelectorField[]> = {
  kubernetes: [
    { id: crypto.randomUUID(), key: "k8s:ns", value: "" },
    { id: crypto.randomUUID(), key: "k8s:pod-label:app", value: "" },
  ],
  docker: [{ id: crypto.randomUUID(), key: "docker:label:app", value: "" }],
  unix: [{ id: crypto.randomUUID(), key: "unix:uid", value: "" }],
};

const COMMON_SELECTOR_KEYS = [
  "k8s:ns",
  "k8s:pod-label:app",
  "k8s:sa",
  "k8s:pod",
  "k8s:pod-name",
  "k8s:service-account",
  "docker:label",
  "docker:image",
  "unix:uid",
  "unix:gid",
  "node:name",
  "node:type",
];

// Function to get the base domain (extract domain, removing tenant subdomain)
function getFullDomain(): string {
  const hostname = window.location.hostname;

  // For local development
  if (
    hostname === "localhost" ||
    hostname === "127.0.0.1" ||
    hostname.startsWith("192.168")
  ) {
    return hostname;
  }

  // Return the full hostname (including tenant subdomain like ssh.app.authsec.dev)
  return hostname;
}

// Dynamic SPIFFE domain prefix
function getSpiffeDomainPrefix(): string {
  const fullDomain = getFullDomain();
  return `spiffe://${fullDomain}/workload/`;
}

export function WorkloadIdentitiesPage() {
  const sessionData = SessionManager.getSession();
  const tenantId = (sessionData?.tenant_id || "").replace(/['"]/g, "");
  const navigate = useNavigate();
  const location = useLocation();
  const { id: entryId } = useParams<{ id: string }>();
  const isEditMode = Boolean(entryId);

  // Get dynamic SPIFFE domain prefix
  const SPIFFE_DOMAIN_PREFIX = getSpiffeDomainPrefix();

  const [registerEntry, { isLoading: isRegistering }] =
    useRegisterEntryMutation();
  const [updateEntry, { isLoading: isUpdating }] = useUpdateEntryMutation();
  const [deleteEntry, { isLoading: isDeleting }] = useDeleteEntryMutation();

  const {
    data: entries = [],
    isLoading: isLoadingEntries,
    refetch: refetchEntries,
  } = useListEntriesQuery({
    tenant_id: tenantId,
    limit: 10,
    offset: 0,
  });
  const { data: editEntry, error: editEntryError } = useGetEntryQuery(
    { entry_id: entryId || "", tenant_id: tenantId },
    { skip: !isEditMode || !entryId }
  );

  const isLoading = isRegistering || isUpdating;

  // Form state - workload name only (not full SPIFFE ID)
  const [workloadName, setWorkloadName] = useState("");
  const [parentId, setParentId] = useState("");
  const [platform, setPlatform] = useState("kubernetes");
  const [ttl, setTtl] = useState("3600");
  const [isAdmin, setIsAdmin] = useState(false);
  const [isDownstream, setIsDownstream] = useState(false);
  const [selectors, setSelectors] = useState<SelectorField[]>(
    PLATFORM_TEMPLATES.kubernetes
  );

  // List state
  const [searchQuery, setSearchQuery] = useState("");
  const { data: agents = [], refetch: refetchAgents } = useListAgentsQuery();

  // Load entry data in edit mode
  useEffect(() => {
    if (isEditMode) {
      if (editEntryError) {
        toast.error("Failed to load entry for editing");
        navigate("/clients/workloads/create");
      } else if (editEntry) {
        // Extract workload name from full SPIFFE ID
        const extractedName = editEntry.spiffe_id.replace(
          SPIFFE_DOMAIN_PREFIX,
          ""
        );
        setWorkloadName(extractedName);
        setParentId(editEntry.parent_id);
        setTtl(editEntry.ttl.toString());
        setIsAdmin(editEntry.admin || false);
        setIsDownstream(editEntry.downstream || false);

        // Load selectors
        if (editEntry.selectors) {
          const selectorEntries = Object.entries(editEntry.selectors).map(
            ([key, val]) => ({
              id: crypto.randomUUID(),
              key,
              value: val as string,
            })
          );
          if (selectorEntries.length > 0) {
            setSelectors(selectorEntries);
          }
        }
      }
    }
  }, [isEditMode, editEntry, editEntryError, navigate]);

  const audienceCopy = isEditMode
    ? {
        title: "Edit Workload",
        description: "Update workload identity configuration",
      }
    : {
        title: "Register Workload",
        description: "Manage and register workload identities",
      };

  const handlePlatformChange = (value: string) => {
    setPlatform(value);
    setSelectors(
      PLATFORM_TEMPLATES[value as keyof typeof PLATFORM_TEMPLATES] || []
    );
  };

  const handleAddSelector = () => {
    setSelectors([
      ...selectors,
      { id: crypto.randomUUID(), key: "", value: "" },
    ]);
  };

  const handleRemoveSelector = (id: string) => {
    if (selectors.length > 1) {
      setSelectors(selectors.filter((s) => s.id !== id));
    }
  };

  const handleSelectorChange = (
    id: string,
    field: "key" | "value",
    value: string
  ) => {
    setSelectors(
      selectors.map((s) => (s.id === id ? { ...s, [field]: value } : s))
    );
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    // Validation
    if (!workloadName.trim()) {
      toast.error("Workload name is required");
      return;
    }

    if (!isEditMode && !parentId) {
      toast.error("Parent Agent is required");
      return;
    }

    const hasValidSelectors = selectors.some(
      (s) => s.key.trim() && s.value.trim()
    );
    if (!hasValidSelectors) {
      toast.error("At least one selector is required");
      return;
    }

    // Check for duplicate selector keys
    const keys = selectors.filter((s) => s.key.trim()).map((s) => s.key.trim());
    const uniqueKeys = new Set(keys);
    if (keys.length !== uniqueKeys.size) {
      toast.error("Duplicate selector keys are not allowed");
      return;
    }

    // Build selectors object
    const selectorsObj: Record<string, string> = {};
    selectors.forEach((selector) => {
      if (selector.key.trim() && selector.value.trim()) {
        selectorsObj[selector.key.trim()] = selector.value.trim();
      }
    });

    // Build full SPIFFE ID
    const fullSpiffeId = `${SPIFFE_DOMAIN_PREFIX}${workloadName.trim()}`;

    try {
      if (isEditMode && entryId) {
        // UPDATE mode
        await updateEntry({
          entry_id: entryId,
          tenant_id: tenantId,
          spiffe_id: fullSpiffeId,
          parent_id: parentId,
          selectors: selectorsObj,
          ttl: parseInt(ttl) || 3600,
          admin: isAdmin,
          downstream: isDownstream,
        }).unwrap();

        toast.success("Workload entry updated successfully!");
        navigate("/clients/workloads/create");
      } else {
        // CREATE mode
        await registerEntry({
          tenant_id: tenantId,
          spiffe_id: fullSpiffeId,
          parent_id: parentId,
          selectors: selectorsObj,
          ttl: parseInt(ttl) || 3600,
          admin: isAdmin,
          downstream: isDownstream,
        }).unwrap();

        toast.success("Workload entry registered successfully!");

        // Navigate to workload certificates page with success state for wizard
        navigate("/clients/workloads", { state: { workloadCreated: true } });
      }
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleEdit = (entry: WorkloadEntry) => {
    navigate(`/clients/workloads/edit/${entry.id}`);
  };

  const handleDelete = async (entry: WorkloadEntry) => {
    if (
      !confirm(
        `Are you sure you want to delete the entry "${entry.spiffe_id}"?`
      )
    ) {
      return;
    }

    try {
      await deleteEntry({
        entry_id: entry.id,
        tenant_id: tenantId,
      }).unwrap();

      toast.success("Workload entry deleted successfully!");
      refetchEntries();
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleCancel = () => {
    if (isEditMode) {
      navigate("/clients/workloads/create");
    } else {
      navigate("/clients/workloads");
    }
  };

  const filteredEntries = entries.filter((entry) =>
    entry.spiffe_id.toLowerCase().includes(searchQuery.toLowerCase())
  );

  const hasValidSelectors = selectors.some(
    (s) => s.key.trim() && s.value.trim()
  );
  const canSubmit = Boolean(
    workloadName.trim() &&
      (isEditMode || parentId) &&
      hasValidSelectors &&
      tenantId &&
      !isLoading
  );

  return (
    <div className="min-h-screen">
      <div className="space-y-6 p-6 max-w-[1600px] mx-auto">
        <header className="bg-card border border-border rounded-sm p-5 shadow-sm">
          <div className="flex justify-between items-center gap-4">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={handleCancel}
                className="h-8 px-2"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <div>
                <h1 className="text-2xl font-semibold">{audienceCopy.title}</h1>
                <p className="text-sm text-foreground mt-1">
                  {audienceCopy.description}
                </p>
              </div>
            </div>
            <Button
              type="submit"
              form="workload-form"
              disabled={!canSubmit}
              className="min-w-[140px]"
            >
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {isEditMode ? "Updating..." : "Registering..."}
                </>
              ) : isEditMode ? (
                "Update Workload"
              ) : (
                "Register Workload"
              )}
            </Button>
          </div>
        </header>

        <div className="flex justify-center">
          {/* Registration/Edit Form */}
          <Card className="!max-w-none w-[95vw]">
            <CardContent className="p-6">
              <h2 className="text-lg font-semibold mb-6">
                {isEditMode ? "Edit Workload" : "Register New Workload"}
              </h2>

              <form
                id="workload-form"
                onSubmit={handleSubmit}
                className="space-y-4"
              >
                {/* Workload Name Input */}
                <div className="space-y-2">
                  <Label
                    htmlFor="workload-name"
                    className="flex items-center gap-2"
                  >
                    Workload ID <span className="text-destructive">*</span>
                  </Label>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-foreground font-mono whitespace-nowrap">
                      {SPIFFE_DOMAIN_PREFIX}
                    </span>
                    <Input
                      id="workload-name"
                      value={workloadName}
                      onChange={(e) => setWorkloadName(e.target.value)}
                      placeholder="service-a"
                      className="font-mono text-sm flex-1"
                      required
                      readOnly={isEditMode}
                      disabled={isEditMode}
                    />
                  </div>
                  <p className="text-xs text-foreground">
                    Enter a name for your workload (e.g., service-a,
                    api-gateway, database)
                    {isEditMode && " (read-only)"}
                  </p>
                  {workloadName && (
                    <div className="mt-2 p-2 bg-muted rounded text-xs font-mono">
                      Full SPIFFE ID: {SPIFFE_DOMAIN_PREFIX}
                      {workloadName}
                    </div>
                  )}
                </div>

                {/* Parent Agent - Only in create mode */}
                {!isEditMode && (
                  <div className="space-y-2">
                    <Label
                      htmlFor="parent-id"
                      className="flex items-center gap-2"
                    >
                      Parent Agent ID{" "}
                      <span className="text-destructive">*</span>
                    </Label>
                    <div className="flex gap-2">
                      <Select
                        value={parentId}
                        onValueChange={setParentId}
                        required
                      >
                        <SelectTrigger
                          id="parent-id"
                          className="flex-1 truncate"
                        >
                          <SelectValue placeholder="Select an agent..." />
                        </SelectTrigger>
                        <SelectContent>
                          {agents.length === 0 ? (
                            <SelectItem value="none" disabled>
                              No agents available
                            </SelectItem>
                          ) : (
                            agents.map((agent) => (
                              <SelectItem
                                key={agent.spiffe_id}
                                value={agent.spiffe_id}
                                className="font-mono text-sm"
                              >
                                <span
                                  className="block truncate"
                                  title={agent.spiffe_id}
                                >
                                  {agent.spiffe_id}
                                </span>
                              </SelectItem>
                            ))
                          )}
                        </SelectContent>
                      </Select>
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        onClick={() => refetchAgents()}
                        title="Refresh agents"
                      >
                        <RefreshCw className="h-4 w-4" />
                      </Button>
                    </div>
                    <p className="text-xs text-foreground">
                      Select the agent SPIFFE ID managing this workload
                    </p>
                  </div>
                )}

                {/* Platform */}
                {!isEditMode && (
                  <div className="space-y-2">
                    <Label>Platform</Label>
                    <div className="grid grid-cols-3 gap-2">
                      {PLATFORMS.map((p) => (
                        <Button
                          key={p.value}
                          type="button"
                          variant={platform === p.value ? "default" : "outline"}
                          onClick={() => handlePlatformChange(p.value)}
                          className="w-full"
                        >
                          {p.label}
                        </Button>
                      ))}
                    </div>
                  </div>
                )}

                {/* Selectors */}
                <div className="space-y-2">
                  <Label className="flex items-center gap-2">
                    Selectors <span className="text-destructive">*</span>
                  </Label>
                  <div className="space-y-2">
                    {selectors.map((selector, _index) => (
                      <div key={selector.id} className="flex gap-2">
                        <Input
                          list="selector-keys"
                          value={selector.key}
                          onChange={(e) =>
                            handleSelectorChange(
                              selector.id,
                              "key",
                              e.target.value
                            )
                          }
                          placeholder="key"
                          className="flex-1"
                        />
                        <Input
                          value={selector.value}
                          onChange={(e) =>
                            handleSelectorChange(
                              selector.id,
                              "value",
                              e.target.value
                            )
                          }
                          placeholder="value"
                          className="flex-1"
                        />
                        {selectors.length > 1 && (
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            onClick={() => handleRemoveSelector(selector.id)}
                            className="text-red-600 hover:text-red-700 hover:bg-red-50"
                          >
                            <X className="h-4 w-4" />
                          </Button>
                        )}
                      </div>
                    ))}
                    <datalist id="selector-keys">
                      {COMMON_SELECTOR_KEYS.map((key) => (
                        <option key={key} value={key} />
                      ))}
                    </datalist>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    onClick={handleAddSelector}
                    className="w-full"
                  >
                    <Plus className="h-4 w-4 mr-2" />
                    Add Selector
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
