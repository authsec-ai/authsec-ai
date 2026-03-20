import * as React from "react";

const MOBILE_BREAKPOINT = 768;
const SIDEBAR_AUTO_COLLAPSE_BREAKPOINT = 1200; // Sidebar should collapse when screen < 1200px

export function useIsMobile() {
  const [isMobile, setIsMobile] = React.useState<boolean | undefined>(undefined);

  React.useEffect(() => {
    const mql = window.matchMedia(`(max-width: ${MOBILE_BREAKPOINT - 1}px)`);
    const onChange = () => {
      setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    };
    mql.addEventListener("change", onChange);
    setIsMobile(window.innerWidth < MOBILE_BREAKPOINT);
    return () => mql.removeEventListener("change", onChange);
  }, []);

  return !!isMobile;
}

export function useResponsiveLayout() {
  const [dimensions, setDimensions] = React.useState({
    width: typeof window !== "undefined" ? window.innerWidth : 1200,
    height: typeof window !== "undefined" ? window.innerHeight : 800,
  });

  React.useEffect(() => {
    const handleResize = () => {
      setDimensions({
        width: window.innerWidth,
        height: window.innerHeight,
      });
    };

    window.addEventListener("resize", handleResize);
    return () => window.removeEventListener("resize", handleResize);
  }, []);

  const isMobile = dimensions.width < MOBILE_BREAKPOINT;
  const shouldAutoCollapseSidebar = dimensions.width < SIDEBAR_AUTO_COLLAPSE_BREAKPOINT;
  const hasConstrainedSpace = dimensions.width < 1400; // Even more constrained

  return {
    width: dimensions.width,
    height: dimensions.height,
    isMobile,
    shouldAutoCollapseSidebar,
    hasConstrainedSpace,
    isDesktop: dimensions.width >= MOBILE_BREAKPOINT,
    isLargeScreen: dimensions.width >= 1400,
  };
}

