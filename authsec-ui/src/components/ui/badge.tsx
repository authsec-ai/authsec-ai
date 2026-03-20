import * as React from "react";
import { Slot } from "@radix-ui/react-slot";
import { cva, type VariantProps } from "class-variance-authority";

import { cn } from "../../lib/utils";

const badgeVariants = cva(
  "inline-flex items-center justify-center text-xs font-medium w-fit whitespace-nowrap shrink-0 gap-1.5 transition-[color] duration-[var(--motion-duration-fast)] ease-[var(--motion-easing-standard)] [&>svg]:size-3 [&>svg]:pointer-events-none",
  {
    variants: {
      variant: {
        default:
          "text-[var(--color-text-primary)]",
        secondary:
          "text-[var(--color-text-primary)]",
        destructive:
          "text-[var(--color-danger)]",
        outline:
          "text-[var(--color-text-primary)]",
      },
    },
    defaultVariants: {
      variant: "default",
    },
  }
);

function Badge({
  className,
  variant,
  asChild = false,
  ...props
}: React.ComponentProps<"span"> &
  VariantProps<typeof badgeVariants> & { asChild?: boolean }) {
  const Comp = asChild ? Slot : "span";
  const resolvedVariant = variant ?? "default";

  return (
    <Comp
      data-slot="badge"
      data-variant={resolvedVariant}
      className={cn(badgeVariants({ variant: resolvedVariant }), className)}
      {...props}
    />
  );
}

export { Badge, badgeVariants };
