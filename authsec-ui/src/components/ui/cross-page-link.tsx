import React from "react";
import { Link } from "react-router-dom";
import { Button } from "./button";
import { Badge } from "./badge";
import { cn } from "../../lib/utils";
import { useCrossPageNavigation, NavigationUtils } from "../../lib/cross-page-navigation";
import type { NavigationRoute, NavigationContext } from "../../lib/cross-page-navigation";
import {
  ExternalLink,
  Users,
  Shield,
  Building,
  Server,
  ArrowRight,
  Eye,
  Filter,
} from "lucide-react";

interface CrossPageLinkProps {
  variant?: "link" | "button" | "badge" | "inline";
  size?: "sm" | "md" | "lg";
  entityType: "user" | "group" | "role" | "resource";
  entityId: string;
  entityName: string;
  action?: "view" | "edit" | "filter";
  count?: number;
  showIcon?: boolean;
  showExternal?: boolean;
  className?: string;
  children?: React.ReactNode;
  preserveContext?: boolean;
  additionalContext?: Record<string, any>;
}

interface FilterLinkProps {
  variant?: "link" | "button" | "badge" | "inline";
  size?: "sm" | "md" | "lg";
  targetPage: NavigationRoute;
  filterKey: string;
  filterValue: string;
  filterLabel: string;
  count?: number;
  showIcon?: boolean;
  className?: string;
  children?: React.ReactNode;
  preserveContext?: boolean;
}

const ENTITY_ICONS = {
  user: Users,
  group: Users,
  role: Shield,
  resource: Building,
};

const ENTITY_COLORS = {
  user: "text-blue-600 bg-blue-50 border-blue-200",
  group: "text-green-600 bg-green-50 border-green-200",
  role: "text-blue-600 bg-blue-50 border-blue-200",
  resource: "text-orange-600 bg-orange-50 border-orange-200",
};

/**
 * Cross-page link component for navigating between entity pages with context
 */
export function CrossPageLink({
  variant = "link",
  size = "md",
  entityType,
  entityId,
  entityName,
  action = "view",
  count,
  showIcon = true,
  showExternal = false,
  className,
  children,
  preserveContext = true,
  additionalContext = {},
}: CrossPageLinkProps) {
  const { navigateWithContext, currentContext } = useCrossPageNavigation();
  const EntityIcon = ENTITY_ICONS[entityType];

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    const context = preserveContext
      ? {
          ...currentContext,
          ...additionalContext,
        }
      : additionalContext;

    const routeMap: Record<string, NavigationRoute> = {
      user: "users",
      group: "groups",
      role: "roles",
      resource: "resources",
    };

    navigateWithContext(routeMap[entityType], {
      ...context,
      entityId,
      action,
    });
  };

  const content = children || (
    <span className="flex items-center gap-1.5">
      {showIcon && <EntityIcon className={cn("h-3 w-3", size === "lg" && "h-4 w-4")} />}
      <span className="truncate">{entityName}</span>
      {count !== undefined && (
        <Badge variant="secondary" className="text-xs ml-1">
          {count}
        </Badge>
      )}
      {showExternal && <ExternalLink className="h-3 w-3 opacity-50" />}
    </span>
  );

  if (variant === "button") {
    return (
      <Button
        variant="outline"
        size={size === "lg" ? "default" : "sm"}
        onClick={handleClick}
        className={cn("justify-start", className)}
      >
        {content}
      </Button>
    );
  }

  if (variant === "badge") {
    return (
      <button
        onClick={handleClick}
        className={cn(
          "inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-medium border transition-colors hover:opacity-80",
          ENTITY_COLORS[entityType],
          className
        )}
      >
        {content}
      </button>
    );
  }

  if (variant === "inline") {
    return (
      <button
        onClick={handleClick}
        className={cn(
          "inline-flex items-center gap-1 text-sm font-medium text-primary hover:text-primary/80 transition-colors",
          className
        )}
      >
        {content}
      </button>
    );
  }

  // Default link variant
  return (
    <button
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:text-primary/80 hover:underline transition-colors",
        className
      )}
    >
      {content}
    </button>
  );
}

/**
 * Filter link component for navigating to filtered lists
 */
