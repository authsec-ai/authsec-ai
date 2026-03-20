import {
  BadgeCheck,
  Bell,
  ChevronsUpDown,
  CreditCard,
  LogOut,
  Sparkles,
  Building,
  Users,
  Settings,
  Plus,
  Mic,
} from "lucide-react";
import { useAuth } from "@/auth/context/AuthContext";
import { useNavigate } from "react-router-dom";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";

import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuGroup,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from "@/components/ui/sidebar";

export function NavUser({
  user,
}: {
  user?: {
    name: string;
    email: string;
    avatar: string;
  };
  } = {}) {
  const { isMobile, state } = useSidebar();
  const isCollapsed = state === "collapsed";
  const navigate = useNavigate();
  const { user: authUser, currentProject, projects, signOut, switchProject } = useAuth();
  const { isAdmin } = useRbacAudience();

  const handleSignOut = async () => {
    await signOut();
  };

  const displayUser = authUser || user;
  const displayName = authUser
    ? authUser.first_name && authUser.last_name
      ? `${authUser.first_name} ${authUser.last_name}`
      : authUser.email.split("@")[0]
    : user?.name || "User";
  const displayEmail = authUser?.email || user?.email || "";

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground group-data-[collapsible=icon]:justify-center"
              tooltip={isCollapsed ? displayName : undefined}
            >
              <Avatar className="h-8 w-8 rounded-lg">
                <AvatarImage src={authUser?.avatar_url || user?.avatar} alt={displayName} />
                <AvatarFallback className="rounded-lg">
                  {displayName
                    .split(" ")
                    .map((n) => n[0])
                    .join("")
                    .toUpperCase()}
                </AvatarFallback>
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight group-data-[collapsible=icon]:hidden">
                <span className="truncate font-semibold">{displayName}</span>
                <span className="truncate text-xs">{displayEmail}</span>
              </div>
              <ChevronsUpDown className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-[--radix-dropdown-menu-trigger-width] min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <Avatar className="h-8 w-8 rounded-lg">
                  <AvatarImage src={authUser?.avatar_url || user?.avatar} alt={displayName} />
                  <AvatarFallback className="rounded-lg">
                    {displayName
                      .split(" ")
                      .map((n) => n[0])
                      .join("")
                      .toUpperCase()}
                  </AvatarFallback>
                </Avatar>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-semibold">{displayName}</span>
                  <span className="truncate text-xs">{displayEmail}</span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />

            {/* Project Section */}
            {currentProject && (
              <>
                <DropdownMenuLabel className="px-2 py-1.5 text-xs font-semibold text-foreground">
                  Current Project
                </DropdownMenuLabel>
                <DropdownMenuGroup>
                  <DropdownMenuItem className="flex items-center gap-2">
                    <Building className="h-4 w-4" />
                    <div className="flex flex-col">
                      <span className="font-medium">{currentProject.name}</span>
                      <span className="text-xs text-foreground">{currentProject.role}</span>
                    </div>
                  </DropdownMenuItem>
                </DropdownMenuGroup>

                {projects.length > 1 && (
                  <>
                    <DropdownMenuSeparator />
                    <DropdownMenuLabel className="px-2 py-1.5 text-xs font-semibold text-foreground">
                      Switch Project
                    </DropdownMenuLabel>
                    <DropdownMenuGroup>
                      {projects
                        .filter((p) => p.id !== currentProject.id)
                        .map((project) => (
                          <DropdownMenuItem
                            key={project.id}
                            onClick={async () => await switchProject(project.id)}
                            className="flex items-center gap-2"
                          >
                            <Building className="h-4 w-4" />
                            <div className="flex flex-col">
                              <span>{project.name}</span>
                              <span className="text-xs text-foreground">
                                {project.role}
                              </span>
                            </div>
                          </DropdownMenuItem>
                        ))}
                    </DropdownMenuGroup>
                  </>
                )}

                {/* Create New Project - Commented out since projects are created automatically */}
                {/* <DropdownMenuSeparator />
                <DropdownMenuGroup>
                  <DropdownMenuItem
                    onClick={() => navigate("/admin/create-workspace")}
                    className="flex items-center gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    <span>Create New Project</span>
                  </DropdownMenuItem>
                </DropdownMenuGroup> */}
                <DropdownMenuSeparator />
              </>
            )}

           
            {isAdmin && (
              <>
                <DropdownMenuLabel className="px-2 py-1.5 text-xs font-semibold text-foreground">
                  Voice Agents
                </DropdownMenuLabel>
                <DropdownMenuGroup>
                  <DropdownMenuItem
                    onClick={() => navigate("/admin/voice-agent")}
                    className="flex items-center gap-2"
                  >
                    <Mic className="h-4 w-4" />
                    <span>Add Voice Agent</span>
                  </DropdownMenuItem>
                </DropdownMenuGroup>
                <DropdownMenuSeparator />
              </>
            )}

            <DropdownMenuItem onClick={handleSignOut} className="text-red-600 focus:text-red-600">
              <LogOut />
              Sign out
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}
