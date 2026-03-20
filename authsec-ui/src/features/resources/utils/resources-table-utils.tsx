import * as React from "react";
import { Badge } from "../../../components/ui/badge";
import { Button } from "../../../components/ui/button";
import { CopyButton } from "../../../components/ui/copy-button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  MoreHorizontal,
  Edit,
  Trash2,
  Code,
  Code2,
  Database,
  Globe,
  Copy,
  Calendar,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { Resource } from "../types";
import type { ResponsiveColumnDef } from "../../../components/ui/responsive-data-table";

// Utility functions for resources
export const ResourcesTableUtils = {
  formatDate: (timestamp: string) => {
    if (!timestamp || timestamp === "0001-01-01T00:00:00Z") return "N/A";
    const now = new Date();
    const time = new Date(timestamp);
    const diffInDays = Math.floor((now.getTime() - time.getTime()) / (1000 * 60 * 60 * 24));

    if (diffInDays === 0) return "Today";
    if (diffInDays === 1) return "Yesterday";
    if (diffInDays < 7) return `${diffInDays} days ago`;
    return time.toLocaleDateString();
  },
};

// Table action handlers interface
export interface ResourcesTableActions {
  onEditResource: (resource: Resource) => void;
  onDeleteResource: (resource: Resource) => void;
  onViewSDK?: (resource: Resource) => void;
}

// Cell: Resource overview (name and description)
export function ResourceNameCell({ resource }: { resource: Resource }) {
  return (
    <div className="space-y-1">
      <div className="flex items-start gap-3 min-w-0">
        <div className="flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-xl bg-primary/10 text-primary">
          <Database className="h-4 w-4" />
        </div>
        <div className="min-w-0 space-y-1">
          <div className="truncate font-medium text-foreground" title={resource.name}>
            {resource.name}
          </div>
          <p className="text-xs text-foreground line-clamp-1" title={resource.description}>
            {resource.description || "No description provided"}
          </p>
        </div>
      </div>
    </div>
  );
}

// Cell: Actions Menu
export function ActionsCell({
  resource,
  actions
}: {
  resource: Resource;
  actions: ResourcesTableActions;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
          <MoreHorizontal className="h-4 w-4" />
          <span className="sr-only">Open menu</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" visualVariant="row-actions" className="w-48">
        {actions.onViewSDK && (
          <DropdownMenuItem
            onClick={() => actions.onViewSDK?.(resource)}
            className="admin-menu-item-sdk"
          >
            <Code2 className="mr-2 h-4 w-4" />
            View SDK Code
          </DropdownMenuItem>
        )}
        <DropdownMenuItem onClick={() => actions.onEditResource(resource)}>
          <Edit className="mr-2 h-4 w-4" />
          Edit Resource
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          onClick={() => actions.onDeleteResource(resource)}
          className="text-destructive focus:text-destructive"
        >
          <Trash2 className="mr-2 h-4 w-4" />
          Delete Resource
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// Generate columns for the resources table
export function createResourcesColumns(actions: ResourcesTableActions): ResponsiveColumnDef<Resource, any>[] {
  return [
    {
      id: "resource",
      accessorKey: "name",
      header: "Resource",
      cell: ({ row }) => <ResourceNameCell resource={row.original} />,
      enableSorting: true,
      responsive: true,
      className: "min-w-[260px]",
      cellClassName: "max-w-0",
    },
    {
      id: "actions",
      header: "Actions",
      cell: ({ row }) => <ActionsCell resource={row.original} actions={actions} />,
      enableSorting: false,
      enableHiding: false,
      responsive: false,
      className: "w-[80px]",
      cellClassName: "text-center",
    },
  ];
}

// Redesigned expanded row component following ClientsPage pattern
export function ResourceExpandedRow({ resource }: { resource: Resource }) {
  // InfoLine component for horizontal label-value pairs (like ClientsPage)
  const InfoLine = ({ label, value, copyable = false }: { label: string; value?: string | number | null; copyable?: boolean }) => {
    if (value === undefined || value === null || value === "") return null;
    return (
      <div className="flex items-center justify-between gap-3">
        <span className="text-xs font-medium text-foreground">{label}</span>
        <div className="flex items-center gap-2 min-w-0">
          <span className="font-mono text-xs truncate text-foreground" title={String(value)}>{String(value)}</span>
          {copyable && <CopyButton text={String(value)} label={label} size="sm" />}
        </div>
      </div>
    );
  };

  return (
    <div className="space-y-5">
      <div className="grid gap-6 md:grid-cols-2">
        {/* Left Column: Resource Details */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Database className="h-4 w-4" />
            Resource Details
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Resource Name" value={resource.name} />
            <InfoLine label="Resource ID" value={resource.id} copyable />
            <InfoLine label="Description" value={resource.description || "No description"} />
          </div>
        </div>

        {/* Right Column: Metadata */}
        <div className="space-y-4">
          <h4 className="flex items-center gap-2 text-sm font-semibold text-foreground">
            <Calendar className="h-4 w-4" />
            Metadata
          </h4>
          <div className="space-y-3 text-sm">
            {resource.updated_at && resource.updated_at !== "0001-01-01T00:00:00Z" && (
              <InfoLine
                label="Updated At"
                value={new Date(resource.updated_at).toLocaleString()}
              />
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
