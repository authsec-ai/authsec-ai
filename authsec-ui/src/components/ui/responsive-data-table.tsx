import * as React from "react";
import {
  useReactTable,
  getCoreRowModel,
  getSortedRowModel,
  getFilteredRowModel,
  flexRender,
  type ColumnDef,
  type SortingState,
  type ColumnFiltersState,
  type VisibilityState,
  type Row,
  type AccessorFn,
  type ColumnDefTemplate,
  type HeaderContext,
  type CellContext,
  type ColumnMeta,
} from "@tanstack/react-table";
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  type DragEndEvent,
} from "@dnd-kit/core";
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { IconGripVertical } from "@tabler/icons-react";
import { ChevronDown, ChevronUp, ChevronRight } from "lucide-react";

import { cn } from "../../lib/utils";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "./table";
import { Button } from "./button";
import { Checkbox } from "./checkbox";
import { DataTablePagination } from "./table-pagination";
import { ResponsiveTableProvider, useResponsiveTableContext } from "./responsive-table";

// Enhanced column definition with responsive features
export interface ResponsiveColumnDef<TData, TValue = unknown> {
  id: string;
  accessorKey?: keyof TData;
  accessorFn?: AccessorFn<TData, TValue>;
  header?: string | ColumnDefTemplate<HeaderContext<TData, TValue>>;
  cell?: ColumnDefTemplate<CellContext<TData, TValue>>;
  footer?: ColumnDefTemplate<HeaderContext<TData, TValue>>;
  enableSorting?: boolean;
  enableHiding?: boolean;
  enableResizing?: boolean;
  size?: number;
  minSize?: number;
  maxSize?: number;
  meta?: ColumnMeta<TData, TValue>;
  responsive?: boolean;
  resizable?: boolean;
  sortable?: boolean;
  className?: string;
  cellClassName?: string;
}

// Table configuration interface
export interface ResponsiveTableConfig<TData> {
  data: TData[];
  columns: ResponsiveColumnDef<TData, any>[];
  features?: {
    selection?: boolean;
    dragDrop?: boolean;
    expandable?: boolean;
    pagination?: boolean;
    sorting?: boolean;
    resizing?: boolean;
  };
  pagination?: {
    pageSize?: number;
    pageSizeOptions?: number[];
    alwaysVisible?: boolean;
  };
  // Selection props for external control
  selectedRowIds?: string[];
  onRowSelectionChange?: (selectedIds: string[]) => void;
  onSelectAll?: () => void;
  rowClassName?: (row: TData) => string | undefined;
  // Expansion props for external control
  expandedRowIds?: string[];
  onExpandedRowsChange?: (expandedIds: string[]) => void;
  onRowClick?: (row: TData) => void;
  renderExpandedRow?: (row: Row<TData>) => React.ReactNode;
  getRowId?: (row: TData) => string;
  className?: string;
  // Pagination props for external control (1-based page index)
  pageIndex?: number;
  onPageIndexChange?: (page: number) => void;
  // Server-side pagination: when provided, skips client-side slicing and uses this as total
  serverTotalItems?: number;
}

// Reusable column resize handle
export function ColumnResizeHandle() {
  const [isResizing, setIsResizing] = React.useState(false);

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    setIsResizing(true);
    const startX = e.clientX;
    const header = (e.target as HTMLElement).closest("th");
    if (!header) return;

    const startWidth = header.offsetWidth;

    const handleMouseMove = (e: MouseEvent) => {
      const diff = e.clientX - startX;
      const newWidth = Math.max(30, startWidth + diff);
      header.style.width = `${newWidth}px`;
    };

    const handleMouseUp = () => {
      setIsResizing(false);
      document.removeEventListener("mousemove", handleMouseMove);
      document.removeEventListener("mouseup", handleMouseUp);
      document.body.style.cursor = "";
      document.body.style.userSelect = "";
    };

    document.body.style.cursor = "col-resize";
    document.body.style.userSelect = "none";

    document.addEventListener("mousemove", handleMouseMove);
    document.addEventListener("mouseup", handleMouseUp);
  };

  return (
    <div
      onMouseDown={handleMouseDown}
      data-ui-part="table-header-resize-handle"
      className={cn(
        "absolute right-0 top-0 z-20 flex h-full w-2 cursor-col-resize items-center justify-center",
        "transition-all duration-[var(--motion-duration-fast)] ease-[var(--motion-easing-standard)]",
        isResizing && "bg-[var(--component-table-row-selected)]"
      )}
      title="Drag to resize column"
    >
      <div
        data-ui-part="table-header-resize-line"
        className={cn(
          "h-6 w-0.5 bg-[var(--component-table-border)] transition-all duration-[var(--motion-duration-fast)] ease-[var(--motion-easing-standard)]",
          isResizing && "bg-[var(--color-primary-strong)]"
        )}
      />
    </div>
  );
}

