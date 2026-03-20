import { useNavigate, useLocation, useSearchParams } from "react-router-dom";
import { useMemo, useCallback } from "react";
import type { CrossPageContext } from "../types/entities";

// Navigation routes for different entity types
export const NAVIGATION_ROUTES = {
  users: "/users",
  groups: "/groups",
  roles: "/roles",
  resources: "/resources",
  clients: "/clients",
} as const;

// Context keys for storing navigation state
export const CONTEXT_KEYS = {
  filters: "filters",
  selectedItems: "selectedItems",
  returnUrl: "returnUrl",
  sourceScreen: "sourceScreen",
  entityId: "entityId",
  action: "action",
} as const;

export type NavigationRoute = keyof typeof NAVIGATION_ROUTES;

// Cross-page navigation context
export interface NavigationContext {
  sourceScreen: string;
  filters?: Record<string, any>;
  selectedItems?: string[];
  returnUrl?: string;
  entityId?: string;
  action?: string;
}

/**
 * Hook for managing cross-page navigation with context preservation
 */
export function useCrossPageNavigation() {
  const navigate = useNavigate();
  const location = useLocation();
  const [searchParams, setSearchParams] = useSearchParams();

  // Parse current context from URL search params
  const currentContext = useMemo((): NavigationContext => {
    const context: NavigationContext = {
      sourceScreen: searchParams.get(CONTEXT_KEYS.sourceScreen) || location.pathname,
    };

    if (searchParams.get(CONTEXT_KEYS.filters)) {
      try {
        context.filters = JSON.parse(searchParams.get(CONTEXT_KEYS.filters) || "{}");
      } catch (e) {
        console.warn("Failed to parse filters from URL:", e);
      }
    }

    if (searchParams.get(CONTEXT_KEYS.selectedItems)) {
      try {
        context.selectedItems = JSON.parse(searchParams.get(CONTEXT_KEYS.selectedItems) || "[]");
      } catch (e) {
        console.warn("Failed to parse selectedItems from URL:", e);
      }
    }

    context.returnUrl = searchParams.get(CONTEXT_KEYS.returnUrl) || undefined;
    context.entityId = searchParams.get(CONTEXT_KEYS.entityId) || undefined;
    context.action = searchParams.get(CONTEXT_KEYS.action) || undefined;

    return context;
  }, [searchParams, location.pathname]);

  /**
   * Navigate to another page with context preservation
   */
  const navigateWithContext = useCallback(
    (
      route: NavigationRoute,
      context?: Partial<NavigationContext>,
      options?: { replace?: boolean }
    ) => {
      const params = new URLSearchParams();
      const navigationContext = { ...currentContext, ...context };

      // Set source screen if not provided
      if (!navigationContext.sourceScreen) {
        navigationContext.sourceScreen = location.pathname;
      }

      // Add context to URL params - only include essential parameters
      Object.entries(navigationContext).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          // Skip unnecessary parameters for cleaner URLs
          if (key === 'sourceScreen' || key === 'action') {
            return;
          }

          if (typeof value === "object") {
            // Only add filters to URL if they have active values
            if (key === 'filters' && !hasActiveFilters(value)) {
              return;
            } else if (key === 'selectedItems' && Array.isArray(value) && value.length === 0) {
              return;
            } else {
              params.set(key, JSON.stringify(value));
            }
          } else {
            params.set(key, String(value));
          }
        }
      });

      const targetUrl = `${NAVIGATION_ROUTES[route]}?${params.toString()}`;
      navigate(targetUrl, { replace: options?.replace });
    },
    [currentContext, location.pathname, navigate]
  );

  /**
   * Navigate back to the source screen with preserved context
   */
  const navigateBack = useCallback(() => {
    if (currentContext.returnUrl) {
      navigate(currentContext.returnUrl);
    } else if (currentContext.sourceScreen && currentContext.sourceScreen !== location.pathname) {
      navigate(currentContext.sourceScreen);
    } else {
      navigate(-1);
    }
  }, [currentContext.returnUrl, currentContext.sourceScreen, location.pathname, navigate]);

  /**
   * Check if filters have meaningful values (not empty/default)
   */
  const hasActiveFilters = (filters: Record<string, any>): boolean => {
    if (!filters) return false;
    
    return Object.entries(filters).some(([key, value]) => {
      if (Array.isArray(value)) return value.length > 0;
      if (typeof value === 'string') return value !== '' && value !== 'all';
      if (typeof value === 'boolean') return value === true; // Only include true boolean values
      return value !== null && value !== undefined;
    });
  };

  /**
   * Update current navigation context
   */
  const updateContext = useCallback((updates: Partial<NavigationContext>) => {
    const newContext = { ...currentContext, ...updates };
    const params = new URLSearchParams(searchParams);

    Object.entries(newContext).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        if (typeof value === "object") {
          // Only add filters to URL if they have active values
          if (key === 'filters' && !hasActiveFilters(value)) {
            params.delete(key);
          } else if (key === 'selectedItems' && Array.isArray(value) && value.length === 0) {
            params.delete(key);
          } else {
            params.set(key, JSON.stringify(value));
          }
        } else {
          // Skip sourceScreen if it's the same as current pathname
          if (key === 'sourceScreen' && value === location.pathname) {
            params.delete(key);
          } else {
            params.set(key, String(value));
          }
        }
      } else {
        params.delete(key);
      }
    });

    setSearchParams(params);
  }, [currentContext, searchParams, setSearchParams, location.pathname]);

  /**
   * Clear navigation context
   */
  const clearContext = useCallback(() => {
    setSearchParams({});
  }, [setSearchParams]);

  /**
   * Clear return context (legacy compatibility)
   */
  const clearReturnContext = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete(CONTEXT_KEYS.returnUrl);
    params.delete(CONTEXT_KEYS.sourceScreen);
    setSearchParams(params);
  }, [searchParams, setSearchParams]);

  /**
   * Get return URL (legacy compatibility)
   */
  const returnTo = currentContext.returnUrl;

  return {
    currentContext,
    navigateWithContext,
    navigateBack,
    updateContext,
    clearContext,
    clearReturnContext,
    returnTo,
  };
}

