import { useEffect, useMemo, useState } from "react";
import { useLocation } from "react-router-dom";
import { Button } from "../ui/button";
import { ModeToggle } from "../mode-toggle";
import { Bell, Monitor, Sparkles } from "lucide-react";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger, useSidebar } from "@/components/ui/sidebar";
import { Breadcrumb } from "./Breadcrumb";
import { useResponsiveLayout } from "@/hooks/use-mobile";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { RbacAudienceSwitcher } from "@/features/rbac/RbacAudienceSwitcher";

interface AppHeaderProps {
  onRightSidebarToggle?: () => void;
  isRightSidebarOpen?: boolean;
}

export function AppHeader({ onRightSidebarToggle, isRightSidebarOpen = false }: AppHeaderProps) {
  const location = useLocation();
  const { shouldAutoCollapseSidebar } = useResponsiveLayout();
  const { open: sidebarOpen } = useSidebar();
  const [showAutoCollapseIndicator, setShowAutoCollapseIndicator] = useState(false);

  const shouldShowIndicator = shouldAutoCollapseSidebar && !sidebarOpen;

  useEffect(() => {
    if (shouldShowIndicator) {
      setShowAutoCollapseIndicator(true);
      const timer = setTimeout(() => {
        setShowAutoCollapseIndicator(false);
      }, 2000);

      return () => clearTimeout(timer);
    } else {
      setShowAutoCollapseIndicator(false);
    }
  }, [shouldShowIndicator]);

  const handleNotifications = () => {};

  const shouldShowRbacSwitcher = useMemo(() => {
    const RBAC_SEGMENTS = ["users", "groups", "roles", "resources", "permissions", "scopes", "api-oauth-scopes", "role-bindings"];
    const segments = location.pathname.split("/").filter(Boolean);

    return segments.some((segment) => RBAC_SEGMENTS.includes(segment));
  }, [location.pathname]);

  return (
    <header className="bg-[var(--app-shell-surface)] text-foreground border-[var(--app-shell-border)] flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4" />

        {/* Auto-collapse indicator */}
        {showAutoCollapseIndicator && (
          <Tooltip>
            <TooltipTrigger asChild>
              <div className="flex items-center gap-1 text-xs text-foreground bg-muted/50 px-2 py-1 rounded-md">
                <Monitor className="h-3 w-3" />
                <span>Auto-collapsed</span>
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p>Sidebar auto-collapsed to prevent horizontal scrolling</p>
            </TooltipContent>
          </Tooltip>
        )}

        <div className="flex-1">
          <Breadcrumb />
        </div>

        {shouldShowRbacSwitcher && (
          <>
            <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4" />
            <RbacAudienceSwitcher />
          </>
        )}

        {/* <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4" /> */}

        <div className="ml-auto flex items-center gap-2">
          {/* Notifications */}
          {/* <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 relative"
            onClick={handleNotifications}
          >
            <Bell className="h-4 w-4" />
            <span className="absolute -top-0.5 -right-0.5 h-2.5 w-2.5 bg-red-500 rounded-full text-[9px] flex items-center justify-center text-white font-medium">
              3
            </span>
          </Button> */}

          <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4" />

          {/* Theme toggle */}
          <ModeToggle />

          {/* <Separator orientation="vertical" className="mx-2 data-[orientation=vertical]:h-4" /> */}

          {/* Ask AI Button
          {onRightSidebarToggle && (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  className={`h-9 px-4 rounded-xl gap-2.5 font-semibold transition-all duration-200 shadow-sm ${
                    isRightSidebarOpen
                      ? "bg-gradient-to-r from-slate-700 to-slate-800 hover:from-slate-800 hover:to-slate-900 text-white shadow-lg shadow-slate-500/25"
                      : "bg-gradient-to-r from-slate-600 to-slate-700 hover:from-slate-700 hover:to-slate-800 text-white hover:shadow-md hover:shadow-slate-500/25"
                  }`}
                  onClick={onRightSidebarToggle}
                >
                  <Sparkles className="h-4 w-4" />
                  <span className="text-sm">Ask AI</span>
                </Button>
              </TooltipTrigger>
              <TooltipContent>
                <p>{isRightSidebarOpen ? "Close" : "Open"} AI Copilot</p>
              </TooltipContent>
            </Tooltip>
          )} */}


        </div>
      </div>
    </header>
  );
}
