import React, { useState, useMemo, useCallback, useEffect } from "react";
import { CardContent } from "@/components/ui/card";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { Plus, RefreshCw, AlertTriangle } from "lucide-react";
import { useGetScopeMappingsQuery } from "@/app/api/scopesApi";
import { useGetEndUserScopeMappingsQuery } from "@/app/api/enduser/scopesApi";
import { resolveTenantId } from "@/utils/workspace";
import { useResponsiveCards } from "../../hooks/use-mobile";
import { EnhancedScopesTable } from "./components/EnhancedScopesTable";
import ScopesFilterCard from "./components/ScopesFilterCard";
import { CreateScopeModal } from "./components/CreateScopeModal";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";
import { SDKQuickHelp, SCOPES_SDK_HELP } from "@/features/sdk";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { Lock, Eye, Key } from "lucide-react";

export interface ScopesQueryParams {
  searchQuery?: string;
}

/**
 * Scopes page component - Manage access scopes/permissions
 *
 * Features:
 * - Modern design based on Groups/Roles pages
 * - Real-time data from AuthSec API
 * - Advanced filtering with search
 * - Beautiful metrics and statistics
 * - Enhanced table with actions
 */
export function ScopesPage() {
  const { isAdmin } = useRbacAudience();
  const location = useLocation();
  const standardNavigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [filters, setFilters] = useState<Partial<ScopesQueryParams>>({});
  const [createScopeModalOpen, setCreateScopeModalOpen] = useState(false);

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "Scopes",
            subtitle: "This Defines permissions boundary in a project",
          }
        : {
            title: "Scopes",
            subtitle: "This Defines permissions boundary in a project",
          },
    [isAdmin]
  );

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["scopes-management"],
  });

  // API data fetching
  const tenantId = resolveTenantId();

  // Conditionally use admin or end-user APIs based on audience
  const {
    data: adminScopeMappings = [],
    isLoading: adminScopesLoading,
    error: adminScopesError,
  } = useGetScopeMappingsQuery(undefined, {
    skip: !tenantId || !isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: endUserScopeMappings = [],
    isLoading: endUserScopesLoading,
    error: endUserScopesError,
  } = useGetEndUserScopeMappingsQuery(undefined, {
    skip: isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  // Select the correct data based on audience
  // Both admin and end-user now use mappings API for scope_name + resources
  const scopes = useMemo(() => {
    if (isAdmin) {
      return adminScopeMappings.map((mapping) => ({
        id: mapping.scope_name,
        name: mapping.scope_name,
        resources: mapping.resources || [],
      }));
    }
    return endUserScopeMappings.map((mapping: any) => ({
      id: mapping.scope_name,
      name: mapping.scope_name,
      resources: mapping.resources || [],
    }));
  }, [adminScopeMappings, endUserScopeMappings, isAdmin]);
  const scopesLoading = isAdmin ? adminScopesLoading : endUserScopesLoading;
  const scopesError = isAdmin ? adminScopesError : endUserScopesError;

  // Auto-open modal if query param present (from wizard)
  useEffect(() => {
    if (searchParams.get("openModal") === "create") {
      setCreateScopeModalOpen(true);
      // Clean up query param while preserving location state
      searchParams.delete("openModal");
      setSearchParams(searchParams, { replace: true, state: location.state });
    }
  }, [searchParams, setSearchParams, location.state]);

  // Handle modal close with wizard awareness
  const handleScopeModalSuccess = () => {
    // Don't close modal here - it will be closed by onOpenChange
    // If coming from wizard, navigate back to root with success flag
    if (location.state?.fromWizard) {
      standardNavigate("/", { state: { scopeCreated: true } });
    }
  };

  // Extract error message
  const errorMessage = scopesError
    ? (scopesError as any)?.data?.message || "Failed to fetch scopes data"
    : null;

  // Apply client-side filtering based on filter state
  const filteredScopes = useMemo(() => {
    let result = [...scopes];

    // Apply search filter
    if (filters.searchQuery) {
      const query = filters.searchQuery.toLowerCase();
      result = result.filter((scope) =>
        scope.name.toLowerCase().includes(query)
      );
    }

    return result;
  }, [scopes, filters]);

  const showInitialSkeleton =
    scopesLoading && filteredScopes.length === 0 && !errorMessage;

  // Memoize onFiltersChange to prevent infinite re-renders
  const handleFiltersChange = useCallback(
    (newFilters: Partial<ScopesQueryParams>) => {
      setFilters((prev) => {
        const prevJson = JSON.stringify(prev ?? {});
        const nextJson = JSON.stringify(newFilters ?? {});
        if (prevJson === nextJson) return prev;
        return newFilters;
      });
    },
    []
  );

  const handleCreateScope = () => setCreateScopeModalOpen(true);

  return (
    <div className="min-h-screen" ref={mainAreaRef}>
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title={audienceCopy.title}
          description={audienceCopy.subtitle}
          actions={
            <Button
              onClick={handleCreateScope}
              data-tour-id="create-scope-button"
            >
              <Plus className="mr-2 h-4 w-4" />
              Create Scope
            </Button>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="Understanding Scopes"
          description="Scopes define the boundaries of permissions within your system. They determine what level of access a role has, such as 'read', 'write', or 'delete' capabilities."
          features={[
            {
              text: "Define permission boundaries and access levels",
              icon: Lock,
            },
            {
              text: "Control what actions are allowed on resources",
              icon: Eye,
            },
            { text: "Create OAuth-compatible scope definitions", icon: Key },
          ]}
          featuresTitle="Key Features"
          faqs={[
            {
              id: "1",
              question: "What are scopes?",
              answer:
                "Scopes define the level of access granted by a permission. Common scopes include 'read' (view only), 'write' (create/edit), 'delete' (remove), and 'admin' (full control). They work with roles to create granular permissions.",
            },
            {
              id: "2",
              question: "How do scopes differ from permissions?",
              answer:
                "Scopes are the 'verbs' of your permission system. While permissions combine role + scope + resource, scopes specifically define the action level. For example, a 'read' scope on 'documents' means view-only access.",
            },
            {
              id: "3",
              question: "Can I create custom scopes?",
              answer:
                "Yes! While common scopes like 'read' and 'write' cover most cases, you can create custom scopes for specific business needs, such as 'approve', 'review', or 'publish'.",
            },
          ]}
          faqsTitle="Common Questions"
          storageKey="scopes-page-banner"
          dismissible={true}
        />

        {/* Filter/Search Card */}
        <div data-tour-id="scopes-filters">
          <ScopesFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
            scopesData={scopes}
          />
        </div>

        {/* Table */}
        <div className="scopes-table-container" data-tour-id="scopes-table">
          <TableCard className="transition-all duration-500">
            <CardContent variant="flush">
              {errorMessage ? (
                <div className="flex flex-col items-center justify-center p-12 space-y-4">
                  <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                    <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                  </div>
                  <div className="text-center space-y-1">
                    <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                      Unable to Load Scopes
                    </h3>
                    <p className="text-red-700 dark:text-red-300">
                      {errorMessage}
                    </p>
                  </div>
                </div>
              ) : showInitialSkeleton ? (
                <DataTableSkeleton rows={8} />
              ) : (
                <div className="relative">
                  <EnhancedScopesTable
                    scopes={filteredScopes}
                    isAdmin={isAdmin}
                    tenantId={tenantId || ""}
                  />
                  {scopesLoading && filteredScopes.length > 0 && (
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

      {/* Create Scope Modal */}
      <CreateScopeModal
        open={createScopeModalOpen}
        onOpenChange={setCreateScopeModalOpen}
        onSuccess={handleScopeModalSuccess}
      />

      {/* SDK Quick Help */}
      <SDKQuickHelp helpItems={SCOPES_SDK_HELP} title="Scopes SDK" />
    </div>
  );
}
