import React, { forwardRef, type ReactNode, useState } from "react";

import { cn } from "@/lib/utils";
import { Label } from "@/components/ui/label";
import { Input, type InputProps } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Badge, type BadgeProps } from "@/components/ui/badge";
import { Button, type ButtonProps } from "@/components/ui/button";
import { Copy, Check } from "lucide-react";

type FormRootProps = React.ComponentPropsWithoutRef<"div"> & {
  maxWidth?: number | string;
};

export const FormRoot = forwardRef<HTMLDivElement, FormRootProps>(
  ({ className, maxWidth = "var(--component-form-max-width)", ...props }, ref) => (
    <div
      ref={ref}
      className={cn(
        "mx-auto w-full space-y-[var(--component-form-shell-gap)]",
        className
      )}
      style={{ maxWidth: typeof maxWidth === "number" ? `${maxWidth}px` : maxWidth }}
      {...props}
    />
  )
);

FormRoot.displayName = "FormRoot";

type FormHeaderProps = {
  title: ReactNode;
  description?: ReactNode;
  leading?: ReactNode;
  actions?: ReactNode;
  className?: string;
};

export function FormHeader({ title, description, leading, actions, className }: FormHeaderProps) {
  return (
    <header
      className={cn(
        "flex flex-col gap-4 md:flex-row md:items-start md:justify-between rounded-[var(--radius)] border border-[var(--component-form-section-border)] bg-[var(--component-card-background)] px-[var(--component-form-section-padding-inline)] py-[var(--component-form-section-padding-block)] shadow-[var(--component-card-shadow)] backdrop-blur-sm",
        className
      )}
    >
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:gap-6">
        {leading && <div className="shrink-0">{leading}</div>}
        <div className="space-y-2">
          <div className="text-[length:var(--font-size-heading-md)] font-[var(--font-weight-semibold)] tracking-[var(--letter-spacing-tight)] text-[color:var(--color-text-primary)]">
            {title}
          </div>
          {description && (
            <p className="text-[length:var(--font-size-body-sm)] text-[color:var(--color-text-secondary)] leading-[var(--line-height-body)]">
              {description}
            </p>
          )}
        </div>
      </div>
      {actions && <div className="flex items-center gap-2">{actions}</div>}
    </header>
  );
}

type FormBodyProps = React.ComponentPropsWithoutRef<"div">;

export function FormBody({ className, ...props }: FormBodyProps) {
  return <div className={cn("space-y-[var(--component-form-section-gap)]", className)} {...props} />;
}

type FormSectionProps = React.ComponentPropsWithoutRef<"section">;

export function FormSection({ className, ...props }: FormSectionProps) {
  return (
    <section
      className={cn(
        "rounded-[var(--component-card-radius)] border border-[var(--component-form-section-border)] bg-[var(--component-form-section-background)] shadow-[var(--component-form-section-shadow)] px-[var(--component-form-section-padding-inline)] py-[var(--component-form-section-padding-block)] space-y-[var(--component-form-field-gap)]",
        className
      )}
      {...props}
    />
  );
}

type FormSectionHeaderProps = {
  title: ReactNode;
  description?: ReactNode;
  action?: ReactNode;
  className?: string;
};

export function FormSectionHeader({ title, description, action, className }: FormSectionHeaderProps) {
  return (
    <div className={cn("flex flex-col gap-2 md:flex-row md:items-start md:justify-between", className)}>
      <div className="space-y-1">
        <h3 className="text-[length:var(--font-size-heading-sm)] font-[var(--font-weight-semibold)] tracking-[var(--letter-spacing-tight)] text-[color:var(--color-text-primary)]">
          {title}
        </h3>
        {description && (
          <p className="text-[length:var(--font-size-body-sm)] text-[color:var(--color-text-secondary)] leading-[var(--line-height-body)]">
            {description}
          </p>
        )}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  );
}

type FormFieldProps = {
  label: ReactNode;
  htmlFor?: string;
  description?: ReactNode;
  hint?: ReactNode;
  required?: boolean;
  actions?: ReactNode;
  className?: string;
  children: ReactNode;
};

export function FormField({
  label,
  htmlFor,
  description,
  hint,
  required,
  actions,
  className,
  children,
}: FormFieldProps) {
  return (
    <div className={cn("space-y-[calc(var(--component-form-field-gap)_-_.25rem)]", className)}>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex items-center gap-2 text-[color:var(--component-form-label-color)]">
          <Label
            htmlFor={htmlFor}
            className="text-[length:var(--font-size-body-sm)] font-[var(--component-form-label-weight)]"
          >
            {label}
            {required && <span className="ml-1 text-[color:var(--color-danger)]">*</span>}
          </Label>
        </div>
        {actions}
      </div>
      {description && (
        <p className="text-[length:var(--font-size-body-xs)] text-[color:var(--component-form-helper-color)] leading-[var(--line-height-body)]">
          {description}
        </p>
      )}
      <div className="space-y-2">{children}</div>
      {hint && (
        <p className="text-[length:var(--font-size-body-xs)] text-[color:var(--component-form-helper-color)] leading-[var(--line-height-body)]">
          {hint}
        </p>
      )}
    </div>
  );
}

