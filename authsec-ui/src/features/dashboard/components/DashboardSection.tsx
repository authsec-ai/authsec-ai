import type { ReactNode } from "react";
import { cn } from "@/lib/utils";

interface DashboardSectionProps {
  label?: string;
  title?: string;
  description?: string;
  action?: ReactNode;
  className?: string;
  headerClassName?: string;
  contentClassName?: string;
  children: ReactNode;
}

export function DashboardSection({
  label,
  title,
  description,
  action,
  className,
  headerClassName,
  contentClassName,
  children,
}: DashboardSectionProps) {
  return (
    <section className={cn("space-y-2.5", className)}>
      {(label || title || description || action) && (
        <div
          className={cn(
            "flex items-start justify-between gap-3 px-1.5",
            headerClassName,
          )}
        >
          <div className="min-w-0 space-y-1">
            {label && <div className="dash-eyebrow">{label}</div>}
            {title && <h2 className="dash-section-title">{title}</h2>}
            {description && (
              <p className="dash-section-subtitle line-clamp-2">{description}</p>
            )}
          </div>
          {action && <div className="shrink-0">{action}</div>}
        </div>
      )}
      <div className={contentClassName}>{children}</div>
    </section>
  );
}
