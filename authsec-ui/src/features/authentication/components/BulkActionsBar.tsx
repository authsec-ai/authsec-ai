import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import { X, Power, PowerOff } from "lucide-react";
import type { ApiOidcProvider } from "../utils/oidc-provider-table-utils";

interface BulkActionsBarProps {
  selectedProviders: ApiOidcProvider[];
  onClearSelection: () => void;
  onBulkAction: (action: string) => void;
}

export function BulkActionsBar({
  selectedProviders,
  onClearSelection,
  onBulkAction,
}: BulkActionsBarProps) {
  if (selectedProviders.length === 0) return null;

  // Check if all selected providers are active or inactive
  const allActive = selectedProviders.every((p) => p.is_active);
  const allInactive = selectedProviders.every((p) => !p.is_active);
  const hasActiveAndInactive = !allActive && !allInactive;

  return (
    <div className="fixed bottom-8 left-1/2 z-50 -translate-x-1/2">
      <div className="flex items-center justify-center">
        <div className="inline-flex min-w-[420px] items-center justify-between gap-[var(--space-4)] rounded-[var(--component-card-radius)] border border-[var(--component-card-border)] bg-[color-mix(in oklab,var(--component-card-background) 95%,transparent)] px-[var(--space-5)] py-[var(--space-4)] shadow-[var(--shadow-lg)] backdrop-blur-sm">
          <div className="flex items-center gap-3">
            <Badge variant="secondary" className="px-3 py-1">
              {selectedProviders.length} selected
            </Badge>
            <Button variant="ghost" size="sm" onClick={onClearSelection} className="h-8 w-8 p-0">
              <X className="h-4 w-4" />
            </Button>
          </div>

          <div className="flex items-center gap-2">
            <Button
              onClick={() => onBulkAction("activate")}
              disabled={allActive}
              size="sm"
              variant="outline"
            >
              <Power className="mr-2 h-4 w-4" />
              Activate
            </Button>

            <Button
              onClick={() => onBulkAction("deactivate")}
              disabled={allInactive}
              size="sm"
              variant="outline"
            >
              <PowerOff className="mr-2 h-4 w-4" />
              Deactivate
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
