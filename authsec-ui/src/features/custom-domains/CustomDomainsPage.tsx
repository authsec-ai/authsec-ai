import { useState, useMemo } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { CardContent } from "@/components/ui/card";
import { PageHeader } from "@/components/layout/PageHeader";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { TableCard } from "@/theme/components/cards";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import {
  Plus,
  RefreshCw,
  Search,
  AlertTriangle,
  Globe,
  Shield,
  CheckCircle,
  Sparkles,
} from "lucide-react";
import { useListDomainsQuery } from "@/app/api/domainApi";
import { SessionManager } from "@/utils/sessionManager";
import {
  AddDomainModal,
  DeleteDomainDialog,
  DomainCard,
  DomainEmptyState,
} from "./components";
import type { CustomDomain } from "@/app/api/domainApi";

export function CustomDomainsPage() {
  const sessionData = SessionManager.getSession();
  const tenantId = sessionData?.tenant_id;

  // State
  const [searchQuery, setSearchQuery] = useState("");
  const [addModalOpen, setAddModalOpen] = useState(false);
  const [deleteDialog, setDeleteDialog] = useState<{
    open: boolean;
    domain: CustomDomain | null;
  }>({ open: false, domain: null });

  // API data fetching
  const {
    data: domains = [],
    isLoading,
    isFetching,
    error,
    refetch,
  } = useListDomainsQuery({ tenant_id: tenantId || "" }, { skip: !tenantId });

  // Error handling
  const errorMessage = error
    ? (error as any)?.data?.message ||
      (error as any)?.data?.error ||
      "Failed to fetch domains"
    : null;

  const showInitialSkeleton =
    isLoading && domains.length === 0 && !errorMessage;

  // Filter domains by search query
  const filteredDomains = useMemo(() => {
    if (!searchQuery.trim()) return domains;
    const query = searchQuery.toLowerCase();
    return domains.filter((domain) =>
      domain.domain.toLowerCase().includes(query),
    );
  }, [domains, searchQuery]);

  // Handlers
  const handleAddDomain = () => setAddModalOpen(true);

  const handleDeleteClick = (domain: CustomDomain) => {
    setDeleteDialog({ open: true, domain });
  };

  const handleDeleteClose = () => {
    setDeleteDialog({ open: false, domain: null });
  };

  return (
    <div className="min-h-screen">
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title="Custom Domains"
          description="Manage custom domains for branded authentication experiences"
          actions={
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="icon"
                onClick={() => refetch()}
                disabled={isFetching}
              >
                <RefreshCw
                  className={`h-4 w-4 ${isFetching ? "animate-spin" : ""}`}
                />
              </Button>
              <Button onClick={handleAddDomain}>
                <Plus className="h-4 w-4 mr-2" />
                Add Domain
              </Button>
            </div>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="Custom Domain Setup"
          description="Add custom domains to your tenant for branded authentication experiences. 
            Verify domain ownership via DNS TXT records to ensure security."
          features={[
            {
              text: "DNS verification for security",
              icon: Shield,
            },
            {
              text: "Support for multiple domains",
              icon: CheckCircle,
            },
            {
              text: "Set a primary domain for default flows",
              icon: Sparkles,
            },
          ]}
          faqsTitle="Common questions:"
          faqs={[
            {
              id: "1",
              question: "How do I verify my domain?",
              answer:
                "After adding a domain, you'll receive DNS TXT record details. Add these records to your DNS provider (e.g., Cloudflare, Route53). DNS changes may take 5-15 minutes to propagate. Then click 'Verify DNS' to complete verification.",
            },
            {
              id: "2",
              question: "What is a primary domain?",
              answer:
                "The primary domain is used as the default for authentication flows. Only verified domains can be set as primary. You can have one primary domain at a time.",
            },
            {
              id: "3",
              question: "Can I use multiple custom domains?",
              answer:
                "Yes! You can add and verify multiple custom domains. Each domain will receive its own DNS verification records. Only one can be the primary domain at a time.",
            },
          ]}
          storageKey="custom-domains-page-banner"
          dismissible={true}
        />

        {/* Search Bar */}
        {domains.length > 0 && (
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
              <Input
                placeholder="Search domains..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>
        )}

        {/* Content */}
        <TableCard>
          <CardContent variant="flush" className="p-0">
            {errorMessage ? (
              // Error state
              <div className="flex flex-col items-center justify-center p-12 text-center">
                <AlertTriangle className="h-10 w-10 text-destructive mb-4" />
                <h3 className="text-lg font-semibold mb-2">
                  Unable to Load Domains
                </h3>
                <p className="text-muted-foreground mb-4">{errorMessage}</p>
                <Button variant="outline" onClick={() => refetch()}>
                  <RefreshCw className="h-4 w-4 mr-2" />
                  Try Again
                </Button>
              </div>
            ) : showInitialSkeleton ? (
              // Loading skeleton
              <div className="p-6">
                <DataTableSkeleton rows={4} columns={3} />
              </div>
            ) : domains.length === 0 ? (
              // Empty state
              <DomainEmptyState onAddDomain={handleAddDomain} />
            ) : filteredDomains.length === 0 ? (
              // No search results
              <div className="flex flex-col items-center justify-center p-12 text-center">
                <Search className="h-10 w-10 text-muted-foreground mb-4" />
                <h3 className="text-lg font-semibold mb-2">No Domains Found</h3>
                <p className="text-muted-foreground">
                  No domains match your search "{searchQuery}"
                </p>
              </div>
            ) : (
              // Domain grid
              <div className="relative">
                <div className="grid gap-4 p-6 md:grid-cols-2 lg:grid-cols-3">
                  {filteredDomains.map((domain) => (
                    <DomainCard
                      key={domain.id}
                      domain={domain}
                      onDelete={handleDeleteClick}
                    />
                  ))}
                </div>

                {/* Refresh overlay */}
                {isFetching && domains.length > 0 && (
                  <div className="absolute inset-0 bg-background/50 backdrop-blur-sm flex items-center justify-center">
                    <RefreshCw className="h-6 w-6 animate-spin text-muted-foreground" />
                    <span className="ml-2 text-sm text-muted-foreground">
                      Refreshing...
                    </span>
                  </div>
                )}
              </div>
            )}
          </CardContent>
        </TableCard>
      </div>

      {/* Modals */}
      <AddDomainModal open={addModalOpen} onOpenChange={setAddModalOpen} />

      <DeleteDomainDialog
        open={deleteDialog.open}
        onOpenChange={(open: boolean) =>
          setDeleteDialog((prev) => ({ ...prev, open }))
        }
        domain={deleteDialog.domain}
        onSuccess={handleDeleteClose}
      />
    </div>
  );
}
