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
  Server,
  Container,
  Box,
  Cpu,
  Copy,
  RefreshCw,
  Eye,
} from "lucide-react";
import type { AdaptiveColumn } from "@/components/ui/adaptive-table";
import type { AgentRecord } from "../../../app/api/workloadsApi";

export interface DisplayAgent {
  id: string;
  spiffeId: string;
  nodeId: string;
  attestationType: string;
  status: string;
  lastSeen: string;
  createdAt: string;
  raw: AgentRecord;
}

// Table action handlers interface
export interface AgentsTableActions {
  onRefresh: (agent: DisplayAgent) => void;
  onViewDetails: (agent: DisplayAgent) => void;
}

// Get icon for attestation type
const getAttestationIcon = (type: string) => {
  const lowerType = type.toLowerCase();
  if (lowerType.includes("k8s") || lowerType.includes("kubernetes"))
    return Container;
  if (lowerType.includes("container") || lowerType.includes("docker"))
    return Box;
  if (lowerType.includes("process") || lowerType.includes("unix")) return Cpu;
  return Server;
};

// Format relative time
const formatRelativeTime = (dateString: string) => {
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffMins < 1) return "Just now";
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays < 7) return `${diffDays}d ago`;

  return date.toLocaleDateString();
};

// Cell: SPIFFE ID
export function AgentSpiffeIdCell({ agent }: { agent: DisplayAgent }) {
  const displayId =
    agent.spiffeId.length > 40
      ? `${agent.spiffeId.substring(0, 40)}...`
      : agent.spiffeId;

  return (
    <div className="flex items-center gap-2 min-w-0">
      <code
        className="text-xs font-mono text-foreground truncate"
        title={agent.spiffeId}
      >
        {displayId}
      </code>
      <CopyButton
        text={agent.spiffeId}
        label="SPIFFE ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Cell: Node ID
export function AgentNodeIdCell({ agent }: { agent: DisplayAgent }) {
  return (
    <div className="flex items-center gap-2">
      <code className="text-xs font-mono bg-slate-100 dark:bg-slate-800 px-2 py-1 rounded text-slate-700 dark:text-slate-300">
        {agent.nodeId}
      </code>
      <CopyButton
        text={agent.nodeId}
        label="Node ID"
        variant="ghost"
        size="sm"
        className="h-5 w-5 p-0 flex-shrink-0"
      />
    </div>
  );
}

// Cell: Status
export function AgentStatusCell({ agent }: { agent: DisplayAgent }) {
  const status = agent.status.toLowerCase();
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

// Cell: Attestation Type
export function AgentAttestationTypeCell({ agent }: { agent: DisplayAgent }) {
  const AttestationIcon = getAttestationIcon(agent.attestationType);
  const displayType = agent.attestationType.toLowerCase();

  return (
    <div className="flex items-center gap-2">
      <AttestationIcon className="w-4 h-4 text-foreground" />
      <Badge variant="outline" className="capitalize">
        {displayType}
      </Badge>
    </div>
  );
}

// Cell: Last Heartbeat
export function AgentLastHeartbeatCell({ agent }: { agent: DisplayAgent }) {
  const relativeTime = formatRelativeTime(agent.lastSeen);

  return (
    <div className="text-sm text-foreground">
      <div className="font-medium">{relativeTime}</div>
      <div className="text-xs text-foreground/70">
        {new Date(agent.lastSeen).toLocaleString()}
      </div>
    </div>
  );
}

// Cell: Created At
export function AgentCreatedAtCell({ agent }: { agent: DisplayAgent }) {
  return (
    <div className="text-sm text-foreground">
      {new Date(agent.createdAt).toLocaleDateString()}
    </div>
  );
}

// Cell: Actions
export function AgentActionsCell({
  agent,
  actions,
}: {
  agent: DisplayAgent;
  actions: AgentsTableActions;
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
        <DropdownMenuItem onClick={() => actions.onViewDetails(agent)}>
          <Eye className="mr-2 h-4 w-4" />
          View Details
        </DropdownMenuItem>
        <DropdownMenuItem onClick={() => actions.onRefresh(agent)}>
          <RefreshCw className="mr-2 h-4 w-4" />
          Refresh Status
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

// Expanded row for agent details
export function AgentExpandedRow({
  agent,
  actions,
}: {
  agent: DisplayAgent;
  actions?: AgentsTableActions;
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
            Agent Information
          </h4>
          <div className="space-y-3 text-sm">
            <InfoLine label="Agent ID" value={agent.id} copyable />
            <InfoLine label="SPIFFE ID" value={agent.spiffeId} copyable />
            <InfoLine label="Node ID" value={agent.nodeId} copyable />
            <InfoLine label="Attestation Type" value={agent.attestationType} />
            <InfoLine label="Status" value={agent.status} />
            <InfoLine label="Created At" value={agent.createdAt} />
            <InfoLine label="Last Seen" value={agent.lastSeen} />
          </div>
        </div>

        {/* Status & Connectivity */}
        <div className="space-y-4">
          <h4 className="text-sm font-semibold text-foreground">
            Status & Connectivity
          </h4>
          <div className="space-y-4 text-sm">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-foreground">
                Current Status
              </span>
              <AgentStatusCell agent={agent} />
            </div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-foreground">
                Last Heartbeat
              </span>
              <span className="text-xs text-foreground">
                {formatRelativeTime(agent.lastSeen)}
              </span>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-foreground">
                Uptime
              </span>
              <span className="text-xs text-foreground">
                {formatRelativeTime(agent.createdAt)}
              </span>
            </div>
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
              variant="outline"
              size="sm"
              onClick={() => actions.onViewDetails(agent)}
              className="flex items-center gap-2"
            >
              <Eye className="h-4 w-4" />
              View Details
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => actions.onRefresh(agent)}
              className="flex items-center gap-2"
            >
              <RefreshCw className="h-4 w-4" />
              Refresh Status
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

// Create columns for AdaptiveTable
export function createAgentsTableColumns(
  actions: AgentsTableActions
): AdaptiveColumn<DisplayAgent>[] {
  return [
    {
      id: "spiffeId",
      header: "SPIFFE ID",
      accessorKey: "spiffeId",
      alwaysVisible: true,
      enableSorting: true,
      resizable: true,
      approxWidth: 300,
      cell: ({ row }) => <AgentSpiffeIdCell agent={row.original} />,
    },
    {
      id: "nodeId",
      header: "Node ID",
      accessorKey: "nodeId",
      priority: 1,
      enableSorting: true,
      resizable: true,
      approxWidth: 200,
      cell: ({ row }) => <AgentNodeIdCell agent={row.original} />,
    },
    {
      id: "status",
      header: "Status",
      accessorKey: "status",
      priority: 2,
      enableSorting: true,
      resizable: true,
      approxWidth: 120,
      cell: ({ row }) => <AgentStatusCell agent={row.original} />,
    },
    {
      id: "attestationType",
      header: "Attestation Type",
      accessorKey: "attestationType",
      priority: 3,
      enableSorting: true,
      resizable: true,
      approxWidth: 150,
      cell: ({ row }) => <AgentAttestationTypeCell agent={row.original} />,
    },
    {
      id: "lastSeen",
      header: "Last Heartbeat",
      accessorKey: "lastSeen",
      priority: 4,
      enableSorting: true,
      resizable: true,
      approxWidth: 180,
      cell: ({ row }) => <AgentLastHeartbeatCell agent={row.original} />,
    },
    {
      id: "createdAt",
      header: "Created At",
      accessorKey: "createdAt",
      priority: 5,
      enableSorting: true,
      resizable: true,
      approxWidth: 180,
      cell: ({ row }) => <AgentCreatedAtCell agent={row.original} />,
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
      approxWidth: 100,
      cell: ({ row }) => (
        <AgentActionsCell agent={row.original} actions={actions} />
      ),
    },
  ];
}

// Transform AgentRecord to DisplayAgent
export function transformAgentRecord(agent: AgentRecord): DisplayAgent {
  return {
    id: agent.id,
    spiffeId: agent.spiffe_id,
    nodeId: agent.node_id,
    attestationType: agent.attestation_type,
    status: agent.status,
    lastSeen: agent.last_seen,
    createdAt: agent.created_at,
    raw: agent,
  };
}
