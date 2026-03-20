import { useState, useMemo } from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../components/ui/button";
import { Input } from "../../components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "../../components/ui/card";
import { MetricCard, MetricCardGrid } from "../../components/ui/metric-card";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "../../components/ui/select";
import { Label } from "../../components/ui/label";
import { Search, Key, Lock, AlertTriangle, Database, Eye, Filter, RotateCcw } from "lucide-react";

// Import components
import { EnhancedVaultTable, BulkActionsBar } from "./components";

// Mock data for vault secrets
const mockSecrets = [
  {
    id: "secret_1",
    name: "database-connection-string",
    type: "connection_string" as const,
    description: "database connection string",
    tags: ["production", "database"],
    createdAt: "2024-01-15T10:00:00Z",
    updatedAt: "2024-01-20T14:30:00Z",
    expiresAt: "2024-12-31T23:59:59Z",
    accessCount: 45,
    lastAccessed: "2024-01-22T09:15:00Z",
    isExpired: false,
  },
  {
    id: "secret_2",
    name: "api-key-stripe",
    type: "api_key" as const,
    description: "Stripe payment  API key",
    tags: ["payment", "api"],
    createdAt: "2024-01-10T09:00:00Z",
    updatedAt: "2024-01-18T16:45:00Z",
    expiresAt: "2024-06-30T23:59:59Z",
    accessCount: 128,
    lastAccessed: "2024-01-22T11:30:00Z",
    isExpired: false,
  },
  {
    id: "secret_3",
    name: "jwt-signing-key",
    type: "certificate" as const,
    description: "JWT token certificate",
    tags: ["auth", "security"],
    createdAt: "2024-01-05T08:00:00Z",
    updatedAt: "2024-01-15T12:00:00Z",
    expiresAt: "2024-02-01T00:00:00Z",
    accessCount: 89,
    lastAccessed: "2024-01-21T14:20:00Z",
    isExpired: true,
  },
];

type SecretType = "api_key" | "password" | "certificate" | "connection_string" | "token";

/**
 * Vault & Secrets page component - Manage secrets, API keys, and certificates
 *
 * Features:
 * - Secret management with encryption
 * - Access tracking and audit logs
 * - Expiration monitoring
 * - Type-based categorization
 * - Secure viewing and copying
 */
