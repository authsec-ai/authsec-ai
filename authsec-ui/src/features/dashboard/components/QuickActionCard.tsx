import React from "react";
import { useNavigate } from "react-router-dom";
import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";

interface QuickActionCardProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  ctaLabel: string;
  ctaLink: string;
  navigationState?: any;
  statusBadge?: string;
  statusColor?: "success" | "warning" | "info" | "default";
  gradient: string;
  iconBg: string;
  iconColor: string;
  secondaryAction?: {
    label: string;
    link: string;
  };
  onCustomAction?: () => void;
}

export function QuickActionCard({
  icon: Icon,
  title,
  description,
  ctaLabel,
  ctaLink,
  navigationState,
  statusBadge,
  statusColor = "default",
  gradient: _gradient,
  iconBg,
  iconColor,
  secondaryAction,
  onCustomAction,
}: QuickActionCardProps) {
  const navigate = useNavigate();

  const handlePrimaryClick = () => {
    if (onCustomAction) {
      onCustomAction();
    } else {
      navigate(ctaLink, navigationState ? { state: navigationState } : undefined);
    }
  };

  const handleSecondaryClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (secondaryAction) {
      navigate(secondaryAction.link);
    }
  };

  const statusVariants = {
    success: "bg-emerald-500/10 text-emerald-700 dark:bg-emerald-400/20 dark:text-emerald-300",
    warning: "bg-amber-500/10 text-amber-700 dark:bg-amber-400/20 dark:text-amber-300",
    info: "bg-blue-500/10 text-blue-700 dark:bg-blue-400/20 dark:text-blue-300",
    default: "bg-slate-500/10 text-slate-700 dark:bg-slate-400/20 dark:text-slate-300",
  };

  return (
    <div
      className="relative h-full"
    >
      <div
        className="dash-panel dash-panel-interactive relative overflow-hidden p-6 cursor-pointer h-full flex flex-col"
        onClick={handlePrimaryClick}
      >
        {/* Status Badge */}
        {statusBadge && (
          <div className="absolute top-4 right-4">
            <Badge
              variant="outline"
              className={`text-xs font-medium ${statusVariants[statusColor]}`}
            >
              {statusBadge}
            </Badge>
          </div>
        )}

        {/* Icon */}
        <div
          className={`p-3 ${iconBg} rounded-[var(--dash-radius-md)] border border-[var(--dash-border-soft)] w-fit mb-4`}
        >
          <Icon className={`h-6 w-6 ${iconColor}`} />
        </div>

        {/* Content */}
        <div className="flex-1 mb-4 min-h-[80px]">
          <h3 className="text-lg font-semibold text-foreground mb-2">
            {title}
          </h3>
          <p className="text-sm text-foreground leading-relaxed line-clamp-3">
            {description}
          </p>
        </div>

        {/* Actions */}
        <div className="flex flex-col gap-2 min-h-[76px]">
          <Button
            variant="outline"
            className="w-full dash-btn-secondary"
            onClick={handlePrimaryClick}
          >
            {ctaLabel}
          </Button>

          {secondaryAction ? (
            <Button
              variant="ghost"
              size="sm"
              className="w-full text-xs dash-btn-inline"
              onClick={handleSecondaryClick}
            >
              {secondaryAction.label}
            </Button>
          ) : (
            <div className="h-8" />
          )}
        </div>

        {/* Decorative Circle */}
        <div className="absolute top-0 right-0 w-24 h-24 rounded-full -mr-12 -mt-12 pointer-events-none border border-[var(--dash-border-soft)] bg-[var(--dash-surface-panel-alt)] opacity-50" />
      </div>
    </div>
  );
}
