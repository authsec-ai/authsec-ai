import { useCallback, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CardContent } from "@/components/ui/card";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { DataTableSkeleton } from "@/components/ui/table-skeleton";
import { FilterCard, TableCard } from "@/theme/components/cards";
import {
  AlertTriangle,
  Eye,
  MoreHorizontal,
  Pencil,
  Power,
  Search,
  Trash2,
} from "lucide-react";
import { AdaptiveTable, type AdaptiveColumn } from "@/components/ui/adaptive-table";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { getErrorMessage } from "@/lib/error-utils";
import { toast } from "@/lib/toast";
import {
  useDeleteDelegationPolicyMutation,
  useListDelegationPoliciesQuery,
  useUpdateDelegationPolicyMutation,
} from "@/app/api/trustDelegationApi";
import { TrustDelegationPageFrame } from "./components/TrustDelegationPageFrame";
import { TrustDelegationInfoBanner } from "./components/TrustDelegationInfoBanner";
import type { DelegationPolicyUI } from "./types";
import { getTrustDelegationErrorMessage, humanizePermissionKey } from "./utils";

function PolicyLabelCell({ policy }: { policy: DelegationPolicyUI }) {
  return (
    <div className="flex min-w-0 w-full items-start">
      <div className="min-w-0 flex-1 space-y-1 overflow-hidden">
        <div
          className="truncate text-[14px] font-semibold leading-5 text-foreground"
          title={`${policy.roleName} · ${policy.agentType}`}
        >
          {policy.roleName} · {policy.agentType}
        </div>
        <div
          className="truncate font-mono text-[12.5px] leading-5 text-muted-foreground"
          title={policy.clientLabel}
        >
          {policy.clientLabel}
        </div>
        <div className="truncate text-xs text-muted-foreground">
          {policy.allowedPermissions.length} actions · {policy.maxTtlLabel}
        </div>
      </div>
    </div>
  );
}

function PolicyCountCell({ policy }: { policy: DelegationPolicyUI }) {
  return (
    <span className="text-sm font-medium text-foreground">
      {policy.allowedPermissions.length}
    </span>
  );
}

function PolicyDurationCell({ policy }: { policy: DelegationPolicyUI }) {
  return (
    <span className="text-sm text-foreground">
      {policy.maxTtlLabel}
    </span>
  );
}

