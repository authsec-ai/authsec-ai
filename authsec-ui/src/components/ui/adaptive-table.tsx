import * as React from "react";
import type { Row } from "@tanstack/react-table";

import {
  ResponsiveDataTable,
  type ResponsiveColumnDef,
  type ResponsiveTableConfig,
} from "./responsive-data-table";
import { ResponsiveTableProvider } from "./responsive-table";

type AdaptiveLayout = "minimal" | "compact" | "medium" | "standard" | "full";

export interface AdaptiveColumn<TData, TValue = unknown>
  extends ResponsiveColumnDef<TData, TValue> {
  priority?: number;
  alwaysVisible?: boolean;
  approxWidth?: number;
}

interface AdaptiveTableProps<TData> {
  tableId: string;
  data: TData[];
  columns: AdaptiveColumn<TData, any>[];
  rowClassName?: ResponsiveTableConfig<TData>["rowClassName"];
  enableSelection?: boolean;
  selectedRowIds?: string[];
  onRowSelectionChange?: (selectedIds: string[]) => void;
  onSelectAll?: () => void;
  enableExpansion?: boolean;
  expandedRowIds?: string[];
  onExpandedRowsChange?: (expandedIds: string[]) => void;
  renderExpandedRow?: (row: Row<TData>) => React.ReactNode;
  enableSorting?: boolean;
  enableResizing?: boolean;
  enablePagination?: boolean;
  pagination?: ResponsiveTableConfig<TData>["pagination"];
  pageIndex?: number;
  onPageIndexChange?: (page: number) => void;
  serverTotalItems?: number;
  getRowId: (row: TData) => string;
  className?: string;
}

const DEFAULT_COLUMN_WIDTH = 220;
const SELECTION_COLUMN_WIDTH = 56;
const EXPAND_COLUMN_WIDTH = 48;

export function AdaptiveTable<TData>({
  tableId,
  data,
  columns,
  rowClassName,
  enableSelection = true,
  selectedRowIds,
  onRowSelectionChange,
  onSelectAll,
  enableExpansion = false,
  expandedRowIds,
  onExpandedRowsChange,
  renderExpandedRow,
  enableSorting = true,
  enableResizing = true,
  enablePagination = true,
  pagination = {
    pageSize: 10,
    pageSizeOptions: [5, 10, 25, 50, 100],
    alwaysVisible: true,
  },
  pageIndex,
  onPageIndexChange,
  serverTotalItems,
  getRowId,
  className,
}: AdaptiveTableProps<TData>) {
  const containerRef = React.useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = React.useState(0);

  React.useEffect(() => {
    if (typeof window === "undefined" || !containerRef.current) return;

    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        if (entry.target === containerRef.current) {
          setContainerWidth(entry.contentRect.width);
        }
      }
    });

    observer.observe(containerRef.current);

    return () => observer.disconnect();
  }, []);

  const columnIds = React.useMemo(() => columns.map((column) => column.id), [columns]);

  const alwaysVisibleColumns = React.useMemo(
    () => columns.filter((column) => column.alwaysVisible),
    [columns]
  );

  const optionalColumns = React.useMemo(
    () =>
      columns
        .filter((column) => !column.alwaysVisible)
        .sort((a, b) => (a.priority ?? 10) - (b.priority ?? 10)),
    [columns]
  );

  const visibleColumnSet = React.useMemo(() => {
    const fallbackSet = new Set(alwaysVisibleColumns.map((column) => column.id));

    if (containerWidth <= 0) {
      return fallbackSet;
    }

    const reservedWidth =
      alwaysVisibleColumns.reduce(
        (sum, column) => sum + (column.approxWidth ?? DEFAULT_COLUMN_WIDTH),
        0
      ) +
      (enableSelection ? SELECTION_COLUMN_WIDTH : 0) +
      (enableExpansion && renderExpandedRow ? EXPAND_COLUMN_WIDTH : 0);

    let remainingWidth = Math.max(containerWidth - reservedWidth, 0);
    const dynamicSet = new Set(alwaysVisibleColumns.map((column) => column.id));

    for (const column of optionalColumns) {
      const width = column.approxWidth ?? DEFAULT_COLUMN_WIDTH;
      if (remainingWidth >= width) {
        dynamicSet.add(column.id);
        remainingWidth -= width;
      } else {
        break;
      }
    }

    return dynamicSet.size > 0 ? dynamicSet : fallbackSet;
  }, [
    alwaysVisibleColumns,
    optionalColumns,
    containerWidth,
    enableSelection,
    enableExpansion,
    renderExpandedRow,
  ]);

  const visibilityConfig = React.useMemo(() => {
    const layouts: AdaptiveLayout[] = ["minimal", "compact", "medium", "standard", "full"];

    const baseVisibility: Record<string, boolean> = {};
    columnIds.forEach((id) => {
      baseVisibility[id] = visibleColumnSet.has(id);
    });
    baseVisibility.checkbox = enableSelection;
    baseVisibility.dragHandle = false;
    baseVisibility.expand = enableExpansion && Boolean(renderExpandedRow);

    return layouts.reduce<Record<AdaptiveLayout, Record<string, boolean>>>((acc, layout) => {
      acc[layout] = { ...baseVisibility };
      return acc;
    }, {} as Record<AdaptiveLayout, Record<string, boolean>>);
  }, [
    columnIds,
    visibleColumnSet,
    enableSelection,
    enableExpansion,
    renderExpandedRow,
  ]);

  const tableConfig: ResponsiveTableConfig<TData> = React.useMemo(
    () => ({
      data,
      columns,
      features: {
        selection: enableSelection,
        dragDrop: false,
        expandable: enableExpansion && Boolean(renderExpandedRow),
        pagination: enablePagination,
        sorting: enableSorting,
        resizing: enableResizing,
      },
      pagination,
      selectedRowIds,
      onRowSelectionChange,
      onSelectAll,
      expandedRowIds,
      onExpandedRowsChange,
      renderExpandedRow,
      getRowId,
      rowClassName,
      className,
      pageIndex,
      onPageIndexChange,
      serverTotalItems,
    }),
    [
      data,
      columns,
      enableSelection,
      enableExpansion,
      renderExpandedRow,
      enablePagination,
      enableSorting,
      enableResizing,
      pagination,
      selectedRowIds,
      onRowSelectionChange,
      onSelectAll,
      expandedRowIds,
      onExpandedRowsChange,
      getRowId,
      rowClassName,
      className,
      pageIndex,
      onPageIndexChange,
      serverTotalItems,
    ]
  );

  return (
    <ResponsiveTableProvider
      tableType={`adaptive-${tableId}`}
      visibilityConfig={visibilityConfig}
    >
      <div ref={containerRef} className="w-full overflow-x-hidden">
        <ResponsiveDataTable {...tableConfig} />
      </div>
    </ResponsiveTableProvider>
  );
}
