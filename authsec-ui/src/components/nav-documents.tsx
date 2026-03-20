import { type LucideIcon, ChevronRight } from "lucide-react";
import { Fragment } from "react";
import { useLocation } from "react-router-dom";

import {
  SidebarGroup,
  SidebarGroupLabel,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarMenuSub,
  SidebarMenuSubButton,
  SidebarMenuSubItem,
  useSidebar,
} from "@/components/ui/sidebar";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/ui/collapsible";

export function NavDocuments({
  items,
  label,
}: {
  items: {
    name?: string;
    title?: string;
    url: string;
    icon: LucideIcon;
    onClick?: () => void;
    items?: {
      title: string;
      url: string;
      icon?: LucideIcon;
      onClick?: () => void;
    }[];
  }[];
  label?: string;
}) {
  const { state } = useSidebar();
  const isCollapsed = state === "collapsed";
  const location = useLocation();

  // Helper function to check if any subitem URL matches the current pathname
  const isSubmenuActive = (subItems: { url: string }[] | undefined): boolean => {
    if (!subItems) return false;
    return subItems.some((subItem) => location.pathname === subItem.url);
  };

  return (
    <SidebarGroup>
      {!isCollapsed && label && <SidebarGroupLabel>{label}</SidebarGroupLabel>}
      <SidebarMenu>
        {items.map((item, index) => {
          const itemName = item.name || item.title || item.url;
          const itemKey = `${itemName}-${index}`;

          if (isCollapsed) {
            if (item.items && item.items.length > 0) {
              return (
                <Fragment key={`${itemKey}-collapsed`}>
                  {item.items.map((subItem, subIndex) => {
                    const SubItemIcon = subItem.icon || item.icon;
                    return (
                      <SidebarMenuItem
                        key={`${itemKey}-collapsed-sub-${subIndex}`}
                      >
                        <SidebarMenuButton
                          tooltip={`${itemName} • ${subItem.title}`}
                          onClick={subItem.onClick}
                          className="cursor-pointer select-none"
                        >
                          {SubItemIcon && <SubItemIcon />}
                          <span>{subItem.title}</span>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    );
                  })}
                </Fragment>
              );
            }

            return (
              <SidebarMenuItem key={`${itemKey}-collapsed`}>
                <SidebarMenuButton
                  tooltip={itemName}
                  onClick={item.onClick}
                  className="cursor-pointer select-none"
                >
                  {item.icon && <item.icon />}
                  <span>{itemName}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            );
          }

          // If item has sub-items, render collapsible menu
          if (item.items && item.items.length > 0) {
            const shouldBeOpen = isSubmenuActive(item.items);

            return (
              <Collapsible
                key={itemKey}
                asChild
                defaultOpen={shouldBeOpen}
                className="group/collapsible"
              >
                <SidebarMenuItem>
                  <CollapsibleTrigger asChild>
                    <SidebarMenuButton
                      tooltip={itemName}
                      className="select-none"
                    >
                      {item.icon && <item.icon />}
                      <span>{itemName}</span>
                      <ChevronRight className="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90" />
                    </SidebarMenuButton>
                  </CollapsibleTrigger>
                  <CollapsibleContent>
                    <SidebarMenuSub>
                      {item.items.map((subItem, subIndex) => (
                        <SidebarMenuSubItem
                          key={`${subItem.title}-${subIndex}`}
                        >
                          <SidebarMenuSubButton
                            onClick={subItem.onClick}
                            className="cursor-pointer select-none"
                          >
                            <span>{subItem.title}</span>
                          </SidebarMenuSubButton>
                        </SidebarMenuSubItem>
                      ))}
                    </SidebarMenuSub>
                  </CollapsibleContent>
                </SidebarMenuItem>
              </Collapsible>
            );
          }

          // Regular item without sub-menu
          return (
            <SidebarMenuItem key={itemKey}>
              <SidebarMenuButton
                onClick={item.onClick}
                className="cursor-pointer select-none"
              >
                <item.icon />
                <span>{itemName}</span>
              </SidebarMenuButton>
            </SidebarMenuItem>
          );
        })}
      </SidebarMenu>
    </SidebarGroup>
  );
}
