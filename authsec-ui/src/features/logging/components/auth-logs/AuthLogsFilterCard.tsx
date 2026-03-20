import { useState, useEffect } from "react";
import { CardContent } from "../../../../components/ui/card";
import { FilterCard as FilterShell } from "@/theme/components/cards";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../../../components/ui/select";
import { Button } from "../../../../components/ui/button";
import type { AuthLog } from "../../../../types/entities";

interface AuthLogsFilterParams {
  logType?: AuthLog["logType"] | "all";
  clientType?: AuthLog["clientType"] | "all";
  status?: AuthLog["status"] | "all";
  authMethod?: AuthLog["authMethod"] | "all";
  timeRange?: string;
}

interface AuthLogsFilterCardProps {
  onFiltersChange: (filters: AuthLogsFilterParams) => void;
  initialFilters: AuthLogsFilterParams;
  onGroupByClick: () => void;
}

export function AuthLogsFilterCard({
  onFiltersChange,
  initialFilters,
  onGroupByClick,
}: AuthLogsFilterCardProps) {
  const [filters, setFilters] = useState<AuthLogsFilterParams>(initialFilters);

  useEffect(() => {
    onFiltersChange(filters);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const handleFilterChange = (
    key: keyof AuthLogsFilterParams,
    value: string
  ) => {
    setFilters((prev) => ({ ...prev, [key]: value }));
  };

  const clearFilters = () => {
    setFilters({});
  };

  const activeFiltersCount = Object.keys(filters).filter(
    (key) =>
      filters[key as keyof AuthLogsFilterParams] &&
      filters[key as keyof AuthLogsFilterParams] !== "all"
  ).length;

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
            <Select
              value={filters.logType || "all"}
              onValueChange={(value) => handleFilterChange("logType", value)}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Log Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="authn">Authentication</SelectItem>
                <SelectItem value="authz">Authorization</SelectItem>
              </SelectContent>
            </Select>

            <Select
              value={filters.clientType || "all"}
              onValueChange={(value) => handleFilterChange("clientType", value)}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Client Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Clients</SelectItem>
                <SelectItem value="mcp_server">MCP Server</SelectItem>
                <SelectItem value="ai_agent">AI Agent</SelectItem>
              </SelectContent>
            </Select>

            <Select
              value={filters.status || "all"}
              onValueChange={(value) => handleFilterChange("status", value)}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="success">Success</SelectItem>
                <SelectItem value="failure">Failure</SelectItem>
              </SelectContent>
            </Select>

            <Button
              variant="outline"
              size="sm"
              onClick={onGroupByClick}
              className="h-9 text-sm whitespace-nowrap"
            >
              Group By Users
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
    </FilterShell>
  );
}
