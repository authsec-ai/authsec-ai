interface ProgressIndicatorProps {
  completed: number;
  total: number;
}

export function ProgressIndicator({ completed, total }: ProgressIndicatorProps) {
  const percentage = total > 0 ? (completed / total) * 100 : 0;

  return (
    <div className="flex items-center gap-2.5">
      <span className="text-sm font-medium text-foreground opacity-70">
        {completed}/{total}
      </span>
      <div className="h-1.5 w-20 rounded-full bg-[var(--color-surface-subtle)]">
        <div
          className="h-full rounded-full bg-[var(--color-primary)] transition-all duration-300"
          style={{ width: `${percentage}%` }}
        />
      </div>
    </div>
  );
}
