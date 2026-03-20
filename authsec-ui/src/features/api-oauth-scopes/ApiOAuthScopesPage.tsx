import React, { useState, useMemo, useCallback } from "react";
import { CardContent } from "@/components/ui/card";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { Plus, RefreshCw, AlertTriangle } from "lucide-react";
import { useGetAdminApiOAuthScopesQuery } from "@/app/api/admin/apiOAuthScopesApi";
import { useGetEndUserApiOAuthScopesQuery } from "@/app/api/enduser/apiOAuthScopesApi";
import { useResponsiveCards } from "@/hooks/use-mobile";
import { EnhancedApiOAuthScopesTable } from "./components/EnhancedApiOAuthScopesTable";
import ApiOAuthScopesFilterCard from "./components/ApiOAuthScopesFilterCard";
import { CreateApiOAuthScopeMappingModal } from "./components/CreateApiOAuthScopeMappingModal";
import { EditApiOAuthScopeModal } from "./components/EditApiOAuthScopeModal";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { TableCard } from "@/theme/components/cards";
import { PageHeader } from "@/components/layout/PageHeader";
import { Button } from "@/components/ui/button";
import type { ApiOAuthScopesQueryParams, ApiOAuthScope } from "./types";
import { SDKQuickHelp, OAUTH_API_SDK_HELP } from "@/features/sdk";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { Key, Lock, Zap } from "lucide-react";

/**
 * API/OAuth Scopes page component - Manage OAuth scope mappings
 *
 * Features:
 * - Context-aware for admin/enduser
 * - Real-time data from AuthSec API
 * - Advanced filtering with search
 * - Create/Edit/Delete scope mappings
 * - Table with: id, name, description, permissions_linked, created_at columns
 */