/**
 * Specific navigation utilities for different entity types
 */
export const NavigationUtils = {
  /**
   * Navigate to user details with context
   */
  goToUser: (
    userId: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("users", {
        ...context,
        entityId: userId,
        action: "view",
      });
    }
  },

  /**
   * Navigate to group details with context
   */
  goToGroup: (
    groupId: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("groups", {
        ...context,
        entityId: groupId,
        action: "view",
      });
    }
  },

  /**
   * Navigate to role details with context
   */
  goToRole: (
    roleId: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("roles", {
        ...context,
        entityId: roleId,
        action: "view",
      });
    }
  },

  /**
   * Navigate to resource details with context
   */
  goToResource: (
    resourceId: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("resources", {
        ...context,
        entityId: resourceId,
        action: "view",
      });
    }
  },

  /**
   * Navigate to users filtered by group
   */
  goToUsersInGroup: (
    groupId: string,
    groupName: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("users", {
        ...context,
        filters: {
          ...context?.filters,
          group: groupId,
          groupName,
        },
        action: "filter",
      });
    }
  },

  /**
   * Navigate to users filtered by role
   */
  goToUsersWithRole: (
    roleId: string,
    roleName: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("users", {
        ...context,
        filters: {
          ...context?.filters,
          role: roleId,
          roleName,
        },
        action: "filter",
      });
    }
  },

  /**
   * Navigate to groups with role filter
   */
  goToGroupsWithRole: (
    roleId: string,
    roleName: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("groups", {
        ...context,
        filters: {
          ...context?.filters,
          role: roleId,
          roleName,
        },
        action: "filter",
      });
    }
  },

  /**
   * Navigate to roles filtered by resource
   */
  goToRolesWithResource: (
    resourceId: string,
    resourceName: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("roles", {
        ...context,
        filters: {
          ...context?.filters,
          resource: resourceId,
          resourceName,
        },
        action: "filter",
      });
    }
  },

  /**
   * Navigate to resources filtered by role
   */
  goToResourcesForRole: (
    roleId: string,
    roleName: string,
    context?: Partial<NavigationContext>,
    navigate?: (route: NavigationRoute, context?: Partial<NavigationContext>) => void
  ) => {
    if (navigate) {
      navigate("resources", {
        ...context,
        filters: {
          ...context?.filters,
          role: roleId,
          roleName,
        },
        action: "filter",
      });
    }
  },

};

