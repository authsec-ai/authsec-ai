import * as React from "react";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { CopyButton } from "../../../components/ui/copy-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Trash2,
  Server,
  Container,
  Box,
  Cpu,
  Copy,
  Edit,
} from "lucide-react";
import type { AdaptiveColumn } from "@/components/ui/adaptive-table";

export interface DisplayWorkload {
  id: string;
  spiffeId: string;
  type: string;
  selectors: string[];
  status: string;
  createdAt: string;
  parentId?: string;
  ttl?: number;
  admin?: boolean;
  downstream?: boolean;
  raw: any;
}

// Table action handlers interface
export interface WorkloadsTableActions {
  onEdit: (workload: DisplayWorkload) => void;
  onDelete: (workload: DisplayWorkload) => void;
}

// Get icon for workload type
const getTypeIcon = (type: string) => {
  const lowerType = type.toLowerCase();
  if (lowerType.includes("k8s") || lowerType.includes("kubernetes"))
    return Container;
  if (lowerType.includes("container") || lowerType.includes("docker"))
    return Box;
  if (lowerType.includes("process") || lowerType.includes("unix")) return Cpu;
  return Server;
};

// Truncate ID for display (show first 8 chars)
const truncateId = (id: string, length: number = 8): string => {
  if (id.length <= length) return id;
  return `${id.substring(0, length)}...`;
};

// Truncate SPIFFE ID path for display (show last part)
const truncateSpiffeId = (spiffeId: string): string => {
  const parts = spiffeId.split("/");
  if (parts.length <= 3) return spiffeId;
  return `.../${parts.slice(-2).join("/")}`;
};