function PolicyExpandedRow({ policy }: { policy: DelegationPolicyUI }) {
  const infoRows = [
    { label: "Role", value: policy.roleName },
    { label: "Target Type", value: policy.agentType },
    { label: "Client", value: policy.clientLabel },
    { label: "Maximum Duration", value: policy.maxTtlLabel },
    { label: "Status", value: policy.enabled ? "Enabled" : "Disabled" },
  ];

  return (
    <div className="min-w-0">
      <div className="grid gap-5 md:grid-cols-[minmax(0,1fr)_1px_minmax(0,1fr)]">
        <div className="min-w-0 space-y-3.5">
          <h4 className="text-[14px] font-semibold text-foreground">
            Trust Delegation Details
          </h4>
          <div className="space-y-2.5 text-sm">
            {infoRows.map((item) => (
              <div
                key={item.label}
                className="flex items-center justify-between gap-3"
              >
                <span className="text-[12px] font-medium text-muted-foreground">
                  {item.label}
                </span>
                <span
                  className="truncate text-right text-[13px] text-foreground"
                  title={item.value}
                >
                  {item.value}
                </span>
              </div>
            ))}
          </div>
        </div>

        <div data-slot="table-expanded-divider" className="hidden w-px md:block" />

        <div className="min-w-0 space-y-3.5">
          <h4 className="text-[14px] font-semibold text-foreground">
            Allowed Actions
          </h4>
          <div className="flex flex-wrap gap-1.5">
            {policy.allowedPermissions.length > 0 ? (
              policy.allowedPermissions.map((permission) => (
                <Badge key={permission} variant="outline">
                  {humanizePermissionKey(permission)}
                </Badge>
              ))
            ) : (
              <span className="text-[13px] text-muted-foreground">
                No actions configured
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function PolicyStatusCell({ enabled }: { enabled: boolean }) {
  return (
    <Badge variant={enabled ? "default" : "secondary"}>
      {enabled ? "Enabled" : "Disabled"}
    </Badge>
  );
}

function PolicyActionsCell({
  policy,
  onView,
  onEdit,
  onToggleEnabled,
  onDelete,
}: {
  policy: DelegationPolicyUI;
  onView: (policy: DelegationPolicyUI) => void;
  onEdit: (policy: DelegationPolicyUI) => void;
  onToggleEnabled: (policy: DelegationPolicyUI) => void;
  onDelete: (policy: DelegationPolicyUI) => void;
}) {
  return (
    <div className="flex items-center justify-end">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button variant="ghost" size="sm" className="admin-row-icon-btn h-8 w-8 p-0">
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
          <DropdownMenuContent align="end" visualVariant="row-actions" className="w-44">
          <DropdownMenuItem onClick={() => onView(policy)}>
            <Eye className="mr-2 h-4 w-4" />
            View trust delegation
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onEdit(policy)}>
            <Pencil className="mr-2 h-4 w-4" />
            Edit trust delegation
          </DropdownMenuItem>
          <DropdownMenuItem onClick={() => onToggleEnabled(policy)}>
            <Power className="mr-2 h-4 w-4" />
            {policy.enabled ? "Disable trust delegation" : "Enable trust delegation"}
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={() => onDelete(policy)} className="text-destructive">
            <Trash2 className="mr-2 h-4 w-4" />
            Delete trust delegation
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}

export function TrustDelegationPoliciesPage() {
  const navigate = useNavigate();
  const {
    data: policies = [],
    isFetching,
    error,
  } = useListDelegationPoliciesQuery();
  const [updatePolicy] = useUpdateDelegationPolicyMutation();
  const [deletePolicy] = useDeleteDelegationPolicyMutation();
  const [searchQuery, setSearchQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [roleFilter, setRoleFilter] = useState("all");
  const [targetTypeFilter, setTargetTypeFilter] = useState("all");
  const [clientFilter, setClientFilter] = useState("all");
  const [deleteTarget, setDeleteTarget] = useState<DelegationPolicyUI | null>(null);

  const filteredPolicies = useMemo(() => {
    const normalizedSearch = searchQuery.trim().toLowerCase();

    return policies.filter((policy) => {
      if (statusFilter !== "all") {
        const expectedEnabled = statusFilter === "enabled";
        if (policy.enabled !== expectedEnabled) return false;
      }

      if (roleFilter !== "all" && policy.roleName !== roleFilter) {
        return false;
      }

      if (targetTypeFilter !== "all" && policy.agentType !== targetTypeFilter) {
        return false;
      }

      if (clientFilter !== "all" && policy.clientId !== clientFilter) {
        return false;
      }

      if (!normalizedSearch) return true;

      return [
        policy.roleName,
        policy.agentType,
        policy.clientLabel,
        policy.allowedPermissions.join(" "),
      ]
        .join(" ")
        .toLowerCase()
        .includes(normalizedSearch);
    });
  }, [clientFilter, policies, roleFilter, searchQuery, statusFilter, targetTypeFilter]);

  const roles = Array.from(new Set(policies.map((policy) => policy.roleName)));
  const targetTypes = Array.from(new Set(policies.map((policy) => policy.agentType)));
  const clients = Array.from(
    new Map(
      policies.map((policy) => [
        policy.clientId,
        { id: policy.clientId, label: policy.clientLabel },
      ]),
    ).values(),
  );
  const activeFiltersCount =
    (searchQuery.trim() ? 1 : 0) +
    (statusFilter !== "all" ? 1 : 0) +
    (roleFilter !== "all" ? 1 : 0) +
    (targetTypeFilter !== "all" ? 1 : 0) +
    (clientFilter !== "all" ? 1 : 0);
  const errorMessage = error
    ? getTrustDelegationErrorMessage(
        error,
        "Unable to load trust delegation records right now.",
      )
    : null;
  const showInitialSkeleton =
    isFetching && filteredPolicies.length === 0 && !errorMessage;

  const handleClearFilters = () => {
    setSearchQuery("");
    setStatusFilter("all");
    setRoleFilter("all");
    setTargetTypeFilter("all");
    setClientFilter("all");
  };

  const handleDelete = async () => {
    if (!deleteTarget) return;

    try {
      await deletePolicy(deleteTarget.id).unwrap();
      toast.success("Trust delegation deleted");
      setDeleteTarget(null);
    } catch (requestError) {
      toast.error(
        getErrorMessage(requestError, "Failed to delete trust delegation"),
      );
    }
  };

  const renderExpandedRow = useCallback(
    (row: { original: DelegationPolicyUI }) => (
      <PolicyExpandedRow policy={row.original} />
    ),
    [],
  );

  const columns = useMemo<AdaptiveColumn<DelegationPolicyUI>[]>(() => {
    return [
      {
        id: "policy",
        header: "Trust Delegation",
        accessorFn: (policy) => `${policy.roleName} ${policy.agentType} ${policy.clientLabel}`,
        alwaysVisible: true,
        enableSorting: true,
        resizable: true,
        approxWidth: 320,
        cell: ({ row }) => <PolicyLabelCell policy={row.original} />,
      },
      {
        id: "actionsCount",
        header: "Allowed Actions",
        accessorFn: (policy) => policy.allowedPermissions.length,
        priority: 2,
        enableSorting: true,
        resizable: true,
        approxWidth: 130,
        cell: ({ row }) => <PolicyCountCell policy={row.original} />,
      },
      {
        id: "duration",
        header: "Max Duration",
        accessorKey: "maxTtlLabel",
        priority: 3,
        enableSorting: true,
        resizable: true,
        approxWidth: 150,
        cell: ({ row }) => <PolicyDurationCell policy={row.original} />,
      },
      {
        id: "status",
        header: "Status",
        accessorFn: (policy) => (policy.enabled ? 1 : 0),
        priority: 1,
        enableSorting: true,
        resizable: true,
        approxWidth: 140,
        cell: ({ row }) => <PolicyStatusCell enabled={row.original.enabled} />,
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
          <PolicyActionsCell
            policy={row.original}
            onView={(policy) => navigate(`/trust-delegation/${policy.id}`)}
            onEdit={(policy) => navigate(`/trust-delegation/${policy.id}/edit`)}
            onToggleEnabled={(policy) => {
              void (async () => {
                try {
                  await updatePolicy({
                    id: policy.id,
                    body: {
                      role_name: policy.roleName,
                      agent_type: policy.agentType,
                      allowed_permissions: policy.allowedPermissions,
                      max_ttl_seconds: policy.maxTtlSeconds,
                      enabled: !policy.enabled,
                      client_id: policy.clientId,
                    },
                  }).unwrap();

                  toast.success(
                    policy.enabled
                      ? "Trust delegation disabled"
                      : "Trust delegation enabled",
                  );
                } catch (requestError) {
                  toast.error(
                    getErrorMessage(
                      requestError,
                      "Failed to update trust delegation",
                    ),
                  );
                }
              })();
            }}
            onDelete={(policy) => setDeleteTarget(policy)}
          />
        ),
      },
    ];
  }, [navigate, updatePolicy]);

  return (
    <TrustDelegationPageFrame
      title="Trust Delegation"
      description="Manage saved trust delegation configurations for agents, workloads, and user-linked service identities."
      actions={
        <Button onClick={() => navigate("/trust-delegation/new")}>
          Create trust delegation
        </Button>
      }
    >
      <TrustDelegationInfoBanner />

      <FilterCard>
        <CardContent variant="compact">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
            <div className="flex shrink-0 items-center gap-2">
              <span className="text-sm font-medium text-foreground">Filters</span>
              {activeFiltersCount > 0 && (
                <span className="rounded bg-black/5 px-1.5 py-0.5 text-xs text-foreground dark:bg-white/10">
                  {activeFiltersCount}
                </span>
              )}
            </div>

            <div className="flex w-full flex-1 flex-wrap items-center gap-2">
              <div className="relative min-w-[240px] flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground/50" />
                <Input
                  className="h-9 pl-9 text-sm"
                  placeholder="Search trust delegations..."
                  value={searchQuery}
                  onChange={(event) => setSearchQuery(event.target.value)}
                />
              </div>

              <Select value={roleFilter} onValueChange={setRoleFilter}>
                <SelectTrigger className="h-9 w-[132px] text-sm">
                  <SelectValue placeholder="Role" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Roles</SelectItem>
                  {roles.map((role) => (
                    <SelectItem key={role} value={role}>
                      {role}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="h-9 w-[132px] text-sm">
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Status</SelectItem>
                  <SelectItem value="enabled">Enabled</SelectItem>
                  <SelectItem value="disabled">Disabled</SelectItem>
                </SelectContent>
              </Select>

              <Select value={targetTypeFilter} onValueChange={setTargetTypeFilter}>
                <SelectTrigger className="h-9 w-[148px] text-sm">
                <SelectValue placeholder="Target type" />
              </SelectTrigger>
              <SelectContent>
                  <SelectItem value="all">All Target Types</SelectItem>
                  {targetTypes.map((agentType) => (
                    <SelectItem key={agentType} value={agentType}>
                      {agentType}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={clientFilter} onValueChange={setClientFilter}>
                <SelectTrigger className="h-9 w-[168px] text-sm">
                <SelectValue placeholder="Client" />
              </SelectTrigger>
              <SelectContent>
                  <SelectItem value="all">All Clients</SelectItem>
                  {clients.map((client) => (
                    <SelectItem key={client.id} value={client.id}>
                      {client.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              {activeFiltersCount > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={handleClearFilters}
                  className="h-9 text-sm text-foreground hover:text-foreground"
                >
                  Clear
                </Button>
              )}
            </div>
          </div>
        </CardContent>
      </FilterCard>

      {deleteTarget && (
        <Alert variant="destructive">
          <AlertDescription className="flex flex-wrap items-center justify-between gap-2">
            Deleting this trust delegation stops future issuance but keeps historical audit records.
            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setDeleteTarget(null)}>
                Cancel
              </Button>
              <Button variant="destructive" onClick={handleDelete}>
                Confirm delete
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      <TableCard className="transition-all duration-500">
        <CardContent variant="flush">
          {errorMessage ? (
            <div className="flex flex-col items-center justify-center space-y-4 p-12">
              <div className="rounded-full bg-red-50 p-4 dark:bg-red-950/20">
                <AlertTriangle className="h-8 w-8 text-red-600 dark:text-red-400" />
              </div>
              <div className="space-y-1 text-center">
                <h3 className="text-base font-semibold text-red-900 dark:text-red-100">
                  Unable to Load Trust Delegation
                </h3>
                <p className="text-red-700 dark:text-red-300">{errorMessage}</p>
              </div>
            </div>
          ) : showInitialSkeleton ? (
            <DataTableSkeleton rows={8} />
          ) : (
            <AdaptiveTable
              tableId="trust-delegations-policies"
              data={filteredPolicies}
              columns={columns}
              enableSelection={false}
              enableExpansion
              renderExpandedRow={renderExpandedRow}
              getRowId={(policy) => policy.id}
              enableSorting
              enablePagination
              pagination={{
                pageSize: 10,
                pageSizeOptions: [5, 10, 25, 50],
                alwaysVisible: true,
              }}
            />
          )}
        </CardContent>
      </TableCard>
    </TrustDelegationPageFrame>
  );
}
