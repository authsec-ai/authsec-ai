import * as React from "react";

import { cn } from "../../lib/utils";

function Input({ className, type, ...props }: React.ComponentProps<"input">) {
  return (
    <input
      type={type}
      data-slot="input"
      className={cn(
        "flex h-10 w-full min-w-0 rounded-md border border-[var(--component-input-border)] bg-[var(--component-input-background)] px-3 py-2 text-base text-[var(--component-input-foreground)] shadow-[var(--component-input-shadow)] transition-[background-color,color,border-color,box-shadow] duration-[var(--motion-duration-fast)] ease-[var(--motion-easing-standard)] file:inline-flex file:h-7 file:items-center file:justify-center file:gap-2 file:rounded-md file:border-0 file:bg-transparent file:px-3 file:text-sm file:font-medium placeholder:text-[var(--component-input-placeholder)] selection:bg-[var(--component-button-primary-bg)] selection:text-[var(--component-button-primary-fg)] disabled:pointer-events-none disabled:cursor-not-allowed disabled:bg-[var(--component-input-disabled-background)] disabled:text-[var(--component-input-disabled-foreground)] md:text-sm",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--component-input-focus-ring)] focus-visible:ring-offset-2 focus-visible:ring-offset-background",
        "aria-invalid:ring-destructive/20 dark:aria-invalid:ring-destructive/40 aria-invalid:border-destructive",
        className
      )}
      {...props}
    />
  );
}

export { Input };
