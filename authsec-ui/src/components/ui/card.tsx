import * as React from "react";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../../lib/utils";

const cardVariants = cva(
  "flex flex-col gap-[var(--component-card-gap)] rounded-[var(--component-card-radius)] border border-[var(--component-card-border)] bg-[var(--component-card-background)] text-[var(--component-card-foreground)] shadow-[var(--component-card-shadow)] transition-shadow duration-[var(--motion-duration-base)] ease-[var(--motion-easing-standard)]",
  {
    variants: {
      variant: {
        default: "",
        header:
          "bg-[var(--component-card-variant-header-background)] border-[var(--component-card-variant-header-border)] shadow-[var(--component-card-variant-header-shadow)]",
        filter:
          "bg-[var(--component-card-variant-filter-background)] border-[var(--component-card-variant-filter-border)] shadow-[var(--component-card-variant-filter-shadow)]",
        table:
          "bg-[var(--component-card-variant-table-background)] border-[var(--component-card-variant-table-border)] shadow-[var(--component-card-variant-table-shadow)]",
        pagination:
          "bg-[var(--component-card-variant-pagination-background)] border-[var(--component-card-variant-pagination-border)] shadow-[var(--component-card-variant-pagination-shadow)]",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

const cardContentVariants = cva(
  "px-[var(--component-card-padding-inline)] py-[var(--component-card-padding-block)]",
  {
    variants: {
      variant: {
        default: "",
        flush: "px-0 py-0",
        compact:
          "px-[var(--component-card-content-compact-inline)] py-[var(--component-card-content-compact-block)]",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

type CardVariant = VariantProps<typeof cardVariants>["variant"];
type CardContentVariant = VariantProps<typeof cardContentVariants>["variant"];

type CardProps = React.ComponentPropsWithoutRef<"div"> & {
  variant?: CardVariant;
};

const Card = React.forwardRef<HTMLDivElement, CardProps>(
  ({ className, variant = "default", ...props }, ref) => {
    return (
      <div
        ref={ref}
        data-slot="card"
        data-variant={variant}
        className={cn(cardVariants({ variant }), className)}
        {...props}
      />
    );
  }
);

Card.displayName = "Card";

function CardHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-header"
      className={cn(
        "@container/card-header grid auto-rows-min grid-rows-[auto_auto] items-start gap-1.5 px-[var(--component-card-header-padding-inline)] py-[var(--component-card-header-padding-block)] has-data-[slot=card-action]:grid-cols-[1fr_auto] [.border-b]:pb-[var(--component-card-header-padding-block)]",
        className
      )}
      {...props}
    />
  );
}

function CardTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-title"
      className={cn(
        "font-[var(--font-weight-semibold)] text-[length:var(--font-size-heading-sm)] leading-[var(--line-height-heading)] tracking-[var(--letter-spacing-tight)] text-[color:var(--color-text-primary)]",
        className
      )}
      {...props}
    />
  );
}

function CardDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-description"
      className={cn(
        "text-[length:var(--font-size-body-sm)] leading-[var(--line-height-body)] text-[color:var(--color-text-secondary)]",
        className
      )}
      {...props}
    />
  );
}

function CardAction({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-action"
      className={cn("col-start-2 row-span-2 row-start-1 self-start justify-self-end", className)}
      {...props}
    />
  );
}

type CardContentProps = React.ComponentPropsWithoutRef<"div"> & {
  variant?: CardContentVariant;
};

function CardContent({ className, variant = "default", ...props }: CardContentProps) {
  return (
    <div
      data-slot="card-content"
      className={cn(cardContentVariants({ variant }), className)}
      {...props}
    />
  );
}

function CardFooter({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="card-footer"
      className={cn(
        "flex items-center px-[var(--component-card-footer-padding-inline)] py-[var(--component-card-footer-padding-block)] [.border-t]:pt-[var(--component-card-footer-padding-block)]",
        className
      )}
      {...props}
    />
  );
}

export {
  Card,
  CardHeader,
  CardFooter,
  CardTitle,
  CardAction,
  CardDescription,
  CardContent,
  cardVariants,
  cardContentVariants,
};
export type { CardProps, CardVariant, CardContentProps };
