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
  RefreshCw,
  Download,
  Upload,
  Copy,
  Trash2,
  Lock,
  Unlock,
  Key,
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
            <Button variant="outline" size="sm" onClick={() => onBulkAction("rotate")}>
              <RefreshCw className="mr-2 h-4 w-4" />
              Rotate
            </Button>

            <Button variant="outline" size="sm" onClick={() => onBulkAction("lock")}>
              <Lock className="mr-2 h-4 w-4" />
              Lock
            </Button>

            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm">
                  More Actions
                  <ChevronDown className="ml-2 h-4 w-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem onClick={() => onBulkAction("unlock")}>
                  <Unlock className="mr-2 h-4 w-4" />
                  Unlock Secrets
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("copy")}>
                  <Copy className="mr-2 h-4 w-4" />
                  Copy Values
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("renew")}>
                  <Key className="mr-2 h-4 w-4" />
                  Renew Expiry
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem onClick={() => onBulkAction("export")}>
                  <Download className="mr-2 h-4 w-4" />
                  Export Secrets
                </DropdownMenuItem>
                <DropdownMenuItem onClick={() => onBulkAction("import")}>
                  <Upload className="mr-2 h-4 w-4" />
                  Import Secrets
                </DropdownMenuItem>
                <DropdownMenuSeparator />
                <DropdownMenuItem
                  onClick={() => onBulkAction("delete")}
                  className="text-destructive"
                >
                  <Trash2 className="mr-2 h-4 w-4" />
                  Delete Secrets
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          </div>
        </div>
      </div>
    </div>
  );
}
