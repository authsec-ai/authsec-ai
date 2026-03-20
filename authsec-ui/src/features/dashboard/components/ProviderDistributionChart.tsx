import React from "react";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "../../../components/ui/card";
import { BarChart3 } from "lucide-react";
import type { ProviderDistribution } from "../../../app/api/dashboardApi";
import { getProviderColor, formatProviderName } from "../utils/dashboard-utils";

interface ProviderDistributionChartProps {
  data: ProviderDistribution[];
  isLoading?: boolean;
}

export function ProviderDistributionChart({
  data,
  isLoading = false,
}: ProviderDistributionChartProps) {
  const maxCount = Math.max(...data.map((d) => d.count), 1);
  const totalUsers = data.reduce((sum, item) => sum + item.count, 0);

  if (isLoading) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Provider Distribution
          </CardTitle>
          <CardDescription>Users by authentication provider</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            {[1, 2, 3, 4].map((i) => (
              <div key={i} className="space-y-2">
                <div className="h-4 w-32 bg-muted animate-pulse rounded" />
                <div className="h-6 bg-muted animate-pulse rounded" />
              </div>
            ))}
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!data || data.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <BarChart3 className="h-5 w-5" />
            Provider Distribution
          </CardTitle>
          <CardDescription>Users by authentication provider</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center h-64 text-foreground">
            No user data available
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <BarChart3 className="h-5 w-5" />
          Provider Distribution
        </CardTitle>
        <p className="text-sm text-foreground">
          Users by authentication provider
        </p>
      </CardHeader>
      <CardContent>
        <div className="space-y-4">
          {data.map((item, index) => (
            <div key={item.provider} className="space-y-2">
              <div className="flex items-center justify-between text-sm">
                <div className="flex items-center gap-2">
                  <div
                    className="h-3 w-3 rounded-full"
                    style={{ backgroundColor: getProviderColor(item.provider) }}
                  />
                  <span className="font-medium capitalize">{formatProviderName(item.provider)}</span>
                </div>
                <div className="flex items-center gap-3">
                  <span className="text-foreground">{item.count} users</span>
                  <span className="font-semibold">{item.percentage.toFixed(1)}%</span>
                </div>
              </div>
              <div className="relative h-2 w-full overflow-hidden rounded-full bg-muted">
                <div
                  className="h-full rounded-full transition-all duration-500"
                  style={{
                    width: `${(item.count / maxCount) * 100}%`,
                    backgroundColor: getProviderColor(item.provider),
                  }}
                />
              </div>
            </div>
          ))}
        </div>

        {/* Summary */}
        <div className="mt-6 pt-4 border-t border-border/50">
          <div className="grid grid-cols-2 gap-4 text-sm">
            <div>
              <p className="text-foreground">Total Providers</p>
              <p className="text-lg font-semibold">{data.length}</p>
            </div>
            <div>
              <p className="text-foreground">Total Users</p>
              <p className="text-lg font-semibold">
                {totalUsers.toLocaleString()}
              </p>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
