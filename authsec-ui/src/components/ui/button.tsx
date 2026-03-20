import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../../lib/utils";

const buttonVariants = cva(
  "inline-flex items-center justify-center gap-[var(--component-button-gap)] whitespace-nowrap rounded-[var(--component-button-radius)] text-[length:var(--font-size-body-sm)] font-[var(--component-button-font-weight)] transition-[background-color,color,border-color,box-shadow,transform] duration-[var(--motion-duration-base)] ease-[var(--motion-easing-standard)] disabled:pointer-events-none disabled:opacity-60 [&_svg]:pointer-events-none [&_svg:not([class*='size-'])]:size-4 shrink-0 [&_svg]:shrink-0 outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 focus-visible:ring-offset-background aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
  {
    variants: {
      variant: {
        default:
          "bg-[var(--component-button-primary-bg)] text-white shadow-[var(--shadow-xs)] hover:bg-[var(--component-button-primary-hover-bg)] [&_svg]:text-white",
        destructive:
          "bg-[var(--component-button-destructive-bg)] text-white shadow-[var(--shadow-xs)] hover:bg-[var(--component-button-destructive-hover-bg)] focus-visible:ring-destructive/30 [&_svg]:text-white",
        outline:
          "border border-[var(--component-button-outline-border)] bg-[var(--background)] text-[var(--color-text-primary)] shadow-[var(--shadow-xs)] hover:bg-[var(--component-button-secondary-hover-bg)]",
        secondary:
          "bg-[var(--component-button-secondary-bg)] text-[var(--component-button-secondary-fg)] shadow-[var(--shadow-xs)] hover:bg-[var(--component-button-secondary-hover-bg)]",
        ghost:
          "text-[var(--color-text-primary)] hover:bg-[var(--color-surface-subtle)] hover:text-[var(--color-text-primary)]",
        link: "text-[var(--color-primary)] underline-offset-4 hover:underline",
      },
      size: {
        default:
          "min-h-[var(--component-button-height-md)] px-[var(--component-button-padding-inline-md)] py-[var(--component-button-padding-block)]",
        sm: "min-h-[var(--component-button-height-sm)] px-[var(--component-button-padding-inline-sm)] py-[var(--component-button-padding-block)] text-[var(--font-size-sm)]",
        lg: "min-h-[var(--component-button-height-lg)] px-[var(--component-button-padding-inline-lg)] py-[var(--component-button-padding-block)] text-[var(--font-size-md)]",
        icon: "size-[var(--component-button-height-md)]",
      },
    },
    defaultVariants: {
      variant: "default",
      size: "default",
    },
  }
)

type ButtonProps = React.ComponentPropsWithoutRef<"button"> &
  VariantProps<typeof buttonVariants> & {
    asChild?: boolean;
  };

function Button({
  className,
  variant,
  size,
  asChild = false,
  ...props
}: ButtonProps) {
  const Comp = asChild ? Slot : "button";
  const resolvedVariant = variant ?? "default";
  const resolvedSize = size ?? "default";

  return (
    <Comp
      data-slot="button"
      data-variant={resolvedVariant}
      data-size={resolvedSize}
      className={cn(buttonVariants({ variant: resolvedVariant, size: resolvedSize }), className)}
      {...props}
    />
  );
}

export { Button, buttonVariants };
export type { ButtonProps };