// Hook specifically for tracking main display area and responsive table columns
export function useResponsiveTable() {
  const [mainAreaWidth, setMainAreaWidth] = React.useState(0);
  const mainAreaRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const updateMainAreaWidth = () => {
      if (mainAreaRef.current) {
        const rect = mainAreaRef.current.getBoundingClientRect();
        setMainAreaWidth(rect.width);
      }
    };

    // Initial measurement
    updateMainAreaWidth();

    // Create ResizeObserver to track main area width changes with throttling
    let timeoutId: NodeJS.Timeout;
    const resizeObserver = new ResizeObserver((entries) => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        for (const entry of entries) {
          const width = entry.contentRect.width;
          setMainAreaWidth(width);
        }
      }, 16); // 16ms throttle for 60fps
    });

    if (mainAreaRef.current) {
      resizeObserver.observe(mainAreaRef.current);
    }

    // Also listen to window resize as fallback
    window.addEventListener("resize", updateMainAreaWidth);

    return () => {
      clearTimeout(timeoutId);
      resizeObserver.disconnect();
      window.removeEventListener("resize", updateMainAreaWidth);
    };
  }, []);

  // Define responsive breakpoints based on main area width - less aggressive
  const getTableLayout = (mainWidth: number) => {
    if (mainWidth < 400) {
      return "minimal"; // Very small screens
    } else if (mainWidth < 600) {
      return "compact"; // Small screens
    } else if (mainWidth < 800) {
      return "medium"; // Medium screens
    } else if (mainWidth < 1000) {
      return "standard"; // Standard screens
    } else {
      return "full"; // Large screens - all columns
    }
  };

  const tableLayout = getTableLayout(mainAreaWidth);

  // More aggressive column visibility configurations
  const getColumnVisibility = (layout: string, tableType: string) => {
    const configs = {
      // Policy Table Configurations
      policy: {
        minimal: {
          dragHandle: true,
          checkbox: true,
          policy: true,
          type: false,
          status: true,
          impact: false,
          conditions: false,
          resources: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          policy: true,
          type: true,
          status: true,
          impact: false,
          conditions: false,
          resources: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          policy: true,
          type: true,
          status: true,
          impact: false,
          conditions: true,
          resources: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          policy: true,
          type: true,
          status: true,
          impact: true,
          conditions: true,
          resources: false,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          policy: true,
          type: true,
          status: true,
          impact: true,
          conditions: true,
          resources: true,
          actions: true,
        },
      },
      // Services Table Configurations
      services: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          service: true,
          team: false,
          access: true,
          agents: false,
          requests: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          service: true,
          team: false,
          access: true,
          agents: false,
          requests: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          service: true,
          team: false,
          access: true,
          agents: false,
          requests: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          service: true,
          team: true,
          access: true,
          agents: true,
          requests: false,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          service: true,
          team: true,
          access: true,
          agents: true,
          requests: true,
          actions: true,
        },
      },
      // Roles Table Configurations
      roles: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          role: true,
          typeStatus: false,
          permissions: false,
          usersCount: true,
          groupsCount: false,
          created: false,
          expand: true,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          role: true,
          typeStatus: false,
          permissions: false,
          usersCount: true,
          groupsCount: false,
          created: false,
          expand: true,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          role: true,
          typeStatus: true,
          permissions: false,
          usersCount: true,
          groupsCount: false,
          created: false,
          expand: true,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          role: true,
          typeStatus: true,
          permissions: false,
          usersCount: true,
          groupsCount: false,
          created: false,
          expand: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          role: true,
          typeStatus: true,
          permissions: true,
          usersCount: true,
          groupsCount: true,
          created: true,
          expand: true,
          actions: true,
        },
      },
      scopes: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          scope: true,
          scopeId: true,
          description: false,
          created: false,
          expand: true,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          scope: true,
          scopeId: true,
          description: true,
          created: false,
          expand: true,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          scope: true,
          scopeId: true,
          description: true,
          created: true,
          expand: true,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          scope: true,
          scopeId: true,
          description: true,
          created: true,
          expand: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          scope: true,
          scopeId: true,
          description: true,
          created: true,
          expand: true,
          actions: true,
        },
      },
      permissions: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          role: true,
          scope: false,
          resource: false,
          created: false,
          expand: true,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          role: true,
          scope: true,
          resource: false,
          created: false,
          expand: true,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          role: true,
          scope: true,
          resource: true,
          created: false,
          expand: true,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          role: true,
          scope: true,
          resource: true,
          created: true,
          expand: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          role: true,
          scope: true,
          resource: true,
          created: true,
          expand: true,
          actions: true,
        },
      },
      // Users Table Configurations
      users: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          user: true,
          roleTeam: false,
          status: true,
          activity: false,
          permissions: false,
          expand: true,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          user: true,
          roleTeam: false,
          status: true,
          activity: false,
          permissions: false,
          expand: true,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          user: true,
          roleTeam: true,
          status: true,
          activity: false,
          permissions: false,
          expand: true,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          user: true,
          roleTeam: true,
          status: true,
          activity: true,
          permissions: false,
          expand: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          user: true,
          roleTeam: true,
          status: true,
          activity: true,
          permissions: true,
          expand: true,
          actions: true,
        },
      },
      // Groups Table Configurations
      groups: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          group: true,
          typeStatus: false,
          members: true,
          rules: false,
          created: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          group: true,
          typeStatus: false,
          members: true,
          rules: false,
          created: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          group: true,
          typeStatus: true,
          members: true,
          rules: false,
          created: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          group: true,
          typeStatus: true,
          members: true,
          rules: true,
          created: false,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          group: true,
          typeStatus: true,
          members: true,
          rules: true,
          created: true,
          actions: true,
        },
      },
      // Agents Table Configurations
      agents: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          agent: true,
          type: false,
          status: true,
          activity: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          agent: true,
          type: false,
          status: true,
          activity: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          agent: true,
          type: true,
          status: true,
          activity: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          agent: true,
          type: true,
          status: true,
          activity: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          agent: true,
          type: true,
          status: true,
          activity: true,
          actions: true,
        },
      },
      // Authentication Table Configurations
      auth: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          method: true,
          provider: false,
          clients: false,
          error: false,
          expiry: false,
          status: true,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          method: true,
          provider: false,
          clients: false,
          error: false,
          expiry: false,
          status: true,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          method: true,
          provider: true,
          clients: true,
          error: false,
          expiry: false,
          status: true,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          method: true,
          provider: true,
          clients: true,
          error: true,
          expiry: false,
          status: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          method: true,
          provider: true,
          clients: true,
          error: true,
          expiry: true,
          status: true,
          actions: true,
        },
      },
      // Vault Table Configurations
      vault: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          secret: true,
          typeTags: false,
          status: true,
          activity: false,
          expires: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          secret: true,
          typeTags: false,
          status: true,
          activity: false,
          expires: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          secret: true,
          typeTags: true,
          status: true,
          activity: false,
          expires: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          secret: true,
          typeTags: true,
          status: true,
          activity: true,
          expires: false,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          secret: true,
          typeTags: true,
          status: true,
          activity: true,
          expires: true,
          actions: true,
        },
      },
      // Clients Table Configurations
      clients: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          client: true,
          clientId: false,
          authMethods: false,
          totalUsers: false,
          expand: true,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          client: true,
          clientId: true,
          authMethods: false,
          totalUsers: false,
          expand: true,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          client: true,
          clientId: true,
          authMethods: true,
          totalUsers: false,
          expand: true,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          client: true,
          clientId: true,
          authMethods: true,
          totalUsers: true,
          expand: true,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          client: true,
          clientId: true,
          authMethods: true,
          totalUsers: true,
          expand: true,
          actions: true,
        },
      },
      resources: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          resource: true,
          linkedRoles: false,
          expand: true,
          actions: true,
        },
        compact: {
          dragHandle: false,
          checkbox: true,
          resource: true,
          linkedRoles: true,
          expand: true,
          actions: true,
        },
        medium: {
          dragHandle: false,
          checkbox: true,
          resource: true,
          linkedRoles: true,
          expand: true,
          actions: true,
        },
        standard: {
          dragHandle: false,
          checkbox: true,
          resource: true,
          linkedRoles: true,
          expand: true,
          actions: true,
        },
        full: {
          dragHandle: false,
          checkbox: true,
          resource: true,
          linkedRoles: true,
          expand: true,
          actions: true,
        },
      },
      // Logs Table Configurations
      logs: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          levelEvent: true,
          serviceSource: false,
          userAgent: false,
          timestamp: true,
          details: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          levelEvent: true,
          serviceSource: false,
          userAgent: false,
          timestamp: true,
          details: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          levelEvent: true,
          serviceSource: true,
          userAgent: false,
          timestamp: true,
          details: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          levelEvent: true,
          serviceSource: true,
          userAgent: true,
          timestamp: true,
          details: false,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          levelEvent: true,
          serviceSource: true,
          userAgent: true,
          timestamp: true,
          details: true,
          actions: true,
        },
      },
      externalServices: {
        minimal: {
          dragHandle: false,
          checkbox: true,
          service: true,
          provider: false,
          clientCount: false,
          userTokenCount: false,
          status: true,
          lastSync: false,
          actions: true,
        },
        compact: {
          dragHandle: true,
          checkbox: true,
          service: true,
          provider: true,
          clientCount: false,
          userTokenCount: false,
          status: true,
          lastSync: false,
          actions: true,
        },
        medium: {
          dragHandle: true,
          checkbox: true,
          service: true,
          provider: true,
          clientCount: true,
          userTokenCount: false,
          status: true,
          lastSync: false,
          actions: true,
        },
        standard: {
          dragHandle: true,
          checkbox: true,
          service: true,
          provider: true,
          clientCount: true,
          userTokenCount: true,
          status: true,
          lastSync: false,
          actions: true,
        },
        full: {
          dragHandle: true,
          checkbox: true,
          service: true,
          provider: true,
          clientCount: true,
          userTokenCount: true,
          status: true,
          lastSync: true,
          actions: true,
        },
      },
    };

    return (
      configs[tableType as keyof typeof configs]?.[layout as keyof typeof configs.policy] ||
      configs.policy.full
    );
  };

  return {
    mainAreaRef,
    mainAreaWidth,
    tableLayout,
    getColumnVisibility,
    isMinimal: tableLayout === "minimal",
    isCompact: tableLayout === "compact",
    isMedium: tableLayout === "medium",
    isStandard: tableLayout === "standard",
    isFull: tableLayout === "full",
  };
}