// Cell: Workload ID
export function WorkloadIdCell({ workload }: { workload: DisplayWorkload }) {
  return (
    <div className="flex items-center gap-2 min-w-0">
      <code
        className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300 truncate block"
        title={workload.id}
      >
        {truncateId(workload.id, 12)}
      </code>
      <CopyButton
        text={workload.id}
        label="Workload ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Cell: SPIFFE ID
export function SpiffeIdCell({ workload }: { workload: DisplayWorkload }) {
  return (
    <div className="flex items-center gap-2 min-w-0">
      <code
        className="text-xs font-mono text-foreground truncate block"
        title={workload.spiffeId}
      >
        {truncateSpiffeId(workload.spiffeId)}
      </code>
      <CopyButton
        text={workload.spiffeId}
        label="SPIFFE ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Cell: Type with icon
export function WorkloadTypeCell({ workload }: { workload: DisplayWorkload }) {
  const TypeIcon = getTypeIcon(workload.type);
  const displayType = workload.type.toLowerCase();

  return (
    <div className="flex items-center gap-2">
      <TypeIcon className="w-4 h-4 text-foreground" />
      <Badge variant="outline" className="capitalize">
        {displayType}
      </Badge>
    </div>
  );
}

// Cell: Selectors
export function SelectorsCell({ workload }: { workload: DisplayWorkload }) {
  if (workload.selectors.length === 0) {
    return <span className="text-sm text-foreground">—</span>;
  }

  return (
    <div className="flex flex-wrap items-center gap-2 min-w-0 text-sm text-foreground">
      {workload.selectors.map((selector, idx) => (
        <span
          key={`${workload.id}-selector-${idx}`}
          className="font-mono leading-tight"
          title={selector}
        >
          {selector}
        </span>
      ))}
    </div>
  );
}

// Cell: Status
export function WorkloadStatusCell({
  workload,
}: {
  workload: DisplayWorkload;
}) {
  const status = workload.status.toLowerCase();
  const variant =
    status === "active"
      ? "default"
      : status === "inactive"
      ? "secondary"
      : "outline";

  return (
    <Badge variant={variant} className="capitalize">
      {status}
    </Badge>
  );
}

// Cell: Created Date
export function CreatedAtCell({ workload }: { workload: DisplayWorkload }) {
  return (
    <span className="text-sm text-foreground">{workload.createdAt}</span>
  );
}

// Cell: Parent ID
export function ParentIdCell({ workload }: { workload: DisplayWorkload }) {
  if (!workload.parentId) {
    return <span className="text-sm text-foreground">—</span>;
  }

  return (
    <div className="flex items-center gap-2 min-w-0">
      <code
        className="text-xs font-mono text-foreground truncate block"
        title={workload.parentId}
      >
        {truncateId(workload.parentId, 12)}
      </code>
      <CopyButton
        text={workload.parentId}
        label="Parent ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Cell: TTL
export function TtlCell({ workload }: { workload: DisplayWorkload }) {
  if (workload.ttl === undefined) {
    return <span className="text-sm text-foreground">—</span>;
  }

  return <span className="text-sm text-foreground">{workload.ttl}s</span>;
}

// Cell: Flags (Admin/Downstream)
export function FlagsCell({ workload }: { workload: DisplayWorkload }) {
  const flags = [];
  if (workload.admin) {
    flags.push("Admin");
  }
  if (workload.downstream) {
    flags.push("Downstream");
  }

  if (flags.length === 0) {
    return <span className="text-sm text-foreground">—</span>;
  }

  return (
    <div className="flex gap-1">
      {flags.map((flag) => (
        <Badge key={flag} variant="secondary" className="text-xs">
          {flag}
        </Badge>
      ))}
    </div>
  );
}

// Cell: Actions - Dropdown menu
export function WorkloadActionsCell({
  workload,
  actions,
}: {
  workload: DisplayWorkload;
  actions: WorkloadsTableActions;
}) {
  return (
    <div className="flex items-center justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" visualVariant="row-actions">
          {/* Edit Action */}
          <DropdownMenuItem onClick={() => actions.onEdit(workload)}>
            <Edit className="mr-2 h-4 w-4" />
            Edit Workload
          </DropdownMenuItem>
          {/* Delete Action */}
          <DropdownMenuItem
            onClick={() => actions.onDelete(workload)}
            className="text-destructive"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            Delete Workload
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

// Cell: Actions - Inline buttons
export function WorkloadInlineActionsCell({
  workload,
  actions,
}: {
  workload: DisplayWorkload;
  actions: WorkloadsTableActions;
}) {
  return (
    <div className="flex items-center gap-2">
      <Button
        variant="outline"
        size="sm"
        onClick={() => actions.onEdit(workload)}
      >
        <Edit className="h-4 w-4 mr-1" />
        Edit
      </Button>
      <Button
        variant="outline"
        size="sm"
        onClick={() => actions.onDelete(workload)}
        className="text-red-600 hover:text-red-700 hover:bg-red-50"
      >
        <Trash2 className="h-4 w-4 mr-1" />
        Delete
      </Button>
    </div>
  );
}

// Expanded row for workload details
export function WorkloadExpandedRow({
  workload,
  actions,
}: {
  workload: DisplayWorkload;
  actions?: WorkloadsTableActions;
}) {
  const InfoLine = ({
    label,
    value,
    copyable = false,
  }: {
    label: string;
    value?: string | number | null;
    copyable?: boolean;
  }) => {
    if (value === undefined || value === null || value === "") {
      return null;
    }

    const handleCopy = () => {
      navigator.clipboard.writeText(String(value));
    };

    return (
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-foreground">
          {label}
        </span>
        <div className="flex items-center gap-2 min-w-0">
          <span
            className="font-mono text-xs truncate text-foreground"
            title={String(value)}
          >
            {String(value)}
          </span>
          {copyable && (
            <Copy
              className="h-4 w-4 cursor-pointer text-foreground hover:text-foreground transition-colors flex-shrink-0"
              onClick={handleCopy}
            />
          )}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-5">
      <div className="grid gap-6 md:grid-cols-2">
        {/* Basic Information */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Server className="h-4 w-4" />
            Basic Information
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Workload ID" value={workload.id} copyable />
            <InfoLine label="SPIFFE ID" value={workload.spiffeId} copyable />
            <InfoLine label="Type" value={workload.type} />
            <InfoLine label="Status" value={workload.status} />
            <InfoLine label="Created At" value={workload.createdAt} />
          </div>
        </div>

        {/* Selectors */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-foreground">
            Selectors
          </h4>
          <div className="space-y-4 text-sm">
            {workload.selectors.length > 0 ? (
              <div className="flex flex-wrap gap-2 text-foreground">
                {workload.selectors.map((selector, idx) => (
                  <span
                    key={`expanded-${workload.id}-selector-${idx}`}
                    className="text-xs font-mono leading-tight"
                    title={selector}
                  >
                    {selector}
                  </span>
                ))}
              </div>
            ) : (
              <span className="text-sm text-foreground">
                No selectors configured
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Quick Actions */}
      {actions && (
        <div className="space-y-3 border-t border-border pt-4">
          <h4 className="text-sm font-semibold text-foreground">
            Quick Actions
          </h4>
          <div className="flex flex-wrap gap-3">
            <Button
              variant="destructive"
              size="sm"
              onClick={() => actions.onDelete(workload)}
              className="flex items-center gap-2"
            >
              <Trash2 className="h-4 w-4" />
              Delete Workload
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// Create columns for AdaptiveTable
export function createWorkloadsTableColumns(
  actions: WorkloadsTableActions
): AdaptiveColumn<DisplayWorkload>[] {
  return [
    {
      id: "spiffeId",
      header: "SPIFFE ID",
      accessorKey: "spiffeId",
      alwaysVisible: true,
      enableSorting: true,
      resizable: true,
      approxWidth: 180,
      cell: ({ row }) => <SpiffeIdCell workload={row.original} />,
    },
    {
      id: "type",
      header: "Type",
      accessorKey: "type",
      priority: 1,
      enableSorting: true,
      resizable: true,
      approxWidth: 140,
      cell: ({ row }) => <WorkloadTypeCell workload={row.original} />,
    },
    {
      id: "selectors",
      header: "Selectors",
      accessorKey: "selectors",
      priority: 2,
      enableSorting: false,
      resizable: true,
      approxWidth: 150,
      cell: ({ row }) => <SelectorsCell workload={row.original} />,
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      priority: 3,
      enableSorting: true,
      resizable: true,
      approxWidth: 100,
      cell: ({ row }) => <WorkloadStatusCell workload={row.original} />,
    },
    {
      id: "id",
      header: "Workload ID",
      accessorKey: "id",
      priority: 4,
      enableSorting: true,
      resizable: true,
      approxWidth: 140,
      cell: ({ row }) => <WorkloadIdCell workload={row.original} />,
    },
    {
      id: "createdAt",
      header: "Created",
      accessorKey: "createdAt",
      priority: 5,
      enableSorting: true,
      resizable: true,
      approxWidth: 120,
      cell: ({ row }) => <CreatedAtCell workload={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      alwaysVisible: true,
      enableSorting: false,
      resizable: false,
      size: 80,
      className: "w-[80px] text-right",
      cellClassName: "text-right",
      approxWidth: 80,
      cell: ({ row }) => (
        <WorkloadActionsCell workload={row.original} actions={actions} />
      ),
    },
  ];
}

// Create columns for entries table (similar to WorkloadIdentitiesPage)
export function createEntriesTableColumns(
  actions: WorkloadsTableActions
): AdaptiveColumn<DisplayWorkload>[] {
  return [
    {
      id: "spiffeId",
      header: "SPIFFE ID",
      accessorKey: "spiffeId",
      alwaysVisible: true,
      enableSorting: true,
      resizable: true,
      approxWidth: 180,
      cell: ({ row }) => <SpiffeIdCell workload={row.original} />,
    },
    {
      id: "parentId",
      header: "Parent ID",
      accessorKey: "parentId",
      priority: 1,
      enableSorting: true,
      resizable: true,
      approxWidth: 140,
      cell: ({ row }) => <ParentIdCell workload={row.original} />,
    },
    {
      id: "selectors",
      header: "Selectors",
      accessorKey: "selectors",
      priority: 2,
      enableSorting: false,
      resizable: true,
      approxWidth: 150,
      cell: ({ row }) => <SelectorsCell workload={row.original} />,
    },
    {
      id: "ttl",
      header: "TTL",
      accessorKey: "ttl",
      priority: 3,
      enableSorting: true,
      resizable: true,
      approxWidth: 80,
      cell: ({ row }) => <TtlCell workload={row.original} />,
    },
    {
      id: "createdAt",
      header: "Created",
      accessorKey: "createdAt",
      priority: 5,
      enableSorting: true,
      resizable: true,
      approxWidth: 120,
      cell: ({ row }) => <CreatedAtCell workload={row.original} />,
    },
    {
      id: "actions",
      header: "Actions",
      alwaysVisible: true,
      enableSorting: false,
      resizable: false,
      size: 80,
      className: "w-[80px] text-right",
      cellClassName: "text-right",
      approxWidth: 80,
      cell: ({ row }) => (
        <WorkloadActionsCell workload={row.original} actions={actions} />
      ),
    },
  ];
}
