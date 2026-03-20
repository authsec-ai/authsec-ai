import { useMemo } from "react";
import {
  useListBindingsQuery,
  type RoleBinding,
  type RbacAudience,
} from "@/app/api/bindingsApi";
import { CardContent } from "@/components/ui/card";
import { TableCard } from "@/theme/components/cards";
import { Badge } from "@/components/ui/badge";
import { Loader2 } from "lucide-react";
import {
  ResponsiveDataTable,
  type ResponsiveColumnDef,
  type ResponsiveTableConfig,
} from "@/components/ui/responsive-data-table";
import { ResponsiveTableProvider } from "@/components/ui/responsive-table";
import { MapRoleToScopeModal } from "./MapRoleToScopeModal";
import { formatDistanceToNow } from "date-fns";

interface RoleBindingsTableProps {
  searchQuery: string;
  isMapModalOpen: boolean;
  onMapModalOpenChange: (open: boolean) => void;
  onBindingSuccess?: () => void;
  audience: RbacAudience;
}

export function RoleBindingsTable({
  searchQuery,
  isMapModalOpen,
  onMapModalOpenChange,
  onBindingSuccess,
  audience,
}: RoleBindingsTableProps) {

  // Fetch role bindings with audience
  const { data: bindings = [], isLoading: isLoadingBindings, refetch } = useListBindingsQuery({
    audience,
  });

  // Filter bindings based on search
  const filteredBindings = useMemo(() => {
    if (!searchQuery.trim()) return bindings;

    const query = searchQuery.toLowerCase();
    return bindings.filter(
      (binding) =>
        binding.username?.toLowerCase().includes(query) ||
        binding.email?.toLowerCase().includes(query) ||
        binding.role_name?.toLowerCase().includes(query) ||
        binding.scope_type?.toLowerCase().includes(query) ||
        binding.service_account_id?.toLowerCase().includes(query)
    );
  }, [bindings, searchQuery]);

  const columns: ResponsiveColumnDef<RoleBinding>[] = useMemo(
    () => [
      {
        id: "username",
        header: "User",
        accessorKey: "username",
        cell: ({ row }) => {
          const hasUsername = !!row.original.username;
          const isServiceAccount = !!row.original.service_account_id;

          let displayName = "-";
          if (isServiceAccount) {
            displayName = `SA: ${row.original.service_account_id.substring(0, 12)}...`;
          } else if (hasUsername) {
            displayName = row.original.username;
          } else if (row.original.user_id) {
            displayName = `User ${row.original.user_id.substring(0, 8)}...`;
          }

          return (
            <div className="space-y-0.5">
              <div
                className={`font-medium truncate ${hasUsername || isServiceAccount ? 'text-foreground' : 'text-foreground/60'}`}
                title={row.original.username || row.original.user_id || row.original.service_account_id}
              >
                {displayName}
                {isServiceAccount && (
                  <Badge variant="outline" className="ml-2 text-xs">SDK</Badge>
                )}
              </div>
              {row.original.email && (
                <div className="text-xs text-foreground/70 truncate" title={row.original.email}>
                  {row.original.email}
                </div>
              )}
            </div>
          );
        },
        resizable: true,
        responsive: true,
        cellClassName: "max-w-0",
      },
      {
        id: "role_name",
        header: "Role",
        accessorKey: "role_name",
        cell: ({ row }) => (
          <Badge variant="secondary" className="font-medium">
            {row.original.role_name || "-"}
          </Badge>
        ),
        resizable: true,
        responsive: true,
      },
      {
        id: "scope",
        header: "Scope",
        accessorKey: "scope_type",
        cell: ({ row }) => {
          const hasScope = row.original.scope_type || row.original.scope_id;

          if (!hasScope) {
            return <span className="text-sm text-foreground/50">-</span>;
          }

          return (
            <div className="space-y-1">
              {row.original.scope_type && (
                <Badge variant="outline" className="font-medium">
                  {row.original.scope_type}
                </Badge>
              )}
              {row.original.scope_id && (
                <div className="text-xs text-foreground/60 truncate font-mono" title={row.original.scope_id}>
                  {row.original.scope_id}
                </div>
              )}
            </div>
          );
        },
        resizable: true,
        responsive: true,
      },
      {
        id: "created_at",
        header: "Created",
        accessorKey: "created_at",
        cell: ({ row }) => (
          <div className="text-sm text-foreground">
            {row.original.created_at
              ? formatDistanceToNow(new Date(row.original.created_at), { addSuffix: true })
              : "-"}
          </div>
        ),
        resizable: true,
        responsive: true,
      },
      {
        id: "expires_at",
        header: "Expires",
        accessorKey: "expires_at",
        cell: ({ row }) => {
          if (!row.original.expires_at) {
            return <span className="text-xs font-medium text-foreground">Never</span>;
          }
          const expiresAt = new Date(row.original.expires_at);
          const isExpired = expiresAt < new Date();
          return (
            <div className="space-y-0.5">
              <span className={`inline-flex items-center gap-1.5 text-xs font-medium ${
                isExpired ? "text-red-600 dark:text-red-400" : "text-emerald-600 dark:text-emerald-400"
              }`}>
                <span className={`h-1.5 w-1.5 rounded-full ${isExpired ? "bg-red-500" : "bg-emerald-500"}`} />
                {isExpired ? "Expired" : "Active"}
              </span>
              <div className="text-xs text-foreground">
                {formatDistanceToNow(expiresAt, { addSuffix: true })}
              </div>
            </div>
          );
        },
        resizable: true,
        responsive: true,
      },
    ],
    []
  );

  const tableConfig: ResponsiveTableConfig<RoleBinding> = {
    data: filteredBindings,
    columns,
    features: {
      selection: false,
      dragDrop: false,
      expandable: false,
      pagination: true,
      sorting: true,
      resizing: true,
    },
    pagination: {
      pageSize: 10,
      pageSizeOptions: [5, 10, 25, 50, 100],
      alwaysVisible: true,
    },
    getRowId: (row) => row.id,
  };

  return (
    <ResponsiveTableProvider tableType="roleBindings">
      <div className="role-bindings-table-container">
        <TableCard className="transition-all duration-500">
          <CardContent variant="flush">
            {isLoadingBindings ? (
              <div className="flex items-center justify-center py-16">
                <Loader2 className="h-8 w-8 animate-spin text-foreground/50" />
              </div>
            ) : (
              <ResponsiveDataTable {...tableConfig} />
            )}
          </CardContent>
        </TableCard>
      </div>

      <MapRoleToScopeModal
        open={isMapModalOpen}
        onOpenChange={onMapModalOpenChange}
        onSuccess={() => {
          refetch();
          onBindingSuccess?.();
        }}
        audience={audience}
      />
    </ResponsiveTableProvider>
  );
}
