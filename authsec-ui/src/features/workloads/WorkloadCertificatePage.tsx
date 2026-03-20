import { useState, useMemo } from "react";
import { Button } from "../../components/ui/button";
import { Card, CardContent } from "../../components/ui/card";
import { PageHeader } from "@/components/layout/PageHeader";
import { TableCard } from "@/theme/components/cards";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import {
  Search,
  RefreshCw,
  Eye,
  CheckCircle,
  XCircle,
  Clock,
  Plus,
  ServerCog,
  ChevronRight,
  Server,
  Fingerprint,
  Tags,
  ChevronsRight,
} from "lucide-react";
import { toast } from "react-hot-toast";
import { SessionManager } from "../../utils/sessionManager";
import {
  useListEntriesQuery,
  useDeleteEntryMutation,
  useListAgentsQuery,
  type EntryRecord,
} from "../../app/api/workloadsApi";
import { useNavigate } from "react-router-dom";
import { WorkloadsTable } from "./components/WorkloadsTable";
import { WorkloadsFilterCard } from "./components/WorkloadsFilterCard";
import { DeleteConfirmDialog } from "./components/DeleteConfirmDialog";
import type {
  DisplayWorkload,
  WorkloadsTableActions,
} from "./utils/workloads-table-utils";
import { createEntriesTableColumns } from "./utils/workloads-table-utils";
import { FloatingFAQ } from "@/features/clients/components/FloatingFAQ";
import { WorkloadRegistrationGuide } from "./components/WorkloadRegistrationGuide";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import type { PageInfoBannerSection } from "@/components/shared/PageInfoBanner";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import type {
  FloatingHelpItem,
  FloatingHelpLanguageTab,
} from "@/components/shared/FloatingHelp";
import { SPIRE_FAQ_DATA as AGENT_DEPLOYMENT_DATA } from "@/features/wizards/components/spire-faq-data";

interface CodeExample {
  python: Array<{
    label: string;
    code: string;
  }>;
  typescript: Array<{
    label: string;
    code: string;
  }>;
}

interface FAQItem {
  id: string;
  question: string;
  description: string;
  code?: CodeExample;
  customContent?: React.ReactNode;
}

