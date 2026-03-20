import React, { useState, useEffect, useRef, useMemo } from "react";
import { CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Search } from "lucide-react";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import type { Permission } from "@/app/api/permissionsApi";

export interface PermissionsQueryParams {
  searchQuery?: string;
  actionFilter?: string;
  resourceFilter?: string;
}

interface PermissionsFilterCardProps {
  onFiltersChange: (filters: Partial<PermissionsQueryParams>) => void;
  initialFilters?: Partial<PermissionsQueryParams>;
  permissionsData: Permission[];
}

const PermissionsFilterCardComponent = ({
  onFiltersChange,
  initialFilters = {},
  permissionsData,
}: PermissionsFilterCardProps) => {
  const [searchQuery, setSearchQuery] = useState(initialFilters.searchQuery || "");
  const [actionFilter, setActionFilter] = useState(initialFilters.actionFilter || "");
  const [resourceFilter, setResourceFilter] = useState(initialFilters.resourceFilter || "");

  // Track last emitted filters to avoid emitting duplicates and loops
  const lastEmittedFiltersRef = useRef<string>("");

  // Normalize incoming initialFilters for stable comparisons
  const normalizedInitialFilters = useMemo(() => {
    const f: Partial<PermissionsQueryParams> = {};
    if (initialFilters.searchQuery) f.searchQuery = initialFilters.searchQuery;
    if (initialFilters.actionFilter) f.actionFilter = initialFilters.actionFilter;
    if (initialFilters.resourceFilter) f.resourceFilter = initialFilters.resourceFilter;
    return f;
  }, [
    initialFilters.searchQuery,
    initialFilters.actionFilter,
    initialFilters.resourceFilter,
  ]);

  // Sync local filter controls when initialFilters prop changes (e.g., deep links)
  useEffect(() => {
    setSearchQuery(initialFilters.searchQuery || "");
    setActionFilter(initialFilters.actionFilter || "");
    setResourceFilter(initialFilters.resourceFilter || "");

    // Record last emitted filters as the normalized initial filters to prevent an immediate re-emit loop
    lastEmittedFiltersRef.current = JSON.stringify(normalizedInitialFilters);
  }, [
    initialFilters.searchQuery,
    initialFilters.actionFilter,
    initialFilters.resourceFilter,
    normalizedInitialFilters,
  ]);

  // Apply filters whenever any filter state changes
  useEffect(() => {
    const filters: Partial<PermissionsQueryParams> = {};

    if (searchQuery.trim()) {
      filters.searchQuery = searchQuery.trim();
    }
    if (actionFilter) {
      filters.actionFilter = actionFilter;
    }
    if (resourceFilter) {
      filters.resourceFilter = resourceFilter;
    }

    const next = JSON.stringify(filters);
    if (next !== lastEmittedFiltersRef.current) {
      lastEmittedFiltersRef.current = next;
      onFiltersChange(filters);
    }
  }, [
    searchQuery,
    actionFilter,
    resourceFilter,
    onFiltersChange,
  ]);

  // Extract unique actions and resources from permissions data
  const uniqueActions = useMemo(() => {
    const actions = new Set<string>();
    permissionsData.forEach(p => {
      if (p.action) actions.add(p.action);
    });
    return Array.from(actions).sort();
  }, [permissionsData]);

  const uniqueResources = useMemo(() => {
    const resources = new Set<string>();
    permissionsData.forEach(p => {
      if (p.resource) resources.add(p.resource);
    });
    return Array.from(resources).sort();
  }, [permissionsData]);

  // Count active filters
  const activeFiltersCount =
    (searchQuery.trim() ? 1 : 0) +
    (actionFilter ? 1 : 0) +
    (resourceFilter ? 1 : 0);

  const handleClearFilters = () => {
    setSearchQuery("");
    setActionFilter("");
    setResourceFilter("");
  };

  return (
    <FilterShell>
      <CardContent variant="compact">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
          {/* Title Section */}
          <div className="flex shrink-0 items-center gap-2">
            <span className="text-sm font-medium text-foreground">Filters</span>
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
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-foreground/50" />
              <Input
                placeholder="Search permissions..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-9 text-sm"
              />
            </div>

            {/* Action Filter */}
            <Select value={actionFilter || "all"} onValueChange={(v) => setActionFilter(v === "all" ? "" : v)}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="All actions" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All actions</SelectItem>
                {uniqueActions.map((action) => (
                  <SelectItem key={action} value={action}>
                    {action}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {/* Resource Filter */}
            <Select value={resourceFilter || "all"} onValueChange={(v) => setResourceFilter(v === "all" ? "" : v)}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="All resources" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All resources</SelectItem>
                {uniqueResources.map((resource) => (
                  <SelectItem key={resource} value={resource}>
                    {resource}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            {activeFiltersCount > 0 && (
              <Button variant="ghost" size="sm" onClick={handleClearFilters} className="h-9 text-sm text-foreground hover:text-foreground">
                Clear
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </FilterShell>
  );
};

export const PermissionsFilterCard = React.memo(PermissionsFilterCardComponent);
PermissionsFilterCard.displayName = "PermissionsFilterCard";
