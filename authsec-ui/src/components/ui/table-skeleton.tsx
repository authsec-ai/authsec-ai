import { Skeleton } from "@/components/ui/skeleton";
import { cn } from "@/lib/utils";

interface DataTableSkeletonProps {
  columns?: number;
  rows?: number;
  showSelection?: boolean;
  showActions?: boolean;
  className?: string;
}

const COLUMN_WIDTHS = [
  "w-40",
  "w-48",
  "w-36",
  "w-32",
  "w-44",
];

export function DataTableSkeleton({
  columns = 5,
  rows = 6,
  showSelection = true,
  showActions = true,
  className,
}: DataTableSkeletonProps) {
  return (
    <div className={cn("space-y-4", className)}>
      {/* Header skeleton */}
      <div className="flex items-center gap-3 rounded-sm border border-border/60 bg-white/75 dark:bg-neutral-900/75 px-4 py-3">
        {showSelection && <Skeleton className="h-4 w-4" />}
        {Array.from({ length: columns }).map((_, idx) => (
          <Skeleton
            key={`header-${idx}`}
            className={cn("h-4", COLUMN_WIDTHS[idx % COLUMN_WIDTHS.length])}
          />
        ))}
        {showActions && <Skeleton className="h-4 w-12" />}
      </div>

      {/* Rows skeleton */}
      <div className="space-y-2 rounded-sm border border-border/60 bg-white/65 dark:bg-neutral-900/65 px-4 py-4">
        {Array.from({ length: rows }).map((_, rowIdx) => (
          <div
            key={`row-${rowIdx}`}
            className="flex items-center gap-4 border-b border-border/40 pb-3 last:border-b-0 last:pb-0"
          >
            {showSelection && <Skeleton className="h-4 w-4" />}
            {Array.from({ length: columns }).map((__, colIdx) => (
              <Skeleton
                key={`cell-${rowIdx}-${colIdx}`}
                className={cn(
                  "h-4 flex-1",
                  COLUMN_WIDTHS[colIdx % COLUMN_WIDTHS.length],
                  "max-w-[220px]"
                )}
              />
            ))}
            {showActions && <Skeleton className="h-8 w-16" />}
          </div>
        ))}
      </div>

      {/* Pagination skeleton */}
      <div className="flex items-center justify-between rounded-sm border border-border/60 bg-white/75 dark:bg-neutral-900/75 px-4 py-3">
        <Skeleton className="h-4 w-56" />
        <div className="flex items-center gap-2">
          <Skeleton className="h-8 w-8 rounded-full" />
          <Skeleton className="h-8 w-8 rounded-full" />
          <Skeleton className="h-8 w-8 rounded-full" />
        </div>
      </div>
    </div>
  );
}