type FormActionsProps = React.ComponentPropsWithoutRef<"div">;

export function FormActions({ className, ...props }: FormActionsProps) {
  return (
    <div
      className={cn(
        "sticky bottom-0 flex flex-col gap-3 border-t border-[color-mix(in_oklab,var(--component-form-section-border)_75%,transparent)] bg-[var(--component-form-actions-background)] px-[var(--component-form-section-padding-inline)] py-[var(--space-4)] shadow-[0_-8px_24px_-20px_rgba(15,23,42,0.35)] sm:flex-row sm:items-center sm:justify-end sm:gap-2",
        className
      )}
      {...props}
    />
  );
}

type FormGridProps = React.ComponentPropsWithoutRef<"div"> & {
  columns?: 1 | 2;
};

export function FormGrid({ className, columns = 2, ...props }: FormGridProps) {
  return (
    <div
      className={cn(
        "grid gap-[var(--component-form-field-gap)]",
        columns === 1 ? "grid-cols-1" : "grid-cols-1 md:grid-cols-2",
        className
      )}
      {...props}
    />
  );
}

export function FormDivider() {
  return <div className="border-t border-[var(--component-form-section-border)]" />;
}

type FormCalloutTone = "info" | "success" | "danger" | "neutral";

type FormCalloutProps = {
  icon?: ReactNode;
  title?: ReactNode;
  description?: ReactNode;
  className?: string;
  actions?: ReactNode;
  tone?: FormCalloutTone;
};

const toneContainerClass: Record<FormCalloutTone, string> = {
  info: "bg-[var(--component-form-callout-background)] border-[var(--component-form-callout-border)]",
  neutral: "bg-black/[0.02] dark:bg-white/[0.02] border-black/10 dark:border-white/10",
  success:
    "bg-[color-mix(in_oklab,var(--color-success) 12%,transparent)] border-[color-mix(in_oklab,var(--color-success) 45%,transparent)]",
  danger:
    "bg-[color-mix(in_oklab,var(--color-danger) 12%,transparent)] border-[color-mix(in_oklab,var(--color-danger) 45%,transparent)]",
};

const toneIconClass: Record<FormCalloutTone, string> = {
  info: "bg-[var(--component-form-callout-icon-bg)] text-[var(--component-form-callout-icon-color)]",
  neutral: "bg-black/5 dark:bg-white/10 text-foreground",
  success: "bg-[color-mix(in_oklab,var(--color-success) 25%,transparent)] text-[var(--color-success)]",
  danger: "bg-[color-mix(in_oklab,var(--color-danger) 25%,transparent)] text-[var(--color-danger)]",
};

