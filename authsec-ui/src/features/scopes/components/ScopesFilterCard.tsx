import React, { useState, useEffect, useRef, useMemo } from "react";
import { CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Search } from "lucide-react";
import { FilterCard as FilterShell } from "@/theme/components/cards";

export interface ScopesQueryParams {
  searchQuery?: string;
}

interface ScopeData {
  name?: string;
  created_at?: string;
  [key: string]: any;
}

interface ScopesFilterCardProps {
  onFiltersChange: (filters: Partial<ScopesQueryParams>) => void;
  initialFilters?: Partial<ScopesQueryParams>;
  scopesData?: ScopeData[];
}

const ScopesFilterCard = React.memo(({
  onFiltersChange,
  initialFilters = {},
  scopesData: _scopesData = []
}: ScopesFilterCardProps) => {
  const [searchQuery, setSearchQuery] = useState(initialFilters.searchQuery || "");

  // Track last emitted filters to avoid emitting duplicates and loops
  const lastEmittedFiltersRef = useRef<string>("");

  // Normalize incoming initialFilters for stable comparisons
  const normalizedInitialFilters = useMemo(() => {
    const f: Partial<ScopesQueryParams> = {};
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
    const filters: Partial<ScopesQueryParams> = {};

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
    <FilterShell>
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
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground/50" />
              <Input
                placeholder="Search scopes by name or description..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-9 text-sm"
              />
            </div>

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSearchQuery("")}
                className="h-9 text-sm text-foreground hover:text-foreground"
              >
                Clear
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </FilterShell>
  );
});

ScopesFilterCard.displayName = "ScopesFilterCard";

export default ScopesFilterCard;