const SPRIRE_FAQ_DATA: FloatingHelpItem[] = [
  {
    id: "1",
    question: "Configure workload identity using a SDK.",
    description: "Learn how to Configure workload identity using a SDK",
    code: {
      python: [
        {
          label: "Step 1: Install AuthSec SDK",
          code: `pip install git+https://github.com/authsec-ai/sdk-authsec.git`,
        },
        {
          label: "Import Dependencies",
          code: `from authsec_sdk import QuickStartSVID`,
        },
        {
          label: "Example Usage",
          code: `from AuthSec_SDK import (
    mcp_tool, # unprotected tool decorator
    protected_by_AuthSec, # protected tool decorator
    run_mcp_server_with_oauth, # function to run MCP server with OAuth
    QuickStartSVID  # SPIRE workload identity
)
 
@mcp_tool(
    "get_spire_identity",
    description="Get current SPIRE workload identity (SPIFFE ID and certificate paths)",
    inputSchema={"type": "object", "properties": {}}
)
async def get_spire_identity(arguments: dict) -> list:
    """Get SPIRE workload identity information"""
    try:
        svid = await QuickStartSVID.initialize(socket_path="your/agent/path.sock")
        result = {
            "status": "success",
            "spiffe_id": svid.spiffe_id,
            "certificate": str(svid.cert_file_path),
            "private_key": str(svid.key_file_path),
            "ca_bundle": str(svid.ca_file_path),
            "auto_renewal": "enabled (30 min)"
        }
        return [{"type": "text", "text": json.dumps(result, indent=2)}]
    except RuntimeError as e:
        # SPIRE not enabled
        return [{"type": "text", "text": json.dumps({
            "status": "disabled",
            "message": str(e),
            "note": "To enable SPIRE, add 'spire_socket_path' parameter to run_mcp_server_with_oauth()"
        }, indent=2)}]
    except Exception as e:
        # SPIRE enabled but error occurred
        return [{"type": "text", "text": json.dumps({
            "status": "error",
            "error": str(e),
            "note": "SPIRE is enabled but agent connection failed"
        }, indent=2)}]`,
        },
        {
          label: "Main Server Entry Point",
          code: `if __name__ == "__main__":
    import sys
 
    run_mcp_server_with_oauth(
        user_module=sys.modules[__name__],
        client_id="your_client_id",
        app_name="Secure MCP Server with AuthSec",
        host="0.0.0.0",
        port=3008,
    )`,
        },
      ],
      typescript: [
        {
          label: "Define Types",
          code: `import axios from 'axios';

interface CreateClientPayload {
  name: string;
  description: string;
  type: 'mcp_server' | 'app' | 'api';
  active: boolean;
}

interface ClientResponse {
  client_id: string;
  client_secret: string;
  name: string;
  description: string;
}`,
        },
        {
          label: "Create Client Function",
          code: `async function createClient(
  token: string,
  clientName: string,
  description: string
): Promise<ClientResponse | null> {
  const url = 'https://api.authsec.dev/api/v1/clients';

  const headers = {
    'Authorization': \`Bearer \${token}\`,
    'Content-Type': 'application/json'
  };

  const payload: CreateClientPayload = {
    name: clientName,
    description: description,
    type: 'mcp_server',
    active: true
  };

  try {
    const response = await axios.post<ClientResponse>(url, payload, { headers });
    const client = response.data;

    console.log('Client created successfully!');
    console.log(\`Client ID: \${client.client_id}\`);
    console.log(\`Client Secret: \${client.client_secret}\`);

    return client;
  } catch (error) {
    console.error('Failed to create client:', error);
    return null;
  }
}`,
        },
        {
          label: "Usage",
          code: `const token = 'your-access-token';
const client = await createClient(token, 'My MCP Server', 'Production MCP server');`,
        },
      ],
    },
  },
  {
    id: "2",
    question: "Install and configure Spire Agent.",
    description:
      "Learn how to deploy SPIRE agents on Kubernetes, Docker, and VM environments",
    docsLink:
      "https://docs.authsec.dev/sdk/workloads/autonomous-workloads#install-and-configure-spire-agent",
    languageTabs: [
      { key: "kubernetes", label: "Kubernetes" },
      { key: "docker", label: "Docker" },
      { key: "vm", label: "VM" },
    ],
    code: {
      kubernetes: AGENT_DEPLOYMENT_DATA[0].code?.python || [],
      docker: AGENT_DEPLOYMENT_DATA[1].code?.python || [],
      vm: AGENT_DEPLOYMENT_DATA[2].code?.python || [],
    },
  },
  {
    id: "3",
    question: "FAQs relating to M2M Authentication",
    description:
      "Learn about the authentication crisis in modern infrastructure and how M2M Authentication solves it",
    customContent: (
      <div className="flex flex-col items-center justify-center py-12 px-6">
        <div className="max-w-2xl w-full space-y-6 text-center">
          <div className="space-y-3">
            <h3 className="text-xl font-semibold">
              Understanding M2M Authentication
            </h3>
            <p className="text-base leading-relaxed">
              Dive deep into the authentication crisis in modern infrastructure
              and discover how workload identity provides a secure, scalable
              solution for machine-to-machine communication.
            </p>
          </div>

          <a
            href="https://docs.authsec.dev/autonomous-agents/understanding-m2m/"
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 text-lg text-primary hover:underline transition-all"
          >
            Read Full Documentation
            <svg
              className="h-5 w-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14"
              />
            </svg>
          </a>
        </div>
      </div>
    ),
  },
];

type WorkloadStats = {
  active: number;
  expiring: number;
  revoked: number;
  total: number;
  agents: number; // Add agents count
};

type WorkloadsFilterState = {
  searchQuery?: string;
  type?: string;
  status?: string;
};

