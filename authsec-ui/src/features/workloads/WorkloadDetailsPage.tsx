import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../../components/ui/card";
import { Badge } from "../../components/ui/badge";
import { PageHeader } from "@/components/layout/PageHeader";
import {
  ArrowLeft,
  Edit,
  Trash2,
  RefreshCw,
  Server,
  Key,
  Shield,
  Clock,
  Activity,
  CheckCircle2,
  XCircle,
  AlertCircle,
  Copy,
  Eye,
  EyeOff,
  Download,
} from "lucide-react";
import { toast } from "react-hot-toast";
import { SessionManager } from "../../utils/sessionManager";
import {
  useGetWorkloadQuery,
  useDeleteWorkloadMutation,
} from "../../app/api/workloadsApi";
import { DeleteConfirmDialog } from "./components/DeleteConfirmDialog";
import { CopyButton } from "../../components/ui/copy-button";

const getErrorMessage = (error: unknown): string => {
  if (!error) return "An error occurred.";
  if (typeof error === "string") return error;
  if (typeof error === "object") {
    const err = error as { status?: number; data?: unknown; error?: string };
    if (err.data && typeof err.data === "object") {
      const data = err.data as { message?: string; error?: string; detail?: string };
      if (data.message) return data.message;
      if (data.error) return data.error;
      if (data.detail) return data.detail;
    }
    if (err.error) return err.error;
    if (typeof err.status === "number") return `Request failed (${err.status})`;
  }
  return "An error occurred.";
};

const formatDate = (dateString: string | undefined): string => {
  if (!dateString) return "—";
  try {
    return new Date(dateString).toLocaleString();
  } catch {
    return dateString;
  }
};

