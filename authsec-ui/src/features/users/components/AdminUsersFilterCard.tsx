import React, { useState, useEffect, useRef, useMemo } from "react";
import { CardContent } from "@/components/ui/card";
import { FilterCard } from "@/theme/components/cards";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Search } from "lucide-react";

export type AdminUsersFilterState = {
  status?: string;
  provider?: string;
  is_synced?: boolean | null;
  searchQuery?: string;
  providers?: string[];
};

interface AdminUsersFilterCardProps {
  filters: AdminUsersFilterState;
  providers: string[];
  onFiltersChange: (filters: AdminUsersFilterState) => void;
  showProviderFilter?: boolean;
  authsecMode?: boolean;
}

const STATUS_OPTIONS = [
  { value: "all", label: "All Status" },
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "pending", label: "Pending" },
  { value: "locked", label: "Locked" },
];

const AdminUsersFilterCard: React.FC<AdminUsersFilterCardProps> = ({
  filters,
  providers,
  onFiltersChange,
  showProviderFilter = false,
  authsecMode = false,
}) => {
  const [searchQuery, setSearchQuery] = useState(filters.searchQuery || "");
  const [statusFilter, setStatusFilter] = useState<string>(filters.status || "all");
  const [providerFilter, setProviderFilter] = useState<string>(filters.provider || "all");

  const lastEmittedFiltersRef = useRef<string>("");

  // Filter providers based on mode
  const availableProviders = useMemo(() => {
    if (authsecMode) {
      // Exclude AD and Entra providers
      return providers.filter(p =>
        p !== 'ad_sync' &&
        p !== 'entra_id' &&
        p !== 'azure_ad'
      );
    }
    return providers;
  }, [providers, authsecMode]);

  // Sync local state when filters prop changes
  useEffect(() => {
    setSearchQuery(filters.searchQuery || "");
    setStatusFilter(filters.status || "all");
    setProviderFilter(filters.provider || "all");
  }, [filters.searchQuery, filters.status, filters.provider]);

  // Apply filters whenever state changes
  useEffect(() => {
    const newFilters: AdminUsersFilterState = {};

    if (searchQuery.trim()) {
      newFilters.searchQuery = searchQuery.trim();
    }

    if (statusFilter !== "all") {
      newFilters.status = statusFilter;
    }

    if (providerFilter !== "all") {
      newFilters.provider = providerFilter;
    }

    const next = JSON.stringify(newFilters);
    if (next !== lastEmittedFiltersRef.current) {
      lastEmittedFiltersRef.current = next;
      onFiltersChange(newFilters);
    }
  }, [searchQuery, statusFilter, providerFilter, onFiltersChange]);

  const handleReset = () => {
    setSearchQuery("");
    setStatusFilter("all");
    setProviderFilter("all");
    lastEmittedFiltersRef.current = "";
    onFiltersChange({});
  };

  // Count active filters
  const activeFiltersCount = (
    (searchQuery.trim() ? 1 : 0) +
    (statusFilter !== "all" ? 1 : 0) +
    (providerFilter !== "all" ? 1 : 0)
  );

  return (
    <FilterCard>
      <CardContent variant="compact">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
          {/* Title Section */}
          <div className="flex shrink-0 items-center gap-2">
            <span className="text-sm font-medium text-foreground">
              Filters
            </span>
            {activeFiltersCount > 0 && (
              <span className="text-xs text-foreground bg-black/5 dark:bg-white/10 px-1.5 py-0.5 rounded">
                {activeFiltersCount}
              </span>
            )}
          </div>

          {/* Filters */}
          <div className="flex w-full flex-1 flex-wrap items-center gap-2">
            {/* Search Input */}
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground" />
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
                {STATUS_OPTIONS.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {showProviderFilter && availableProviders.length > 0 && (
              <Select value={providerFilter} onValueChange={setProviderFilter}>
                <SelectTrigger className="w-[150px] h-9 text-sm">
                  <SelectValue placeholder="Provider" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Providers</SelectItem>
                  {availableProviders.map((provider) => (
                    <SelectItem key={provider} value={provider}>
                      {provider === 'ad_sync' ? 'Active Directory' :
                       provider === 'entra_id' ? 'Azure Entra ID' :
                       provider === 'azure_ad' ? 'Azure AD' :
                       provider.charAt(0).toUpperCase() + provider.slice(1)}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleReset}
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
};

export default AdminUsersFilterCard;