// Reusable responsive table cell
export function ResponsiveTableCell({
  children,
  className,
  ...props
}: React.ComponentProps<typeof TableCell>) {
  return (
    <TableCell className={cn("px-3 sm:px-4 min-w-0", className)} {...props}>
      {children}
    </TableCell>
  );
}

// Reusable responsive table head
export function ResponsiveTableHead({
  children,
  resizable = false,
  className,
  ...props
}: React.ComponentProps<typeof TableHead> & { resizable?: boolean }) {
  return (
    <TableHead
      className={cn("px-3 sm:px-4 font-semibold relative min-w-0", className)}
      {...props}
    >
      <span className="truncate">{children}</span>
      {resizable && <ColumnResizeHandle />}
    </TableHead>
  );
}

// Draggable row component
interface DraggableRowProps<TData> {
  row: Row<TData>;
  children: React.ReactNode;
  isDragDisabled?: boolean;
  onRowClick?: (row: TData) => void;
  interactive?: boolean;
  isExpanded?: boolean;
  rowClassName?: (row: TData) => string | undefined;
  rowParity?: "odd" | "even";
}

function DraggableRow<TData>({
  row,
  children,
  isDragDisabled,
  onRowClick,
  interactive = false,
  isExpanded = false,
  rowClassName,
  rowParity,
}: DraggableRowProps<TData>) {
  const { attributes, listeners, setNodeRef, transform, transition, isDragging } = useSortable({
    id: row.id,
    disabled: isDragDisabled,
  });

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  };

  const handleRowClick = (e: React.MouseEvent) => {
    // Don't trigger row click if clicking on interactive elements
    const target = e.target as HTMLElement;
    if (
      target.closest('button') ||
      target.closest('a') ||
      target.closest('[role="checkbox"]') ||
      target.closest('.no-row-click')
    ) {
      return;
    }
    onRowClick?.(row.original);
  };

  return (
    <TableRow
      ref={setNodeRef}
      data-row-parity={rowParity}
      data-clickable={interactive ? "true" : "false"}
      data-expanded={isExpanded ? "true" : "false"}
      style={style}
      className={cn(
        "relative group transition-all duration-200",
        interactive && "cursor-pointer hover:bg-muted/30",
        isDragging && "shadow-lg bg-background border-2 border-primary/20",
        rowClassName?.(row.original)
      )}
      onClick={interactive ? handleRowClick : undefined}
    >
      {children}
    </TableRow>
  );
}