export function WorkloadDetailsPage() {
  const navigate = useNavigate();
  const { workloadId } = useParams<{ workloadId: string }>();
  const session = SessionManager.getSession();

  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [showSecrets, setShowSecrets] = useState(false);

  const {
    data: workload,
    isLoading,
    error,
    refetch,
  } = useGetWorkloadQuery(
    { workload_id: workloadId || "" },
    {
      skip: !session?.token || !workloadId,
      refetchOnMountOrArgChange: true,
    }
  );

  const [deleteWorkload, { isLoading: isDeleting }] = useDeleteWorkloadMutation();

  const handleDelete = async () => {
    if (!workload) return;

    try {
      const id = workload.id || workload.workload_id;
      if (!id) {
        toast.error("Workload ID not found");
        return;
      }
      await deleteWorkload({ workload_id: id }).unwrap();
      toast.success("Workload deleted successfully");
      navigate("/clients/workloads");
    } catch (error) {
      toast.error(getErrorMessage(error));
    }
  };

  const handleEdit = () => {
    if (!workload) return;
    const id = workload.id || workload.workload_id;
    navigate(`/clients/workloads/edit/${id}`);
  };

  const handleExport = () => {
    if (!workload) return;
    const exportData = {
      id: workload.id || workload.workload_id,
      spiffe_id: workload.spiffe_id || workload.spiffeId,
      type: workload.type,
      selectors: workload.selectors,
      vault_role: workload.vault_role,
      status: workload.status,
      attestation_type: workload.attestation_type,
      created_at: workload.created_at,
      updated_at: workload.updated_at,
    };

    const blob = new Blob([JSON.stringify(exportData, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `workload-${exportData.id}.json`;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
    toast.success("Workload configuration exported");
  };

  if (isLoading) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
        <div className="container mx-auto max-w-7xl px-6 py-8">
          <div className="flex items-center justify-center h-64">
            <RefreshCw className="h-8 w-8 animate-spin text-foreground" />
          </div>
        </div>
      </div>
    );
  }

  if (error || !workload) {
    return (
      <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
        <div className="container mx-auto max-w-7xl px-6 py-8">
          <div className="flex flex-col items-center justify-center h-64 space-y-4">
            <AlertCircle className="h-12 w-12 text-red-500" />
            <div className="text-center">
              <h3 className="text-lg font-semibold">Workload Not Found</h3>
              <p className="text-sm text-foreground mt-1">
                {error ? getErrorMessage(error) : "The requested workload could not be found."}
              </p>
            </div>
            <Button onClick={() => navigate("/clients/workloads")} variant="outline">
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back to Workloads
            </Button>
          </div>
        </div>
      </div>
    );
  }

  const spiffeId = workload.spiffe_id || workload.spiffeId || "—";
  const workloadIdValue = workload.id || workload.workload_id || "—";
  const workloadType = workload.type || "unknown";
  const status = workload.status || "unknown";
  const attestationType = workload.attestation_type || "—";
  const vaultRole = workload.vault_role || "—";

  // Parse selectors
  const selectors = workload.selectors || {};
  const selectorsArray = typeof selectors === "object" && !Array.isArray(selectors)
    ? Object.entries(selectors).map(([key, value]) => ({ key, value: String(value || "") }))
    : [];

  const getStatusVariant = (status: string) => {
    const normalized = status.toLowerCase();
    if (["active", "attested", "issued"].includes(normalized)) return "default";
    if (["pending", "registering"].includes(normalized)) return "secondary";
    if (["expired", "revoked", "inactive"].includes(normalized)) return "destructive";
    return "outline";
  };

  const getStatusIcon = (status: string) => {
    const normalized = status.toLowerCase();
    if (["active", "attested", "issued"].includes(normalized)) return <CheckCircle2 className="h-4 w-4" />;
    if (["pending", "registering"].includes(normalized)) return <Clock className="h-4 w-4" />;
    if (["expired", "revoked", "inactive"].includes(normalized)) return <XCircle className="h-4 w-4" />;
    return <AlertCircle className="h-4 w-4" />;
  };

  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="container mx-auto max-w-7xl px-6 py-8 space-y-6">
        {/* Header */}
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
              <PageHeader
                title="Workload Details"
                description="View comprehensive information about this workload"
              />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => refetch()}
              className="gap-2"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={handleExport}
              className="gap-2"
            >
              <Download className="h-4 w-4" />
              Export
            </Button>
            <Button
              variant="default"
              size="sm"
              onClick={handleEdit}
              className="gap-2"
            >
              <Edit className="h-4 w-4" />
              Edit
            </Button>
            <Button
              variant="destructive"
              size="sm"
              onClick={() => setDeleteDialogOpen(true)}
              className="gap-2"
            >
              <Trash2 className="h-4 w-4" />
              Delete
            </Button>
          </div>
        </div>

        <div className="grid gap-6 md:grid-cols-2">
          {/* Basic Information */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                Basic Information
              </CardTitle>
              <CardDescription>Core workload identification details</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Workload ID</label>
                <div className="flex items-center gap-2">
                  <code className="text-sm font-mono bg-muted px-3 py-2 rounded flex-1">
                    {workloadIdValue}
                  </code>
                  <CopyButton text={workloadIdValue} label="Workload ID" />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">SPIFFE ID</label>
                <div className="flex items-center gap-2">
                  <code className="text-sm font-mono bg-muted px-3 py-2 rounded flex-1 break-all">
                    {spiffeId}
                  </code>
                  <CopyButton text={spiffeId} label="SPIFFE ID" />
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Type</label>
                <Badge variant="outline" className="capitalize">
                  {workloadType}
                </Badge>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Status</label>
                <div className="flex items-center gap-2">
                  <Badge variant={getStatusVariant(status)} className="gap-1.5 capitalize">
                    {getStatusIcon(status)}
                    {status}
                  </Badge>
                </div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Attestation Type</label>
                <div className="text-sm font-mono">{attestationType}</div>
              </div>

              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Vault Role</label>
                <div className="text-sm font-mono">{vaultRole}</div>
              </div>
            </CardContent>
          </Card>

          {/* Selectors */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Shield className="h-5 w-5" />
                Selectors
              </CardTitle>
              <CardDescription>Kubernetes selectors for workload attestation</CardDescription>
            </CardHeader>
            <CardContent>
              {selectorsArray.length > 0 ? (
                <div className="space-y-3">
                  {selectorsArray.map(({ key, value }) => (
                    <div key={key} className="space-y-1.5">
                      <label className="text-xs font-medium text-foreground uppercase">
                        {key.replace("k8s:", "")}
                      </label>
                      <div className="flex items-center gap-2">
                        <code className="text-sm font-mono bg-muted px-3 py-2 rounded flex-1">
                          {value || "—"}
                        </code>
                        {value && <CopyButton text={value} label={key} />}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="text-center py-8 text-sm text-foreground">
                  No selectors configured
                </div>
              )}
            </CardContent>
          </Card>

          {/* Timestamps */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Clock className="h-5 w-5" />
                Timestamps
              </CardTitle>
              <CardDescription>Creation and modification times</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Created At</label>
                <div className="text-sm">{formatDate(workload.created_at)}</div>
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium text-foreground">Updated At</label>
                <div className="text-sm">{formatDate(workload.updated_at)}</div>
              </div>
            </CardContent>
          </Card>

          {/* Health Status (Placeholder) */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Activity className="h-5 w-5" />
                Health Status
              </CardTitle>
              <CardDescription>Current attestation and health status</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <span className="text-sm text-foreground">Last Attestation</span>
                  <Badge variant="outline">Not Available</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-foreground">Attestation Success Rate</span>
                  <Badge variant="outline">—</Badge>
                </div>
                <div className="flex items-center justify-between">
                  <span className="text-sm text-foreground">Certificate Expiry</span>
                  <Badge variant="outline">—</Badge>
                </div>
                <p className="text-xs text-foreground mt-4">
                  Health monitoring features coming soon
                </p>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Raw Data (Debug View) */}
        <Card>
          <CardHeader>
            <div className="flex items-center justify-between">
              <div>
                <CardTitle className="flex items-center gap-2">
                  <Key className="h-5 w-5" />
                  Raw Configuration
                </CardTitle>
                <CardDescription>Complete workload data from API</CardDescription>
              </div>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setShowSecrets(!showSecrets)}
                className="gap-2"
              >
                {showSecrets ? (
                  <>
                    <EyeOff className="h-4 w-4" />
                    Hide
                  </>
                ) : (
                  <>
                    <Eye className="h-4 w-4" />
                    Show
                  </>
                )}
              </Button>
            </div>
          </CardHeader>
          {showSecrets && (
            <CardContent>
              <pre className="text-xs font-mono bg-muted p-4 rounded overflow-auto max-h-96">
                {JSON.stringify(workload, null, 2)}
              </pre>
            </CardContent>
          )}
        </Card>
      </div>

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onConfirm={handleDelete}
        workloadId={workloadIdValue}
        workloadName={spiffeId}
        isLoading={isDeleting}
      />
    </div>
  );
}
