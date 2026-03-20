"use client";
import * as React from "react";

import { cn } from "../../lib/utils";

type TableProps = React.ComponentProps<"table"> & {
  containerClassName?: string;
  bordered?: boolean;
};

function Table({ className, containerClassName, bordered = true, ...props }: TableProps) {
  return (
    <div
      data-slot="table-container"
      data-bordered={bordered ? "true" : "false"}
      className={cn(
        "relative w-full overflow-x-auto-hidden overflow-y-hidden rounded-none bg-[var(--component-table-surface)] shadow-none",
        bordered && "border border-[var(--component-table-border)]",
        containerClassName
      )}
    >
      <table
        data-slot="table"
        className={cn(
          "w-full caption-bottom text-[length:var(--font-size-body-sm)] text-[color:var(--color-text-secondary)]",
          className
        )}
        {...props}
      />
    </div>
  );
}

function TableHeader({ className, ...props }: React.ComponentProps<"thead">) {
  return (
    <thead
      data-slot="table-header"
      className={cn("[&_tr]:border-b [&_tr]:border-[var(--component-table-border)]", className)}
      {...props}
    />
  );
}

function TableBody({ className, ...props }: React.ComponentProps<"tbody">) {
  return (
    <tbody
      data-slot="table-body"
      className={cn("[&_tr:last-child]:border-0", className)}
      {...props}
    />
  );
}

function TableFooter({ className, ...props }: React.ComponentProps<"tfoot">) {
  return (
    <tfoot
      data-slot="table-footer"
      className={cn(
        "border-t border-[var(--component-table-border)] bg-[var(--component-table-header-bg)] font-medium [&>tr]:last:border-b-0",
        className
      )}
      {...props}
    />
  );
}

function TableRow({ className, ...props }: React.ComponentProps<"tr">) {
  return (
    <tr
      data-slot="table-row"
      className={cn(
        "border-b border-[var(--component-table-border)] transition-colors hover:bg-[var(--component-table-row-hover)] data-[state=selected]:bg-[var(--component-table-row-selected)]",
        className
      )}
      {...props}
    />
  );
}

function TableHead({ className, ...props }: React.ComponentProps<"th">) {
  return (
    <th
      data-slot="table-head"
      className={cn(
        "h-[var(--component-table-header-height)] px-[var(--component-table-cell-padding-inline)] text-left align-middle font-[var(--font-weight-medium)] text-[length:var(--font-size-body-sm)] tracking-[var(--letter-spacing-normal)] text-[color:var(--component-table-header-fg)] whitespace-nowrap [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        className
      )}
      {...props}
    />
  );
}

function TableCell({ className, ...props }: React.ComponentProps<"td">) {
  return (
    <td
      data-slot="table-cell"
      className={cn(
        "px-[var(--component-table-cell-padding-inline)] py-[var(--component-table-cell-padding-block)] align-middle whitespace-nowrap text-[length:var(--font-size-body-sm)] text-[color:var(--color-text-primary)] leading-[var(--line-height-body)] [&:has([role=checkbox])]:pr-0 [&>[role=checkbox]]:translate-y-[2px]",
        className
      )}
      {...props}
    />
  );
}

function TableCaption({ className, ...props }: React.ComponentProps<"caption">) {
  return (
    <caption
      data-slot="table-caption"
      className={cn(
        "mt-4 text-[length:var(--font-size-body-xs)] text-[color:var(--color-text-secondary)]",
        className
      )}
      {...props}
    />
  );
}

export { Table, TableHeader, TableBody, TableFooter, TableHead, TableRow, TableCell, TableCaption };
