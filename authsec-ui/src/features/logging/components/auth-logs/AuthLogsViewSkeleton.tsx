import { Skeleton } from "../../../../components/ui/skeleton";

export function AuthLogsViewSkeleton() {
  return (
    <div className="overflow-hidden border border-slate-200 dark:border-neutral-900 bg-white dark:bg-neutral-950 shadow-lg dark:shadow-[0_24px_40px_rgba(5,5,8,0.45)]">
      {/* Console Controls Header */}
      <div className="flex items-center justify-between px-6 py-4 bg-slate-100 dark:bg-neutral-950/90 border-b border-slate-200 dark:border-neutral-900">
        {/* Left side: Terminal icon + title + badge */}
        <div className="flex items-center gap-3">
          <Skeleton className="h-8 w-48 rounded-full" />
          <Skeleton className="h-5 w-24 rounded-full" />
        </div>

        {/* Right side: Control buttons */}
        <div className="flex items-center gap-2">
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-9 w-24" />
          <Skeleton className="h-9 w-24" />
        </div>
      </div>

      {/* Console Output Area */}
      <div className="bg-slate-50 dark:bg-neutral-950/70 border-b border-slate-200 dark:border-neutral-900">
        <div className="h-[600px] w-full overflow-hidden">
          <div className="p-6 space-y-4">
            {/* Render 10 log entry skeletons */}
            {Array.from({ length: 10 }).map((_, i) => (
              <div key={i} className="space-y-2">
                {/* Main log line */}
                <div className="flex items-start gap-3">
                  <Skeleton className="h-5 w-5 flex-shrink-0 rounded" />
                  <Skeleton
                    className="h-4"
                    style={{
                      width: `${60 + Math.random() * 30}%`,
                    }}
                  />
                </div>
                {/* Detail lines (indented) */}
                <div className="ml-8 space-y-1.5">
                  <Skeleton className="h-3 w-3/4" />
                  <Skeleton className="h-3 w-2/3" />
                  <Skeleton className="h-3 w-1/2" />
                </div>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Console Footer */}
      <div className="flex items-center justify-between px-6 py-3 bg-slate-100 dark:bg-neutral-950/90">
        <Skeleton className="h-4 w-32" />
        <Skeleton className="h-4 w-48" />
      </div>
    </div>
  );
}
