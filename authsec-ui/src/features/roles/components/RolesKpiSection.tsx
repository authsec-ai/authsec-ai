import React from "react";
import { MetricCard, MetricCardGrid } from "@/components/ui/metric-card";
import { Users, PackageOpen, SlidersHorizontal, Clock } from "lucide-react";

interface RolesKpiSectionProps {
  totalRoles: number;
  unusedRoles: number;
  broadRoles: number;
  pendingChanges: number;
  onKpiClick?: (filter: "all" | "unused" | "broad" | "pending") => void;
}

/**
 * KPI card section for the Roles dashboard.
 * Abstracted into its own component to keep RolesPage below the 300-line limit
 * and to enable future reuse in other contexts (e.g. overview dashboards).
 */
export function RolesKpiSection({
  totalRoles,
  unusedRoles,
  broadRoles,
  pendingChanges,
  onKpiClick,
}: RolesKpiSectionProps) {
  return (
    <MetricCardGrid>
      <MetricCard
        title="Total Roles"
        value={totalRoles.toString()}
        footer={{ primary: "All role definitions", icon: Users }}
        onClick={() => onKpiClick?.("all")}
      />
      <MetricCard
        title="Unused Roles"
        value={unusedRoles.toString()}
        footer={{ primary: "0 users & 0 groups", icon: PackageOpen }}
        colorVariant="amber"
        onClick={() => onKpiClick?.("unused")}
      />
      <MetricCard
        title="Broad Roles"
        value={broadRoles.toString()}
        footer={{ primary: "≥10 resources", icon: SlidersHorizontal }}
        colorVariant="green"
        onClick={() => onKpiClick?.("broad")}
      />
      <MetricCard
        title="Pending Changes"
        value={pendingChanges.toString()}
        footer={{ primary: "Drafts awaiting review", icon: Clock }}
        colorVariant="purple"
        onClick={() => onKpiClick?.("pending")}
      />
    </MetricCardGrid>
  );
}
 