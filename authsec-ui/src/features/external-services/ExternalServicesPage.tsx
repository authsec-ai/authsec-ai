import { useState, useMemo, useRef } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { PageHeader } from "@/components/layout/PageHeader";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { useGetExternalServicesQuery } from "@/app/api/externalServiceApi";
import { CloudCog, Plus, RefreshCw, ChevronsRight } from "lucide-react";
import {
  ExternalServiceFiltersCard,
  EnhancedExternalServicesTable,
} from "./components";
import { ExternalServicesStatisticsBar } from "./components/ExternalServicesStatisticsBar";
import { useScrollRestore } from "@/hooks/use-scroll-restore";
import { useResponsiveCards } from "@/hooks/use-mobile";
import { toast } from "@/lib/toast";
import { FloatingFAQ } from "@/features/clients/components/FloatingFAQ";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";

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
  code: CodeExample;
}

const EXTERNAL_SERVICES_FAQ_DATA: FAQItem[] = [
  {
    id: "1",
    question: "What's the process to integrate External Service?",
    description:
      "Learn how to create and manage secrets programmatically using the AuthSec SDK.",
    code: {
      python: [
        {
          label: "Step 1: Install AuthSec SDK",
          code: `pip install git+https://github.com/authsec-ai/sdk-authsec.git`,
        },
        {
          label: "Import Dependencies",
          code: `from authsec_sdk import protected_by_AuthSec, ServiceAccessSDK`,
        },
        {
          label: "Create Protected Function",
          code: `@protected_by_AuthSec("list_my_repos", scopes=["read"])
async def list_my_repos(arguments: dict, session) -> list:
    """List user's GitHub repositories."""

    # Create services SDK
    services_sdk = ServiceAccessSDK(session)

    # Fetch GitHub token from Vault (secure!)
    github_token = await services_sdk.get_service_token("your_token")

    # Call GitHub API
    async with aiohttp.ClientSession() as http:
        async with http.get(
            'https://api.github.com/user/repos',
            headers={'Authorization': f'Bearer {github_token}'}
        ) as response:
            repos = await response.json()

    # Format response
    repo_list = "\n".join([
        f"- {repo['full_name']} ({repo['stargazers_count']})"
        for repo in repos[:10]
    ])

    return [{
        "type": "text",
        "text": f"Your GitHub Repositories:\n{repo_list}"
    }]`,
        },
      ],
      typescript: [
        {
          label: "Usage",
          code: `const token = 'your-access-token';
const client = await createClient(token, 'My MCP Server', 'Production MCP server');`,
        },
      ],
    },
  },
];

