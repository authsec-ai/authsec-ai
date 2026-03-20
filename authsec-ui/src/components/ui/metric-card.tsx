import * as React from "react";
import { IconTrendingUp, IconTrendingDown } from "@tabler/icons-react";
import type { LucideIcon } from "lucide-react";

import { cn } from "../../lib/utils";
import { Card, CardHeader, CardDescription, CardContent } from "./card";
import { Badge } from "./badge";

type ColorVariant = "default" | "blue" | "green" | "amber" | "purple" | "red" | "cyan";

interface MetricCardProps {
  title: string;
  value: string | number;
  description?: string;
  icon?: LucideIcon;
  trend?: {
    value: number;
    isPositive?: boolean;
    label?: string;
  };
  footer?: {
    primary: string;
    secondary?: string;
    icon?: LucideIcon;
  };
  onClick?: () => void;
  className?: string;
  colorVariant?: ColorVariant;
}

export function MetricCard({
  title,
  value,
  description: _description,
  icon: Icon,
  trend,
  footer,
  onClick,
  className,
  colorVariant = "default",
}: MetricCardProps) {
  const TrendIcon = trend?.isPositive !== false ? IconTrendingUp : IconTrendingDown;
  const FooterIcon = footer?.icon;

  return (
    <Card
      className={cn(
        "relative overflow-hidden border-l-4 border-l-black bg-gradient-to-br from-gray-100 to-transparent dark:from-white/10",
        onClick && "cursor-pointer hover:shadow-md transition-shadow",
        className
      )}
      onClick={onClick}
    >
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            {Icon && <Icon className="h-5 w-5 text-black dark:text-white" />}
            <CardDescription className="text-black dark:text-white font-medium">
              {title}
            </CardDescription>
          </div>
          {FooterIcon && <FooterIcon className="h-4 w-4 text-black dark:text-white" />}
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold text-black dark:text-white">{value}</div>
        {footer && (
          <>
            <div className="text-xs text-neutral-700 dark:text-neutral-300 mt-1">
              {footer.primary}
              {trend && (
                <>
                  {" • "}
                  <span
                    className={cn(
                      "inline-flex items-center",
                      trend.isPositive !== false
                        ? "text-green-600 dark:text-green-400"
                        : "text-red-600 dark:text-red-400"
                    )}
                  >
                    <TrendIcon className="w-3 h-3 mr-1" />
                    {trend.isPositive !== false ? "+" : ""}
                    {trend.value}%
                  </span>
                </>
              )}
            </div>
            {footer.secondary && (
              <div className="text-xs text-foreground mt-2">{footer.secondary}</div>
            )}
          </>
        )}
      </CardContent>
    </Card>
  );
}

interface MetricCardGridProps {
  children: React.ReactNode;
  enhanced?: boolean;
  className?: string;
}

export function MetricCardGrid({ children, enhanced = true, className }: MetricCardGridProps) {
  return (
    <div className={cn("grid gap-4 grid-cols-1 md:grid-cols-2 lg:grid-cols-4", className)}>
      {children}
    </div>
  );
}
