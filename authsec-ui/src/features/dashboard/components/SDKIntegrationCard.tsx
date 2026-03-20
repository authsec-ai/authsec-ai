import React from "react";
import { Button } from "../../../components/ui/button";
import { Badge } from "../../../components/ui/badge";
import { ArrowRight, Lock } from "lucide-react";

interface SDKIntegrationCardProps {
  icon: React.ComponentType<{ className?: string }>;
  title: string;
  description: string;
  ctaLabel: string;
  onClick: () => void;
  gradient: string;
  iconBg: string;
  iconColor: string;
  badge?: string;
  badgeColor?: "success" | "warning" | "info" | "default";
  disabled?: boolean;
  comingSoon?: boolean;
}

export function SDKIntegrationCard({
  icon: Icon,
  title,
  description,
  ctaLabel,
  onClick,
  gradient: _gradient,
  iconBg,
  iconColor,
  badge,
  badgeColor = "default",
  disabled = false,
  comingSoon = false,
}: SDKIntegrationCardProps) {
  const badgeVariants = {
    success: "bg-emerald-500/10 text-emerald-700 dark:bg-emerald-400/20 dark:text-emerald-300 border-emerald-200 dark:border-emerald-800",
    warning: "bg-amber-500/10 text-amber-700 dark:bg-amber-400/20 dark:text-amber-300 border-amber-200 dark:border-amber-800",
    info: "bg-blue-500/10 text-blue-700 dark:bg-blue-400/20 dark:text-blue-300 border-blue-200 dark:border-blue-800",
    default: "bg-slate-500/10 text-slate-700 dark:bg-slate-400/20 dark:text-slate-300 border-slate-200 dark:border-slate-700",
  };

  return (
    <div
      className="relative h-full"
    >
      <div
        className={`dash-panel relative overflow-hidden p-6 h-full flex flex-col ${
          disabled ? "opacity-60 cursor-not-allowed" : "dash-panel-interactive"
        }`}
      >
        {/* Badge */}
        {(badge || comingSoon) && (
          <div className="absolute top-4 right-4">
            {comingSoon ? (
              <Badge
                variant="outline"
                className="bg-slate-500/10 text-slate-700 dark:bg-slate-400/20 dark:text-slate-300 border-slate-200 dark:border-slate-700"
              >
                Coming Soon
              </Badge>
            ) : badge ? (
              <Badge
                variant="outline"
                className={`text-xs font-medium ${badgeVariants[badgeColor]}`}
              >
                {badge}
              </Badge>
            ) : null}
          </div>
        )}

        {/* Icon */}
        <div className="relative mb-4">
          <div
            className={`p-3 ${iconBg} rounded-[var(--dash-radius-md)] border border-[var(--dash-border-soft)] w-fit`}
          >
            <Icon className={`h-6 w-6 ${iconColor}`} />
            {disabled && (
              <div className="absolute inset-0 flex items-center justify-center">
                <Lock className="h-4 w-4 text-slate-400" />
              </div>
            )}
          </div>
        </div>

        {/* Content */}
        <div className="flex-1 mb-6">
          <h3 className="text-lg font-semibold text-foreground mb-2">
            {title}
          </h3>
          <p className="text-sm text-foreground leading-relaxed">
            {description}
          </p>
        </div>

        {/* CTA Button */}
        <Button
          onClick={onClick}
          disabled={disabled}
          variant="outline"
          className="w-full dash-btn-secondary group"
        >
          {ctaLabel}
          {!disabled && (
            <ArrowRight className="h-4 w-4 ml-2 group-hover:translate-x-1 transition-transform" />
          )}
        </Button>

        {/* Decorative Circle */}
        <div className="absolute top-0 right-0 w-24 h-24 rounded-full -mr-12 -mt-12 pointer-events-none border border-[var(--dash-border-soft)] bg-[var(--dash-surface-panel-alt)] opacity-50" />
      </div>
    </div>
  );
}
