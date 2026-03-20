import * as React from "react";
import { useResponsiveTable } from "@/hooks/use-mobile";

interface ResponsiveTableContextValue {
  visibleColumns: Record<string, boolean>;
  tableLayout: string;
  mainAreaWidth: number;
  isCompact: boolean;
  isMedium: boolean;
  isStandard: boolean;
  isFull: boolean;
}

const ResponsiveTableContext = React.createContext<ResponsiveTableContextValue | null>(null);

export function useResponsiveTableContext() {
  const context = React.useContext(ResponsiveTableContext);
  if (!context) {
    throw new Error("useResponsiveTableContext must be used within a ResponsiveTableProvider");
  }
  return context;
}

type ResponsiveTableType =
  | "services"
  | "roles"
  | "users"
  | "groups"
  | "externalServices"
  | "agents"
  | "auth"
  | "vault"
  | "logs"
  | "clients"
  | "resources"
  | string;

interface ResponsiveTableProviderProps {
  children: React.ReactNode;
  tableType: ResponsiveTableType;
  visibilityConfig?: Partial<Record<"minimal" | "compact" | "medium" | "standard" | "full", Record<string, boolean>>>;
}

export function ResponsiveTableProvider({
  children,
  tableType,
  visibilityConfig,
}: ResponsiveTableProviderProps) {
  const {
    mainAreaRef,
    tableLayout,
    getColumnVisibility,
    mainAreaWidth,
    isCompact,
    isMedium,
    isStandard,
    isFull,
  } = useResponsiveTable();

  const visibleColumns = React.useMemo(() => {
    if (visibilityConfig) {
      const layoutVisibility = visibilityConfig[tableLayout as keyof typeof visibilityConfig];
      if (layoutVisibility) {
        return layoutVisibility;
      }
      if (visibilityConfig.full) {
        return visibilityConfig.full;
      }
    }
    return getColumnVisibility(tableLayout, tableType);
  }, [tableLayout, tableType, getColumnVisibility, visibilityConfig]);

  const contextValue: ResponsiveTableContextValue = {
    visibleColumns,
    tableLayout,
    mainAreaWidth,
    isCompact,
    isMedium,
    isStandard,
    isFull,
  };

  return (
    <ResponsiveTableContext.Provider value={contextValue}>
      <div className="w-full space-y-6" ref={mainAreaRef}>
        {children}
      </div>
    </ResponsiveTableContext.Provider>
  );
}

// Utility component for conditional column rendering
interface ConditionalColumnProps {
  show: boolean;
  children: React.ReactNode;
}

export function ConditionalColumn({ show, children }: ConditionalColumnProps) {
  if (!show) return null;
  return <>{children}</>;
}

// Utility hook for getting column count for colSpan
export function useColumnCount(_tableType: string) {
  const { visibleColumns } = useResponsiveTableContext();

  return React.useMemo(() => {
    return Object.values(visibleColumns).filter(Boolean).length;
  }, [visibleColumns]);
}

// Column visibility helpers for each table type
export const ColumnVisibility = {
  // Policy table columns
  Policy: {
    DragHandle: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.dragHandle ? <>{children}</> : null;
    },
    Checkbox: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.checkbox ? <>{children}</> : null;
    },
    Type: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.type ? <>{children}</> : null;
    },
    Status: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.status ? <>{children}</> : null;
    },
    Impact: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.impact ? <>{children}</> : null;
    },
    Conditions: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.conditions ? <>{children}</> : null;
    },
    Resources: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.resources ? <>{children}</> : null;
    },
    Actions: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.actions ? <>{children}</> : null;
    },
  },

  // Services table columns
  Services: {
    DragHandle: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.dragHandle ? <>{children}</> : null;
    },
    Checkbox: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.checkbox ? <>{children}</> : null;
    },
    Service: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.service ? <>{children}</> : null;
    },
    Type: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.type ? <>{children}</> : null;
    },
    Health: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.health ? <>{children}</> : null;
    },
    Connections: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.connections ? <>{children}</> : null;
    },
    Uptime: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.uptime ? <>{children}</> : null;
    },
    Actions: ({ children }: { children: React.ReactNode }) => {
      const { visibleColumns } = useResponsiveTableContext();
      return visibleColumns.actions ? <>{children}</> : null;
    },
  },

  // Add more table types as needed...
};

// Layout indicator component
export function TableLayoutIndicator() {
  const { tableLayout, mainAreaWidth } = useResponsiveTableContext();

  if (process.env.NODE_ENV !== "development") return null;

  return (
    <div className="fixed bottom-4 right-4 bg-background border rounded-lg p-2 text-xs font-mono shadow-lg z-50">
      <div>
        Layout: <span className="font-bold">{tableLayout}</span>
      </div>
      <div>
        Width: <span className="font-bold">{mainAreaWidth}px</span>
      </div>
    </div>
  );
}