// Hook for responsive card grids that adapts to main content area width
export function useResponsiveCards() {
  const [mainAreaWidth, setMainAreaWidth] = React.useState(0);
  const mainAreaRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const updateMainAreaWidth = () => {
      if (mainAreaRef.current) {
        const rect = mainAreaRef.current.getBoundingClientRect();
        setMainAreaWidth(rect.width);
      }
    };

    // Initial measurement
    updateMainAreaWidth();

    // Create ResizeObserver to track main area width changes with throttling
    let timeoutId: NodeJS.Timeout;
    const resizeObserver = new ResizeObserver((entries) => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        for (const entry of entries) {
          const width = entry.contentRect.width;
          setMainAreaWidth(width);
        }
      }, 16); // 16ms throttle for 60fps
    });

    if (mainAreaRef.current) {
      resizeObserver.observe(mainAreaRef.current);
    }

    // Also listen to window resize as fallback
    window.addEventListener("resize", updateMainAreaWidth);

    return () => {
      clearTimeout(timeoutId);
      resizeObserver.disconnect();
      window.removeEventListener("resize", updateMainAreaWidth);
    };
  }, []);

  // Define responsive card grid configurations based on main area width
  const getCardGridConfig = (mainWidth: number) => {
    if (mainWidth < 480) {
      return {
        layout: "minimal",
        gridClasses: "grid-cols-1",
        cardSize: "compact",
        showDescription: false,
        showSubMetrics: false,
      };
    } else if (mainWidth < 640) {
      return {
        layout: "compact",
        gridClasses: "grid-cols-1 sm:grid-cols-2",
        cardSize: "standard",
        showDescription: true,
        showSubMetrics: false,
      };
    } else if (mainWidth < 900) {
      return {
        layout: "medium",
        gridClasses: "grid-cols-1 sm:grid-cols-2 md:grid-cols-3",
        cardSize: "standard",
        showDescription: true,
        showSubMetrics: true,
      };
    } else if (mainWidth < 1200) {
      return {
        layout: "standard",
        gridClasses: "grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4",
        cardSize: "standard",
        showDescription: true,
        showSubMetrics: true,
      };
    } else {
      return {
        layout: "full",
        gridClasses: "grid-cols-1 sm:grid-cols-2 md:grid-cols-3 xl:grid-cols-4",
        cardSize: "standard",
        showDescription: true,
        showSubMetrics: true,
      };
    }
  };

  const cardGridConfig = getCardGridConfig(mainAreaWidth);

  return {
    mainAreaRef,
    mainAreaWidth,
    cardGridConfig,
    isMinimal: cardGridConfig.layout === "minimal",
    isCompact: cardGridConfig.layout === "compact",
    isMedium: cardGridConfig.layout === "medium",
    isStandard: cardGridConfig.layout === "standard",
    isFull: cardGridConfig.layout === "full",
  };
}
