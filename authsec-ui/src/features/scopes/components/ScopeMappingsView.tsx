import { useState, useMemo } from "react";
import { useGetScopeMappingsQuery } from "@/app/api/scopesApi";
import { useGetPermissionResourcesQuery } from "@/app/api/permissionsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Loader2, Download, Search, CheckCircle2, Circle } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";

export function ScopeMappingsView() {
  const { audience } = useRbacAudience();
  const [searchQuery, setSearchQuery] = useState("");

  // Fetch scope mappings
  const { data: scopeMappings = [], isLoading: isLoadingMappings } = useGetScopeMappingsQuery();

  // Fetch all available resources
  const { data: allResources = [], isLoading: isLoadingResources } = useGetPermissionResourcesQuery({
    audience,
  });

  // Filter mappings based on search
  const filteredMappings = useMemo(() => {
    if (!searchQuery.trim()) return scopeMappings;

    const query = searchQuery.toLowerCase();
    return scopeMappings.filter(
      (mapping) =>
        mapping.scope_name.toLowerCase().includes(query) ||
        mapping.resources.some((r) => r.toLowerCase().includes(query))
    );
  }, [scopeMappings, searchQuery]);

  // Get unique resources from mappings
  const mappedResources = useMemo(() => {
    const resourceSet = new Set<string>();
    filteredMappings.forEach((mapping) => {
      mapping.resources.forEach((resource) => resourceSet.add(resource));
    });
    return Array.from(resourceSet).sort();
  }, [filteredMappings]);

  // Export mappings as CSV
  const handleExportCSV = () => {
    if (filteredMappings.length === 0) return;

    // Create CSV header
    const headers = ["Scope Name", "Resources", "Resource Count"];
    const csvRows = [headers.join(",")];

    // Add data rows
    filteredMappings.forEach((mapping) => {
      const row = [
        mapping.scope_name,
        `"${mapping.resources.join(", ")}"`,
        mapping.resources.length,
      ];
      csvRows.push(row.join(","));
    });

    // Create and download file
    const csvContent = csvRows.join("\n");
    const blob = new Blob([csvContent], { type: "text/csv" });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `scope-mappings-${new Date().toISOString().split("T")[0]}.csv`;
    link.click();
    window.URL.revokeObjectURL(url);
  };

  if (isLoadingMappings || isLoadingResources) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-foreground" />
        </CardContent>
      </Card>
    );
  }

  if (scopeMappings.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <Circle className="h-12 w-12 text-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">No Scope Mappings</h3>
          <p className="text-sm text-foreground max-w-md">
            Create scopes with associated resources to see mappings here.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      {/* Header with search and export */}
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Scope-Resource Mappings</CardTitle>
              <CardDescription>
                Visual representation of which resources are associated with each scope
              </CardDescription>
            </div>
            <Button onClick={handleExportCSV} variant="outline" size="sm">
              <Download className="mr-2 h-4 w-4" />
              Export CSV
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-foreground" />
            <Input
              placeholder="Search scopes or resources..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="pl-10"
            />
          </div>
        </CardContent>
      </Card>

      {/* Stats Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Total Scopes</CardDescription>
            <CardTitle className="text-3xl">{filteredMappings.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Unique Resources</CardDescription>
            <CardTitle className="text-3xl">{mappedResources.length}</CardTitle>
          </CardHeader>
        </Card>
        <Card>
          <CardHeader className="pb-3">
            <CardDescription>Avg Resources per Scope</CardDescription>
            <CardTitle className="text-3xl">
              {filteredMappings.length > 0
                ? (
                    filteredMappings.reduce((sum, m) => sum + m.resources.length, 0) /
                    filteredMappings.length
                  ).toFixed(1)
                : "0"}
            </CardTitle>
          </CardHeader>
        </Card>
      </div>

      {/* Mappings Table */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Scope Mappings</CardTitle>
          <CardDescription>
            {filteredMappings.length} scope{filteredMappings.length !== 1 ? "s" : ""} found
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[250px]">Scope Name</TableHead>
                  <TableHead>Associated Resources</TableHead>
                  <TableHead className="text-right w-[120px]">Resource Count</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredMappings.map((mapping) => (
                  <TableRow key={mapping.scope_name}>
                    <TableCell>
                      <div className="font-mono font-medium">{mapping.scope_name}</div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-2 text-sm text-foreground">
                        {mapping.resources.map((resource) => (
                          <span key={resource} className="font-mono leading-tight" title={resource}>
                            {resource}
                          </span>
                        ))}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <span className="text-sm text-foreground">
                        {mapping.resources.length}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>

      {/* Matrix View */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Resource Matrix</CardTitle>
          <CardDescription>
            Check marks indicate which resources are included in each scope
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="overflow-x-auto-hidden">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="sticky left-0 bg-background z-10 w-[200px]">
                    Scope / Resource
                  </TableHead>
                  {mappedResources.map((resource) => (
                    <TableHead key={resource} className="text-center min-w-[120px]">
                      <div className="font-mono text-xs">{resource}</div>
                    </TableHead>
                  ))}
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredMappings.map((mapping) => (
                  <TableRow key={mapping.scope_name}>
                    <TableCell className="sticky left-0 bg-background z-10 font-mono font-medium">
                      {mapping.scope_name}
                    </TableCell>
                    {mappedResources.map((resource) => (
                      <TableCell key={resource} className="text-center">
                        {mapping.resources.includes(resource) ? (
                          <CheckCircle2 className="h-5 w-5 text-green-600 mx-auto" />
                        ) : (
                          <Circle className="h-5 w-5 text-foreground/20 mx-auto" />
                        )}
                      </TableCell>
                    ))}
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
