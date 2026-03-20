import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { Input } from "../../components/ui/input";
import { Label } from "../../components/ui/label";
import { Badge } from "../../components/ui/badge";
import { ArrowLeft, Send, Plus, X, Loader2, Info } from "lucide-react";
import { toast } from "react-hot-toast";
import { SessionManager } from "../../utils/sessionManager";
import {
  useRegisterWorkloadMutation,
  useUpdateWorkloadMutation,
  type RegisterWorkloadRequest,
  type UpdateWorkloadRequest,
  type K8sSelectors,
  useGetWorkloadQuery,
} from "../../app/api/workloadsApi";

type SelectorField = {
  id: string;
  key: string;
  value: string;
};

const PARENT_AGENTS = [
  { value: "spiffe://authsec.dev/agent/node-1", label: "Node 1 Agent" },
  { value: "spiffe://authsec.dev/agent/node-2", label: "Node 2 Agent" },
  { value: "spiffe://authsec.dev/agent/node-3", label: "Node 3 Agent" },
  { value: "spiffe://authsec.dev/agent/k8s-cluster-1", label: "K8s Cluster 1" },
];

const PLATFORMS = [
  { value: "kubernetes", label: "Kubernetes" },
  { value: "docker", label: "Docker" },
  { value: "unix", label: "Unix" },
  { value: "vm", label: "Virtual Machine" },
];

const COMMON_SELECTOR_KEYS = [
  "k8s:namespace",
  "k8s:pod",
  "k8s:pod-name",
  "k8s:service-account",
  "k8s:sa",
  "node:name",
  "node:type",
];

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

