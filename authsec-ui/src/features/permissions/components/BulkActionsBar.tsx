import { FloatingBulkBar } from "@/components/shared/FloatingBulkBar";

interface BulkActionsBarProps {
  selectedCount: number;
  onDeleteSelected: () => void;
  onClearSelection: () => void;
}

export function BulkActionsBar({
  selectedCount,
  onDeleteSelected,
  onClearSelection,
}: BulkActionsBarProps) {
  return (
    <FloatingBulkBar
      selectedCount={selectedCount}
      confirmLabel="Delete"
      onConfirm={onDeleteSelected}
      onClear={onClearSelection}
    />
  );
}
