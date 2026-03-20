import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "../../../components/ui/dropdown-menu";
import {
  X,
  ChevronDown,
  Settings,
  Power,
  PowerOff,
  Copy,
  Trash2,
  Download,
  Shield,
  Key,
  Users,
  ShieldCheck,
  UserCheck,
  FileKey,
  RotateCcw,
  Activity,
} from "lucide-react";

interface BulkActionsBarProps {
  selectedCount: number;
  onClearSelection: () => void;
  onBulkAction: (action: string) => void;
}

export function BulkActionsBar({
  selectedCount,
  onClearSelection,
  onBulkAction,
}: BulkActionsBarProps) {
  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50">
      <div className="bg-card border border-border rounded-lg shadow-lg p-4 min-w-[400px]">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-3">
            <Badge variant="secondary" className="px-3 py-1">
              {selectedCount} selected
            </Badge>
            <Button variant="ghost" size="sm" onClick={onClearSelection} className="h-8 w-8 p-0">
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="flex items-center gap-2">
            {/* Primary Quick Actions */}
            <Button variant="outline" size="sm" onClick={() => onBulkAction("enable")}>
              <Power className="mr-2 h-4 w-4" />
              Enable
            </Button>

            <Button variant="outline" size="sm" onClick={() => onBulkAction("disable")}>
              <PowerOff className="mr-2 h-4 w-4" />
              Disable
            </Button>

            <Button variant="outline" size="sm" onClick={() => onBulkAction("bulk-assign-roles")}>
              <UserCheck className="mr-2 h-4 w-4" />
              Assign Roles
            </Button>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm">
                  Security Actions
                  <ChevronDown className="ml-2 h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end" className="w-56">
                {/* Security Operations */}
                <DropdownMenuItem onClick={() => onBulkAction("bulk-configure-mfa")}>
                  <ShieldCheck className="mr-2 h-4 w-4" />
                  Configure MFA
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("bulk-apply-policies")}>
                  <FileKey className="mr-2 h-4 w-4" />
                  Apply Policies
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("bulk-update-auth-type")}>
                  <Shield className="mr-2 h-4 w-4" />
                  Update Auth Type
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("bulk-reset-auth-stats")}>
                  <RotateCcw className="mr-2 h-4 w-4" />
                  Reset Auth Stats
                </DropdownMenuItem>
                
                <DropdownMenuSeparator />
                
                {/* Management Operations */}
                <DropdownMenuItem onClick={() => onBulkAction("bulk-duplicate")}>
                  <Copy className="mr-2 h-4 w-4" />
                  Duplicate Clients
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("bulk-export-config")}>
                  <Download className="mr-2 h-4 w-4" />
                  Export Configuration
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("bulk-security-report")}>
                  <Activity className="mr-2 h-4 w-4" />
                  Generate Security Report
                </DropdownMenuItem>
                
                <DropdownMenuSeparator />
                
                {/* Destructive Operations */}
                <DropdownMenuItem
                  onClick={() => onBulkAction("bulk-delete")}
                  className="text-destructive focus:text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete Clients
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </div>
  );
}