export function CreateWorkloadPage() {
  const navigate = useNavigate();
  const { id: workloadId } = useParams<{ id: string }>();
  const isEditMode = Boolean(workloadId);

  const [registerWorkload, { isLoading: isRegistering }] =
    useRegisterWorkloadMutation();
  const [updateWorkload, { isLoading: isUpdating }] =
    useUpdateWorkloadMutation();

  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id || "";

  const { data: existingWorkload } = useGetWorkloadQuery(
    { workload_id: workloadId || "" },
    { skip: !isEditMode || !workloadId || !sessionData?.token }
  );

  // Form fields
  const [spiffeId, setSpiffeId] = useState("");
  const [parentAgent, setParentAgent] = useState("");
  const [agentName, setAgentName] = useState("");
  const [parentAgentId, setParentAgentId] = useState("");
  const [platform, setPlatform] = useState("");
  const [selectors, setSelectors] = useState<SelectorField[]>([
    { id: crypto.randomUUID(), key: "", value: "" },
  ]);

  // Load workload data in edit mode
  useEffect(() => {
    if (isEditMode && existingWorkload && workloadId) {
      if (existingWorkload.spiffe_id || existingWorkload.spiffeId) {
        setSpiffeId(
          existingWorkload.spiffe_id || existingWorkload.spiffeId || ""
        );
      }

      // Load platform from attestation_type
      if (existingWorkload.attestation_type) {
        setPlatform(existingWorkload.attestation_type);
      }

      // Load selectors
      if (
        existingWorkload.selectors &&
        typeof existingWorkload.selectors === "object" &&
        !Array.isArray(existingWorkload.selectors)
      ) {
        const selectorEntries = Object.entries(
          existingWorkload.selectors as K8sSelectors
        )
          .filter(([, val]) => val !== undefined && val !== "")
          .map(([key, val]) => ({
            id: crypto.randomUUID(),
            key,
            value: val as string,
          }));

        if (selectorEntries.length > 0) {
          setSelectors(selectorEntries);
        }
      }
    }
  }, [isEditMode, existingWorkload, workloadId]);

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

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();

    // Validation
    if (!isEditMode && !spiffeId.trim()) {
      toast.error("SPIFFE ID is required");
      return;
    }

    if (!isEditMode && !parentAgent) {
      toast.error("Parent Agent is required");
      return;
    }

    if (!isEditMode && !parentAgentId.trim()) {
      toast.error("Parent Agent ID is required");
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

    try {
      // Build selectors object
      const selectorsObj: K8sSelectors = {};
      selectors.forEach((selector) => {
        if (selector.key.trim() && selector.value.trim()) {
          selectorsObj[selector.key.trim()] = selector.value.trim();
        }
      });

      if (isEditMode && workloadId) {
        const updatePayload: UpdateWorkloadRequest = {
          workload_id: workloadId,
          selectors: selectorsObj,
          vault_role: undefined,
          status: "active",
          attestation_type: platform || "kubernetes",
        };
        if (tenantId) {
          updatePayload.tenant_id = tenantId;
        }

        await updateWorkload(updatePayload).unwrap();
        toast.success("Workload updated successfully!");
        navigate("/clients/workloads");
      } else {
        const payload: RegisterWorkloadRequest = {
          selectors: selectorsObj,
          vault_role: undefined,
          status: "active",
          attestation_type: platform || "kubernetes",
        };
        if (tenantId) {
          payload.tenant_id = tenantId;
        }
        if (spiffeId.trim()) {
          payload.spiffe_id = spiffeId.trim();
        }
        if (parentAgentId.trim()) {
          payload.parent_id = parentAgentId.trim();
        }
        if (platform) {
          payload.type = platform;
        }

        await registerWorkload(payload).unwrap();
        toast.success("Workload registered successfully!");
        navigate("/clients/workloads");
      }
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const isLoading = isRegistering || isUpdating;
  const hasValidSelectors = selectors.some(
    (s) => s.key.trim() && s.value.trim()
  );
  const canSubmit = Boolean(
    (isEditMode || (spiffeId.trim() && parentAgent && parentAgentId.trim())) &&
      hasValidSelectors &&
      tenantId &&
      !isLoading
  );

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="container mx-auto space-y-6 px-6 py-8">
        {/* Header */}
        <header className="bg-card border border-border rounded-lg p-6 shadow-sm">
          <div className="flex items-start justify-between gap-4">
            <div className="flex items-center gap-3">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => navigate("/clients/workloads")}
                className="h-8 px-2"
              >
                <ArrowLeft className="h-4 w-4" />
              </Button>
              <div>
                <h1 className="text-2xl font-semibold">
                  {isEditMode ? "Edit Workload" : "Register Workload"}
                </h1>
                <p className="text-sm text-foreground mt-1">
                  {isEditMode
                    ? "Update workload configuration and selectors"
                    : "Create a new workload registration with SPIFFE identity"}
                </p>
              </div>
            </div>
            <Button
              type="submit"
              form="register-workload-form"
              disabled={!canSubmit}
              className="min-w-[140px]"
            >
              {isLoading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  {isEditMode ? "Updating..." : "Registering..."}
                </>
              ) : (
                <>
                  <Send className="mr-2 h-4 w-4" />
                  {isEditMode ? "Update" : "Register"}
                </>
              )}
            </Button>
          </div>
        </header>

        <form
          id="register-workload-form"
          onSubmit={handleSubmit}
          className="space-y-6"
        >
          {/* SPIFFE Identity */}
          <div className="bg-card border border-border rounded-lg p-6 shadow-sm space-y-4">
            <div className="space-y-1">
              <h3 className="text-lg font-medium text-foreground">
                SPIFFE Identity
              </h3>
              <p className="text-foreground text-sm">
                Unique identifier for the workload
              </p>
            </div>

            <div className="space-y-2">
              <Label
                htmlFor="spiffeId"
                className="text-sm font-medium flex items-center gap-2"
              >
                SPIFFE ID
                {!isEditMode && <span className="text-destructive">*</span>}
              </Label>
              <Input
                id="spiffeId"
                value={spiffeId}
                onChange={(e) => setSpiffeId(e.target.value)}
                placeholder="spiffe://authsec.dev/workload/my-service"
                className="h-10 font-mono"
                required={!isEditMode}
                readOnly={isEditMode}
              />
              <p className="text-xs text-foreground">
                Format: spiffe://domain/path/to/workload
                {isEditMode && " (read-only)"}
              </p>
            </div>
          </div>

          {/* Parent Agent Configuration - Only show in create mode */}
          {!isEditMode && (
            <div className="bg-card border border-border rounded-lg p-6 shadow-sm space-y-4">
              <div className="space-y-1">
                <h3 className="text-lg font-medium text-foreground">
                  Parent Agent
                </h3>
                <p className="text-foreground text-sm">
                  Configure the parent agent for this workload
                </p>
              </div>

              <div className="grid grid-cols-1 gap-4">
                <div className="space-y-2">
                  <Label
                    htmlFor="parentAgent"
                    className="text-sm font-medium flex items-center gap-2"
                  >
                    Select Parent Agent
                    <span className="text-destructive">*</span>
                  </Label>
                  <Select
                    value={parentAgent}
                    onValueChange={setParentAgent}
                    required
                  >
                    <SelectTrigger id="parentAgent" className="h-10">
                      <SelectValue placeholder="Choose a parent agent..." />
                    </SelectTrigger>
                    <SelectContent>
                      {PARENT_AGENTS.map((agent) => (
                        <SelectItem key={agent.value} value={agent.value}>
                          {agent.label}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>

                <div className="space-y-2">
                  <Label htmlFor="agentName" className="text-sm font-medium">
                    Agent Name
                  </Label>
                  <Input
                    id="agentName"
                    value={agentName}
                    onChange={(e) => setAgentName(e.target.value)}
                    placeholder="e.g., payment-service-agent"
                    className="h-10"
                  />
                  <p className="text-xs text-foreground">
                    Optional friendly name for the agent
                  </p>
                </div>

                <div className="space-y-2">
                  <Label
                    htmlFor="parentAgentId"
                    className="text-sm font-medium flex items-center gap-2"
                  >
                    Parent Agent ID
                    <span className="text-destructive">*</span>
                  </Label>
                  <Input
                    id="parentAgentId"
                    value={parentAgentId}
                    onChange={(e) => setParentAgentId(e.target.value)}
                    placeholder="Enter parent agent ID"
                    className="h-10 font-mono"
                    required
                  />
                </div>
              </div>
            </div>
          )}

          {/* Platform */}
          <div className="bg-card border border-border rounded-lg p-6 shadow-sm space-y-4">
            <div className="space-y-1">
              <h3 className="text-lg font-medium text-foreground">Platform</h3>
              <p className="text-foreground text-sm">
                Workload platform type
              </p>
            </div>

            <div className="space-y-2">
              <Label htmlFor="platform" className="text-sm font-medium">
                Platform Type
              </Label>
              <Select value={platform} onValueChange={setPlatform}>
                <SelectTrigger id="platform" className="h-10">
                  <SelectValue placeholder="Select platform..." />
                </SelectTrigger>
                <SelectContent>
                  {PLATFORMS.map((p) => (
                    <SelectItem key={p.value} value={p.value}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Selectors */}
          <div className="bg-card border border-border rounded-lg p-6 shadow-sm space-y-4">
            <div className="space-y-1">
              <h3 className="text-lg font-medium text-foreground flex items-center gap-2">
                Selectors
                <span className="text-destructive">*</span>
              </h3>
              <p className="text-foreground text-sm">
                Define workload selectors for attestation (at least one
                required)
              </p>
            </div>

            <div className="space-y-3">
              {selectors.map((selector, index) => (
                <div key={selector.id} className="flex gap-3 items-start">
                  <div className="flex-1 grid grid-cols-2 px-5 gap-3">
                    <div className="space-y-2">
                      <Label htmlFor={`key-${selector.id}`} className="text-sm">
                        Key{" "}
                        {index === 0 && (
                          <span className="text-destructive">*</span>
                        )}
                      </Label>
                      <Input
                        id={`key-${selector.id}`}
                        list={`selector-keys-${selector.id}`}
                        value={selector.key}
                        onChange={(e) =>
                          handleSelectorChange(
                            selector.id,
                            "key",
                            e.target.value
                          )
                        }
                        placeholder="e.g., k8s:namespace"
                        className="h-10"
                      />
                      <datalist id={`selector-keys-${selector.id}`}>
                        {COMMON_SELECTOR_KEYS.map((key) => (
                          <option key={key} value={key} />
                        ))}
                      </datalist>
                    </div>

                    <div className="space-y-2">
                      <Label
                        htmlFor={`value-${selector.id}`}
                        className="text-sm"
                      >
                        Value{" "}
                        {index === 0 && (
                          <span className="text-destructive">*</span>
                        )}
                      </Label>
                      <Input
                        id={`value-${selector.id}`}
                        value={selector.value}
                        onChange={(e) =>
                          handleSelectorChange(
                            selector.id,
                            "value",
                            e.target.value
                          )
                        }
                        placeholder="e.g., production"
                        className="h-10"
                      />
                    </div>
                  </div>

                  {selectors.length > 1 && (
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      onClick={() => handleRemoveSelector(selector.id)}
                      className="mt-8 text-red-600 hover:text-red-700 hover:bg-red-50 dark:hover:bg-red-950/20"
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  )}
                </div>
              ))}
              <div className="mx-5">
                <Button
                  type="button"
                  variant="outline"
                  onClick={handleAddSelector}
                  className="mt-2"
                >
                  <Plus className="mr-2 h-4 w-4" />
                  Add Selector
                </Button>
              </div>

              {!hasValidSelectors && (
                <div className="flex items-start gap-2 p-3 bg-yellow-50 dark:bg-yellow-950/20 border border-yellow-200 dark:border-yellow-800 rounded-md">
                  <Info className="h-4 w-4 text-yellow-600 dark:text-yellow-400 mt-0.5 flex-shrink-0" />
                  <p className="text-sm text-yellow-800 dark:text-yellow-200">
                    At least one selector with both key and value is required
                  </p>
                </div>
              )}

              <div className="bg-blue-50 dark:bg-blue-950/20 border border-blue-200 dark:border-blue-800 rounded-lg p-4 mt-4">
                <p className="text-sm text-blue-800 dark:text-blue-200">
                  <strong>Example Selectors:</strong>
                  <br />
                  • k8s:namespace → production
                  <br />
                  • k8s:pod-name → web-service-pod-1
                  <br />
                  • k8s:sa → database-client
                  <br />• node:name → node-1
                </p>
              </div>
            </div>
          </div>
        </form>
      </div>
    </div>
  );
}
