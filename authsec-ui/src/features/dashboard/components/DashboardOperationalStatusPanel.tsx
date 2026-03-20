import type { KeyboardEvent, MouseEvent } from "react";
import { ShieldCheck, Logs, Building2, Users, ArrowRight } from "lucide-react";
import { DashboardSection } from "./DashboardSection";
import { DashboardStatusTag, type DashboardStatusTone } from "./DashboardStatusTag";

export interface DashboardOperationalStatusItem {
  id: string;
  label: string;
  detail: string;
  statusLabel: string;
  tone: DashboardStatusTone;
}

interface DashboardOperationalStatusPanelProps {
  items: DashboardOperationalStatusItem[];
  recommendation?: {
    title: string;
    detail: string;
  };
  onRecommendationAction?: () => void;
  onItemAction?: (item: DashboardOperationalStatusItem) => void;
}

const iconById = {
  "ad-sync": Users,
  "auth-methods": ShieldCheck,
  domains: Building2,
  logging: Logs,
} as const;

export function DashboardOperationalStatusPanel({
  items,
  recommendation,
  onRecommendationAction,
  onItemAction,
}: DashboardOperationalStatusPanelProps) {
  const handleRecommendationKeyDown = (event: KeyboardEvent<HTMLDivElement>) => {
    if (!onRecommendationAction) return;
    if (event.key === "Enter" || event.key === " ") {
      event.preventDefault();
      onRecommendationAction();
    }
  };

  const handleItemKeyDown =
    (item: DashboardOperationalStatusItem) => (event: KeyboardEvent<HTMLDivElement>) => {
      if (!onItemAction) return;
      if (event.key === "Enter" || event.key === " ") {
        event.preventDefault();
        onItemAction(item);
      }
    };

  const handleItemClick =
    (item: DashboardOperationalStatusItem) => (_event: MouseEvent<HTMLDivElement>) => {
      if (!onItemAction) return;
      onItemAction(item);
    };

  return (
    <DashboardSection
      label="Operations"
      title="Operational Status"
      description="Current readiness across the most-used configuration areas."
    >
      <div className="space-y-2">
        {recommendation && (
          <div
            className="dash-callout-focus p-3"
            data-clickable={onRecommendationAction ? "true" : undefined}
            role={onRecommendationAction ? "button" : undefined}
            tabIndex={onRecommendationAction ? 0 : undefined}
            onClick={onRecommendationAction}
            onKeyDown={handleRecommendationKeyDown}
          >
            <div className="dash-callout-focus-body flex items-start justify-between gap-3">
              <div className="min-w-0 space-y-1">
                <div className="flex flex-wrap items-center gap-2">
                  <div className="dash-eyebrow">Next Recommended Action</div>
                  <DashboardStatusTag label="Priority" tone="accent" />
                </div>
                <div className="text-base font-semibold leading-tight dash-text-1">
                  {recommendation.title}
                </div>
                <div className="text-xs dash-text-2 leading-relaxed">
                  {recommendation.detail}
                </div>
              </div>
              <div className="flex h-8 w-8 items-center justify-center rounded-md border border-[var(--dash-border-soft)] bg-[var(--dash-surface-panel-alt)] text-[var(--dash-accent)] shrink-0">
                <ArrowRight className="h-4 w-4" />
              </div>
            </div>
          </div>
        )}

        <div className="dash-group-panel divide-y dash-divider">
          {items.map((item) => {
            const Icon = iconById[item.id as keyof typeof iconById] ?? ShieldCheck;
            return (
              <div
                key={item.id}
                className="dash-row-flat px-3 py-2.5"
                role={onItemAction ? "button" : undefined}
                tabIndex={onItemAction ? 0 : undefined}
                onClick={handleItemClick(item)}
                onKeyDown={handleItemKeyDown(item)}
              >
                <div className="grid gap-3 sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-center">
                  <div className="dash-icon-chip" data-tone={item.tone}>
                    <Icon className="h-4 w-4" />
                  </div>

                  <div className="min-w-0 space-y-0.5">
                    <div className="text-sm font-semibold dash-text-1">
                      {item.label}
                    </div>
                    <div className="text-xs dash-text-2 line-clamp-1">
                      {item.detail}
                    </div>
                  </div>

                  <div className="flex items-center sm:justify-end">
                    <DashboardStatusTag label={item.statusLabel} tone={item.tone} />
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    </DashboardSection>
  );
}
