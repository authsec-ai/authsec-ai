import React, { useState, useEffect, useCallback, useRef } from "react";
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
import { Search, Server, Globe } from "lucide-react";

// Provider options - static for better reliability
const DEFAULT_PROVIDER_OPTIONS = [
  { value: "github", label: "GitHub", icon: Globe },
  { value: "google", label: "Google", icon: Globe },
  { value: "microsoft", label: "Microsoft", icon: Globe },
  { value: "slack", label: "Slack", icon: Globe },
  { value: "custom", label: "Custom", icon: Server },
];

// Status options
const STATUS_OPTIONS = [
  { value: "all", label: "All Status" },
  { value: "connected", label: "Connected" },
  { value: "needs_consent", label: "Needs Consent" },
  { value: "error", label: "Error" },
];

interface ExternalServiceFilters {
  searchQuery?: string;
  provider?: string;
  status?: string;
}

interface ExternalServiceFiltersCardProps {
  onFiltersChange?: (filters: Partial<ExternalServiceFilters>) => void;
  initialFilters?: Partial<ExternalServiceFilters>;
  // Legacy props for backward compatibility
  searchTerm?: string;
  onSearchTermChange?: (v: string) => void;
  providerFilter?: string;
  onProviderFilterChange?: (v: string) => void;
  statusFilter?: string;
  onStatusFilterChange?: (v: string) => void;
  showAdvancedFilters?: boolean;
  onToggleAdvancedFilters?: () => void;
  clientFilter?: string;
  onClientFilterChange?: (v: string) => void;
  clients?: { id: string; name: string }[];
  onResetFilters?: () => void;
}

const ExternalServiceFiltersCard = React.memo(
  ({
    // New props
    onFiltersChange,
    initialFilters = {},
    // Legacy props
    searchTerm = "",
    onSearchTermChange,
    providerFilter = "all",
    onProviderFilterChange,
    statusFilter = "all",
    onStatusFilterChange,
    showAdvancedFilters,
    onToggleAdvancedFilters,
    clientFilter,
    onClientFilterChange,
    clients = [],
    onResetFilters,
  }: ExternalServiceFiltersCardProps) => {
    // Use legacy props for now to maintain compatibility
    const [searchQuery, setSearchQuery] = useState(searchTerm);
    const [provider, setProvider] = useState(providerFilter);
    const [status, setStatus] = useState(statusFilter);

    // Track last emitted filters to avoid emitting duplicates and loops
    const lastEmittedFiltersRef = useRef<string>("");

    // Sync with legacy props when they change
    useEffect(() => {
      setSearchQuery(searchTerm);
    }, [searchTerm]);

    useEffect(() => {
      setProvider(providerFilter);
    }, [providerFilter]);

    useEffect(() => {
      setStatus(statusFilter);
    }, [statusFilter]);

    // Handle search changes
    const handleSearchChange = (value: string) => {
      setSearchQuery(value);
      onSearchTermChange?.(value);
    };

    // Handle provider changes
    const handleProviderChange = (value: string) => {
      setProvider(value);
      onProviderFilterChange?.(value);
    };

    // Handle status changes
    const handleStatusChange = (value: string) => {
      setStatus(value);
      onStatusFilterChange?.(value);
    };

    // Reset all filters
    const resetFilters = useCallback(() => {
      setSearchQuery("");
      setProvider("all");
      setStatus("all");
      onSearchTermChange?.("");
      onProviderFilterChange?.("all");
      onStatusFilterChange?.("all");
      onResetFilters?.();
    }, [
      onSearchTermChange,
      onProviderFilterChange,
      onStatusFilterChange,
      onResetFilters,
    ]);

    // Count active filters
    const activeFiltersCount =
      (searchQuery.trim() ? 1 : 0) +
      (provider !== "all" ? 1 : 0) +
      (status !== "all" ? 1 : 0);

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
                  placeholder="Search external services..."
                  value={searchQuery}
                  onChange={(e) => handleSearchChange(e.target.value)}
                  className="pl-9 h-9 text-sm"
                />
              </div>

              <Select value={provider} onValueChange={handleProviderChange}>
                <SelectTrigger className="w-[130px] h-9 text-sm">
                  <SelectValue placeholder="Provider" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Providers</SelectItem>
                  {DEFAULT_PROVIDER_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>

              <Select value={status} onValueChange={handleStatusChange}>
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

              {activeFiltersCount > 0 && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={resetFilters}
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
);

ExternalServiceFiltersCard.displayName = "ExternalServiceFiltersCard";

export { ExternalServiceFiltersCard };
