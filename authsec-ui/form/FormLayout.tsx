import * as React from "react";
import { cn } from "@/lib/utils";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Separator } from "@/components/ui/separator";

// Form Row - for horizontal field layouts
interface FormRowProps {
  children: React.ReactNode;
  columns?: 1 | 2 | 3 | 4;
  className?: string;
}

export function FormRow({ children, columns = 2, className }: FormRowProps) {
  return (
    <div
      className={cn(
        "grid gap-6",
        columns === 1 && "grid-cols-1",
        columns === 2 && "grid-cols-1 md:grid-cols-2",
        columns === 3 && "grid-cols-1 md:grid-cols-3",
        columns === 4 && "grid-cols-1 md:grid-cols-2 lg:grid-cols-4",
        className
      )}
    >
      {children}
    </div>
  );
}

// Form Section - groups related fields together
interface FormSectionProps {
  title?: string;
  description?: string;
  icon?: React.ReactNode;
  children: React.ReactNode;
  className?: string;
  variant?: "default" | "card" | "ghost";
  action?: React.ReactNode;
}

export function FormSection({
  title,
  description,
  icon,
  children,
  className,
  variant = "default",
  action,
}: FormSectionProps) {
  if (variant === "card") {
    return (
      <Card className={cn("border-muted/60 shadow-sm overflow-hidden", className)}>
        {(title || description) && (
          <CardHeader className="bg-muted/5 border-b border-border/50 pb-4">
            <div className="flex items-start justify-between gap-4">
              <div className="flex items-start gap-3">
                {icon && (
                  <div className="p-2 rounded-lg bg-primary/10 text-primary mt-0.5">
                    {icon}
                  </div>
                )}
                <div className="space-y-1">
                  {title && <CardTitle className="text-lg font-semibold tracking-tight">{title}</CardTitle>}
                  {description && (
                    <CardDescription className="text-sm text-muted-foreground/80 leading-relaxed">
                      {description}
                    </CardDescription>
                  )}
                </div>
              </div>
              {action && <div>{action}</div>}
            </div>
          </CardHeader>
        )}
        <CardContent className="p-6 space-y-6">{children}</CardContent>
      </Card>
    );
  }

  return (
    <div className={cn("space-y-6", className)}>
      {(title || description) && (
        <div className="space-y-2 pb-2">
          <div className="flex items-center justify-between">
            {title && (
              <div className="flex items-center gap-2">
                {icon && <span className="text-primary">{icon}</span>}
                <h3 className="text-lg font-semibold text-foreground tracking-tight">{title}</h3>
              </div>
            )}
            {action && <div>{action}</div>}
          </div>
          {description && <p className="text-sm text-muted-foreground/80 leading-relaxed max-w-3xl">{description}</p>}
          {variant !== "ghost" && <Separator className="mt-4" />}
        </div>
      )}
      <div className="space-y-6">{children}</div>
    </div>
  );
}

// Form Container - main wrapper for all forms
interface FormContainerProps extends React.ComponentProps<"form"> {
  children: React.ReactNode;
  spacing?: "compact" | "normal" | "relaxed";
  layout?: "default" | "centered";
}

export function FormContainer({
  children,
  spacing = "normal",
  layout = "default",
  className,
  ...formProps
}: FormContainerProps) {
  return (
    <form
      className={cn(
        spacing === "compact" && "space-y-4",
        spacing === "normal" && "space-y-8",
        spacing === "relaxed" && "space-y-10",
        layout === "centered" && "mx-auto max-w-2xl",
        className
      )}
      {...formProps}
    >
      {children}
    </form>
  );
}

// Form Actions - for submit/cancel buttons
interface FormActionsProps {
  children: React.ReactNode;
  align?: "left" | "right" | "center" | "between";
  className?: string;
  sticky?: boolean;
}

export function FormActions({ children, align = "right", className, sticky }: FormActionsProps) {
  return (
    <div
      className={cn(
        "flex gap-4 pt-6 mt-8 border-t",
        align === "left" && "justify-start",
        align === "right" && "justify-end",
        align === "center" && "justify-center",
        align === "between" && "justify-between",
        sticky && "sticky bottom-0 bg-background/95 backdrop-blur py-4 z-10 border-t shadow-sm -mx-6 px-6 -mb-6",
        className
      )}
    >
      {children}
    </div>
  );
}

// Form Grid - for complex layouts
interface FormGridProps {
  children: React.ReactNode;
  className?: string;
}

export function FormGrid({ children, className }: FormGridProps) {
  return <div className={cn("grid gap-8", className)}>{children}</div>;
}