export function FormCallout({ icon, title, description, actions, className, tone = "info" }: FormCalloutProps) {
  return (
    <div
      className={cn(
        "flex flex-col gap-3 rounded-[var(--radius)] border px-5 py-4",
        toneContainerClass[tone],
        className
      )}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex gap-3">
          {icon && (
            <div
              className={cn(
                "mt-1 flex h-9 w-9 shrink-0 items-center justify-center rounded-full",
                toneIconClass[tone]
              )}
            >
              {icon}
            </div>
          )}
          <div className="space-y-1">
            {title && (
              <div className="text-[length:var(--font-size-body-sm)] font-[var(--font-weight-semibold)] text-[color:var(--color-text-primary)]">
                {title}
              </div>
            )}
            {description && (
              <p className="text-[length:var(--font-size-body-xs)] leading-[var(--line-height-body)] text-[color:var(--component-form-helper-color)]">
                {description}
              </p>
            )}
          </div>
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

// ============================================================================
// STANDARDIZED FORM COMPONENTS - Centralized Aesthetics System
// ============================================================================

/**
 * FormInput - Standardized input with consistent height and styling
 * Usage: <FormInput placeholder="..." value={...} onChange={...} />
 */
export const FormInput = forwardRef<HTMLInputElement, InputProps>(
  ({ className, ...props }, ref) => (
    <Input
      ref={ref}
      className={cn("h-12", className)}
      {...props}
    />
  )
);

FormInput.displayName = "FormInput";

/**
 * FormSelect - Standardized select wrapper (exports Select directly)
 */
export { Select as FormSelect };

/**
 * FormSelectTrigger - Standardized select trigger with consistent height
 */
export const FormSelectTrigger = forwardRef<
  React.ElementRef<typeof SelectTrigger>,
  React.ComponentPropsWithoutRef<typeof SelectTrigger>
>(({ className, ...props }, ref) => (
  <SelectTrigger
    ref={ref}
    className={cn("h-12 w-full max-w-full border-[var(--component-form-field-border)] bg-black/[0.02] dark:bg-white/[0.02] text-left", className)}
    {...props}
  />
));

FormSelectTrigger.displayName = "FormSelectTrigger";

/**
 * FormSelectContent, FormSelectItem, FormSelectValue - Re-exports for convenience
 */
export { SelectContent as FormSelectContent, SelectItem as FormSelectItem, SelectValue as FormSelectValue };

/**
 * FormBadge - Compact badge with form-appropriate styling
 * Variants:
 * - 'scope': Blue themed badge for OAuth scopes
 * - 'default', 'outline', 'secondary': Standard badge variants with compact sizing
 *
 * Usage: <FormBadge variant="scope">openid</FormBadge>
 */
type FormBadgeVariant = "default" | "outline" | "secondary" | "scope";

interface FormBadgeProps extends Omit<BadgeProps, "variant"> {
  variant?: FormBadgeVariant;
}

export const FormBadge = forwardRef<HTMLDivElement, FormBadgeProps>(
  ({ variant = "default", className, children, ...props }, ref) => {
    const scopeStyles = "bg-blue-50/80 text-blue-700 border-blue-200/70 dark:bg-blue-950/20 dark:text-blue-300 dark:border-blue-800/50";

    return (
      <Badge
        ref={ref}
        variant={variant === "scope" ? "outline" : variant}
        className={cn(
          "text-xs font-medium px-2 py-0.5",
          variant === "scope" && scopeStyles,
          className
        )}
        {...props}
      >
        {children}
      </Badge>
    );
  }
);

FormBadge.displayName = "FormBadge";

/**
 * FormBadgeGroup - Container for badge groups with consistent spacing
 * Automatically handles empty states
 *
 * Usage:
 * <FormBadgeGroup emptyText="No scopes selected">
 *   {scopes.map(s => <FormBadge key={s} variant="scope">{s}</FormBadge>)}
 * </FormBadgeGroup>
 */
interface FormBadgeGroupProps extends React.ComponentPropsWithoutRef<"div"> {
  emptyText?: string;
}

export const FormBadgeGroup = forwardRef<HTMLDivElement, FormBadgeGroupProps>(
  ({ children, emptyText = "No items", className, ...props }, ref) => {
    const hasChildren = React.Children.count(children) > 0;

    return (
      <div
        ref={ref}
        className={cn(
          "flex flex-wrap gap-1.5 rounded-[var(--radius)] border border-black/10 dark:border-white/10 bg-black/[0.02] dark:bg-white/[0.02] p-4 min-h-[3rem]",
          className
        )}
        {...props}
      >
        {hasChildren ? (
          children
        ) : (
          <span className="text-sm text-foreground">{emptyText}</span>
        )}
      </div>
    );
  }
);

FormBadgeGroup.displayName = "FormBadgeGroup";

/**
 * FormCopyField - Input with integrated copy button
 * Handles copy functionality and visual feedback automatically
 *
 * Usage:
 * <FormCopyField
 *   value="https://example.com/callback"
 *   label="Callback URL"
 *   onCopy={() => toast.success("Copied!")}
 * />
 */
interface FormCopyFieldProps extends Omit<InputProps, "readOnly"> {
  onCopy?: () => void;
  label?: string;
}

export const FormCopyField = forwardRef<HTMLInputElement, FormCopyFieldProps>(
  ({ value, onCopy, label, className, ...props }, ref) => {
    const [copied, setCopied] = useState(false);

    const handleCopy = async () => {
      if (typeof value === "string") {
        try {
          await navigator.clipboard.writeText(value);
          setCopied(true);
          onCopy?.();
          setTimeout(() => setCopied(false), 2000);
        } catch (error) {
          console.error("Failed to copy:", error);
        }
      }
    };

    return (
      <div className="flex flex-col gap-3 sm:flex-row sm:items-stretch">
        <FormInput
          ref={ref}
          value={value}
          readOnly
          className={cn("font-mono", className)}
          {...props}
        />
        <Button
          type="button"
          variant="outline"
          onClick={handleCopy}
          className="h-12 sm:w-auto sm:px-6"
        >
          {copied ? (
            <>
              <Check className="mr-2 h-4 w-4" /> Copied
            </>
          ) : (
            <>
              <Copy className="mr-2 h-4 w-4" /> Copy
            </>
          )}
        </Button>
      </div>
    );
  }
);

FormCopyField.displayName = "FormCopyField";

/**
 * FormButton - Standardized button for forms
 * Automatically has proper height to match form inputs
 *
 * Usage: <FormButton variant="outline">Cancel</FormButton>
 */
export const FormButton = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, ...props }, ref) => (
    <Button
      ref={ref}
      className={cn("h-12", className)}
      {...props}
    />
  )
);

FormButton.displayName = "FormButton";
