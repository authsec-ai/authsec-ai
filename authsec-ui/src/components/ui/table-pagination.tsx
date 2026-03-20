import { Button } from "@/components/ui/button";
import { CardContent } from "@/components/ui/card";
import { PaginationCard } from "@/theme/components/cards";
import { cn } from "@/lib/utils";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface TablePaginationProps {
  currentPage: number;
  totalPages: number;
  pageSize: number;
  totalItems: number;
  startIndex: number;
  endIndex: number;
  onPageChange: (page: number) => void;
  onPageSizeChange: (size: number) => void;
  pageSizeOptions?: number[];
  alwaysVisible?: boolean;
  className?: string;
}

export function DataTablePagination({
  currentPage,
  totalPages,
  pageSize,
  totalItems,
  startIndex,
  endIndex,
  onPageChange,
  alwaysVisible = false,
  className,
}: TablePaginationProps) {
  const safeTotalPages = Math.max(totalPages, 1);

  if (safeTotalPages <= 1 && !alwaysVisible) return null;

  const handlePageChange = (page: number) => {
    const clamped = Math.min(Math.max(page, 1), safeTotalPages);
    onPageChange(clamped);
  };

  return (
    <div className={cn(className)}>
      <PaginationCard className="border-t border-x-0 border-b-0 rounded-none border-[var(--editorial-border-soft)]">
        <CardContent className="px-4 py-2.5 sm:px-5">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <span className="text-sm text-[var(--editorial-text-2)]">
              Showing <span className="font-medium">{startIndex + 1}</span> to{" "}
              <span className="font-medium">{endIndex}</span> of{" "}
              <span className="font-medium">{totalItems}</span> entries
            </span>

            <div className="flex flex-1 items-center justify-start gap-4 sm:justify-end">
              <div className="flex items-center gap-2">
                <span className="hidden text-sm text-[var(--editorial-text-2)] sm:inline">Page</span>
                <Button
                  variant="outline"
                  size="icon"
                  className="admin-icon-btn-subtle h-8 w-8 rounded-md shadow-none"
                  onClick={() => handlePageChange(currentPage - 1)}
                  disabled={currentPage === 1}
                  aria-label="Go to previous page"
                >
                  <ChevronLeft className="h-4 w-4" />
                </Button>

                <span className="text-sm font-medium text-[var(--editorial-text-1)]">
                  {Math.min(currentPage, safeTotalPages)}
                </span>
                <span className="text-sm text-[var(--editorial-text-2)]">/ {safeTotalPages}</span>

                <Button
                  variant="outline"
                  size="icon"
                  className="admin-icon-btn-subtle h-8 w-8 rounded-md shadow-none"
                  onClick={() => handlePageChange(currentPage + 1)}
                  disabled={currentPage === safeTotalPages}
                  aria-label="Go to next page"
                >
                  <ChevronRight className="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        </CardContent>
      </PaginationCard>
    </div>
  );
}
