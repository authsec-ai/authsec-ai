import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Skeleton } from "@/components/ui/skeleton";

export function ClientsPageSkeleton() {
  return (
    <div className="min-h-screen bg-gradient-to-br from-slate-50 via-white to-slate-100/50 dark:from-neutral-950 dark:via-neutral-900 dark:to-stone-950">
      <div className="space-y-8 p-6 max-w-7xl mx-auto">
        {/* Header Card with Modern Design */}
        <Card className="border-0 bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm ring-1 ring-slate-200/50 dark:ring-neutral-800/50 p-0">
          <CardContent className="p-8">
            {/* Top row - Welcome and Actions */}
            <div className="flex flex-col lg:flex-row lg:items-start lg:justify-between gap-8">
              <div className="flex items-start gap-5">
                <div className="relative">
                  <div className="absolute inset-0 bg-slate-200/50 dark:bg-neutral-700/30 rounded-xl blur-sm"></div>
                  <div className="relative p-4 bg-white dark:bg-neutral-800/80 rounded-xl shadow-sm ring-1 ring-slate-200/50 dark:ring-neutral-700/50">
                    <Skeleton className="h-7 w-7" />
                  </div>
                </div>
                <div className="space-y-3">
                  <Skeleton className="h-8 w-48" />
                  <Skeleton className="h-5 w-80" />
                </div>
              </div>
              <div className="flex flex-wrap gap-3">
                <Skeleton className="h-11 w-40" />
              </div>
            </div>

            {/* Bottom row - KPIs */}
            <div className="flex gap-8 pt-6">
              <div className="flex flex-col items-center justify-center px-6 pt-6 pb-0 min-w-[140px]">
                <div className="flex items-center gap-3 mb-2">
                  <Skeleton className="h-5 w-5" />
                  <Skeleton className="h-8 w-12" />
                </div>
                <Skeleton className="h-4 w-20" />
              </div>
              <div className="flex flex-col items-center justify-center px-6 pt-6 pb-0 min-w-[140px]">
                <div className="flex items-center gap-3 mb-2">
                  <Skeleton className="h-5 w-5" />
                  <Skeleton className="h-8 w-12" />
                </div>
                <Skeleton className="h-4 w-24" />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Filter/Search Card */}
        <Card className="border-0 bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm ring-1 ring-slate-200/50 dark:ring-neutral-800/50">
          <CardHeader className="pb-6">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <Skeleton className="w-2 h-8 rounded-full" />
                <div className="space-y-2">
                  <Skeleton className="h-6 w-32" />
                  <Skeleton className="h-4 w-56" />
                </div>
              </div>
              <Skeleton className="h-8 w-24" />
            </div>
          </CardHeader>
          <CardContent className="space-y-4">
            <div className="flex flex-col gap-4 md:flex-row md:items-center">
              <div className="flex-1">
                <Skeleton className="h-10 w-full" />
              </div>
              <div className="flex gap-2">
                <Skeleton className="h-10 w-32" />
                <Skeleton className="h-10 w-32" />
                <Skeleton className="h-10 w-32" />
                <Skeleton className="h-10 w-10" />
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Table Card */}
        <Card className="border-0 bg-white/80 dark:bg-neutral-900/80 backdrop-blur-sm ring-1 ring-slate-200/50 dark:ring-neutral-800/50 p-0">
          <CardContent className="p-0">
            <div className="space-y-4 p-6">
              {/* Table Header */}
              <div className="grid grid-cols-8 gap-4 py-3 border-b border-slate-200/50 dark:border-neutral-700/50">
                <Skeleton className="h-4 w-4" />
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-24" />
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-16" />
                <Skeleton className="h-4 w-20" />
                <Skeleton className="h-4 w-16" />
              </div>

              {/* Table Rows */}
              {Array.from({ length: 8 }).map((_, i) => (
                <div key={i} className="grid grid-cols-8 gap-4 py-4 border-b last:border-b-0 border-slate-200/30 dark:border-neutral-700/30">
                  <Skeleton className="h-4 w-4" />
                  <div className="space-y-1">
                    <Skeleton className="h-4 w-24" />
                    <Skeleton className="h-3 w-32" />
                  </div>
                  <Skeleton className="h-6 w-16 rounded-full" />
                  <div className="space-y-1">
                    <Skeleton className="h-4 w-20" />
                    <Skeleton className="h-3 w-24" />
                  </div>
                  <Skeleton className="h-4 w-16" />
                  <Skeleton className="h-4 w-20" />
                  <Skeleton className="h-4 w-16" />
                  <div className="flex gap-1">
                    <Skeleton className="h-8 w-8" />
                    <Skeleton className="h-8 w-8" />
                    <Skeleton className="h-8 w-8" />
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>

        {/* Pagination Skeleton */}
        <div className="flex items-center justify-between">
          <Skeleton className="h-4 w-48" />
          <div className="flex items-center gap-2">
            <Skeleton className="h-8 w-20" />
            <div className="flex items-center gap-1">
              {Array.from({ length: 5 }).map((_, i) => (
                <Skeleton key={i} className="h-8 w-8" />
              ))}
            </div>
            <Skeleton className="h-8 w-16" />
            <Skeleton className="h-8 w-16" />
          </div>
        </div>
      </div>
    </div>
  );
}

export function ClientsTableSkeleton() {
  return (
    <div className="space-y-4 p-6">
      {/* Table Header */}
      <div className="grid grid-cols-8 gap-4 py-3 border-b border-slate-200/50 dark:border-neutral-700/50">
        <Skeleton className="h-4 w-4" />
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-4 w-16" />
        <Skeleton className="h-4 w-20" />
        <Skeleton className="h-4 w-16" />
      </div>

      {/* Table Rows */}
      {Array.from({ length: 10 }).map((_, i) => (
        <div
          key={i}
          className="grid grid-cols-8 gap-4 py-4 border-b last:border-b-0 border-slate-200/30 dark:border-neutral-700/30"
        >
          <Skeleton className="h-4 w-4" />
          <div className="space-y-1">
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-3 w-32" />
          </div>
          <Skeleton className="h-6 w-16 rounded-full" />
          <div className="space-y-1">
            <Skeleton className="h-4 w-20" />
            <Skeleton className="h-3 w-24" />
          </div>
          <Skeleton className="h-4 w-16" />
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-4 w-16" />
          <div className="flex gap-1">
            <Skeleton className="h-8 w-8" />
            <Skeleton className="h-8 w-8" />
            <Skeleton className="h-8 w-8" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function ClientsKPISkeleton() {
  return (
    <div className="flex gap-8">
      <div className="flex flex-col items-center justify-center px-6 pt-6 pb-0 min-w-[140px]">
        <div className="flex items-center gap-3 mb-2">
          <Skeleton className="h-5 w-5" />
          <Skeleton className="h-8 w-12" />
        </div>
        <Skeleton className="h-4 w-20" />
      </div>
      <div className="flex flex-col items-center justify-center px-6 pt-6 pb-0 min-w-[140px]">
        <div className="flex items-center gap-3 mb-2">
          <Skeleton className="h-5 w-5" />
          <Skeleton className="h-8 w-12" />
        </div>
        <Skeleton className="h-4 w-24" />
      </div>
    </div>
  );
}

// Refreshing overlay skeleton for when data is being refreshed
export function RefreshingSkeleton() {
  return (
    <div className="absolute inset-0 bg-white/90 dark:bg-neutral-900/90 backdrop-blur-sm flex items-center justify-center z-10 rounded-lg">
      <div className="flex items-center gap-3 text-sm text-slate-600 dark:text-neutral-400 bg-white dark:bg-neutral-800 px-4 py-3 rounded-lg shadow-lg ring-1 ring-slate-200 dark:ring-neutral-700">
        <div className="animate-spin rounded-full h-5 w-5 border-2 border-slate-300 dark:border-neutral-600 border-t-slate-600 dark:border-t-neutral-300"></div>
        <span className="font-medium">Refreshing clients...</span>
      </div>
    </div>
  );
}
