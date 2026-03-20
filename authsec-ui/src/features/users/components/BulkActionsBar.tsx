import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import {
  X,
  UserCheck,
  UserX,
  Shield,
  Trash2,
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
            <Button variant="outline" size="sm" onClick={() => onBulkAction("activate")}>
              <UserCheck className="mr-2 h-4 w-4" />
              Activate
            </Button>

            <Button variant="outline" size="sm" onClick={() => onBulkAction("deactivate")}>
              <UserX className="mr-2 h-4 w-4" />
              Deactivate
            </Button>

            <Button variant="outline" size="sm" onClick={() => onBulkAction("assign-role")}>
              <Shield className="mr-2 h-4 w-4" />
              Assign Role
            </Button>

            <Button
              variant="outline"
              size="sm"
              onClick={() => onBulkAction("delete")}
              className="text-destructive hover:text-destructive"
            >
              <Trash2 className="mr-2 h-4 w-4" />
              Delete
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
