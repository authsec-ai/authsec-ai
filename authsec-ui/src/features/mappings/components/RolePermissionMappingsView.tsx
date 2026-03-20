import { useMemo } from "react";
import { useGetPermissionsQuery } from "@/app/api/permissionsApi";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import { Card, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Loader2, Circle } from "lucide-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { TableCard } from "@/theme/components/cards";

interface RolePermissionMappingsViewProps {
  searchQuery: string;
}

export function RolePermissionMappingsView({ searchQuery }: RolePermissionMappingsViewProps) {
  const { audience } = useRbacAudience();

  // Fetch permissions
  const { data: permissions = [], isLoading: isLoadingPermissions } = useGetPermissionsQuery({
    audience,
  });

  // Group permissions by role
  const roleMappings = useMemo(() => {
    const mappings = new Map<string, string[]>();

    permissions.forEach((permission) => {
      const roles = permission.role_names || [];
      roles.forEach((role) => {
        if (!mappings.has(role)) {
          mappings.set(role, []);
        }
        mappings.get(role)!.push(permission.full_permission_string);
      });
    });

    return Array.from(mappings.entries()).map(([role, perms]) => ({
      role_name: role,
      permissions: perms,
    }));
  }, [permissions]);

  // Filter mappings based on search
  const filteredMappings = useMemo(() => {
    if (!searchQuery.trim()) return roleMappings;

    const query = searchQuery.toLowerCase();
    return roleMappings.filter(
      (mapping) =>
        mapping.role_name.toLowerCase().includes(query) ||
        mapping.permissions.some((p) => p.toLowerCase().includes(query))
    );
  }, [roleMappings, searchQuery]);

  if (isLoadingPermissions) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-16">
          <Loader2 className="h-8 w-8 animate-spin text-foreground" />
        </CardContent>
      </Card>
    );
  }

  if (roleMappings.length === 0) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center justify-center py-16 text-center">
          <Circle className="h-12 w-12 text-foreground mb-4" />
          <h3 className="text-lg font-semibold mb-2">No Role-Permission Mappings</h3>
          <p className="text-sm text-foreground max-w-md">
            Create roles with associated permissions to see mappings here.
          </p>
        </CardContent>
      </Card>
    );
  }

  return (
    <TableCard>
      <CardContent className="space-y-4">
        <div>
          <h3 className="text-lg font-semibold">Mappings</h3>
          <p className="text-sm text-foreground">
            {filteredMappings.length} role{filteredMappings.length !== 1 ? "s" : ""} found
          </p>
        </div>
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[250px]">Role Name</TableHead>
                <TableHead>Associated Permissions</TableHead>
                <TableHead className="text-right w-[120px]">Permission Count</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filteredMappings.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={3} className="py-10 text-center text-foreground">
                    No role-permission mappings found. Create roles with permissions to see mappings here.
                  </TableCell>
                </TableRow>
              ) : (
                filteredMappings.map((mapping) => (
                  <TableRow key={mapping.role_name}>
                    <TableCell>
                      <div className="font-mono font-medium">{mapping.role_name}</div>
                    </TableCell>
                    <TableCell>
                      <div className="flex flex-wrap gap-1.5">
                        {mapping.permissions.slice(0, 3).map((permission) => (
                          <Badge key={permission} variant="secondary" className="text-xs">
                            {permission}
                          </Badge>
                        ))}
                        {mapping.permissions.length > 3 && (
                          <Badge variant="outline" className="text-xs">
                            +{mapping.permissions.length - 3} more
                          </Badge>
                        )}
                      </div>
                    </TableCell>
                    <TableCell className="text-right">
                      <Badge variant="outline">{mapping.permissions.length}</Badge>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </TableCard>
  );
}
