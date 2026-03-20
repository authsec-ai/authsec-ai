import { useLocation, useNavigate } from "react-router-dom";
import { useAppDispatch } from "../../app/hooks";
import { setCurrentPage } from "../../app/slices/uiSlice";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { useEffect, useMemo, useState, useCallback } from "react";
import { toast } from "react-hot-toast";
import {
  Award,
  Code2,
  ClipboardList,
  CloudCog,
  Crosshair,
  FileLock2,
  Globe,
  GlobeLock,
  Key,
  LayoutDashboard,
  Network,
  ScrollText,
  Server,
  ServerCog,
  ShieldCheck,
  ShieldPlus,
  UserCog,
  Users,
  UserPlus,
  Workflow,
  Bot,
} from "lucide-react";
import { NavMain } from "@/components/nav-main";
import { NavDocuments } from "@/components/nav-documents";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { AuthSecLogo } from "@/components/ui/authsec-logo";
import { resolveTenantId } from "@/utils/workspace";
import { cn } from "@/lib/utils";

type NavItem = {
  title: string;
  url: string;
  icon: any;
  isActive?: boolean;
  items?: { title: string; url: string; icon?: any }[];
};

const STATIC_NAV_DATA = {
  navMain: [
    {
      title: "Dashboard",
      url: "/dashboard",
      icon: LayoutDashboard,
    },
  ],
  navSecurity: [
    {
      title: "Authentication",
      url: "/authentication",
      icon: ShieldCheck,
    },
    {
      title: "Trust Delegation",
      url: "/trust-delegation",
      icon: ShieldPlus,
    },
    {
      title: "SDK Hub",
      url: "/sdk",
      icon: Code2,
    },
    {
      title: "External Services & Secrets",
      url: "/external-services",
      icon: CloudCog,
    },
    {
      title: "Custom Domains",
      url: "/custom-domains",
      icon: Globe,
    },
  ],
  navClients: [
    {
      title: "Clients",
      url: "#",
      icon: Server,
      items: [
        {
          title: "MCP Servers / AI Agents",
          url: "/clients/mcp",
          icon: Bot,
        },
      ],
    },
  ],
  navM2M: [
    {
      title: "Workloads",
      url: "#",
      icon: Workflow,
      items: [
        {
          title: "Autonomous Workloads",
          url: "/clients/workloads",
          icon: Award,
        },
        {
          title: "SPIRE Agents",
          url: "/clients/agents",
          icon: Network,
        },
      ],
    },
  ],
  documents: [
    {
      name: "Logs",
      url: "#",
      icon: ScrollText,
      items: [
        {
          title: "Auth Logs",
          url: "/logs/auth",
          icon: FileLock2,
        },
        {
          title: "Audit Logs",
          url: "/logs/audit",
          icon: ClipboardList,
        },
        {
          title: "M2M Logs",
          url: "/logs/m2m",
          icon: ServerCog,
        },
      ],
    },
  ],
};

const CONTEXT_AWARE_NAV = [
  {
    title: "Users",
    url: "/users",
    icon: Users,
  },
];

const CONTEXT_AWARE_RBAC = [
  {
    title: "RBAC",
    url: "#",
    icon: Key,
    items: [
      {
        title: "Permissions and Resources",
        url: "/permissions",
        icon: ShieldPlus,
      },
      {
        title: "Roles and mapped permissions",
        url: "/roles",
        icon: UserCog,
      },
      {
        title: "Scopes",
        url: "/scopes",
        icon: Crosshair,
      },
      {
        title: "Role Bindings",
        url: "/role-bindings",
        icon: UserPlus,
      },
      {
        title: "API/OAuth Scopes",
        url: "/api-oauth-scopes",
        icon: GlobeLock,
      },
    ],
  },
];

