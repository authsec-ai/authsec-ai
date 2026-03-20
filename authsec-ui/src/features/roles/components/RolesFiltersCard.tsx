import React from "react";
import { CardContent } from "@/components/ui/card";
import { FilterCard } from "@/theme/components/cards";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { IconFilter, IconSearch } from "@tabler/icons-react";
import { Switch } from "@/components/ui/switch";

interface RolesFiltersCardProps {
  /* Primary filters */
  searchTerm: string;
  onSearchTermChange: (v: string) => void;
  environmentFilter: string;
  onEnvironmentFilterChange: (v: string) => void;
  typeFilter: string;
  onTypeFilterChange: (v: string) => void;
  statusFilter: string;
  onStatusFilterChange: (v: string) => void;

  /* Advanced toggling */
  showAdvancedFilters: boolean;
  onToggleAdvancedFilters: () => void;

  /* Advanced filters */
  resourceFilter: string;
  onResourceFilterChange: (v: string) => void;
  resources: { id: string; display_name: string }[];

  userCountFilter: string;
  onUserCountFilterChange: (v: string) => void;

  permissionCountFilter: string;
  onPermissionCountFilterChange: (v: string) => void;

  onResetFilters: () => void;
  unusedOnly: boolean;
  onUnusedToggle: (val: boolean) => void;
}

/**
 * Extracted filter/search UI for the Roles dashboard. Keeps RolesPage compact
 * and allows future reuse (e.g. in a dedicated sidebar variant).
 */
export function RolesFiltersCard({
  searchTerm,
  onSearchTermChange,
  environmentFilter,
  onEnvironmentFilterChange,
  typeFilter,
  onTypeFilterChange,
  statusFilter,
  onStatusFilterChange,
  showAdvancedFilters,
  onToggleAdvancedFilters,
  resourceFilter,
  onResourceFilterChange,
  resources,
  userCountFilter,
  onUserCountFilterChange,
  permissionCountFilter,
  onPermissionCountFilterChange,
  onResetFilters,
  unusedOnly,
  onUnusedToggle,
}: RolesFiltersCardProps) {
  const activeFiltersCount =
    (searchTerm.trim() ? 1 : 0) +
    (environmentFilter !== "all" ? 1 : 0) +
    (typeFilter !== "all" ? 1 : 0) +
    (statusFilter !== "all" ? 1 : 0) +
    (resourceFilter !== "all" ? 1 : 0) +
    (userCountFilter !== "all" ? 1 : 0) +
    (permissionCountFilter !== "all" ? 1 : 0) +
    (unusedOnly ? 1 : 0);

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
              <IconSearch className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground/50" />
              <Input
                placeholder="Search roles..."
                value={searchTerm}
                onChange={(e) => onSearchTermChange(e.target.value)}
                className="pl-9 h-9 text-sm"
              />
            </div>

            <Select value={environmentFilter} onValueChange={onEnvironmentFilterChange}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Env" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Env</SelectItem>
                <SelectItem value="dev">Dev</SelectItem>
                <SelectItem value="stage">Stage</SelectItem>
                <SelectItem value="prod">Prod</SelectItem>
              </SelectContent>
            </Select>

            <Select value={typeFilter} onValueChange={onTypeFilterChange}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                <SelectItem value="system">System</SelectItem>
                <SelectItem value="custom">Custom</SelectItem>
              </SelectContent>
            </Select>

            <Select value={statusFilter} onValueChange={onStatusFilterChange}>
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="inactive">Inactive</SelectItem>
              </SelectContent>
            </Select>

            <div className="flex items-center gap-2">
              <Label htmlFor="unusedToggle" className="whitespace-nowrap text-sm text-foreground">
                Unused
              </Label>
              <Switch id="unusedToggle" checked={unusedOnly} onCheckedChange={onUnusedToggle} />
            </div>

            <Button
              variant="ghost"
              size="sm"
              onClick={onToggleAdvancedFilters}
              className="h-9 text-sm text-foreground hover:text-foreground"
            >
              <IconFilter className="mr-1 h-4 w-4" />
              Advanced
            </Button>

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={onResetFilters}
                className="h-9 text-sm text-foreground hover:text-foreground"
              >
                Clear
              </Button>
            )}
          </div>
        </div>

        {showAdvancedFilters && (
          <div className="pt-3 mt-3 border-t border-border">
            <div className="flex flex-wrap items-center gap-2">
              <Select value={resourceFilter} onValueChange={onResourceFilterChange}>
                <SelectTrigger className="w-[130px] h-9 text-sm">
                  <SelectValue placeholder="Resource" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Resources</SelectItem>
                  {resources.map((resource) => (
                    <SelectItem key={resource.id} value={resource.id}>
                      {resource.display_name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={userCountFilter} onValueChange={onUserCountFilterChange}>
                <SelectTrigger className="w-[130px] h-9 text-sm">
                  <SelectValue placeholder="User Count" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Any Count</SelectItem>
                  <SelectItem value="none">None (0)</SelectItem>
                  <SelectItem value="low">Low (1-5)</SelectItem>
                  <SelectItem value="medium">Medium (6-20)</SelectItem>
                  <SelectItem value="high">High (20+)</SelectItem>
                </SelectContent>
              </Select>

              <Select value={permissionCountFilter} onValueChange={onPermissionCountFilterChange}>
                <SelectTrigger className="w-[130px] h-9 text-sm">
                  <SelectValue placeholder="Permissions" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">Any Count</SelectItem>
                  <SelectItem value="none">None (0)</SelectItem>
                  <SelectItem value="low">Low (1-5)</SelectItem>
                  <SelectItem value="medium">Medium (6-15)</SelectItem>
                  <SelectItem value="high">High (15+)</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>
        )}
      </CardContent>
    </FilterCard>
  );
}
 