import React, { useState, useEffect, useRef, useMemo } from "react";
import { CardContent } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Search } from "lucide-react";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import type { ApiOAuthScopesQueryParams } from "../types";

interface ApiOAuthScopesFilterCardProps {
  onFiltersChange: (filters: Partial<ApiOAuthScopesQueryParams>) => void;
  initialFilters?: Partial<ApiOAuthScopesQueryParams>;
}

const ApiOAuthScopesFilterCard = React.memo(
  ({ onFiltersChange, initialFilters = {} }: ApiOAuthScopesFilterCardProps) => {
    const [searchQuery, setSearchQuery] = useState(
      initialFilters.searchQuery || ""
    );
    const lastEmittedFiltersRef = useRef<string>("");

    const normalizedInitialFilters = useMemo(() => {
      const f: Partial<ApiOAuthScopesQueryParams> = {};
      if (initialFilters.searchQuery) f.searchQuery = initialFilters.searchQuery;
      return f;
    }, [initialFilters.searchQuery]);

    useEffect(() => {
      setSearchQuery(initialFilters.searchQuery || "");
      lastEmittedFiltersRef.current = JSON.stringify(normalizedInitialFilters);
    }, [initialFilters.searchQuery, normalizedInitialFilters]);

    useEffect(() => {
      const filters: Partial<ApiOAuthScopesQueryParams> = {};

      if (searchQuery.trim()) {
        filters.searchQuery = searchQuery.trim();
      }

      const next = JSON.stringify(filters);
      if (next !== lastEmittedFiltersRef.current) {
        lastEmittedFiltersRef.current = next;
        onFiltersChange(filters);
      }
    }, [searchQuery, onFiltersChange]);

    const activeFiltersCount = searchQuery.trim() ? 1 : 0;

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
                  placeholder="Search by ID, name, or description..."
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
  }
);

ApiOAuthScopesFilterCard.displayName = "ApiOAuthScopesFilterCard";

export default ApiOAuthScopesFilterCard;
