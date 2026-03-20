import * as React from "react";
import type { Row } from "@tanstack/react-table";
import {
  AdaptiveTable,
  type AdaptiveColumn,
} from "@/components/ui/adaptive-table";
import {
  createWorkloadsTableColumns,
  createEntriesTableColumns,
  WorkloadExpandedRow,
  type DisplayWorkload,
  type WorkloadsTableActions,
} from "../utils/workloads-table-utils";

interface WorkloadsTableProps {
  workloads: DisplayWorkload[];
  selectedWorkloadIds?: string[];
  onSelectionChange?: (selectedIds: string[]) => void;
  onSelectAll?: () => void;
  actions: WorkloadsTableActions;
  useEntriesColumns?: boolean;
}

export function WorkloadsTable({
  workloads,
  selectedWorkloadIds = [],
  onSelectionChange,
  onSelectAll,
  actions,
  useEntriesColumns = false,
}: WorkloadsTableProps) {
  const columns = React.useMemo<AdaptiveColumn<DisplayWorkload>[]>(() => {
    return useEntriesColumns
      ? createEntriesTableColumns(actions)
      : createWorkloadsTableColumns(actions);
  }, [actions, useEntriesColumns]);

  const renderExpandedRow = React.useCallback(
    (row: Row<DisplayWorkload>) => (
      <WorkloadExpandedRow workload={row.original} actions={actions} />
    ),
    [actions]
  );

  return (
    <AdaptiveTable
      tableId="workloads"
      data={workloads}
      columns={columns}
      enableSelection
      selectedRowIds={selectedWorkloadIds}
      onRowSelectionChange={onSelectionChange}
      onSelectAll={onSelectAll}
      enableExpansion={true}
      renderExpandedRow={renderExpandedRow}
      getRowId={(workload) => workload.id}
      enableSorting
      enablePagination
      pagination={{
        pageSize: 10,
        pageSizeOptions: [5, 10, 25, 50],
        alwaysVisible: true,
      }}
    />
  );
}