export function VaultPage() {
  const navigate = useNavigate();
  const [searchQuery, setSearchQuery] = useState("");
  const [typeFilter, setTypeFilter] = useState<SecretType | "all">("all");
  const [statusFilter, setStatusFilter] = useState<"all" | "active" | "expired">("all");
  const [showAdvancedFilters, setShowAdvancedFilters] = useState(false);
  const [selectedSecrets, setSelectedSecrets] = useState<string[]>([]);
  const [visibleSecrets, setVisibleSecrets] = useState<Set<string>>(new Set());

  // Advanced filtering state
  const [tagsFilter, setTagsFilter] = useState("");
  const [expiryFilter, setExpiryFilter] = useState<string>("all");
  const [accessCountFilter, setAccessCountFilter] = useState<string>("all");

  // Filter secrets based on search and filters
  const filteredSecrets = useMemo(() => {
    return mockSecrets.filter((secret) => {
      const matchesSearch =
        secret.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
        secret.description.toLowerCase().includes(searchQuery.toLowerCase()) ||
        secret.tags.some((tag) => tag.toLowerCase().includes(searchQuery.toLowerCase()));

      const matchesType = typeFilter === "all" || secret.type === typeFilter;
      const matchesStatus =
        statusFilter === "all" ||
        (statusFilter === "expired" && secret.isExpired) ||
        (statusFilter === "active" && !secret.isExpired);

      const matchesTags =
        !tagsFilter ||
        secret.tags.some((tag) => tag.toLowerCase().includes(tagsFilter.toLowerCase()));

      const expiry = new Date(secret.expiresAt);
      const now = new Date();
      const daysToExpiry = Math.floor((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
      const matchesExpiry =
        expiryFilter === "all" ||
        (expiryFilter === "soon" && daysToExpiry <= 30 && daysToExpiry > 0) ||
        (expiryFilter === "expired" && secret.isExpired) ||
        (expiryFilter === "long" && daysToExpiry > 30);

      const matchesAccessCount =
        accessCountFilter === "all" ||
        (accessCountFilter === "high" && secret.accessCount > 100) ||
        (accessCountFilter === "medium" && secret.accessCount >= 50 && secret.accessCount <= 100) ||
        (accessCountFilter === "low" && secret.accessCount < 50);

      return (
        matchesSearch &&
        matchesType &&
        matchesStatus &&
        matchesTags &&
        matchesExpiry &&
        matchesAccessCount
      );
    });
  }, [searchQuery, typeFilter, statusFilter, tagsFilter, expiryFilter, accessCountFilter]);

  // Selection handlers
  const handleSelectAll = () => {
    if (selectedSecrets.length === filteredSecrets.length) {
      setSelectedSecrets([]);
    } else {
      setSelectedSecrets(filteredSecrets.map((s) => s.id));
    }
  };

  const handleSelectSecret = (secretId: string) => {
    setSelectedSecrets((prev) =>
      prev.includes(secretId) ? prev.filter((id) => id !== secretId) : [...prev, secretId]
    );
  };

  const handleClearSelection = () => {
    setSelectedSecrets([]);
  };

  // Reset filters
  const resetFilters = () => {
    setSearchQuery("");
    setTypeFilter("all");
    setStatusFilter("all");
    setTagsFilter("");
    setExpiryFilter("all");
    setAccessCountFilter("all");
  };

  // Bulk actions
  const handleBulkAction = (action: string) => {
    console.log(`Bulk ${action} for secrets:`, selectedSecrets);
    setSelectedSecrets([]);
  };

  // Secret management functions
  const handleEditSecret = (secretId: string) => {
    navigate(`/vault/${secretId}/edit`);
  };

  const handleCopySecret = (secretId: string) => {
    const secret = mockSecrets.find((s) => s.id === secretId);
    if (secret) {
      navigator.clipboard.writeText(`[SECRET_VALUE_FOR_${secret.name.toUpperCase()}]`);
    }
  };

  const handleDeleteSecret = (secretId: string) => {
    console.log("Delete secret:", secretId);
  };

  /**
   * Toggle secret visibility
   */
  const toggleSecretVisibility = (secretId: string) => {
    setVisibleSecrets((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(secretId)) {
        newSet.delete(secretId);
      } else {
        newSet.add(secretId);
      }
      return newSet;
    });
  };

  /**
   * Check if secret is expiring soon (within 30 days)
   */
  const isExpiringSoon = (expiresAt: string) => {
    const expiry = new Date(expiresAt);
    const now = new Date();
    const diffInDays = Math.floor((expiry.getTime() - now.getTime()) / (1000 * 60 * 60 * 24));
    return diffInDays <= 30 && diffInDays > 0;
  };

  return (
    <div className="min-h-screen flex flex-col p-6">
    <div className="space-y-6 overflow-hidden">
      {/* Header with Quick Actions */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Vault & Secrets</h1>
          <p className="text-foreground">
            Securely manage API keys, passwords, certificates, and other secrets
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={() => navigate("/vault/import")}>
            <Database className="h-4 w-4" />
            <span className="hidden lg:inline">Import Secrets</span>
          </Button>
        </div>
      </div>

      {/* Enhanced Stats Cards */}
      <MetricCardGrid>
        <MetricCard
          title="Total Secrets"
          value={filteredSecrets.length}
          colorVariant="blue"
          footer={{
            primary: `${filteredSecrets.filter((s) => !s.isExpired).length} active secrets`,
            secondary: "Secret management overview",
            icon: Key,
          }}
        />

        <MetricCard
          title="Expiring Soon"
          value={filteredSecrets.filter((s) => isExpiringSoon(s.expiresAt)).length}
          colorVariant="amber"
          trend={{
            value: filteredSecrets.filter((s) => isExpiringSoon(s.expiresAt)).length > 0 ? 15 : 0,
            isPositive: false,
          }}
          footer={{
            primary: "Within 30 days",
            secondary: "Requires attention",
            icon: AlertTriangle,
          }}
        />

        <MetricCard
          title="Expired"
          value={filteredSecrets.filter((s) => s.isExpired).length}
          colorVariant="red"
          footer={{
            primary: "Needs renewal",
            secondary: "Security risk",
            icon: Lock,
          }}
        />

        <MetricCard
          title="Total Access"
          value={filteredSecrets.reduce((acc, secret) => acc + secret.accessCount, 0)}
          colorVariant="green"
          footer={{
            primary: "All time accesses",
            secondary: "Usage analytics",
            icon: Eye,
          }}
        />
      </MetricCardGrid>

      {/* Search and Filters */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle className="text-base">Filters & Search</CardTitle>
              <CardDescription>Filter and search through secrets</CardDescription>
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setShowAdvancedFilters(!showAdvancedFilters)}
            >
              <Filter className="mr-2 h-4 w-4" />
              Advanced
            </Button>
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          {/* Primary Filters */}
          <div className="flex flex-col gap-4 md:flex-row md:items-center">
            <div className="flex-1">
              <div className="relative">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground" />
                <Input
                  placeholder="Search secrets by name, description, or tags..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="pl-10"
                />
              </div>
            </div>
            <div className="flex gap-2">
              <Select
                value={typeFilter}
                onValueChange={(value) => setTypeFilter(value as SecretType | "all")}
              >
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="Type" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Types</SelectItem>
                  <SelectItem value="api_key">API Key</SelectItem>
                  <SelectItem value="password">Password</SelectItem>
                  <SelectItem value="certificate">Certificate</SelectItem>
                  <SelectItem value="connection_string">Connection String</SelectItem>
                  <SelectItem value="token">Token</SelectItem>
                </SelectContent>
              </Select>

              <Select
                value={statusFilter}
                onValueChange={(value) => setStatusFilter(value as "all" | "active" | "expired")}
              >
                <SelectTrigger className="w-32">
                  <SelectValue placeholder="Status" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All Status</SelectItem>
                  <SelectItem value="active">Active</SelectItem>
                  <SelectItem value="expired">Expired</SelectItem>
                </SelectContent>
              </Select>

              <Button variant="outline" size="sm" onClick={resetFilters}>
                <RotateCcw className="h-4 w-4" />
              </Button>
            </div>
          </div>

          {/* Advanced Filters */}
          {showAdvancedFilters && (
            <div className="pt-4 border-t">
              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
                <div className="space-y-2">
                  <Label>Tags Filter</Label>
                  <Input
                    placeholder="Filter by tags..."
                    value={tagsFilter}
                    onChange={(e) => setTagsFilter(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Expiry Status</Label>
                  <Select value={expiryFilter} onValueChange={setExpiryFilter}>
                    <SelectTrigger>
                      <SelectValue placeholder="All Expiry" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Expiry</SelectItem>
                      <SelectItem value="soon">Expiring Soon</SelectItem>
                      <SelectItem value="expired">Expired</SelectItem>
                      <SelectItem value="long">Long Term</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Access Count</Label>
                  <Select value={accessCountFilter} onValueChange={setAccessCountFilter}>
                    <SelectTrigger>
                      <SelectValue placeholder="All Access" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="all">All Access</SelectItem>
                      <SelectItem value="high">High (100+)</SelectItem>
                      <SelectItem value="medium">Medium (50-100)</SelectItem>
                      <SelectItem value="low">Low (&lt;50)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Enhanced Vault Table */}
      <EnhancedVaultTable
        data={filteredSecrets}
        selectedSecrets={selectedSecrets}
        onSelectAll={handleSelectAll}
        onSelectSecret={handleSelectSecret}
        onEditSecret={handleEditSecret}
        onCopySecret={handleCopySecret}
        onDeleteSecret={handleDeleteSecret}
        onToggleVisibility={toggleSecretVisibility}
        visibleSecrets={visibleSecrets}
        onCreateSecret={() => navigate("/vault/create")}
      />

      {/* Bulk Actions Bar */}
      {selectedSecrets.length > 0 && (
        <BulkActionsBar
          selectedCount={selectedSecrets.length}
          onClearSelection={handleClearSelection}
          onBulkAction={handleBulkAction}
        />
      )}
    </div>
    </div>
  );
}
