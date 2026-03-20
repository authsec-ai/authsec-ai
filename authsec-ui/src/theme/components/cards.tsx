import { forwardRef } from "react";

import { Card, type CardProps } from "@/components/ui/card";
import { cn } from "@/lib/utils";

type VariantCardProps = Omit<CardProps, "variant">;

export const HeaderCard = forwardRef<HTMLDivElement, VariantCardProps>(
  ({ className, ...props }, ref) => (
    <Card
      ref={ref}
      variant="header"
      className={cn(className)}
      {...props}
    />
  )
);

HeaderCard.displayName = "HeaderCard";

export const FilterCard = forwardRef<HTMLDivElement, VariantCardProps>(
  ({ className, ...props }, ref) => (
    <Card
      ref={ref}
      variant="filter"
      className={cn(className)}
      {...props}
    />
  )
);

FilterCard.displayName = "FilterCard";

export const TableCard = forwardRef<HTMLDivElement, VariantCardProps>(
  ({ className, ...props }, ref) => (
    <Card
      ref={ref}
      variant="table"
      className={cn(className)}
      {...props}
    />
  )
);

TableCard.displayName = "TableCard";

export const PaginationCard = forwardRef<HTMLDivElement, VariantCardProps>(
  ({ className, ...props }, ref) => (
    <Card
      ref={ref}
      variant="pagination"
      className={cn(className)}
      {...props}
    />
  )
);

PaginationCard.displayName = "PaginationCard";
