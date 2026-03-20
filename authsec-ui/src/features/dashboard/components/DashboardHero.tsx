import { ArrowRight, CheckCircle2, X } from "lucide-react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DashboardStatusTag } from "./DashboardStatusTag";
import { DashboardTile } from "./DashboardTile";

export interface DashboardHeroStep {
  id: number | string;
  title: string;
  description: string;
  done: boolean;
}

interface DashboardHeroProps {
  title: string;
  subtitle: string;
  steps: DashboardHeroStep[];
  completed: number;
  total: number;
  complete?: boolean;
  completeTitle?: string;
  completeDescription?: string;
  primaryActionLabel?: string;
  onPrimaryAction?: () => void;
  primaryActionDisabled?: boolean;
  secondaryActionLabel?: string;
  onSecondaryAction?: () => void;
  onDismiss?: () => void;
}

export function DashboardHero({
  title,
  subtitle,
  steps,
  completed,
  total,
  complete = false,
  completeTitle,
  completeDescription,
  primaryActionLabel,
  onPrimaryAction,
  primaryActionDisabled = false,
  secondaryActionLabel,
  onSecondaryAction,
  onDismiss,
}: DashboardHeroProps) {
  const percentage = total > 0 ? Math.round((completed / total) * 100) : 0;

  if (complete) {
    return (
      <DashboardTile variant="panel" className="p-4 sm:p-5">
        <div className="flex items-start justify-between gap-3">
          <div className="flex min-w-0 items-start gap-3">
            <div className="dash-icon-chip" data-tone="success">
              <CheckCircle2 className="h-4 w-4" />
            </div>
            <div className="min-w-0 space-y-1">
              <div className="flex flex-wrap items-center gap-2">
                <h3 className="text-sm font-semibold dash-text-1">
                  {completeTitle ?? title}
                </h3>
                <DashboardStatusTag label="Complete" tone="success" />
              </div>
              <p className="text-xs dash-text-2">
                {completeDescription ?? subtitle}
              </p>
              <div className="text-[11px] dash-text-3">
                Setup progress {completed}/{total}
              </div>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {secondaryActionLabel && onSecondaryAction && (
              <Button
                variant="ghost"
                size="sm"
                className="dash-btn-inline"
                onClick={onSecondaryAction}
              >
                {secondaryActionLabel}
              </Button>
            )}
            {onDismiss && (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 rounded-md dash-btn-inline p-0"
                onClick={onDismiss}
                aria-label="Dismiss activation card"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>
      </DashboardTile>
    );
  }

  return (
    <DashboardTile variant="active" className="p-4 sm:p-5 lg:p-6">
      <div className="space-y-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="min-w-0 space-y-1.5">
            <div className="flex flex-wrap items-center gap-2">
              <h3 className="text-base font-semibold dash-text-1">{title}</h3>
              <DashboardStatusTag
                label={completed === 0 ? "Ready" : "In Progress"}
                tone={completed === 0 ? "neutral" : "accent"}
              />
            </div>
            <p className="text-sm dash-text-2">{subtitle}</p>
          </div>

          <div className="ml-auto flex items-center gap-2">
            {primaryActionLabel && onPrimaryAction && (
              <Button
                size="sm"
                className="bg-[var(--brand-blue-600)] text-white hover:bg-[var(--brand-blue-700)] min-w-[8.5rem]"
                onClick={onPrimaryAction}
                disabled={primaryActionDisabled}
              >
                {primaryActionLabel}
                {!primaryActionDisabled && <ArrowRight className="h-3.5 w-3.5" />}
              </Button>
            )}
            {secondaryActionLabel && onSecondaryAction && (
              <Button
                variant="ghost"
                size="sm"
                className="dash-btn-inline"
                data-tone="accent"
                onClick={onSecondaryAction}
              >
                {secondaryActionLabel}
              </Button>
            )}
            {onDismiss && (
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 rounded-md dash-btn-inline p-0"
                onClick={onDismiss}
                aria-label="Dismiss activation card"
              >
                <X className="h-4 w-4" />
              </Button>
            )}
          </div>
        </div>

        <div className="dash-panel-alt p-3">
          <div className="grid gap-2 sm:grid-cols-[auto_1fr_auto] sm:items-center">
            <span className="dash-kpi-label">Setup Progress</span>
            <div className="dash-progress-track">
              <div
                className="dash-progress-fill"
                style={{ width: `${percentage}%` }}
              />
            </div>
            <span className="text-xs font-semibold dash-text-2">
              {completed}/{total} ({percentage}%)
            </span>
          </div>
        </div>

        <div className="grid gap-2 lg:grid-cols-2">
          {steps.map((step) => (
            <div
              key={step.id}
              className={cn(
                "dash-panel p-3",
                step.done
                  ? "border-[var(--dash-border-soft)]"
                  : "border-[var(--dash-border-strong)]",
              )}
            >
              <div className="flex items-start gap-2.5">
                <div className="mt-0.5 shrink-0">
                  {step.done ? (
                    <CheckCircle2 className="h-4 w-4 text-[var(--dash-success)]" />
                  ) : (
                    <div className="flex h-4 w-4 items-center justify-center rounded-full border border-[var(--dash-border-strong)] text-[10px] font-semibold dash-text-2">
                      {step.id}
                    </div>
                  )}
                </div>
                <div className="min-w-0">
                  <p className="text-sm font-medium dash-text-1">{step.title}</p>
                  <p className="text-xs dash-text-2 line-clamp-2">
                    {step.description}
                  </p>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </DashboardTile>
  );
}
