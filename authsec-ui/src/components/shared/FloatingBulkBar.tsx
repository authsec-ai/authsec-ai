import React from "react";
import { Button } from "@/components/ui/button";
import { Trash2, X } from "lucide-react";

interface FloatingBulkBarProps {
  selectedCount: number;
  onConfirm: () => void;
  onClear: () => void;
  confirmLabel?: string;
}

export function FloatingBulkBar({
  selectedCount,
  onConfirm,
  onClear,
  confirmLabel = "Delete",
}: FloatingBulkBarProps) {
  if (selectedCount === 0) return null;

  return (
    <div className="fixed bottom-6 left-1/2 z-50 w-full max-w-md -translate-x-1/2 px-4">
      <div className="flex items-center justify-between gap-4 rounded-2xl bg-slate-900/95 px-5 py-4 text-white shadow-2xl ring-1 ring-slate-800 dark:bg-white/95 dark:text-slate-900 dark:ring-slate-200">
        <div className="flex items-center gap-2">
          <span className="text-base font-semibold">{selectedCount}</span>
          <span className="text-sm uppercase tracking-wide text-white/70 dark:text-slate-600">Selected</span>
        </div>

        <div className="flex items-center gap-3">
          <Button
            variant="ghost"
            size="sm"
            onClick={onConfirm}
            className="h-9 bg-white/10 text-white hover:bg-white/20 dark:bg-slate-900/10 dark:text-slate-900 dark:hover:bg-slate-900/20"
          >
            <Trash2 className="mr-2 h-4 w-4" />
            {confirmLabel}
          </Button>
          <Button
            variant="ghost"
            size="icon"
            onClick={onClear}
            className="h-9 w-9 bg-white/5 text-white hover:bg-white/15 dark:bg-slate-900/10 dark:text-slate-900 dark:hover:bg-slate-900/20"
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}