export function AppSidebar({
  className,
  style,
  ...props
}: React.ComponentProps<typeof Sidebar>) {
  const location = useLocation();
  const navigate = useNavigate();
  const dispatch = useAppDispatch();
  const { audience } = useRbacAudience();
  const [tenantId, setTenantId] = useState<string | null>(null);

  useEffect(() => {
    if (typeof window === "undefined") return;
    try {
      setTenantId(resolveTenantId());
    } catch (error) {
      console.error("Failed to resolve tenant ID:", error);
      setTenantId(null);
    }
  }, []);

  const contextPrefix = audience === "admin" ? "/admin" : "/enduser";

  const handleNavigation = useCallback(
    (path: string, pageId: string) => {
      navigate(path);
      dispatch(setCurrentPage(pageId));
    },
    [navigate, dispatch],
  );

  const addContextToUrls = useCallback(
    (items: NavItem[], prefix: string): NavItem[] => {
      return items.map((item) => ({
        ...item,
        url: item.url === "#" ? item.url : `${prefix}${item.url}`,
        items: item.items?.map((subItem) => ({
          ...subItem,
          url: `${prefix}${subItem.url}`,
        })),
      }));
    },
    [],
  );

  const updateActiveStates = useCallback(
    (items: NavItem[]): NavItem[] => {
      return items.map((item) => ({
        ...item,
        isActive:
          location.pathname === item.url ||
          (item.url !== "#" &&
            item.url !== "/" &&
            location.pathname.startsWith(`${item.url}/`)) ||
          (item.url === "/dashboard" && location.pathname === "/"),
      }));
    },
    [location.pathname],
  );

  const addNavigationHandlers = useCallback(
    (items: any[]) => {
      return items.map((item) => ({
        ...item,
        onClick:
          item.url !== "#"
            ? () =>
                handleNavigation(
                  item.url,
                  (item.title || item.name || "")
                    .toLowerCase()
                    .replace(/\s+/g, "-"),
                )
            : undefined,
        items: item.items?.map((subItem) => ({
          ...subItem,
          onClick: () =>
            handleNavigation(
              subItem.url,
              subItem.title.toLowerCase().replace(/\s+/g, "-"),
            ),
        })),
      }));
    },
    [handleNavigation],
  );

  const navData = useMemo(
    () => ({
      navMain: addNavigationHandlers(
        updateActiveStates(STATIC_NAV_DATA.navMain),
      ),
      navClients: addNavigationHandlers(STATIC_NAV_DATA.navClients),
      navM2M: addNavigationHandlers(STATIC_NAV_DATA.navM2M),
      navContextAware: addNavigationHandlers(
        updateActiveStates(addContextToUrls(CONTEXT_AWARE_NAV, contextPrefix)),
      ),
      navRbac: addNavigationHandlers(
        addContextToUrls(CONTEXT_AWARE_RBAC, contextPrefix),
      ),
      navSecurity: addNavigationHandlers(
        updateActiveStates(STATIC_NAV_DATA.navSecurity),
      ),
      documents: addNavigationHandlers(STATIC_NAV_DATA.documents),
    }),
    [
      contextPrefix,
      addContextToUrls,
      updateActiveStates,
      addNavigationHandlers,
    ],
  );

  const handleTenantIdClick = useCallback(
    async (e: React.MouseEvent) => {
      e.stopPropagation();
      if (tenantId) {
        try {
          await navigator.clipboard.writeText(tenantId);
          toast.success("Tenant ID copied to clipboard");
        } catch (err) {
          console.error("Failed to copy:", err);
          toast.error("Failed to copy Tenant ID");
        }
      }
    },
    [tenantId],
  );

  const tenantIdLabel = useMemo(() => {
    if (!tenantId) return "Not available";
    if (tenantId.length <= 24) return tenantId;
    return `${tenantId.slice(0, 12)}…${tenantId.slice(-8)}`;
  }, [tenantId]);

  return (
    <Sidebar
      collapsible="icon"
      className={cn(
        "border-r border-[var(--app-shell-border)] bg-[var(--app-shell-surface)] [&_[data-slot=sidebar-inner]]:bg-[var(--app-shell-surface)]",
        className,
      )}
      style={
        {
          "--sidebar-surface": "var(--app-shell-surface)",
          "--sidebar-border": "var(--app-shell-border)",
          ...style,
        } as React.CSSProperties
      }
      {...props}
    >
      <SidebarHeader className="h-(--header-height) justify-center gap-0 border-b border-[var(--app-shell-border)] p-2">
        <SidebarMenu className="px-2 group-data-[collapsible=icon]:px-0">
          <SidebarMenuItem>
            <SidebarMenuButton
              className="h-auto min-h-10 items-center rounded-md px-2.5 py-1.5 group-data-[collapsible=icon]:min-h-8 group-data-[collapsible=icon]:justify-center group-data-[collapsible=icon]:p-0 hover:bg-sidebar-accent/60"
              onClick={() => handleNavigation("/dashboard", "dashboard")}
            >
              <div className="flex w-full min-w-0 items-center gap-2 group-data-[collapsible=icon]:w-auto group-data-[collapsible=icon]:justify-center">
                <div className="flex h-6 w-6 shrink-0 items-center justify-center">
                  <AuthSecLogo className="size-5" />
                </div>
                <div className="flex min-w-0 flex-1 flex-col gap-0 group-data-[collapsible=icon]:hidden">
                  <span className="truncate text-[13px] font-semibold leading-tight tracking-tight text-sidebar-foreground">
                    AuthSec
                  </span>
                  <span
                    className="block truncate text-[9px] font-mono leading-tight text-sidebar-foreground/60 transition-colors hover:text-sidebar-foreground"
                    title={tenantId ?? undefined}
                    onClick={handleTenantIdClick}
                  >
                    {tenantIdLabel}
                  </span>
                </div>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>

      <SidebarContent>
        <NavMain items={navData.navMain} />
        <NavDocuments items={navData.navClients} />
        <NavDocuments items={navData.navM2M} />
        <NavMain items={navData.navContextAware} />
        <NavDocuments items={navData.navRbac} />
        <NavMain items={navData.navSecurity} />
        <NavDocuments items={navData.documents} />
      </SidebarContent>

      <SidebarFooter className="mt-auto gap-0 border-t border-[var(--app-shell-border)]">
        <NavUser />
      </SidebarFooter>
    </Sidebar>
  );
}