// Main responsive data table component
export function ResponsiveDataTable<TData>({
  data,
  columns,
  features = {},
  pagination = {},
  selectedRowIds,
  onRowSelectionChange,
  onSelectAll,
  rowClassName,
  expandedRowIds,
  onExpandedRowsChange,
  onRowClick,
  renderExpandedRow,
  getRowId = (row: any) => row.id || row.toString(),
  className,
  pageIndex,
  onPageIndexChange,
  serverTotalItems,
}: ResponsiveTableConfig<TData>) {
  const { visibleColumns } = useResponsiveTableContext();

  const enabledFeatures = {
    selection: false,
    dragDrop: false,
    expandable: false,
    pagination: false,
    sorting: true,
    resizing: true,
    ...features,
  };

  const paginationConfig = {
    pageSize: 10,
    pageSizeOptions: [5, 10, 25, 50, 100],
    ...pagination,
  };

  // Table state
  const [sorting, setSorting] = React.useState<SortingState>([]);
  const [columnFilters, setColumnFilters] = React.useState<ColumnFiltersState>([]);
  const [columnVisibility, setColumnVisibility] = React.useState<VisibilityState>({});
  const [internalRowSelection, setInternalRowSelection] = React.useState({});
  const [currentPage, setCurrentPage] = React.useState(pageIndex ?? 1);
  const [pageSize, setPageSize] = React.useState(paginationConfig.pageSize);

  // Sync external pageIndex → internal when it changes
  React.useEffect(() => {
    if (pageIndex !== undefined) setCurrentPage(pageIndex);
  }, [pageIndex]);
  const [expandedRows, setExpandedRows] = React.useState<Set<string>>(new Set());

  // Use external selection state when provided, otherwise use internal state
  const isExternalSelection = selectedRowIds !== undefined && onRowSelectionChange !== undefined;
  
  // Use external expansion state when provided, otherwise use internal state
  const isExternalExpansion = expandedRowIds !== undefined && onExpandedRowsChange !== undefined;
  
  // Convert expandedRowIds array to Set for easier manipulation
  const externalExpandedRows = React.useMemo(() => {
    return isExternalExpansion && expandedRowIds ? new Set(expandedRowIds) : new Set();
  }, [isExternalExpansion, expandedRowIds]);
  
  // Use external or internal expansion state
  const activeExpandedRows = isExternalExpansion ? externalExpandedRows : expandedRows;

  // Convert selectedRowIds array to row selection object for react-table
  const rowSelection = React.useMemo(() => {
    if (isExternalSelection && selectedRowIds) {
      const selectionObj: Record<string, boolean> = {};
      selectedRowIds.forEach((id) => {
        selectionObj[id] = true;
      });
      return selectionObj;
    }
    return internalRowSelection;
  }, [isExternalSelection, selectedRowIds, internalRowSelection]);

  // Handle row selection changes
  const handleRowSelectionChange = React.useCallback(
    (updaterOrValue: any) => {
      if (isExternalSelection && onRowSelectionChange) {
        // Convert row selection object back to array of IDs
        const newSelection =
          typeof updaterOrValue === "function" ? updaterOrValue(rowSelection) : updaterOrValue;
        const selectedIds = Object.keys(newSelection).filter((id) => newSelection[id]);
        onRowSelectionChange(selectedIds);
      } else {
        setInternalRowSelection(updaterOrValue);
      }
    },
    [isExternalSelection, onRowSelectionChange, rowSelection]
  );

  // Handle row click to toggle expansion
  const handleRowClickInternal = React.useCallback(
    (row: TData) => {
      const rowId = getRowId(row);

      // If expandable is enabled and there's a render function, toggle expansion
      if (enabledFeatures.expandable && renderExpandedRow) {
        if (isExternalExpansion && onExpandedRowsChange) {
          // Handle external expansion state
          const newExpanded = new Set(activeExpandedRows);
          const wasExpanded = newExpanded.has(rowId);
          if (wasExpanded) {
            newExpanded.delete(rowId);
          } else {
            newExpanded.add(rowId);
          }
          onExpandedRowsChange(Array.from(newExpanded) as string[]);
        } else {
          // Handle internal expansion state
          setExpandedRows((prev) => {
            const newSet = new Set(prev);
            const wasExpanded = newSet.has(rowId);
            if (wasExpanded) {
              newSet.delete(rowId);
            } else {
              newSet.add(rowId);
            }
            return newSet;
          });
        }
      }

      // Call the user-provided onRowClick if it exists
      onRowClick?.(row);
    },
    [enabledFeatures.expandable, renderExpandedRow, getRowId, isExternalExpansion,
     onExpandedRowsChange, activeExpandedRows, onRowClick]
  );

  const hasRowInteraction = Boolean(
    (enabledFeatures.expandable && renderExpandedRow) || onRowClick,
  );

  // Enhanced columns with system columns and responsive filtering
  const enhancedColumns = React.useMemo(() => {
    const cols: ResponsiveColumnDef<TData, any>[] = [];

    // Add drag handle column (only if visible and enabled)
    if (enabledFeatures.dragDrop && visibleColumns.dragHandle) {
      cols.push({
        id: "drag-handle",
        header: () => "",
        cell: () => (
          <Button
            variant="ghost"
            size="icon"
            className="admin-icon-btn-subtle text-foreground size-7 hover:bg-muted cursor-grab active:cursor-grabbing opacity-50 group-hover:opacity-100 transition-opacity"
          >
            <IconGripVertical className="text-foreground size-3" />
          </Button>
        ),
        size: 32,
        enableSorting: false,
        enableHiding: false,
      } as ResponsiveColumnDef<TData, any>);
    }

    // Add selection column (only if visible and enabled)
    if (enabledFeatures.selection && visibleColumns.checkbox) {
      cols.push({
        id: "select",
        header: ({ table }: { table: any }) => (
          <Checkbox
            checked={table.getIsAllPageRowsSelected()}
            onCheckedChange={(value) => {
              if (isExternalSelection && onSelectAll) {
                onSelectAll();
              } else {
                table.toggleAllPageRowsSelected(!!value);
              }
            }}
            aria-label="Select all"
          />
        ),
        cell: ({ row }: { row: any }) => (
          <Checkbox
            checked={row.getIsSelected()}
            onCheckedChange={(value) => row.toggleSelected(!!value)}
            aria-label="Select row"
          />
        ),
        size: 48,
        enableSorting: false,
        enableHiding: false,
      } as ResponsiveColumnDef<TData, any>);
    }

    // Filter user columns based on responsive visibility
    const visibleUserColumns = columns.filter((column) => {
      const columnId = column.id;
      // Check if this column should be visible based on responsive settings
      if (columnId && visibleColumns.hasOwnProperty(columnId)) {
        return visibleColumns[columnId as keyof typeof visibleColumns];
      }
      // If not in responsive config, assume it should be visible
      return true;
    });

    cols.push(...visibleUserColumns);

    // Add expand column (only if visible and enabled)
    if (enabledFeatures.expandable && renderExpandedRow && visibleColumns.expand) {
      cols.push({
        id: "expand",
        header: () => "",
        cell: ({ row }: { row: any }) => {
          const rowId = getRowId(row.original);
          const isExpanded = activeExpandedRows.has(rowId);
          return (
            <Button
              variant="ghost"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                if (isExternalExpansion && onExpandedRowsChange) {
                  // Handle external expansion state
                  const newExpanded = new Set(activeExpandedRows);
                  if (newExpanded.has(rowId)) {
                    newExpanded.delete(rowId);
                  } else {
                    newExpanded.add(rowId);
                  }
                  onExpandedRowsChange(Array.from(newExpanded) as string[]);
                } else {
                  // Handle internal expansion state
                  setExpandedRows((prev) => {
                    const newSet = new Set(prev);
                    if (newSet.has(rowId)) {
                      newSet.delete(rowId);
                    } else {
                      newSet.add(rowId);
                    }
                    return newSet;
                  });
                }
              }}
              className="admin-row-icon-btn h-8 w-8 p-0"
            >
              {isExpanded ? (
                <ChevronDown className="h-4 w-4" />
              ) : (
                <ChevronRight className="h-4 w-4" />
              )}
            </Button>
          );
        },
        size: 32,
        enableSorting: false,
        enableHiding: false,
      } as ResponsiveColumnDef<TData, any>);
    }

    return cols;
  }, [columns, enabledFeatures, activeExpandedRows, renderExpandedRow, getRowId, JSON.stringify(visibleColumns), isExternalExpansion, onExpandedRowsChange]);

  // Optimized pagination logic with memoization
  const paginationData = React.useMemo(() => {
    if (!enabledFeatures.pagination) {
      return {
        totalPages: 1,
        startIndex: 0,
        paginatedData: data,
        endIndex: data.length,
        totalItems: data.length,
      };
    }

    if (serverTotalItems !== undefined) {
      // Server-side pagination: data is already the current page's slice
      const totalPages = Math.max(Math.ceil(serverTotalItems / pageSize), 1);
      const startIndex = (currentPage - 1) * pageSize;
      const endIndex = startIndex + data.length;
      return { totalPages, startIndex, paginatedData: data, endIndex, totalItems: serverTotalItems };
    }

    const totalPages = Math.ceil(data.length / pageSize);
    const startIndex = (currentPage - 1) * pageSize;
    const endIndex = Math.min(startIndex + pageSize, data.length);
    const paginatedData = data.slice(startIndex, endIndex);

    return { totalPages, startIndex, paginatedData, endIndex, totalItems: data.length };
  }, [data, pageSize, currentPage, enabledFeatures.pagination, serverTotalItems]);

  const { totalPages, startIndex, paginatedData, endIndex, totalItems } = paginationData;

  React.useEffect(() => {
    if (!enabledFeatures.pagination || serverTotalItems !== undefined) return;
    const safeTotalPages = Math.max(Math.ceil(data.length / pageSize), 1);
    if (currentPage > safeTotalPages) {
      setCurrentPage(safeTotalPages);
    }
  }, [enabledFeatures.pagination, data.length, pageSize, currentPage, serverTotalItems]);

  // React Table instance with optimized configuration
  const table = useReactTable({
    data: paginatedData,
    columns: enhancedColumns as ColumnDef<TData>[],
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: enabledFeatures.sorting ? getSortedRowModel() : undefined,
    getFilteredRowModel: getFilteredRowModel(),
    onSortingChange: setSorting,
    onColumnFiltersChange: setColumnFilters,
    onColumnVisibilityChange: setColumnVisibility,
    onRowSelectionChange: handleRowSelectionChange,
    enableColumnResizing: enabledFeatures.resizing,
    columnResizeMode: "onChange",
    enableSorting: enabledFeatures.sorting,
    state: {
      sorting,
      columnFilters,
      columnVisibility,
      rowSelection,
    },
    getRowId,
    enableRowSelection: enabledFeatures.selection,
  });

  // Optimized responsive behavior with debouncing
  React.useEffect(() => {
    let timeoutId: NodeJS.Timeout;

    const handleResize = () => {
      clearTimeout(timeoutId);
      timeoutId = setTimeout(() => {
        const tableHeaders = document.querySelectorAll('th[style*="width"]');
        tableHeaders.forEach((header) => {
          const element = header as HTMLElement;
          if (window.innerWidth < 768) {
            element.style.width = "auto";
            element.style.minWidth = "auto";
          }
        });
      }, 150); // Debounce resize events
    };

    window.addEventListener("resize", handleResize);
    handleResize();

    return () => {
      window.removeEventListener("resize", handleResize);
      clearTimeout(timeoutId);
    };
  }, []);

  // Drag and drop sensors
  const sensors = useSensors(
    useSensor(PointerSensor),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    })
  );

  const handleDragEnd = (event: DragEndEvent) => {
    const { active, over } = event;

    if (over && active.id !== over.id) {
      // Handle row reordering logic here
      console.log("Reorder:", active.id, "to", over.id);
    }
  };

  return (
    <div
      className={cn("w-full space-y-4", className)}
      data-slot="responsive-data-table"
      data-table-has-row-interaction={hasRowInteraction ? "true" : "false"}
    >
      <div className="relative flex flex-col" data-slot="responsive-data-table-shell">
        <div
          className="overflow-x-auto-hidden scrollbar-thin scrollbar-thumb-gray-300 scrollbar-track-gray-100"
          data-slot="responsive-data-table-scroller"
        >
          <DndContext
            sensors={sensors}
            collisionDetection={closestCenter}
            onDragEnd={handleDragEnd}
          >
            <SortableContext
              items={paginatedData.map((item) => getRowId(item))}
              strategy={verticalListSortingStrategy}
            >
              <Table
                bordered={false}
                className="w-full min-w-0"
                style={{
                  tableLayout: "auto",
                  maxWidth: "100%",
                  width: "100%",
                }}
              >
                <TableHeader className="sticky top-0 z-10 bg-transparent">
                  {table.getHeaderGroups().map((headerGroup) => (
                    <TableRow
                      key={headerGroup.id}
                      className="border-b bg-transparent hover:bg-transparent focus-within:bg-transparent"
                    >
                      {headerGroup.headers.map((header) => {
                        const columnDef = header.column.columnDef as ResponsiveColumnDef<TData>;
                        return (
                          <ResponsiveTableHead
                            key={header.id}
                            resizable={enabledFeatures.resizing && columnDef.resizable}
                            className={columnDef.className}
                          >
                            <div className="flex items-center justify-between">
                              {flexRender(header.column.columnDef.header, header.getContext())}
                              {header.column.getCanSort() && enabledFeatures.sorting && (
                                <Button
                                  variant="ghost"
                                  size="sm"
                                  onClick={() => header.column.toggleSorting()}
                                  className={cn(
                                    "admin-icon-btn-subtle admin-table-header-sort-btn ml-2 h-6 w-6 p-0 rounded-md transition-opacity",
                                    header.column.getIsSorted()
                                      ? "opacity-100"
                                      : "opacity-35 focus-visible:opacity-100"
                                  )}
                                >
                                  {header.column.getIsSorted() === "desc" ? (
                                    <ChevronDown className="h-3 w-3" />
                                  ) : header.column.getIsSorted() === "asc" ? (
                                    <ChevronUp className="h-3 w-3" />
                                  ) : (
                                    <ChevronUp className="h-3 w-3 opacity-50" />
                                  )}
                                </Button>
                              )}
                            </div>
                          </ResponsiveTableHead>
                        );
                      })}
                    </TableRow>
                  ))}
                </TableHeader>
                <TableBody>
                  {table.getRowModel().rows?.length ? (
                    table.getRowModel().rows.map((row, rowIndex) => {
                      const rowId = getRowId(row.original);
                      const isExpanded = activeExpandedRows.has(rowId);
                      const rowParity = rowIndex % 2 === 0 ? "odd" : "even";

                      return (
                        <React.Fragment key={row.id}>
                          {enabledFeatures.dragDrop ? (
                            <DraggableRow
                              row={row}
                              onRowClick={handleRowClickInternal}
                              interactive={hasRowInteraction}
                              isExpanded={isExpanded}
                              rowClassName={rowClassName}
                              rowParity={rowParity}
                            >
                              {row.getVisibleCells().map((cell) => {
                                const columnDef = cell.column
                                  .columnDef as ResponsiveColumnDef<TData>;
                                return (
                                  <ResponsiveTableCell
                                    key={cell.id}
                                    className={columnDef.cellClassName}
                                  >
                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                  </ResponsiveTableCell>
                                );
                              })}
                            </DraggableRow>
                          ) : (
                            <TableRow
                              data-row-parity={rowParity}
                              data-clickable={hasRowInteraction ? "true" : "false"}
                              data-expanded={isExpanded ? "true" : "false"}
                              className={cn(
                                "transition-colors",
                                hasRowInteraction && "cursor-pointer",
                                rowClassName?.(row.original)
                              )}
                              onClick={hasRowInteraction ? (e) => {
                                // Don't trigger row click if clicking on interactive elements
                                const target = e.target as HTMLElement;
                                const clickedButton = target.closest('button');
                                const clickedLink = target.closest('a');
                                const clickedCheckbox = target.closest('[role="checkbox"]');
                                const clickedNoClick = target.closest('.no-row-click');
                                const clickedDropdown = target.closest('[data-slot="dropdown-menu"]') ||
                                                       target.closest('[data-slot="dropdown-menu-content"]') ||
                                                       target.closest('[data-slot="dropdown-menu-trigger"]');

                                if (
                                  clickedButton ||
                                  clickedLink ||
                                  clickedCheckbox ||
                                  clickedNoClick ||
                                  clickedDropdown
                                ) {
                                  return;
                                }
                                handleRowClickInternal(row.original);
                              } : undefined}
                            >
                              {row.getVisibleCells().map((cell) => {
                                const columnDef = cell.column
                                  .columnDef as ResponsiveColumnDef<TData>;
                                return (
                                  <ResponsiveTableCell
                                    key={cell.id}
                                    className={columnDef.cellClassName}
                                  >
                                    {flexRender(cell.column.columnDef.cell, cell.getContext())}
                                  </ResponsiveTableCell>
                                );
                              })}
                            </TableRow>
                          )}

                          {isExpanded && renderExpandedRow && (
                            <TableRow
                              data-slot="table-expanded-row"
                              data-parent-row-parity={rowParity}
                            >
                              <TableCell
                                data-slot="table-expanded-cell"
                                colSpan={row.getVisibleCells().length}
                                className="p-0"
                              >
                                <div data-slot="table-expanded-panel">
                                  {renderExpandedRow(row)}
                                </div>
                              </TableCell>
                            </TableRow>
                          )}
                        </React.Fragment>
                      );
                    })
                  ) : (
                    <TableRow>
                      <TableCell colSpan={enhancedColumns.length} className="h-24 text-center">
                        No results.
                      </TableCell>
                    </TableRow>
                  )}
                </TableBody>
              </Table>
            </SortableContext>
          </DndContext>
        </div>

        {enabledFeatures.pagination && (totalPages > 1 || paginationConfig.alwaysVisible) && (
          <DataTablePagination
            currentPage={currentPage}
            totalPages={totalPages}
            pageSize={pageSize}
            totalItems={totalItems}
            startIndex={startIndex}
            endIndex={endIndex}
            onPageChange={(page) => { setCurrentPage(page); onPageIndexChange?.(page); }}
            onPageSizeChange={(size) => {
              setPageSize(size);
              setCurrentPage(1);
            }}
            pageSizeOptions={paginationConfig.pageSizeOptions}
            alwaysVisible={paginationConfig.alwaysVisible}
          />
        )}
      </div>
    </div>
  );
}