export function FilterLink({
  variant = "link",
  size = "md",
  targetPage,
  filterKey,
  filterValue,
  filterLabel,
  count,
  showIcon = true,
  className,
  children,
  preserveContext = true,
}: FilterLinkProps) {
  const { navigateWithContext, currentContext } = useCrossPageNavigation();

  const handleClick = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    const context: Partial<NavigationContext> = preserveContext ? currentContext : {};

    navigateWithContext(targetPage, {
      ...context,
      filters: {
        ...(context.filters || {}),
        [filterKey]: filterValue,
        [`${filterKey}Name`]: filterLabel,
      },
      action: "filter",
    });
  };

  const content = children || (
    <span className="flex items-center gap-1.5">
      {showIcon && <Filter className={cn("h-3 w-3", size === "lg" && "h-4 w-4")} />}
      <span className="truncate">{filterLabel}</span>
      {count !== undefined && (
        <Badge variant="secondary" className="text-xs ml-1">
          {count}
        </Badge>
      )}
      <ArrowRight className="h-3 w-3 opacity-50" />
    </span>
  );

  if (variant === "button") {
    return (
      <Button
        variant="outline"
        size={size === "lg" ? "default" : "sm"}
        onClick={handleClick}
        className={cn("justify-start", className)}
      >
        {content}
      </Button>
    );
  }

  if (variant === "badge") {
    return (
      <button
        onClick={handleClick}
        className={cn(
          "inline-flex items-center px-2.5 py-0.5 rounded-md text-xs font-medium border transition-colors hover:opacity-80",
          "text-gray-600 bg-gray-50 border-gray-200",
          className
        )}
      >
        {content}
      </button>
    );
  }

  if (variant === "inline") {
    return (
      <button
        onClick={handleClick}
        className={cn(
          "inline-flex items-center gap-1 text-sm font-medium text-primary hover:text-primary/80 transition-colors",
          className
        )}
      >
        {content}
      </button>
    );
  }

  // Default link variant
  return (
    <button
      onClick={handleClick}
      className={cn(
        "inline-flex items-center gap-1.5 text-sm font-medium text-primary hover:text-primary/80 hover:underline transition-colors",
        className
      )}
    >
      {content}
    </button>
  );
}

/**
 * Return navigation component for breadcrumb-style navigation
 */
export function ReturnLink({ className }: { className?: string }) {
  const { navigateBack, currentContext } = useCrossPageNavigation();

  if (!currentContext.sourceScreen || currentContext.sourceScreen === window.location.pathname) {
    return null;
  }

  const routeLabels: Record<string, string> = {
    "/users": "Users",
    "/groups": "Groups",
    "/roles": "Roles & Policies",
    "/resources": "Resources & Scopes",
    "/clients": "Clients",
  };

  const sourceLabel = Object.keys(routeLabels).find((route) =>
    currentContext.sourceScreen?.startsWith(route)
  );

  const label = sourceLabel ? routeLabels[sourceLabel] : "Back";

  return (
    <button
      onClick={navigateBack}
      className={cn(
        "inline-flex items-center gap-1.5 text-sm text-foreground hover:text-foreground transition-colors",
        className
      )}
    >
      <ArrowRight className="h-3 w-3 rotate-180" />
      <span>Back to {label}</span>
    </button>
  );
}

/**
 * Quick action components for common navigation patterns
 */
export const QuickLinks = {
  /**
   * Link to view users in a group
   */
  UsersInGroup: ({
    groupId,
    groupName,
    count,
  }: {
    groupId: string;
    groupName: string;
    count?: number;
  }) => (
    <FilterLink
      targetPage="users"
      filterKey="group"
      filterValue={groupId}
      filterLabel={groupName}
      count={count}
      variant="inline"
    />
  ),

  /**
   * Link to view users with a role
   */
  UsersWithRole: ({
    roleId,
    roleName,
    count,
  }: {
    roleId: string;
    roleName: string;
    count?: number;
  }) => (
    <FilterLink
      targetPage="users"
      filterKey="role"
      filterValue={roleId}
      filterLabel={roleName}
      count={count}
      variant="inline"
    />
  ),

  /**
   * Link to view groups with a role
   */
  GroupsWithRole: ({
    roleId,
    roleName,
    count,
  }: {
    roleId: string;
    roleName: string;
    count?: number;
  }) => (
    <FilterLink
      targetPage="groups"
      filterKey="role"
      filterValue={roleId}
      filterLabel={roleName}
      count={count}
      variant="inline"
    />
  ),

  /**
   * Link to view roles with a resource
   */
  RolesWithResource: ({
    resourceId,
    resourceName,
    count,
  }: {
    resourceId: string;
    resourceName: string;
    count?: number;
  }) => (
    <FilterLink
      targetPage="roles"
      filterKey="resource"
      filterValue={resourceId}
      filterLabel={resourceName}
      count={count}
      variant="inline"
    />
  ),

  /**
   * Link to view resources for a role
   */
  ResourcesForRole: ({
    roleId,
    roleName,
    count,
  }: {
    roleId: string;
    roleName: string;
    count?: number;
  }) => (
    <FilterLink
      targetPage="resources"
      filterKey="role"
      filterValue={roleId}
      filterLabel={roleName}
      count={count}
      variant="inline"
    />
  ),
};
