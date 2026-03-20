import React, { useState, useEffect, useRef, useMemo } from "react";
import { CardContent } from "@/components/ui/card";
import { FilterCard } from "@/theme/components/cards";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import {
  Search,
  Globe,
  Building2,
  Settings
} from "lucide-react";
import type { UsersQueryParams } from "@/app/api/enduser/usersApi";
import { useGetAllClientsQuery, type GetClientsRequest } from "@/app/api/clientApi";
import { resolveTenantId } from "@/utils/workspace";

// Provider options - static for better reliability
const DEFAULT_PROVIDER_OPTIONS = [
  { value: "google", label: "Google", icon: Globe },
  { value: "github", label: "GitHub", icon: Globe },
  { value: "custom", label: "Custom", icon: Globe },
  { value: "ad_sync", label: "AD Sync", icon: Building2 },
  { value: "entra_id", label: "Entra ID", icon: Globe },
  { value: "local", label: "Local", icon: Settings },
];

// Status options
const STATUS_OPTIONS = [
  { value: "all", label: "All Status" },
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "blocked", label: "Blocked" },
];

interface UserData {
  provider?: string;
  [key: string]: any;
}

export interface UsersFilterCardProps {
  onFiltersChange: (filters: Partial<UsersQueryParams>) => void;
  initialFilters?: Partial<UsersQueryParams>;
  usersData?: UserData[];
}

const UsersFilterCard = React.memo(({
  onFiltersChange,
  initialFilters = {},
  usersData: _usersData = []
}: UsersFilterCardProps) => {
  const providerOptions = DEFAULT_PROVIDER_OPTIONS;
  const [searchQuery, setSearchQuery] = useState(initialFilters.searchQuery || "");
  const [statusFilter, setStatusFilter] = useState<string>(
    initialFilters.active === true ? "active" :
    initialFilters.active === false ? "inactive" : "all"
  );
  const [providerFilter, setProviderFilter] = useState<string[]>(
    initialFilters.provider ? [initialFilters.provider] : []
  );

  // Client filter state
  const tenantId = resolveTenantId();
  const [selectedClientId, setSelectedClientId] = useState<string>(
    initialFilters.client_id || ""
  );

  const clientsQueryArgs = useMemo(() => {
    if (!tenantId) return undefined;
    return {
      tenant_id: tenantId,
      active_only: false,
      filters: {},
    } as GetClientsRequest;
  }, [tenantId]);

  // Fetch clients using query hook (same endpoint as Clients page)
  const { data: clientsData, isLoading: clientsLoading } = useGetAllClientsQuery(
    clientsQueryArgs as GetClientsRequest,
    {
      skip: !clientsQueryArgs,
      refetchOnMountOrArgChange: true,
      refetchOnFocus: false,
      refetchOnReconnect: false,
    }
  );

  // Extract clients from response
  const clients = useMemo(() => {
    if (clientsData?.clients) {
      return Array.isArray(clientsData.clients) ? clientsData.clients : [];
    }
    return [];
  }, [clientsData]);

  // No default client selection; filter is optional

  // Track last emitted filters to avoid emitting duplicates and loops
  const lastEmittedFiltersRef = useRef<string>("");

  // Normalize incoming initialFilters for stable comparisons
  const normalizedInitialFilters = useMemo(() => {
    const f: Partial<UsersQueryParams> = {};
    if (initialFilters.searchQuery) f.searchQuery = initialFilters.searchQuery;
    if (initialFilters.active !== undefined) f.active = initialFilters.active;
    if (initialFilters.provider) f.provider = initialFilters.provider;
    if (initialFilters.client_id) f.client_id = initialFilters.client_id;
    return f;
  }, [
    initialFilters.searchQuery,
    initialFilters.active,
    initialFilters.provider,
    initialFilters.client_id,
  ]);

  // Sync local filter controls when initialFilters prop changes
  useEffect(() => {
    setSearchQuery(initialFilters.searchQuery || "");
    setStatusFilter(
      initialFilters.active === true ? "active" :
      initialFilters.active === false ? "inactive" : "all"
    );
    setProviderFilter(initialFilters.provider ? [initialFilters.provider] : []);
    setSelectedClientId(initialFilters.client_id || "");

    lastEmittedFiltersRef.current = JSON.stringify(normalizedInitialFilters);
  }, [
    initialFilters.searchQuery,
    initialFilters.active,
    initialFilters.provider,
    initialFilters.client_id,
    normalizedInitialFilters,
  ]);

  // Apply filters whenever any filter state changes
  useEffect(() => {
    const filters: Partial<UsersQueryParams> = {};

    if (searchQuery.trim()) {
      filters.searchQuery = searchQuery.trim();
    }

    if (statusFilter === "active") {
      filters.active = true;
    } else if (statusFilter === "inactive") {
      filters.active = false;
    }

    if (providerFilter.length > 0) {
      filters.provider = providerFilter[0];
    }

    if (selectedClientId) {
      filters.client_id = selectedClientId;
    }

    const next = JSON.stringify(filters);
    if (next !== lastEmittedFiltersRef.current) {
      lastEmittedFiltersRef.current = next;
      onFiltersChange(filters);
    }
  }, [
    searchQuery,
    statusFilter,
    providerFilter,
    selectedClientId,
    onFiltersChange,
  ]);

  // Count active filters
  const activeFiltersCount = (
    (searchQuery.trim() ? 1 : 0) +
    (statusFilter !== "all" ? 1 : 0) +
    (providerFilter.length > 0 ? 1 : 0) +
    (selectedClientId ? 1 : 0)
  );

  return (
    <FilterCard>
      <CardContent variant="compact">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div className="flex shrink-0 items-center gap-2">
            <span className="text-sm font-medium text-foreground">Filters</span>
            {activeFiltersCount > 0 && (
              <span className="text-xs text-foreground bg-black/5 dark:bg-white/10 px-1.5 py-0.5 rounded">
                {activeFiltersCount}
              </span>
            )}
          </div>

          <div className="flex w-full flex-1 flex-wrap items-center gap-2">
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground/50" />
              <Input
                placeholder="Search by name, email, or username..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-9 text-sm"
              />
            </div>

            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                {STATUS_OPTIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={selectedClientId || "all"}
              onValueChange={(value) => setSelectedClientId(value === "all" ? "" : value)}
              disabled={clientsLoading}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder={clientsLoading ? "Loading..." : "Client"} />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Clients</SelectItem>
                {clientsLoading ? (
                  <SelectItem value="loading" disabled>
                    <div className="flex items-center gap-2">
                      <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-primary"></div>
                      Loading...
                    </div>
                  </SelectItem>
                ) : clients.length === 0 ? (
                  <SelectItem value="none" disabled>
                    No clients available
                  </SelectItem>
                ) : (
                  clients.map((client: any) => (
                    <SelectItem key={client.client_id} value={client.client_id}>
                      {client.name || client.client_name || client.client_id}
                    </SelectItem>
                  ))
                )}
              </SelectContent>
            </Select>

            <Select value={providerFilter[0] || "all"} onValueChange={(value) => setProviderFilter(value === "all" ? [] : [value])}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Provider" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Providers</SelectItem>
                {providerOptions.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => {
                  setSearchQuery("");
                  setStatusFilter("all");
                  setProviderFilter([]);
                  setSelectedClientId("");
                  lastEmittedFiltersRef.current = "";
                  onFiltersChange({});
                }}
                className="h-9 text-sm text-foreground hover:text-foreground"
              >
                Clear
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </FilterCard>
  );
});

UsersFilterCard.displayName = "UsersFilterCard";

export default UsersFilterCard;
