import React, { useState, useEffect, useRef } from "react";
import { CardContent } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { FilterCard } from "@/theme/components/cards";
import {
  Select,
  SelectTrigger,
  SelectValue,
  SelectContent,
  SelectItem,
} from "@/components/ui/select";
import { Badge } from "@/components/ui/badge";
import { Search } from "lucide-react";
import type { AuthMethodStatus } from "../types";

// Provider type options
const PROVIDER_TYPE_OPTIONS = [
  { value: "oidc", label: "OIDC" },
  { value: "saml", label: "SAML" },
  { value: "email-pass", label: "Email/Pass" },
  { value: "oauth2", label: "OAuth2" },
  { value: "webauthn", label: "WebAuthn" },
  { value: "totp", label: "Time-based OTP" },
];

// Status options
const STATUS_OPTIONS = [
  { value: "active", label: "Active" },
  { value: "inactive", label: "Inactive" },
  { value: "maintenance", label: "Maintenance" },
];

interface AuthenticationFilters {
  providerType?: string;
  status?: string;
  clientId?: string;
  searchQuery?: string;
}

interface AuthenticationFilterCardProps {
  onFiltersChange: (filters: Partial<AuthenticationFilters>) => void;
  initialFilters?: Partial<AuthenticationFilters>;
  clients?: any[];
  loadingClients?: boolean;
  selectedClientId?: string;
  onClientChange?: (clientId: string) => void;
}

const AuthenticationFilterCard = React.memo(({
  onFiltersChange,
  initialFilters = {},
  clients = [],
  loadingClients = false,
  selectedClientId,
  onClientChange
}: AuthenticationFilterCardProps) => {
  const [providerTypeFilter, setProviderTypeFilter] = useState<string>(
    initialFilters.providerType || "all"
  );
  const [statusFilter, setStatusFilter] = useState<string>(
    initialFilters.status || "all"
  );
  const [searchQuery, setSearchQuery] = useState<string>(
    initialFilters.searchQuery || ""
  );

  // Use refs to track last emitted values to prevent infinite loops
  const lastEmittedFiltersRef = useRef<string>("");

  // Apply filters whenever filter state changes - only emit if changed
  useEffect(() => {
    const filters: Partial<AuthenticationFilters> = {};

    if (providerTypeFilter !== "all") {
      filters.providerType = providerTypeFilter;
    }

    if (statusFilter !== "all") {
      filters.status = statusFilter;
    }

    if (searchQuery.trim()) {
      filters.searchQuery = searchQuery.trim();
    }

    const filtersJson = JSON.stringify(filters);
    if (filtersJson !== lastEmittedFiltersRef.current) {
      lastEmittedFiltersRef.current = filtersJson;
      onFiltersChange(filters);
    }
  }, [providerTypeFilter, statusFilter, searchQuery, onFiltersChange]);

  const clearFilters = () => {
    setProviderTypeFilter("all");
    setStatusFilter("all");
    setSearchQuery("");
    onClientChange?.(""); // Clear client selection
  };

  const activeFiltersCount =
    (providerTypeFilter !== "all" ? 1 : 0) +
    (statusFilter !== "all" ? 1 : 0) +
    (selectedClientId ? 1 : 0) + // Count client filter as active
    (searchQuery.trim() ? 1 : 0); // Count search as active

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
              <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground/50" />
              <Input
                type="text"
                placeholder="Search providers..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9 h-9 text-sm"
              />
            </div>

            <Select
              value={selectedClientId || "all"}
              onValueChange={(value) => {
                onClientChange?.(value === "all" ? "" : value);
              }}
              disabled={loadingClients}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder={loadingClients ? "Loading..." : "Client"} />
              </SelectTrigger>
              <SelectContent>
                {loadingClients ? (
                  <SelectItem value="loading" disabled>
                    <div className="flex items-center gap-2">
                      <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-primary"></div>
                      Loading...
                    </div>
                  </SelectItem>
                ) : (
                  <>
                    <SelectItem value="all">All Clients</SelectItem>
                    {clients.length === 0 ? (
                      <SelectItem value="none" disabled>
                        No clients available
                      </SelectItem>
                    ) : (
                      clients.map((client) => (
                        <SelectItem key={client.client_id} value={client.client_id}>
                          {String(client.name || client.client_id || 'Unnamed Client')}
                        </SelectItem>
                      ))
                    )}
                  </>
                )}
              </SelectContent>
            </Select>

            <Select
              value={providerTypeFilter}
              onValueChange={(value) => {
                setProviderTypeFilter(value);
              }}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Types</SelectItem>
                {PROVIDER_TYPE_OPTIONS.map(option => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>

            <Select
              value={statusFilter}
              onValueChange={(value) => {
                setStatusFilter(value);
              }}
            >
              <SelectTrigger className="w-[130px] h-9 text-sm">
                <SelectValue placeholder="Status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Status</SelectItem>
                {STATUS_OPTIONS.map(option => (
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
});

AuthenticationFilterCard.displayName = "AuthenticationFilterCard";

export default AuthenticationFilterCard;