export function ApiOAuthScopesPage() {
  const { isAdmin } = useRbacAudience();
  const [filters, setFilters] = useState<Partial<ApiOAuthScopesQueryParams>>(
    {},
  );
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editingScope, setEditingScope] = useState<ApiOAuthScope | null>(null);

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["api-oauth-scopes-intro"],
  });

  const audienceCopy = useMemo(
    () =>
      isAdmin
        ? {
            title: "API/OAuth Scopes",
            subtitle: "Manage OAuth scope mappings for API authorization",
          }
        : {
            title: "API/OAuth Scopes",
            subtitle: "Configure OAuth scopes for your applications",
          },
    [isAdmin],
  );

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  // Conditionally use admin or end-user APIs based on audience
  const {
    data: adminScopes = [],
    isLoading: adminScopesLoading,
    error: adminScopesError,
  } = useGetAdminApiOAuthScopesQuery(undefined, {
    skip: !isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  const {
    data: endUserScopes = [],
    isLoading: endUserScopesLoading,
    error: endUserScopesError,
  } = useGetEndUserApiOAuthScopesQuery(undefined, {
    skip: isAdmin,
    refetchOnMountOrArgChange: true,
    refetchOnFocus: true,
    refetchOnReconnect: true,
  });

  // Select the correct data based on audience
  const scopes = isAdmin ? adminScopes : endUserScopes;
  const scopesLoading = isAdmin ? adminScopesLoading : endUserScopesLoading;
  const scopesError = isAdmin ? adminScopesError : endUserScopesError;

  // Extract error message
  const errorMessage = scopesError
    ? (scopesError as any)?.data?.message ||
      "Failed to fetch API/OAuth scopes data"
    : null;

  // Apply client-side filtering based on filter state
  const filteredScopes = useMemo(() => {
    // Ensure scopes is an array
    const scopesArray = Array.isArray(scopes) ? scopes : [];
    let result = [...scopesArray];

    // Apply search filter
    if (filters.searchQuery) {
      const query = filters.searchQuery.toLowerCase();
      result = result.filter(
        (scope) =>
          scope.id.toLowerCase().includes(query) ||
          scope.name.toLowerCase().includes(query) ||
          scope.description?.toLowerCase().includes(query),
      );
    }

    return result;
  }, [scopes, filters]);

  const showInitialSkeleton =
    scopesLoading && filteredScopes.length === 0 && !errorMessage;

  // Memoize onFiltersChange to prevent infinite re-renders
  const handleFiltersChange = useCallback(
    (newFilters: Partial<ApiOAuthScopesQueryParams>) => {
      setFilters((prev) => {
        const prevJson = JSON.stringify(prev ?? {});
        const nextJson = JSON.stringify(newFilters ?? {});
        if (prevJson === nextJson) return prev;
        return newFilters;
      });
    },
    [],
  );

  const handleCreateMapping = () => setCreateModalOpen(true);

  const handleEdit = useCallback((scope: ApiOAuthScope) => {
    setEditingScope(scope);
  }, []);

  return (
    <div className="min-h-screen" ref={mainAreaRef}>
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title={audienceCopy.title}
          description={audienceCopy.subtitle}
          actions={
            <Button
              onClick={handleCreateMapping}
              data-tour-id="create-api-scope-button"
            >
              <Plus className="mr-2 h-4 w-4" />
              Add mapping
            </Button>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="Understanding API/OAuth Scopes"
          description="API/OAuth scopes define what resources and actions third-party applications can access on behalf of users. They're essential for secure API authorization and OAuth flows."
          features={[
            {
              text: "Define OAuth scope mappings for secure authorization",
              icon: Key,
            },
            {
              text: "Control third-party application access levels",
              icon: Lock,
            },
            { text: "Enable standard OAuth 2.1 flows", icon: Zap },
          ]}
          featuresTitle="Key Features"
          faqs={[
            {
              id: "1",
              question: "What are API/OAuth scopes?",
              answer:
                "API/OAuth scopes are standardized permission strings (like 'read:users' or 'write:documents') that external applications request when users authorize them. They limit what the application can do on the user's behalf.",
            },
            {
              id: "2",
              question: "How are OAuth scopes different from regular scopes?",
              answer:
                "OAuth scopes are specifically designed for third-party API access and follow OAuth 2.1 standards. They're presented to users during authorization ('This app wants to...') and control what external applications can access.",
            },
            {
              id: "3",
              question: "How do I create an OAuth scope mapping?",
              answer:
                "Click 'Add mapping' to define a new OAuth scope. Specify the scope identifier (e.g., 'read:profile'), description (shown to users), and map it to your internal permissions. This allows external apps to request that specific access level.",
            },
          ]}
          faqsTitle="Common Questions"
          storageKey="api-oauth-scopes-page-banner"
          dismissible={true}
        />

        {/* Filter/Search Card */}
        <div data-tour-id="api-scopes-filters">
          <ApiOAuthScopesFilterCard
            onFiltersChange={handleFiltersChange}
            initialFilters={filters}
          />
        </div>

        {/* Table */}
        <div
          className="api-oauth-scopes-table-container"
          data-tour-id="api-scopes-table"
        >
          <style>{`
            .api-oauth-scopes-table-container [data-slot="table-container"] {
              border: none !important;
              background: transparent !important;
            }
            .api-oauth-scopes-table-container [data-slot="table-header"] {
              background: transparent !important;
            }
            .api-oauth-scopes-table-container .bg-muted\\/50,
            .api-oauth-scopes-table-container .bg-muted\\/30,
            .api-oauth-scopes-table-container [class*="bg-muted"] {
              background: transparent !important;
            }
            .api-oauth-scopes-table-container .shadow-xl {
              box-shadow: none !important;
            }
          `}</style>
          <TableCard className="transition-all duration-500">
            <CardContent variant="flush">
              {errorMessage ? (
                <div className="flex flex-col items-center justify-center p-12 space-y-4">
                  <div className="p-4 bg-red-50 dark:bg-red-950/20 rounded-full">
                    <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
                  </div>
                  <div className="text-center space-y-1">
                    <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                      Unable to Load API/OAuth Scopes
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
                  <EnhancedApiOAuthScopesTable
                    scopes={filteredScopes}
                    isAdmin={isAdmin}
                    onEdit={handleEdit}
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

      {/* Create Mapping Modal */}
      <CreateApiOAuthScopeMappingModal
        open={createModalOpen}
        onOpenChange={setCreateModalOpen}
      />

      {/* Edit Mapping Modal */}
      {editingScope && (
        <EditApiOAuthScopeModal
          scope={editingScope}
          open={!!editingScope}
          onOpenChange={(open) => !open && setEditingScope(null)}
        />
      )}

      {/* SDK Quick Help */}
      <SDKQuickHelp title="OAuth Scopes SDK" helpItems={OAUTH_API_SDK_HELP} />
    </div>
  );
}
