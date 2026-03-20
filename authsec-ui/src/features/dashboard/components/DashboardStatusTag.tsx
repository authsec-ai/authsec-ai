import { cn } from "@/lib/utils";

export type DashboardStatusTone =
  | "neutral"
  | "muted"
  | "accent"
  | "success"
  | "warning"
  | "danger";

interface DashboardStatusTagProps {
  label: string;
  tone?: DashboardStatusTone;
  className?: string;
}

export function DashboardStatusTag({
  label,
  tone = "neutral",
  className,
}: DashboardStatusTagProps) {
  return (
    <span className={cn("dash-status-tag", className)} data-tone={tone}>
      {label}
    </span>
  );
}

