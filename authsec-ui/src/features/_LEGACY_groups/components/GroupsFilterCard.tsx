import React, { useState, useEffect, useRef, useMemo } from "react";
import { CardContent } from "@/components/ui/card";
import { FilterCard } from "@/theme/components/cards";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import {
  Search,
} from "lucide-react";

// Group query parameters interface
export interface GroupsQueryParams {
  searchQuery?: string;
}

interface GroupData {
  name?: string;
  created_at?: string;
  [key: string]: any;
}

export interface GroupsFilterCardProps {
  onFiltersChange: (filters: Partial<GroupsQueryParams>) => void;
  initialFilters?: Partial<GroupsQueryParams>;
  groupsData?: GroupData[];
}

const GroupsFilterCard = React.memo(({
  onFiltersChange,
  initialFilters = {},
  groupsData: _groupsData = []
}: GroupsFilterCardProps) => {
  const [searchQuery, setSearchQuery] = useState(initialFilters.searchQuery || "");

  // Track last emitted filters to avoid emitting duplicates and loops
  const lastEmittedFiltersRef = useRef<string>("");

  // Normalize incoming initialFilters for stable comparisons
  const normalizedInitialFilters = useMemo(() => {
    const f: Partial<GroupsQueryParams> = {};
    if (initialFilters.searchQuery) f.searchQuery = initialFilters.searchQuery;
    return f;
  }, [
    initialFilters.searchQuery,
  ]);

  // Sync local filter controls when initialFilters prop changes (e.g., deep links)
  useEffect(() => {
    setSearchQuery(initialFilters.searchQuery || "");

    // Record last emitted filters as the normalized initial filters to prevent an immediate re-emit loop
    lastEmittedFiltersRef.current = JSON.stringify(normalizedInitialFilters);
  }, [
    initialFilters.searchQuery,
    normalizedInitialFilters,
  ]);

  // Apply filters whenever any filter state changes
  useEffect(() => {
    const filters: Partial<GroupsQueryParams> = {};

    if (searchQuery.trim()) {
      filters.searchQuery = searchQuery.trim();
    }

    const next = JSON.stringify(filters);
    if (next !== lastEmittedFiltersRef.current) {
      lastEmittedFiltersRef.current = next;
      onFiltersChange(filters);
    }
  }, [
    searchQuery,
    onFiltersChange,
  ]);

  // Count active filters
  const activeFiltersCount = (searchQuery.trim() ? 1 : 0);

  return (
    <FilterCard>
      <CardContent variant="compact">
        <div className="flex flex-col items-start gap-[var(--space-4)] lg:flex-row lg:items-center">
          {/* Title Section */}
          <div className="flex shrink-0 items-center gap-3">
            <div className="h-8 w-1 rounded-full bg-[var(--color-primary)]/35" />
            <h3 className="whitespace-nowrap text-[length:var(--font-size-heading-xs)] font-[var(--font-weight-semibold)] tracking-[var(--letter-spacing-tight)] text-[color:var(--color-text-primary)]">
              Filters
            </h3>
            {activeFiltersCount > 0 && (
              <Badge variant="secondary" className="font-[var(--font-weight-medium)]">
                {activeFiltersCount}
              </Badge>
            )}
          </div>

          {/* Filters */}
          <div className="flex w-full flex-1 flex-wrap gap-[var(--space-3)]">
            {/* Search Input */}
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400" />
              <Input
                placeholder="Search groups by name or description..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-10 h-[var(--component-input-height)] bg-white dark:bg-neutral-800 border-slate-300 dark:border-neutral-600"
              />
            </div>

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSearchQuery("")}
                className="flex-1 min-w-[100px] h-[var(--component-input-height)]"
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

GroupsFilterCard.displayName = "GroupsFilterCard";

export default GroupsFilterCard;