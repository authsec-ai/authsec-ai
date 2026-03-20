import { DashboardStatusTag, type DashboardStatusTone } from "./DashboardStatusTag";
import { DashboardTile } from "./DashboardTile";

export interface DashboardSummaryRailItem {
  id: string;
  label: string;
  value: string;
  meta?: string;
  tagLabel?: string;
  tagTone?: DashboardStatusTone;
}

interface DashboardSummaryRailProps {
  items: DashboardSummaryRailItem[];
}

export function DashboardSummaryRail({ items }: DashboardSummaryRailProps) {
  return (
    <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-5">
      {items.map((item, index) => (
        <DashboardTile
          key={item.id}
          variant={index % 2 === 0 ? "panel" : "alt"}
          className="p-3"
        >
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0 space-y-1">
              <div className="dash-kpi-label">{item.label}</div>
              <div className="dash-kpi-value truncate">{item.value}</div>
              {item.meta && (
                <div className="dash-kpi-meta line-clamp-1">{item.meta}</div>
              )}
            </div>
            {item.tagLabel && (
              <DashboardStatusTag
                label={item.tagLabel}
                tone={item.tagTone ?? "neutral"}
                className="mt-0.5"
              />
            )}
          </div>
        </DashboardTile>
      ))}
    </div>
  );
}

