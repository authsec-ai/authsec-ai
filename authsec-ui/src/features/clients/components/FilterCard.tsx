import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { CardContent } from "@/components/ui/card"
import { FilterCard as ThemedFilterCard } from "@/theme/components/cards"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Badge } from "@/components/ui/badge"
import { Search } from "lucide-react"
import type { ClientsFilters } from "@/types/entities"

export function FilterCard({
  setFilters,
}: {
  setFilters: (filters: Partial<ClientsFilters>) => void;
}) {
  // Track search text and access filter; type/auth filters removed per latest UX
  const [searchQuery, setSearchQuery] = useState("")
  const [accessFilter, setAccessFilter] = useState("all")

  // Apply filters whenever any filter state changes
  // Use a debounced approach to prevent excessive updates
  useEffect(() => {
    const timeoutId = setTimeout(() => {
      const filters: Partial<ClientsFilters> = {};

      const trimmedSearch = searchQuery.trim();
      if (trimmedSearch) {
        filters.name = trimmedSearch;
        filters.email = trimmedSearch;
        filters.search = trimmedSearch;
      }

      if (accessFilter !== "all") {
        const normalizedStatus = accessFilter === "active"
          ? "Active"
          : accessFilter === "disabled"
          ? "Disabled"
          : accessFilter.charAt(0).toUpperCase() + accessFilter.slice(1);
        filters.status = normalizedStatus;
        filters.access_status = accessFilter as "active" | "restricted" | "disabled";
      }

      setFilters(filters);
    }, 300); // Debounce for 300ms

    return () => clearTimeout(timeoutId);
  }, [
    searchQuery,
    accessFilter,
    setFilters
  ]);

  const activeFiltersCount = [
    searchQuery,
    accessFilter !== "all" ? accessFilter : "",
  ].filter(Boolean).length

  const handleClearFilters = () => {
    setSearchQuery("");
    setAccessFilter("all");
  };

  return (
    <ThemedFilterCard>
      <CardContent variant="compact">
        <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
          <div className="flex shrink-0 items-center gap-2">
            <span
              data-slot="filter-label"
              className="text-xs font-semibold tracking-[0.12em] uppercase text-[var(--editorial-text-3)]"
            >
              Filters
            </span>
            {activeFiltersCount > 0 && (
              <span
                data-slot="filter-count"
                className="rounded-full border border-[var(--editorial-border-soft)] bg-[var(--editorial-panel)] px-2 py-0.5 text-[11px] font-medium text-[var(--editorial-text-2)]"
              >
                {activeFiltersCount}
              </span>
            )}
          </div>

          <div className="flex w-full flex-1 flex-wrap items-center gap-2">
            <div className="flex-1 min-w-[200px] relative">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--editorial-text-3)]" />
              <Input
                placeholder="Search by name or endpoint..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-9 text-sm"
              />
            </div>

            <Select value={accessFilter} onValueChange={setAccessFilter}>
              <SelectTrigger size="sm" className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Access" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Access</SelectItem>
                <SelectItem value="active">Active</SelectItem>
                <SelectItem value="restricted">Restricted</SelectItem>
                <SelectItem value="disabled">Disabled</SelectItem>
              </SelectContent>
            </Select>

            {activeFiltersCount > 0 && (
              <Button
                variant="ghost"
                size="sm"
                onClick={handleClearFilters}
                className="h-9 rounded-md text-sm text-[var(--editorial-text-2)] hover:text-[var(--editorial-text-1)] hover:bg-[var(--editorial-panel)]"
              >
                Clear
              </Button>
            )}
          </div>
        </div>
      </CardContent>
    </ThemedFilterCard>
  )
}