/**
 * Deep linking patterns for URLs
 */
export const DeepLinkPatterns = {
  /**
   * Generate a deep link to a user with context
   */
  userLink: (userId: string, action?: string) => {
    const params = new URLSearchParams();
    params.set(CONTEXT_KEYS.entityId, userId);
    if (action) params.set(CONTEXT_KEYS.action, action);
    return `${NAVIGATION_ROUTES.users}?${params.toString()}`;
  },

  /**
   * Generate a deep link to a group with context
   */
  groupLink: (groupId: string, action?: string) => {
    const params = new URLSearchParams();
    params.set(CONTEXT_KEYS.entityId, groupId);
    if (action) params.set(CONTEXT_KEYS.action, action);
    return `${NAVIGATION_ROUTES.groups}?${params.toString()}`;
  },

  /**
   * Generate a deep link to a role with context
   */
  roleLink: (roleId: string, action?: string) => {
    const params = new URLSearchParams();
    params.set(CONTEXT_KEYS.entityId, roleId);
    if (action) params.set(CONTEXT_KEYS.action, action);
    return `${NAVIGATION_ROUTES.roles}?${params.toString()}`;
  },

  /**
   * Generate a deep link to a resource with context
   */
  resourceLink: (resourceId: string, action?: string) => {
    const params = new URLSearchParams();
    params.set(CONTEXT_KEYS.entityId, resourceId);
    if (action) params.set(CONTEXT_KEYS.action, action);
    return `${NAVIGATION_ROUTES.resources}?${params.toString()}`;
  },

  /**
   * Generate a filtered list link
   */
  filteredListLink: (route: NavigationRoute, filters: Record<string, any>) => {
    const params = new URLSearchParams();
    params.set(CONTEXT_KEYS.filters, JSON.stringify(filters));
    params.set(CONTEXT_KEYS.action, "filter");
    return `${NAVIGATION_ROUTES[route]}?${params.toString()}`;
  },

};

/**
 * Context preservation utilities
 */
export const ContextUtils = {
  /**
   * Preserve current page state for navigation
   */
  preservePageState: (
    currentFilters: Record<string, any>,
    selectedItems: string[],
    sourcePage: string
  ): NavigationContext => {
    return {
      sourceScreen: sourcePage,
      filters: currentFilters,
      selectedItems,
      returnUrl: window.location.href,
    };
  },

  /**
   * Apply preserved context to page state
   */
  applyPreservedContext: (
    context: NavigationContext,
    setFilters: (filters: Record<string, any>) => void,
    setSelectedItems: (items: string[]) => void
  ) => {
    if (context.filters) {
      setFilters(context.filters);
    }
    if (context.selectedItems) {
      setSelectedItems(context.selectedItems);
    }
  },

  /**
   * Check if user came from a specific page
   */
  isFromPage: (context: NavigationContext, pageRoute: string): boolean => {
    return context.sourceScreen?.startsWith(pageRoute) || false;
  },

  /**
   * Get return breadcrumb information
   */
  getReturnBreadcrumb: (context: NavigationContext): { label: string; path: string } | null => {
    if (!context.sourceScreen) return null;

    const routeLabels: Record<string, string> = {
      "/users": "Users",
      "/groups": "Groups",
      "/roles": "Roles & Policies",
      "/resources": "Resources & Scopes",
      "/clients": "Clients",
    };

    const route = Object.keys(routeLabels).find((route) => context.sourceScreen?.startsWith(route));

    if (route) {
      return {
        label: routeLabels[route],
        path: context.returnUrl || context.sourceScreen,
      };
    }

    return null;
  },

};
