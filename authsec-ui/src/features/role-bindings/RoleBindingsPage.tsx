import React, { useState, useMemo, useEffect } from "react";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { PageHeader } from "@/components/layout/PageHeader";
import { Plus, Search } from "lucide-react";
import { RoleBindingsTable } from "./components/RoleBindingsTable";
import { useResponsiveCards } from "../../hooks/use-mobile";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import { Button } from "@/components/ui/button";
import { CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { SDKQuickHelp, ROLE_BINDINGS_SDK_HELP } from "@/features/sdk";
import { useTourStep, TOUR_REGISTRY } from "@/features/guided-tour";
import { PageInfoBanner } from "@/components/shared/PageInfoBanner";
import { Link2, UserCheck, Shield } from "lucide-react";
import { useLocation, useNavigate, useSearchParams } from "react-router-dom";

/**
 * Role Bindings page component - Visualize RBAC role bindings
 *
 * Features:
 * - Role bindings with user/role details
 * - Search and filter capabilities
 * - Optional filters: user_id, role_id, scope_type
 * - RBAC audience support (admin vs end-user)
 */
export function RoleBindingsPage() {
  const { isAdmin, audience } = useRbacAudience();
  const location = useLocation();
  const standardNavigate = useNavigate();
  const [searchParams, setSearchParams] = useSearchParams();
  const [roleScopeSearch, setRoleScopeSearch] = useState("");
  const [mapModalOpen, setMapModalOpen] = useState(false);

  // Initialize guided tour
  useTourStep({
    tourConfig: TOUR_REGISTRY["role-bindings-management"],
  });

  // Responsive card system
  const { mainAreaRef } = useResponsiveCards();

  // Auto-open modal if query param present (from wizard)
  useEffect(() => {
    if (searchParams.get("openModal") === "create") {
      setMapModalOpen(true);
      // Clean up query param while preserving location state
      searchParams.delete("openModal");
      setSearchParams(searchParams, { replace: true, state: location.state });
    }
  }, [searchParams, setSearchParams, location.state]);

  // Handle modal close with wizard awareness
  const handleBindingModalSuccess = () => {
    // Don't close modal here - it will be closed by onOpenChange
    // If coming from wizard, navigate back to root with success flag
    if (location.state?.fromWizard) {
      standardNavigate("/", { state: { bindingCreated: true } });
    }
  };

  // Copy based on audience
  const copy = useMemo(
    () => ({
      title: "Role Bindings",
      description: isAdmin
        ? "View and manage all role bindings with user and role details"
        : "View role bindings assigned to users in your organization",
      buttonText: "Create binding",
    }),
    [isAdmin],
  );

  return (
    <div className="min-h-screen" ref={mainAreaRef}>
      <div className="space-y-4 p-6 max-w-10xl mx-auto">
        {/* Header */}
        <PageHeader
          title={copy.title}
          description={copy.description}
          actions={
            <Button
              onClick={() => setMapModalOpen(true)}
              data-tour-id="create-binding-button"
            >
              <Plus className="mr-2 h-4 w-4" />
              {copy.buttonText}
            </Button>
          }
        />

        {/* Info Banner */}
        <PageInfoBanner
          title="Understanding Role Bindings"
          description="Role bindings connect users with roles, defining who has what permissions. They're the bridge between your users and their access rights in the system."
          features={[
            { text: "Assign roles to users and groups", icon: Link2 },
            { text: "Grant scoped access to specific resources", icon: Shield },
            { text: "Manage user permissions efficiently", icon: UserCheck },
          ]}
          featuresTitle="Key Capabilities"
          faqs={[
            {
              id: "1",
              question: "What are role bindings?",
              answer:
                "Role bindings are the associations between users and roles. When you bind a user to a role, they inherit all permissions that role has. Think of it as assigning job responsibilities to team members.",
            },
            {
              id: "2",
              question: "Can one user have multiple role bindings?",
              answer:
                "Yes! Users can be assigned multiple roles simultaneously. For example, a user might have both 'Editor' and 'Reviewer' roles, giving them a combined set of permissions from both roles.",
            },
            {
              id: "3",
              question: "How do scoped bindings work?",
              answer:
                "Scoped bindings restrict role permissions to specific resources or contexts. For example, you can bind a user to the 'Admin' role only for a specific project, rather than the entire system.",
            },
          ]}
          faqsTitle="Common Questions"
          storageKey="role-bindings-page-banner"
          dismissible={true}
        />

        {/* Filter Card */}
        <div data-tour-id="bindings-filters">
          <FilterShell>
            <CardContent variant="compact">
              <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
                <div className="flex shrink-0 items-center gap-2">
                  <span className="text-sm font-medium text-foreground">
                    Filters
                  </span>
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

        {/* Role Bindings Table */}
        <div data-tour-id="bindings-table">
          <RoleBindingsTable
            searchQuery={roleScopeSearch}
            isMapModalOpen={mapModalOpen}
            onMapModalOpenChange={setMapModalOpen}
            onBindingSuccess={handleBindingModalSuccess}
            audience={audience}
          />
        </div>
      </div>

      {/* SDK Quick Help */}
      <SDKQuickHelp
        title="Role Bindings SDK"
        helpItems={ROLE_BINDINGS_SDK_HELP}
      />
    </div>
  );
}