const normalizeSelectors = (selectors: EntryRecord["selectors"]): string[] => {
  // Handle K8sSelectors object format (new API format)
  if (selectors && typeof selectors === "object" && !Array.isArray(selectors)) {
    return Object.entries(selectors)
      .filter(([, val]) => typeof val === "string" || typeof val === "number")
      .map(([key, val]) => `${key}:${val}`)
      .filter((item) => item.trim().length > 0);
  }

  // Handle array format (legacy format)
  if (!Array.isArray(selectors)) return [];
  return selectors
    .map((selector) => {
      if (typeof selector === "string") {
        return selector;
      }

      if (selector && typeof selector === "object") {
        const typedSelector = selector as Record<string, unknown>;
        const type = typedSelector.type as string | undefined;
        const value = typedSelector.value as string | undefined;
        const match = typedSelector.match as string | undefined;

        if (type && value) {
          return `${type}:${value}`;
        }

        if (type && match) {
          return `${type}:${match}`;
        }

        const parts = Object.entries(typedSelector)
          .filter(
            ([, val]) => typeof val === "string" || typeof val === "number",
          )
          .map(([key, val]) => `${key}:${val}`);

        if (parts.length > 0) {
          return parts.join("|");
        }

        return JSON.stringify(selector);
      }

      return "";
    })
    .filter((item) => typeof item === "string" && item.trim().length > 0);
};

const normalizeStatus = (status: unknown): string => {
  if (typeof status === "string" && status.trim().length > 0) {
    return status.toLowerCase();
  }
  return "unknown";
};

const parseDateValue = (value: unknown): Date | null => {
  if (value === null || value === undefined) {
    return null;
  }

  if (typeof value === "number") {
    const millis = value > 1e12 ? value : value * 1000;
    const date = new Date(millis);
    return Number.isNaN(date.getTime()) ? null : date;
  }

  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) return null;

    const numeric = Number(trimmed);
    if (!Number.isNaN(numeric)) {
      return parseDateValue(numeric);
    }

    const date = new Date(trimmed);
    return Number.isNaN(date.getTime()) ? null : date;
  }

  return null;
};

const formatDate = (value: unknown): string => {
  const date = parseDateValue(value);
  if (!date) {
    return typeof value === "string" && value.trim().length > 0 ? value : "—";
  }
  return date.toLocaleString();
};

const mapEntryRecord = (entry: EntryRecord, index: number): DisplayWorkload => {
  // Convert selectors object to array format for display
  const selectors = Object.entries(entry.selectors).map(
    ([key, value]) => `${key}:${value}`,
  );

  return {
    id: entry.id,
    spiffeId: entry.spiffe_id,
    parentId: entry.parent_id,
    selectors,
    ttl: entry.ttl,
    admin: entry.admin,
    downstream: entry.downstream,
    createdAt: formatDate(entry.created_at),
    status: "active", // Entries are typically active
    type: "entry", // Default type for entries
    raw: entry,
  };
};

