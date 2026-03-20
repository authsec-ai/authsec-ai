import React from "react";
import { Card } from "@/components/ui/card";
import { cn } from "@/lib/utils";

interface PageHeaderProps {
  title: string;
  description?: string;
  actions?: React.ReactNode;
  badge?: React.ReactNode;
  className?: string;
  /** Use "inline" for simple buttons on the right, "below" for complex actions like tabs spanning full width */
  actionsPosition?: "inline" | "below";
}

export function PageHeader({
  title,
  description,
  actions,
  badge,
  className,
  actionsPosition = "inline",
}: PageHeaderProps) {
  return (
    <Card
      variant="header"
      data-slot="page-shell-header"
      className={cn(
        "p-4 sm:p-6",
        className
      )}
    >
      <div
        data-slot="page-header"
        data-actions-position={actionsPosition}
        className="flex flex-col gap-4"
      >
        {/* Title row with optional inline actions */}
        <div
          data-slot="page-header-main"
          className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between"
        >
          <div data-slot="page-header-copy" className="min-w-0">
            <div data-slot="page-header-title-row" className="flex items-center gap-3">
              <h1
                data-slot="page-header-title"
                className="text-xl sm:text-2xl font-semibold tracking-tight text-foreground"
              >
                {title}
              </h1>
              {badge}
            </div>
            {description && (
              <p data-slot="page-header-description" className="mt-1 text-sm">
                {description}
              </p>
            )}
          </div>
          {/* Inline actions on the right */}
          {actions && actionsPosition === "inline" && (
            <div data-slot="page-header-actions" className="shrink-0">
              {actions}
            </div>
          )}
        </div>
        {/* Actions row - full width below title (for tabs, etc.) */}
        {actions && actionsPosition === "below" && (
          <div data-slot="page-header-actions" className="w-full">
            {actions}
          </div>
        )}
      </div>
    </Card>
  );
}
