import type { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

export type DashboardTileVariant = "panel" | "alt" | "active";

interface DashboardTileProps extends HTMLAttributes<HTMLDivElement> {
  variant?: DashboardTileVariant;
  interactive?: boolean;
}

export function DashboardTile({
  variant = "panel",
  interactive = false,
  className,
  ...props
}: DashboardTileProps) {
  return (
    <div
      className={cn(
        variant === "panel" && "dash-panel",
        variant === "alt" && "dash-panel-alt",
        variant === "active" && "dash-panel-active",
        interactive && "dash-panel-interactive dash-anim",
        className,
      )}
      {...props}
    />
  );
}

