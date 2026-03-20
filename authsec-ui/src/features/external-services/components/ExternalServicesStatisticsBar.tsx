import React from "react";
import { Card } from "@/components/ui/card";
import { Activity, Users } from "lucide-react";

interface ExternalServicesStatisticsBarProps {
  totalServices: number;
  integrationsUsed: number;
  usersWithAccess: number;
}

export function ExternalServicesStatisticsBar({
  totalServices,
  integrationsUsed,
  usersWithAccess,
}: ExternalServicesStatisticsBarProps) {
  return (
    <div className="grid gap-4 md:grid-cols-3 mb-6">
      <Card className="p-4 bg-card border-0">
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-primary/10 p-2">
            <Activity className="h-5 w-5 text-primary" />
          </div>
          <div className="space-y-0.5">
            <p className="text-xs font-medium text-foreground uppercase tracking-wide">
              Total Services
            </p>
            <p className="text-2xl font-bold text-foreground">{totalServices}</p>
          </div>
        </div>
      </Card>

      <Card className="p-4 bg-card border-0">
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-green-500/10 p-2">
            <Activity className="h-5 w-5 text-green-600 dark:text-green-400" />
          </div>
          <div className="space-y-0.5">
            <p className="text-xs font-medium text-foreground uppercase tracking-wide">
              Integrations Used
            </p>
            <p className="text-2xl font-bold text-foreground">{integrationsUsed}</p>
          </div>
        </div>
      </Card>

      <Card className="p-4 bg-card border-0">
        <div className="flex items-center gap-3">
          <div className="rounded-lg bg-blue-500/10 p-2">
            <Users className="h-5 w-5 text-blue-600 dark:text-blue-400" />
          </div>
          <div className="space-y-0.5">
            <p className="text-xs font-medium text-foreground uppercase tracking-wide">
              Users with Access
            </p>
            <p className="text-2xl font-bold text-foreground">{usersWithAccess}</p>
          </div>
        </div>
      </Card>
    </div>
  );
}