export function WorkloadCertificatePage() {
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id || "";
  const navigate = useNavigate();

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["workload-certificates-intro"],
  });

  const [filters, setFilters] = useState<WorkloadsFilterState>({});
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [workloadToDelete, setWorkloadToDelete] =
    useState<DisplayWorkload | null>(null);
  const [selectedWorkloadIds, setSelectedWorkloadIds] = useState<string[]>([]);

  const {
    data: entriesData,
    isLoading: entriesLoading,
    isFetching: entriesFetching,
    error: entriesError,
    refetch,
  } = useListEntriesQuery(
    { tenant_id: tenantId },
    {
      skip: !sessionData?.token,
      refetchOnMountOrArgChange: true,
      refetchOnFocus: true,
      refetchOnReconnect: true,
    },
  );

  // Fetch agents data for the count
  const { data: agentsData, isLoading: agentsLoading } = useListAgentsQuery(
    undefined,
    {
      skip: !sessionData?.token,
    },
  );

  const [deleteEntry, { isLoading: isDeleting }] = useDeleteEntryMutation();

  const entries = useMemo(
    () => (entriesData ? entriesData.map(mapEntryRecord) : []),
    [entriesData],
  );

  // Calculate stats
  const stats: WorkloadStats = {
    active: entries.length, // All entries are considered active
    revoked: 0, // Entries don't have revoked status
    expiring: 0, // Entries don't expire like certificates
    total: entries.length,
    agents: agentsData?.length || 0, // Use actual agent count from API
  };

  const filteredEntries = useMemo(() => {
    if (!entries.length) return [];

    const searchQuery = filters.searchQuery?.trim().toLowerCase();
    const typeFilter = filters.type?.toLowerCase();
    const statusFilter = filters.status?.toLowerCase();

    return entries.filter((entry) => {
      const matchesSearch = searchQuery
        ? entry.spiffeId.toLowerCase().includes(searchQuery) ||
          entry.id.toLowerCase().includes(searchQuery) ||
          entry.parentId?.toLowerCase().includes(searchQuery) ||
          entry.selectors.some((selector) =>
            selector.toLowerCase().includes(searchQuery),
          )
        : true;

      const matchesType =
        typeFilter && typeFilter !== "all"
          ? entry.type.toLowerCase() === typeFilter
          : true;

      const matchesStatus =
        statusFilter && statusFilter !== "all"
          ? entry.status.toLowerCase() === statusFilter
          : true;

      return matchesSearch && matchesType && matchesStatus;
    });
  }, [entries, filters]);

  const handleFiltersChange = (newFilters: WorkloadsFilterState) => {
    setFilters(newFilters);
  };

  const handleEditClick = (workload: DisplayWorkload) => {
    navigate(`/clients/workloads/edit/${workload.id}`);
  };

  const handleDeleteClick = (workload: DisplayWorkload) => {
    setWorkloadToDelete(workload);
    setDeleteDialogOpen(true);
  };

  const handleDeleteConfirm = async () => {
    if (!workloadToDelete) return;

    try {
      await deleteEntry({
        entry_id: workloadToDelete.id,
        tenant_id: tenantId,
      }).unwrap();
      toast.success("Entry deleted successfully");
      setDeleteDialogOpen(false);
      setWorkloadToDelete(null);
    } catch (error) {
      toast.error("Failed to delete entry");
    }
  };

  const actions: WorkloadsTableActions = {
    onEdit: handleEditClick,
    onDelete: handleDeleteClick,
  };

  const showInitialSkeleton = entriesLoading && !entriesData;
  const isTableLoading = entriesFetching && entriesData;

  return (
    <div className="min-h-screen">
      <div className="space-y-6 p-6 max-w-[1600px] mx-auto">
        <PageHeader
          title="Autonomous Workload"
          description="Monitor registered workload entries and their configurations"
          actions={
            <Button
              onClick={() => navigate("/clients/workloads/create")}
              data-tour-id="register-workload-button"
            >
              <Plus className="mr-2 h-4 w-4" />
              Register Workload
            </Button>
          }
        />

        {/* Informative Banner */}
        <PageInfoBanner
          title="Zero-Trust Workload Identity"
          description="Authenticate autonomous workloads using SPIFFE/SPIRE with X.509-SVID certificates. Enable secure service-to-service communication without shared secrets."
          featuresTitle="Use cases:"
          features={[
            { text: "Microservices authentication" },
            { text: "Service-to-service communication" },
            { text: "Automated certificate rotation" },
          ]}
          primaryAction={{
            label: "Read docs",
            onClick: () =>
              window.open(
                "https://docs.authsec.dev/autonomous-agents/configure-workload",
                "_blank",
              ),
            variant: "outline",
            className:
              "bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs",
            icon: ChevronsRight,
          }}
          faqsTitle="Common questions:"
          faqs={[
            {
              id: "how-register",
              question: "What is a Workload?",
              answer:
                "A workload represents a running process or service that needs a cryptographic identity. In SPIRE, each workload receives a unique X.509-SVID certificate for secure authentication.",
            },
            {
              id: "what-is-svid",
              question: "What is an X.509-SVID?",
              answer:
                "An X.509-SVID (SPIFFE Verifiable Identity Document) is a cryptographic certificate that proves a workload's identity. It contains a SPIFFE ID and is automatically rotated by SPIRE to maintain zero-trust security.",
            },

            {
              id: "what-selectors",
              question: "What are selectors?",
              answer:
                "Selectors are attestation criteria that identify workloads (e.g., k8s:pod-name, unix:uid, docker:image-id). The SPIRE agent uses them to verify workload identity before issuing certificates.",
            },
          ]}
          storageKey="workload-page"
          dismissible={true}
        />

        {/* Filter Card */}
        <div data-tour-id="certificates-filters">
          <WorkloadsFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
          />
        </div>

        {/* Table */}
        <div
          className="workloads-table-container"
          data-tour-id="certificates-table"
        >
          <style>{`
            .workloads-table-container [data-slot="table-container"] {
              border: none !important;
              background: transparent !important;
            }
            .workloads-table-container [data-slot="table-header"] {
              background: transparent !important;
            }
            .workloads-table-container .bg-muted\\/50,
            .workloads-table-container .bg-muted\\/30,
            .workloads-table-container [class*="bg-muted"] {
              background: transparent !important;
            }
            .workloads-table-container .hover\\:bg-muted\\/50:hover,
            .workloads-table-container .hover\\:bg-muted\\/30:hover {
              background: rgba(148, 163, 184, 0.1) !important;
            }
            .workloads-table-container .border,
            .workloads-table-container .border-b {
              border-color: rgba(148, 163, 184, 0.2) !important;
            }
            .workloads-table-container .shadow-xl {
              box-shadow: none !important;
            }
          `}</style>
          <TableCard className="transition-all duration-500">
            <CardContent variant="flush">
              {errorMessage ? (
                <div className="p-8 text-center">
                  <div className="flex flex-col items-center space-y-4">
                    <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                      <ServerCog className="h-8 w-8 text-red-600 dark:text-red-400" />
                    </div>
                    <div>
                      <h3 className="text-lg font-semibold text-red-900 dark:text-red-100">
                        Unable to Load Workloads
                      </h3>
                      <p className="text-red-700 dark:text-red-300 mt-1">
                        {errorMessage}
                      </p>
                    </div>
                  </div>
                </div>
              ) : showInitialSkeleton ? (
                <DataTableSkeleton rows={10} />
              ) : (
                <div className="relative">
                  <WorkloadsTable
                    workloads={filteredEntries}
                    selectedWorkloadIds={selectedWorkloadIds}
                    onSelectionChange={setSelectedWorkloadIds}
                    onSelectAll={() => {
                      if (
                        selectedWorkloadIds.length === filteredEntries.length
                      ) {
                        setSelectedWorkloadIds([]);
                      } else {
                        setSelectedWorkloadIds(
                          filteredEntries.map((w) => w.id),
                        );
                      }
                    }}
                    actions={actions}
                    useEntriesColumns={true}
                  />
                  {isTableLoading && filteredEntries.length > 0 && (
                    <div className="absolute inset-0 bg-white/50 dark:bg-neutral-900/50 backdrop-blur-sm flex items-center justify-center">
                      <div className="flex items-center space-x-2">
                        <RefreshCw className="h-4 w-4 animate-spin" />
                        <span className="text-sm font-medium">
                          Refreshing...
                        </span>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </TableCard>
        </div>
      </div>

      <FloatingFAQ faqData={SPRIRE_FAQ_DATA} />

      {/* Delete Confirmation Dialog */}
      <DeleteConfirmDialog
        open={deleteDialogOpen}
        onOpenChange={setDeleteDialogOpen}
        onConfirm={handleDeleteConfirm}
        workloadId={workloadToDelete?.id || ""}
        workloadName={workloadToDelete?.spiffeId}
        isLoading={isDeleting}
      />
    </div>
  );
}
