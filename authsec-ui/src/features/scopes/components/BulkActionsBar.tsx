import { FloatingBulkBar } from "@/components/shared/FloatingBulkBar";

interface BulkActionsBarProps {
  selectedCount: number;
  onBulkAction: (action: string) => void;
  onClearSelection: () => void;
}

export function BulkActionsBar({
  selectedCount,
  onBulkAction,
  onClearSelection,
}: BulkActionsBarProps) {
  return (
    <FloatingBulkBar
      selectedCount={selectedCount}
      confirmLabel="Delete"
      onConfirm={() => onBulkAction("delete")}
      onClear={onClearSelection}
    />
  );
}
