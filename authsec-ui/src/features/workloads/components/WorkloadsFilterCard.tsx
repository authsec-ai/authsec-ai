import { useState, useEffect } from "react";
import { CardContent } from "../../../components/ui/card";
import { FilterCard } from "@/theme/components/cards";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../components/ui/select";
import { Input } from "../../../components/ui/input";
import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import { Search } from "lucide-react";

interface WorkloadsFilterParams {
  searchQuery?: string;
  type?: string;
  status?: string;
}

interface WorkloadsFilterCardProps {
  onFiltersChange: (filters: WorkloadsFilterParams) => void;
  initialFilters: WorkloadsFilterParams;
}

export function WorkloadsFilterCard({
  onFiltersChange,
  initialFilters,
}: WorkloadsFilterCardProps) {
  const [filters, setFilters] = useState<WorkloadsFilterParams>(initialFilters);

  useEffect(() => {
    onFiltersChange(filters);
  }, [filters, onFiltersChange]);

  const handleFilterChange = (
    key: keyof WorkloadsFilterParams,
    value: string
  ) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const clearFilters = () => {
    setFilters({});
  };

  const activeFiltersCount = Object.keys(filters).filter((key) => {
    const value = filters[key as keyof WorkloadsFilterParams];
    return value && value !== "all" && value !== "";
  }).length;

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
                placeholder="Search by SPIFFE ID, workload ID, or selectors..."
                value={filters.searchQuery || ""}
                onChange={(e) =>
                  handleFilterChange("searchQuery", e.target.value)
                }
                className="pl-9 h-9 text-sm"
              />
            </div>

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={clearFilters}
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
}
