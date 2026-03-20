import { Skeleton } from "@/components/ui/skeleton";

export function UsersTableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div className="space-y-5 p-6">
      <div className="flex items-center justify-between">
        <Skeleton className="h-6 w-40 bg-slate-200/70 dark:bg-neutral-700/70" />
        <Skeleton className="h-9 w-32 bg-slate-200/70 dark:bg-neutral-700/70" />
      </div>
      <div className="space-y-3">
        {Array.from({ length: rows }).map((_, index) => (
          <div
            key={index}
            className="flex items-center gap-3 rounded-xl border border-slate-200/60 bg-slate-50/60 p-3 dark:border-neutral-700/60 dark:bg-neutral-800/40"
          >
            <Skeleton className="h-5 w-5 rounded bg-slate-200/70 dark:bg-neutral-700/70" />
            <div className="flex flex-1 flex-col gap-2 sm:flex-row sm:items-center sm:gap-4">
              <Skeleton className="h-4 w-48 max-w-[220px] bg-slate-200/70 dark:bg-neutral-700/70" />
              <Skeleton className="h-4 w-64 max-w-[280px] bg-slate-200/70 dark:bg-neutral-700/70" />
            </div>
            <Skeleton className="hidden h-4 w-24 bg-slate-200/70 dark:bg-neutral-700/70 md:block" />
            <Skeleton className="hidden h-8 w-20 rounded bg-slate-200/70 dark:bg-neutral-700/70 lg:block" />
          </div>
        ))}
      </div>
    </div>
  );
}

