import { Skeleton } from "@/components/ui/skeleton";

export function AuthTableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="space-y-5 p-6">
      {/* Search/Filter Bar Skeleton */}
      <div className="flex items-center justify-between gap-3">
        <Skeleton className="h-10 flex-1 bg-slate-200/70 dark:bg-neutral-700/70" />
        <Skeleton className="h-10 w-32 bg-slate-200/70 dark:bg-neutral-700/70" />
        <Skeleton className="h-10 w-32 bg-slate-200/70 dark:bg-neutral-700/70" />
      </div>

      {/* Table Rows Skeleton */}
      <div className="space-y-3">
        {Array.from({ length: rows }).map((_, index) => (
          <div
            key={index}
            className="flex items-center gap-4 rounded-xl border border-slate-200/60 bg-slate-50/60 p-4 dark:border-neutral-700/60 dark:bg-neutral-800/40"
          >
            {/* Checkbox */}
            <Skeleton className="h-5 w-5 rounded bg-slate-200/70 dark:bg-neutral-700/70" />

            {/* Provider Icon + Name */}
            <div className="flex flex-1 items-center gap-3">
              <Skeleton className="h-10 w-10 rounded-full bg-slate-200/70 dark:bg-neutral-700/70" />
              <div className="flex flex-col gap-2 flex-1">
                <Skeleton className="h-4 w-40 max-w-[180px] bg-slate-200/70 dark:bg-neutral-700/70" />
                <Skeleton className="h-3 w-24 max-w-[100px] bg-slate-200/70 dark:bg-neutral-700/70" />
              </div>
            </div>

            {/* Status Badge */}
            <Skeleton className="hidden h-6 w-20 rounded-full bg-slate-200/70 dark:bg-neutral-700/70 md:block" />

            {/* Configuration */}
            <div className="hidden lg:flex flex-col gap-2">
              <Skeleton className="h-3 w-24 bg-slate-200/70 dark:bg-neutral-700/70" />
              <Skeleton className="h-3 w-32 bg-slate-200/70 dark:bg-neutral-700/70" />
            </div>

            {/* Endpoints */}
            <div className="hidden xl:flex flex-col gap-2">
              <Skeleton className="h-3 w-20 bg-slate-200/70 dark:bg-neutral-700/70" />
              <Skeleton className="h-3 w-28 bg-slate-200/70 dark:bg-neutral-700/70" />
            </div>

            {/* Actions */}
            <Skeleton className="h-8 w-8 rounded bg-slate-200/70 dark:bg-neutral-700/70" />
          </div>
        ))}
      </div>

      {/* Pagination Skeleton */}
      <div className="flex items-center justify-between pt-4">
        <Skeleton className="h-4 w-48 bg-slate-200/70 dark:bg-neutral-700/70" />
        <div className="flex items-center gap-2">
          <Skeleton className="h-9 w-24 bg-slate-200/70 dark:bg-neutral-700/70" />
          <Skeleton className="h-9 w-20 bg-slate-200/70 dark:bg-neutral-700/70" />
          <Skeleton className="h-9 w-24 bg-slate-200/70 dark:bg-neutral-700/70" />
        </div>
      </div>
    </div>
  );
}
