import type { ComponentType, KeyboardEvent, MouseEvent } from "react";
import { ArrowRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DashboardStatusTag, type DashboardStatusTone } from "./DashboardStatusTag";
import { DashboardTile } from "./DashboardTile";

export type DashboardActionPriority = "primary" | "secondary" | "inline";
export type DashboardActionTileLayout = "stack" | "row";

interface DashboardActionTileProps {
  icon: ComponentType<{ className?: string }>;
  iconTone?: DashboardStatusTone;
  title: string;
  description: string;
  statusLabel?: string;
  statusTone?: DashboardStatusTone;
  meta?: string;
  primaryActionLabel: string;
  onPrimaryAction: () => void;
  primaryActionPriority?: DashboardActionPriority;
  secondaryActionLabel?: string;
  onSecondaryAction?: () => void;
  disabled?: boolean;
  layout?: DashboardActionTileLayout;
  framed?: boolean;
  revealActionsOnHover?: boolean;
  className?: string;
}

export function DashboardActionTile({
  icon: Icon,
  iconTone = "neutral",
  title,
  description,
  statusLabel,
  statusTone = "neutral",
  meta,
  primaryActionLabel,
  onPrimaryAction,
  primaryActionPriority = "secondary",
  secondaryActionLabel,
  onSecondaryAction,
  disabled = false,
  layout = "stack",
  framed = true,
  revealActionsOnHover = false,
  className,
}: DashboardActionTileProps) {
  const isClickable = !disabled;

  const isActionTarget = (target: EventTarget | null) =>
    target instanceof HTMLElement &&
    Boolean(
      target.closest(
        "button, a, input, textarea, select, [role='button'], [data-dash-action]",
      ),
    );

  const handleTileClick = (event: MouseEvent<HTMLDivElement>) => {
    if (!isClickable || isActionTarget(event.target)) return;
    onPrimaryAction();
  };

  const handleTileKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!isClickable || isActionTarget(event.target)) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onPrimaryAction();
    }
  };

  const primaryVariant =
    primaryActionPriority === "primary"
      ? "default"
      : primaryActionPriority === "secondary"
        ? "outline"
        : "ghost";

  const primaryClass =
    primaryActionPriority === "primary"
      ? "dash-btn-primary"
      : primaryActionPriority === "secondary"
        ? "dash-btn-secondary"
        : "dash-btn-inline";

  if (layout === "row") {
    const rowContent = (
      <div className="grid gap-3 md:grid-cols-[auto_minmax(0,1fr)_auto] md:items-center">
        <div className="dash-icon-chip" data-tone={iconTone}>
          <Icon className="h-4 w-4" />
        </div>

        <div className="min-w-0 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="text-sm font-semibold dash-text-1">{title}</h3>
            {statusLabel && (
              <DashboardStatusTag label={statusLabel} tone={statusTone} />
            )}
          </div>
          <p className="text-xs dash-text-2 line-clamp-1">{description}</p>
          {meta && <p className="text-[11px] dash-text-3 line-clamp-1">{meta}</p>}
        </div>

        <div
          className={cn(
            "flex items-center justify-start gap-1.5 md:justify-end",
            revealActionsOnHover &&
              !disabled &&
              "md:opacity-0 md:pointer-events-none md:translate-x-1 md:transition-[opacity,transform] md:duration-150 md:ease-out group-hover/dash-row:md:opacity-100 group-hover/dash-row:md:pointer-events-auto group-hover/dash-row:md:translate-x-0 group-focus-within/dash-row:md:opacity-100 group-focus-within/dash-row:md:pointer-events-auto group-focus-within/dash-row:md:translate-x-0",
          )}
        >
          <Button
            variant={primaryVariant}
            size="sm"
            data-dash-action
            className={cn(
              primaryClass,
              primaryActionPriority !== "inline" && "min-w-[6.75rem]",
            )}
            onClick={(event) => {
              event.stopPropagation();
              onPrimaryAction();
            }}
            disabled={disabled}
          >
            {primaryActionLabel}
            <ArrowRight className="h-3 w-3" />
          </Button>

          {secondaryActionLabel && onSecondaryAction && (
            <Button
              variant="ghost"
              size="sm"
              data-dash-action
              className="dash-btn-inline"
              onClick={(event) => {
                event.stopPropagation();
                onSecondaryAction?.();
              }}
              disabled={disabled}
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
          role={isClickable ? "button" : undefined}
          tabIndex={isClickable ? 0 : undefined}
          aria-disabled={disabled || undefined}
          onClick={handleTileClick}
          onKeyDown={handleTileKeyDown}
        >
          {rowContent}
        </div>
      );
    }

    return (
      <DashboardTile
        variant="panel"
        interactive={!disabled}
        className={cn("group/dash-row p-3", disabled && "pointer-events-none", className)}
        role={isClickable ? "button" : undefined}
        tabIndex={isClickable ? 0 : undefined}
        aria-disabled={disabled || undefined}
        onClick={handleTileClick}
        onKeyDown={handleTileKeyDown}
      >
        {rowContent}
      </DashboardTile>
    );
  }

  return (
    <DashboardTile
      variant="panel"
      interactive={!disabled}
      className={cn("h-full p-3", disabled && "pointer-events-none", className)}
      role={isClickable ? "button" : undefined}
      tabIndex={isClickable ? 0 : undefined}
      aria-disabled={disabled || undefined}
      onClick={handleTileClick}
      onKeyDown={handleTileKeyDown}
    >
      <div className="flex h-full flex-col gap-3">
        <div className="flex items-start justify-between gap-3">
          <div className="dash-icon-chip" data-tone={iconTone}>
            <Icon className="h-4 w-4" />
          </div>
          {statusLabel && (
            <DashboardStatusTag label={statusLabel} tone={statusTone} />
          )}
        </div>

        <div className="space-y-1">
          <h3 className="text-sm font-semibold dash-text-1">{title}</h3>
          <p className="text-xs dash-text-2 line-clamp-2">{description}</p>
          {meta && <p className="text-[11px] dash-text-3 line-clamp-1">{meta}</p>}
        </div>

        <div className="mt-auto flex items-center justify-between gap-2">
          <Button
            variant={primaryVariant}
            size="sm"
            data-dash-action
            className={cn(
              primaryClass,
              primaryActionPriority !== "inline" && "min-w-[7.5rem]",
            )}
            onClick={(event) => {
              event.stopPropagation();
              onPrimaryAction();
            }}
            disabled={disabled}
          >
            {primaryActionLabel}
            <ArrowRight className="h-3 w-3" />
          </Button>

          {secondaryActionLabel && onSecondaryAction && (
            <Button
              variant="ghost"
              size="sm"
              data-dash-action
              className="dash-btn-inline"
              onClick={(event) => {
                event.stopPropagation();
                onSecondaryAction?.();
              }}
              disabled={disabled}
            >
              {secondaryActionLabel}
            </Button>
          )}
        </div>
      </div>
    </DashboardTile>
  );
}