export function ExternalServicesPage() {
  const navigate = useNavigate();
  const mainAreaRef = useRef<HTMLDivElement>(null);
  useScrollRestore(mainAreaRef, "externalServicesPage");
  const { mainAreaRef: responsiveRef } = useResponsiveCards();

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["external-services-intro"],
  });

  // API data fetching
  const {
    data: services = [],
    isLoading,
    isFetching,
    error,
  } = useGetExternalServicesQuery(undefined, {
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  // Local state
  const [selectedServices, setSelectedServices] = useState<string[]>([]);
  const [search, setSearch] = useState("");
  const [providerFilter, setProviderFilter] = useState("all");
  const [statusFilter, setStatusFilter] = useState("all");
  const [clientFilter, setClientFilter] = useState("all");
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false);

  // Filtering logic
  const filtered = useMemo(() => {
    return services.filter((s) => {
      if (search && !s.name.toLowerCase().includes(search.toLowerCase()))
        return false;
      if (
        providerFilter !== "all" &&
        s.type.toLowerCase() !== providerFilter.toLowerCase()
      )
        return false;
      if (statusFilter !== "all" && s.auth_type !== statusFilter) return false;
      return true;
    });
  }, [services, search, providerFilter, statusFilter]);

  // Event handlers
  const handleServiceSelect = (id: string) => {
    const newSelected = selectedServices.includes(id)
      ? selectedServices.filter((s) => s !== id)
      : [...selectedServices, id];
    setSelectedServices(newSelected);
  };

  const handleSelectAll = () => {
    setSelectedServices(
      selectedServices.length === filtered.length
        ? []
        : filtered.map((s) => s.id),
    );
  };

  const handleCreateService = () => {
    navigate("/external-services/add");
  };

  const handleResetFilters = () => {
    setSearch("");
    setProviderFilter("all");
    setStatusFilter("all");
    setClientFilter("all");
    setShowAdvancedFilters(false);
    setSelectedServices([]);
  };

  // Get error message if exists
  const errorMessage = error
    ? (error as { data?: { message?: string } })?.data?.message ||
      "Failed to fetch external services data"
    : null;

  const isTableLoading = isLoading || isFetching;
  const showInitialSkeleton =
    isLoading && filtered.length === 0 && !errorMessage;

  return (
    <div className="min-h-screen" ref={responsiveRef}>
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title="External Services And Secrets Management"
          description="Manage OAuth connections and integrations for third-party APIs"
          actions={
            <Button
              onClick={handleCreateService}
              data-tour-id="add-service-button"
            >
              <Plus className="h-4 w-4 mr-2" />
              Add Service
            </Button>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="External Service Integration"
          description="Securely connect and manage third-party services with OAuth 2.0 token storage and access control."
          featuresTitle="Key capabilities:"
          features={[
            { text: "Secure OAuth token storage in vault" },
            { text: "Service-to-service authentication" },
            { text: "Automated token refresh and management" },
          ]}
          primaryAction={{
            label: "Read docs",
            onClick: () =>
              window.open(
                "https://docs.authsec.dev/enterprise-features/category/external-services-5",
                "_blank",
              ),
            variant: "outline",
            className:
              "bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs",
            icon: ChevronsRight,
          }}
          storageKey="external-services-page"
          dismissible={true}
        />

        {/* Filters */}
        <div>
          <ExternalServiceFiltersCard
            searchTerm={search}
            onSearchTermChange={setSearch}
            providerFilter={providerFilter}
            onProviderFilterChange={setProviderFilter}
            statusFilter={statusFilter}
            onStatusFilterChange={setStatusFilter}
            showAdvancedFilters={showAdvancedFilters}
            onToggleAdvancedFilters={() =>
              setShowAdvancedFilters(!showAdvancedFilters)
            }
            clientFilter={clientFilter}
            onClientFilterChange={setClientFilter}
            clients={[]}
            onResetFilters={handleResetFilters}
          />
        </div>

        {/* Statistics Bar */}
        {!errorMessage && filtered.length > 0 && (
          <div>
            <ExternalServicesStatisticsBar
              totalServices={filtered.length}
              integrationsUsed={
                filtered.filter((s) => s.agent_accessible).length
              }
              usersWithAccess={0}
            />
          </div>
        )}

        {/* Table */}
        <div>
          {errorMessage ? (
            <Card className="border-0 bg-card">
              <CardContent className="p-8 text-center">
                <div className="flex flex-col items-center space-y-4">
                  <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                    <CloudCog className="h-8 w-8 text-red-600 dark:text-red-400" />
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-red-900 dark:text-red-100">
                      Unable to Load External services and secrets management
                    </h3>
                    <p className="text-red-700 dark:text-red-300 mt-1">
                      {errorMessage}
                    </p>
                  </div>
                </div>
              </CardContent>
            </Card>
          ) : showInitialSkeleton ? (
            <Card className="border-0 bg-card">
              <CardContent className="p-4">
                <DataTableSkeleton rows={8} />
              </CardContent>
            </Card>
          ) : (
            <div className="relative">
              <EnhancedExternalServicesTable
                data={filtered}
                selectedServices={selectedServices}
                onSelectAll={handleSelectAll}
                onSelectService={handleServiceSelect}
                onCreateService={handleCreateService}
              />
              {isTableLoading && filtered.length > 0 && (
                <div className="absolute inset-0 bg-background/50 backdrop-blur-sm flex items-center justify-center rounded-xl">
                  <div className="flex items-center space-x-2 text-sm font-medium text-foreground">
                    <RefreshCw className="h-4 w-4 animate-spin" />
                    <span>Refreshing services...</span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Floating FAQ */}
      <FloatingFAQ faqData={EXTERNAL_SERVICES_FAQ_DATA} />
    </div>
  );
}
