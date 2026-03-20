import React, { useState } from "react";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { PageHeader } from "@/components/layout/PageHeader";
import { Plus, Search } from "lucide-react";
import { RoleScopeMappingsView } from "./components/RoleScopeMappingsView";
import { useResponsiveCards } from "../../hooks/use-mobile";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import { Button } from "@/components/ui/button";
import { CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import { SDKQuickHelp, ROLE_BINDINGS_SDK_HELP } from "@/features/sdk";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";

/**
 * Mappings page component - Visualize RBAC role bindings
 *
 * Features:
 * - Role bindings with user/role details
 * - Search and filter capabilities
 * - Optional filters: user_id, role_id, scope_type
 */
export function MappingsPage() {
  const { isAdmin } = useRbacAudience();
  const [roleScopeSearch, setRoleScopeSearch] = useState("");
  const [mapModalOpen, setMapModalOpen] = useState(false);

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY['mappings-management'],
  });

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  return (
    <div className="min-h-screen" ref={mainAreaRef}>
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title={isAdmin ? "Role Bindings" : "Role Bindings"}
          description={
            isAdmin
              ? "View and manage all role bindings with user and role details"
              : "View role bindings assigned to users"
          }
          actions={
            <Button onClick={() => setMapModalOpen(true)} data-tour-id="create-mapping-button">
              <Plus className="mr-2 h-4 w-4" />
              Create binding
            </Button>
          }
        />

        {/* Filter Card */}
        <div data-tour-id="mappings-filters">
          <FilterShell>
            <CardContent variant="compact">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
                <div className="flex shrink-0 items-center gap-2">
                  <span className="text-sm font-medium text-foreground">Filters</span>
                  {roleScopeSearch.trim() && (
                    <span className="text-xs text-foreground bg-black/5 dark:bg-white/10 px-1.5 py-0.5 rounded">
                      1
                    </span>
                  )}
                </div>
                <div className="flex w-full flex-1 flex-wrap items-center gap-2">
                  <div className="flex-1 min-w-[200px] relative">
                    <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground/50" />
                    <Input
                      placeholder="Search users, roles, or scopes..."
                      value={roleScopeSearch}
                      onChange={(e) => setRoleScopeSearch(e.target.value)}
                      className="pl-9 h-9 text-sm"
                    />
                  </div>
                  {roleScopeSearch.trim() && (
                    <Button
                      variant="ghost"
                      size="sm"
                      className="h-9 text-sm text-foreground hover:text-foreground"
                      onClick={() => setRoleScopeSearch("")}
                    >
                      Clear
                    </Button>
                  )}
                </div>
              </div>
            </CardContent>
          </FilterShell>
        </div>

        {/* Role Bindings View */}
        <div data-tour-id="mappings-table">
          <RoleScopeMappingsView
            searchQuery={roleScopeSearch}
            isMapModalOpen={mapModalOpen}
            onMapModalOpenChange={setMapModalOpen}
          />
        </div>
      </div>

      {/* SDK Quick Help */}
      <SDKQuickHelp
        entityType="Role Bindings"
        helpItems={ROLE_BINDINGS_SDK_HELP}
      />
    </div>
  );
}
