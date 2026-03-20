import type { ComponentType, KeyboardEvent, MouseEvent } from "react";
import { ArrowRight, CheckCircle2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DashboardStatusTag, type DashboardStatusTone } from "./DashboardStatusTag";
import { DashboardTile } from "./DashboardTile";

interface DashboardTaskRowProps {
  icon: ComponentType<{ className?: string }>;
  iconTone?: DashboardStatusTone;
  title: string;
  description: string;
  isCompleted?: boolean;
  statusLabel: string;
  statusTone: DashboardStatusTone;
  primaryActionLabel: string;
  onPrimaryAction: () => void;
  secondaryActionLabel?: string;
  onSecondaryAction?: () => void;
  framed?: boolean;
  primaryActionStyle?: "secondary" | "inline";
  revealActionsOnHover?: boolean;
  className?: string;
}

export function DashboardTaskRow({
  icon: Icon,
  iconTone = "neutral",
  title,
  description,
  isCompleted = false,
  statusLabel,
  statusTone,
  primaryActionLabel,
  onPrimaryAction,
  secondaryActionLabel,
  onSecondaryAction,
  framed = true,
  primaryActionStyle = "secondary",
  revealActionsOnHover = false,
  className,
}: DashboardTaskRowProps) {
  const isActionTarget = (
    target: EventTarget | null,
    currentTarget: HTMLDivElement,
  ) => {
    if (!(target instanceof HTMLElement)) return false;

    const interactiveAncestor = target.closest(
      "button, a, input, textarea, select, [role='button'], [data-dash-action]",
    );

    return Boolean(interactiveAncestor && interactiveAncestor !== currentTarget);
  };

  const handleRowClick = (event: MouseEvent<HTMLDivElement>) => {
    if (isActionTarget(event.target, event.currentTarget)) return;
    onPrimaryAction();
  };

  const handleRowKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (isActionTarget(event.target, event.currentTarget)) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onPrimaryAction();
    }
  };

  const primaryClass =
    primaryActionStyle === "inline" ? "dash-btn-inline" : "dash-btn-secondary";
  const primaryVariant = primaryActionStyle === "inline" ? "ghost" : "outline";

  const rowContent = (
    <div className="grid gap-3 md:grid-cols-[auto_minmax(0,1fr)_auto] md:items-center">
      <div className="dash-icon-chip" data-tone={iconTone}>
        <Icon className="h-4 w-4" />
      </div>

      <div className="min-w-0 space-y-1">
        <div className="flex flex-wrap items-center gap-2">
          <h3 className="text-sm font-semibold dash-text-1">{title}</h3>
          <DashboardStatusTag label={statusLabel} tone={statusTone} />
          {isCompleted && (
            <CheckCircle2 className="h-4 w-4 text-[var(--dash-success)]" />
          )}
        </div>
        <p className="text-xs dash-text-2 line-clamp-1">{description}</p>
      </div>

      <div
        className={cn(
          "flex items-center justify-start gap-1.5 md:justify-end",
          revealActionsOnHover &&
            "md:opacity-0 md:pointer-events-none md:translate-x-1 md:transition-[opacity,transform] md:duration-150 md:ease-out group-hover/dash-row:md:opacity-100 group-hover/dash-row:md:pointer-events-auto group-hover/dash-row:md:translate-x-0 group-focus-within/dash-row:md:opacity-100 group-focus-within/dash-row:md:pointer-events-auto group-focus-within/dash-row:md:translate-x-0",
        )}
      >
        <Button
          size="sm"
          variant={primaryVariant}
          data-dash-action
          className={cn(
            primaryClass,
            primaryActionStyle !== "inline" && "min-w-[6.5rem]",
          )}
          onClick={(event) => {
            event.stopPropagation();
            onPrimaryAction();
          }}
        >
          {primaryActionLabel}
          <ArrowRight className="h-3 w-3" />
        </Button>
        {secondaryActionLabel && onSecondaryAction && (
          <Button
            size="sm"
            variant="ghost"
            data-dash-action
            className="dash-btn-inline"
            onClick={(event) => {
              event.stopPropagation();
              onSecondaryAction?.();
            }}
          >
            {secondaryActionLabel}
          </Button>
        )}
      </div>
    </div>
  );

  if (!framed) {
    return (
      <div
        className={cn("group/dash-row dash-row-flat px-3 py-2.5", className)}
        role="button"
        tabIndex={0}
        onClick={handleRowClick}
        onKeyDown={handleRowKeyDown}
      >
        {rowContent}
      </div>
    );
  }

  return (
    <DashboardTile
      variant={isCompleted ? "alt" : "panel"}
      interactive
      className={cn("group/dash-row p-2.5", className)}
      role="button"
      tabIndex={0}
      onClick={handleRowClick}
      onKeyDown={handleRowKeyDown}
    >
      {rowContent}
    </DashboardTile>
  );
}
