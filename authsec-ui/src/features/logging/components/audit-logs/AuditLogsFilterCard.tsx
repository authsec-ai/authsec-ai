import { useState, useEffect } from "react";
import { CardContent } from "../../../../components/ui/card";
import { FilterCard } from "@/theme/components/cards";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../../components/ui/select";
import { Button } from "../../../../components/ui/button";
import { ArrowDown, ArrowUp } from "lucide-react";
import type { AuditLog } from "../../../../types/entities";

interface AuditLogsFilterParams {
  action?: AuditLog["action"] | "all";
  resourceType?: AuditLog["resourceType"] | "all";
  severity?: AuditLog["severity"] | "all";
  category?: AuditLog["category"] | "all";
  timeRange?: string;
  showRollbackOnly?: boolean;
  sort_by?: "ts" | "service" | "event_type" | "operation";
  sort_desc?: boolean;
  event_type?: string;
}

interface AuditLogsFilterCardProps {
  onFiltersChange: (filters: AuditLogsFilterParams) => void;
  initialFilters: AuditLogsFilterParams;
}

export function AuditLogsFilterCard({
  onFiltersChange,
  initialFilters,
}: AuditLogsFilterCardProps) {
  const [filters, setFilters] = useState<AuditLogsFilterParams>(initialFilters);

  useEffect(() => {
    onFiltersChange(filters);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const handleFilterChange = (
    key: keyof AuditLogsFilterParams,
    value: string | boolean
  ) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const clearFilters = () => {
    setFilters({});
  };

  const activeFiltersCount = Object.keys(filters).filter((key) => {
    const value = filters[key as keyof AuditLogsFilterParams];
    return value && value !== "all" && value !== false;
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
            <Select
              value={filters.action || "all"}
              onValueChange={(value) => handleFilterChange("action", value)}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Action" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Actions</SelectItem>
                <SelectItem value="create">Created</SelectItem>
                <SelectItem value="update">Updated</SelectItem>
                <SelectItem value="delete">Deleted</SelectItem>
              </SelectContent>
            </Select>

            {/* <Select
              value={filters.resourceType || "all"}
              onValueChange={(value) => handleFilterChange("resourceType", value)}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Resource" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Resources</SelectItem>
                <SelectItem value="user">User</SelectItem>
                <SelectItem value="group">Group</SelectItem>
                <SelectItem value="role">Role</SelectItem>
                <SelectItem value="client">Client</SelectItem>
                <SelectItem value="resource">Resource</SelectItem>
                <SelectItem value="auth_method">Auth Method</SelectItem>
                <SelectItem value="config">Config</SelectItem>
              </SelectContent>
            </Select> */}

            <Select
              value={filters.timeRange || "all"}
              onValueChange={(value) => handleFilterChange("timeRange", value)}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Time" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Time</SelectItem>
                <SelectItem value="5m">Last 5 minutes</SelectItem>
                <SelectItem value="1h">Last hour</SelectItem>
                <SelectItem value="24h">Last 24 hours</SelectItem>
                <SelectItem value="7d">Last 7 days</SelectItem>
              </SelectContent>
            </Select>

            <Select
              value={filters.sort_by || "ts"}
              onValueChange={(value) =>
                handleFilterChange(
                  "sort_by",
                  value as "ts" | "service" | "event_type" | "operation"
                )
              }
            >
              <SelectTrigger className="w-[150px] h-9 text-sm">
                <SelectValue placeholder="Sort By" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="ts">Timestamp</SelectItem>
                <SelectItem value="service">Service</SelectItem>
                <SelectItem value="event_type">Event Type</SelectItem>
                <SelectItem value="operation">Operation</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              size="sm"
              onClick={() =>
                handleFilterChange("sort_desc", !filters.sort_desc)
              }
              className="h-9 w-[150px] text-sm"
              title={
                filters.sort_desc === false
                  ? "Sort ascending (oldest first)"
                  : "Sort descending (newest first)"
              }
            >
              {filters.sort_desc === false ? (
                <>
                  <ArrowUp className="h-4 w-4" />
                  <span className="text-xs">Oldest first</span>
                </>
              ) : (
                <>
                  <ArrowDown className="h-4 w-4" />
                  <span className="text-xs">Newest first</span>
                </>
              )}
            </Button>

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
